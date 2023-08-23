package transactor

import (
	"errors"

	"github.com/DOIDFoundation/node/doid"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/cosmos/iavl"
)

type Register struct{}

func (r *Register) Validate(state *iavl.ImmutableTree, t tx.TypedTx) error {
	args := t.(*tx.Register)
	if len(args.Owner) == 0 {
		return errors.New("missing args: Owner")
	}
	if args.Signature == nil {
		return errors.New("missing args: Signature")
	}

	if err := doid.ValidateDoidName(args.DOID, doid.ClassCNameLength); err != nil {
		return err
	}

	existsOwner, err := state.Get(doid.DOIDHash(args.DOID))
	if err != nil {
		return err
	}
	if existsOwner != nil {
		return errors.New("doidname has already been registered")
	}

	valid := doid.ValidateDoidNameSignatrue(args.DOID, args.Owner, args.Signature)
	if valid {
		return nil
	} else {
		return errors.New("invalid signature")
	}
}

func (r *Register) Apply(tree *iavl.MutableTree, t tx.TypedTx) (resultCode, error) {
	register, ok := t.(*tx.Register)
	if !ok {
		return resRejected, errors.New("bad tx type")
	}
	key := doid.DOIDHash(register.DOID)
	has, err := tree.Has(key)
	if err != nil {
		return resRejected, err
	}

	if has {
		return resRejected, errors.New("name exists")
	}

	_, err = tree.Set(key, register.Owner)
	if err != nil {
		return resRejected, err
	}

	err = doid.UpdateOwnerDOIDNames(tree, register.Owner, register.DOID, true)
	if err != nil {
		return resRejected, err
	}
	return resSuccess, nil
}

func init() {
	registerTransactor(tx.TypeRegister, &Register{})
	registerTransactor(tx.TypeReserve, &Reserve{})
	registerTransactor(tx.TypeUpdate, &Update{})
}
