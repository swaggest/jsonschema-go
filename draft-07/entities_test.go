package jsonschema_test

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	jsoniter "github.com/json-iterator/go"
	sejson "github.com/segmentio/encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	jsonschema "github.com/swaggest/jsonschema-go/draft-07"
	"github.com/yudai/gojsondiff/formatter"
)

var jsoni = jsoniter.ConfigCompatibleWithStandardLibrary

func TestSchema_MarshalJSON_roundtrip_draft7(t *testing.T) {
	data, err := ioutil.ReadFile("../resources/schema/draft-07.json")
	require.NoError(t, err)

	s := jsonschema.Schema{}
	require.NoError(t, json.Unmarshal(data, &s))

	marshaled, err := json.Marshal(s)
	require.NoError(t, err)
	assertjson.Comparer{
		FormatterConfig: formatter.AsciiFormatterConfig{
			Coloring: true,
		},
	}.Equal(t, data, marshaled)
}

// Pointers:
// BenchmarkSchema_UnmarshalJSON-4   	    3699	    311725 ns/op	   98225 B/op	    1256 allocs/op
// BenchmarkSchema_MarshalJSON-4     	    5398	    216649 ns/op	   45012 B/op	     931 allocs/op

// Values:
// BenchmarkSchema_UnmarshalJSON-4   	    3667	    315218 ns/op	  104194 B/op	    1217 allocs/op
// BenchmarkSchema_MarshalJSON-4     	    5157	    212650 ns/op	   44995 B/op	     931 allocs/op

func BenchmarkSchema_UnmarshalJSON(b *testing.B) {
	data, err := ioutil.ReadFile("../resources/schema/draft-07.json")
	require.NoError(b, err)
	b.ReportAllocs()
	b.ResetTimer()

	s := jsonschema.Schema{}
	for i := 0; i < b.N; i++ {
		_ = json.Unmarshal(data, &s)
	}
}

func BenchmarkSchema_UnmarshalJSON_segment(b *testing.B) {
	data, err := ioutil.ReadFile("../resources/schema/draft-07.json")
	require.NoError(b, err)
	b.ReportAllocs()
	b.ResetTimer()

	s := jsonschema.Schema{}
	for i := 0; i < b.N; i++ {
		_ = sejson.Unmarshal(data, &s)
	}
}

func BenchmarkSchema_UnmarshalJSON_jsoniter(b *testing.B) {
	data, err := ioutil.ReadFile("../resources/schema/draft-07.json")
	require.NoError(b, err)
	b.ReportAllocs()
	b.ResetTimer()

	s := jsonschema.Schema{}
	for i := 0; i < b.N; i++ {
		_ = jsoni.Unmarshal(data, &s)
	}
}

func BenchmarkSchema_MarshalJSON(b *testing.B) {
	data, err := ioutil.ReadFile("../resources/schema/draft-07.json")
	require.NoError(b, err)
	s := jsonschema.Schema{}
	require.NoError(b, json.Unmarshal(data, &s))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(&s)
	}
}

func BenchmarkSchema_MarshalJSON_segment(b *testing.B) {
	data, err := ioutil.ReadFile("../resources/schema/draft-07.json")
	require.NoError(b, err)
	s := jsonschema.Schema{}
	require.NoError(b, sejson.Unmarshal(data, &s))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = sejson.Marshal(&s)
	}
}

func BenchmarkSchema_MarshalJSON_jsoniter(b *testing.B) {
	data, err := ioutil.ReadFile("../resources/schema/draft-07.json")
	require.NoError(b, err)
	s := jsonschema.Schema{}
	require.NoError(b, jsoni.Unmarshal(data, &s))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = jsoni.Marshal(&s)
	}
}
