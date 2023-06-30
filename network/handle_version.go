package network

import (
	"bytes"
	"encoding/gob"
)

type version struct {
	//Version  byte
	Height   int64
	AddrFrom string
}

func (v version) serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(v)
	if err != nil {
		panic(err)
	}
	return result.Bytes()
}

func (v *version) deserialize(d []byte) {
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(v)
	if err != nil {
		panic(err)
	}

}
