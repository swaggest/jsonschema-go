package jsonschema_test

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/swaggest/jsonschema-go"

	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
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

// Sample results:
//
// BenchmarkSchema_UnmarshalJSON-4            	     901	   1425573 ns/op	  196753 B/op	    2229 allocs/op
// BenchmarkSchema_UnmarshalJSON_segment-4    	    1006	   1188726 ns/op	  196462 B/op	    2224 allocs/op
// BenchmarkSchema_UnmarshalJSON_jsoniter-4   	    1257	    943759 ns/op	  203267 B/op	    2377 allocs/op
// BenchmarkSchema_MarshalJSON-4              	    2252	    483460 ns/op	   95878 B/op	    1139 allocs/op
// BenchmarkSchema_MarshalJSON_segment-4      	    2616	    477669 ns/op	   96716 B/op	    1135 allocs/op
// BenchmarkSchema_MarshalJSON_jsoniter-4     	    2689	    446078 ns/op	   95704 B/op	    1135 allocs/op

//with easy json
// BenchmarkSchema_UnmarshalJSON_raw-4        	   10000	    109209 ns/op	   31803 B/op	     454 allocs/op
//BenchmarkSchema_UnmarshalJSON-4            	    1348	    879458 ns/op	  207066 B/op	    2142 allocs/op
//BenchmarkSchema_UnmarshalJSON_segment-4    	    1262	    841815 ns/op	  206753 B/op	    2136 allocs/op
//BenchmarkSchema_UnmarshalJSON_jsoniter-4   	    1189	    942688 ns/op	  213574 B/op	    2290 allocs/op
//BenchmarkSchema_MarshalJSON_raw-4          	   11662	     99129 ns/op	   29589 B/op	     617 allocs/op
//BenchmarkSchema_MarshalJSON-4              	    3078	    362781 ns/op	  108275 B/op	    1145 allocs/op
//BenchmarkSchema_MarshalJSON_segment-4      	    2966	    368278 ns/op	  109104 B/op	    1141 allocs/op
//BenchmarkSchema_MarshalJSON_jsoniter-4     	    3460	    337175 ns/op	  108121 B/op	    1141 allocs/op

// without easyjson
//BenchmarkSchema_UnmarshalJSON_raw-4        	   10000	    119292 ns/op	   31802 B/op	     454 allocs/op
//BenchmarkSchema_UnmarshalJSON-4            	    1308	    909309 ns/op	  196621 B/op	    2221 allocs/op
//BenchmarkSchema_UnmarshalJSON_segment-4    	    1258	    961455 ns/op	  196333 B/op	    2216 allocs/op
//BenchmarkSchema_UnmarshalJSON_jsoniter-4   	     963	   1055032 ns/op	  203141 B/op	    2369 allocs/op
//BenchmarkSchema_MarshalJSON_raw-4          	   11077	    103489 ns/op	   29589 B/op	     617 allocs/op
//BenchmarkSchema_MarshalJSON-4              	    2637	    556107 ns/op	   95886 B/op	    1139 allocs/op
//BenchmarkSchema_MarshalJSON_segment-4      	    2745	    432062 ns/op	   96715 B/op	    1135 allocs/op
//BenchmarkSchema_MarshalJSON_jsoniter-4     	    2982	    395426 ns/op	   95705 B/op	    1135 allocs/op

func BenchmarkSchema_UnmarshalJSON_raw(b *testing.B) {
	data, err := ioutil.ReadFile("../resources/schema/draft-07.json")
	require.NoError(b, err)
	b.ReportAllocs()
	b.ResetTimer()

	var s interface{}

	for i := 0; i < b.N; i++ {
		_ = json.Unmarshal(data, &s)
	}
}

func BenchmarkSchema_UnmarshalJSON(b *testing.B) {
	data, err := ioutil.ReadFile("../resources/schema/draft-07.json")
	require.NoError(b, err)
	b.ReportAllocs()
	b.ResetTimer()

	s := jsonschema.SchemaOrBool{}

	for i := 0; i < b.N; i++ {
		_ = json.Unmarshal(data, &s)
	}
}

func BenchmarkSchema_MarshalJSON_raw(b *testing.B) {
	data, err := ioutil.ReadFile("../resources/schema/draft-07.json")
	require.NoError(b, err)

	var s interface{}

	require.NoError(b, json.Unmarshal(data, &s))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(&s)
	}
}

func BenchmarkSchema_MarshalJSON(b *testing.B) {
	data, err := ioutil.ReadFile("../resources/schema/draft-07.json")
	require.NoError(b, err)

	s := jsonschema.SchemaOrBool{}
	require.NoError(b, json.Unmarshal(data, &s))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(&s)
	}
}
