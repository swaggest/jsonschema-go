//go:build go1.18
// +build go1.18

package jsonschema_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
)

func TestReflector_Reflect_generic(t *testing.T) {
	type helloOutput struct {
		Now     time.Time `header:"X-Now" json:"-"`
		Message string    `json:"message"`
	}

	type helloOutput2 struct {
		Bar string `json:"bar"`
	}

	type APIResponse[T any] struct {
		Data *T `json:"data"`
	}

	var ar struct {
		Foo APIResponse[helloOutput]  `json:"foo"`
		Bar APIResponse[helloOutput2] `json:"bar"`
	}

	r := jsonschema.Reflector{}

	s, err := r.Reflect(ar, jsonschema.StripDefinitionNamePrefix("JsonschemaGoTest"))
	require.NoError(t, err)
	assertjson.EqualMarshal(t, []byte(`{
	  "definitions":{
		"APIResponse[HelloOutput2]":{
		  "properties":{"data":{"$ref":"#/definitions/HelloOutput2"}},
		  "type":"object"
		},
		"APIResponse[HelloOutput]":{
		  "properties":{"data":{"$ref":"#/definitions/HelloOutput"}},
		  "type":"object"
		},
		"HelloOutput":{"properties":{"message":{"type":"string"}},"type":"object"},
		"HelloOutput2":{"properties":{"bar":{"type":"string"}},"type":"object"}
	  },
	  "properties":{
		"bar":{"$ref":"#/definitions/APIResponse[HelloOutput2]"},
		"foo":{"$ref":"#/definitions/APIResponse[HelloOutput]"}
	  },
	  "type":"object"
	}`), s)

	r = jsonschema.Reflector{}
	s, err = r.Reflect(ar)
	require.NoError(t, err)
	assertjson.EqualMarshal(t, []byte(`{
	  "definitions":{
		"JsonschemaGoTestAPIResponse[JsonschemaGoTestHelloOutput2]":{
		  "properties":{"data":{"$ref":"#/definitions/JsonschemaGoTestHelloOutput2"}},
		  "type":"object"
		},
		"JsonschemaGoTestAPIResponse[JsonschemaGoTestHelloOutput]":{
		  "properties":{"data":{"$ref":"#/definitions/JsonschemaGoTestHelloOutput"}},
		  "type":"object"
		},
		"JsonschemaGoTestHelloOutput":{"properties":{"message":{"type":"string"}},"type":"object"},
		"JsonschemaGoTestHelloOutput2":{"properties":{"bar":{"type":"string"}},"type":"object"}
	  },
	  "properties":{
		"bar":{
		  "$ref":"#/definitions/JsonschemaGoTestAPIResponse[JsonschemaGoTestHelloOutput2]"
		},
		"foo":{
		  "$ref":"#/definitions/JsonschemaGoTestAPIResponse[JsonschemaGoTestHelloOutput]"
		}
	  },
	  "type":"object"
	}`), s)
}
