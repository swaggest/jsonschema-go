// Package sample contains test data.
package sample

// TestSampleStruct is a test dummy
type TestSampleStruct struct {
	SimpleFloat64 float64 `json:"simple_float64"`
	SimpleBool    bool    `json:"simple_bool"`

	Sub      TestSubStruct   `json:"sub"`
	SubSlice []TestSubStruct `json:"sub_slice"`

	AnonTypeStruct struct {
		FieldOne int `json:"int"`
	} `json:"anon_type_struct"`
}

// TestSubStruct is a test dummy
type TestSubStruct struct {
	SubInt int `json:"sample_int"`
}
