package tx

import (
	"encoding/json"
	"fmt"
	"reflect"

	bidmap "github.com/graytonio/go-bidirectional-map"
)

// Type of transaction.
//
// A uint8 for rlp serializing, but marshalled as string in JSON.
type Type uint8

// ----------------------------------------------------------------
// Modify these to add new type of transaction

// Transaction types, append only.
const (
	TypeRegister Type = iota
	TypeReserve  Type = 1
)

// Transaction types strings, for json marshalling/unmarshalling.
var typeStrings = bidmap.NewMap(map[Type]string{
	TypeRegister: "register",
	TypeReserve:  "reserve",
})

func NewTypedTx(t Type) TypedTx {
	switch t {
	case TypeRegister:
		return new(Register)
	case TypeReserve:
		return new(Reserve)
	default:
		return nil
	}
}

// ----------------------------------------------------------------
// Json marshalling

var txTypeT = reflect.TypeOf((*Type)(nil))

// MarshalText implements encoding.TextMarshaler.
func (t Type) MarshalText() ([]byte, error) {
	val, ok := typeStrings.GetP(t)
	if !ok {
		return nil, &lookupError{fmt.Sprintf("type string not found for tx type: %d", t)}
	}
	return []byte(val), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *Type) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return &json.UnmarshalTypeError{Value: "non-string", Type: txTypeT}
	}
	err := t.UnmarshalText(input[1 : len(input)-1])
	if _, ok := err.(*lookupError); ok {
		return &json.UnmarshalTypeError{Value: err.Error(), Type: txTypeT}
	}
	return err
}

// UnmarshalText implements encoding.TextUnmarshaler
func (t *Type) UnmarshalText(input []byte) error {
	val, ok := typeStrings.GetS(string(input))
	if !ok {
		return &lookupError{fmt.Sprintf("tx type not found for: %s", string(input))}
	}
	*t = val
	return nil
}

// String returns the type string.
func (t Type) String() string {
	val, _ := typeStrings.GetP(t)
	return val
}

type lookupError struct{ msg string }

func (err lookupError) Error() string { return err.msg }

func isString(input []byte) bool {
	return len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"'
}
