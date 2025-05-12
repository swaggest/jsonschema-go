package jsonschema_test

import (
	"fmt"

	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
)

// ParentOfExposer is an example structure.
type ParentOfExposer struct {
	Bar Exposer `json:"bar"`
}

// Exposer is an example structure.
type Exposer struct {
	Foo string `json:"foo"`
}

var _ jsonschema.Exposer = Exposer{}

// JSONSchema returns raw JSON Schema bytes.
// Fields and tags of structure are ignored.
func (Exposer) JSONSchema() (jsonschema.Schema, error) {
	var schema jsonschema.Schema

	schema.AddType(jsonschema.Object)
	schema.WithDescription("Custom description.")
	schema.WithPropertiesItem("foo", jsonschema.String.ToSchemaOrBool())

	return schema, nil
}

func ExampleExposer() {
	reflector := jsonschema.Reflector{}

	s, err := reflector.Reflect(ParentOfExposer{}, jsonschema.StripDefinitionNamePrefix("JsonschemaGoTest"))
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
	//     "Exposer":{
	//       "description":"Custom description.",
	//       "properties":{"foo":{"type":"string"}},"type":"object"
	//     }
	//   },
	//   "properties":{"bar":{"$ref":"#/definitions/Exposer"}},"type":"object"
	// }
}
