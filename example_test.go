package jsonschema_test

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/swaggest/jsonschema-go"
)

// WeirdResp hides sample structure.
type WeirdResp interface {
	Boo()
}

// UUID represents type owned by 3rd party library.
type UUID [16]byte

// Resp is a sample structure.
type Resp struct {
	Field1 int    `json:"field1"`
	Field2 string `json:"field2"`
	Info   struct {
		Foo string  `json:"foo" default:"baz" required:"true" pattern:"\\d+"`
		Bar float64 `json:"bar" description:"This is Bar."`
	} `json:"info"`
	Parent               *Resp                  `json:"parent"`
	Map                  map[string]int64       `json:"map"`
	MapOfAnything        map[string]interface{} `json:"mapOfAnything"`
	ArrayOfAnything      []interface{}          `json:"arrayOfAnything"`
	Whatever             interface{}            `json:"whatever"`
	NullableWhatever     *interface{}           `json:"nullableWhatever,omitempty"`
	RecursiveArray       []WeirdResp            `json:"recursiveArray"`
	RecursiveStructArray []Resp                 `json:"recursiveStructArray"`
	UUID                 UUID                   `json:"uuid"`
}

// Description implements jsonschema.Described.
func (r *Resp) Description() string {
	return "This is a sample response."
}

// Title implements jsonschema.Titled.
func (r *Resp) Title() string {
	return "Sample Response"
}

var (
	_ jsonschema.Preparer = &Resp{}
)

func (r *Resp) PrepareJSONSchema(s *jsonschema.Schema) error {
	s.WithExtraPropertiesItem("x-foo", "bar")
	return nil
}

func ExampleReflector_Reflect() {
	reflector := jsonschema.Reflector{}

	// Create custom schema mapping for 3rd party type.
	uuidDef := jsonschema.Schema{}
	uuidDef.AddType(jsonschema.String)
	uuidDef.WithFormat("uuid")
	uuidDef.WithExamples("248df4b7-aa70-47b8-a036-33ac447e668d")

	// Map 3rd party type with your own schema.
	reflector.AddTypeMapping(UUID{}, uuidDef)

	// Map the type that does not expose schema information to a type with schema information.
	reflector.AddTypeMapping(new(WeirdResp), new(Resp))

	// Create schema from Go value.
	schema, err := reflector.Reflect(new(Resp))
	if err != nil {
		log.Fatal(err)
	}

	j, err := json.MarshalIndent(schema, "", " ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(j))

	// Output:
	// {
	//  "title": "Sample Response",
	//  "description": "This is a sample response.",
	//  "definitions": {
	//   "JsonschemaGoTestResp": {
	//    "type": "null",
	//    "x-foo": "bar"
	//   },
	//   "JsonschemaGoTestUUID": {
	//    "examples": [
	//     "248df4b7-aa70-47b8-a036-33ac447e668d"
	//    ],
	//    "type": "string",
	//    "format": "uuid"
	//   }
	//  },
	//  "properties": {
	//   "arrayOfAnything": {
	//    "items": {},
	//    "type": "array"
	//   },
	//   "field1": {
	//    "type": "integer"
	//   },
	//   "field2": {
	//    "type": "string"
	//   },
	//   "info": {
	//    "required": [
	//     "foo"
	//    ],
	//    "properties": {
	//     "bar": {
	//      "description": "This is Bar.",
	//      "type": "number"
	//     },
	//     "foo": {
	//      "pattern": "\\d+",
	//      "type": "string"
	//     }
	//    },
	//    "type": "object"
	//   },
	//   "map": {
	//    "additionalProperties": {
	//     "type": "integer"
	//    },
	//    "type": "object"
	//   },
	//   "mapOfAnything": {
	//    "additionalProperties": {},
	//    "type": "object"
	//   },
	//   "nullableWhatever": {
	//    "type": "null"
	//   },
	//   "parent": {
	//    "$ref": "#/definitions/JsonschemaGoTestResp"
	//   },
	//   "recursiveArray": {
	//    "items": {
	//     "$ref": "#/definitions/JsonschemaGoTestResp"
	//    },
	//    "type": "array"
	//   },
	//   "recursiveStructArray": {
	//    "items": {
	//     "$ref": "#/definitions/JsonschemaGoTestResp"
	//    },
	//    "type": "array"
	//   },
	//   "uuid": {
	//    "$ref": "#/definitions/JsonschemaGoTestUUID"
	//   },
	//   "whatever": {}
	//  },
	//  "type": "object",
	//  "x-foo": "bar"
	// }
}

func ExampleReflector_Reflect_simple() {
	type MyStruct struct {
		Amount float64 `json:"amount" minimum:"10.5" example:"20.6" required:"true"`
		Abc    string  `json:"abc" pattern:"[abc]"`
	}

	reflector := jsonschema.Reflector{}

	schema, err := reflector.Reflect(MyStruct{})
	if err != nil {
		log.Fatal(err)
	}

	j, err := json.MarshalIndent(schema, "", " ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(j))

	// Output:
	// {
	//  "required": [
	//   "amount"
	//  ],
	//  "properties": {
	//   "abc": {
	//    "pattern": "[abc]",
	//    "type": "string"
	//   },
	//   "amount": {
	//    "examples": [
	//     20.6
	//    ],
	//    "minimum": 10.5,
	//    "type": "number"
	//   }
	//  },
	//  "type": "object"
	// }
}
