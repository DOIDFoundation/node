package doid

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"time"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/rpc"
	"github.com/DOIDFoundation/node/store"
	"github.com/DOIDFoundation/node/types"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/libs/cli"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/viper"
)

type PublicTransactionPoolAPI struct {
	chain      *core.BlockChain
	stateStore *store.StateStore
}

type TransactionArgs struct {
	DOID  string            `json:"DOID"`
	Owner cmtbytes.HexBytes `json:"owner"`
	Signature cmtbytes.HexBytes `json:"signature"`
	Prv cmtbytes.HexBytes `json:"private"`
}

type DOIDName struct {
	DOID  string            `json:"DOID"`
	Owner cmtbytes.HexBytes `json:"owner"`
}

func (api *PublicTransactionPoolAPI) SendTransaction(args TransactionArgs) (cmtbytes.HexBytes, error) {
	if args.Owner == nil {
		return nil, errors.New("missing args: Owner")
	}
	if args.Signature == nil{
		return nil, errors.New("missing args: Signature")
	}
	nameHash :=  crypto.Keccak256([]byte(args.DOID))

	if(!crypto.VerifySignature(args.Owner.Bytes(), nameHash,  args.Signature[:len(args.Signature)-1])){
		return nil, errors.New("invalid signature from owner")
	}
	_, err := api.stateStore.Set(nameHash, args.Owner.Bytes())
	if err != nil {
		return nil, err
	}
	current := api.chain.CurrentBlock()
	if current == nil {
		current = types.NewBlockWithHeader(new(types.Header))
	}
	header := types.CopyHeader(current.Header)
	header.Height = header.Height.Add(header.Height, big.NewInt(1))
	header.ParentHash = current.Hash()
	header.Time = time.Now()
	hash, err := api.stateStore.Commit()
	if err != nil {
		api.stateStore.Rollback()
		return nil, err
	}
	header.Root = hash
	block := types.NewBlockWithHeader(header)
	api.chain.SetHead(block)
	return nil, nil
}

func (api *PublicTransactionPoolAPI) GetOwner(params DOIDName) (string, error) {
	owner, err :=api.stateStore.Get(crypto.Keccak256([]byte(params.DOID)))
	if err != nil{
		return "", err
	}
	fmt.Println(owner, params.DOID)
	return string(owner), nil
}

func (api *PublicTransactionPoolAPI) Sign(args TransactionArgs) (string, error) {
	nameHash :=  crypto.Keccak256([]byte(args.DOID))
	prv, err := crypto.HexToECDSA(args.Prv.String())
	if err!= nil{
		return "", errors.New("invalid prv"+ err.Error())
	}
	sig, err := crypto.Sign(nameHash, prv)
	fmt.Println(sig)
	if err != nil{
		return "", errors.New(err.Error())
	}
	sigStr := hex.EncodeToString(sig)
	return sigStr, nil
}

func RegisterAPI(chain *core.BlockChain) {
	homeDir := viper.GetString(cli.HomeFlag)
	db, err := cosmosdb.NewDB("state", cosmosdb.GoLevelDBBackend, filepath.Join(homeDir, "data"))
	if err != nil {
		return
	}

	stateStore, err := store.NewStateStore(db)
	if err != nil {
		db.Close()
		return
	}

	api := &PublicTransactionPoolAPI{chain: chain, stateStore: stateStore}
	rpc.RegisterName("doid", api)
}
