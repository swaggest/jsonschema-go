package refl_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/jsonschema-go/internal/refl"
	"github.com/swaggest/jsonschema-go/internal/refl/internal/sample"
)

func TestDeepIndirect(t *testing.T) {
	assert.Equal(t, reflect.Struct, refl.DeepIndirect(reflect.TypeOf(new(***sample.TestSampleStruct))).Kind())
}
