package jsonschema_test

import (
	"github.com/stretchr/testify/require"
	jsonschema "github.com/swaggest/jsonschema-go/draft-07"
	"testing"
)

type MyStruct struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName" required:"true"`
	Age       int    `json:"age"`
}

func TestGenerator_Parse(t *testing.T) {
	g := jsonschema.Generator{}
	_, err := g.Parse(new(MyStruct))
	require.NoError(t, err)

}
