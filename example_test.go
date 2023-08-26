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

func ExampleReflector_InlineDefinition() {
	reflector := jsonschema.Reflector{}

	// Create custom schema mapping for 3rd party type.
	uuidDef := jsonschema.Schema{}
	uuidDef.AddType(jsonschema.String)
	uuidDef.WithFormat("uuid")
	uuidDef.WithExamples("248df4b7-aa70-47b8-a036-33ac447e668d")

	// Map 3rd party type with your own schema.
	reflector.AddTypeMapping(UUID{}, uuidDef)
	reflector.InlineDefinition(UUID{})

	type MyStruct struct {
		ID UUID `json:"id"`
	}

	schema, _ := reflector.Reflect(MyStruct{})

	schemaJSON, _ := json.MarshalIndent(schema, "", " ")

	fmt.Println(string(schemaJSON))
	// Output:
	// {
	//  "properties": {
	//   "id": {
	//    "examples": [
	//     "248df4b7-aa70-47b8-a036-33ac447e668d"
	//    ],
	//    "type": "string",
	//    "format": "uuid"
	//   }
	//  },
	//  "type": "object"
	// }
}

func ExampleReflector_AddTypeMapping_schema() {
	reflector := jsonschema.Reflector{}

	// Create custom schema mapping for 3rd party type.
	uuidDef := jsonschema.Schema{}
	uuidDef.AddType(jsonschema.String)
	uuidDef.WithFormat("uuid")
	uuidDef.WithExamples("248df4b7-aa70-47b8-a036-33ac447e668d")

	// Map 3rd party type with your own schema.
	reflector.AddTypeMapping(UUID{}, uuidDef)

	type MyStruct struct {
		ID UUID `json:"id"`
	}

	schema, _ := reflector.Reflect(MyStruct{})

	schemaJSON, _ := json.MarshalIndent(schema, "", " ")

	fmt.Println(string(schemaJSON))
	// Output:
	// {
	//  "definitions": {
	//   "JsonschemaGoTestUUID": {
	//    "examples": [
	//     "248df4b7-aa70-47b8-a036-33ac447e668d"
	//    ],
	//    "type": "string",
	//    "format": "uuid"
	//   }
	//  },
	//  "properties": {
	//   "id": {
	//    "$ref": "#/definitions/JsonschemaGoTestUUID"
	//   }
	//  },
	//  "type": "object"
	// }
}

func ExampleReflector_AddTypeMapping_type() {
	reflector := jsonschema.Reflector{}

	// Map 3rd party type with a different type.
	// Reflector will perceive all UUIDs as plain strings.
	reflector.AddTypeMapping(UUID{}, "")

	type MyStruct struct {
		ID UUID `json:"id"`
	}

	schema, _ := reflector.Reflect(MyStruct{})

	schemaJSON, _ := json.MarshalIndent(schema, "", " ")

	fmt.Println(string(schemaJSON))
	// Output:
	// {
	//  "properties": {
	//   "id": {
	//    "type": "string"
	//   }
	//  },
	//  "type": "object"
	// }
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
	reflector.DefaultOptions = append(reflector.DefaultOptions, jsonschema.InterceptDefName(
		func(t reflect.Type, defaultDefName string) string {
			return strings.TrimPrefix(defaultDefName, "JsonschemaGoTest")
		},
	))

	// Create schema from Go value.
	schema, err := reflector.Reflect(new(Resp))
	if err != nil {
		log.Fatal(err)
	}

	j, err := assertjson.MarshalIndentCompact(schema, "", "  ", 80)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(j))

	// Output:
	// {
	//   "title":"Sample Response","description":"This is a sample response.",
	//   "definitions":{
	//     "NamedAnything":{},
	//     "UUID":{
	//       "examples":["248df4b7-aa70-47b8-a036-33ac447e668d"],"type":"string",
	//       "format":"uuid"
	//     }
	//   },
	//   "properties":{
	//     "arrayOfAnything":{"items":{},"type":"array"},
	//     "arrayOfNamedAnything":{"items":{"$ref":"#/definitions/NamedAnything"},"type":"array"},
	//     "field1":{"type":"integer"},"field2":{"type":"string"},
	//     "info":{
	//       "required":["foo"],
	//       "properties":{
	//         "bar":{"description":"This is Bar.","type":"number"},
	//         "foo":{"default":"baz","pattern":"\\d+","type":"string"}
	//       },
	//       "type":"object"
	//     },
	//     "map":{"additionalProperties":{"type":"integer"},"type":"object"},
	//     "mapOfAnything":{"additionalProperties":{},"type":"object"},
	//     "nullableWhatever":{},"parent":{"$ref":"#"},
	//     "recursiveArray":{"items":{"$ref":"#"},"type":"array"},
	//     "recursiveStructArray":{"items":{"$ref":"#"},"type":"array"},
	//     "uuid":{"$ref":"#/definitions/UUID"},"whatever":{}
	//   },
	//   "type":"object","x-foo":"bar"
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

func ExampleInterceptProp() {
	reflector := jsonschema.Reflector{}

	type Test struct {
		ID      int     `json:"id" minimum:"123" default:"200"`
		Name    string  `json:"name" minLength:"10"`
		Skipped float64 `json:"skipped"`
	}

	s, err := reflector.Reflect(new(Test),
		// PropertyNameMapping allows configuring property names without field tag.
		jsonschema.InterceptProp(
			func(params jsonschema.InterceptPropParams) error {
				switch params.Name {
				// You can alter reflected schema by updating propertySchema.
				case "id":
					if params.Processed {
						params.PropertySchema.WithDescription("This is ID.")
						// You can access schema that holds the property.
						params.PropertySchema.Parent.WithDescription("Schema with ID.")
					}

				// Or you can entirely remove property from parent schema with a sentinel error.
				case "skipped":
					return jsonschema.ErrSkipProperty
				}

				return nil
			},
		),
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

func ExampleReflector_Reflect_default() {
	type MyStruct struct {
		A []string       `json:"a" default:"[A,B,C]"` // For an array of strings, comma-separated values in square brackets can be used.
		B []int          `json:"b" default:"[1,2,3]"` // Other non-scalar values are parsed as JSON without type checking.
		C []string       `json:"c" default:"[\"C\",\"B\",\"A\"]"`
		D int            `json:"d" default:"123"` // Scalar values are parsed according to their type.
		E string         `json:"e" default:"abc"`
		F map[string]int `json:"f" default:"{\"foo\":1,\"bar\":2}"`
	}

	type Invalid struct {
		I []int `json:"i" default:"[C,B,A]"` // Value with invalid JSON is ignored for types other than []string (and equivalent).
	}

	r := jsonschema.Reflector{}
	s, _ := r.Reflect(MyStruct{})
	_, err := r.Reflect(Invalid{})

	j, _ := assertjson.MarshalIndentCompact(s, "", " ", 80)

	fmt.Println("MyStruct:", string(j))
	fmt.Println("Invalid error:", err.Error())
	// Output:
	// MyStruct: {
	//  "properties":{
	//   "a":{"default":["A","B","C"],"items":{"type":"string"},"type":["array","null"]},
	//   "b":{"default":[1,2,3],"items":{"type":"integer"},"type":["array","null"]},
	//   "c":{"default":["C","B","A"],"items":{"type":"string"},"type":["array","null"]},
	//   "d":{"default":123,"type":"integer"},"e":{"default":"abc","type":"string"},
	//   "f":{
	//    "default":{"bar":2,"foo":1},"additionalProperties":{"type":"integer"},
	//    "type":["object","null"]
	//   }
	//  },
	//  "type":"object"
	// }
	// Invalid error: I: parsing default as JSON: invalid character 'C' looking for beginning of value
}

func ExampleReflector_Reflect_virtualStruct() {
	s := jsonschema.Struct{}
	s.SetTitle("Test title")
	s.SetDescription("Test description")
	s.DefName = "TestStruct"
	s.Nullable = true

	s.Fields = append(s.Fields, jsonschema.Field{
		Name:  "Foo",
		Value: "abc",
		Tag:   `json:"foo" minLength:"3"`,
	})

	r := jsonschema.Reflector{}
	schema, _ := r.Reflect(s)
	j, _ := assertjson.MarshalIndentCompact(schema, "", " ", 80)

	fmt.Println("Standalone:", string(j))

	type MyStruct struct {
		jsonschema.Struct // Can be embedded.

		Bar int `json:"bar"`

		Nested jsonschema.Struct `json:"nested"` // Can be nested.
	}

	ms := MyStruct{}
	ms.Nested = s
	ms.Struct = s

	schema, _ = r.Reflect(ms)
	j, _ = assertjson.MarshalIndentCompact(schema, "", " ", 80)

	fmt.Println("Nested:", string(j))

	// Output:
	// Standalone: {
	//  "title":"Test title","description":"Test description",
	//  "properties":{"foo":{"minLength":3,"type":"string"}},"type":"object"
	// }
	// Nested: {
	//  "title":"Test title","description":"Test description",
	//  "properties":{
	//   "bar":{"type":"integer"},"foo":{"minLength":3,"type":"string"},
	//   "nested":{"$ref":"#"}
	//  },
	//  "type":"object"
	// }
}
