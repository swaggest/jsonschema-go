package jsonschema

import "github.com/swaggest/jsonschema-go/refl"

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

func InlineRefs(pc *ParseContext) {
	pc.InlineRefs = true
}

type ParseContext struct {
	DefinitionsPrefix string
	PropertyNameTag   string
	InlineRefs        bool
	InlineRoot        bool

	Path            []string
	definitions     map[refl.TypeString]CoreSchemaMetaSchema // list of all definition objects
	definitionRefs  map[refl.TypeString]Ref
	definitionAlloc map[string]refl.TypeString // index of allocated TypeNames
	typeCycles      map[refl.TypeString]bool
}
