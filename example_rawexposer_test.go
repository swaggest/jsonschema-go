package jsonschema_test

import (
	"fmt"

	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
)

// ParentOfRawExposer is an example structure.
type ParentOfRawExposer struct {
	Bar RawExposer `json:"bar"`
}

// RawExposer is an example structure.
type RawExposer struct {
	Foo string `json:"foo"`
}

var _ jsonschema.RawExposer = RawExposer{}

// JSONSchemaBytes returns raw JSON Schema bytes.
// Fields and tags of structure are ignored.
func (s RawExposer) JSONSchemaBytes() ([]byte, error) {
	return []byte(`{"description":"Custom description.","type":"object","properties":{"foo":{"type":"string"}}}`), nil
}

func ExampleRawExposer() {
	reflector := jsonschema.Reflector{}

	s, err := reflector.Reflect(ParentOfRawExposer{}, jsonschema.StripDefinitionNamePrefix("JsonschemaGoTest"))
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
	//     "RawExposer":{
	//       "description":"Custom description.",
	//       "properties":{"foo":{"type":"string"}},"type":"object"
	//     }
	//   },
	//   "properties":{"bar":{"$ref":"#/definitions/RawExposer"}},"type":"object"
	// }
}
