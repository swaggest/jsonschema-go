package jsonschema_test

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jsonschema "github.com/swaggest/jsonschema-go/draft-07"
	"testing"
)

type MyStruct struct {
	FirstName string `json:"firstName" format:"date-time"`
	LastName  string `json:"lastName" required:"true"`
	Age       int    `json:"age"`
}

func TestGenerator_Parse(t *testing.T) {
	g := jsonschema.Generator{}
	schema, err := g.Parse(new(MyStruct))
	require.NoError(t, err)

	j, err := json.Marshal(schema)
	require.NoError(t, err)
	assert.Equal(t,
		`{"required":["lastName"],"properties":{"age":{"type":"integer"},"firstName":{"type":"string","format":"date-time"},"lastName":{"type":"string"}},"type":"object"}`,
		string(j),
	)
}
