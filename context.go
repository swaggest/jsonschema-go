package jsonschema

import (
	"reflect"

	"github.com/swaggest/refl"
)

// DefinitionsPrefix sets up location for newly created references, default "#/definitions/".
func DefinitionsPrefix(prefix string) func(*ReflectContext) {
	return func(pc *ReflectContext) {
		pc.DefinitionsPrefix = prefix
	}
}

// PropertyNameTag sets up which field tag to use for property name, default "json".
func PropertyNameTag(tag string) func(*ReflectContext) {
	return func(pc *ReflectContext) {
		pc.PropertyNameTag = tag
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
	return func(pc *ReflectContext) {
		if pc.InterceptType != nil {
			prev := pc.InterceptType
			pc.InterceptType = func(v reflect.Value, s *Schema) (b bool, err error) {
				ret, err := prev(v, s)
				if err != nil || ret {
					return ret, err
				}

				return f(v, s)
			}
		} else {
			pc.InterceptType = f
		}
	}
}

// InterceptProperty adds hook to customize property schema.
func InterceptProperty(f InterceptPropertyFunc) func(*ReflectContext) {
	return func(pc *ReflectContext) {
		if pc.InterceptProperty != nil {
			prev := pc.InterceptProperty
			pc.InterceptProperty = func(name string, field reflect.StructField, propertySchema *Schema) error {
				err := prev(name, field, propertySchema)
				if err != nil {
					return err
				}

				return f(name, field, propertySchema)
			}
		} else {
			pc.InterceptProperty = f
		}
	}
}

// InlineRefs prevents references.
func InlineRefs(pc *ReflectContext) {
	pc.InlineRefs = true
}

// InlineRoot prevents referencing root schema.
func InlineRoot(pc *ReflectContext) {
	pc.InlineRoot = true
}

// ReflectContext accompanies single reflect operation.
type ReflectContext struct {
	DefinitionsPrefix string
	PropertyNameTag   string
	InlineRefs        bool
	InlineRoot        bool
	InterceptType     InterceptTypeFunc
	InterceptProperty InterceptPropertyFunc

	Path           []string
	definitions    map[refl.TypeString]Schema // list of all definition objects
	definitionRefs map[refl.TypeString]Ref
	typeCycles     map[refl.TypeString]bool
}
