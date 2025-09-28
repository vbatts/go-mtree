package mtree

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var mockTime = time.Unix(1337888823, 0)

// Here be some dodgy testing. In particular, we have to mess around with some
// of the FsEval functions. In particular, we change all of the FileInfos to a
// different value.

type mockFileInfo struct {
	os.FileInfo
}

func (fi mockFileInfo) Mode() os.FileMode {
	return os.FileMode(fi.FileInfo.Mode() | 0777)
}

func (fi mockFileInfo) ModTime() time.Time {
	return mockTime
}

type MockFsEval struct {
	open, lstat, readdir, keywordFunc int
}

// Open must have the same semantics as os.Open.
func (fs *MockFsEval) Open(path string) (*os.File, error) {
	fs.open++
	return os.Open(path)
}

// Lstat must have the same semantics as os.Lstat.
func (fs *MockFsEval) Lstat(path string) (os.FileInfo, error) {
	fs.lstat++
	fi, err := os.Lstat(path)
	return mockFileInfo{fi}, err
}

// Readdir must have the same semantics as calling os.Open on the given
// path and then returning the result of (*os.File).Readdir(-1).
func (fs *MockFsEval) Readdir(path string) ([]os.FileInfo, error) {
	fs.readdir++
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	fis, err := fh.Readdir(-1)
	if err != nil {
		return nil, err
	}
	for idx := range fis {
		fis[idx] = mockFileInfo{fis[idx]}
	}
	return fis, nil
}

// KeywordFunc must return a wrapper around the provided function (in other
// words, the returned function must refer to the same keyword).
func (fs *MockFsEval) KeywordFunc(fn KeywordFunc) KeywordFunc {
	fs.keywordFunc++
	return fn
}

//gocyclo:ignore
func TestCheckFsEval(t *testing.T) {
	dir := t.TempDir()

	content := []byte("If you hide your ignorance, no one will hit you and you'll never learn.")
	tmpfn := filepath.Join(dir, "tmpfile")
	require.NoError(t, os.WriteFile(tmpfn, content, 0451))

	// Walk this tempdir
	mock := &MockFsEval{}
	dh, err := Walk(dir, nil, append(DefaultKeywords, "sha1"), mock)
	require.NoErrorf(t, err, "walk %s (mock FsEval)", dir)

	// Make sure that mock functions have been called.
	assert.NotZero(t, mock.open, "mock.Open not called")
	assert.NotZero(t, mock.lstat, "mock.Lstat not called")
	assert.NotZero(t, mock.readdir, "mock.Readdir not called")
	assert.NotZero(t, mock.keywordFunc, "mock.KeywordFunc not called")

	// Check for sanity. This ought to pass.
	mock = &MockFsEval{}
	res, err := Check(dir, dh, nil, mock)
	require.NoErrorf(t, err, "check %s (mock FsEval)", dir)
	if !assert.Empty(t, res) {
		pprintInodeDeltas(t, res)
	}
	// Make sure that mock functions have been called.
	assert.NotZero(t, mock.open, "mock.Open not called")
	assert.NotZero(t, mock.lstat, "mock.Lstat not called")
	assert.NotZero(t, mock.readdir, "mock.Readdir not called")
	assert.NotZero(t, mock.keywordFunc, "mock.KeywordFunc not called")

	// This should FAIL.
	res, err = Check(dir, dh, nil, nil)
	require.NoErrorf(t, err, "walk %s", dir)
	if !assert.NotEmpty(t, res) {
		pprintInodeDeltas(t, res)
	}

	// Modify the metadata so you can get the right output.
	require.NoError(t, os.Chmod(tmpfn, 0777))
	require.NoError(t, os.Chtimes(tmpfn, mockTime, mockTime))
	require.NoError(t, os.Chmod(dir, 0777))
	require.NoError(t, os.Chtimes(dir, mockTime, mockTime))

	// It should now succeed.
	res, err = Check(dir, dh, nil, nil)
	require.NoErrorf(t, err, "check %s", dir)
	if !assert.Empty(t, res) {
		pprintInodeDeltas(t, res)
	}
}
