package tx

import (
	"encoding/json"
	"errors"
)

func MarshalJSON(tx TypedTx) ([]byte, error) {
	jsonTx := struct {
		Type Type    `json:"type"`
		Data TypedTx `json:"data"`
	}{Type: tx.Type(), Data: tx}
	return json.Marshal(jsonTx)
}

func UnmarshalJSON(data []byte) (TypedTx, error) {
	var ret struct {
		Type json.RawMessage `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, err
	}
	if len(ret.Type) == 0 {
		return nil, errors.New("missing field: type")
	}
	if len(ret.Data) == 0 {
		return nil, errors.New("missing field: data")
	}
	var txType Type
	if err := txType.UnmarshalJSON(ret.Type); err != nil {
		return nil, err
	}
	t := NewTypedTx(txType)
	if err := json.Unmarshal(ret.Data, t); err != nil {
		return nil, err
	}
	return t, nil
}
