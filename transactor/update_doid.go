package transactor

import (
	"bytes"
	"errors"

	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/cosmos/iavl"
	"github.com/ethereum/go-ethereum/crypto"
)

type UpdateDOID struct{}

func (u *UpdateDOID) Validate(state *iavl.ImmutableTree, t tx.TypedTx) error {
	args := t.(*tx.UpdateDOID)
	if len(args.Owner) == 0 {
		return errors.New("missing args: Owner")
	}
	if args.Signature == nil {
		return errors.New("missing args: Signature")
	}

	existsOwner, err := state.Get(types.DOIDHash(args.DOID))
	if err != nil {
		return err
	}
	if existsOwner == nil {
		return errors.New("doidname hasn't been registered")
	}
	if bytes.Equal(existsOwner, args.Owner) {
		return errors.New("update to same owner")
	}

	message := crypto.Keccak256(append([]byte(args.DOID), args.Owner...))
	recovered, err := crypto.SigToPub(message, args.Signature)
	if err != nil {
		return errors.New("invalid args: Signature")
	}
	recoveredAddr := crypto.PubkeyToAddress(*recovered)
	if !bytes.Equal(recoveredAddr.Bytes(), args.Owner.Bytes()) {
		return errors.New("invalid signature")
	}
	return nil
}

func (u *UpdateDOID) Apply(tree *iavl.MutableTree, t tx.TypedTx) (resultCode, error) {
	reserve, ok := t.(*tx.Reserve)
	if !ok {
		return resRejected, errors.New("bad tx type")
	}
	key := types.DOIDHash(reserve.DOID)
	has, err := tree.Has(key)
	if err != nil {
		return resRejected, err
	}

	if !has {
		return resRejected, errors.New("name not exists")
	}

	_, err = tree.Set(key, reserve.Owner)
	if err != nil {
		return resRejected, err
	}
	return resSuccess, nil
}
