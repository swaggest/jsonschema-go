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
	schema, err := reflector.Reflect(s{}, jsonschema.InterceptSchema(
		func(params jsonschema.InterceptSchemaParams) (stop bool, err error) {
			assert.NotNil(t, params.Context)

			if _, ok := params.Value.Interface().(*multipart.File); ok {
				params.Schema.AddType(jsonschema.String)
				params.Schema.WithFormat("binary")

				return true, nil
			}

			return false, nil
		},
	))
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

	assertjson.EqualMarshal(t, []byte(`{
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
			"enumedPtr":{"$ref":"#/definitions/JsonschemaGoTestEnumed"},
			"firstName":{"type":"string"},"height":{"type":"integer"},
			"lastName":{"type":"string"},"meta":{},
			"role":{"description":"The role of person.","type":"string"}
		  },
		  "type":"object"
		}
	  },
	  "properties":{
		"chiefOfMorale":{"$ref":"#/definitions/JsonschemaGoTestPerson"},
		"employees":{"items":{"$ref":"#/definitions/JsonschemaGoTestPerson"},"type":"array"}
	  },
	  "type":"object"
	}`), schema)
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

func TestReflector_Reflect_structDefPtr(t *testing.T) {
	type person struct {
		Name string `json:"name"`
	}

	type org struct {
		P1 *person `json:"p1,omitempty"`
		P2 *person `json:"p2"`
	}

	reflector := jsonschema.Reflector{}
	s, err := reflector.Reflect(org{})

	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "definitions":{
		"JsonschemaGoTestPerson":{"properties":{"name":{"type":"string"}},"type":"object"}
	  },
	  "properties":{
		"p1":{"$ref":"#/definitions/JsonschemaGoTestPerson"},
		"p2":{"$ref":"#/definitions/JsonschemaGoTestPerson"}
	  },
	  "type":"object"
	}`), s)

	s, err = reflector.Reflect(org{}, func(rc *jsonschema.ReflectContext) {
		rc.EnvelopNullability = true
	})

	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "definitions":{
		"JsonschemaGoTestPerson":{"properties":{"name":{"type":"string"}},"type":"object"}
	  },
	  "properties":{
		"p1":{"$ref":"#/definitions/JsonschemaGoTestPerson"},
		"p2":{
		  "anyOf":[{"type":"null"},{"$ref":"#/definitions/JsonschemaGoTestPerson"}]
		}
	  },
	  "type":"object"
	}`), s)
}

func TestReflector_Reflect_collectDefinitions(t *testing.T) {
	reflector := jsonschema.Reflector{}

	schemas := map[string]jsonschema.Schema{}

	schema, err := reflector.Reflect(Org{}, jsonschema.CollectDefinitions(func(name string, schema jsonschema.Schema) {
		schemas[name] = schema
	}))
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "title":"Organization",
	  "properties":{
		"chiefOfMorale":{"$ref":"#/definitions/JsonschemaGoTestPerson"},
		"employees":{"items":{"$ref":"#/definitions/JsonschemaGoTestPerson"},"type":"array"}
	  },
	  "type":"object"
	}`), schema)

	assertjson.EqualMarshal(t, []byte(`{
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
		  "enumedPtr":{"$ref":"#/definitions/JsonschemaGoTestEnumed"},
		  "firstName":{"type":"string"},"height":{"type":"integer"},
		  "lastName":{"type":"string"},"meta":{},
		  "role":{"description":"The role of person.","type":"string"}
		},
		"type":"object"
	  }
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

	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{
		"parent":{"$ref":"#"},"siblings":{"items":{"$ref":"#"},"type":"array"},
		"val":{"type":"string"}
	  },
	  "type":"object"
	}`), s)
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
	rf.DefaultOptions = append(rf.DefaultOptions, jsonschema.InterceptDefName(
		func(_ reflect.Type, defaultDefName string) string {
			return strings.TrimPrefix(defaultDefName, "JsonschemaGoTest")
		},
	))

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

	assertjson.EqualMarshal(t, []byte(`{
	  "$ref":"#/definitions/JsonschemaGoTestMapDateTime",
	  "definitions":{
		"JsonschemaGoTestMapDateTime":{
		  "properties":{
			"items":{
			  "additionalProperties":{"$ref":"#/definitions/JsonschemaGoTestSimpleDateTime"},
			  "type":"object"
			}
		  },
		  "type":"object"
		},
		"JsonschemaGoTestSimpleDateTime":{
		  "properties":{"time":{"type":"string","format":"date-time"}},
		  "type":"object"
		}
	  }
	}`), s)
}

func TestReflector_Reflect_pointer_envelop(t *testing.T) {
	type St struct {
		A int `json:"a"`
	}

	type NamedMap map[string]St

	type Cont struct {
		PtrOmitempty      *St           `json:"ptrOmitempty,omitempty"`
		Ptr               *St           `json:"ptr"`
		Val               St            `json:"val"`
		SliceOmitempty    []St          `json:"sliceOmitempty,omitempty" minItems:"3"`
		Slice             []St          `json:"slice" minItems:"2"`
		MapOmitempty      map[string]St `json:"mapOmitempty,omitempty" minProperties:"3"`
		Map               map[string]St `json:"map" minProperties:"2"`
		NamedMapOmitempty NamedMap      `json:"namedMapOmitempty,omitempty" minProperties:"1"`
		NamedMap          NamedMap      `json:"namedMap" minProperties:"5"`
	}

	s, err := (&jsonschema.Reflector{}).Reflect(Cont{}, func(rc *jsonschema.ReflectContext) {
		rc.EnvelopNullability = true
	})
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "definitions":{
		"JsonschemaGoTestNamedMap":{
		  "additionalProperties":{"$ref":"#/definitions/JsonschemaGoTestSt"},
		  "type":"object"
		},
		"JsonschemaGoTestSt":{"properties":{"a":{"type":"integer"}},"type":"object"}
	  },
	  "properties":{
		"map":{
		  "minProperties":2,
		  "additionalProperties":{"$ref":"#/definitions/JsonschemaGoTestSt"},
		  "type":["object","null"]
		},
		"mapOmitempty":{
		  "minProperties":3,
		  "additionalProperties":{"$ref":"#/definitions/JsonschemaGoTestSt"},
		  "type":"object"
		},
		"namedMap":{
		  "minProperties":5,
		  "anyOf":[{"type":"null"},{"$ref":"#/definitions/JsonschemaGoTestNamedMap"}]
		},
		"namedMapOmitempty":{"$ref":"#/definitions/JsonschemaGoTestNamedMap","minProperties":1},
		"ptr":{"anyOf":[{"type":"null"},{"$ref":"#/definitions/JsonschemaGoTestSt"}]},
		"ptrOmitempty":{"$ref":"#/definitions/JsonschemaGoTestSt"},
		"slice":{
		  "items":{"$ref":"#/definitions/JsonschemaGoTestSt"},"minItems":2,
		  "type":["array","null"]
		},
		"sliceOmitempty":{
		  "items":{"$ref":"#/definitions/JsonschemaGoTestSt"},"minItems":3,
		  "type":"array"
		},
		"val":{"$ref":"#/definitions/JsonschemaGoTestSt"}
	  },
	  "type":"object"
	}`), s)
}

func TestReflector_Reflect_pointer(t *testing.T) {
	type St struct {
		A int `json:"a"`
	}

	type NamedMap map[string]St

	type Cont struct {
		PtrOmitempty      *St           `json:"ptrOmitempty,omitempty"`
		Ptr               *St           `json:"ptr"`
		Val               St            `json:"val"`
		SliceOmitempty    []St          `json:"sliceOmitempty,omitempty" minItems:"3"`
		Slice             []St          `json:"slice" minItems:"2"`
		MapOmitempty      map[string]St `json:"mapOmitempty,omitempty" minProperties:"3"`
		Map               map[string]St `json:"map" minProperties:"2"`
		NamedMapOmitempty NamedMap      `json:"namedMapOmitempty,omitempty" minProperties:"1"`
		NamedMap          NamedMap      `json:"namedMap" minProperties:"5"`
	}

	s, err := (&jsonschema.Reflector{}).Reflect(Cont{})
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "definitions":{
		"JsonschemaGoTestNamedMap":{
		  "additionalProperties":{"$ref":"#/definitions/JsonschemaGoTestSt"},
		  "type":"object"
		},
		"JsonschemaGoTestSt":{"properties":{"a":{"type":"integer"}},"type":"object"}
	  },
	  "properties":{
		"map":{
		  "minProperties":2,
		  "additionalProperties":{"$ref":"#/definitions/JsonschemaGoTestSt"},
		  "type":["object","null"]
		},
		"mapOmitempty":{
		  "minProperties":3,
		  "additionalProperties":{"$ref":"#/definitions/JsonschemaGoTestSt"},
		  "type":"object"
		},
		"namedMap":{"$ref":"#/definitions/JsonschemaGoTestNamedMap","minProperties":5},
		"namedMapOmitempty":{"$ref":"#/definitions/JsonschemaGoTestNamedMap","minProperties":1},
		"ptr":{"$ref":"#/definitions/JsonschemaGoTestSt"},
		"ptrOmitempty":{"$ref":"#/definitions/JsonschemaGoTestSt"},
		"slice":{
		  "items":{"$ref":"#/definitions/JsonschemaGoTestSt"},"minItems":2,
		  "type":["array","null"]
		},
		"sliceOmitempty":{
		  "items":{"$ref":"#/definitions/JsonschemaGoTestSt"},"minItems":3,
		  "type":"array"
		},
		"val":{"$ref":"#/definitions/JsonschemaGoTestSt"}
	  },
	  "type":"object"
	}`), s)
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

// ISOWeek is an ISO week.
type PtrRawSchema string

// JSONSchemaBytes returns JSON Schema definition.
func (*PtrRawSchema) JSONSchemaBytes() ([]byte, error) {
	return []byte(`{"type": "string","examples": ["foo"]}`), nil
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

type PtrSchema string

// JSONSchemaBytes returns JSON Schema definition.
func (*PtrSchema) JSONSchema() (jsonschema.Schema, error) {
	s := jsonschema.Schema{}

	s.AddType(jsonschema.String)
	s.WithExamples("bar")

	return s, nil
}

func TestExposer(t *testing.T) {
	type Some struct {
		Week       ISOWeek       `json:"week"`
		PtrWeek    *ISOWeek      `json:"ptr_week"`
		Raw        PtrRawSchema  `json:"raw"`
		PtrRaw     *PtrRawSchema `json:"ptr_raw"`
		Country    ISOCountry    `json:"country" deprecated:"true"`
		PtrCountry ISOCountry    `json:"ptr_country" deprecated:"true"`
		PtrExp     PtrSchema     `json:"ptr_exp"`
	}

	s, err := (&jsonschema.Reflector{}).Reflect(Some{})
	require.NoError(t, err)

	assertjson.EqMarshal(t, `{
	  "definitions":{
		"JsonschemaGoTestISOCountry":{
		  "description":"ISO Country","examples":["US"],"maxLength":2,"minLength":2,
		  "pattern":"^[a-zA-Z]{2}$","type":"string"
		},
		"JsonschemaGoTestISOWeek":{
		  "description":"ISO Week","examples":["2018-W43"],
		  "pattern":"^[0-9]{4}-W(0[1-9]|[1-4][0-9]|5[0-3])$","type":"string"
		}
	  },
	  "properties":{
		"country":{"$ref":"#/definitions/JsonschemaGoTestISOCountry","deprecated":true},
		"ptr_country":{"$ref":"#/definitions/JsonschemaGoTestISOCountry","deprecated":true},
		"ptr_exp":{"examples":["bar"],"type":"string"},
		"ptr_raw":{"examples":["foo"],"type":["string","null"]},
		"ptr_week":{"$ref":"#/definitions/JsonschemaGoTestISOWeek"},
		"raw":{"examples":["foo"],"type":"string"},
		"week":{"$ref":"#/definitions/JsonschemaGoTestISOWeek"}
	  },
	  "type":"object"
	}`, s)
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
		unexportedStruct jsonschema.Struct
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

func TestPreparer(t *testing.T) {
	r := jsonschema.Reflector{}

	s, err := r.Reflect(nullFloat{})
	require.NoError(t, err)
	assertjson.EqualMarshal(t, []byte(`{"type":["null", "number"]}`), s)
}

func TestReflector_Reflect_inclineScalar(t *testing.T) {
	type Symbol string

	type topTracesInput struct {
		RootSymbol Symbol `json:"rootSymbol" minLength:"5" example:"my_func" default:"main()"`
	}

	r := jsonschema.Reflector{}
	s, err := r.Reflect(topTracesInput{})
	require.NoError(t, err)
	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{
		"rootSymbol":{"default":"main()","examples":["my_func"],"minLength":5,"type":"string"}
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
	require.NoError(t, err)
	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{
		"map":{
		  "additionalProperties":{"type":["null","number"]},
		  "type":["object","null"]
		},
		"slice":{"items":{"type":["null","number"]},"type":["array","null"]}
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
	require.NoError(t, err)

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
	require.NoError(t, err)

	assertjson.EqMarshal(t, `{
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
		  "title":"Person","required":["lastName"],
		  "properties":{
			"birthDate":{"type":"string","format":"date"},
			"createdAt":{"type":"string","format":"date-time"},
			"date":{"type":"string","format":"date"},
			"deathDate":{"type":["null","string"],"format":"date"},
			"deletedAt":{"type":["null","string"],"format":"date-time"},
			"enumed":{"$ref":"#/definitions/Enumed"},
			"enumedPtr":{"$ref":"#/definitions/Enumed"},
			"firstName":{"type":"string"},"height":{"type":"integer"},
			"lastName":{"type":"string"},"meta":{},
			"role":{"description":"The role of person.","type":"string"}
		  },
		  "type":"object"
		}
	  },
	  "properties":{"foo":{"type":"string"}},"type":"object",
	  "if":{"$ref":"#/definitions/Entity"},"then":{"type":"string"},
	  "else":{"$ref":"#/definitions/Person"},
	  "allOf":[{"type":"number"},{"type":"integer"}],
	  "anyOf":[{"type":"string"},{"type":"integer"}],
	  "oneOf":[
		{"$ref":"#/definitions/Person"},{"$ref":"#/definitions/Enumed"},
		{"type":"string"}
	  ],
	  "not":{"$ref":"#/definitions/Person"}
	}`, s)
}

func TestReflector_Reflect_jsonEmptyName(t *testing.T) {
	type Test struct {
		Foo string `json:",omitempty"`
		Bar int    `json:""`
		Baz bool   `json:"-"`
	}

	r := jsonschema.Reflector{}

	s, err := r.Reflect(Test{})
	require.NoError(t, err)

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
	require.NoError(t, err)

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
	require.NoError(t, err)

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
	require.NoError(t, err)

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
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "title":"Test","description":"This is a test.","additionalProperties":false,
	  "properties":{"foo":{"type":"string"}},"type":"object"
	}`), s)

	// Failure scenarios.
	_, err = r.Reflect(struct {
		_ string `additionalProperties:"abc"`
	}{})
	require.EqualError(t, err, "failed to parse bool value abc in tag additionalProperties: strconv.ParseBool: parsing \"abc\": invalid syntax")

	_, err = r.Reflect(struct {
		_ string `minProperties:"abc"`
	}{})
	assert.EqualError(t, err, "failed to parse int value abc in tag minProperties: strconv.ParseInt: parsing \"abc\": invalid syntax")
}

func TestReflector_Reflect_parentTagsExample(t *testing.T) {
	type Test struct {
		Foo string   `json:"foo" query:"foo"`
		_   struct{} `title:"Test" example:"{\"foo\":\"abc\"}"` // Tags of unnamed field are applied to parent schema.

		// There can be more than one field to set up parent schema.
		// Types of such fields are not relevant, only tags matter.
		_ string `query:"_" additionalProperties:"false" description:"This is a test."`
	}

	r := jsonschema.Reflector{}

	s, err := r.Reflect(Test{})
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "title":"Test","description":"This is a test.","examples":[{"foo":"abc"}],
	  "additionalProperties":false,"properties":{"foo":{"type":"string"}},
	  "type":"object"
	}`), s)
}

func TestReflector_Reflect_parentTagsFiltered(t *testing.T) {
	type Test struct {
		Foo string   `json:"foo" query:"foo"`
		_   struct{} `title:"Test" example:"{\"foo\":\"abc\"}"` // Tags of unnamed field are applied to parent schema.

		// There can be more than one field to set up parent schema.
		// Types of such fields are not relevant, only tags matter.
		_ string `query:"_" additionalProperties:"false" description:"This is a test."`
	}

	r := jsonschema.Reflector{}

	s, err := r.Reflect(Test{})
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "title":"Test","description":"This is a test.","examples":[{"foo":"abc"}],
	  "additionalProperties":false,"properties":{"foo":{"type":"string"}},
	  "type":"object"
	}`), s)

	s, err = r.Reflect(Test{}, func(rc *jsonschema.ReflectContext) {
		rc.UnnamedFieldWithTag = true
		rc.PropertyNameTag = "json"
	})
	require.NoError(t, err)

	// No parent schema update for json, as tag is missing in unnamed field.
	assertjson.EqualMarshal(t, []byte(`{"properties":{"foo":{"type":"string"}},"type":"object"}`), s)

	s, err = r.Reflect(Test{}, func(rc *jsonschema.ReflectContext) {
		rc.UnnamedFieldWithTag = true
		rc.PropertyNameTag = "query"
	})
	require.NoError(t, err)

	// Parent schema is updated for query, as tag is present in unnamed field.
	assertjson.EqualMarshal(t, []byte(`{
	  "description":"This is a test.","additionalProperties":false,
	  "properties":{"foo":{"type":"string"}},"type":"object"
	}`), s)

	// Failure scenarios.
	_, err = r.Reflect(struct {
		_ string `additionalProperties:"abc"`
	}{})
	require.EqualError(t, err, "failed to parse bool value abc in tag additionalProperties: strconv.ParseBool: parsing \"abc\": invalid syntax")

	_, err = r.Reflect(struct {
		_ string `minProperties:"abc"`
	}{})
	require.EqualError(t, err, "failed to parse int value abc in tag minProperties: strconv.ParseInt: parsing \"abc\": invalid syntax")
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

func (w withTextMarshaler) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.Type = nil
	schema.AddType(jsonschema.String)

	return nil
}

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
		  "properties":{"foo":{"default":"bar","examples":["baz"],"type":"string"}},
		  "type":"object"
		}
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
		  "properties":{"du":{"type":"integer"},"foo":{"type":"string"}},
		  "type":"object"
		}
	  }
	}`), s)
}

func TestReflector_Reflect_examples(t *testing.T) {
	type WantExample struct {
		A string   `json:"a" example:"example of a"`
		B []string `json:"b" example:"[\"example of b\"]"`
		C int      `json:"c" examples:"[\"foo\", 2, 3]" example:"123"`
	}

	reflector := jsonschema.Reflector{}
	schema, err := reflector.Reflect(WantExample{})
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{
		"a":{"examples":["example of a"],"type":"string"},
		"b":{
		  "examples":[["example of b"]],"items":{"type":"string"},
		  "type":["array","null"]
		},
		"c":{"examples":[123,"foo",2,3],"type":"integer"}
	  },
	  "type":"object"
	}`), schema)
}

func TestReflector_Reflect_namedSlice(t *testing.T) {
	type PanicType []string

	type PanicStruct struct {
		IPPolicy PanicType `json:"ip_policy" example:"[\"127.0.0.1\"]"`
	}

	reflector := jsonschema.Reflector{}
	schema, err := reflector.Reflect(PanicStruct{})
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "definitions":{
		"JsonschemaGoTestPanicType":{"items":{"type":"string"},"type":["array","null"]}
	  },
	  "properties":{
		"ip_policy":{
		  "$ref":"#/definitions/JsonschemaGoTestPanicType",
		  "examples":[["127.0.0.1"]]
		}
	  },
	  "type":"object"
	}`), schema)
}

func TestReflector_Reflect_uuid(t *testing.T) {
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
		ID UUID `json:"uuid"`
	}

	s, err := reflector.Reflect(MyStruct{})
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{
		"uuid":{
		  "examples":["248df4b7-aa70-47b8-a036-33ac447e668d"],"type":"string",
		  "format":"uuid"
		}
	  },
	  "type":"object"
	}`), s)
}

func TestInterceptNullability(t *testing.T) {
	r := jsonschema.Reflector{}

	s, err := r.Reflect(Org{}, jsonschema.InterceptNullability(func(params jsonschema.InterceptNullabilityParams) {
		assert.NotNil(t, params.Context)

		if params.Type.Kind() == reflect.Ptr {
			params.Schema.AddType(jsonschema.Null)
		}
	}))

	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
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
			"enumedPtr":{"$ref":"#/definitions/JsonschemaGoTestEnumed"},
			"firstName":{"type":"string"},"height":{"type":"integer"},
			"lastName":{"type":"string"},"meta":{"type":"null"},
			"role":{"description":"The role of person.","type":"string"}
		  },
		  "type":["object","null"]
		}
	  },
	  "properties":{
		"chiefOfMorale":{"$ref":"#/definitions/JsonschemaGoTestPerson"},
		"employees":{"items":{"$ref":"#/definitions/JsonschemaGoTestPerson"},"type":"array"}
	  },
	  "type":"object"
	}`), s)
}

func TestReflector_Reflect_pointer_sharing(t *testing.T) {
	type UUID [16]byte

	r := jsonschema.Reflector{}

	uuidDef := jsonschema.Schema{}
	uuidDef.AddType(jsonschema.String)
	uuidDef.WithFormat("uuid")

	r.AddTypeMapping(UUID{}, uuidDef)
	r.InlineDefinition(UUID{})

	type StructWithNullable struct {
		NullableID    *UUID `json:"nullable_id"`
		NonNullableID UUID  `json:"non_nullable_id"`
	}

	s, err := r.Reflect(StructWithNullable{})
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{
		"non_nullable_id":{"type":"string","format":"uuid"},
		"nullable_id":{"type":["string","null"],"format":"uuid"}
	  },
	  "type":"object"
	}`), s)
}

type tt struct{}

func (t *tt) UnmarshalText(_ []byte) error {
	return nil
}

func (t tt) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.Type = nil
	schema.AddType(jsonschema.Number)

	return nil
}

func (t tt) MarshalText() (text []byte, err error) {
	return []byte("foo"), nil
}

func TestReflector_Reflect_issue64(t *testing.T) {
	r := jsonschema.Reflector{}
	s, err := r.Reflect(tt{})
	require.NoError(t, err)
	j, err := json.Marshal(s)
	require.NoError(t, err)

	assert.Equal(t, `{"type":"number"}`, string(j))
}

func TestPropertyNameTag(t *testing.T) {
	r := jsonschema.Reflector{}

	type MyStruct struct {
		A string `query:"a"`
		B int    `query:"b" form:"bf"`
		C bool   `form:"cf" json:"cj"`
	}

	s, err := r.Reflect(MyStruct{}, jsonschema.PropertyNameTag("query", "form", "json"))
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{"a":{"type":"string"},"b":{"type":"integer"},"cf":{"type":"boolean"}},
	  "type":"object"
	}`), s)
}

type myString string

func (m myString) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.WithTitle(string(m))

	return nil
}

func TestReflector_Reflect_value_propagation(t *testing.T) {
	r := jsonschema.Reflector{}
	r.InlineDefinition(myString(""))

	type myStruct struct {
		A []myString          `json:"a"`
		B map[string]myString `json:"b"`
		C myString            `json:"c"`
		D struct {
			E myString `json:"e"`
		} `json:"d"`
	}

	v := myStruct{}
	v.A = []myString{"aaa"}
	v.B = map[string]myString{"foo": "bbb"}
	v.C = "ccc"
	v.D.E = "eee"

	s, err := r.Reflect(v)
	require.NoError(t, err)

	// Values from reflected sample may be used in schema.
	assertjson.EqualMarshal(t, []byte(`{
	  "properties":{
		"a":{"items":{"title":"aaa","type":"string"},"type":["array","null"]},
		"b":{
		  "additionalProperties":{"title":"bbb","type":"string"},
		  "type":["object","null"]
		},
		"c":{"title":"ccc","type":"string"},
		"d":{"properties":{"e":{"title":"eee","type":"string"}},"type":"object"}
	  },
	  "type":"object"
	}`), s)
}

func TestReflector_Reflect_skipProperty(t *testing.T) {
	type TimeEntry struct {
		Foo string `json:"foo"`
	}

	type SomeStruct struct {
		ActivityID   int    `json:"activityID" db:"activity_id" required:"true"`
		ProjectID    int    `json:"projectID" db:"project_id" required:"true"`
		Name         string `json:"name" db:"name" required:"true"`
		Description  string `json:"description" db:"description" required:"true"`
		IsProductive bool   `json:"isProductive" db:"is_productive" required:"true"`

		TimeEntries *[]TimeEntry `json:"timeEntries" db:"time_entries" openapi-go:"ignore"`
		// xo fields
		_exists, _deleted bool
	}

	reflector := jsonschema.Reflector{}
	reflector.DefaultOptions = append(reflector.DefaultOptions, jsonschema.InterceptProp(func(params jsonschema.InterceptPropParams) error {
		assert.NotNil(t, params.Context)

		if params.Field.Tag.Get("openapi-go") == "ignore" {
			return jsonschema.ErrSkipProperty
		}

		return nil
	}))

	st := SomeStruct{}

	s, err := reflector.Reflect(st)
	require.NoError(t, err)
	assertjson.EqualMarshal(t, []byte(`{
	  "required":["activityID","projectID","name","description","isProductive"],
	  "properties":{
		"activityID":{"type":"integer"},"description":{"type":"string"},
		"isProductive":{"type":"boolean"},"name":{"type":"string"},
		"projectID":{"type":"integer"}
	  },
	  "type":"object"
	}`), s)
}

func TestReflector_Reflect_example(t *testing.T) {
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
		func(_ reflect.Type, defaultDefName string) string {
			return strings.TrimPrefix(defaultDefName, "JsonschemaGoTest")
		},
	))

	// Create schema from Go value.
	schema, err := reflector.Reflect(new(Resp))
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`{
	  "title":"Sample Response","description":"This is a sample response.",
	  "definitions":{
		"NamedAnything":{},
		"UUID":{
		  "examples":["248df4b7-aa70-47b8-a036-33ac447e668d"],"type":"string",
		  "format":"uuid"
		}
	  },
	  "properties":{
		"arrayOfAnything":{"items":{},"type":"array"},
		"arrayOfNamedAnything":{"items":{"$ref":"#/definitions/NamedAnything"},"type":"array"},
		"field1":{"type":"integer"},"field2":{"type":"string"},
		"info":{
		  "required":["foo"],
		  "properties":{
			"bar":{"description":"This is Bar.","type":"number"},
			"foo":{"default":"baz","pattern":"\\d+","type":"string"}
		  },
		  "type":"object"
		},
		"map":{"additionalProperties":{"type":"integer"},"type":"object"},
		"mapOfAnything":{"additionalProperties":{},"type":"object"},
		"nullableWhatever":{},"parent":{"$ref":"#"},
		"recursiveArray":{"items":{"$ref":"#"},"type":"array"},
		"recursiveStructArray":{"items":{"$ref":"#"},"type":"array"},
		"uuid":{"$ref":"#/definitions/UUID"},"whatever":{}
	  },
	  "type":"object","x-foo":"bar"
	}`), schema)
}

func TestReflector_Reflect_inlineRefs_typeCycle(t *testing.T) {
	type Data struct {
		ID   string `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	}

	type ExampleEvent struct {
		ID          string `json:"id,omitempty"`
		NewData     Data   `json:"new_data"`
		CurrentData Data   `json:"current_data"`
		OldData     Data   `json:"old_data"`
	}

	ref := jsonschema.Reflector{}

	gen, err := ref.Reflect(&ExampleEvent{}, jsonschema.InlineRefs)

	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
	  "properties":{
		"current_data":{
		  "properties":{"id":{"type":"string"},"name":{"type":"string"}},
		  "type":"object"
		},
		"id":{"type":"string"},
		"new_data":{
		  "properties":{"id":{"type":"string"},"name":{"type":"string"}},
		  "type":"object"
		},
		"old_data":{
		  "properties":{"id":{"type":"string"},"name":{"type":"string"}},
		  "type":"object"
		}
	  },
	  "type":"object"
	}`, gen)
}

func TestReflector_Reflect_deeplyEmbedded(t *testing.T) {
	r := jsonschema.Reflector{}

	type Embed struct {
		Foo string `json:"foo" minLength:"5"`
		Bar int    `json:"bar" minimum:"3"`
	}

	type DeeplyEmbedded struct {
		Embed
	}

	type My struct {
		*DeeplyEmbedded

		Baz float64 `json:"baz" title:"Bazzz."`
	}

	val := My{}
	val.DeeplyEmbedded = &DeeplyEmbedded{}
	val.Foo = "abcde"
	val.Bar = 123
	val.Baz = 4.56

	assertjson.EqMarshal(t, `{"foo":"abcde","bar":123,"baz":4.56}`, val)

	s, err := r.Reflect(val)
	require.NoError(t, err)

	assertjson.EqMarshal(t, `{
	  "properties":{
		"bar":{"minimum":3,"type":"integer"},
		"baz":{"title":"Bazzz.","type":"number"},
		"foo":{"minLength":5,"type":"string"}
	  },
	  "type":"object"
	}`, s)
}

func TestReflector_Reflect_deeplyEmbedded_emptyJSONTags(t *testing.T) {
	r := jsonschema.Reflector{}

	type Embed struct {
		Foo string `json:",omitempty" minLength:"5"` // Empty name in tag results in Go field name.
		Bar int    `json:"bar" minimum:"3"`
	}

	type DeeplyEmbedded struct {
		Embed `json:""`
	}

	type My struct {
		*DeeplyEmbedded `json:",inline"` // `inline` does not have any specific handling by encoding/json.

		Baz float64 `json:"baz" title:"Bazzz."`
	}

	val := My{}
	val.DeeplyEmbedded = &DeeplyEmbedded{}
	val.Foo = "abcde"
	val.Bar = 123
	val.Baz = 4.56

	assertjson.EqMarshal(t, `{"Foo":"abcde","bar":123,"baz":4.56}`, val)

	s, err := r.Reflect(val)
	require.NoError(t, err)

	assertjson.EqMarshal(t, `{
	  "properties":{
		"bar":{"minimum":3,"type":"integer"},
		"baz":{"title":"Bazzz.","type":"number"},
		"Foo":{"minLength":5,"type":"string"}
	  },
	  "type":"object"
	}`, s)
}

func TestReflector_Reflect_deeplyEmbedded_validJSONTags(t *testing.T) {
	r := jsonschema.Reflector{}

	type Embed struct {
		Foo string `json:"foo" minLength:"5"`
		Bar int    `json:"bar" minimum:"3"`
	}

	type DeeplyEmbedded struct {
		Embed `json:"emb"`
	}

	type My struct {
		*DeeplyEmbedded `json:"deep,inline"` // `inline` does not have any specific handling by encoding/json.

		Baz float64 `json:"baz" title:"Bazzz."`
	}

	val := My{}
	val.DeeplyEmbedded = &DeeplyEmbedded{}
	val.Foo = "abcde"
	val.Bar = 123
	val.Baz = 4.56

	assertjson.EqMarshal(t, `{"deep":{"emb":{"foo":"abcde","bar":123}},"baz":4.56}`, val)

	s, err := r.Reflect(val)
	require.NoError(t, err)

	assertjson.EqMarshal(t, `{
	  "definitions":{
		"JsonschemaGoTestDeeplyEmbedded":{
		  "properties":{"emb":{"$ref":"#/definitions/JsonschemaGoTestEmbed"}},
		  "type":"object"
		},
		"JsonschemaGoTestEmbed":{
		  "properties":{
			"bar":{"minimum":3,"type":"integer"},
			"foo":{"minLength":5,"type":"string"}
		  },
		  "type":"object"
		}
	  },
	  "properties":{
		"baz":{"title":"Bazzz.","type":"number"},
		"deep":{"$ref":"#/definitions/JsonschemaGoTestDeeplyEmbedded"}
	  },
	  "type":"object"
	}`, s)
}

func TestReflector_Reflect_deeplyEmbeddedUnexported(t *testing.T) {
	r := jsonschema.Reflector{}

	type Embed struct {
		Foo string `json:"foo" minLength:"5"`
		Bar int    `json:"bar" minimum:"3"`
	}

	type deeplyEmbedded struct {
		Embed
	}

	type My struct {
		deeplyEmbedded

		Baz float64 `json:"baz" title:"Bazzz."`
	}

	val := My{}
	val.Foo = "abcde"
	val.Bar = 123
	val.Baz = 4.56

	assertjson.EqMarshal(t, `{"foo":"abcde","bar":123,"baz":4.56}`, val)

	s, err := r.Reflect(val)
	require.NoError(t, err)

	assertjson.EqMarshal(t, `{
	  "properties":{
		"bar":{"minimum":3,"type":"integer"},
		"baz":{"title":"Bazzz.","type":"number"},
		"foo":{"minLength":5,"type":"string"}
	  },
	  "type":"object"
	}`, s)
}

func TestReflector_Reflect_nullable(t *testing.T) {
	r := jsonschema.Reflector{}

	type My struct {
		List1 []string       `json:"l1"`
		List2 []int          `json:"l2"`
		List3 []string       `json:"l3" nullable:"false"`
		S1    string         `json:"s1" nullable:"true"`
		S2    *string        `json:"s2" nullable:"false"`
		Map1  map[string]int `json:"m1"`
		Map2  map[string]int `json:"m2" nullable:"false"`
	}

	s, err := r.Reflect(My{})
	require.NoError(t, err)

	assertjson.EqMarshal(t, `{
	  "properties":{
		"l1":{"items":{"type":"string"},"type":["array","null"]},
		"l2":{"items":{"type":"integer"},"type":["array","null"]},
		"l3":{"items":{"type":"string"},"type":"array"},
		"m1":{"additionalProperties":{"type":"integer"},"type":["object","null"]},
		"m2":{"additionalProperties":{"type":"integer"},"type":"object"},
		"s1":{"type":["string","null"]},
		"s2":{"type":"string"}
	  },
	  "type":"object"
	}`, s)
}

func TestReflector_Reflect_customTags(t *testing.T) {
	r := jsonschema.Reflector{}

	type My struct {
		Foo string `json:"foo" validate:"required" description:"This is foo."`
	}

	type Parent struct {
		MySlice []*My `json:"my,omitempty" validate:"required" description:"The required array."`
	}

	s, err := r.Reflect(Parent{}, jsonschema.InterceptProp(func(params jsonschema.InterceptPropParams) error {
		if !params.Processed {
			return nil
		}

		if v, ok := params.Field.Tag.Lookup("validate"); ok {
			if strings.Contains(v, "required") {
				params.ParentSchema.Required = append(params.ParentSchema.Required, params.Name)
			}
		}

		return nil
	}))
	require.NoError(t, err)

	assertjson.EqMarshal(t, `{
	  "required":["my"],
	  "definitions":{
		"JsonschemaGoTestMy":{
		  "required":["foo"],
		  "properties":{"foo":{"description":"This is foo.","type":"string"}},
		  "type":"object"
		}
	  },
	  "properties":{
		"my":{
		  "description":"The required array.",
		  "items":{"$ref":"#/definitions/JsonschemaGoTestMy"},"type":"array"
		}
	  },
	  "type":"object"
	}`, s)
}

func TestReflector_Reflect_customTime(t *testing.T) {
	type MyTime time.Time

	type MyPtrTime *time.Time

	type MyStruct struct {
		T1 MyTime     `json:"t1"`
		T2 *MyTime    `json:"t2"`
		T3 *MyPtrTime `json:"t3"`
	}

	r := jsonschema.Reflector{}
	s, err := r.Reflect(MyStruct{})

	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
	  "definitions":{"JsonschemaGoTestMyTime":{"type":"object"}},
	  "properties":{
		"t1":{"$ref":"#/definitions/JsonschemaGoTestMyTime"},
		"t2":{"$ref":"#/definitions/JsonschemaGoTestMyTime"},
		"t3":{"type":["null","string"],"format":"date-time"}
	  },
	  "type":"object"
	}`, s)
}

func TestReflector_Reflect_selfReference(t *testing.T) {
	type SubEntity struct {
		Self *SubEntity `json:"self"`
	}

	type Req struct {
		SubEntity *SubEntity `json:"subentity"`
	}

	r := jsonschema.Reflector{}
	s, err := r.Reflect(Req{})

	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
	  "definitions":{
		"JsonschemaGoTestSubEntity":{
		  "properties":{"self":{"$ref":"#/definitions/JsonschemaGoTestSubEntity"}},
		  "type":"object"
		}
	  },
	  "properties":{"subentity":{"$ref":"#/definitions/JsonschemaGoTestSubEntity"}},
	  "type":"object"
	}`, s)
}

func TestReflector_Reflect_multipleTags(t *testing.T) {
	type GetReq struct {
		InQuery1 int     `query:"in_query1" required:"true" description:"Query parameter." json:"q1"`
		InQuery3 int     `query:"in_query3" required:"true" description:"Query parameter." json:"q3"`
		InPath   int     `path:"in_path" json:"p"`
		InCookie string  `cookie:"in_cookie" deprecated:"true" json:"c"`
		InHeader float64 `header:"in_header" json:"h"`
	}

	r := jsonschema.Reflector{}

	s, err := r.Reflect(GetReq{}, jsonschema.PropertyNameTag("query"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
	  "required":["in_query1","in_query3"],
	  "properties":{
		"in_query1":{"description":"Query parameter.","type":"integer"},
		"in_query3":{"description":"Query parameter.","type":"integer"}
	  },
	  "type":"object"
	}`, s)

	s, err = r.Reflect(GetReq{}, jsonschema.PropertyNameTag("path"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"properties":{"in_path":{"type":"integer"}},"type":"object"}`, s)

	s, err = r.Reflect(GetReq{}, jsonschema.PropertyNameTag("json"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
	  "required":["q1","q3"],
	  "properties":{
		"c":{"type":"string","deprecated":true},"h":{"type":"number"},
		"p":{"type":"integer"},
		"q1":{"description":"Query parameter.","type":"integer"},
		"q3":{"description":"Query parameter.","type":"integer"}
	  },
	  "type":"object"
	}`, s)
}

func TestReflector_Reflect_embedded(t *testing.T) {
	type A struct {
		FieldA int `json:"field_a"`
	}

	type C struct {
		jsonschema.EmbedReferencer
		FieldC int `json:"field_c"`
	}

	type B struct {
		A      `refer:"true"`
		FieldB int `json:"field_b"`
		C
	}

	r := jsonschema.Reflector{}

	s, err := r.Reflect(B{})
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
	  "definitions":{
		"JsonschemaGoTestA":{"properties":{"field_a":{"type":"integer"}},"type":"object"},
		"JsonschemaGoTestC":{"properties":{"field_c":{"type":"integer"}},"type":"object"}
	  },
	  "properties":{"field_b":{"type":"integer"}},"type":"object",
	  "allOf":[
		{"$ref":"#/definitions/JsonschemaGoTestA"},
		{"$ref":"#/definitions/JsonschemaGoTestC"}
	  ]
	}`, s)
}

func (*UUID) UnmarshalText(_ []byte) error {
	return nil
}

func (UUID) MarshalText() (text []byte, err error) {
	return []byte("248df4b7-aa70-47b8-a036-33ac447e668d"), nil
}

func TestReflector_Reflect_textMarshaler(t *testing.T) {
	type T struct {
		ID UUID `json:"id"` // UUID has type [16]byte, but implements encoding.TextMarshaler
	}

	reflector := jsonschema.Reflector{}
	schema, err := reflector.Reflect(T{})
	require.NoError(t, err)

	assertjson.EqMarshal(t, `{
	  "properties":{"id":{"type":"string"}},
	  "type":"object"
	}`, schema)
}

type rawExposer func() ([]byte, error)

func (r rawExposer) JSONSchemaBytes() ([]byte, error) {
	return r()
}

func TestReflector_AddOperation_rawSchema(t *testing.T) {
	r := jsonschema.Reflector{}

	type My struct {
		A interface{} `json:"a"`
		B interface{} `json:"b"`
	}

	m := My{
		A: rawExposer(func() ([]byte, error) {
			return []byte(`{"type":"object","properties":{"foo":{"type":"integer"}}}`), nil
		}),
		B: rawExposer(func() ([]byte, error) {
			return []byte(`{"type":"object","properties":{"bar":{"type":"integer"}}}`), nil
		}),
	}

	s, err := r.Reflect(m)
	require.NoError(t, err)

	assertjson.EqMarshal(t, `{
	  "properties":{
		"a":{"properties":{"foo":{"type":"integer"}},"type":"object"},
		"b":{"properties":{"bar":{"type":"integer"}},"type":"object"}
	  },
	  "type":"object"
	}`, s)
}

type Discover string

const (
	DiscoverAll  Discover = "all"
	DiscoverNone Discover = "none"
)

func (d *Discover) Enum() []interface{} {
	return []interface{}{DiscoverAll, DiscoverNone}
}

func TestReflector_Reflect_ptrDefault(t *testing.T) {
	type NewThing struct {
		DiscoverMode *Discover `json:"discover,omitempty" default:"all"`
	}

	r := jsonschema.Reflector{}

	s, err := r.Reflect(NewThing{})
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
	  "definitions":{
		"JsonschemaGoTestDiscover":{"enum":["all","none"],"type":["null","string"]}
	  },
	  "properties":{
		"discover":{"$ref":"#/definitions/JsonschemaGoTestDiscover","default":"all"}
	  },
	  "type":"object"
	}`, s)
}

func TestReflector_Reflect_nilPreparer(t *testing.T) {
	var o *Org

	r := jsonschema.Reflector{}

	s, err := r.Reflect(o)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
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
			"enumedPtr":{"$ref":"#/definitions/JsonschemaGoTestEnumed"},
			"firstName":{"type":"string"},"height":{"type":"integer"},
			"lastName":{"type":"string"},"meta":{},
			"role":{"description":"The role of person.","type":"string"}
		  },
		  "type":"object"
		}
	  },
	  "properties":{
		"chiefOfMorale":{"$ref":"#/definitions/JsonschemaGoTestPerson"},
		"employees":{"items":{"$ref":"#/definitions/JsonschemaGoTestPerson"},"type":"array"}
	  },
	  "type":"object"
	}`, s)
}

type withPtrPreparer string

func (w *withPtrPreparer) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.WithTitle(string(*w))

	return nil
}

type withValPreparer string

func (w withValPreparer) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.WithTitle(string(w))

	return nil
}

func TestReflector_Reflect_Preparer(t *testing.T) {
	r := jsonschema.Reflector{}

	var (
		np *withPtrPreparer
		nv *withValPreparer
	)

	s, err := r.Reflect(np)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"title":"","type":["null","string"]}`, s)

	s, err = r.Reflect(nv)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"title":"","type":["null","string"]}`, s)

	s, err = r.Reflect(withValPreparer("test1"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"title":"test1","type":"string"}`, s)

	s, err = r.Reflect(withPtrPreparer("test2"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"title":"test2","type":"string"}`, s)
}

type withPtrExposer string

func (w *withPtrExposer) JSONSchema() (jsonschema.Schema, error) {
	s := jsonschema.Schema{}
	s.WithTitle(string(*w))

	return s, nil
}

type withValExposer string

func (w withValExposer) JSONSchema() (jsonschema.Schema, error) {
	s := jsonschema.Schema{}
	s.WithTitle(string(w))

	return s, nil
}

func TestReflector_Reflect_Exposer(t *testing.T) {
	r := jsonschema.Reflector{}

	var (
		np *withPtrExposer
		nv *withValExposer
	)

	s, err := r.Reflect(np)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"title":""}`, s)

	s, err = r.Reflect(nv)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"title":""}`, s)

	s, err = r.Reflect(withValExposer("test1"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"title":"test1"}`, s)

	s, err = r.Reflect(withPtrExposer("test2"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"title":"test2"}`, s)
}

type withPtrRawExposer string

func (w *withPtrRawExposer) JSONSchemaBytes() ([]byte, error) {
	return []byte(`{"title":"` + string(*w) + `"}`), nil
}

type withValRawExposer string

func (w withValRawExposer) JSONSchemaBytes() ([]byte, error) {
	return []byte(`{"title":"` + string(w) + `"}`), nil
}

func TestReflector_Reflect_RawExposer(t *testing.T) {
	r := jsonschema.Reflector{}

	var (
		np *withPtrRawExposer
		nv *withValRawExposer
	)

	s, err := r.Reflect(np)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"title":""}`, s)

	s, err = r.Reflect(nv)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"title":""}`, s)

	s, err = r.Reflect(withValRawExposer("test1"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"title":"test1"}`, s)

	s, err = r.Reflect(withPtrRawExposer("test2"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"title":"test2"}`, s)
}

type withPtrEnum string

func (w *withPtrEnum) Enum() []interface{} {
	return []interface{}{string(*w)}
}

type withValEnum string

func (w withValEnum) Enum() []interface{} {
	return []interface{}{string(w)}
}

func TestReflector_Reflect_Enum(t *testing.T) {
	r := jsonschema.Reflector{}

	var (
		np *withPtrEnum
		nv *withValEnum
	)

	s, err := r.Reflect(np)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"enum":[""],"type":["null","string"]}`, s)

	s, err = r.Reflect(nv)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"enum":[""],"type":["null","string"]}`, s)

	s, err = r.Reflect(withValEnum("test1"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"enum":["test1"],"type":"string"}`, s)

	s, err = r.Reflect(withPtrEnum("test2"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"enum":["test2"],"type":"string"}`, s)
}

type withPtrNamedEnum string

func (w *withPtrNamedEnum) NamedEnum() ([]interface{}, []string) {
	return []interface{}{string(*w)}, []string{"n:" + string(*w)}
}

type withValNamedEnum string

func (w withValNamedEnum) NamedEnum() ([]interface{}, []string) {
	return []interface{}{string(w)}, []string{"n:" + string(w)}
}

func TestReflector_Reflect_NamedEnum(t *testing.T) {
	r := jsonschema.Reflector{}

	var (
		np *withPtrNamedEnum
		nv *withValNamedEnum
	)

	s, err := r.Reflect(np)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"enum":[""],"type":["null","string"],"x-enum-names":["n:"]}`, s)

	s, err = r.Reflect(nv)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"enum":[""],"type":["null","string"],"x-enum-names":["n:"]}`, s)

	s, err = r.Reflect(withValNamedEnum("test1"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"enum":["test1"],"type":"string","x-enum-names":["n:test1"]}`, s)

	s, err = r.Reflect(withPtrNamedEnum("test2"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"enum":["test2"],"type":"string","x-enum-names":["n:test2"]}`, s)
}

type withPtrOneOfExposer string

func (w *withPtrOneOfExposer) JSONSchemaOneOf() []interface{} {
	return []interface{}{withValPreparer(*w), withPtrPreparer("2:" + *w)}
}

type withValOneOfExposer string

func (w withValOneOfExposer) JSONSchemaOneOf() []interface{} {
	return []interface{}{withValPreparer(w), withPtrPreparer("2:" + w)}
}

func TestReflector_Reflect_OneOfExposer(t *testing.T) {
	r := jsonschema.Reflector{}

	var (
		np *withPtrOneOfExposer
		nv *withValOneOfExposer
	)

	s, err := r.Reflect(np)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
        	            	  "type":["null","string"],
        	            	  "oneOf":[{"title":"","type":"string"},{"title":"2:","type":"string"}]
        	            	}`, s)

	s, err = r.Reflect(nv)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
        	            	  "type":["null","string"],
        	            	  "oneOf":[{"title":"","type":"string"},{"title":"2:","type":"string"}]
        	            	}`, s)

	s, err = r.Reflect(withValOneOfExposer("test1"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
        	            	  "type":"string",
        	            	  "oneOf":[{"title":"test1","type":"string"},{"title":"2:test1","type":"string"}]
        	            	}`, s)

	s, err = r.Reflect(withPtrOneOfExposer("test2"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
        	            	  "type":"string",
        	            	  "oneOf":[{"title":"test2","type":"string"},{"title":"2:test2","type":"string"}]
        	            	}`, s)
}

type withPtrAnyOfExposer string

func (w *withPtrAnyOfExposer) JSONSchemaAnyOf() []interface{} {
	return []interface{}{withValPreparer(*w), withPtrPreparer("2:" + *w)}
}

type withValAnyOfExposer string

func (w withValAnyOfExposer) JSONSchemaAnyOf() []interface{} {
	return []interface{}{withValPreparer(w), withPtrPreparer("2:" + w)}
}

func TestReflector_Reflect_AnyOfExposer(t *testing.T) {
	r := jsonschema.Reflector{}

	var (
		np *withPtrAnyOfExposer
		nv *withValAnyOfExposer
	)

	s, err := r.Reflect(np)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
        	            	  "type":["null","string"],
        	            	  "anyOf":[{"title":"","type":"string"},{"title":"2:","type":"string"}]
        	            	}`, s)

	s, err = r.Reflect(nv)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
        	            	  "type":["null","string"],
        	            	  "anyOf":[{"title":"","type":"string"},{"title":"2:","type":"string"}]
        	            	}`, s)

	s, err = r.Reflect(withValAnyOfExposer("test1"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
        	            	  "type":"string",
        	            	  "anyOf":[{"title":"test1","type":"string"},{"title":"2:test1","type":"string"}]
        	            	}`, s)

	s, err = r.Reflect(withPtrAnyOfExposer("test2"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
        	            	  "type":"string",
        	            	  "anyOf":[{"title":"test2","type":"string"},{"title":"2:test2","type":"string"}]
        	            	}`, s)
}

type withPtrAllOfExposer string

func (w *withPtrAllOfExposer) JSONSchemaAllOf() []interface{} {
	return []interface{}{withValPreparer(*w), withPtrPreparer("2:" + *w)}
}

type withValAllOfExposer string

func (w withValAllOfExposer) JSONSchemaAllOf() []interface{} {
	return []interface{}{withValPreparer(w), withPtrPreparer("2:" + w)}
}

func TestReflector_Reflect_AllOfExposer(t *testing.T) {
	r := jsonschema.Reflector{}

	var (
		np *withPtrAllOfExposer
		nv *withValAllOfExposer
	)

	s, err := r.Reflect(np)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
        	            	  "type":["null","string"],
        	            	  "allOf":[{"title":"","type":"string"},{"title":"2:","type":"string"}]
        	            	}`, s)

	s, err = r.Reflect(nv)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
        	            	  "type":["null","string"],
        	            	  "allOf":[{"title":"","type":"string"},{"title":"2:","type":"string"}]
        	            	}`, s)

	s, err = r.Reflect(withValAllOfExposer("test1"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
        	            	  "type":"string",
        	            	  "allOf":[{"title":"test1","type":"string"},{"title":"2:test1","type":"string"}]
        	            	}`, s)

	s, err = r.Reflect(withPtrAllOfExposer("test2"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
        	            	  "type":"string",
        	            	  "allOf":[{"title":"test2","type":"string"},{"title":"2:test2","type":"string"}]
        	            	}`, s)
}

type withPtrNotExposer string

func (w *withPtrNotExposer) JSONSchemaNot() interface{} {
	return withValPreparer(*w)
}

type withValNotExposer string

func (w withValNotExposer) JSONSchemaNot() interface{} {
	return withValPreparer(w)
}

func TestReflector_Reflect_NotExposer(t *testing.T) {
	r := jsonschema.Reflector{}

	var (
		np *withPtrNotExposer
		nv *withValNotExposer
	)

	s, err := r.Reflect(np)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":["null","string"],"not":{"title":"","type":"string"}}`, s)

	s, err = r.Reflect(nv)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":["null","string"],"not":{"title":"","type":"string"}}`, s)

	s, err = r.Reflect(withValNotExposer("test1"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":"string","not":{"title":"test1","type":"string"}}`, s)

	s, err = r.Reflect(withPtrNotExposer("test2"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":"string","not":{"title":"test2","type":"string"}}`, s)
}

type withPtrIfExposer string

func (w *withPtrIfExposer) JSONSchemaIf() interface{} {
	return withValPreparer(*w)
}

type withValIfExposer string

func (w withValIfExposer) JSONSchemaIf() interface{} {
	return withValPreparer(w)
}

func TestReflector_Reflect_IfExposer(t *testing.T) {
	r := jsonschema.Reflector{}

	var (
		np *withPtrIfExposer
		nv *withValIfExposer
	)

	s, err := r.Reflect(np)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":["null","string"],"if":{"title":"","type":"string"}}`, s)

	s, err = r.Reflect(nv)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":["null","string"],"if":{"title":"","type":"string"}}`, s)

	s, err = r.Reflect(withValIfExposer("test1"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":"string","if":{"title":"test1","type":"string"}}`, s)

	s, err = r.Reflect(withPtrIfExposer("test2"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":"string","if":{"title":"test2","type":"string"}}`, s)
}

type withPtrThenExposer string

func (w *withPtrThenExposer) JSONSchemaThen() interface{} {
	return withValPreparer(*w)
}

type withValThenExposer string

func (w withValThenExposer) JSONSchemaThen() interface{} {
	return withValPreparer(w)
}

func TestReflector_Reflect_ThenExposer(t *testing.T) {
	r := jsonschema.Reflector{}

	var (
		np *withPtrThenExposer
		nv *withValThenExposer
	)

	s, err := r.Reflect(np)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":["null","string"],"then":{"title":"","type":"string"}}`, s)

	s, err = r.Reflect(nv)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":["null","string"],"then":{"title":"","type":"string"}}`, s)

	s, err = r.Reflect(withValThenExposer("test1"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":"string","then":{"title":"test1","type":"string"}}`, s)

	s, err = r.Reflect(withPtrThenExposer("test2"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":"string","then":{"title":"test2","type":"string"}}`, s)
}

type withPtrElseExposer string

func (w *withPtrElseExposer) JSONSchemaElse() interface{} {
	return withValPreparer(*w)
}

type withValElseExposer string

func (w withValElseExposer) JSONSchemaElse() interface{} {
	return withValPreparer(w)
}

func TestReflector_Reflect_ElseExposer(t *testing.T) {
	r := jsonschema.Reflector{}

	var (
		np *withPtrElseExposer
		nv *withValElseExposer
	)

	s, err := r.Reflect(np)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":["null","string"],"else":{"title":"","type":"string"}}`, s)

	s, err = r.Reflect(nv)
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":["null","string"],"else":{"title":"","type":"string"}}`, s)

	s, err = r.Reflect(withValElseExposer("test1"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":"string","else":{"title":"test1","type":"string"}}`, s)

	s, err = r.Reflect(withPtrElseExposer("test2"))
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{"type":"string","else":{"title":"test2","type":"string"}}`, s)
}

func TestReflector_Reflect_byteSlice(t *testing.T) {
	r := jsonschema.Reflector{}

	type S struct {
		A []byte           `json:"a" description:"Hello world!"`
		B json.RawMessage  `json:"b" description:"I am a RawMessage."`
		C *json.RawMessage `json:"c" description:"I am a RawMessage pointer."`
	}

	s1 := S{
		A: []byte("hello world!"),
		B: []byte(`{"foo":"bar"}`),
	}
	s1.C = &s1.B

	v, err := json.Marshal(s1)
	require.NoError(t, err)
	// []byte is marshaled to base64, RawMessage value and pointer are passed as is.
	assert.Equal(t, `{"a":"aGVsbG8gd29ybGQh","b":{"foo":"bar"},"c":{"foo":"bar"}}`, string(v))

	var s2 S

	require.NoError(t, json.Unmarshal(v, &s2))
	assert.Equal(t, "hello world!", string(s2.A)) // []byte is unmarshaled from base64.
	assert.Equal(t, `{"foo":"bar"}`, string(s2.B))
	assert.Equal(t, `{"foo":"bar"}`, string(*s2.C))

	s, err := r.Reflect(S{})
	require.NoError(t, err)
	assertjson.EqMarshal(t, `{
	  "properties":{
		"a":{"description":"Hello world!","type":"string","format":"base64"},
		"b":{"description":"I am a RawMessage."},
		"c":{"description":"I am a RawMessage pointer."}
	  },
	  "type":"object"
	}`, s)
}
