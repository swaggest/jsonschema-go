package openapi3

import (
	jsonschema "github.com/swaggest/jsonschema-go/draft-07"
	"github.com/swaggest/jsonschema-go/refl"
	"net/http"
	"strconv"
	"strings"
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
		g.parseRequestBody(o, input, "json", "application/json"),
		g.parseRequestBody(o, input, "formData", "application/x-www-form-urlencoded"),
	)
}

func (g *Generator) parseRequestBody(o *Operation, input interface{}, tag, mime string) error {
	schema, err := g.Parse(input,
		jsonschema.DefinitionsPrefix("#/components/schemas/"+strings.Title(tag)),
		jsonschema.PropertyNameTag(tag),
	)
	if err != nil {
		return err
	}

	mt := MediaType{
		Schema: &SchemaOrRef{
			SchemaReference: &SchemaReference{Ref: *schema.Ref},
		},
		Example:       nil,
		Examples:      nil,
		Encoding:      nil,
		MapOfAnything: nil,
	}

	for name, def := range schema.Definitions {
		if g.Spec.Components == nil {
			g.Spec.Components = &Components{}
		}
		if g.Spec.Components.Schemas == nil {
			g.Spec.Components.Schemas = &ComponentsSchemas{}
		}
		s := SchemaOrRef{}
		s.FromJSONSchema(def)

		g.Spec.Components.Schemas.WithMapOfSchemaOrRefValuesItem(strings.Title(tag)+name, s)
	}

	if o.RequestBody == nil {
		o.RequestBody = &RequestBodyOrRef{}
	}

	if o.RequestBody.RequestBody == nil {
		o.RequestBody.RequestBody = &RequestBody{}
	}

	if o.RequestBody.RequestBody.Content == nil {
		o.RequestBody.RequestBody.Content = map[string]MediaType{}
	}
	o.RequestBody.RequestBody.Content[mime] = mt

	return nil
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
				Name:            name,
				In:              in,
				Description:     prop.TypeObject.Description,
				Required:        nil,
				Deprecated:      s.Schema.Deprecated,
				AllowEmptyValue: nil,
				Style:           nil,
				Explode:         nil,
				AllowReserved:   nil,
				Schema:          &s,
				Content:         nil,
				Example:         nil,
				Examples:        nil,
				Location:        nil,
				MapOfAnything:   nil,
			},
		}

		if in == ParameterInPath || required[name] {
			p.Parameter.WithRequired(true)
		}

		o.Parameters = append(o.Parameters, p)
	}

	return nil
}

func (g *Generator) parseResponseHeader(output interface{}) (map[string]HeaderOrRef, error) {
	schema, err := g.Parse(output,
		jsonschema.DefinitionsPrefix("#/components/headers/"),
		jsonschema.InlineRefs,
		jsonschema.PropertyNameTag("header"),
	)
	if err != nil {
		return nil, err
	}

	required := map[string]bool{}
	for _, name := range schema.Required {
		required[name] = true
	}

	res := make(map[string]HeaderOrRef, len(schema.Properties))

	for name, prop := range schema.Properties {
		s := SchemaOrRef{}
		s.FromJSONSchema(prop)

		header := Header{
			Description:     prop.TypeObject.Description,
			Deprecated:      s.Schema.Deprecated,
			AllowEmptyValue: nil,
			Explode:         nil,
			AllowReserved:   nil,
			Schema:          &s,
			Content:         nil,
			Example:         nil,
			Examples:        nil,
			MapOfAnything:   nil,
		}

		if required[name] {
			header.WithRequired(true)
		}

		res[name] = HeaderOrRef{
			Header: &header,
		}
	}

	return res, nil
}

func (g *Generator) SetJSONResponse(o *Operation, output interface{}) error {
	schema, err := g.Parse(output, jsonschema.DefinitionsPrefix("#/components/schemas/"))
	if err != nil {
		return err
	}

	if o.Responses.MapOfResponseOrRefValues == nil {
		o.Responses.MapOfResponseOrRefValues = make(map[string]ResponseOrRef, 1)
	}

	resp := Response{
		Content: map[string]MediaType{
			"application/json": {
				Schema: &SchemaOrRef{
					SchemaReference: &SchemaReference{Ref: *schema.Ref},
				},
				Example:       nil,
				Examples:      nil,
				Encoding:      nil,
				MapOfAnything: nil,
			},
		},
	}

	resp.Headers, err = g.parseResponseHeader(output)
	if err != nil {
		return err
	}

	for name, def := range schema.Definitions {
		if g.Spec.Components == nil {
			g.Spec.Components = &Components{}
		}
		if g.Spec.Components.Schemas == nil {
			g.Spec.Components.Schemas = &ComponentsSchemas{}
		}
		s := SchemaOrRef{}
		s.FromJSONSchema(def)

		g.Spec.Components.Schemas.WithMapOfSchemaOrRefValuesItem(name, s)
	}

	o.Responses.MapOfResponseOrRefValues[strconv.Itoa(http.StatusOK)] = ResponseOrRef{
		Response: &resp,
	}

	return nil
}
