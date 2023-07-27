package transactor

import (
	"bytes"
	"errors"

	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/cosmos/iavl"
	"github.com/ethereum/go-ethereum/crypto"
)

type Register struct{}

func (r *Register) Validate(state *iavl.ImmutableTree, t tx.TypedTx) error {
	args := t.(*tx.Register)
	if len(args.Owner) == 0 {
		return errors.New("missing args: Owner")
	}
	if args.From == nil {
		return errors.New("missing args: From")
	}
	if args.Signature == nil {
		return errors.New("missing args: Signature")
	}

	existsOwner, err := state.Get(types.DOIDHash(args.DOID))
	if err != nil {
		return err
	}
	if existsOwner != nil {
		return errors.New("doidname has already been registered")
	}

	message := crypto.Keccak256(append([]byte(args.DOID), args.Owner...))
	recovered, err := crypto.SigToPub(message, args.Signature)
	if err != nil {
		return errors.New("invalid args: Signature")
	}
	recoveredAddr := crypto.PubkeyToAddress(*recovered)
	if !bytes.Equal(recoveredAddr.Bytes(), args.From.Bytes()) {
		return errors.New("invalid signature")
	}
	return nil
}

func (r *Register) Apply(tree *iavl.MutableTree, t tx.TypedTx) (resultCode, error) {
	register, ok := t.(*tx.Register)
	if !ok {
		return resRejected, errors.New("bad tx type")
	}
	key := types.DOIDHash(register.DOID)
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
	return resSuccess, nil
}

func init() {
	registerTransactor(tx.TypeRegister, &Register{})
}
