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

// HijackFunc can intercept reflection to control or modify schema.
//
// True bool result demands no further processing for the Schema.
type HijackFunc func(reflect.Value, *Schema) (bool, error)

// HijackType adds hook to customize schema.
func HijackType(f HijackFunc) func(*ReflectContext) {
	return func(pc *ReflectContext) {
		if pc.HijackType != nil {
			prev := pc.HijackType
			pc.HijackType = func(v reflect.Value, s *Schema) (b bool, err error) {
				ret, err := prev(v, s)
				if err != nil || ret {
					return ret, err
				}

				return f(v, s)
			}
		} else {
			pc.HijackType = f
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
	HijackType        HijackFunc

	Path             []string
	WalkedProperties []string
	definitions      map[refl.TypeString]Schema // list of all definition objects
	definitionRefs   map[refl.TypeString]Ref
	typeCycles       map[refl.TypeString]bool
}
