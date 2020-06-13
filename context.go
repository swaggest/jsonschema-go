package jsonschema

import (
	"reflect"

	"github.com/swaggest/refl"
)

// CollectDefinitions enables collecting definitions with provided func instead of result schema.
func CollectDefinitions(f func(name string, schema Schema)) func(*ReflectContext) {
	return func(rc *ReflectContext) {
		rc.CollectDefinitions = f
	}
}

// DefinitionsPrefix sets up location for newly created references, default "#/definitions/".
func DefinitionsPrefix(prefix string) func(*ReflectContext) {
	return func(rc *ReflectContext) {
		rc.DefinitionsPrefix = prefix
	}
}

// PropertyNameTag sets up which field tag to use for property name, default "json".
func PropertyNameTag(tag string) func(*ReflectContext) {
	return func(rc *ReflectContext) {
		rc.PropertyNameTag = tag
	}
}

// InterceptTypeFunc can intercept type reflection to control or modify schema.
//
// True bool result demands no further processing for the Schema.
type InterceptTypeFunc func(reflect.Value, *Schema) (bool, error)

// InterceptPropertyFunc can intercept field reflection to control or modify schema.
type InterceptPropertyFunc func(name string, field reflect.StructField, propertySchema *Schema) error

// InterceptType adds hook to customize schema.
func InterceptType(f InterceptTypeFunc) func(*ReflectContext) {
	return func(rc *ReflectContext) {
		if rc.InterceptType != nil {
			prev := rc.InterceptType
			rc.InterceptType = func(v reflect.Value, s *Schema) (b bool, err error) {
				ret, err := prev(v, s)
				if err != nil || ret {
					return ret, err
				}

				return f(v, s)
			}
		} else {
			rc.InterceptType = f
		}
	}
}

// InterceptProperty adds hook to customize property schema.
func InterceptProperty(f InterceptPropertyFunc) func(*ReflectContext) {
	return func(rc *ReflectContext) {
		if rc.InterceptProperty != nil {
			prev := rc.InterceptProperty
			rc.InterceptProperty = func(name string, field reflect.StructField, propertySchema *Schema) error {
				err := prev(name, field, propertySchema)
				if err != nil {
					return err
				}

				return f(name, field, propertySchema)
			}
		} else {
			rc.InterceptProperty = f
		}
	}
}

// InlineRefs prevents references.
func InlineRefs(rc *ReflectContext) {
	rc.InlineRefs = true
}

// RootNullable enables nullability (by pointer) for root schema, disabled by default.
func RootNullable(rc *ReflectContext) {
	rc.RootNullable = true
}

// RootRef enables referencing root schema.
func RootRef(rc *ReflectContext) {
	rc.RootRef = true
}

// ReflectContext accompanies single reflect operation.
type ReflectContext struct {
	CollectDefinitions func(name string, schema Schema)
	DefinitionsPrefix  string
	PropertyNameTag    string
	InlineRefs         bool
	RootRef            bool
	RootNullable       bool
	InterceptType      InterceptTypeFunc
	InterceptProperty  InterceptPropertyFunc

	Path           []string
	definitions    map[refl.TypeString]Schema // list of all definition objects
	definitionRefs map[refl.TypeString]Ref
	typeCycles     map[refl.TypeString]bool
	rootDefName    string
}

func (rc *ReflectContext) getDefinition(ref string) Schema {
	for ts, r := range rc.definitionRefs {
		if r.Path+r.Name == ref {
			return rc.definitions[ts]
		}
	}

	return Schema{}
}
