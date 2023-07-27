package tx_test

import (
	"encoding/json"
	"testing"

	"github.com/DOIDFoundation/node/types/tx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterJSON(t *testing.T) {
	r := tx.Register{DOID: "test"}
	b, err := tx.MarshalJSON(&r)
	require.NoError(t, err)

	f := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(b, &f))
	assert.Equal(t, r.Type().String(), f["type"])

	ret, err := tx.UnmarshalJSON(b)
	assert.NoError(t, err)
	if assert.NotNil(t, ret.(*tx.Register)) {
		assert.Equal(t, r.DOID, ret.(*tx.Register).DOID)
	}
}
