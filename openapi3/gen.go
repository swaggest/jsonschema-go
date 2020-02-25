package openapi3

import (
	jsonschema "github.com/swaggest/jsonschema-go/draft-07"
	"github.com/swaggest/jsonschema-go/refl"
	"net/http"
	"strconv"
)

type Generator struct {
	jsonschema.Generator
	Spec *Spec
}

func (g *Generator) SetRequest(o *Operation, input interface{}) error {
	return refl.JoinErrors(
		g.parseParametersIn(o, input, ParameterInQuery),
		g.parseParametersIn(o, input, ParameterInPath),
		g.parseParametersIn(o, input, ParameterInCookie),
		g.parseParametersIn(o, input, ParameterInHeader),
	)
}

func (g *Generator) parseParametersIn(o *Operation, input interface{}, in ParameterIn) error {
	schema, err := g.Parse(input,
		jsonschema.DefinitionsPrefix("#/components/parameters/"),
		jsonschema.InlineRefs,
		jsonschema.PropertyNameTag(string(in)),
	)
	if err != nil {
		return err
	}

	required := map[string]bool{}
	for _, name := range schema.Required {
		required[name] = true
	}

	for name, prop := range schema.Properties {
		s := SchemaOrRef{}
		s.FromJSONSchema(prop)

		p := ParameterOrRef{
			Parameter: &Parameter{
				Name:             name,
				In:               in,
				Description:      prop.TypeObject.Description,
				Required:         nil,
				Deprecated:       s.Schema.Deprecated,
				AllowEmptyValue:  nil,
				Style:            nil,
				Explode:          nil,
				AllowReserved:    nil,
				Schema:           &s,
				Content:          nil,
				Example:          nil,
				Examples:         nil,
				SchemaXORContent: nil,
				Location:         nil,
				MapOfAnything:    nil,
			},
		}

		if in == ParameterInPath || required[name] {
			p.Parameter.WithRequired(true)
		}

		o.Parameters = append(o.Parameters, p)
	}

	//if schema.Ref != nil {
	//	o.Parameters = append(o.Parameters, ParameterOrRef{
	//		ParameterReference: &ParameterReference{Ref: *schema.Ref},
	//	})
	//}
	//
	//for name, def := range schema.Definitions {
	//	if g.Spec.Components == nil {
	//		g.Spec.Components = &Components{}
	//	}
	//	if g.Spec.Components.Parameters == nil {
	//		g.Spec.Components.Parameters = &ComponentsParameters{}
	//	}
	//	s := SchemaOrRef{}
	//	s.FromJSONSchema(def)
	//
	//	g.Spec.Components.Parameters.WithMapOfParameterOrRefValuesItem(name, ParameterOrRef{
	//		Parameter: (&Parameter{}).WithSchema(s),
	//	})
	//}

	return nil
}

func (g *Generator) SetResponse(o *Operation, output interface{}) error {
	schema, err := g.Parse(output, jsonschema.DefinitionsPrefix("#/components/responses/"))
	if err != nil {
		return err
	}

	if o.Responses.MapOfResponseOrRefValues == nil {
		o.Responses.MapOfResponseOrRefValues = make(map[string]ResponseOrRef, 1)
	}

	o.Responses.MapOfResponseOrRefValues[strconv.Itoa(http.StatusOK)] = ResponseOrRef{
		Response: &Response{
			Description: "desc",
			Headers:     nil,
			Content: map[string]MediaType{
				"application/json": {
					Schema: &SchemaOrRef{
						SchemaReference: &SchemaReference{Ref: *schema.Ref},
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
		s.FromJSONSchema(def)

		g.Spec.Components.Responses.WithMapOfResponseOrRefValuesItem(name, ResponseOrRef{
			Response: (&Response{}).WithContent(map[string]MediaType{"application/json": {Schema: &s}}),
		})
	}

	return nil
}
