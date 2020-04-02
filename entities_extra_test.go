package jsonschema_test

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/jsonschema-go"
	"github.com/yudai/gojsondiff/formatter"
)

func TestSchema_MarshalJSON_roundtrip_draft7(t *testing.T) {
	data, err := ioutil.ReadFile("./resources/schema/draft-07.json")
	require.NoError(t, err)

	s := jsonschema.SchemaOrBool{}
	require.NoError(t, json.Unmarshal(data, &s))

	marshaled, err := json.Marshal(s)
	require.NoError(t, err)
	assertjson.Comparer{
		FormatterConfig: formatter.AsciiFormatterConfig{
			Coloring: true,
		},
	}.Equal(t, data, marshaled)
}

func BenchmarkSchema_UnmarshalJSON_raw(b *testing.B) {
	data, err := ioutil.ReadFile("./resources/schema/draft-07.json")
	require.NoError(b, err)
	b.ReportAllocs()
	b.ResetTimer()

	var s interface{}

	for i := 0; i < b.N; i++ {
		err = json.Unmarshal(data, &s)
		require.NoError(b, err)
	}
}

func BenchmarkSchema_UnmarshalJSON(b *testing.B) {
	data, err := ioutil.ReadFile("./resources/schema/draft-07.json")
	require.NoError(b, err)
	b.ReportAllocs()
	b.ResetTimer()

	s := jsonschema.SchemaOrBool{}

	for i := 0; i < b.N; i++ {
		err = json.Unmarshal(data, &s)
		require.NoError(b, err)
	}
}

func BenchmarkSchema_MarshalJSON_raw(b *testing.B) {
	data, err := ioutil.ReadFile("./resources/schema/draft-07.json")
	require.NoError(b, err)

	var s interface{}

	require.NoError(b, json.Unmarshal(data, &s))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err = json.Marshal(&s)
		require.NoError(b, err)
	}
}

func BenchmarkSchema_MarshalJSON(b *testing.B) {
	data, err := ioutil.ReadFile("./resources/schema/draft-07.json")
	require.NoError(b, err)

	s := jsonschema.SchemaOrBool{}
	require.NoError(b, json.Unmarshal(data, &s))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err = json.Marshal(&s)
		require.NoError(b, err)
	}
}
