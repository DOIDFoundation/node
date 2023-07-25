package doid

import (
	"bytes"
	"encoding/hex"
	"errors"

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

	nameHash := crypto.Keccak256([]byte(args.DOID))
	recovered, err := crypto.SigToPub(nameHash, args.Signature)
	if err != nil {
		return nil, errors.New("invalid args: Signature")
	}
	recoveredAddr := crypto.PubkeyToAddress(*recovered)
	if !bytes.Equal(recoveredAddr.Bytes(), args.From.Bytes()) {
		return nil, errors.New("invalid signature")
	}

	register := tx.Register{DOID: args.DOID, Owner: args.Owner, Signature: args.Signature, NameHash: nameHash, From: args.From}
	t, err := tx.NewTx(&register)
	if err != nil {
		return nil, err
	}
	events.NewTx.Send(t)
	return t.Hash(), nil
}

func (api *PublicTransactionPoolAPI) Sign(args TransactionArgs) (string, error) {
	nameHash := crypto.Keccak256([]byte(args.DOID))
	prv, err := crypto.HexToECDSA(args.Prv.String())
	if err != nil {
		return "", errors.New("invalid prv" + err.Error())
	}
	sig, err := crypto.Sign(nameHash, prv)
	if err != nil {
		return "", errors.New(err.Error())
	}
	sigStr := hex.EncodeToString(sig)
	return sigStr, nil
}
