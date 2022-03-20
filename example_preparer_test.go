package jsonschema_test

import (
	"fmt"

	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
)

// ParentOfPreparer is an example structure.
type ParentOfPreparer struct {
	Bar Preparer `json:"bar"`
}

// Preparer is an example structure.
type Preparer struct {
	Foo string `json:"foo"`
}

var _ jsonschema.Preparer = Preparer{}

func (s Preparer) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.WithDescription("Custom description.")
	schema.Properties["foo"].TypeObject.WithEnum("one", "two", "three")

	return nil
}

func ExamplePreparer() {
	reflector := jsonschema.Reflector{}

	s, err := reflector.Reflect(ParentOfPreparer{}, jsonschema.StripDefinitionNamePrefix("JsonschemaGoTest"))
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
	//     "Preparer":{
	//       "description":"Custom description.",
	//       "properties":{"foo":{"enum":["one","two","three"],"type":"string"}},
	//       "type":"object"
	//     }
	//   },
	//   "properties":{"bar":{"$ref":"#/definitions/Preparer"}},"type":"object"
	// }
}
