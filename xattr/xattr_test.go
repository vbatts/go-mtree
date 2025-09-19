//go:build linux
// +build linux

package xattr

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXattr(t *testing.T) {
	testDir, present := os.LookupEnv("MTREE_TESTDIR")
	if present == false {
		testDir = "."
	}
	fh, err := os.CreateTemp(testDir, "xattr.")
	require.NoError(t, err)

	path := fh.Name()
	defer os.Remove(path)

	require.NoError(t, fh.Close())

	expected := []byte("1234")
	err = Set(path, "user.testing", expected)
	require.NoErrorf(t, err, "set user.testing xattr %s", path)

	l, err := List(path)
	require.NoErrorf(t, err, "list xattr %s", path)
	assert.NotEmptyf(t, l, "expected at least one xattr in list for %s", path)

	got, err := Get(path, "user.testing")
	require.NoErrorf(t, err, "get user.testing xattr %s", path)
	assert.Equalf(t, expected, got, "user.testing xattr %s", path)
}
