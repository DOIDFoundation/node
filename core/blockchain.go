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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/spf13/viper"
)

type BlockChain struct {
	Logger      log.Logger
	blockStore  *store.BlockStore
	latestBlock *types.Block

	chainHeadFeed event.Feed

	stateDb cosmosdb.DB
	state   *iavl.MutableTree
	mu      sync.Mutex
}

func NewBlockChain(logger log.Logger) (*BlockChain, error) {
	blockStore, err := store.NewBlockStore(logger)
	if err != nil {
		return nil, err
	}

	homeDir := viper.GetString(flags.Home)
	db, err := cosmosdb.NewDB("state", cosmosdb.BackendType(viper.GetString(flags.DB_Engine)), filepath.Join(homeDir, "data"))
	if err != nil {
		return nil, err
	}

	bc := &BlockChain{
		blockStore: blockStore,
		Logger:     logger.With("module", "blockchain"),
		stateDb:    db,
	}

	block := blockStore.ReadHeadBlock()
	if block == nil {
		bc.Logger.Info("no head block found, generate from genesis")
		block = types.NewBlockWithHeader(&types.Header{
			Difficulty: big.NewInt(0x1000000),
			Height:     big.NewInt(0),
			Root:       hexutil.MustDecode("0xE3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"),
			TxHash:     hexutil.MustDecode("0xE3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"),
		})
		bc.blockStore.WriteTd(block.Hash(), block.Header.Height.Uint64(), block.Header.Difficulty)
		bc.writeHeadBlock(block)
	} else {
		bc.Logger.Info("load head block", "height", block.Header.Height, "header", block.Header)
	}
	bc.latestBlock = block

	bc.registerEventHandlers()

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

func (bc *BlockChain) writeHeadBlock(block *types.Block) {
	bc.blockStore.WriteBlock(block)
	bc.blockStore.WriteHeadBlockHash(block.Hash())
}

func (bc *BlockChain) SetHead(block *types.Block) {
	bc.Logger.Info("head block", "block", block.Hash(), "header", block.Header)
	td := bc.blockStore.ReadTd(bc.CurrentBlock().Hash(), bc.CurrentBlock().Header.Height.Uint64())
	td.Add(td, block.Header.Difficulty)
	bc.blockStore.WriteTd(block.Hash(), block.Header.Height.Uint64(), td)
	bc.writeHeadBlock(block)
	bc.latestBlock = block
	events.NewChainHead.Send(block)
}

func (bc *BlockChain) BlockByHeight(height uint64) *types.Block {
	return bc.blockStore.ReadBlockByHeight(height)
}

// CurrentBlock retrieves the current head block of the canonical chain. The
// block is retrieved from the blockchain's internal cache.
func (bc *BlockChain) CurrentBlock() *types.Block {
	// return bc.currentBlock.Load().(*types.Block)
	return bc.latestBlock
}

// LatestBlock retrieves the latest head block of the canonical chain. The
// block is retrieved from the blockchain's internal cache.
func (bc *BlockChain) LatestBlock() *types.Block {
	return bc.latestBlock
}

func (bc *BlockChain) LatestState() (*iavl.ImmutableTree, error) {
	state, err := bc.mutableState()
	if err != nil {
		return nil, err
	}
	return state.GetImmutable(bc.LatestBlock().Header.Height.Int64())
}

// newOpenState returns a new mutable tree of latest block. Notice: Do not save
// any modification to avoid collisions.
func (bc *BlockChain) newOpenState() (*iavl.MutableTree, error) {
	tree, err := iavl.NewMutableTree(bc.stateDb, 128, false)
	if err != nil {
		bc.Logger.Error("failed to open block state", "err", err)
		return nil, err
	}
	if _, err := tree.LazyLoadVersionForOverwriting(bc.LatestBlock().Header.Height.Int64()); err != nil {
		bc.Logger.Error("failed to open block state", "err", err)
		return nil, err
	}
	if tree.Version() != bc.LatestBlock().Header.Height.Int64() {
		bc.Logger.Error("version mismatch", "stateVersion", tree.Version(), "latestBlock", bc.LatestBlock().Header.Height)
		return nil, errors.New("bad state version in db")
	}
	hash, err := tree.Hash()
	if err != nil {
		bc.Logger.Error("failed to open block state", "err", err)
		return nil, err
	}
	if !bytes.Equal(hash, bc.LatestBlock().Header.Root) {
		bc.Logger.Error("root hash mismatch", "state", hash, "latestBlock", bc.LatestBlock().Header.Root)
		return nil, errors.New("bad state hash in db")
	}
	return tree, nil
}

func (bc *BlockChain) mutableState() (*iavl.MutableTree, error) {
	if bc.state != nil {
		return bc.state, nil
	}

	bc.mu.Lock()
	defer bc.mu.Unlock()
	if bc.state != nil {
		return bc.state, nil
	}

	tree, err := bc.newOpenState()
	bc.state = tree
	return tree, err
}

func (bc *BlockChain) Simulate(txs types.Txs) (*transactor.ExecutionResult, error) {
	state, err := bc.mutableState()
	if err != nil {
		return nil, err
	}
	defer state.Rollback()
	return transactor.ApplyTxs(state, txs)
}

func (bc *BlockChain) ApplyBlock(block *types.Block) error {
	if big.NewInt(0).Add(bc.LatestBlock().Header.Height, common.Big1).Cmp(block.Header.Height) != 0 {
		return fmt.Errorf("block not contiguous, latest height %v, new block height %v", bc.LatestBlock().Header.Height, block.Header.Height)
	}
	if !bytes.Equal(block.Header.ParentHash, bc.LatestBlock().Hash()) {
		return fmt.Errorf("block not contiguous, latest hash %v, new block's parent %v", bc.LatestBlock().Hash(), block.Header.ParentHash)
	}
	state, err := bc.mutableState()
	if err != nil {
		return err
	}

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

	hash, version, err := state.SaveVersion()
	if err != nil {
		return err
	}
	if block.Header.Height.Cmp(big.NewInt(version)) != 0 {
		state.DeleteVersion(version)
		return fmt.Errorf("state version mismatch, want %v, got %v", block.Header.Height, version)
	}
	if !bytes.Equal(block.Header.Root, hash) {
		state.DeleteVersion(version)
		return fmt.Errorf("state hash mismatch, block root %v, got %v", block.Header.Root, hash)
	}

	bc.SetHead(block)

	return nil
}
