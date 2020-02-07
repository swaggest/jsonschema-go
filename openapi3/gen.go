package openapi3

import (
	jsonschema "github.com/swaggest/jsonschema-go/draft-07"
	"net/http"
	"strconv"
)

type Generator struct {
	jsonschema.Generator
	Spec *Spec
}

func (g *Generator) SetRequest(o *Operation, input interface{}) error {
	schema, err := g.Parse(input)
	if err != nil {
		return err
	}

	o.Parameters = append(o.Parameters, ParameterOrRef{
		ParameterReference: &ParameterReference{Ref: schema.Ref},
	})

	for name, def := range schema.Definitions {
		if g.Spec.Components == nil {
			g.Spec.Components = &Components{}
		}
		if g.Spec.Components.Parameters == nil {
			g.Spec.Components.Parameters = &ComponentsParameters{}
		}
		s := SchemaOrRef{}
		s.FromSchema(def)

		g.Spec.Components.Parameters.WithMapOfParameterOrRefValuesItem(name, ParameterOrRef{
			Parameter: (&Parameter{}).WithSchema(s),
		})
	}

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

	if o.Responses.MapOfResponseOrRefValues == nil {
		o.Responses.MapOfResponseOrRefValues = make(map[string]ResponseOrRef, 1)
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

	for name, def := range schema.Definitions {
		if g.Spec.Components == nil {
			g.Spec.Components = &Components{}
		}
		if g.Spec.Components.Responses == nil {
			g.Spec.Components.Responses = &ComponentsResponses{}
		}
		s := SchemaOrRef{}
		s.FromSchema(def)

		g.Spec.Components.Responses.WithMapOfResponseOrRefValuesItem(name, ResponseOrRef{
			Response: (&Response{}).WithContent(map[string]MediaType{"application/json": {Schema: &s}}),
		})
	}

	return nil
}
