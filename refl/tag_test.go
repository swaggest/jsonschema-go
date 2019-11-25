package refl_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/swgen/refl"
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

func TestObjectHasXFields(t *testing.T) {
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
