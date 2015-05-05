package namespace

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/Sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestEntryLessEqual(t *testing.T) {
	for _, testcase := range []struct {
		// For each case, we expect a to be less than b. The inverse is
		// checked. If they are supposed to be equal, only the inverse
		// checked.
		a     Entry
		b     Entry
		equal bool
	}{
		{
			a:     mustEntry("docker.com/    push         https://registry.docker.com"),
			b:     mustEntry("docker.com/    push         https://registry.docker.com"),
			equal: true,
		},
		{
			a: mustEntry("docker.com/    push         https://aregistry.docker.com"),
			b: mustEntry("docker.com/    push         https://registry.docker.com"),
		},
		{
			a: mustEntry("docker.com/        pull         https://registry.docker.com"),
			b: mustEntry("docker.com/        push         https://registry.docker.com"),
		},
		{
			a: mustEntry("docker.com/        push         https://registry.docker.com"),
			b: mustEntry("foo/               pull         https://foo.com"),
		},
	} {
		if !testcase.equal && !entryLess(testcase.a, testcase.b) {
			t.Fatalf("expected %v less than %v", testcase.a, testcase.b)
		}

		// Opposite must be true
		if entryLess(testcase.b, testcase.a) {
			t.Fatalf("expected %v not less than %v", testcase.b, testcase.a)
		}

		if testcase.equal && !entryEqual(testcase.a, testcase.b) {
			t.Fatalf("expected %v == %v", testcase.a, testcase.b)
		}

		if testcase.equal && !entryEqual(testcase.b, testcase.a) {
			t.Fatalf("expected %v == %v", testcase.a, testcase.b)
		}
	}
}

func TestSortAndInsert(t *testing.T) {
	// completely unsorted junk
	namespaceConfig := `
docker.com/        push         https://registry.docker.com
docker.com/        pull         https://registry.docker.com
docker.com/        pull         https://aregistry.docker.com
foo/               pull         http://foo.com
foo/               pull         http://foo.com
foo/               pull         http://foo.com
foo/               pull         http://foo.com
foo/               pull         http://foo.com
foo/               pull         http://foo.com
foo/               pull         http://foo.com
a/b/c/             push         https://abc.com
`
	entries := mustEntries(namespaceConfig)

	var buf bytes.Buffer
	if err := WriteEntries(&buf, entries); err != nil {
		t.Fatalf("unexpected error writing entries: %v", err)
	}

	sort.Sort(entries.entries)

	// Out expected output.
	expected := strings.TrimSpace(`
a/b/c/         push    https://abc.com
docker.com/    pull    https://aregistry.docker.com
docker.com/    pull    https://registry.docker.com
docker.com/    push    https://registry.docker.com
foo/           pull    http://foo.com
`)

	if strings.TrimSpace(buf.String()) != strings.TrimSpace(expected) {
		t.Fatalf("\n%q\n != \n%q", strings.TrimSpace(buf.String()), strings.TrimSpace(expected))
	}
}

func entryString(entry Entry) string {
	return fmt.Sprintf("%s %s %s", entry.scope, entry.action, strings.Join(entry.args, " "))
}

func assertEntryEqual(t *testing.T, actual, expected Entry) {
	s1 := entryString(actual)
	s2 := entryString(expected)
	if s1 != s2 {
		t.Fatalf("Unexpected entry\n\tExpected: %s\n\tActual:   %s", s2, s1)
	}
}

func assertResolution(t *testing.T, r Resolver, name, matchString string) {
	matchEntries := mustEntries(matchString)

	entries, err := r.Resolve(name)
	if err != nil {
		t.Fatalf("Error resolving namespace: %s", err)
	}

	if len(entries.entries) != len(matchEntries.entries) {
		t.Fatalf("Unexpected number of entries for %q: %d, expected %d", name, len(entries.entries), len(matchEntries.entries))
	}

	for i := range entries.entries {
		assertEntryEqual(t, entries.entries[i], matchEntries.entries[i])
	}
}

// Test case
// No base + discovery
// Base with extension discovery
// Base with no scoped discovery
func TestMultiResolution(t *testing.T) {
	entries1 := mustEntries(`
docker.com/        push         https://registry.base.docker.com
docker.com/        pull         https://registry.base.docker.com
docker.com/        index        https://search.base.docker.com
docker.com/other   ca           https://trust.docker.com/cert.crt
docker.com/other   namespace       docker.com/
docker.com/other/block   namespace
docker.com/other/sub   index    https://search.sub.docker.com
docker.com/other/sub   namespace   docker.com/other
docker.com/extend  namespace       docker.com/
docker.com/img/sub  pull       https://mirror.base.docker.com
docker.com/img/sub  namespace       docker.com/img
`)
	entries2 := mustEntries(`
docker.com/img        push         https://registry.docker.com
docker.com/img        pull         https://registry.docker.com
docker.com/img        index        https://search.docker.com
`)

	resolver := NewMultiResolver(NewSimpleResolver(entries1, false), NewSimpleResolver(entries2, false))

	assertResolution(t, resolver, "docker.com/img", `
docker.com/img        push         https://registry.docker.com
docker.com/img        pull         https://registry.docker.com
docker.com/img        index        https://search.docker.com
`)

	assertResolution(t, resolver, "docker.com/other", `
docker.com/other   ca           https://trust.docker.com/cert.crt
docker.com/other   namespace       docker.com/
docker.com/        push         https://registry.base.docker.com
docker.com/        pull         https://registry.base.docker.com
docker.com/        index        https://search.base.docker.com
`)

	assertResolution(t, resolver, "docker.com/other/sub", `
docker.com/other/sub   index    https://search.sub.docker.com
docker.com/other/sub   namespace   docker.com/other
docker.com/other   ca           https://trust.docker.com/cert.crt
docker.com/other   namespace       docker.com/
docker.com/        push         https://registry.base.docker.com
docker.com/        pull         https://registry.base.docker.com
docker.com/        index        https://search.base.docker.com
`)

	assertResolution(t, resolver, "docker.com/img/sub", `
docker.com/img/sub  pull       https://mirror.base.docker.com
docker.com/img/sub  namespace       docker.com/img
docker.com/img        push         https://registry.docker.com
docker.com/img        pull         https://registry.docker.com
docker.com/img        index        https://search.docker.com
`)

	assertResolution(t, resolver, "docker.com/other/block", `
docker.com/other/block   namespace
`)
}

func mustEntry(s string) Entry {
	entry, err := parseEntry(s)
	if err != nil {
		panic(err)
	}
	return entry
}

func mustEntries(s string) *Entries {
	entries, err := ReadEntries(strings.NewReader(s))
	if err != nil {
		panic(err)
	}

	return entries
}
