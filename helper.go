package jsonschema

import "encoding/json"

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

// Preparer alters reflected JSON Schema.
type Preparer interface {
	PrepareJSONSchema(schema *Schema) error
}

// Exposer exposes JSON Schema.
type Exposer interface {
	JSONSchema() (Schema, error)
}

// RawExposer exposes JSON Schema as JSON bytes.
type RawExposer interface {
	JSONSchemaBytes() ([]byte, error)
}

// JSONSchema implements Exposer.
func (s Schema) JSONSchema() (Schema, error) {
	return s, nil
}

// ToSchemaOrBool creates SchemaOrBool instance from Schema.
func (s *Schema) ToSchemaOrBool() SchemaOrBool {
	return SchemaOrBool{
		TypeObject: s,
	}
}

// Type references simple type.
func (i SimpleType) Type() Type {
	return Type{SimpleTypes: &i}
}

// ToSchemaOrBool creates SchemaOrBool instance from SimpleType.
func (i SimpleType) ToSchemaOrBool() SchemaOrBool {
	return SchemaOrBool{
		TypeObject: (&Schema{}).WithType(i.Type()),
	}
}

// AddType adds simple type to Schema.
//
// If type is already there it is ignored.
func (s *Schema) AddType(t SimpleType) {
	if s.Type == nil {
		s.WithType(t.Type())
		return
	}

	if s.Type.SimpleTypes != nil {
		if *s.Type.SimpleTypes == t {
			return
		}

		s.Type.SliceOfSimpleTypeValues = []SimpleType{*s.Type.SimpleTypes, t}
		s.Type.SimpleTypes = nil

		return
	}

	if len(s.Type.SliceOfSimpleTypeValues) > 0 {
		for _, st := range s.Type.SliceOfSimpleTypeValues {
			if st == t {
				return
			}
		}

		s.Type.SliceOfSimpleTypeValues = append(s.Type.SliceOfSimpleTypeValues, t)
	}
}

// HasType checks if Schema has a simple type.
func (s *Schema) HasType(t SimpleType) bool {
	if s.Type == nil {
		return false
	}

	if s.Type.SimpleTypes != nil {
		return *s.Type.SimpleTypes == t
	}

	if len(s.Type.SliceOfSimpleTypeValues) > 0 {
		for _, st := range s.Type.SliceOfSimpleTypeValues {
			if st == t {
				return true
			}
		}
	}

	return false
}

// JSONSchemaBytes exposes JSON Schema as raw JSON bytes.
func (s SchemaOrBool) JSONSchemaBytes() ([]byte, error) {
	return json.Marshal(s)
}

// JSONSchemaBytes exposes JSON Schema as raw JSON bytes.
func (s Schema) JSONSchemaBytes() ([]byte, error) {
	return json.Marshal(s)
}

// ToSimpleMap encodes JSON Schema as generic map.
func (s SchemaOrBool) ToSimpleMap() (map[string]interface{}, error) {
	var m map[string]interface{}

	if s.TypeBoolean != nil {
		if *s.TypeBoolean {
			return map[string]interface{}{}, nil
		}

		return map[string]interface{}{
			"not": map[string]interface{}{},
		}, nil
	}

	b, err := json.Marshal(s.TypeObject)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}
