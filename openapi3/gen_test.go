package openapi3_test

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/swaggest/jsonschema-go/openapi3"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"testing"
)

type WeirdResp interface {
	Boo()
}

type Resp struct {
	HeaderField string `header:"X-Header-Field" description:"Sample header response."`
	Field1      int    `json:"field1"`
	Field2      string `json:"field2"`
	Info        struct {
		Foo string  `json:"foo" default:"baz" required:"true" pattern:"\\d+"`
		Bar float64 `json:"bar" description:"This is Bar."`
	} `json:"info"`
	Parent               *Resp                  `json:"parent"`
	Map                  map[string]int64       `json:"map"`
	MapOfAnything        map[string]interface{} `json:"mapOfAnything"`
	ArrayOfAnything      []interface{}          `json:"arrayOfAnything"`
	Whatever             interface{}            `json:"whatever"`
	NullableWhatever     *interface{}           `json:"nullableWhatever,omitempty"`
	RecursiveArray       []WeirdResp            `json:"recursiveArray"`
	RecursiveStructArray []Resp                 `json:"recursiveStructArray"`
}

func (r Resp) Describe() string {
	return "This is a sample response."
}

func (r Resp) Title() string {
	return "Sample Response"
}

type Req struct {
	InQuery1 int                   `query:"in_query1" required:"true" description:"Query parameter."`
	InQuery2 int                   `query:"in_query2" required:"true" description:"Query parameter."`
	InQuery3 int                   `query:"in_query3" required:"true" description:"Query parameter."`
	InPath   int                   `path:"in_path"`
	InCookie string                `cookie:"in_cookie" deprecated:"true"`
	InHeader float64               `header:"in_header"`
	InBody1  int                   `json:"in_body1"`
	InBody2  string                `json:"in_body2"`
	InForm1  string                `formData:"in_form1"`
	InForm2  string                `formData:"in_form2"`
	File1    multipart.File        `formData:"upload1"`
	File2    *multipart.FileHeader `formData:"upload2"`
}

type GetReq struct {
	InQuery1 int     `query:"in_query1" required:"true" description:"Query parameter."`
	InQuery2 int     `query:"in_query2" required:"true" description:"Query parameter."`
	InQuery3 int     `query:"in_query3" required:"true" description:"Query parameter."`
	InPath   int     `path:"in_path"`
	InCookie string  `cookie:"in_cookie" deprecated:"true"`
	InHeader float64 `header:"in_header"`
}

func TestGenerator_SetResponse(t *testing.T) {
	g := openapi3.Generator{}

	s := openapi3.Spec{}
	s.Info.Title = "SampleAPI"
	s.Info.Version = "1.2.3"

	g.Spec = &s
	g.AddTypeMapping(new(WeirdResp), new(Resp))

	op := openapi3.Operation{}

	//op.WithDeprecated(true)

	err := g.SetRequest(&op, new(Req), http.MethodPost)
	assert.NoError(t, err)

	err = g.SetJSONResponse(&op, new(WeirdResp), http.StatusOK)
	assert.NoError(t, err)

	err = g.SetJSONResponse(&op, new([]WeirdResp), http.StatusConflict)
	assert.NoError(t, err)

	pathItem := s.Paths.MapOfPathItemValues["/somewhere/{in_path}"]
	pathItem.
		WithSummary("Path Summary").
		WithDescription("Path Description")

	pathItem.WithOperation(http.MethodPost, op)

	op = openapi3.Operation{}

	err = g.SetRequest(&op, new(GetReq), http.MethodGet)
	assert.NoError(t, err)

	err = g.SetJSONResponse(&op, new(Resp), http.StatusOK)
	assert.NoError(t, err)

	pathItem.WithOperation(http.MethodGet, op)

	s.Paths.WithMapOfPathItemValuesItem(
		"/somewhere/{in_path}",
		pathItem,
	)

	b, err := json.MarshalIndent(s, "", " ")
	assert.NoError(t, err)

	ioutil.WriteFile("openapi.json", b, 0640)
	println(string(b))
}
