package main

import (
	"bytes"
	"log"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/iavl"
)

type KVStoreApplication struct {
	db   *cosmosdb.DB
	tree *iavl.MutableTree
}

var _ abcitypes.Application = (*KVStoreApplication)(nil)

func NewKVStoreApplication(db *cosmosdb.DB) *KVStoreApplication {
	tree, err := iavl.NewMutableTree(*db, 128, false)
	if err != nil {
		log.Panicf("error creating iavl multable tree: %v", err)
	}
	return &KVStoreApplication{db: db, tree: tree}
}

func (app *KVStoreApplication) Info(info abcitypes.RequestInfo) abcitypes.ResponseInfo {
	return abcitypes.ResponseInfo{}
}

func (app *KVStoreApplication) Query(req abcitypes.RequestQuery) abcitypes.ResponseQuery {
	resp := abcitypes.ResponseQuery{Key: req.Data}
	item, err := app.tree.Get(req.Data)
	if item != nil {
		resp.Log = "exists"
		resp.Value = item
	} else if err != nil {
		resp.Log = err.Error()
	} else {
		resp.Log = "key does not exist"
	}
	return resp
}

func (app *KVStoreApplication) CheckTx(req abcitypes.RequestCheckTx) abcitypes.ResponseCheckTx {
	code := app.isValid(req.Tx)
	return abcitypes.ResponseCheckTx{Code: code}
}

func (app *KVStoreApplication) isValid(tx []byte) uint32 {
	// check format
	parts := bytes.Split(tx, []byte("="))
	if len(parts) != 2 {
		return 1
	}

	return 0
}

func (app *KVStoreApplication) InitChain(chain abcitypes.RequestInitChain) abcitypes.ResponseInitChain {
	return abcitypes.ResponseInitChain{}
}

func (app *KVStoreApplication) PrepareProposal(proposal abcitypes.RequestPrepareProposal) abcitypes.ResponsePrepareProposal {
	return abcitypes.ResponsePrepareProposal{Txs: proposal.Txs}
}

func (app *KVStoreApplication) ProcessProposal(proposal abcitypes.RequestProcessProposal) abcitypes.ResponseProcessProposal {
	return abcitypes.ResponseProcessProposal{Status: abcitypes.ResponseProcessProposal_ACCEPT}
}

func (app *KVStoreApplication) BeginBlock(block abcitypes.RequestBeginBlock) abcitypes.ResponseBeginBlock {
	return abcitypes.ResponseBeginBlock{}
}

func (app *KVStoreApplication) DeliverTx(req abcitypes.RequestDeliverTx) abcitypes.ResponseDeliverTx {
	if code := app.isValid(req.Tx); code != 0 {
		return abcitypes.ResponseDeliverTx{Code: code}
	}

	parts := bytes.SplitN(req.Tx, []byte("="), 2)
	key, value := parts[0], parts[1]

	if _, err := app.tree.Set(key, value); err != nil {
		log.Panicf("Error writing to database, unable to execute tx: %v", err)
	}

	return abcitypes.ResponseDeliverTx{Code: 0}
}

func (app *KVStoreApplication) EndBlock(block abcitypes.RequestEndBlock) abcitypes.ResponseEndBlock {
	return abcitypes.ResponseEndBlock{}
}

func (app *KVStoreApplication) Commit() abcitypes.ResponseCommit {
	hash, _, err := app.tree.SaveVersion()
	if err != nil {
		log.Panicf("Error writing to database, unable to commit block: %v", err)
	}
	return abcitypes.ResponseCommit{Data: hash}
}

func (app *KVStoreApplication) ListSnapshots(snapshots abcitypes.RequestListSnapshots) abcitypes.ResponseListSnapshots {
	return abcitypes.ResponseListSnapshots{}
}

func (app *KVStoreApplication) OfferSnapshot(snapshot abcitypes.RequestOfferSnapshot) abcitypes.ResponseOfferSnapshot {
	return abcitypes.ResponseOfferSnapshot{}
}

func (app *KVStoreApplication) LoadSnapshotChunk(chunk abcitypes.RequestLoadSnapshotChunk) abcitypes.ResponseLoadSnapshotChunk {
	return abcitypes.ResponseLoadSnapshotChunk{}
}

func (app *KVStoreApplication) ApplySnapshotChunk(chunk abcitypes.RequestApplySnapshotChunk) abcitypes.ResponseApplySnapshotChunk {
	return abcitypes.ResponseApplySnapshotChunk{}
}
