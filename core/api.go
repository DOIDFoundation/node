package core

import (
	"bytes"
	"context"
	"math/big"

	"github.com/DOIDFoundation/node/events"
	"github.com/DOIDFoundation/node/rpc"
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/encodedtx"
	"github.com/DOIDFoundation/node/types/tx"
	ethrpc "github.com/ethereum/go-ethereum/rpc"
)

type API struct {
	chain *BlockChain
}

type SubAPI struct {
	chain *BlockChain
}

func RegisterAPI(chain *BlockChain) {
	rpc.RegisterName("doid", &API{chain: chain})
	rpc.RegisterName("doid", &SubAPI{chain: chain})
}

func (a *API) GetBlockByHeight(height uint64) *types.Block {
	return a.chain.BlockByHeight(height)
}

func (a *API) GetBlockHashByHeight(height uint64) types.Hash {
	return a.chain.blockStore.ReadHashByHeight(height)
}

func (a *API) GetBlockByHash(hash types.Hash) *types.Block {
	return a.chain.blockStore.ReadBlock(*a.chain.blockStore.ReadHeightByHash(hash), hash)
}

func (a *API) GetBlockTD(hash types.Hash) *big.Int {
	return a.chain.blockStore.ReadTd(*a.chain.blockStore.ReadHeightByHash(hash), hash)
}

// func (a *API) GetTransactionByHash(hash types.Hash) types.Tx {
// 	return a.chain.blockStore.ReadTx(hash)
// }

type RpcTransaction struct{
	tx.Register
	Type types.TxType `json:"Type"`
	Hash types.Hash `json:"Hash"`
}

func newRpcTransaction(in types.Tx) *RpcTransaction{
	encTx ,err := encodedtx.FromBytes(in);
	if err != nil{
		return nil;
	}

	result := &RpcTransaction{};
	result.Hash = in.Hash();
	switch encTx.Type{
	case  types.TxTypeRegister:
		registerTx := &tx.Register{};
		err  = encTx.Decode(registerTx)
		if err != nil {
			return nil
		}
		result.DOID = registerTx.DOID;
		result.NameHash = registerTx.NameHash;
		result.Owner = registerTx.Owner;
		result.Type = types.TxTypeRegister;
	}

	return result;
}

func (a *API) GetTransactionByHash(hash types.Hash) *RpcTransaction {
	tx := a.chain.blockStore.ReadTx(hash)
	rpcTx := newRpcTransaction(tx);
	return rpcTx;
}

func (a *API) GetTransactionReceipt(hash types.Hash) *types.StoredReceipt {
	receipt := a.chain.blockStore.ReadReceipt(hash)
	if !bytes.Equal(a.chain.blockStore.ReadHashByHeight(receipt.BlockNumber.Uint64()), receipt.BlockHash) {
		return nil
	} else if block := a.chain.GetBlock(receipt.BlockNumber.Uint64(), receipt.BlockHash); block == nil {
		return nil
	} else if uint(len(block.Txs)) <= receipt.TransactionIndex {
		return nil
	} else if bytes.Equal(block.Txs[receipt.TransactionIndex].Hash(), receipt.TxHash) {
		return receipt
	}
	return nil
}

func (a *API) CurrentHeight() uint64 {
	return a.CurrentBlock().Header.Height.Uint64()
}

func (a *API) CurrentBlock() *types.Block {
	return a.chain.LatestBlock()
}

func (a *API) CurrentTD() *big.Int {
	return a.chain.GetTd()
}

// NewHeads send a notification each time a new (header) block is appended to the chain.
func (api *SubAPI) NewHeads(ctx context.Context) (*ethrpc.Subscription, error) {
	notifier, supported := ethrpc.NotifierFromContext(ctx)
	if !supported {
		return &ethrpc.Subscription{}, ethrpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		events.NewChainHead.Subscribe(string(rpcSub.ID), func(data *types.Block) {
			notifier.Notify(rpcSub.ID, data)
		})

	Wait:
		for {
			select {
			case <-rpcSub.Err():
				break Wait
			case <-notifier.Closed():
				break Wait
			}
		}
		events.NewChainHead.Unsubscribe(string(rpcSub.ID))
	}()

	return rpcSub, nil
}

// NewTransactions send a notification each time a new (header) block is appended to the chain.
func (api *SubAPI) NewTransactions(ctx context.Context) (*ethrpc.Subscription, error) {
	notifier, supported := ethrpc.NotifierFromContext(ctx)
	if !supported {
		return &ethrpc.Subscription{}, ethrpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		events.NewChainHead.Subscribe(string(rpcSub.ID), func(data *types.Block) {
			txs := data.Txs
			for _, tx := range txs {
				notifier.Notify(rpcSub.ID, tx.Hash())
			}

		})

	Wait:
		for {
			select {
			case <-rpcSub.Err():
				break Wait
			case <-notifier.Closed():
				break Wait
			}
		}
		events.NewChainHead.Unsubscribe(string(rpcSub.ID))
	}()

	return rpcSub, nil
}
