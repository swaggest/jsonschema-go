package refl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/swgen/internal/sample"
	"github.com/swaggest/swgen/refl"
)

func TestIsSliceOrMap(t *testing.T) {
	assert.True(t, refl.IsSliceOrMap(new(***[]sample.TestSampleStruct)))
	assert.True(t, refl.IsSliceOrMap(new(***map[string]sample.TestSampleStruct)))
	assert.True(t, refl.IsSliceOrMap([]int{}))
	assert.True(t, refl.IsSliceOrMap(map[int]int{}))
	assert.False(t, refl.IsSliceOrMap(new(***sample.TestSampleStruct)))
	assert.False(t, refl.IsSliceOrMap(nil))
}

func TestIsStruct(t *testing.T) {
	assert.False(t, refl.IsStruct(new(***[]sample.TestSampleStruct)))
	assert.False(t, refl.IsStruct(new(***map[string]sample.TestSampleStruct)))
	assert.False(t, refl.IsStruct([]int{}))
	assert.False(t, refl.IsStruct(map[int]int{}))
	assert.True(t, refl.IsStruct(new(***sample.TestSampleStruct)))
	assert.True(t, refl.IsStruct(sample.TestSampleStruct{}))
	assert.False(t, refl.IsStruct(nil))
}
