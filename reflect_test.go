package jsonschema_test

import (
	"encoding"
	"encoding/json"
	"testing"
	"time"

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
	ChiefOfMoral *Person  `json:"chiefOfMorale"`
	Employees    []Person `json:"employees"`
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
   "type": [
	"null",
	"object"
   ]
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
 "type": ["null", "object"]
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
   "type": [
	"null",
	"object"
   ]
  },
  "JsonschemaGoTestRole": {
   "type": "string"
  }
}`), j)
}

func TestReflector_Reflect_recursiveStruct(t *testing.T) {
	type Rec struct {
		Val      string `json:"val"`
		Parent   *Rec   `json:"parent"`
		Siblings []Rec  `json:"siblings"`
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
		Items map[string]simpleDateTime `json:"items"`
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
