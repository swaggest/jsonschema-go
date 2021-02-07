package jsonschema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/jsonschema-go"
)

func TestDate_MarshalText(t *testing.T) {
	var d jsonschema.Date

	assert.NoError(t, d.UnmarshalText([]byte("2021-05-08")))
	b, err := d.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, "2021-05-08", string(b))

	assert.EqualError(t, d.UnmarshalText([]byte("2021-05-088")), `parsing time "2021-05-088": extra text: "8"`)
}

func TestDate_MarshalJSON(t *testing.T) {
	var d jsonschema.Date

	assert.NoError(t, d.UnmarshalJSON([]byte(`"2021-05-08"`)))
	b, err := d.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `"2021-05-08"`, string(b))

	assert.EqualError(t, d.UnmarshalJSON([]byte(`""2021-05-088"`)), `invalid character '2' after top-level value`)
}
