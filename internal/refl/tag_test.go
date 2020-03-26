package refl_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/jsonschema-go/internal/refl"
)

type (
	structWithEmbedded struct {
		B int `path:"b" json:"-"`
		embedded
	}

	structWithTaggedEmbedded struct {
		B        int `path:"b" json:"-"`
		embedded `json:"emb"`
	}

	structWithIgnoredEmbedded struct {
		B        int `path:"b" json:"-"`
		embedded `json:"-"`
	}

	embedded struct {
		A int `json:"a"`
	}
)

func TestHasTaggedFields(t *testing.T) {
	assert.True(t, refl.HasTaggedFields(new(structWithEmbedded), "json"))
	assert.True(t, refl.HasTaggedFields(new(structWithTaggedEmbedded), "json"))
	assert.False(t, refl.HasTaggedFields(new(structWithIgnoredEmbedded), "json"))

	assert.True(t, refl.HasTaggedFields(new(structWithEmbedded), "path"))
	assert.False(t, refl.HasTaggedFields(new(structWithEmbedded), "query"))

	b, err := json.Marshal(structWithTaggedEmbedded{B: 10, embedded: embedded{A: 20}})
	assert.NoError(t, err)
	assert.Equal(t, `{"emb":{"a":20}}`, string(b))

	b, err = json.Marshal(structWithEmbedded{B: 10, embedded: embedded{A: 20}})
	assert.NoError(t, err)
	assert.Equal(t, `{"a":20}`, string(b))

	b, err = json.Marshal(structWithIgnoredEmbedded{B: 10, embedded: embedded{A: 20}})
	assert.NoError(t, err)
	assert.Equal(t, `{}`, string(b))
}

type schema struct {
	Title      string
	Desc       *string
	Min        *float64
	Max        float64
	Limit      int64
	Offset     *int64
	Deprecated bool
	Required   *bool
}

type value struct {
	Property string `title:"Value" desc:"..." min:"-1.23" max:"10.1" limit:"5" offset:"2" deprecated:"true" required:"f"`
}

func TestPopulateFieldsFromTags(t *testing.T) {
	s := schema{}
	tag := reflect.TypeOf(value{}).Field(0).Tag
	require.NoError(t, refl.PopulateFieldsFromTags(&s, tag))

	assert.Equal(t, "Value", s.Title)
	assert.Equal(t, "...", *s.Desc)
	assert.Equal(t, -1.23, *s.Min)
	assert.Equal(t, 10.1, s.Max)
	assert.Equal(t, int64(5), s.Limit)
	assert.Equal(t, int64(2), *s.Offset)
	assert.Equal(t, true, s.Deprecated)
	assert.Equal(t, false, *s.Required)
}
