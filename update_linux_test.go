package mtree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vbatts/go-mtree/xattr"
)

func init() {
	//logrus.SetLevel(logrus.DebugLevel)
}

//gocyclo:ignore
func TestXattrUpdate(t *testing.T) {
	content := []byte("I know half of you half as well as I ought to")
	// a bit dirty to create/destroy a directory in cwd, but often /tmp is
	// mounted tmpfs and doesn't support xattrs
	dir, err := os.MkdirTemp(".", "test.xattr.restore.")
	require.NoError(t, err)
	defer os.RemoveAll(dir) // clean up

	tmpfn := filepath.Join(dir, "tmpfile")
	require.NoError(t, os.WriteFile(tmpfn, content, 0666))

	if err := xattr.Set(dir, "user.test", []byte("directory")); err != nil {
		t.Skipf("skipping: %q does not support xattrs", dir)
	}
	require.NoError(t, xattr.Set(tmpfn, "user.test", []byte("regular file")))

	// Walk this tempdir
	dh, err := Walk(dir, nil, append(DefaultKeywords, []Keyword{"xattr", "sha1"}...), nil)
	require.NoErrorf(t, err, "walk %s", dir)

	// Now check that we're sane
	res, err := Check(dir, dh, nil, nil)
	require.NoErrorf(t, err, "check %s", dir)
	if !assert.Empty(t, res) {
		pprintInodeDeltas(t, res)
	}

	require.NoError(t, xattr.Set(tmpfn, "user.test", []byte("let it fly")))

	// Now check that we fail the check
	res, err = Check(dir, dh, nil, nil)
	require.NoErrorf(t, err, "check %s", dir)
	assert.NotEmpty(t, res, "should see xattr deltas from check")

	// restore the xattrs to original
	res, err = Update(dir, dh, append(DefaultUpdateKeywords, "xattr"), nil)
	require.NoErrorf(t, err, "update %s", dir)
	if !assert.Empty(t, res, "update implied check") {
		pprintInodeDeltas(t, res)
	}

	// Now check that we're sane again
	res, err = Check(dir, dh, nil, nil)
	require.NoErrorf(t, err, "update %s", dir)
	if !assert.Empty(t, res, "post-update check") {
		pprintInodeDeltas(t, res)
	}

	// TODO make a test for xattr here. Likely in the user space for privileges. Even still this may be prone to error for some tmpfs don't act right with xattrs. :-\
	// I'd hate to have to t.Skip() a test rather than fail altogether.
}
