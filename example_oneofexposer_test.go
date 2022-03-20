package jsonschema_test

import (
	"fmt"

	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
)

// ParentOfOneOfExposer is an example structure.
type ParentOfOneOfExposer struct {
	Bar OneOfExposer `json:"bar"`
}

// OneOfExposer is an example structure.
type OneOfExposer struct{}

type OneOf1 struct {
	Foo string `json:"foo" required:"true"`
}

type OneOf2 struct {
	Baz string `json:"baz" required:"true"`
}

var _ jsonschema.OneOfExposer = OneOfExposer{}

func (OneOfExposer) JSONSchemaOneOf() []interface{} {
	return []interface{}{
		OneOf1{}, OneOf2{},
	}
}

func ExampleOneOfExposer() {
	reflector := jsonschema.Reflector{}

	s, err := reflector.Reflect(ParentOfOneOfExposer{}, jsonschema.StripDefinitionNamePrefix("JsonschemaGoTest"))
	if err != nil {
		panic(err)
	}

	j, err := assertjson.MarshalIndentCompact(s, "", "  ", 80)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(j))
	// Output:
	// {
	//   "definitions":{
	//     "OneOf1":{
	//       "required":["foo"],"properties":{"foo":{"type":"string"}},"type":"object"
	//     },
	//     "OneOf2":{
	//       "required":["baz"],"properties":{"baz":{"type":"string"}},"type":"object"
	//     },
	//     "OneOfExposer":{
	//       "type":"object",
	//       "oneOf":[{"$ref":"#/definitions/OneOf1"},{"$ref":"#/definitions/OneOf2"}]
	//     }
	//   },
	//   "properties":{"bar":{"$ref":"#/definitions/OneOfExposer"}},"type":"object"
	// }
}
