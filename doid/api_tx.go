package doid

import (
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/DOIDFoundation/node/core"
	"github.com/DOIDFoundation/node/events"
	"github.com/DOIDFoundation/node/transactor"
	"github.com/DOIDFoundation/node/types"
	"github.com/DOIDFoundation/node/types/tx"
	"github.com/ethereum/go-ethereum/crypto"
)

type PublicTransactionPoolAPI struct {
	chain *core.BlockChain
}

func (api *PublicTransactionPoolAPI) SendTransaction(input json.RawMessage) (types.Hash, error) {
	args, err := tx.UnmarshalJSON(input)
	if err != nil {
		return nil, err
	}

	state, err := api.chain.LatestState()
	if err != nil {
		return nil, err
	}

	if err := transactor.ValidateTx(state, args); err != nil {
		return nil, err
	}

	t, err := tx.NewTx(args)
	if err != nil {
		return nil, err
	}

	events.NewTx.Send(t)
	return t.Hash(), nil
}

func (api *PublicTransactionPoolAPI) Sign(input json.RawMessage) (string, error) {
	var args tx.Register
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	message := crypto.Keccak256((append([]byte(args.DOID), args.Owner...)))
	var priv struct {
		Prv types.HexBytes `json:"Prv" gencodec:"required"`
	}
	if err := json.Unmarshal(input, &priv); err != nil {
		return "", err
	}
	prv, err := crypto.HexToECDSA(priv.Prv.String())
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
