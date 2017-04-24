package image

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/cli/internal/test"
	"github.com/tiborvass/docker/pkg/testutil"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewImportCommandErrors(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		expectedError   string
		imageImportFunc func(source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error)
	}{
		{
			name:          "wrong-args",
			args:          []string{},
			expectedError: "requires at least 1 argument(s).",
		},
		{
			name:          "import-failed",
			args:          []string{"testdata/import-command-success.input.txt"},
			expectedError: "something went wrong",
			imageImportFunc: func(source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error) {
				return nil, errors.Errorf("something went wrong")
			},
		},
	}
	for _, tc := range testCases {
		buf := new(bytes.Buffer)
		cmd := NewImportCommand(test.NewFakeCli(&fakeClient{imageImportFunc: tc.imageImportFunc}, buf))
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNewImportCommandInvalidFile(t *testing.T) {
	cmd := NewImportCommand(test.NewFakeCli(&fakeClient{}, new(bytes.Buffer)))
	cmd.SetOutput(ioutil.Discard)
	cmd.SetArgs([]string{"testdata/import-command-success.unexistent-file"})
	testutil.ErrorContains(t, cmd.Execute(), "testdata/import-command-success.unexistent-file")
}

func TestNewImportCommandSuccess(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		imageImportFunc func(source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error)
	}{
		{
			name: "simple",
			args: []string{"testdata/import-command-success.input.txt"},
		},
		{
			name: "terminal-source",
			args: []string{"-"},
		},
		{
			name: "double",
			args: []string{"-", "image:local"},
			imageImportFunc: func(source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error) {
				assert.Equal(t, ref, "image:local")
				return ioutil.NopCloser(strings.NewReader("")), nil
			},
		},
		{
			name: "message",
			args: []string{"--message", "test message", "-"},
			imageImportFunc: func(source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error) {
				assert.Equal(t, options.Message, "test message")
				return ioutil.NopCloser(strings.NewReader("")), nil
			},
		},
		{
			name: "change",
			args: []string{"--change", "ENV DEBUG true", "-"},
			imageImportFunc: func(source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error) {
				assert.Equal(t, options.Changes[0], "ENV DEBUG true")
				return ioutil.NopCloser(strings.NewReader("")), nil
			},
		},
	}
	for _, tc := range testCases {
		buf := new(bytes.Buffer)
		cmd := NewImportCommand(test.NewFakeCli(&fakeClient{imageImportFunc: tc.imageImportFunc}, buf))
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		assert.NoError(t, cmd.Execute())
	}
}
