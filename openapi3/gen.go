package openapi3

import (
	jsonschema "github.com/swaggest/jsonschema-go/draft-07"
	"net/http"
	"strconv"
)

type Generator struct {
	jsonschema.Generator
}

func (g *Generator) SetRequest(o *Operation, input interface{}) error {
	schema, err := g.Parse(input)
	if err != nil {
		return err
	}

	o.Parameters = append(o.Parameters, ParameterOrRef{
		ParameterReference: &ParameterReference{Ref: schema.Ref},
	})
	return nil
}

func (g *Generator) SetResponse(o *Operation, output interface{}) error {
	schema, err := g.Parse(output)
	if err != nil {
		return err
	}

	if o.Responses == nil {
		o.Responses = &Responses{}
	}

	o.Responses.MapOfResponseOrRefValues[strconv.Itoa(http.StatusOK)] = ResponseOrRef{
		Response: &Response{
			Description: nil,
			Headers:     nil,
			Content: map[string]MediaType{
				"application/json": {
					Schema: &SchemaOrRef{
						SchemaReference: &SchemaReference{Ref: schema.Ref},
					},
					Encoding: map[string]Encoding{},
				},
			},
			Links:         nil,
			MapOfAnything: nil,
		},
	}

	return nil
}
