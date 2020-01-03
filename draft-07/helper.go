package jsonschema

import (
	"encoding/json"
	"errors"
)

const (
	// XEnumNames is the name of JSON property to store names of enumerated values.
	XEnumNames = "x-enum-names"
)

// NamedEnum returns the enumerated acceptable values with according string names.
type NamedEnum interface {
	NamedEnum() ([]interface{}, []string)
}

// Enum returns the enumerated acceptable values.
type Enum interface {
	Enum() []interface{}
}

// Exporter returns JSON Schema in library agnostic way.
type Exporter interface {
	JSONSchema() (map[string]interface{}, error)
}

// Customizer alters reflected JSON Schema.
type Customizer interface {
	CustomizeJSONSchema(schema *CoreSchemaMetaSchema) error
}

// JSONSchema exports JSON Schema as a map.
func (i CoreSchemaMetaSchema) JSONSchema() (map[string]interface{}, error) {
	jsonBytes, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	var decoded interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	if err != nil {
		return nil, err
	}

	if m, ok := decoded.(map[string]interface{}); ok {
		return m, nil
	}

	return nil, errors.New("invalid json, map expected")
}

// Type references simple type.
func (i SimpleTypes) Type() Type {
	return Type{SimpleTypes: &i}
}

// Ptr references simple type.
func (i SimpleTypes) Ptr() *SimpleTypes {
	return &i
}
