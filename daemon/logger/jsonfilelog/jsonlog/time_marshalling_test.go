package jsonlog // import "github.com/tiborvass/docker/daemon/logger/jsonfilelog/jsonlog"

import (
	"testing"
	"time"

	"github.com/tiborvass/docker/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFastTimeMarshalJSONWithInvalidYear(t *testing.T) {
	aTime := time.Date(-1, 1, 1, 0, 0, 0, 0, time.Local)
	_, err := fastTimeMarshalJSON(aTime)
	testutil.ErrorContains(t, err, "year outside of range")

	anotherTime := time.Date(10000, 1, 1, 0, 0, 0, 0, time.Local)
	_, err = fastTimeMarshalJSON(anotherTime)
	testutil.ErrorContains(t, err, "year outside of range")
}

func TestFastTimeMarshalJSON(t *testing.T) {
	aTime := time.Date(2015, 5, 29, 11, 1, 2, 3, time.UTC)
	json, err := fastTimeMarshalJSON(aTime)
	require.NoError(t, err)
	assert.Equal(t, "\"2015-05-29T11:01:02.000000003Z\"", json)

	location, err := time.LoadLocation("Europe/Paris")
	require.NoError(t, err)

	aTime = time.Date(2015, 5, 29, 11, 1, 2, 3, location)
	json, err = fastTimeMarshalJSON(aTime)
	require.NoError(t, err)
	assert.Equal(t, "\"2015-05-29T11:01:02.000000003+02:00\"", json)
}
