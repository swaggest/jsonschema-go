package jsonschema_test

import (
	"fmt"

	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
)

// StructureWithPreparer is an example structure.
type StructureWithPreparer struct {
	Foo string `json:"foo"`
}

var _ jsonschema.Preparer = StructureWithPreparer{}

func (s StructureWithPreparer) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.WithDescription("Custom description.")
	schema.WithEnum("one", "two", "three")

	return nil
}

func ExamplePreparer() {
	reflector := jsonschema.Reflector{}

	s, err := reflector.Reflect(StructureWithPreparer{})
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
	//   "description":"Custom description.","properties":{"foo":{"type":"string"}},
	//   "enum":["one","two","three"],"type":"object"
	// }
}
