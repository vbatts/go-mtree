package mtree

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalk(t *testing.T) {
	walkDh, err := Walk(".", nil, append(DefaultKeywords, "sha1"), nil)
	require.NoError(t, err, "walk .")
	walkEntries := countTypes(walkDh)

	fh, err := os.CreateTemp(t.TempDir(), "walk.")
	require.NoError(t, err)
	defer fh.Close()

	n, err := walkDh.WriteTo(fh)
	require.NoError(t, err, "write directory hierarchy representation")
	assert.NotZero(t, n, "output mtree spec should be non-empty")

	_, err = fh.Seek(0, 0)
	require.NoError(t, err)

	specDh, err := ParseSpec(fh)
	require.NoErrorf(t, err, "parse spec %s", fh.Name())

	specEntries := countTypes(specDh)

	assert.Equal(t, walkEntries, specEntries, "round-trip specfile should have the same set of entries")
}

func TestWalkDirectory(t *testing.T) {
	dh, err := Walk(".", []ExcludeFunc{ExcludeNonDirectories}, []Keyword{"type"}, nil)
	require.NoError(t, err, "walk . (ExcludeNonDirectories)")

	for i := range dh.Entries {
		for _, keyval := range dh.Entries[i].AllKeys() {
			if dh.Entries[i].Type == FullType || dh.Entries[i].Type == RelativeType {
				if keyval.Keyword() == "type" && keyval.Value() != "dir" {
					t.Errorf("expected only directories, but %q is a %q", dh.Entries[i].Name, keyval.Value())
				}
			}
		}
	}
}
