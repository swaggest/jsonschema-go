package openapi3_test

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/swaggest/jsonschema-go/openapi3"
	"net/http"
	"testing"
)

func TestGenerator_SetResponse(t *testing.T) {
	type Req struct {
		InQuery int    `query:"in_query"`
		InPath  int    `path:"in_path"`
		InBody1 int    `json:"in_body1"`
		InBody2 string `json:"in_body2"`
	}

	type Resp struct {
		Field1 int    `json:"field1"`
		Field2 string `json:"field2"`
	}

	g := openapi3.Generator{}

	s := openapi3.Spec{}
	s.Info = &openapi3.Info{}
	s.Info.WithTitle("SampleAPI")

	g.Spec = &s

	op := openapi3.Operation{}

	op.WithDeprecated(true)

	err := g.SetRequest(&op, new(Req))
	assert.NoError(t, err)

	err = g.SetResponse(&op, new(Resp))
	assert.NoError(t, err)

	s.Paths = &openapi3.Paths{}

	s.Paths.WithMapOfPathItemValuesItem(
		"/somewhere/{in_path}",
		*((&openapi3.PathItem{}).
			WithSummary("Path Summary").
			WithDescription("Path Description").
			WithMapOfOperationValuesItem(http.MethodGet, op)),
	)

	b, err := json.MarshalIndent(s, "", " ")
	assert.NoError(t, err)

	println(string(b))
}
