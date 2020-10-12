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

// IsTrivial is true if schema does not contain validation constraints other than type.
func (s SchemaOrBool) IsTrivial() bool {
	if s.TypeBoolean != nil && !*s.TypeBoolean {
		return false
	}

	if s.TypeObject != nil {
		return s.TypeObject.IsTrivial()
	}

	return true
}

// IsTrivial is true if schema does not contain validation constraints other than type.
//
// Trivial schema can define trivial items or properties.
// This flag can be used to skip validation of structures that check types during decoding.
//   nolint:gocyclo
func (s Schema) IsTrivial() bool {
	if len(s.AllOf) > 0 || len(s.AnyOf) > 0 || len(s.OneOf) > 0 || s.Not != nil ||
		s.If != nil || s.Then != nil || s.Else != nil {
		return false
	}

	if s.MultipleOf != nil || s.Minimum != nil || s.Maximum != nil ||
		s.ExclusiveMinimum != nil || s.ExclusiveMaximum != nil {
		return false
	}

	if s.MinLength != 0 || s.MaxLength != nil || s.Pattern != nil || s.Format != nil {
		return false
	}

	if s.MinItems != 0 || s.MaxItems != nil || s.UniqueItems != nil || s.Contains != nil {
		return false
	}

	if s.MinProperties != 0 || s.MaxProperties != nil || len(s.Required) > 0 || len(s.PatternProperties) > 0 {
		return false
	}

	if len(s.Dependencies) > 0 || s.PropertyNames != nil || s.Const != nil || len(s.Enum) > 0 {
		return false
	}

	if s.Type != nil && len(s.Type.SliceOfSimpleTypeValues) > 1 && !s.HasType(Null) {
		return false
	}

	if s.Ref != nil {
		return false
	}

	if s.Items != nil && (len(s.Items.SchemaArray) > 0 || !s.Items.SchemaOrBool.IsTrivial()) {
		return false
	}

	if s.AdditionalItems != nil && !s.AdditionalItems.IsTrivial() {
		return false
	}

	if s.AdditionalProperties != nil && !s.AdditionalProperties.IsTrivial() {
		return false
	}

	if len(s.Properties) > 0 {
		for _, ps := range s.Properties {
			if !ps.IsTrivial() {
				return false
			}
		}
	}

	return true
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
