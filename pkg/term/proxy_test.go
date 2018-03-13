package term // import "github.com/tiborvass/docker/pkg/term"

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestEscapeProxyRead(t *testing.T) {
	escapeKeys, _ := ToBytes("DEL")
	keys, _ := ToBytes("a,b,c,+")
	reader := NewEscapeProxy(bytes.NewReader(keys), escapeKeys)
	buf := make([]byte, len(keys))
	nr, err := reader.Read(buf)
	assert.NilError(t, err)
	assert.Equal(t, nr, len(keys), fmt.Sprintf("nr %d should be equal to the number of %d", nr, len(keys)))
	assert.Assert(t, is.DeepEqual(keys, buf), "keys & the read buffer should be equal")

	keys, _ = ToBytes("")
	reader = NewEscapeProxy(bytes.NewReader(keys), escapeKeys)
	buf = make([]byte, len(keys))
	nr, err = reader.Read(buf)
	assert.Assert(t, is.ErrorContains(err, ""), "Should throw error when no keys are to read")
	assert.Equal(t, nr, 0, "nr should be zero")
	assert.Check(t, is.Len(keys, 0))
	assert.Check(t, is.Len(buf, 0))

	escapeKeys, _ = ToBytes("ctrl-x,ctrl-@")
	keys, _ = ToBytes("DEL")
	reader = NewEscapeProxy(bytes.NewReader(keys), escapeKeys)
	buf = make([]byte, len(keys))
	nr, err = reader.Read(buf)
	assert.NilError(t, err)
	assert.Equal(t, nr, 1, fmt.Sprintf("nr %d should be equal to the number of 1", nr))
	assert.Assert(t, is.DeepEqual(keys, buf), "keys & the read buffer should be equal")

	escapeKeys, _ = ToBytes("ctrl-c")
	keys, _ = ToBytes("ctrl-c")
	reader = NewEscapeProxy(bytes.NewReader(keys), escapeKeys)
	buf = make([]byte, len(keys))
	nr, err = reader.Read(buf)
	assert.Error(t, err, "read escape sequence")
	assert.Equal(t, nr, 0, "nr should be equal to 0")
	assert.Assert(t, is.DeepEqual(keys, buf), "keys & the read buffer should be equal")

	escapeKeys, _ = ToBytes("ctrl-c,ctrl-z")
	keys, _ = ToBytes("ctrl-c,ctrl-z")
	reader = NewEscapeProxy(bytes.NewReader(keys), escapeKeys)
	buf = make([]byte, 1)
	nr, err = reader.Read(buf)
	assert.NilError(t, err)
	assert.Equal(t, nr, 0, "nr should be equal to 0")
	assert.Assert(t, is.DeepEqual(keys[0:1], buf), "keys & the read buffer should be equal")
	nr, err = reader.Read(buf)
	assert.Error(t, err, "read escape sequence")
	assert.Equal(t, nr, 0, "nr should be equal to 0")
	assert.Assert(t, is.DeepEqual(keys[1:], buf), "keys & the read buffer should be equal")

	escapeKeys, _ = ToBytes("ctrl-c,ctrl-z")
	keys, _ = ToBytes("ctrl-c,DEL,+")
	reader = NewEscapeProxy(bytes.NewReader(keys), escapeKeys)
	buf = make([]byte, 1)
	nr, err = reader.Read(buf)
	assert.NilError(t, err)
	assert.Equal(t, nr, 0, "nr should be equal to 0")
	assert.Assert(t, is.DeepEqual(keys[0:1], buf), "keys & the read buffer should be equal")
	buf = make([]byte, len(keys))
	nr, err = reader.Read(buf)
	assert.NilError(t, err)
	assert.Equal(t, nr, len(keys), fmt.Sprintf("nr should be equal to %d", len(keys)))
	assert.Assert(t, is.DeepEqual(keys, buf), "keys & the read buffer should be equal")

	escapeKeys, _ = ToBytes("ctrl-c,ctrl-z")
	keys, _ = ToBytes("ctrl-c,DEL")
	reader = NewEscapeProxy(bytes.NewReader(keys), escapeKeys)
	buf = make([]byte, 1)
	nr, err = reader.Read(buf)
	assert.NilError(t, err)
	assert.Equal(t, nr, 0, "nr should be equal to 0")
	assert.Assert(t, is.DeepEqual(keys[0:1], buf), "keys & the read buffer should be equal")
	buf = make([]byte, len(keys))
	nr, err = reader.Read(buf)
	assert.NilError(t, err)
	assert.Equal(t, nr, len(keys), fmt.Sprintf("nr should be equal to %d", len(keys)))
	assert.Assert(t, is.DeepEqual(keys, buf), "keys & the read buffer should be equal")
}
