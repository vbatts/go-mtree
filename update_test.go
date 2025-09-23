//go:build go1.7
// +build go1.7

package mtree

import (
	"container/heap"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

//gocyclo:ignore
func TestUpdate(t *testing.T) {
	content := []byte("I know half of you half as well as I ought to")
	dir := t.TempDir()

	tmpfn := filepath.Join(dir, "tmpfile")
	require.NoError(t, os.WriteFile(tmpfn, content, 0666))

	// Walk this tempdir
	dh, err := Walk(dir, nil, append(DefaultKeywords, "sha1"), nil)
	require.NoErrorf(t, err, "walk %s", dir)

	// Touch a file, so the mtime changes.
	now := time.Now()
	require.NoError(t, os.Chtimes(tmpfn, now, now))
	require.NoError(t, os.Chmod(tmpfn, os.FileMode(0600)))

	// Changing user is a little tough, but the group can be changed by a limited user to any group that the user is a member of. So just choose one that is not the current main group.
	u, err := user.Current()
	require.NoError(t, err, "get current user")

	ugroups, err := u.GroupIds()
	require.NoError(t, err, "get current user groups")
	for _, ugroup := range ugroups {
		if ugroup == u.Gid {
			continue
		}
		gid, err := strconv.Atoi(ugroup)
		require.NoErrorf(t, err, "parse group %q", ugroup)
		require.NoError(t, os.Lchown(tmpfn, -1, gid))
	}

	// Check for sanity. This ought to have failures
	res, err := Check(dir, dh, nil, nil)
	require.NoErrorf(t, err, "check %s", dir)
	assert.NotEmpty(t, res, "should see mtime/chown/chtimes deltas from check")
	//dh.WriteTo(os.Stdout)

	res, err = Update(dir, dh, DefaultUpdateKeywords, nil)
	require.NoErrorf(t, err, "update %d", err)
	if !assert.Empty(t, res, "update implied check") {
		pprintInodeDeltas(t, res)
	}

	// Now check that we're sane again
	res, err = Check(dir, dh, nil, nil)
	require.NoErrorf(t, err, "update %s", dir)
	if !assert.Empty(t, res, "post-update check") {
		pprintInodeDeltas(t, res)
	}
}

func TestPathUpdateHeap(t *testing.T) {
	h := &pathUpdateHeap{
		pathUpdate{Path: "not/the/longest"},
		pathUpdate{Path: "almost/the/longest"},
		pathUpdate{Path: "."},
		pathUpdate{Path: "short"},
	}
	heap.Init(h)
	v := "this/is/one/is/def/the/longest"
	heap.Push(h, pathUpdate{Path: v})

	longest := len(v)
	var p string
	for h.Len() > 0 {
		p = heap.Pop(h).(pathUpdate).Path
		assert.LessOrEqual(t, len(p), longest, "expected next path %q to be shorter", p)
	}
	assert.Equal(t, ".", p, ". should be the last path")
}
