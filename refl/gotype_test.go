package refl_test

//func TestGoType(t *testing.T) {
//	assert.Equal(
//		t,
//		"github.com/swaggest/swgen/internal/sample.TestSampleStruct",
//		refl.GoType(reflect.TypeOf(sample.TestSampleStruct{})),
//	)
//	assert.Equal(
//		t,
//		"*github.com/swaggest/swgen/internal/sample.TestSampleStruct",
//		refl.GoType(reflect.TypeOf(new(sample.TestSampleStruct))),
//	)
//	assert.Equal(
//		t,
//		"*github.com/swaggest/swgen/internal/sample.TestSampleStruct",
//		refl.GoType(reflect.TypeOf(new(sample.TestSampleStruct))),
//	)
//	assert.Equal(
//		t,
//		"*github.com/swaggest/swgen/internal/Fancy-Path.Sample::fancypath.Sample",
//		refl.GoType(reflect.TypeOf(new(fancypath.Sample))),
//	)
//	assert.Equal(
//		t,
//		"*[]map[*github.com/swaggest/swgen/internal/Fancy-Path.Sample::fancypath.Sample]github.com/swaggest/swgen/internal/Fancy-Path.Sample::fancypath.Sample",
//		refl.GoType(reflect.TypeOf(new([]map[*fancypath.Sample]fancypath.Sample))),
//	)
//}
