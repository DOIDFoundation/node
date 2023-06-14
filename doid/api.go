package doid

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"time"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/flags"
	"github.com/DOIDFoundation/node/rpc"
	"github.com/DOIDFoundation/node/store"
	"github.com/DOIDFoundation/node/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/viper"
)

type PublicTransactionPoolAPI struct {
	chain      *core.BlockChain
	stateStore *store.StateStore
}

type TransactionArgs struct {
	DOID      string     `json:"DOID"`
	Owner     types.Hash `json:"owner"`
	Signature types.Hash `json:"signature"`
	Prv       types.Hash `json:"private"`
}

type DOIDName struct {
	DOID  string     `json:"DOID"`
	Owner types.Hash `json:"owner"`
}

func (api *PublicTransactionPoolAPI) SendTransaction(args TransactionArgs) (types.Hash, error) {
	if args.Owner == nil {
		return nil, errors.New("missing args: Owner")
	}
	if args.Signature == nil {
		return nil, errors.New("missing args: Signature")
	}

	nameHash := crypto.Keccak256([]byte(args.DOID))
	recovered, err := crypto.SigToPub(nameHash, args.Signature)
	if err != nil {
		return nil, errors.New("invalid args: Signature")
	}
	recoveredAddr := crypto.PubkeyToAddress(*recovered)
	if !bytes.Equal(recoveredAddr.Bytes(), args.Owner.Bytes()) {
		return nil, errors.New("invalid signature from owner")
	}
	_, err = api.stateStore.Set(nameHash, args.Owner.Bytes())
	if err != nil {
		return nil, err
	}
	current := api.chain.LatestBlock()
	if current == nil {
		current = types.NewBlockWithHeader(new(types.Header))
	}
	header := types.CopyHeader(current.Header)
	header.Height.Add(header.Height, big.NewInt(1))
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
	owner, err := api.stateStore.Get(crypto.Keccak256([]byte(params.DOID)))
	if err != nil {
		return "", err
	}
	ownerAddress := hex.EncodeToString(owner)
	return ownerAddress, nil
}

func (api *PublicTransactionPoolAPI) Sign(args TransactionArgs) (string, error) {
	nameHash := crypto.Keccak256([]byte(args.DOID))
	prv, err := crypto.HexToECDSA(args.Prv.String())
	if err != nil {
		return "", errors.New("invalid prv" + err.Error())
	}
	sig, err := crypto.Sign(nameHash, prv)
	fmt.Println(sig)
	if err != nil {
		return "", errors.New(err.Error())
	}
	sigStr := hex.EncodeToString(sig)
	return sigStr, nil
}

func RegisterAPI(chain *core.BlockChain) error {
	homeDir := viper.GetString(flags.Home)
	db, err := cosmosdb.NewDB("state", cosmosdb.GoLevelDBBackend, filepath.Join(homeDir, "data"))
	if err != nil {
		return err
	}

	stateStore, err := store.NewStateStore(db)
	if err != nil {
		db.Close()
		return err
	}

	api := &PublicTransactionPoolAPI{chain: chain, stateStore: stateStore}
	rpc.RegisterName("doid", api)
	return nil
}
