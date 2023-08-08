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
		"struct1":{
		  "title":"T2","properties":{"quux":{"minLength":3,"type":"string"}},
		  "type":"object"
		}
	  },
	  "properties":{
		"b4r":{"minimum":3,"type":"integer"},
		"b4z":{"items":{"type":"integer"},"minItems":4,"type":["array","null"]},
		"fo0":{"minLength":3,"type":"string"},
		"other":{"$ref":"#/definitions/struct1"},
		"pers":{"$ref":"#/definitions/JsonschemaGoTestPerson"},
		"recursion":{"$ref":"#"}
	  },
	  "type":"object"
	}`, sc)
}
