package streamformatter // import "github.com/tiborvass/docker/pkg/streamformatter"

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/tiborvass/docker/pkg/jsonmessage"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestRawProgressFormatterFormatStatus(t *testing.T) {
	sf := rawProgressFormatter{}
	res := sf.formatStatus("ID", "%s%d", "a", 1)
	assert.Check(t, is.Equal("a1\r\n", string(res)))
}

func TestRawProgressFormatterFormatProgress(t *testing.T) {
	sf := rawProgressFormatter{}
	jsonProgress := &jsonmessage.JSONProgress{
		Current: 15,
		Total:   30,
		Start:   1,
	}
	res := sf.formatProgress("id", "action", jsonProgress, nil)
	out := string(res)
	assert.Check(t, strings.HasPrefix(out, "action [===="))
	assert.Check(t, is.Contains(out, "15B/30B"))
	assert.Check(t, strings.HasSuffix(out, "\r"))
}

func TestFormatStatus(t *testing.T) {
	res := FormatStatus("ID", "%s%d", "a", 1)
	expected := `{"status":"a1","id":"ID"}` + streamNewline
	assert.Check(t, is.Equal(expected, string(res)))
}

func TestFormatError(t *testing.T) {
	res := FormatError(errors.New("Error for formatter"))
	expected := `{"errorDetail":{"message":"Error for formatter"},"error":"Error for formatter"}` + "\r\n"
	assert.Check(t, is.Equal(expected, string(res)))
}

func TestFormatJSONError(t *testing.T) {
	err := &jsonmessage.JSONError{Code: 50, Message: "Json error"}
	res := FormatError(err)
	expected := `{"errorDetail":{"code":50,"message":"Json error"},"error":"Json error"}` + streamNewline
	assert.Check(t, is.Equal(expected, string(res)))
}

func TestJsonProgressFormatterFormatProgress(t *testing.T) {
	sf := &jsonProgressFormatter{}
	jsonProgress := &jsonmessage.JSONProgress{
		Current: 15,
		Total:   30,
		Start:   1,
	}
	res := sf.formatProgress("id", "action", jsonProgress, &AuxFormatter{Writer: &bytes.Buffer{}})
	msg := &jsonmessage.JSONMessage{}

	assert.NilError(t, json.Unmarshal(res, msg))
	assert.Check(t, is.Equal("id", msg.ID))
	assert.Check(t, is.Equal("action", msg.Status))

	// jsonProgress will always be in the format of:
	// [=========================>                         ]      15B/30B 412910h51m30s
	// The last entry '404933h7m11s' is the timeLeftBox.
	// However, the timeLeftBox field may change as jsonProgress.String() depends on time.Now().
	// Therefore, we have to strip the timeLeftBox from the strings to do the comparison.

	// Compare the jsonProgress strings before the timeLeftBox
	expectedProgress := "[=========================>                         ]      15B/30B"
	// if terminal column is <= 110, expectedProgressShort is expected.
	expectedProgressShort := "      15B/30B"
	if !(strings.HasPrefix(msg.ProgressMessage, expectedProgress) ||
		strings.HasPrefix(msg.ProgressMessage, expectedProgressShort)) {
		t.Fatalf("ProgressMessage without the timeLeftBox must be %s or %s, got: %s",
			expectedProgress, expectedProgressShort, msg.ProgressMessage)
	}

	assert.Check(t, is.DeepEqual(jsonProgress, msg.Progress))
}

func TestJsonProgressFormatterFormatStatus(t *testing.T) {
	sf := jsonProgressFormatter{}
	res := sf.formatStatus("ID", "%s%d", "a", 1)
	assert.Check(t, is.Equal(`{"status":"a1","id":"ID"}`+streamNewline, string(res)))
}

func TestNewJSONProgressOutput(t *testing.T) {
	b := bytes.Buffer{}
	b.Write(FormatStatus("id", "Downloading"))
	_ = NewJSONProgressOutput(&b, false)
	assert.Check(t, is.Equal(`{"status":"Downloading","id":"id"}`+streamNewline, b.String()))
}

func TestAuxFormatterEmit(t *testing.T) {
	b := bytes.Buffer{}
	aux := &AuxFormatter{Writer: &b}
	sampleAux := &struct {
		Data string
	}{"Additional data"}
	err := aux.Emit(sampleAux)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(`{"aux":{"Data":"Additional data"}}`+streamNewline, b.String()))
}
