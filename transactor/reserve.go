package transactor

import (
	"bytes"
	"errors"

	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/cosmos/iavl"
	"github.com/ethereum/go-ethereum/crypto"
)

var admin = types.HexToAddress("f39Fd6e51aad88F6F4ce6aB8827279cffFb92266")

type Reserve struct{}

func (r *Reserve) Validate(state *iavl.ImmutableTree, t tx.TypedTx) error {
	args := t.(*tx.Reserve)
	if len(args.Admin) == 0 {
		return errors.New("missing args: Owner")
	}
	if args.Signature == nil {
		return errors.New("missing args: Signature")
	}
	if !bytes.Equal(args.Admin.Bytes(), admin.Bytes()) {
		return errors.New("invalid admin address")
	}

	existsOwner, err := state.Get(types.DOIDHash(args.DOID))
	if err != nil {
		return err
	}
	if existsOwner != nil {
		return errors.New("doidname has already been registered")
	}

	message := crypto.Keccak256(append([]byte(args.DOID), args.Admin...))
	recovered, err := crypto.SigToPub(message, args.Signature)
	if err != nil {
		return errors.New("invalid args: Signature")
	}
	recoveredAddr := crypto.PubkeyToAddress(*recovered)
	if !bytes.Equal(recoveredAddr.Bytes(), args.Admin.Bytes()) {
		return errors.New("invalid signature")
	}
	return nil
}

func (r *Reserve) Apply(tree *iavl.MutableTree, t tx.TypedTx) (resultCode, error) {
	reserve, ok := t.(*tx.Reserve)
	if !ok {
		return resRejected, errors.New("bad tx type")
	}
	key := types.DOIDHash(reserve.DOID)
	has, err := tree.Has(key)
	if err != nil {
		return resRejected, err
	}

	if has {
		return resRejected, errors.New("name exists")
	}

	_, err = tree.Set(key, reserve.Owner)
	if err != nil {
		return resRejected, err
	}
	return resSuccess, nil
}
