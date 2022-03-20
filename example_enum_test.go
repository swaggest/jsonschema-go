package jsonschema_test

import (
	"fmt"

	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
)

type WeekDay string

func (WeekDay) Enum() []interface{} {
	return []interface{}{
		"Monday",
		"Tuesday",
		"Wednesday",
		"Thursday",
		"Friday",
		"Saturday",
		"Sunday",
	}
}

type Shop struct {
	Days  []WeekDay `json:"days,omitempty"`  // This property uses dedicated named type to express enum.
	Days2 []string  `json:"days2,omitempty"` // This property uses schema preparer to set up enum.

	// This scalar property uses field tag to set up enum.
	Day string `json:"day" enum:"Monday,Tuesday,Wednesday,Thursday,Friday,Saturday,Sunday"`
}

var _ jsonschema.Preparer = Shop{}

func (Shop) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.Properties["days2"].TypeObject.WithEnum(
		"Monday",
		"Tuesday",
		"Wednesday",
		"Thursday",
		"Friday",
		"Saturday",
		"Sunday",
	)

	return nil
}

func ExampleEnum() {
	reflector := jsonschema.Reflector{}

	s, err := reflector.Reflect(Shop{}, jsonschema.StripDefinitionNamePrefix("JsonschemaGoTest"))
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
	//   "definitions":{
	//     "WeekDay":{
	//       "enum":["Monday","Tuesday","Wednesday","Thursday","Friday","Saturday","Sunday"],
	//       "type":"string"
	//     }
	//   },
	//   "properties":{
	//     "day":{
	//       "enum":["Monday","Tuesday","Wednesday","Thursday","Friday","Saturday","Sunday"],
	//       "type":"string"
	//     },
	//     "days":{"items":{"$ref":"#/definitions/WeekDay"},"type":"array"},
	//     "days2":{
	//       "items":{"type":"string"},
	//       "enum":["Monday","Tuesday","Wednesday","Thursday","Friday","Saturday","Sunday"],
	//       "type":"array"
	//     }
	//   },
	//   "type":"object"
	// }
}
