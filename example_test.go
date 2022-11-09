package jsonschema_test

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
)

// WeirdResp hides sample structure.
type WeirdResp interface {
	Boo()
}

// NamedAnything is an empty interface.
type NamedAnything interface{}

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
	Parent               *Resp                  `json:"parent,omitempty"`
	Map                  map[string]int64       `json:"map,omitempty"`
	MapOfAnything        map[string]interface{} `json:"mapOfAnything,omitempty"`
	ArrayOfAnything      []interface{}          `json:"arrayOfAnything,omitempty"`
	ArrayOfNamedAnything []NamedAnything        `json:"arrayOfNamedAnything,omitempty"`
	Whatever             interface{}            `json:"whatever"`
	NullableWhatever     *interface{}           `json:"nullableWhatever,omitempty"`
	RecursiveArray       []WeirdResp            `json:"recursiveArray,omitempty"`
	RecursiveStructArray []Resp                 `json:"recursiveStructArray,omitempty"`
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

var _ jsonschema.Preparer = &Resp{}

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

	// Modify default definition names to better match your packages structure.
	reflector.InterceptDefName(func(t reflect.Type, defaultDefName string) string {
		return strings.TrimPrefix(defaultDefName, "JsonschemaGoTest")
	})

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
	//   "NamedAnything": {},
	//   "UUID": {
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
	//   "arrayOfNamedAnything": {
	//    "items": {
	//     "$ref": "#/definitions/NamedAnything"
	//    },
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
	//      "default": "baz",
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
	//   "nullableWhatever": {},
	//   "parent": {
	//    "type": "null",
	//    "allOf": [
	//     {
	//      "$ref": "#"
	//     }
	//    ]
	//   },
	//   "recursiveArray": {
	//    "items": {
	//     "$ref": "#"
	//    },
	//    "type": "array"
	//   },
	//   "recursiveStructArray": {
	//    "items": {
	//     "$ref": "#"
	//    },
	//    "type": "array"
	//   },
	//   "uuid": {
	//    "$ref": "#/definitions/UUID"
	//   },
	//   "whatever": {}
	//  },
	//  "type": "object",
	//  "x-foo": "bar"
	// }
}

func ExampleReflector_Reflect_simple() {
	type MyStruct struct {
		Amount float64  `json:"amount" minimum:"10.5" example:"20.6" required:"true"`
		Abc    string   `json:"abc" pattern:"[abc]"`
		_      struct{} `additionalProperties:"false"`                   // Tags of unnamed field are applied to parent schema.
		_      struct{} `title:"My Struct" description:"Holds my data."` // Multiple unnamed fields can be used.
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
	//  "title": "My Struct",
	//  "description": "Holds my data.",
	//  "required": [
	//   "amount"
	//  ],
	//  "additionalProperties": false,
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

func ExamplePropertyNameMapping() {
	reflector := jsonschema.Reflector{}

	type Test struct {
		ID   int    `minimum:"123" default:"200"`
		Name string `minLength:"10"`
	}

	s, err := reflector.Reflect(new(Test),
		// PropertyNameMapping allows configuring property names without field tag.
		jsonschema.PropertyNameMapping(map[string]string{
			"ID":   "ident",
			"Name": "last_name",
		}))
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
	//   "properties":{
	//     "ident":{"default":200,"minimum":123,"type":"integer"},
	//     "last_name":{"minLength":10,"type":"string"}
	//   },
	//   "type":"object"
	// }
}

func ExampleInterceptProperty() {
	reflector := jsonschema.Reflector{}

	type Test struct {
		ID      int     `json:"id" minimum:"123" default:"200"`
		Name    string  `json:"name" minLength:"10"`
		Skipped float64 `json:"skipped"`
	}

	s, err := reflector.Reflect(new(Test),
		// PropertyNameMapping allows configuring property names without field tag.
		jsonschema.InterceptProperty(func(name string, field reflect.StructField, propertySchema *jsonschema.Schema) error {
			switch name {
			// You can alter reflected schema by updating propertySchema.
			case "id":
				propertySchema.WithDescription("This is ID.")
				// You can access schema that holds the property.
				propertySchema.Parent.WithDescription("Schema with ID.")

			// Or you can entirely remove property from parent schema with a sentinel error.
			case "skipped":
				return jsonschema.ErrSkipProperty
			}

			return nil
		}),
	)
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
	//   "description":"Schema with ID.",
	//   "properties":{
	//     "id":{"description":"This is ID.","default":200,"minimum":123,"type":"integer"},
	//     "name":{"minLength":10,"type":"string"}
	//   },
	//   "type":"object"
	// }
}

func ExampleOneOf() {
	r := jsonschema.Reflector{}

	type Test struct {
		Foo jsonschema.OneOfExposer `json:"foo"`
		Bar jsonschema.OneOfExposer `json:"bar"`
	}

	tt := Test{
		Foo: jsonschema.OneOf(1.23, "abc"),
		Bar: jsonschema.OneOf(123, true),
	}

	s, _ := r.Reflect(tt, jsonschema.RootRef)
	b, _ := assertjson.MarshalIndentCompact(s, "", " ", 100)

	fmt.Println("Complex schema:", string(b))

	s, _ = r.Reflect(jsonschema.OneOf(123, true), jsonschema.RootRef)
	b, _ = assertjson.MarshalIndentCompact(s, "", " ", 100)

	fmt.Println("Simple schema:", string(b))

	// Output:
	// Complex schema: {
	//  "$ref":"#/definitions/JsonschemaGoTestTest",
	//  "definitions":{
	//   "JsonschemaGoTestTest":{
	//    "properties":{
	//     "bar":{"oneOf":[{"type":"integer"},{"type":"boolean"}]},
	//     "foo":{"oneOf":[{"type":"number"},{"type":"string"}]}
	//    },
	//    "type":"object"
	//   }
	//  }
	// }
	// Simple schema: {"oneOf":[{"type":"integer"},{"type":"boolean"}]}
}

func ExampleAnyOf() {
	r := jsonschema.Reflector{}

	type Test struct {
		Foo jsonschema.AnyOfExposer `json:"foo"`
		Bar jsonschema.AnyOfExposer `json:"bar"`
	}

	tt := Test{
		Foo: jsonschema.AnyOf(1.23, "abc"),
		Bar: jsonschema.AnyOf(123, true),
	}

	s, _ := r.Reflect(tt, jsonschema.RootRef)
	b, _ := assertjson.MarshalIndentCompact(s, "", " ", 100)

	fmt.Println("Complex schema:", string(b))

	s, _ = r.Reflect(jsonschema.AnyOf(123, true), jsonschema.RootRef)
	b, _ = assertjson.MarshalIndentCompact(s, "", " ", 100)

	fmt.Println("Simple schema:", string(b))

	// Output:
	// Complex schema: {
	//  "$ref":"#/definitions/JsonschemaGoTestTest",
	//  "definitions":{
	//   "JsonschemaGoTestTest":{
	//    "properties":{
	//     "bar":{"anyOf":[{"type":"integer"},{"type":"boolean"}]},
	//     "foo":{"anyOf":[{"type":"number"},{"type":"string"}]}
	//    },
	//    "type":"object"
	//   }
	//  }
	// }
	// Simple schema: {"anyOf":[{"type":"integer"},{"type":"boolean"}]}
}

func ExampleAllOf() {
	r := jsonschema.Reflector{}

	type Test struct {
		Foo jsonschema.AllOfExposer `json:"foo"`
		Bar jsonschema.AllOfExposer `json:"bar"`
	}

	tt := Test{
		Foo: jsonschema.AllOf(1.23, "abc"),
		Bar: jsonschema.AllOf(123, true),
	}

	s, _ := r.Reflect(tt, jsonschema.RootRef)
	b, _ := assertjson.MarshalIndentCompact(s, "", " ", 100)

	fmt.Println("Complex schema:", string(b))

	s, _ = r.Reflect(jsonschema.AllOf(123, true), jsonschema.RootRef)
	b, _ = assertjson.MarshalIndentCompact(s, "", " ", 100)

	fmt.Println("Simple schema:", string(b))

	// Output:
	// Complex schema: {
	//  "$ref":"#/definitions/JsonschemaGoTestTest",
	//  "definitions":{
	//   "JsonschemaGoTestTest":{
	//    "properties":{
	//     "bar":{"allOf":[{"type":"integer"},{"type":"boolean"}]},
	//     "foo":{"allOf":[{"type":"number"},{"type":"string"}]}
	//    },
	//    "type":"object"
	//   }
	//  }
	// }
	// Simple schema: {"allOf":[{"type":"integer"},{"type":"boolean"}]}
}
