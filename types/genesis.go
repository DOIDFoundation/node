package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func GenesisHeader(networkId int) *Header {
	switch networkId {
	case 1:
		panic("mainnet not launched, start with flag --testnet")
	default:
		return &Header{
			Difficulty: big.NewInt(0x1000000),
			Height:     big.NewInt(1), // starts from 1 because iavl can not go back to zero
			Root:       hexutil.MustDecode("0xE3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"),
			TxHash:     hexutil.MustDecode("0xE3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"),
		}
	}
}
