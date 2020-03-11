package jsonschema

import (
	"github.com/swaggest/jsonschema-go/refl"
	"reflect"
)

func DefinitionsPrefix(prefix string) func(*ParseContext) {
	return func(pc *ParseContext) {
		pc.DefinitionsPrefix = prefix
	}
}

func PropertyNameTag(tag string) func(*ParseContext) {
	return func(pc *ParseContext) {
		pc.PropertyNameTag = tag
	}
}

func HijackType(f func(t reflect.Type, s *CoreSchemaMetaSchema) (bool, error)) func(*ParseContext) {
	return func(pc *ParseContext) {
		if pc.HijackType != nil {
			prev := pc.HijackType
			pc.HijackType = func(t reflect.Type, s *CoreSchemaMetaSchema) (b bool, err error) {
				ret, err := prev(t, s)
				if err != nil || ret {
					return ret, err
				}
				return f(t, s)
			}
		} else {
			pc.HijackType = f
		}
	}
}

func InlineRefs(pc *ParseContext) {
	pc.InlineRefs = true
}

type ParseContext struct {
	DefinitionsPrefix string
	PropertyNameTag   string
	InlineRefs        bool
	InlineRoot        bool
	HijackType        func(t reflect.Type, s *CoreSchemaMetaSchema) (bool, error)

	Path             []string
	WalkedProperties []string
	definitions      map[refl.TypeString]CoreSchemaMetaSchema // list of all definition objects
	definitionRefs   map[refl.TypeString]Ref
	definitionAlloc  map[string]refl.TypeString // index of allocated TypeNames
	typeCycles       map[refl.TypeString]bool
}
