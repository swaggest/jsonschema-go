package jsonschema_test

import (
	"context"
	"encoding"
	"encoding/json"
	"mime/multipart"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
)

type Role struct {
	Level string
	Title string
}

func (r Role) MarshalText() ([]byte, error) {
	return []byte(r.Level + ":" + r.Title), nil
}

func (r *Role) UnmarshalText([]byte) error {
	r.Level = "l"
	r.Title = "t"

	return nil
}

type Entity struct {
	CreatedAt time.Time        `json:"createdAt"`
	DeletedAt *time.Time       `json:"deletedAt"`
	BirthDate jsonschema.Date  `json:"birthDate"`
	DeathDate *jsonschema.Date `json:"deathDate"`
	Meta      *json.RawMessage `json:"meta"`
}

type Person struct {
	Entity
	BirthDate string  `json:"date" format:"date"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName" required:"true"`
	Height    int     `json:"height"`
	Role      Role    `json:"role" description:"The role of person."`
	Enumed    Enumed  `json:"enumed"`
	EnumedPtr *Enumed `json:"enumedPtr"`
}

type Enumed string

func (e Enumed) Enum() []interface{} {
	return []interface{}{
		"foo", "bar",
	}
}

var (
	_ encoding.TextUnmarshaler = &Role{}
	_ encoding.TextMarshaler   = &Role{}
	_ jsonschema.Preparer      = Org{}
	_ jsonschema.Preparer      = &Person{}
)

func (p *Person) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.WithTitle("Person")

	return nil
}

type Org struct {
	ChiefOfMoral *Person  `json:"chiefOfMorale,omitempty"`
	Employees    []Person `json:"employees,omitempty"`
}

func (o Org) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.WithTitle("Organization")

	return nil
}

func TestReflector_Reflect_namedInterface(t *testing.T) {
	type s struct {
		Upload multipart.File `json:"upload"`
	}

	reflector := jsonschema.Reflector{}
	schema, err := reflector.Reflect(s{}, jsonschema.InterceptType(func(v reflect.Value, s *jsonschema.Schema) (bool, error) {
		if _, ok := v.Interface().(*multipart.File); ok {
			s.AddType(jsonschema.String)
			s.WithFormat("binary")

			return true, nil
		}

		return false, nil
	}))
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "definitions":{"MultipartFile":{"type":["null","string"],"format":"binary"}},
	  "properties":{"upload":{"$ref":"#/definitions/MultipartFile"}},
	  "type":"object"
	}`), schema)
}

func TestReflector_Reflect(t *testing.T) {
	reflector := jsonschema.Reflector{}
	schema, err := reflector.Reflect(Org{})
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`
{
  "title":"Organization",
  "definitions":{
	"JsonschemaGoTestEnumed":{"enum":["foo","bar"],"type":"string"},
	"JsonschemaGoTestPerson":{
	  "title":"Person","required":["lastName"],
	  "properties":{
		"birthDate":{"type":"string","format":"date"},
		"createdAt":{"type":"string","format":"date-time"},
		"date":{"type":"string","format":"date"},
		"deathDate":{"type":["null","string"],"format":"date"},
		"deletedAt":{"type":["null","string"],"format":"date-time"},
		"enumed":{"$ref":"#/definitions/JsonschemaGoTestEnumed"},
		"enumedPtr":{
		 "anyOf": [
          {
		   "$ref": "#/definitions/JsonschemaGoTestEnumed"
   		  }
		 ],
		 "type": "null"
		},
		"firstName":{"type":"string"},"height":{"type":"integer"},
		"lastName":{"type":"string"},"meta":{},
		"role":{
		  "$ref":"#/definitions/JsonschemaGoTestRole",
		  "description":"The role of person."
		}
	  },
	  "type":"object"
	},
	"JsonschemaGoTestRole":{"type":"string"}
  },
  "properties":{
	"chiefOfMorale":{
	 "anyOf": [
	  {
	   "$ref": "#/definitions/JsonschemaGoTestPerson",
	   "type": "object"
	  }
	 ],
	 "type": "null"
	},
	"employees":{"items":{"$ref":"#/definitions/JsonschemaGoTestPerson"},"type":"array"}
  },
  "type":"object"
}
`), schema)
}

func TestReflector_Reflect_inlineStruct(t *testing.T) {
	type structWithInline struct {
		Data struct {
			Deeper struct {
				A string `json:"a"`
			} `json:"deeper"`
		} `json:"data"`
	}

	reflector := jsonschema.Reflector{}
	schema, err := reflector.Reflect(new(structWithInline))
	require.NoError(t, err)

	j, err := json.MarshalIndent(schema, "", " ")
	require.NoError(t, err)

	assertjson.Equal(t, []byte(`
{
 "properties": {
  "data": {
   "properties": {
	"deeper": {
	 "properties": {
	  "a": {
	   "type": "string"
	  }
	 },
	 "type": "object"
	}
   },
   "type": "object"
  }
 },
 "type": "object"
}`), j, string(j))
}

func TestReflector_Reflect_rootNullable(t *testing.T) {
	type structWithInline struct {
		Data struct {
			Deeper struct {
				A string `json:"a"`
			} `json:"deeper"`
		} `json:"data"`
	}

	reflector := jsonschema.Reflector{}
	schema, err := reflector.Reflect(new(structWithInline), jsonschema.RootNullable)
	require.NoError(t, err)

	j, err := json.MarshalIndent(schema, "", " ")
	require.NoError(t, err)

	assertjson.Equal(t, []byte(`
{
 "properties": {
  "data": {
   "properties": {
	"deeper": {
	 "properties": {
	  "a": {
	   "type": "string"
	  }
	 },
	 "type": "object"
	}
   },
   "type": "object"
  }
 },
 "type": ["object", "null"]
}`), j, string(j))
}

func TestReflector_Reflect_collectDefinitions(t *testing.T) {
	reflector := jsonschema.Reflector{}

	schemas := map[string]jsonschema.Schema{}

	schema, err := reflector.Reflect(Org{}, jsonschema.CollectDefinitions(func(name string, schema jsonschema.Schema) {
		schemas[name] = schema
	}))
	require.NoError(t, err)

	j, err := json.MarshalIndent(schema, "", " ")
	require.NoError(t, err)

	assertjson.Equal(t, []byte(`
{
 "title": "Organization",
 "properties": {
  "chiefOfMorale": {
   "anyOf": [
    {
     "$ref": "#/definitions/JsonschemaGoTestPerson",
     "type": "object"
    }
   ],
   "type": "null"
  },
  "employees": {
   "items": {
	"$ref": "#/definitions/JsonschemaGoTestPerson"
   },
   "type": "array"
  }
 },
 "type": "object"
}`), j, string(j))

	assertjson.EqualMarshal(t, []byte(`
{
  "JsonschemaGoTestEnumed":{"enum":["foo","bar"],"type":"string"},
  "JsonschemaGoTestPerson":{
	"title":"Person","required":["lastName"],
	"properties":{
	  "birthDate":{"type":"string","format":"date"},
	  "createdAt":{"type":"string","format":"date-time"},
	  "date":{"type":"string","format":"date"},
	  "deathDate":{"type":["null","string"],"format":"date"},
	  "deletedAt":{"type":["null","string"],"format":"date-time"},
	  "enumed":{"$ref":"#/definitions/JsonschemaGoTestEnumed"},
	  "enumedPtr":{
	   "anyOf": [
	    {
	     "$ref": "#/definitions/JsonschemaGoTestEnumed"
	    }
	   ],
	   "type": "null"
      },
	  "firstName":{"type":"string"},"height":{"type":"integer"},
	  "lastName":{"type":"string"},"meta":{},
	  "role":{
		"$ref":"#/definitions/JsonschemaGoTestRole",
		"description":"The role of person."
	  }
	},
	"type":"object"
  },
  "JsonschemaGoTestRole":{"type":"string"}
}`), schemas)
}

func TestReflector_Reflect_recursiveStruct(t *testing.T) {
	type Rec struct {
		Val      string `json:"val"`
		Parent   *Rec   `json:"parent,omitempty"`
		Siblings []Rec  `json:"siblings,omitempty"`
	}

	s, err := (&jsonschema.Reflector{}).Reflect(Rec{})
	require.NoError(t, err)

	j, err := json.Marshal(s)
	require.NoError(t, err)

	assertjson.Equal(t, []byte(`{
		"properties": {
		 "parent":{
		  "anyOf": [
		   {
		    "$ref": "#"
		   }
		  ],
		  "type": "null"
		 },
   		 "siblings":{"items":{"$ref":"#"},"type":"array"},
		 "val":{"type":"string"}},"type":"object"
	}`), j, string(j))
}

func TestReflector_Reflect_mapping(t *testing.T) {
	type simpleTestReplacement struct {
		ID  uint64 `json:"id"`
		Cat string `json:"category"`
	}

	type deepReplacementTag struct {
		TestField1 string `json:"test_field_1" type:"number" format:"double"`
	}

	type testWrapParams struct {
		SimpleTestReplacement simpleTestReplacement `json:"simple_test_replacement"`
		DeepReplacementTag    deepReplacementTag    `json:"deep_replacement"`
	}

	rf := jsonschema.Reflector{}
	rf.AddTypeMapping(simpleTestReplacement{}, "")
	rf.InterceptDefName(func(t reflect.Type, defaultDefName string) string {
		return strings.TrimPrefix(defaultDefName, "JsonschemaGoTest")
	})

	s, err := rf.Reflect(testWrapParams{}, jsonschema.RootRef)
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "$ref":"#/definitions/TestWrapParams",
	  "definitions":{
		"DeepReplacementTag":{
		  "properties":{"test_field_1":{"type":"string","format":"double"}},
		  "type":"object"
		},
		"TestWrapParams":{
		  "properties":{
			"deep_replacement":{"$ref":"#/definitions/DeepReplacementTag"},
			"simple_test_replacement":{"type":"string"}
		  },
		  "type":"object"
		}
	  }
	}`), s)
}

func TestReflector_Reflect_map(t *testing.T) {
	type simpleDateTime struct {
		Time time.Time `json:"time"`
	}

	type mapDateTime struct {
		Items map[string]simpleDateTime `json:"items,omitempty"`
	}

	s, err := (&jsonschema.Reflector{}).Reflect(mapDateTime{}, jsonschema.RootRef)
	require.NoError(t, err)

	j, err := json.MarshalIndent(s, "", " ")
	require.NoError(t, err)

	assertjson.Equal(t, []byte(`{
        	            	 "$ref": "#/definitions/JsonschemaGoTestMapDateTime",
        	            	 "definitions": {
        	            	  "JsonschemaGoTestMapDateTime": {
        	            	   "properties": {
        	            	    "items": {
        	            	     "additionalProperties": {
        	            	      "$ref": "#/definitions/JsonschemaGoTestSimpleDateTime"
        	            	     },
        	            	     "type": "object"
        	            	    }
        	            	   },
        	            	   "type": "object"
        	            	  },
        	            	  "JsonschemaGoTestSimpleDateTime": {
        	            	   "properties": {
        	            	    "time": {
        	            	     "type": "string",
        	            	     "format": "date-time"
        	            	    }
        	            	   },
        	            	   "type": "object"
        	            	  }
        	            	 }
        	            	}`), j, string(j))
}

func TestReflector_Reflect_pointer_envelop(t *testing.T) {
	type St struct {
		A int `json:"a"`
	}

	type NamedMap map[string]St

	type Cont struct {
		PtrOmitempty      *St            `json:"ptrOmitempty,omitempty"`
		Ptr               *St            `json:"ptr"`
		Val               St             `json:"val"`
		SliceOmitempty    []St           `json:"sliceOmitempty,omitempty" minItems:"3"`
		Slice             []St           `json:"slice" minItems:"2"`
		PointerMap        *map[string]St `json:"pointerMap" minProperties:"2"`
		MapOmitempty      map[string]St  `json:"mapOmitempty,omitempty" minProperties:"3"`
		Map               map[string]St  `json:"map" minProperties:"2"`
		NamedMapOmitempty NamedMap       `json:"namedMapOmitempty,omitempty" minProperties:"1"`
		NamedMap          NamedMap       `json:"namedMap" minProperties:"5"`
		PointerNamedMap   *NamedMap      `json:"pointerNamedMap" minProperties:"5"`
	}

	s, err := (&jsonschema.Reflector{}).Reflect(Cont{}, func(rc *jsonschema.ReflectContext) {
		rc.EnvelopNullability = true
	})
	require.NoError(t, err)

	j, err := json.MarshalIndent(s, "", " ")
	require.NoError(t, err)

	assertjson.Equal(t, []byte(`{
        	            	 "definitions": {
        	            	  "JsonschemaGoTestNamedMap": {
        	            	   "additionalProperties": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "type": "object"
        	            	  },
        	            	  "JsonschemaGoTestSt": {
        	            	   "properties": {
        	            	    "a": {
        	            	     "type": "integer"
        	            	    }
        	            	   },
        	            	   "type": "object"
        	            	  }
        	            	 },
        	            	 "properties": {
        	            	  "map": {
        	            	   "minProperties": 2,
        	            	   "additionalProperties": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "type": "object"
        	            	  },
        	            	  "mapOmitempty": {
        	            	   "minProperties": 3,
        	            	   "additionalProperties": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "type": "object"
        	            	  },
                              "pointerMap": {
                               "additionalProperties": {
                                "$ref": "#/definitions/JsonschemaGoTestSt"
                               },
                               "minProperties": 2,
                               "type": [
                                 "null",
                                 "object"
                               ]
                              },
        	            	  "namedMap": {
        	            	   "minProperties": 5,
							   "$ref": "#/definitions/JsonschemaGoTestNamedMap"
        	            	  },
        	            	  "namedMapOmitempty": {
        	            	   "$ref": "#/definitions/JsonschemaGoTestNamedMap",
        	            	   "minProperties": 1
        	            	  },
                              "pointerNamedMap": {
                               "anyOf": [
                                {
                                 "$ref": "#/definitions/JsonschemaGoTestNamedMap"
                                }
                               ],
                               "minProperties": 5,
                               "type": "null"
                              },
        	            	  "ptr": {
        	            	   "anyOf": [
        	            	    {
        	            	     "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	    }
        	            	   ],
							   "type": "null"
        	            	  },
        	            	  "ptrOmitempty": {
                               "anyOf": [
                                {
                                 "$ref": "#/definitions/JsonschemaGoTestSt",
                                 "type": "object"
                                }
                               ],
                               "type": "null"
        	            	  },
        	            	  "slice": {
        	            	   "items": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "minItems": 2,
        	            	   "type": "array"
        	            	  },
        	            	  "sliceOmitempty": {
        	            	   "items": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "minItems": 3,
        	            	   "type": "array"
        	            	  },
        	            	  "val": {
        	            	   "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	  }
        	            	 },
        	            	 "type": "object"
        	            	}`), j, string(j))
}

func TestReflector_Reflect_pointer(t *testing.T) {
	type St struct {
		A int `json:"a"`
	}

	type NamedMap map[string]St

	type Cont struct {
		PtrOmitempty      *St            `json:"ptrOmitempty,omitempty"`
		Ptr               *St            `json:"ptr"`
		Val               St             `json:"val"`
		SliceOmitempty    []St           `json:"sliceOmitempty,omitempty" minItems:"3"`
		Slice             []St           `json:"slice" minItems:"2"`
		MapOmitempty      map[string]St  `json:"mapOmitempty,omitempty" minProperties:"3"`
		Map               map[string]St  `json:"map" minProperties:"2"`
		PointerMap        *map[string]St `json:"pointerMap" minProperties:"2"`
		NamedMapOmitempty NamedMap       `json:"namedMapOmitempty,omitempty" minProperties:"1"`
		NamedMap          NamedMap       `json:"namedMap" minProperties:"5"`
		PointerNamedMap   *NamedMap      `json:"pointerNamedMap" minProperties:"5"`
	}

	s, err := (&jsonschema.Reflector{}).Reflect(Cont{})
	require.NoError(t, err)

	j, err := json.MarshalIndent(s, "", " ")
	require.NoError(t, err)

	assertjson.Equal(t, []byte(`{
        	            	 "definitions": {
        	            	  "JsonschemaGoTestNamedMap": {
        	            	   "additionalProperties": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "type": "object"
        	            	  },
        	            	  "JsonschemaGoTestSt": {
        	            	   "properties": {
        	            	    "a": {
        	            	     "type": "integer"
        	            	    }
        	            	   },
        	            	   "type": "object"
        	            	  }
        	            	 },
        	            	 "properties": {
        	            	  "map": {
        	            	   "minProperties": 2,
        	            	   "additionalProperties": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "type": "object"
        	            	  },
        	            	  "mapOmitempty": {
        	            	   "minProperties": 3,
        	            	   "additionalProperties": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "type": "object"
        	            	  },
                              "pointerMap": {
                               "additionalProperties": {
                                "$ref": "#/definitions/JsonschemaGoTestSt"
                               },
                               "minProperties": 2,
                               "type": [
                                "null",
                                "object"
                               ]
                              },
        	            	  "namedMap": {
        	            	   "$ref": "#/definitions/JsonschemaGoTestNamedMap",
        	            	   "minProperties": 5
        	            	  },
        	            	  "namedMapOmitempty": {
        	            	   "$ref": "#/definitions/JsonschemaGoTestNamedMap",
        	            	   "minProperties": 1
        	            	  },
                              "pointerNamedMap": {
                               "anyOf": [
                                {
                                 "$ref": "#/definitions/JsonschemaGoTestNamedMap"
                                }
                               ],
                               "minProperties": 5,
                               "type": "null"
                              },
        	            	  "ptr": {
                               "anyOf": [
                                {
 								 "$ref": "#/definitions/JsonschemaGoTestSt"
                                }
                               ],
                               "type": "null"
        	            	  },
        	            	  "ptrOmitempty": {
                               "anyOf": [
                                {
                                 "$ref": "#/definitions/JsonschemaGoTestSt",
                                 "type": "object"
                                }
                               ],
                               "type": "null"
        	            	  },
        	            	  "slice": {
        	            	   "items": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "minItems": 2,
        	            	   "type": "array"
        	            	  },
        	            	  "sliceOmitempty": {
        	            	   "items": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "minItems": 3,
        	            	   "type": "array"
        	            	  },
        	            	  "val": {
        	            	   "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	  }
        	            	 },
        	            	 "type": "object"
        	            	}`), j, string(j))
}

var (
	_ jsonschema.RawExposer = ISOWeek("")
	_ jsonschema.Exposer    = ISOCountry("")
)

// ISOWeek is an ISO week.
type ISOWeek string

// JSONSchemaBytes returns JSON Schema definition.
func (ISOWeek) JSONSchemaBytes() ([]byte, error) {
	return []byte(`{
		"type": "string",
		"examples": ["2018-W43"],
		"description": "ISO Week",
		"pattern": "^[0-9]{4}-W(0[1-9]|[1-4][0-9]|5[0-3])$"
	}`), nil
}

type ISOCountry string

// JSONSchemaBytes returns JSON Schema definition.
func (ISOCountry) JSONSchema() (jsonschema.Schema, error) {
	s := jsonschema.Schema{}

	s.AddType(jsonschema.String)
	s.WithExamples("US")
	s.WithDescription("ISO Country")
	s.WithPattern("^[a-zA-Z]{2}$")
	s.WithMinLength(2)
	s.WithMaxLength(2)

	return s, nil
}

func TestExposer(t *testing.T) {
	type Some struct {
		Week    ISOWeek    `json:"week"`
		Country ISOCountry `json:"country" deprecated:"true"`
	}

	s, err := (&jsonschema.Reflector{}).Reflect(Some{})
	require.NoError(t, err)

	j, err := json.MarshalIndent(s, "", " ")
	require.NoError(t, err)

	assertjson.Equal(t, []byte(`{
        	            	 "definitions": {
        	            	  "JsonschemaGoTestISOCountry": {
        	            	   "description": "ISO Country",
        	            	   "examples": [
        	            	    "US"
        	            	   ],
        	            	   "maxLength": 2,
        	            	   "minLength": 2,
        	            	   "pattern": "^[a-zA-Z]{2}$",
        	            	   "type": "string"
        	            	  },
        	            	  "JsonschemaGoTestISOWeek": {
        	            	   "description": "ISO Week",
        	            	   "examples": [
        	            	    "2018-W43"
        	            	   ],
        	            	   "pattern": "^[0-9]{4}-W(0[1-9]|[1-4][0-9]|5[0-3])$",
        	            	   "type": "string"
        	            	  }
        	            	 },
        	            	 "properties": {
        	            	  "country": {
        	            	   "$ref": "#/definitions/JsonschemaGoTestISOCountry",
        	            	   "deprecated": true
        	            	  },
        	            	  "week": {
        	            	   "$ref": "#/definitions/JsonschemaGoTestISOWeek"
        	            	  }
        	            	 },
        	            	 "type": "object"
        	            	}`), j, string(j))
}

type Identity struct {
	ID string `path:"id"`
}

type Data []string

type PathParamAndBody struct {
	Identity
	Data
}

func TestSkipEmbeddedMapsSlices(t *testing.T) {
	reflector := jsonschema.Reflector{}

	s, err := reflector.Reflect(new(PathParamAndBody),
		jsonschema.PropertyNameTag("path"), jsonschema.SkipEmbeddedMapsSlices)

	require.NoError(t, err)

	j, err := json.Marshal(s)
	require.NoError(t, err)

	expected := []byte(`{"properties":{"id":{"type":"string"}},"type":"object"}`)
	assertjson.Equal(t, expected, j, string(j))

	s, err = reflector.Reflect(new(PathParamAndBody))

	require.NoError(t, err)

	j, err = json.Marshal(s)
	require.NoError(t, err)

	expected = []byte(`{"items":{"type":"string"},"type":["null","array"]}`)
	assertjson.Equal(t, expected, j, string(j))
}

func TestReflector_Reflect_propertyNameMapping(t *testing.T) {
	reflector := jsonschema.Reflector{}

	type Test struct {
		ID               int    `minimum:"123" default:"200"`
		Name             string `minLength:"10"`
		UntaggedUnmapped int
		unexported       int
	}

	s, err := reflector.Reflect(new(Test),
		jsonschema.PropertyNameMapping(map[string]string{
			"ID":   "ident",
			"Name": "last_name",
		}))

	require.NoError(t, err)

	j, err := json.Marshal(s)
	require.NoError(t, err)

	expected := []byte(`{"properties":{"ident":{"minimum":123,"type":"integer","default":200},` +
		`"last_name":{"minLength":10,"type":"string"}},"type":"object"}`)
	assertjson.Equal(t, expected, j, string(j))

	require.NoError(t, err)
}

func TestMakePropertyNameMapping(t *testing.T) {
	type Test struct {
		ID   int    `path:"ident" minimum:"123" default:"200"`
		Name string `path:"last_name" minLength:"10"`
	}

	assert.Equal(
		t,
		map[string]string{"ID": "ident", "Name": "last_name"},
		jsonschema.MakePropertyNameMapping(new(Test), "path"),
	)
}

type nullFloat struct {
	Valid bool
	Float float64
}

var _ jsonschema.Preparer = nullFloat{}

func (n nullFloat) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.TypeEns().WithSliceOfSimpleTypeValues(jsonschema.Null, jsonschema.Number)

	schema.TypeEns().SimpleTypes = nil

	return nil
}

func TestInterceptType(t *testing.T) {
	r := jsonschema.Reflector{}

	s, err := r.Reflect(nullFloat{})
	assert.NoError(t, err)
	assertjson.EqualMarshal(t, []byte(`{"type":["null", "number"]}`), s)
}

func TestReflector_Reflect_Ref(t *testing.T) {
	type Symbol string

	type topTracesInput struct {
		RootSymbol Symbol `json:"rootSymbol" minLength:"5" example:"my_func" default:"main()"`
	}

	r := jsonschema.Reflector{}
	s, err := r.Reflect(topTracesInput{})
	assert.NoError(t, err)
	assertjson.EqualMarshal(t, []byte(`{
	  "definitions":{"JsonschemaGoTestSymbol":{"type":"string"}},
	  "properties":{
		"rootSymbol":{
		  "$ref":"#/definitions/JsonschemaGoTestSymbol","default":"main()",
		  "examples":["my_func"],"minLength":5
		}
	  },
	  "type":"object"
	}`), s)
}

func TestReflector_Reflect_MapOfOptionals(t *testing.T) {
	type Symbol string

	type Optionals struct {
		Map   map[Symbol]*float64 `json:"map"`
		Slice []*float64          `json:"slice"`
	}

	r := jsonschema.Reflector{}
	s, err := r.Reflect(Optionals{})
	assert.NoError(t, err)
	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{
		"map":{
		  "additionalProperties":{"type":["null","number"]},
		  "type":"object"
		},
		"slice":{"items":{"type":["null","number"]},"type":"array"}
	  },
	  "type":"object"
	}`), s)
}

func TestReflector_Reflect_InlineValue(t *testing.T) {
	type InlineValues struct {
		One   string `json:"one" default:"un"`
		Two   string `json:"two" const:"deux"`
		Three string `json:"three" default:"trois" const:"3"`
		Four  int    `json:"four" const:"4"`
	}

	r := jsonschema.Reflector{}
	s, err := r.Reflect(InlineValues{})
	assert.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{
	    "one":{"default":"un","type":"string"},
	    "two":{"const":"deux","type":"string"},
	    "three":{"default":"trois","const":"3","type":"string"},
	    "four":{"const":4,"type":"integer"}
	  },
	  "type":"object"
	}`), s)
}

type WithSubSchemas struct {
	Foo string `json:"foo"`
}

func (WithSubSchemas) JSONSchemaOneOf() []interface{} {
	return []interface{}{
		Person{},
		Enumed(""),
		"",
	}
}

func (WithSubSchemas) JSONSchemaAnyOf() []interface{} {
	return []interface{}{
		"",
		123,
	}
}

func (WithSubSchemas) JSONSchemaAllOf() []interface{} {
	return []interface{}{
		1.23,
		123,
	}
}

func (WithSubSchemas) JSONSchemaNot() interface{} {
	return Person{}
}

func (WithSubSchemas) JSONSchemaIf() interface{} {
	return Entity{}
}

func (WithSubSchemas) JSONSchemaThen() interface{} {
	return Role{}
}

func (WithSubSchemas) JSONSchemaElse() interface{} {
	return Person{}
}

func TestReflector_Reflect_sub_schema(t *testing.T) {
	r := jsonschema.Reflector{}

	s, err := r.Reflect(WithSubSchemas{}, jsonschema.StripDefinitionNamePrefix("JsonschemaGoTest"))
	assert.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "definitions":{
		"Entity":{
		  "properties":{
			"birthDate":{"type":"string","format":"date"},
			"createdAt":{"type":"string","format":"date-time"},
			"deathDate":{"type":["null","string"],"format":"date"},
			"deletedAt":{"type":["null","string"],"format":"date-time"},"meta":{}
		  },
		  "type":"object"
		},
		"Enumed":{"enum":["foo","bar"],"type":"string"},
		"Person":{
		  "required":["lastName"],
		  "properties":{
			"birthDate":{"type":"string","format":"date"},
			"createdAt":{"type":"string","format":"date-time"},
			"date":{"type":"string","format":"date"},
			"deathDate":{"type":["null","string"],"format":"date"},
			"deletedAt":{"type":["null","string"],"format":"date-time"},
			"enumed":{"$ref":"#/definitions/Enumed"},
			"enumedPtr":{
			 "anyOf": [
			  {
			   "$ref": "#/definitions/Enumed"
			  }
			 ],
			 "type": "null"
			},
			"firstName":{"type":"string"},"height":{"type":"integer"},
			"lastName":{"type":"string"},"meta":{},
			"role":{"$ref":"#/definitions/Role","description":"The role of person."}
		  },
		  "type":"object"
		},
		"Role":{"type":"string"}
	  },
	  "properties":{"foo":{"type":"string"}},"type":"object",
	  "if":{"$ref":"#/definitions/Entity"},"then":{"$ref":"#/definitions/Role"},
	  "else":{"$ref":"#/definitions/Person"},
	  "allOf":[{"type":"number"},{"type":"integer"}],
	  "anyOf":[{"type":"string"},{"type":"integer"}],
	  "oneOf":[
		{"$ref":"#/definitions/Person"},{"$ref":"#/definitions/Enumed"},
		{"type":"string"}
	  ],
	  "not":{"$ref":"#/definitions/Person"}
	}`), s)
}

func TestReflector_Reflect_jsonEmptyName(t *testing.T) {
	type Test struct {
		Foo string `json:",omitempty"`
		Bar int    `json:""`
		Baz bool   `json:"-"`
	}

	r := jsonschema.Reflector{}

	s, err := r.Reflect(Test{})
	assert.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{"Bar":{"type":"integer"},"Foo":{"type":"string"}},
	  "type":"object"
	}`), s)
}

func TestReflector_Reflect_processWithoutTags_true(t *testing.T) {
	type Test struct {
		Foo string
		Bar int
		Baz bool `json:"baz"`
		qux string
	}

	r := jsonschema.Reflector{}

	s, err := r.Reflect(Test{}, jsonschema.ProcessWithoutTags)
	assert.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{"Bar":{"type":"integer"},"Foo":{"type":"string"},"baz":{"type":"boolean"}},
	  "type":"object"
	}`), s)
}

func TestReflector_Reflect_processWithoutTags_tolerateUnknownTypes(t *testing.T) {
	type Test struct {
		Foo  string
		Bar  int
		Baz  bool `json:"baz"`
		Fun  func()
		Chan chan bool
		qux  string
	}

	r := jsonschema.Reflector{}

	s, err := r.Reflect(Test{}, jsonschema.ProcessWithoutTags, jsonschema.SkipUnsupportedProperties)
	assert.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{"Bar":{"type":"integer"},"Foo":{"type":"string"},"baz":{"type":"boolean"}},
	  "type":"object"
	}`), s)
}

func TestReflector_Reflect_processWithoutTags_false(t *testing.T) {
	type Test struct {
		Foo string
		Bar int
		Baz bool `json:"baz"`
		qux string
	}

	r := jsonschema.Reflector{}

	s, err := r.Reflect(Test{})
	assert.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{"baz":{"type":"boolean"}},
	  "type":"object"
	}`), s)
}

func TestReflector_Reflect_parentTags(t *testing.T) {
	type Test struct {
		Foo string   `json:"foo"`
		_   struct{} `title:"Test"` // Tags of unnamed field are applied to parent schema.

		// There can be more than one field to set up parent schema.
		// Types of such fields are not relevant, only tags matter.
		_ string `additionalProperties:"false" description:"This is a test."`
	}

	r := jsonschema.Reflector{}

	s, err := r.Reflect(Test{})
	assert.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "title":"Test","description":"This is a test.","additionalProperties":false,
	  "properties":{"foo":{"type":"string"}},"type":"object"
	}`), s)

	// Failure scenarios.
	_, err = r.Reflect(struct {
		_ string `additionalProperties:"abc"`
	}{})
	assert.EqualError(t, err, "failed to parse bool value abc in tag additionalProperties: strconv.ParseBool: parsing \"abc\": invalid syntax")

	_, err = r.Reflect(struct {
		_ string `minProperties:"abc"`
	}{})
	assert.EqualError(t, err, "failed to parse int value abc in tag minProperties: strconv.ParseInt: parsing \"abc\": invalid syntax")
}

func TestReflector_Reflect_parentTagsFiltered(t *testing.T) {
	type Test struct {
		Foo string   `json:"foo" query:"foo"`
		_   struct{} `title:"Test"` // Tags of unnamed field are applied to parent schema.

		// There can be more than one field to set up parent schema.
		// Types of such fields are not relevant, only tags matter.
		_ string `query:"_" additionalProperties:"false" description:"This is a test."`
	}

	r := jsonschema.Reflector{}

	s, err := r.Reflect(Test{}, func(rc *jsonschema.ReflectContext) {
		rc.UnnamedFieldWithTag = true
		rc.PropertyNameTag = "json"
	})
	assert.NoError(t, err)

	// No parent schema update for json, as tag is missing in unnamed field.
	assertjson.EqualMarshal(t, []byte(`{"properties":{"foo":{"type":"string"}},"type":"object"}`), s)

	s, err = r.Reflect(Test{}, func(rc *jsonschema.ReflectContext) {
		rc.UnnamedFieldWithTag = true
		rc.PropertyNameTag = "query"
	})
	assert.NoError(t, err)

	// Parent schema is updated for query, as tag is present in unnamed field.
	assertjson.EqualMarshal(t, []byte(`{
	  "description":"This is a test.","additionalProperties":false,
	  "properties":{"foo":{"type":"string"}},"type":"object"
	}`), s)

	// Failure scenarios.
	_, err = r.Reflect(struct {
		_ string `additionalProperties:"abc"`
	}{})
	assert.EqualError(t, err, "failed to parse bool value abc in tag additionalProperties: strconv.ParseBool: parsing \"abc\": invalid syntax")

	_, err = r.Reflect(struct {
		_ string `minProperties:"abc"`
	}{})
	assert.EqualError(t, err, "failed to parse int value abc in tag minProperties: strconv.ParseInt: parsing \"abc\": invalid syntax")
}

func TestReflector_Reflect_context(t *testing.T) {
	type ctxKey struct{}

	type Test struct {
		Foo string `json:"foo"`
	}

	r := jsonschema.Reflector{}

	_, err := r.Reflect(new(Test),
		func(rc *jsonschema.ReflectContext) {
			rc.Context = context.WithValue(rc.Context, ctxKey{}, true)
		},
		func(rc *jsonschema.ReflectContext) {
			assert.Equal(t, true, rc.Value(ctxKey{}))
		},
	)

	require.NoError(t, err)
}

func TestOneOf(t *testing.T) {
	r := jsonschema.Reflector{}

	type Test struct {
		Foo jsonschema.OneOfExposer `json:"foo"`
		Bar jsonschema.OneOfExposer `json:"bar"`
	}

	tt := Test{
		Foo: jsonschema.OneOf(1.23, "abc"),
		Bar: jsonschema.OneOf(123, true),
	}

	s, err := r.Reflect(tt, jsonschema.RootRef)
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "$ref":"#/definitions/JsonschemaGoTestTest",
	  "definitions":{
		"JsonschemaGoTestTest":{
		  "properties":{
			"bar":{"oneOf":[{"type":"integer"},{"type":"boolean"}]},
			"foo":{"oneOf":[{"type":"number"},{"type":"string"}]}
		  },
		  "type":"object"
		}
	  }
	}`), s)

	s, err = r.Reflect(jsonschema.OneOf(123, true), jsonschema.RootRef)
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{"oneOf":[{"type":"integer"},{"type":"boolean"}]}`), s)
}

func TestAnyOf(t *testing.T) {
	r := jsonschema.Reflector{}

	type Test struct {
		Foo jsonschema.AnyOfExposer `json:"foo"`
		Bar jsonschema.AnyOfExposer `json:"bar"`
	}

	tt := Test{
		Foo: jsonschema.AnyOf(1.23, "abc"),
		Bar: jsonschema.AnyOf(123, true),
	}

	s, err := r.Reflect(tt, jsonschema.RootRef)
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "$ref":"#/definitions/JsonschemaGoTestTest",
	  "definitions":{
		"JsonschemaGoTestTest":{
		  "properties":{
			"bar":{"anyOf":[{"type":"integer"},{"type":"boolean"}]},
			"foo":{"anyOf":[{"type":"number"},{"type":"string"}]}
		  },
		  "type":"object"
		}
	  }
	}`), s)

	s, err = r.Reflect(jsonschema.AnyOf(123, true), jsonschema.RootRef)
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{"anyOf":[{"type":"integer"},{"type":"boolean"}]}`), s)
}

func TestAllOf(t *testing.T) {
	r := jsonschema.Reflector{}

	type Test struct {
		Foo jsonschema.AllOfExposer `json:"foo"`
		Bar jsonschema.AllOfExposer `json:"bar"`
	}

	tt := Test{
		Foo: jsonschema.AllOf(1.23, "abc"),
		Bar: jsonschema.AllOf(123, true),
	}

	s, err := r.Reflect(tt, jsonschema.RootRef)
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "$ref":"#/definitions/JsonschemaGoTestTest",
	  "definitions":{
		"JsonschemaGoTestTest":{
		  "properties":{
			"bar":{"allOf":[{"type":"integer"},{"type":"boolean"}]},
			"foo":{"allOf":[{"type":"number"},{"type":"string"}]}
		  },
		  "type":"object"
		}
	  }
	}`), s)

	s, err = r.Reflect(jsonschema.AllOf(123, true), jsonschema.RootRef)
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{"allOf":[{"type":"integer"},{"type":"boolean"}]}`), s)
}

type withTextMarshaler int

func (w *withTextMarshaler) UnmarshalText(_ []byte) error {
	*w = 1

	return nil
}

func (w withTextMarshaler) MarshalText() ([]byte, error) {
	return []byte("bar"), nil
}

func TestReflector_Reflect_defaultTextMarshaler(t *testing.T) {
	type test struct {
		Foo withTextMarshaler `json:"foo" default:"bar" example:"baz"`
	}

	v, err := json.Marshal(test{})
	require.NoError(t, err)

	assert.Equal(t, `{"foo":"bar"}`, string(v))

	r := jsonschema.Reflector{}

	s, err := r.Reflect(test{}, jsonschema.RootRef)
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "$ref":"#/definitions/JsonschemaGoTestTest",
	  "definitions":{
		"JsonschemaGoTestTest":{
		  "properties":{
			"foo":{
			  "$ref":"#/definitions/JsonschemaGoTestWithTextMarshaler",
			  "default":"bar","examples":["baz"]
			}
		  },
		  "type":"object"
		},
		"JsonschemaGoTestWithTextMarshaler":{"type":"string"}
	  }
	}`), s)
}

func TestReflector_Reflect_skipNonConstraints(t *testing.T) {
	type test struct {
		Foo withTextMarshaler `json:"foo" default:"bar" example:"baz"`
		Du  time.Duration     `json:"du" default:"10s"`
	}

	v, err := json.Marshal(test{})
	require.NoError(t, err)

	assert.Equal(t, `{"foo":"bar","du":0}`, string(v))

	r := jsonschema.Reflector{}

	s, err := r.Reflect(test{}, jsonschema.RootRef, func(rc *jsonschema.ReflectContext) {
		rc.SkipNonConstraints = true
	})
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "$ref":"#/definitions/JsonschemaGoTestTest",
	  "definitions":{
		"JsonschemaGoTestTest":{
		  "properties":{
			"du":{"$ref":"#/definitions/TimeDuration"},
			"foo":{"$ref":"#/definitions/JsonschemaGoTestWithTextMarshaler"}
		  },
		  "type":"object"
		},
		"JsonschemaGoTestWithTextMarshaler":{"type":"string"},
		"TimeDuration":{"type":"integer"}
	  }
	}`), s)
}

func TestReflector_Reflect_examples(t *testing.T) {
	type WantExample struct {
		A string   `json:"a" example:"example of a"`
		B []string `json:"b" example:"example of b"`
		C int      `json:"c" examples:"[\"foo\", 2, 3]" example:"123"`
	}

	reflector := jsonschema.Reflector{}
	schema, err := reflector.Reflect(WantExample{})
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{
		"a":{"examples":["example of a"],"type":"string"},
		"b":{
		  "items":{"examples":["example of b"],"type":"string"},
		  "type":"array"
		},
		"c":{"examples":[123,"foo",2,3],"type":"integer"}
	  },
	  "type":"object"
	}`), schema)
}

func TestReflector_Reflect_namedSlice(t *testing.T) {
	type PanicType []string

	type PanicStruct struct {
		IPPolicy PanicType `json:"ip_policy" example:"127.0.0.1"`
	}

	reflector := jsonschema.Reflector{}
	schema, err := reflector.Reflect(PanicStruct{})
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "definitions":{
		"JsonschemaGoTestPanicType":{"items":{"type":"string"},"type":"array"}
	  },
	  "properties":{"ip_policy":{"$ref":"#/definitions/JsonschemaGoTestPanicType"}},
	  "type":"object"
	}`), schema)
}
