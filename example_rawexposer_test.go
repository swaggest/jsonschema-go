package jsonschema_test

import (
	"fmt"

	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
)

// StructureWithRawExposer is an example structure.
type StructureWithRawExposer struct {
	Foo string `json:"foo"`
}

var _ jsonschema.RawExposer = StructureWithRawExposer{}

func (s StructureWithRawExposer) JSONSchemaBytes() ([]byte, error) {
	return []byte(``)
}

func ExampleRawExposer() {
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
