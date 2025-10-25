package mtree

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// simple walk of current directory, and immediately check it.
// may not be parallelizable.
func TestCheck(t *testing.T) {
	dh, err := Walk(".", nil, append(DefaultKeywords, []Keyword{"sha1", "xattr"}...), nil)
	require.NoError(t, err, "walk .")

	res, err := Check(".", dh, nil, nil)
	require.NoError(t, err, "check .")

	if !assert.Empty(t, res, "check after no changes should have no diff") {
		pprintInodeDeltas(t, res)
	}
}

// make a directory, walk it, check it, modify the timestamp and ensure it fails.
// only check again for size and sha1, and ignore time, and ensure it passes
func TestCheckKeywords(t *testing.T) {
	content := []byte("I know half of you half as well as I ought to")
	dir := t.TempDir()

	tmpfn := filepath.Join(dir, "tmpfile")
	require.NoError(t, os.WriteFile(tmpfn, content, 0666))

	// Walk this tempdir
	dh, err := Walk(dir, nil, append(DefaultKeywords, "sha1"), nil)
	require.NoErrorf(t, err, "walk %s", dir)

	// Check for sanity. This ought to pass.
	res, err := Check(dir, dh, nil, nil)
	require.NoErrorf(t, err, "check %s", dir)
	if !assert.Empty(t, res, "check after no changes should have no diff") {
		pprintInodeDeltas(t, res)
	}

	// Touch a file, so the mtime changes.
	newtime := time.Date(2006, time.February, 1, 3, 4, 5, 0, time.UTC)
	require.NoError(t, os.Chtimes(tmpfn, newtime, newtime))

	// Check again. This ought to fail.
	res, err = Check(dir, dh, nil, nil)
	require.NoErrorf(t, err, "check %s", dir)

	if assert.NotEmpty(t, res, "should get a delta after mtime change") {
		assert.Len(t, res, 1, "should only be one changed file entry")
		diff := res[0]
		assert.Equal(t, Modified, diff.Type(), "expected to get modified entry")
		kds := diff.Diff()
		if assert.Len(t, kds, 1, "should have one key different after mtime change") {
			kd := kds[0]
			assert.Equal(t, Modified, kd.Type(), "after mtime change key delta type should be modified")
			assert.Equal(t, Keyword("time"), kd.Name(), "after mtime change key delta should be 'time'")
			assert.NotNil(t, kd.Old(), "after mtime change key delta Old")
			assert.NotNil(t, kd.New(), "after mtime change key delta New")
		}
	}

	// Check again, but only sha1 and mode. This ought to pass.
	res, err = Check(dir, dh, []Keyword{"sha1", "mode"}, nil)
	require.NoErrorf(t, err, "check .", err)
	if !assert.Empty(t, res, "check (~time) should have no diff") {
		pprintInodeDeltas(t, res)
	}
}

func ExampleCheck() {
	dh, err := Walk(".", nil, append(DefaultKeywords, "sha1"), nil)
	if err != nil {
		// handle error ...
	}

	res, err := Check(".", dh, nil, nil)
	if err != nil {
		// handle error ...
	}
	if len(res) > 0 {
		// handle failed validity ...
	}
}

// Tests default action for evaluating a symlink, which is just to compare the
// link itself, not to follow it
func TestDefaultBrokenLink(t *testing.T) {
	dir := "./testdata/dirwithbrokenlink"

	dh, err := Walk(dir, nil, append(DefaultKeywords, "sha1"), nil)
	require.NoErrorf(t, err, "walk %s", dir)

	res, err := Check(dir, dh, nil, nil)
	require.NoErrorf(t, err, "check %s", dir)

	if !assert.Empty(t, res, "check after no changes should have no diff") {
		pprintInodeDeltas(t, res)
	}
}

// https://github.com/vbatts/go-mtree/issues/8
func TestTimeComparison(t *testing.T) {
	dir := t.TempDir()

	// This is the format of time from FreeBSD
	spec := `
/set type=file time=5.000000000
.               type=dir
    file       time=5.000000000
..
`

	fh, err := os.Create(filepath.Join(dir, "file"))
	require.NoError(t, err)

	// This is what mode we're checking for. Round integer of epoch seconds
	epoch := time.Unix(5, 0)
	require.NoError(t, os.Chtimes(fh.Name(), epoch, epoch))
	require.NoError(t, os.Chtimes(dir, epoch, epoch))
	require.NoError(t, fh.Close())

	dh, err := ParseSpec(bytes.NewBufferString(spec))
	require.NoError(t, err, "parse specfile")

	res, err := Check(dir, dh, nil, nil)
	require.NoErrorf(t, err, "check %s against spec", dir)
	if !assert.Empty(t, res) {
		pprintInodeDeltas(t, res)
	}
}

func TestTarTime(t *testing.T) {
	dir := t.TempDir()

	// This is the format of time from FreeBSD
	spec := `
/set type=file time=5.454353132
.               type=dir time=5.123456789
    file       time=5.911134111
..
`

	fh, err := os.Create(filepath.Join(dir, "file"))
	require.NoError(t, err)

	// This is what mode we're checking for. Round integer of epoch seconds
	epoch := time.Unix(5, 0)
	require.NoError(t, os.Chtimes(fh.Name(), epoch, epoch))
	require.NoError(t, os.Chtimes(dir, epoch, epoch))
	require.NoError(t, fh.Close())

	dh, err := ParseSpec(bytes.NewBufferString(spec))
	require.NoError(t, err, "parse specfile")

	keywords := dh.UsedKeywords()
	assert.ElementsMatch(t, keywords, []Keyword{"type", "time"}, "UsedKeywords")

	// make sure "time" keyword works
	res1, err := Check(dir, dh, keywords, nil)
	require.NoErrorf(t, err, "check %s (UsedKeywords)", dir)
	assert.NotEmpty(t, res1, "check should have errors when time mismatched")

	// make sure tar_time wins
	res2, err := Check(dir, dh, append(keywords, "tar_time"), nil)
	require.NoErrorf(t, err, "check %s (UsedKeywords + tar_time)", dir)
	if !assert.Empty(t, res2, "tar_time should check against truncated timestamp") {
		pprintInodeDeltas(t, res2)
	}
}

func TestIgnoreComments(t *testing.T) {
	dir := t.TempDir()

	// This is the format of time from FreeBSD
	spec := `
/set type=file time=5.000000000
.               type=dir
    file1       time=5.000000000
..
`

	fh, err := os.Create(filepath.Join(dir, "file1"))
	require.NoError(t, err)
	// This is what mode we're checking for. Round integer of epoch seconds
	epoch := time.Unix(5, 0)
	require.NoError(t, os.Chtimes(fh.Name(), epoch, epoch))
	require.NoError(t, os.Chtimes(dir, epoch, epoch))
	require.NoError(t, fh.Close())

	dh, err := ParseSpec(bytes.NewBufferString(spec))
	require.NoError(t, err, "parse specfile")

	res, err := Check(dir, dh, nil, nil)
	require.NoErrorf(t, err, "check %s", dir)
	if !assert.Empty(t, res) {
		pprintInodeDeltas(t, res)
	}

	// now change the spec to a comment that looks like an actual Entry but has
	// whitespace in front of it
	spec = `
/set type=file time=5.000000000
.               type=dir
    file1       time=5.000000000
	#file2 		time=5.000000000
..
`
	dh, err = ParseSpec(bytes.NewBufferString(spec))
	require.NoError(t, err, "parse specfile")

	res, err = Check(dir, dh, nil, nil)
	require.NoErrorf(t, err, "check %s", dir)
	if !assert.Empty(t, res) {
		pprintInodeDeltas(t, res)
	}
}

func TestCheckNeedsEncoding(t *testing.T) {
	dir := t.TempDir()

	fh, err := os.Create(filepath.Join(dir, "file[ "))
	require.NoError(t, err)
	require.NoError(t, fh.Close())

	fh, err = os.Create(filepath.Join(dir, "    , should work"))
	require.NoError(t, err)
	require.NoError(t, fh.Close())

	dh, err := Walk(dir, nil, DefaultKeywords, nil)
	require.NoErrorf(t, err, "walk %s", dir)

	res, err := Check(dir, dh, nil, nil)
	require.NoErrorf(t, err, "check %s", dir)
	if !assert.Empty(t, res) {
		pprintInodeDeltas(t, res)
	}
}
