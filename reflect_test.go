package jsonschema_test

import (
	"encoding"
	"encoding/json"
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
	Meta      *json.RawMessage `json:"meta"`
}

type Person struct {
	Entity
	BirthDate string `json:"date" format:"date"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName" required:"true"`
	Height    int    `json:"height"`
	Role      Role   `json:"role" description:"The role of person."`
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

func TestReflector_Reflect(t *testing.T) {
	reflector := jsonschema.Reflector{}
	schema, err := reflector.Reflect(Org{})
	require.NoError(t, err)

	j, err := json.MarshalIndent(schema, "", " ")
	require.NoError(t, err)

	assertjson.Equal(t, []byte(`
{
 "title": "Organization",
 "definitions": {
  "JsonschemaGoTestPerson": {
   "title": "Person",
   "required": [
	"lastName"
   ],
   "properties": {
	"createdAt": {
	 "type": "string",
	 "format": "date-time"
	},
	"date": {
	 "type": "string",
	 "format": "date"
	},
	"deletedAt": {
	 "type": [
	  "null",
	  "string"
	 ],
	 "format": "date-time"
	},
	"firstName": {
	 "type": "string"
	},
	"height": {
	 "type": "integer"
	},
	"lastName": {
	 "type": "string"
	},
	"meta": {},
	"role": {
	 "$ref": "#/definitions/JsonschemaGoTestRole",
	 "description": "The role of person."
	}
   },
   "type": "object"
  },
  "JsonschemaGoTestRole": {
   "type": "string"
  }
 },
 "properties": {
  "chiefOfMorale": {
   "$ref": "#/definitions/JsonschemaGoTestPerson"
  },
  "employees": {
   "items": {
	"$ref": "#/definitions/JsonschemaGoTestPerson"
   },
   "type": "array"
  }
 },
 "type": "object"
}
`), j, string(j))
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
   "$ref": "#/definitions/JsonschemaGoTestPerson"
  },
  "employees": {
   "items": {
	"$ref": "#/definitions/JsonschemaGoTestPerson"
   },
   "type": "array"
  }
 },
 "type": "object"
}
`), j, string(j))

	j, err = json.MarshalIndent(schemas, "", " ")
	require.NoError(t, err)

	assertjson.Equal(t, []byte(`
{  
  "JsonschemaGoTestPerson": {
   "title": "Person",
   "required": [
	"lastName"
   ],
   "properties": {
	"createdAt": {
	 "type": "string",
	 "format": "date-time"
	},
	"date": {
	 "type": "string",
	 "format": "date"
	},
	"deletedAt": {
	 "type": [
	  "null",
	  "string"
	 ],
	 "format": "date-time"
	},
	"firstName": {
	 "type": "string"
	},
	"height": {
	 "type": "integer"
	},
	"lastName": {
	 "type": "string"
	},
	"meta": {},
	"role": {
	 "$ref": "#/definitions/JsonschemaGoTestRole",
	 "description": "The role of person."
	}
   },
   "type": "object"
  },
  "JsonschemaGoTestRole": {
   "type": "string"
  }
}`), j)
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

	assertjson.Equal(t, []byte(`{"properties":{"parent":{"$ref":"#"},"siblings":{"items":{"$ref":"#"},"type":"array"},
		"val":{"type":"string"}},"type":"object"}`), j, string(j))
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

	s, err := rf.Reflect(testWrapParams{}, jsonschema.RootRef)
	require.NoError(t, err)

	j, err := json.MarshalIndent(s, "", " ")
	require.NoError(t, err)

	assertjson.Equal(t, []byte(`{
        	            	 "$ref": "#/definitions/JsonschemaGoTestTestWrapParams",
        	            	 "definitions": {
        	            	  "JsonschemaGoTestDeepReplacementTag": {
        	            	   "properties": {
        	            	    "test_field_1": {
        	            	     "type": "string",
        	            	     "format": "double"
        	            	    }
        	            	   },
        	            	   "type": "object"
        	            	  },
        	            	  "JsonschemaGoTestTestWrapParams": {
        	            	   "properties": {
        	            	    "deep_replacement": {
        	            	     "$ref": "#/definitions/JsonschemaGoTestDeepReplacementTag"
        	            	    },
        	            	    "simple_test_replacement": {
        	            	     "type": "string"
        	            	    }
        	            	   },
        	            	   "type": "object"
        	            	  }
        	            	 }
        	            	}`), j, string(j))
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
        	            	   "type": [
        	            	    "object",
        	            	    "null"
        	            	   ]
        	            	  },
        	            	  "mapOmitempty": {
        	            	   "minProperties": 3,
        	            	   "additionalProperties": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "type": "object"
        	            	  },
        	            	  "namedMap": {
        	            	   "minProperties": 5,
        	            	   "anyOf": [
        	            	    {
        	            	     "type": "null"
        	            	    },
        	            	    {
        	            	     "$ref": "#/definitions/JsonschemaGoTestNamedMap"
        	            	    }
        	            	   ]
        	            	  },
        	            	  "namedMapOmitempty": {
        	            	   "$ref": "#/definitions/JsonschemaGoTestNamedMap",
        	            	   "minProperties": 1
        	            	  },
        	            	  "ptr": {
        	            	   "anyOf": [
        	            	    {
        	            	     "type": "null"
        	            	    },
        	            	    {
        	            	     "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	    }
        	            	   ]
        	            	  },
        	            	  "ptrOmitempty": {
        	            	   "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	  },
        	            	  "slice": {
        	            	   "items": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "minItems": 2,
        	            	   "type": [
        	            	    "array",
        	            	    "null"
        	            	   ]
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

	j, err := json.MarshalIndent(s, "", " ")
	require.NoError(t, err)

	assertjson.Equal(t, []byte(`{
        	            	 "definitions": {
        	            	  "JsonschemaGoTestNamedMap": {
        	            	   "additionalProperties": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "type": [
        	            	    "object",
        	            	    "null"
        	            	   ]
        	            	  },
        	            	  "JsonschemaGoTestSt": {
        	            	   "properties": {
        	            	    "a": {
        	            	     "type": "integer"
        	            	    }
        	            	   },
        	            	   "type": [
        	            	    "object",
        	            	    "null"
        	            	   ]
        	            	  }
        	            	 },
        	            	 "properties": {
        	            	  "map": {
        	            	   "minProperties": 2,
        	            	   "additionalProperties": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "type": [
        	            	    "object",
        	            	    "null"
        	            	   ]
        	            	  },
        	            	  "mapOmitempty": {
        	            	   "minProperties": 3,
        	            	   "additionalProperties": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "type": "object"
        	            	  },
        	            	  "namedMap": {
        	            	   "$ref": "#/definitions/JsonschemaGoTestNamedMap",
        	            	   "minProperties": 5
        	            	  },
        	            	  "namedMapOmitempty": {
        	            	   "$ref": "#/definitions/JsonschemaGoTestNamedMap",
        	            	   "minProperties": 1
        	            	  },
        	            	  "ptr": {
        	            	   "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	  },
        	            	  "ptrOmitempty": {
        	            	   "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	  },
        	            	  "slice": {
        	            	   "items": {
        	            	    "$ref": "#/definitions/JsonschemaGoTestSt"
        	            	   },
        	            	   "minItems": 2,
        	            	   "type": [
        	            	    "array",
        	            	    "null"
        	            	   ]
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
		Country ISOCountry `json:"country"`
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
        	            	   "$ref": "#/definitions/JsonschemaGoTestISOCountry"
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
		ID   int    `minimum:"123" default:"200"`
		Name string `minLength:"10"`
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
