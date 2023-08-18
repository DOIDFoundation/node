package transactor

import (
	"bytes"
	"errors"

	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/cosmos/iavl"
)

type Update struct{}

func (u *Update) Validate(state *iavl.ImmutableTree, t tx.TypedTx) error {
	args := t.(*tx.Update)
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

	valid := ValidateDoidNameSignatrue(args.DOID, existsOwner, args.Signature)
	if valid {
		return nil
	} else {
		return errors.New("invalid signature")
	}
}

func (u *Update) Apply(tree *iavl.MutableTree, t tx.TypedTx) (resultCode, error) {
	update, ok := t.(*tx.Update)
	if !ok {
		return resRejected, errors.New("bad tx type")
	}
	key := types.DOIDHash(update.DOID)
	has, err := tree.Has(key)
	if err != nil {
		return resRejected, err
	}

	if !has {
		return resRejected, errors.New("name not exists")
	}
	ownerBefore, err := tree.Get(key)
	if err != nil {
		return resRejected, err
	}
	err = types.UpdateOwnerDOIDNames(tree, ownerBefore, update.DOID, false)
	if err != nil {
		return resRejected, err
	}
	err = types.UpdateOwnerDOIDNames(tree, update.Owner, update.DOID, true)
	if err != nil {
		return resRejected, err
	}

	_, err = tree.Set(key, update.Owner)
	if err != nil {
		return resRejected, err
	}

	return resSuccess, nil
}
