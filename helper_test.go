package jsonschema_test

import (
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
