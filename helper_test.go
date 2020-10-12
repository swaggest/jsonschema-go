package jsonschema_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/jsonschema-go"
)

func TestSchemaOrBool_JSONSchemaBytes(t *testing.T) {
	s := jsonschema.Schema{}
	s.AddType(jsonschema.String)

	b, err := s.ToSchemaOrBool().JSONSchemaBytes()
	require.NoError(t, err)
	assert.Equal(t, `{"type":"string"}`, string(b))

	b, err = s.JSONSchemaBytes()
	require.NoError(t, err)
	assert.Equal(t, `{"type":"string"}`, string(b))

	m, err := s.ToSchemaOrBool().ToSimpleMap()
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"type": "string"}, m)

	sbf := jsonschema.SchemaOrBool{}
	sbf.WithTypeBoolean(false)
	m, err = sbf.ToSimpleMap()
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"not": map[string]interface{}{}}, m)

	sbt := jsonschema.SchemaOrBool{}
	sbt.WithTypeBoolean(true)
	m, err = sbt.ToSimpleMap()
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{}, m)
}

func TestSchema_IsTrivial(t *testing.T) {
	for _, s := range []struct {
		isTrivial bool
		name      string
		schema    string
	}{
		{true, "true schema", "true"},
		{false, "false schema", "false"},
		{true, "empty schema", "{}"},
		{true, "type object", `{"type":"object", "additionalProperties":{"type":"integer"}}`},
		{
			false, "type object with non-trivial members",
			`{"type":"object", "additionalProperties":{"type":"integer","minimum":3}}`,
		},
		{
			true, "type object with properties",
			`{"type":"object", "properties":{"foo":{"type":"integer"}}}`,
		},
		{
			false, "type object with non-trivial members",
			`{"type":"object", "properties":{"foo":{"type":"integer","minimum":3}}}`,
		},
		{false, "type fixed array", `{"type":"array", "items":[{"type":"string"}]}`},
		{true, "type array", `{"type":"array", "items":{"type":"string"}}`},
		{
			false, "type array with non-trivial members",
			`{"type":"array", "items":{"type":"string", "format":"email"}}`,
		},
		{true, "type array additionalItems", `{"type":"array", "additionalItems":{"type":"string"}}`},
		{
			false, "type array additionalItems with non-trivial members",
			`{"type":"array", "additionalItems":{"type":"string", "format":"email"}}`,
		},
		{true, "scalar type", `{"type":"integer"}`},
		{true, "scalar nullable type", `{"type":["integer", "null"]}`},
		{false, "scalar types", `{"type":["integer", "string"]}`},
		{false, "with format", `{"format":"email"}`},
		{false, "with not", `{"not":true}`},
		{false, "with allOf", `{"allOf":[true]}`},
		{false, "with enum", `{"enum":[1,2,3]}`},
		{false, "with minItems", `{"minItems":5}`},
		{false, "with minProperties", `{"minProperties":5}`},
		{false, "with $ref", `{"$ref":"#/definitions/foo","definitions":{"foo":true}}`},
	} {
		s := s

		t.Run(s.name, func(t *testing.T) {
			var schema jsonschema.SchemaOrBool

			assert.NoError(t, json.Unmarshal([]byte(s.schema), &schema))
			assert.Equal(t, s.isTrivial, schema.IsTrivial())
		})
	}
}
