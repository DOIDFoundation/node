package core

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"sync"

	"github.com/DOIDFoundation/node/events"
	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/store"
	"github.com/DOIDFoundation/node/transactor"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/iavl"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/spf13/viper"
)

var (
	// ErrUnknownAncestor is returned when validating a block requires an ancestor
	// that is unknown.
	ErrUnknownAncestor = errors.New("unknown ancestor")

	// ErrFutureBlock is returned when a block's timestamp is in the future according
	// to the current node.
	ErrFutureBlock = errors.New("block in the future")
)

type BlockChain struct {
	Logger      log.Logger
	mu          sync.Mutex
	blockStore  *store.BlockStore
	latestBlock *types.Block
	latestTD    *big.Int

	stateDb cosmosdb.DB
	state   *iavl.MutableTree
}

func NewBlockChain(logger log.Logger) (*BlockChain, error) {
	bc := &BlockChain{
		Logger: logger.With("module", "blockchain"),
	}
	failed := true
	var err error
	if bc.blockStore, err = store.NewBlockStore(logger); err != nil {
		bc.Logger.Error("failed to new block store", "err", err)
		return nil, err
	}

	defer func() {
		if failed {
			bc.blockStore.Close()
		}
	}()

	homeDir := viper.GetString(flags.Home)
	if bc.stateDb, err = cosmosdb.NewDB("state", cosmosdb.BackendType(viper.GetString(flags.DB_Engine)), filepath.Join(homeDir, "data")); err != nil {
		bc.Logger.Error("failed to new state db", "err", err)
		return nil, err
	}

	defer func() {
		if failed {
			bc.stateDb.Close()
		}
	}()

	if bc.state, err = iavl.NewMutableTree(bc.stateDb, 128, false); err != nil {
		bc.Logger.Error("failed to new block state", "err", err)
		return nil, err
	}

	block := bc.blockStore.ReadHeadBlock()
	if block == nil {
		bc.Logger.Info("no head block found, generate from genesis")
		block = types.NewBlockWithHeader(&types.Header{
			Difficulty: big.NewInt(0x1000000),
			Height:     big.NewInt(1), // starts from 1 because iavl can not go back to zero
			Root:       hexutil.MustDecode("0xE3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"),
			TxHash:     hexutil.MustDecode("0xE3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"),
		})
		bc.latestTD = new(big.Int)
		bc.writeBlock(block)
		bc.blockStore.WriteHashByHeight(block.Header.Height.Uint64(), block.Hash())
		bc.blockStore.WriteHeadBlockHash(block.Hash())
		bc.latestBlock = block
		if hash, version, err := bc.state.SaveVersion(); err != nil || version != 1 || !bytes.Equal(hash, block.Header.Root) {
			logger.Error("failed to save genesis block state", "err", err, "block", block.Header, "newHash", hash, "newVersion", version)
			return nil, err
		}
	} else {
		bc.Logger.Info("load head block", "height", block.Header.Height, "header", block.Header)
		bc.latestTD = bc.blockStore.ReadTd(block.Hash(), block.Header.Height.Uint64())
		bc.latestBlock = block
		if err = bc.loadLatestState(); err != nil {
			logger.Error("failed to load latest block state", "err", err, "block", block.Header)
			return nil, err
		}
	}

	bc.registerEventHandlers()
	failed = false
	return bc, nil
}

func (bc *BlockChain) Close() {
	events.NewNetworkBlock.Unsubscribe("blockchain")
	if err := bc.stateDb.Close(); err != nil {
		bc.Logger.Error("error closing state database", "err", err)
	}
	if err := bc.blockStore.Close(); err != nil {
		bc.Logger.Error("error closing block store", "err", err)
	}
}

func (bc *BlockChain) registerEventHandlers() {
	events.NewNetworkBlock.Subscribe("blockchain", func(block *types.Block) {
		if err := bc.ApplyBlock(block); err != nil {
			bc.Logger.Error("bad block from network", "err", err, "block", block.Hash(), "header", block.Header)
			// @todo check if fork happened
			events.ForkDetected.Send(struct{}{})
		}
	})
}

func (bc *BlockChain) setHead(block *types.Block) {
	bc.Logger.Info("head block", "block", block.Hash(), "header", block.Header)
	bc.blockStore.WriteHeadBlockHash(block.Hash())
	bc.latestBlock = block
	events.NewChainHead.Send(block)
}

func (bc *BlockChain) BlockByHeight(height uint64) *types.Block {
	return bc.blockStore.ReadBlock(height, bc.blockStore.ReadHashByHeight(height))
}

// LatestBlock retrieves the latest head block of the canonical chain. The
// block is retrieved from the blockchain's internal cache.
func (bc *BlockChain) LatestBlock() *types.Block {
	return bc.latestBlock
}

func (bc *BlockChain) LatestState() (*iavl.ImmutableTree, error) {
	return bc.state.GetImmutable(bc.state.Version())
}

// loadLatestState loads mutable tree of latest block.
func (bc *BlockChain) loadLatestState() error {
	tree := bc.state
	latestHeight := bc.LatestBlock().Header.Height.Int64()
	if latestHeight == 0 {
		// this should not happen, as block starts from 1
		panic("should not load state with height 0")
	} else if version, err := tree.LazyLoadVersionForOverwriting(latestHeight); err != nil {
		bc.Logger.Error("failed to open block state for overwriting", "err", err, "want", latestHeight, "latest", version)
		return err
	}
	if tree.Version() != latestHeight {
		bc.Logger.Error("version mismatch", "stateVersion", tree.Version(), "latestBlock", bc.LatestBlock().Header.Height)
		return errors.New("bad state version in db")
	}
	hash, err := tree.Hash()
	if err != nil {
		bc.Logger.Error("failed to open block state", "err", err)
		return err
	}
	if !bytes.Equal(hash, bc.LatestBlock().Header.Root) {
		bc.Logger.Error("root hash mismatch", "state", hash, "latestBlock", bc.LatestBlock().Header.Root)
		return errors.New("bad state hash in db")
	}
	return nil
}

func (bc *BlockChain) Simulate(txs types.Txs) (*transactor.ExecutionResult, error) {
	defer bc.state.Rollback()
	return transactor.ApplyTxs(bc.state, txs)
}

func (bc *BlockChain) applyBlockAndWrite(block *types.Block) error {
	state := bc.state

	result, err := transactor.ApplyTxs(state, block.Data.Txs)
	if err != nil {
		return err
	}
	if !bytes.Equal(block.Header.Root, result.StateRoot) ||
		!bytes.Equal(block.Header.ReceiptHash, result.ReceiptRoot) ||
		!bytes.Equal(block.Header.TxHash, result.TxRoot) {
		state.Rollback()
		bc.Logger.Debug("block apply result mismatch", "header", block.Header, "result", result)
		return errors.New("block apply result mismatch")
	}
	// @todo process rejected txs and receipts

	// save state, block and total difficulty
	hash, version, err := state.SaveVersion()
	if err != nil {
		return err
	}
	if block.Header.Height.Int64() != version {
		state.DeleteVersion(version)
		return fmt.Errorf("state version mismatch, want %v, got %v", block.Header.Height, version)
	}
	if !bytes.Equal(block.Header.Root, hash) {
		state.DeleteVersion(version)
		return fmt.Errorf("state hash mismatch, block root %v, got %v", block.Header.Root, hash)
	}
	bc.writeBlock(block)
	return nil
}

func (bc *BlockChain) writeBlock(block *types.Block) {
	bc.blockStore.WriteBlock(block)
	bc.latestTD.Add(bc.latestTD, block.Header.Difficulty)
	bc.blockStore.WriteTd(block.Hash(), block.Header.Height.Uint64(), bc.latestTD)
}

func (bc *BlockChain) ApplyBlock(block *types.Block) error {
	if bc.LatestBlock().Header.Height.Uint64()+1 != block.Header.Height.Uint64() {
		return fmt.Errorf("block not contiguous, latest height %v, new block height %v", bc.LatestBlock().Header.Height, block.Header.Height)
	}
	if !bytes.Equal(block.Header.ParentHash, bc.LatestBlock().Hash()) {
		return fmt.Errorf("block not contiguous, latest hash %v, new block's parent %v", bc.LatestBlock().Hash(), block.Header.ParentHash)
	}

	bc.mu.Lock()
	defer bc.mu.Unlock()
	if err := bc.applyBlockAndWrite(block); err != nil {
		return err
	}
	bc.blockStore.WriteHashByHeight(block.Header.Height.Uint64(), block.Hash())
	bc.setHead(block)

	return nil
}

func (bc *BlockChain) InsertBlocks(blocks []*types.Block) error {
	if len(blocks) == 0 {
		return nil
	}

	// we can not find first block's parent
	first := blocks[0]
	localParent := bc.BlockByHeight(first.Header.Height.Uint64() - 1)
	if localParent == nil || !bytes.Equal(localParent.Hash(), first.Header.ParentHash) {
		return ErrUnknownAncestor
	}

	// check if blocks to import are contiguous
	var prev *types.Block
	for _, block := range blocks {
		if prev != nil && block.Header.Height.Uint64() != prev.Header.Height.Uint64()+1 && !bytes.Equal(block.Header.ParentHash, prev.Hash()) {
			bc.Logger.Error("Non contiguous block insert", "height", block.Header.Height, "hash", block.Hash(),
				"parent", block.Header.ParentHash, "prev", prev.Header)
			return errors.New("blocks not continues")
		}
		prev = block
	}

	bc.mu.Lock()
	defer bc.mu.Unlock()

	currentTd := new(big.Int)
	currentTd.Set(bc.latestTD)

	bc.rewindToBlock(localParent)

	for _, block := range blocks {
		if err := bc.applyBlockAndWrite(block); err != nil {
			// error occurs, rollback, maybe we can reapply dropped blocks here to speed up
			bc.rewindToBlock(localParent)
			return err
		}
	}

	bc.setHead(blocks[len(blocks)-1])
	return nil
}

func (bc *BlockChain) rewindToBlock(block *types.Block) {
	bc.latestBlock = block
	if err := bc.loadLatestState(); err != nil {
		bc.Logger.Error("failed to rewind state", "err", err, "height", block.Header.Height)
		panic(err)
	}
	bc.latestTD = bc.blockStore.ReadTd(block.Hash(), block.Header.Height.Uint64())
}
