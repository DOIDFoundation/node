package core

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"sync"

	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/store"
	"github.com/DOIDFoundation/node/transactor"
	"github.com/DOIDFoundation/node/types"
	"github.com/cometbft/cometbft/libs/log"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/iavl"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/spf13/viper"
)

type BlockChain struct {
	logger      log.Logger
	blockStore  *store.BlockStore
	latestBlock *types.Block

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
		logger:     logger.With("module", "blockchain"),
		stateDb:    db,
	}

	var block *types.Block = nil
	headBlockHash := blockStore.ReadHeadBlockHash()
	if headBlockHash != nil {
		block = blockStore.ReadBlock(headBlockHash)
		bc.logger.Info("found head block", "hash", headBlockHash, "block", block)
	}
	if block == nil {
		block = types.NewBlockWithHeader(&types.Header{
			Difficulty: big.NewInt(0x1000000),
			Height:     big.NewInt(0),
			Root:       hexutil.MustDecode("0xE3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"),
			TxHash:     hexutil.MustDecode("0xE3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"),
		})
	}
	bc.latestBlock = block
	return bc, nil
}

func (bc *BlockChain) Close() {
	if err := bc.stateDb.Close(); err != nil {
		bc.logger.Error("error closing state database", "err", err)
	}
	if err := bc.blockStore.Close(); err != nil {
		bc.logger.Error("error closing block store", "err", err)
	}
}

func (bc *BlockChain) SetHead(block *types.Block) {
	bc.logger.Info("head block", "head", block.Header)
	bc.blockStore.WriteBlock(block)
	bc.blockStore.WriteHeadBlockHash(block.Hash())
	bc.latestBlock = block
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
		bc.logger.Error("failed to open block state", "err", err)
		return nil, err
	}
	if _, err := tree.LazyLoadVersionForOverwriting(bc.LatestBlock().Header.Height.Int64()); err != nil {
		bc.logger.Error("failed to open block state", "err", err)
		return nil, err
	}
	if tree.Version() != bc.LatestBlock().Header.Height.Int64() {
		bc.logger.Error("version mismatch", "stateVersion", tree.Version(), "latestBlock", bc.LatestBlock().Header.Height)
		return nil, errors.New("bad state version in db")
	}
	hash, err := tree.Hash()
	if err != nil {
		bc.logger.Error("failed to open block state", "err", err)
		return nil, err
	}
	if !bytes.Equal(hash, bc.LatestBlock().Header.Root) {
		bc.logger.Error("root hash mismatch", "state", hash, "latestBlock", bc.LatestBlock().Header.Root)
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

func (bc *BlockChain) Simulate(txs types.Txs) (types.Hash, error) {
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

	hash, err := transactor.ApplyTxs(state, block.Data.Txs)
	if err != nil {
		return err
	}
	if !bytes.Equal(block.Header.Root, hash) {
		state.Rollback()
		return fmt.Errorf("state hash mismatch, block root %v, got %v", block.Header.Root, hash)
	}

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
