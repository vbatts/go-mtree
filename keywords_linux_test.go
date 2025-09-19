//go:build linux
// +build linux

package mtree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vbatts/go-mtree/xattr"
)

//gocyclo:ignore
func TestXattr(t *testing.T) {
	testDir, present := os.LookupEnv("MTREE_TESTDIR")
	if present == false {
		// a bit dirty to create/destroy a directory in cwd,
		// but often /tmp is mounted tmpfs and doesn't support
		// xattrs
		testDir = "."
	}
	dir, err := os.MkdirTemp(testDir, "test.xattrs.")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	fh, err := os.Create(filepath.Join(dir, "file"))
	require.NoError(t, err)

	_, err = fh.WriteString("howdy")
	require.NoError(t, err)

	err = fh.Sync()
	require.NoError(t, err)

	_, err = fh.Seek(0, 0)
	require.NoError(t, err)

	require.NoError(t, os.Symlink("./no/such/path", filepath.Join(dir, "symlink")))

	if err := xattr.Set(dir, "user.test", []byte("directory")); err != nil {
		t.Skipf("skipping: %q does not support xattrs", dir)
	}
	require.NoError(t, xattr.Set(filepath.Join(dir, "file"), "user.test", []byte("regular file")))

	dirstat, err := os.Lstat(dir)
	require.NoError(t, err)

	// Check the directory
	kvs, err := xattrKeywordFunc(dir, dirstat, nil)
	require.NoError(t, err, "xattr keyword fn")
	assert.NotEmpty(t, kvs, "expected to get a keyval from xattr keyword fn")

	filestat, err := fh.Stat()
	require.NoError(t, err)

	// Check the regular file
	kvs, err = xattrKeywordFunc(filepath.Join(dir, "file"), filestat, fh)
	require.NoError(t, err, "xattr keyword fn")
	assert.NotEmpty(t, kvs, "expected to get a keyval from xattr keyword fn")

	linkstat, err := os.Lstat(filepath.Join(dir, "symlink"))
	require.NoError(t, err)

	// Check a broken symlink
	_, err = xattrKeywordFunc(filepath.Join(dir, "symlink"), linkstat, nil)
	require.NoError(t, err, "xattr keyword fn broken symlink")
}
