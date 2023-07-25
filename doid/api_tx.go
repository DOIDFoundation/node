package doid

import (
	"bytes"
	"encoding/hex"
	"errors"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/events"
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/ethereum/go-ethereum/crypto"
)

type TransactionArgs struct {
	DOID      string     `json:"DOID"`
	Owner     types.Hash `json:"owner"` // owner of doid
	From      types.Hash `json:"from"`  // signing address
	Signature types.Hash `json:"signature"`
	Prv       types.Hash `json:"private"`
}

type PublicTransactionPoolAPI struct {
	chain *core.BlockChain
}

func (api *PublicTransactionPoolAPI) SendTransaction(args TransactionArgs) (types.Hash, error) {
	if args.Owner == nil {
		return nil, errors.New("missing args: Owner")
	}
	if args.From == nil {
		return nil, errors.New("missing args: From")
	}
	if args.Signature == nil {
		return nil, errors.New("missing args: Signature")
	}

	state, err := api.chain.LatestState()
	if err != nil {
		return nil, err
	}
	existsOwner, err := state.Get(types.DOIDHash(args.DOID))
	if err != nil {
		return nil, err
	}
	if existsOwner != nil {
		return nil, errors.New("doidname has already been registered")
	}

	message := crypto.Keccak256(append([]byte(args.DOID), args.Owner...))
	recovered, err := crypto.SigToPub(message, args.Signature)
	if err != nil {
		return nil, errors.New("invalid args: Signature")
	}
	recoveredAddr := crypto.PubkeyToAddress(*recovered)
	if !bytes.Equal(recoveredAddr.Bytes(), args.From.Bytes()) {
		return nil, errors.New("invalid signature")
	}

	nameHash := crypto.Keccak256([]byte(args.DOID))
	register := tx.Register{DOID: args.DOID, Owner: args.Owner, Signature: args.Signature, NameHash: nameHash, From: args.From}
	t, err := tx.NewTx(&register)
	if err != nil {
		return nil, err
	}
	events.NewTx.Send(t)
	return t.Hash(), nil
}

func (api *PublicTransactionPoolAPI) Sign(args TransactionArgs) (string, error) {
	message := crypto.Keccak256((append([]byte(args.DOID), args.Owner...)))
	prv, err := crypto.HexToECDSA(args.Prv.String())
	if err != nil {
		return "", errors.New("invalid prv" + err.Error())
	}
	sig, err := crypto.Sign(message, prv)
	if err != nil {
		return "", errors.New(err.Error())
	}
	sigStr := hex.EncodeToString(sig)
	return sigStr, nil
}
