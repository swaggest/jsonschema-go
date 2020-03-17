package jsonschema

import (
	"reflect"

	"github.com/swaggest/jsonschema-go/refl"
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

func HijackType(f func(v reflect.Value, s *Schema) (bool, error)) func(*ParseContext) {
	return func(pc *ParseContext) {
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

func InlineRefs(pc *ParseContext) {
	pc.InlineRefs = true
}

func InlineRoot(pc *ParseContext) {
	pc.InlineRoot = true
}

type ParseContext struct {
	DefinitionsPrefix string
	PropertyNameTag   string
	InlineRefs        bool
	InlineRoot        bool
	HijackType        func(v reflect.Value, s *Schema) (bool, error)

	Path             []string
	WalkedProperties []string
	definitions      map[refl.TypeString]Schema // list of all definition objects
	definitionRefs   map[refl.TypeString]Ref
	typeCycles       map[refl.TypeString]bool
}
