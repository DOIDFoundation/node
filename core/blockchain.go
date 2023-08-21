package core

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"sync"

	"github.com/DOIDFoundation/node/config"
	"github.com/DOIDFoundation/node/events"
	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/store"
	"github.com/DOIDFoundation/node/transactor"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/iavl"
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
		block = types.NewBlockWithHeader(GenesisHeader(config.NetworkID))
		bc.latestTD = new(big.Int)
		bc.writeBlockAndTd(block)
		bc.blockStore.WriteHashByHeight(block.Header.Height.Uint64(), block.Hash(), block.Header.Miner)
		bc.blockStore.WriteHeadBlockHash(block.Hash())
		if hash, version, err := bc.state.SaveVersion(); err != nil || version != 1 || !bytes.Equal(hash, block.Header.Root) {
			logger.Error("failed to save genesis block state", "err", err, "block", block.Header, "newHash", hash, "newVersion", version)
			return nil, err
		}
	}
	bc.Logger.Info("load head block", "height", block.Header.Height, "header", block.Header)
	bc.rewindToBlock(block)

	bc.registerEventHandlers()
	RegisterAPI(bc)
	failed = false
	return bc, nil
}

func (bc *BlockChain) NewHeaderChain() *HeaderChain {
	return newHeaderChain(bc.blockStore, bc.Logger)
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
	events.NewNetworkBlock.Subscribe("blockchain", func(data events.BlockWithTd) {
		block := data.Block
		if bytes.Equal(bc.blockStore.ReadHashByHeight(block.Header.Height.Uint64()), block.Hash()) {
			bc.Logger.Debug("block from network already known", "block", block.Hash(), "header", block.Header)
			return
		}
		if block.Header.Height.Uint64() == bc.latestBlock.Header.Height.Uint64()+1 {
			if err := bc.ApplyBlock(block); err == nil {
				return
			} else if !errors.Is(err, types.ErrNotContiguous) {
				bc.Logger.Info("discard block from network", "err", err, "block", block.Hash(), "header", block.Header)
				return
			}
		}
		if data.Td.Cmp(bc.latestTD) > 0 {
			bc.Logger.Info("better network td, maybe a fork", "block", block.Hash(), "header", block.Header, "td", data.Td)
			events.ForkDetected.Send(struct{}{})
		}
	})
}

func (bc *BlockChain) setHead(block *types.Block) {
	bc.Logger.Info("head block", "block", block.Hash(), "header", block.Header)
	if bc.state.Version() != block.Header.Height.Int64() {
		bc.Logger.Error("block height and state version mismatch", "state", bc.state.Version(), "header", block.Header)
		panic("state and block mismatch")
	}
	bc.blockStore.WriteHeadBlockHash(block.Hash())
	events.NewChainHead.Send(block)
}

func (bc *BlockChain) HasBlock(height uint64, hash types.Hash) bool {
	return bytes.Equal(bc.blockStore.ReadHashByHeight(height), hash)
}

func (bc *BlockChain) GetBlock(height uint64, hash types.Hash) *types.Block {
	return bc.blockStore.ReadBlock(height, hash)
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
	if err := block.Header.IsValid(bc.LatestBlock().Header); err != nil {
		bc.Logger.Error("invalid block", "header", block.Header, "last", bc.LatestBlock().Header, "hash", bc.LatestBlock().Hash(), "err", err)
		return err
	}

	state := bc.state

	result, err := transactor.ApplyTxs(state, block.Txs)
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
	bc.writeBlockAndTd(block)
	bc.latestBlock = block
	return nil
}

func (bc *BlockChain) writeBlockAndTd(block *types.Block) {
	bc.blockStore.WriteBlock(block)
	bc.latestTD.Add(bc.latestTD, block.Header.Difficulty)
	bc.blockStore.WriteTd(block.Header.Height.Uint64(), block.Hash(), bc.latestTD)
}

func (bc *BlockChain) ApplyBlock(block *types.Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if err := bc.applyBlockAndWrite(block); err != nil {
		return err
	}
	bc.blockStore.WriteHashByHeight(block.Header.Height.Uint64(), block.Hash(), block.Header.Miner)
	bc.setHead(block)

	return nil
}

func (bc *BlockChain) rewindToBlock(block *types.Block) {
	bc.latestBlock = block
	if err := bc.loadLatestState(); err != nil {
		bc.Logger.Error("failed to rewind state", "err", err, "height", block.Header.Height)
		panic(err)
	}
	bc.latestTD = bc.blockStore.ReadTd(block.Header.Height.Uint64(), block.Hash())
}

func (bc *BlockChain) GetTd() *big.Int {
	return bc.latestTD
}

func (bc *BlockChain) ApplyHeaderChain(hc *HeaderChain) error {
	td := hc.GetTd()
	if td == nil || len(hc.headers) == 0 {
		return nil
	}
	currentBlock := bc.LatestBlock()
	currentTd := bc.GetTd()
	if td.Cmp(currentTd) <= 0 {
		return errors.New("total difficulty lower than current")
	}

	// check if we can find first block's parent
	first := hc.getBlock(hc.headers[0].Height, hc.headers[0].Hash)
	localParent := bc.BlockByHeight(first.Header.Height.Uint64() - 1)
	if localParent == nil || !bytes.Equal(localParent.Hash(), first.Header.ParentHash) {
		return ErrUnknownAncestor
	}

	bc.mu.Lock()
	defer bc.mu.Unlock()

	// rewind state to parent of first block
	bc.rewindToBlock(localParent)
	// td is also set to parent of first block
	td = bc.blockStore.ReadTd(localParent.Header.Height.Uint64(), localParent.Hash())

	// apply all valid blocks and sum total difficulty
	var latest *types.Block
	for _, header := range hc.headers {
		block := hc.getBlock(header.Height, header.Hash)
		if err := bc.applyBlockAndWrite(block); err != nil {
			bc.Logger.Error("failed to apply block", "err", err, "header", block.Header)
			break
		}
		td.Add(td, block.Header.Difficulty)
		latest = block
	}

	// check total difficulty to see if we can switch
	if latest != nil && currentTd.Cmp(td) < 0 {
		// reset hash by height to new chain
		bc.blockStore.DeleteHashByHeightFrom(localParent.Header.Height.Uint64() + 1)
		for _, header := range hc.headers {
			if header.Height > latest.Header.Height.Uint64() {
				break
			}
			bc.blockStore.WriteHashByHeight(header.Height, header.Hash, header.Miner)
		}
		// switch to new chain
		bc.setHead(latest)
		return nil
	}

	// rollback
	bc.rewindToBlock(localParent)
	for i := localParent.Header.Height.Uint64() + 1; i <= currentBlock.Header.Height.Uint64(); i++ {
		block := bc.blockStore.ReadBlock(i, bc.blockStore.ReadHashByHeight(i))
		if err := bc.applyBlockAndWrite(block); err != nil {
			bc.Logger.Error("failed to apply block", "err", err, "header", block.Header)
			panic("failed to rollback blocks")
		}
	}
	bc.setHead(currentBlock)
	return errors.New("total difficulty lower than current")
}
