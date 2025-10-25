package mtree

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pprintInodeDeltas(t *testing.T, deltas []InodeDelta) {
	for idx, delta := range deltas {
		var str string
		if buf, err := json.MarshalIndent(delta, "", "  "); err == nil {
			str = string(buf)
		} else {
			str = delta.String()
		}
		t.Logf("diff[%d] = %s", idx, str)
	}
}

// simple walk of current directory, and immediately check it.
// may not be parallelizable.
func TestCompare(t *testing.T) {
	old, err := Walk(".", nil, append(DefaultKeywords, "sha1"), nil)
	require.NoError(t, err, "walk .")

	new, err := Walk(".", nil, append(DefaultKeywords, "sha1"), nil)
	require.NoError(t, err, "walk .")

	res, err := Compare(old, new, nil)
	require.NoError(t, err, "compare")

	if !assert.Empty(t, res, "compare after no changes should have no diff") {
		pprintInodeDeltas(t, res)
	}
}

//gocyclo:ignore
func TestCompareModified(t *testing.T) {
	dir := t.TempDir()

	// Create a bunch of objects.
	tmpfile := filepath.Join(dir, "tmpfile")
	require.NoError(t, os.WriteFile(tmpfile, []byte("some content here"), 0666))

	tmpdir := filepath.Join(dir, "testdir")
	require.NoError(t, os.Mkdir(tmpdir, 0755))

	tmpsubfile := filepath.Join(tmpdir, "anotherfile")
	require.NoError(t, os.WriteFile(tmpsubfile, []byte("some different content"), 0666))

	// Walk the current state.
	old, err := Walk(dir, nil, append(DefaultKeywords, "sha1"), nil)
	require.NoErrorf(t, err, "walk %s", dir)

	// Overwrite the content in one of the files.
	require.NoError(t, os.WriteFile(tmpsubfile, []byte("modified content"), 0666))

	// Walk the new state.
	new, err := Walk(dir, nil, append(DefaultKeywords, "sha1"), nil)
	require.NoErrorf(t, err, "walk %s", dir)

	// Compare.
	diffs, err := Compare(old, new, nil)
	require.NoError(t, err, "compare")

	// 1 object
	if !assert.Len(t, diffs, 1, "unexpected diff count") {
		pprintInodeDeltas(t, diffs)
	}

	// These cannot fail.
	tmpsubfile, _ = filepath.Rel(dir, tmpsubfile)
	for _, diff := range diffs {
		if assert.Equal(t, tmpsubfile, diff.Path()) {
			assert.Equalf(t, Modified, diff.Type(), "unexpected diff type for %s", diff.Path())
			assert.NotNil(t, diff.Diff(), "Diff for modified diff")
			assert.NotNil(t, diff.Old(), "Old for modified diff")
			assert.NotNil(t, diff.New(), "New for modified diff")
		}
	}
}

//gocyclo:ignore
func TestCompareMissing(t *testing.T) {
	dir := t.TempDir()

	// Create a bunch of objects.
	tmpfile := filepath.Join(dir, "tmpfile")
	require.NoError(t, os.WriteFile(tmpfile, []byte("some content here"), 0666))

	tmpdir := filepath.Join(dir, "testdir")
	require.NoError(t, os.Mkdir(tmpdir, 0755))

	tmpsubfile := filepath.Join(tmpdir, "anotherfile")
	require.NoError(t, os.WriteFile(tmpsubfile, []byte("some different content"), 0666))

	// Walk the current state.
	old, err := Walk(dir, nil, append(DefaultKeywords, "sha1"), nil)
	require.NoErrorf(t, err, "walk %s", dir)

	// Delete the objects.
	require.NoError(t, os.RemoveAll(tmpfile))
	require.NoError(t, os.RemoveAll(tmpsubfile))
	require.NoError(t, os.RemoveAll(tmpdir))

	// Walk the new state.
	new, err := Walk(dir, nil, append(DefaultKeywords, "sha1"), nil)
	require.NoErrorf(t, err, "walk %s", dir)

	// Compare.
	diffs, err := Compare(old, new, nil)
	require.NoError(t, err, "compare")

	// 3 objects + the changes to '.'
	if !assert.Len(t, diffs, 4, "unexpected diff count") {
		pprintInodeDeltas(t, diffs)
	}

	// These cannot fail.
	tmpfile, _ = filepath.Rel(dir, tmpfile)
	tmpdir, _ = filepath.Rel(dir, tmpdir)
	tmpsubfile, _ = filepath.Rel(dir, tmpsubfile)

	for _, diff := range diffs {
		switch diff.Path() {
		case ".":
			// ignore these changes
		case tmpfile, tmpdir, tmpsubfile:
			assert.Equalf(t, Missing, diff.Type(), "unexpected diff type for %s", diff.Path())
			assert.Nil(t, diff.Diff(), "Diff for missing diff")
			assert.NotNil(t, diff.Old(), "Old for missing diff")
			assert.Nil(t, diff.New(), "New for missing diff")
		default:
			t.Errorf("unexpected diff found: %#v", diff)
		}
	}
}

//gocyclo:ignore
func TestCompareExtra(t *testing.T) {
	dir := t.TempDir()

	// Walk the current state.
	old, err := Walk(dir, nil, append(DefaultKeywords, "sha1"), nil)
	require.NoErrorf(t, err, "walk %s", dir)

	// Create a bunch of objects.
	tmpfile := filepath.Join(dir, "tmpfile")
	require.NoError(t, os.WriteFile(tmpfile, []byte("some content here"), 0666))

	tmpdir := filepath.Join(dir, "testdir")
	require.NoError(t, os.Mkdir(tmpdir, 0755))

	tmpsubfile := filepath.Join(tmpdir, "anotherfile")
	require.NoError(t, os.WriteFile(tmpsubfile, []byte("some different content"), 0666))

	// Walk the new state.
	new, err := Walk(dir, nil, append(DefaultKeywords, "sha1"), nil)
	require.NoErrorf(t, err, "walk %s", dir)

	// Compare.
	diffs, err := Compare(old, new, nil)
	require.NoError(t, err, "compare")

	// 3 objects + the changes to '.'
	if !assert.Len(t, diffs, 4, "unexpected diff count") {
		pprintInodeDeltas(t, diffs)
	}

	// These cannot fail.
	tmpfile, _ = filepath.Rel(dir, tmpfile)
	tmpdir, _ = filepath.Rel(dir, tmpdir)
	tmpsubfile, _ = filepath.Rel(dir, tmpsubfile)

	for _, diff := range diffs {
		switch diff.Path() {
		case ".":
			// ignore these changes
		case tmpfile, tmpdir, tmpsubfile:
			assert.Equalf(t, Extra, diff.Type(), "unexpected diff type for %s", diff.Path())
			assert.Nil(t, diff.Diff(), "Diff for extra diff")
			assert.Nil(t, diff.Old(), "Old for extra diff")
			assert.NotNil(t, diff.New(), "New for extra diff")
		default:
			t.Errorf("unexpected diff found: %#v", diff)
		}
	}
}

func TestCompareKeySubset(t *testing.T) {
	dir := t.TempDir()

	// Create a bunch of objects.
	tmpfile := filepath.Join(dir, "tmpfile")
	require.NoError(t, os.WriteFile(tmpfile, []byte("some content here"), 0666))

	tmpdir := filepath.Join(dir, "testdir")
	require.NoError(t, os.Mkdir(tmpdir, 0755))

	tmpsubfile := filepath.Join(tmpdir, "anotherfile")
	require.NoError(t, os.WriteFile(tmpsubfile, []byte("aaa"), 0666))

	// Walk the current state.
	old, err := Walk(dir, nil, append(DefaultKeywords, "sha1"), nil)
	require.NoErrorf(t, err, "walk %s", dir)

	// Overwrite the content in one of the files, but without changing the size.
	require.NoError(t, os.WriteFile(tmpsubfile, []byte("bbb"), 0666))

	// Walk the new state.
	new, err := Walk(dir, nil, append(DefaultKeywords, "sha1"), nil)
	require.NoErrorf(t, err, "walk %s", dir)

	// Compare.
	diffs, err := Compare(old, new, []Keyword{"size"})
	require.NoError(t, err, "compare")

	// 0 objects
	if !assert.Empty(t, diffs, "size-only compare should not return any entries") {
		pprintInodeDeltas(t, diffs)
	}
}

func TestCompareKeyDelta(t *testing.T) {
	dir := t.TempDir()

	// Create a bunch of objects.
	tmpfile := filepath.Join(dir, "tmpfile")
	require.NoError(t, os.WriteFile(tmpfile, []byte("some content here"), 0666))

	tmpdir := filepath.Join(dir, "testdir")
	require.NoError(t, os.Mkdir(tmpdir, 0755))

	tmpsubfile := filepath.Join(tmpdir, "anotherfile")
	require.NoError(t, os.WriteFile(tmpsubfile, []byte("aaa"), 0666))

	// Walk the current state.
	manifestKeywords := append(DefaultKeywords[:], "sha1digest")
	old, err := Walk(dir, nil, manifestKeywords, nil)
	require.NoErrorf(t, err, "walk %s", dir)

	t.Run("Extra-Key", func(t *testing.T) {
		extraKeyword := Keyword("sha256digest")
		newManifestKeywords := append(manifestKeywords[:], extraKeyword)

		new, err := Walk(dir, nil, newManifestKeywords, nil)
		require.NoErrorf(t, err, "walk %s", dir)

		diffs, err := Compare(old, new, nil)
		require.NoError(t, err, "compare")

		assert.NotEmpty(t, diffs, "extra keys in manifest should result in deltas")
		for _, diff := range diffs {
			if assert.Equal(t, Modified, diff.Type(), "extra keyword diff element should be 'modified'") {
				kds := diff.Diff()
				if assert.Len(t, kds, 1, "should only get a single key delta") {
					kd := kds[0]
					assert.Equalf(t, Extra, kd.Type(), "key %q", kd.Name())
					assert.Equal(t, extraKeyword, kd.Name())
					assert.Nil(t, kd.Old(), "Old for extra keyword delta")
					assert.NotNil(t, kd.New(), "New for extra keyword delta")
				}
			}
		}
	})

	t.Run("Missing-Key", func(t *testing.T) {
		missingKeyword := Keyword("sha1digest")
		newManifestKeywords := slices.DeleteFunc(manifestKeywords[:], func(kw Keyword) bool {
			return kw == missingKeyword
		})

		new, err := Walk(dir, nil, newManifestKeywords, nil)
		require.NoErrorf(t, err, "walk %s", dir)

		diffs, err := Compare(old, new, nil)
		require.NoError(t, err, "compare")

		assert.NotEmpty(t, diffs, "missing keys in manifest should result in deltas")
		for _, diff := range diffs {
			if assert.Equal(t, Modified, diff.Type(), "missing keyword diff element should be 'modified'") {
				kds := diff.Diff()
				if assert.Len(t, kds, 1, "should only get a single key delta") {
					kd := kds[0]
					assert.Equalf(t, Missing, kd.Type(), "key %q", kd.Name())
					assert.Equal(t, missingKeyword, kd.Name())
					assert.NotNil(t, kd.Old(), "Old for missing keyword delta")
					assert.Nil(t, kd.New(), "New for missing keyword delta")
				}
			}
		}
	})
}

//gocyclo:ignore
func TestTarCompare(t *testing.T) {
	dir := t.TempDir()

	// Create a bunch of objects.
	tmpfile := filepath.Join(dir, "tmpfile")
	require.NoError(t, os.WriteFile(tmpfile, []byte("some content"), 0644))

	tmpdir := filepath.Join(dir, "testdir")
	require.NoError(t, os.Mkdir(tmpdir, 0755))

	tmpsubfile := filepath.Join(tmpdir, "anotherfile")
	require.NoError(t, os.WriteFile(tmpsubfile, []byte("aaa"), 0644))

	// Create a tar-like archive.
	compareFiles := []fakeFile{
		{"./", "", 0700, tar.TypeDir, 100, 0, nil},
		{"tmpfile", "some content", 0644, tar.TypeReg, 100, 0, nil},
		{"testdir/", "", 0755, tar.TypeDir, 100, 0, nil},
		{"testdir/anotherfile", "aaa", 0644, tar.TypeReg, 100, 0, nil},
	}

	for _, file := range compareFiles {
		path := filepath.Join(dir, file.Name)

		// Change the time to something known with nanosec != 0.
		chtime := time.Unix(file.Sec, 987654321)
		require.NoError(t, os.Chtimes(path, chtime, chtime))
	}

	// Walk the current state.
	old, err := Walk(dir, nil, append(DefaultKeywords, "sha1"), nil)
	require.NoErrorf(t, err, "walk %s", dir)

	ts, err := makeTarStream(compareFiles)
	require.NoError(t, err, "make tar stream")

	str := NewTarStreamer(bytes.NewBuffer(ts), nil, append(DefaultTarKeywords, "sha1"))

	n, err := io.Copy(io.Discard, str)
	require.NoError(t, err, "read full tar stream")
	require.Greater(t, n, int64(0), "tar stream should be non-empty")
	require.NoError(t, str.Close(), "close tar stream")

	new, err := str.Hierarchy()
	require.NoError(t, err, "TarStreamer Hierarchy")
	require.NotNil(t, new, "TarStreamer Hierarchy")

	// Compare.
	diffs, err := Compare(old, new, append(DefaultTarKeywords, "sha1"))
	require.NoError(t, err, "compare")

	// 0 objects, but there are bugs in tar generation.
	if len(diffs) > 0 {
		actualFailure := false
		for i, delta := range diffs {
			// XXX: Tar generation is slightly broken, so we need to ignore some bugs.
			if delta.Path() == "." && delta.Type() == Modified {
				// FIXME: This is a known bug.
				t.Logf("'.' is different in the tar -- this is a bug in the tar generation")

				// The tar generation bug means that '.' is missing a bunch of keys.
				allMissing := true
				for _, keyDelta := range delta.Diff() {
					if keyDelta.Type() != Missing {
						allMissing = false
					}
				}
				if !allMissing {
					t.Errorf("'.' has changed in a way not consistent with known bugs")
				}

				continue
			}

			// XXX: Another bug.
			keys := delta.Diff()
			if len(keys) == 1 && keys[0].Name() == "size" && keys[0].Type() == Missing {
				// FIXME: Also a known bug with tar generation dropping size=.
				t.Logf("'%s' is missing a size= keyword -- a bug in tar generation", delta.Path())

				continue
			}

			actualFailure = true
			buf, err := json.MarshalIndent(delta, "", "  ")
			if err == nil {
				t.Logf("FAILURE: diff[%d] = %s", i, string(buf))
			} else {
				t.Logf("FAILURE: diff[%d] = %s", i, delta)
			}
		}

		if actualFailure {
			t.Errorf("expected the diff length to be 0, got %d", len(diffs))
		}
	}
}
