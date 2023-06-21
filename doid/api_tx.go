package doid

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/ethereum/go-ethereum/crypto"
)

type TransactionArgs struct {
	DOID      string     `json:"DOID"`
	Owner     types.Hash `json:"owner"`
	Signature types.Hash `json:"signature"`
	Prv       types.Hash `json:"private"`
}

type PublicTransactionPoolAPI struct {
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

	register := tx.Register{DOID: args.DOID, Owner: args.Owner, Signature: args.Signature, NameHash: nameHash}
	t, err := tx.NewTx(&register)
	if err != nil {
		return nil, err
	}
	core.EventInstance().FireEvent(types.EventNewTx, &t)
	return t.Hash(), nil
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
