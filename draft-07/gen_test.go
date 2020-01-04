package jsonschema_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	jsonschema "github.com/swaggest/jsonschema-go/draft-07"
)

type Person struct {
	Datetime  string    `json:"datetime" format:"date-time"`
	CreatedAt time.Time `json:"createdAt"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName" required:"true"`
	Age       int       `json:"age"`
}

type Org struct {
	Employees []Person `json:"employees"`
}

func (o Org) CustomizeJSONSchema(schema *jsonschema.CoreSchemaMetaSchema) error {
	title := "Organization"
	schema.Title = &title
	return nil
}

func TestGenerator_Parse(t *testing.T) {
	g := jsonschema.Generator{}
	schema, err := g.Parse(new(Org))
	require.NoError(t, err)

	j, err := json.MarshalIndent(schema, "", " ")
	require.NoError(t, err)
	assertjson.Equal(t, []byte(`
{
 "properties": {
  "employees": {
   "items": {
	"required": [
	 "lastName"
	],
	"properties": {
	 "age": {
	  "type": "integer"
	 },
	 "firstName": {
	  "type": "string",
	  "format": "date-time"
	 },
	 "lastName": {
	  "type": "string"
	 }
	},
	"type": "object"
   },
   "type": "array"
  }
 },
 "type": "object",
 "title": "Organization"
}
`), j, string(j))
}
