package refl_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/swgen/internal/sample"
	"github.com/swaggest/swgen/refl"
)

func TestDeepIndirect(t *testing.T) {
	assert.Equal(t, reflect.Struct, refl.DeepIndirect(reflect.TypeOf(new(***sample.TestSampleStruct))).Kind())
}
