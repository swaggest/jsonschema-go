package jsonschema

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
