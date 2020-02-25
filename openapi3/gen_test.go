package openapi3_test

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/swaggest/jsonschema-go/openapi3"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestGenerator_SetResponse(t *testing.T) {
	type Req struct {
		InQuery  int     `query:"in_query" required:"true" description:"Query parameter."`
		InPath   int     `path:"in_path"`
		InCookie string  `cookie:"in_cookie" deprecated:"true"`
		InHeader float64 `header:"in_header"`
		InBody1  int     `json:"in_body1"`
		InBody2  string  `json:"in_body2"`
	}

	type Resp struct {
		Field1 int    `json:"field1"`
		Field2 string `json:"field2"`
		Parent *Resp  `json:"parent"`
	}

	g := openapi3.Generator{}

	s := openapi3.Spec{}
	s.Info.Title = "SampleAPI"
	s.Info.Version = "1.2.3"

	g.Spec = &s

	op := openapi3.Operation{}

	//op.WithDeprecated(true)

	err := g.SetRequest(&op, new(Req))
	assert.NoError(t, err)

	err = g.SetResponse(&op, new(Resp))
	assert.NoError(t, err)

	s.Paths.WithMapOfPathItemValuesItem(
		"/somewhere/{in_path}",
		*((&openapi3.PathItem{}).
			WithSummary("Path Summary").
			WithDescription("Path Description").
			WithOperation(http.MethodGet, op)),
	)

	b, err := json.MarshalIndent(s, "", " ")
	assert.NoError(t, err)

	ioutil.WriteFile("openapi.json", b, 0640)
	println(string(b))
}
