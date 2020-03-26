package refl_test

import (
	"reflect"
	"testing"

	fancypath "github.com/swaggest/jsonschema-go/internal/refl/internal/Fancy-Path"
	"github.com/swaggest/jsonschema-go/internal/refl/internal/sample"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/jsonschema-go/internal/refl"
)

func TestGoType(t *testing.T) {
	assert.Equal(
		t,
		refl.TypeString("github.com/swaggest/jsonschema-go/internal/refl/internal/sample.TestSampleStruct"),
		refl.GoType(reflect.TypeOf(sample.TestSampleStruct{})),
	)
	assert.Equal(
		t,
		refl.TypeString("*github.com/swaggest/jsonschema-go/internal/refl/internal/sample.TestSampleStruct"),
		refl.GoType(reflect.TypeOf(new(sample.TestSampleStruct))),
	)
	assert.Equal(
		t,
		refl.TypeString("*github.com/swaggest/jsonschema-go/internal/refl/internal/Fancy-Path.Sample::fancypath.Sample"),
		refl.GoType(reflect.TypeOf(new(fancypath.Sample))),
	)
}
