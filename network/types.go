package network

import "math/big"

type BlockInfo struct {
	Height *big.Int `json:"height"           gencodec:"required"`
}

type BlockHeight struct {
	Height *big.Int `json:"height"           gencodec:"required"`
}
