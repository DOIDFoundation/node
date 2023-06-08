package types

import (
	cmttypes "github.com/cometbft/cometbft/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type Address = cmttypes.Address
type BlockNonce = ethtypes.BlockNonce
type Data = cmttypes.Data
type Tx = cmttypes.Tx

type TxType = uint8

// Transaction types, append only.
const (
	TxTypeRegister TxType = iota
)

type TypedTx interface {
	Type() TxType
}

var EncodeNonce = ethtypes.EncodeNonce
