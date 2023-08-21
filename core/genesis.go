package core

import (
	"math/big"

	"github.com/DOIDFoundation/node/config"
	"github.com/DOIDFoundation/node/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func GenesisHeader(networkId config.Network) *types.Header {
	switch networkId {
	default:
		panic("network not supported, start with flag --testnet")
	case config.TestNet:
		return &types.Header{
			Difficulty: big.NewInt(0x1000000),
			Height:     big.NewInt(1), // starts from 1 because iavl can not go back to zero
			Root:       hexutil.MustDecode("0xE3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"),
			TxHash:     hexutil.MustDecode("0xE3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"),
		}
	}
}
