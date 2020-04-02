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
	schema, err := reflector.Reflect(structWithInline{})
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
