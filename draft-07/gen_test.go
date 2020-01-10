package jsonschema_test

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	jsonschema "github.com/swaggest/jsonschema-go/draft-07"
	"testing"
	"time"
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
	CreatedAt time.Time       `json:"createdAt"`
	DeletedAt *time.Time      `json:"deletedAt"`
	Meta      json.RawMessage `json:"meta"`
}

type Person struct {
	Entity
	BirthDate string `json:"date" format:"date"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName" required:"true"`
	Height    int    `json:"height"`
	Role      Role   `json:"role" description:"The role of person."`
}

func (p *Person) CustomizeJSONSchema(schema *jsonschema.CoreSchemaMetaSchema) error {
	schema.WithTitle("Person")
	return nil
}

type Org struct {
	ChiefOfMoral *Person  `json:"chiefOfMorale"`
	Employees    []Person `json:"employees"`
}

func (o Org) CustomizeJSONSchema(schema *jsonschema.CoreSchemaMetaSchema) error {
	schema.WithTitle("Organization")
	return nil
}

func TestGenerator_Parse(t *testing.T) {
	g := jsonschema.Generator{}
	schema, err := g.Parse(Org{})
	require.NoError(t, err)

	j, err := json.MarshalIndent(schema, "", " ")
	require.NoError(t, err)
	assertjson.Equal(t, []byte(`
{
 "$ref": "#/definitions/github.com/swaggest/jsonschema-go/draft-07_test.Org::jsonschema_test.Org",
 "definitions": {
  "github.com/swaggest/jsonschema-go/draft-07_test.Org::jsonschema_test.Org": {
   "title": "Organization",
   "properties": {
	"chiefOfMorale": {
	 "$ref": "#/definitions/github.com/swaggest/jsonschema-go/draft-07_test.Person::jsonschema_test.Person"
	},
	"employees": {
	 "items": {
	  "$ref": "#/definitions/github.com/swaggest/jsonschema-go/draft-07_test.Person::jsonschema_test.Person"
	 },
	 "type": "array"
	}
   },
   "type": "object"
  },
  "github.com/swaggest/jsonschema-go/draft-07_test.Person::jsonschema_test.Person": {
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
	 "$ref": "#/definitions/github.com/swaggest/jsonschema-go/draft-07_test.Role::jsonschema_test.Role",
	 "description": "The role of person."
	}
   },
   "type": [
	"null",
	"object"
   ]
  },
  "github.com/swaggest/jsonschema-go/draft-07_test.Role::jsonschema_test.Role": {
   "type": "string"
  }
 }
}
`), j, string(j))
}
