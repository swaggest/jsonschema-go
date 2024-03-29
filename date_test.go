package jsonschema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/jsonschema-go"
)

func TestDate_MarshalText(t *testing.T) {
	var d jsonschema.Date

	require.NoError(t, d.UnmarshalText([]byte("2021-05-08")))
	b, err := d.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, "2021-05-08", string(b))

	assert.Error(t, d.UnmarshalText([]byte("2021-05-088")))
}

func TestDate_MarshalJSON(t *testing.T) {
	var d jsonschema.Date

	require.NoError(t, d.UnmarshalJSON([]byte(`"2021-05-08"`)))
	b, err := d.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"2021-05-08"`, string(b))

	assert.Error(t, d.UnmarshalJSON([]byte(`""2021-05-088"`)))
}
