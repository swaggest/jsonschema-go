package jsonschema_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
)

func TestReflector_Reflect_Struct(t *testing.T) {
	r := jsonschema.Reflector{}

	s := jsonschema.Struct{}
	s.SetTitle("Test title")
	s.SetDescription("Test description")
	s.DefName = "TestStruct"
	s.Nullable = true

	s.Fields = append(s.Fields, jsonschema.Field{
		Name:  "Foo",
		Value: "abc",
		Tag:   `json:"fo0" minLength:"3"`,
	})

	s.Fields = append(s.Fields, jsonschema.Field{
		Name:  "Bar",
		Value: 123,
		Tag:   `json:"b4r" minimum:"3"`,
	})

	s.Fields = append(s.Fields, jsonschema.Field{
		Name:  "Baz",
		Value: []int{},
		Tag:   `json:"b4z" minItems:"4"`,
	})

	s.Fields = append(s.Fields, jsonschema.Field{
		Name:  "Pers",
		Value: Person{},
		Tag:   `json:"pers"`,
	})

	s.Fields = append(s.Fields, jsonschema.Field{
		Name:  "Recursion",
		Value: s,
		Tag:   `json:"recursion"`,
	})

	s2 := jsonschema.Struct{}
	s2.SetTitle("T2")
	s2.DefName = "TestStruct2"

	s2.Fields = append(s2.Fields, jsonschema.Field{
		Name:  "Quux",
		Value: "abc",
		Tag:   `json:"quux" minLength:"3"`,
	})

	s.Fields = append(s.Fields, jsonschema.Field{
		Name:  "Other",
		Value: s2,
		Tag:   `json:"other"`,
	})

	s2.DefName = ""

	s.Fields = append(s.Fields, jsonschema.Field{
		Name:  "Another",
		Value: s2,
		Tag:   `json:"another"`,
	})

	sc, err := r.Reflect(s)
	require.NoError(t, err)

	assertjson.EqMarshal(t, `{
	  "title":"Test title","description":"Test description",
	  "definitions":{
		"JsonschemaGoTestEnumed":{"enum":["foo","bar"],"type":"string"},
		"JsonschemaGoTestPerson":{
		  "required":["lastName"],
		  "properties":{
			"birthDate":{"type":"string","format":"date"},
			"createdAt":{"type":"string","format":"date-time"},
			"date":{"type":"string","format":"date"},
			"deathDate":{"type":["null","string"],"format":"date"},
			"deletedAt":{"type":["null","string"],"format":"date-time"},
			"enumed":{"$ref":"#/definitions/JsonschemaGoTestEnumed"},
			"enumedPtr":{"$ref":"#/definitions/JsonschemaGoTestEnumed"},
			"firstName":{"type":"string"},"height":{"type":"integer"},
			"lastName":{"type":"string"},"meta":{},
			"role":{"description":"The role of person.","type":"string"}
		  },
		  "type":"object"
		},
		"TestStruct2":{
		  "title":"T2","properties":{"quux":{"minLength":3,"type":"string"}},
		  "type":"object"
		},
		"struct1":{
		  "title":"T2","properties":{"quux":{"minLength":3,"type":"string"}},
		  "type":"object"
		}
	  },
	  "properties":{
		"another":{"$ref":"#/definitions/struct1"},
		"b4r":{"minimum":3,"type":"integer"},
		"b4z":{"items":{"type":"integer"},"minItems":4,"type":["array","null"]},
		"fo0":{"minLength":3,"type":"string"},
		"other":{"$ref":"#/definitions/TestStruct2"},
		"pers":{"$ref":"#/definitions/JsonschemaGoTestPerson"},
		"recursion":{"$ref":"#"}
	  },
	  "type":"object"
	}`, sc)
}

func TestReflector_Reflect_StructEmbed(t *testing.T) {
	type dynamicInput struct {
		jsonschema.Struct

		// Type is a static field example.
		Type string `query:"type"`
	}

	type dynamicOutput struct {
		// Embedded jsonschema.Struct exposes dynamic fields for documentation.
		jsonschema.Struct

		jsonFields   map[string]interface{}
		headerFields map[string]string

		// Status is a static field example.
		Status string `json:"status"`
	}

	dynIn := dynamicInput{}
	dynIn.DefName = "DynIn123"
	dynIn.Struct.Fields = []jsonschema.Field{
		{Name: "Foo", Value: 123, Tag: `header:"foo" enum:"123,456,789"`},
		{Name: "Bar", Value: "abc", Tag: `query:"bar"`},
	}

	dynOut := dynamicOutput{}
	dynOut.DefName = "DynOut123"
	dynOut.Struct.Fields = []jsonschema.Field{
		{Name: "Foo", Value: 123, Tag: `header:"foo" enum:"123,456,789"`},
		{Name: "Bar", Value: "abc", Tag: `json:"bar"`},
	}

	type S struct {
		In  dynamicInput  `json:"in"`
		Out dynamicOutput `json:"out"`
	}

	s := S{
		In:  dynIn,
		Out: dynOut,
	}

	r := jsonschema.Reflector{}

	ss, err := r.Reflect(s, func(rc *jsonschema.ReflectContext) {
		rc.PropertyNameTag = "json"
		rc.PropertyNameAdditionalTags = []string{"header", "query"}
	})
	require.NoError(t, err)

	assertjson.EqMarshal(t, `{
	  "definitions":{
		"DynIn123":{
		  "properties":{
			"bar":{"type":"string"},
			"foo":{"enum":["123","456","789"],"type":"integer"},
			"type":{"type":"string"}
		  },
		  "type":"object"
		},
		"DynOut123":{
		  "properties":{
			"bar":{"type":"string"},
			"foo":{"enum":["123","456","789"],"type":"integer"},
			"status":{"type":"string"}
		  },
		  "type":"object"
		}
	  },
	  "properties":{
		"in":{"$ref":"#/definitions/DynIn123"},
		"out":{"$ref":"#/definitions/DynOut123"}
	  },
	  "type":"object"
	}`, ss)
}

func TestReflector_Reflect_StructExample(t *testing.T) {
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

	t.Run("standalone", func(t *testing.T) {
		schema, err := r.Reflect(s)
		require.NoError(t, err)

		assertjson.EqMarshal(t, `{
		  "title":"Test title","description":"Test description",
		  "properties":{"foo":{"minLength":3,"type":"string"}},"type":"object"
		}`, schema)
	})

	type MyStruct struct {
		jsonschema.Struct // Can be structPtr.

		Bar int `json:"bar"`

		Nested jsonschema.Struct `json:"nested"` // Can be nested.
	}

	ms := MyStruct{}
	ms.Nested = s
	ms.Struct = s

	t.Run("nested", func(t *testing.T) {
		schema, err := r.Reflect(ms)
		require.NoError(t, err)
		assertjson.EqMarshal(t, `{
		  "title":"Test title","description":"Test description",
		  "properties":{
			"bar":{"type":"integer"},"foo":{"minLength":3,"type":"string"},
			"nested":{"$ref":"#"}
		  },
		  "type":"object"
		}`, schema)
	})
}
