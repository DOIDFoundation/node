package doid

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	doidapi "github.com/DOIDFoundation/node/doid/doidapi"
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/ethereum/go-ethereum/crypto"
)


type PublicTransactionPoolAPI struct {
	b doidapi.Backend
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

func NewPublicTransactionPoolAPI(b doidapi.Backend) *PublicTransactionPoolAPI{
	return &PublicTransactionPoolAPI{b}
}

func (api *PublicTransactionPoolAPI) SendTransaction(args TransactionArgs) (types.Hash, error) {
	if args.Owner == nil {
		return nil, errors.New("missing args: Owner")
	}
	if args.Signature == nil {
		return nil, errors.New("missing args: Signature")
	}

	nameHash :=  crypto.Keccak256([]byte(args.DOID))
	recovered,err := crypto.SigToPub(nameHash, args.Signature)
	if err != nil{
		return nil, errors.New("invalid args: Signature")
	}
	recoveredAddr := crypto.PubkeyToAddress(*recovered)
	if (!bytes.Equal(recoveredAddr.Bytes() , args.Owner.Bytes())){
		return nil, errors.New("invalid signature from owner")
	}
	
	register := tx.Register{DOID: args.DOID, Owner: args.Owner, Signature: args.Signature, NameHash: nameHash}
	encodeTx, _ := tx.NewTx(&register)
	api.b.SendTransaction(&encodeTx)

	// _, err = api.stateStore.Set(nameHash, args.Owner.Bytes())
	// if err != nil {
	// 	return nil, err
	// }
	// current := api.chain.CurrentBlock()
	// if current == nil {
	// 	current = types.NewBlockWithHeader(new(types.Header))
	// }
	// header := types.CopyHeader(current.Header)
	// header.Height.Add(header.Height, big.NewInt(1))
	// header.ParentHash = current.Hash()
	// header.Time = time.Now()
	// hash, err := api.stateStore.Commit()
	// if err != nil {
	// 	api.stateStore.Rollback()
	// 	return nil, err
	// }
	// header.Root = hash
	// block := types.NewBlockWithHeader(header)
	// api.chain.SetHead(block)
	return nil, nil
}

// func (api *PublicTransactionPoolAPI) GetOwner(params DOIDName) (string, error) {
// 	owner, err := api.b.Doid.StateStore().Get(crypto.Keccak256([]byte(params.DOID)))
// 	if err != nil {
// 		return "", err
// 	}
// 	ownerAddress := hex.EncodeToString(owner)
// 	return ownerAddress, nil
// }

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



// func RegisterAPI(node node.Node) error {
// 	homeDir := viper.GetString(cli.HomeFlag)
// 	db, err := cosmosdb.NewDB("state", cosmosdb.GoLevelDBBackend, filepath.Join(homeDir, "data"))
// 	if err != nil {
// 		return err
// 	}

// 	stateStore, err := store.NewStateStore(db)
// 	if err != nil {
// 		db.Close()
// 		return err
// 	}

// 	// NewPublicTransactionPoolAPI(node)
// 	api := &PublicTransactionPoolAPI{mempool: node.Mempool() ,chain: node.Chain(), stateStore: stateStore}
// 	rpc.RegisterName("doid", api)
// 	return nil
// }

// func RegisterAPI(name string, receiver interface {})error{
// 	homeDir := viper.GetString(cli.HomeFlag)
// 	db, err := cosmosdb.NewDB("state", cosmosdb.GoLevelDBBackend, filepath.Join(homeDir, "data"))
// 	if err != nil {
// 		return err
// 	}

// 	stateStore, err := store.NewStateStore(db)
// 	if err != nil {
// 		db.Close()
// 		return err
// 	}

// 	// NewPublicTransactionPoolAPI(node)
// 	re := reflect.ValueOf(receiver)
// 	if re.Kind() != reflect.Struct{
// 		return errors.New("invalid node type")
// 	}
// 	node := re.MethodByName("Node")
// 	ret := node.Call([]reflect.Value{})
	
// 	api := &PublicTransactionPoolAPI{mempool: node.Node().Mempool() ,chain: node.Chain(), stateStore: stateStore}
// 	rpc.RegisterName("doid", api)
// 	return nil

// }