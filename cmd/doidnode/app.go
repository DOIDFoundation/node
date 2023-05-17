package main

import (
	"bytes"
	"fmt"
	"log"
	"strconv"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	crypto "github.com/cometbft/cometbft/crypto"

	"github.com/dgraph-io/badger/v3"
)

type KVStoreApplication struct {
	db           *badger.DB
	onGoingBlock *badger.Txn
}

type DoidTransactionType struct {
	Owner crypto.Address `json:"owner"`
	Name string `json:"name"`
	Code uint64 `json:"code"`
	Signature []byte    `json:"signature"`
}


var _ abcitypes.Application = (*KVStoreApplication)(nil)

func NewKVStoreApplication(db *badger.DB) *KVStoreApplication {
	return &KVStoreApplication{db: db}
}

func (app *KVStoreApplication) Info(info abcitypes.RequestInfo) abcitypes.ResponseInfo {
	return abcitypes.ResponseInfo{}
}

func (app *KVStoreApplication) Query(req abcitypes.RequestQuery) abcitypes.ResponseQuery {
	resp := abcitypes.ResponseQuery{Key: req.Data}

	dbErr := app.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(req.Data)
		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}
			resp.Log = "key does not exist"
			return nil
		}

		return item.Value(func(val []byte) error {
			resp.Log = "exists"
			resp.Value = val
			return nil
		})
	})
	if dbErr != nil {
		log.Panicf("Error reading database, unable to execute query: %v", dbErr)
	}
	return resp
}

func (app *KVStoreApplication) CheckTx(req abcitypes.RequestCheckTx) abcitypes.ResponseCheckTx {

	code := app.isValid(req.Tx)
	return abcitypes.ResponseCheckTx{Code: code}
}

func (app *KVStoreApplication) isValid(tx []byte) uint32 {
	// check format
	pairs := bytes.Split(tx, []byte(","))
	if len(pairs) <= 0 {
		return 1
	}

	doidTx := DoidTransactionType{}
	for i := 0; i < len(pairs); i++ {
		kv := bytes.Split(pairs[i], []byte("="));
		if len(kv) != 2 {
			return 1;
		}
		key,value := kv[0], kv[1];
		switch string(key) {
        case "owner":
           doidTx.Owner = value
        case "name":
            doidTx.Name = string((value));   
        case "signature":     
           doidTx.Signature = value  
		case "code":
			doidTx.Code, _ =  strconv.ParseUint(string(value), 10, 64)
        }
	}
	fmt.Println("----------------", doidTx.Code, doidTx.Name, doidTx.Owner, doidTx.Signature)

	// TODO: check signature

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
	app.onGoingBlock = app.db.NewTransaction(true)
	return abcitypes.ResponseBeginBlock{}
}

func (app *KVStoreApplication) DeliverTx(req abcitypes.RequestDeliverTx) abcitypes.ResponseDeliverTx {
	if code := app.isValid(req.Tx); code != 0 {
		return abcitypes.ResponseDeliverTx{Code: code}
	}

	parts := bytes.SplitN(req.Tx, []byte("="), 2)
	key, value := parts[0], parts[1]

	if err := app.onGoingBlock.Set(key, value); err != nil {
		log.Panicf("Error writing to database, unable to execute tx: %v", err)
	}

	return abcitypes.ResponseDeliverTx{Code: 0}
}

func (app *KVStoreApplication) EndBlock(block abcitypes.RequestEndBlock) abcitypes.ResponseEndBlock {
	return abcitypes.ResponseEndBlock{}
}

func (app *KVStoreApplication) Commit() abcitypes.ResponseCommit {
	if err := app.onGoingBlock.Commit(); err != nil {
		log.Panicf("Error writing to database, unable to commit block: %v", err)
	}
	return abcitypes.ResponseCommit{Data: []byte{}}
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
