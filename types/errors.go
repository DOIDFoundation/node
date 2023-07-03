package types

import "errors"

var (
	ErrNotContiguous = errors.New("block not contiguous")
	ErrFutureBlock   = errors.New("block in the future")
	ErrInvalidBlock  = errors.New("invalid block")
)
