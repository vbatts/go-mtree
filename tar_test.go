package mtree

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleStreamer() {
	fh, err := os.Open("./testdata/test.tar")
	if err != nil {
		// handle error ...
	}
	str := NewTarStreamer(fh, nil, nil)
	if err := extractTar("/tmp/dir", str); err != nil {
		// handle error ...
	}

	dh, err := str.Hierarchy()
	if err != nil {
		// handle error ...
	}

	res, err := Check("/tmp/dir/", dh, nil, nil)
	if err != nil {
		// handle error ...
	}
	if len(res) > 0 {
		// handle validation issue ...
	}
}
func extractTar(root string, tr io.Reader) error {
	return nil
}

func TestTar(t *testing.T) {
	/*
		data, err := makeTarStream()
		if err != nil {
			t.Fatal(err)
		}
		buf := bytes.NewBuffer(data)
		str := NewTarStreamer(buf, append(DefaultKeywords, "sha1"))
	*/
	/*
		// open empty folder and check size.
		fh, err := os.Open("./testdata/empty")
		if err != nil {
			t.Fatal(err)
		}
		log.Println(fh.Stat())
		fh.Close() */
	fh, err := os.Open("./testdata/test.tar")
	require.NoError(t, err)
	defer fh.Close()

	str := NewTarStreamer(fh, nil, append(DefaultKeywords, "sha1"))

	n, err := io.Copy(io.Discard, str)
	require.NoError(t, err, "read full tar stream")
	require.NotZero(t, n, "tar stream should be non-empty")
	require.NoError(t, str.Close(), "close tar stream")

	// get DirectoryHierarchy struct from walking the tar archive
	tdh, err := str.Hierarchy()
	require.NoError(t, err, "TarStreamer Hierarchy")
	require.NotNil(t, tdh, "TarStreamer Hierarchy")

	testDir, present := os.LookupEnv("MTREE_TESTDIR")
	if present == false {
		testDir = "."
	}
	testPath := filepath.Join(testDir, "test.mtree")
	fh, err = os.Create(testPath)
	require.NoError(t, err)
	defer os.Remove(testPath)

	// put output of tar walk into test.mtree
	_, err = tdh.WriteTo(fh)
	require.NoError(t, err)
	require.NoError(t, fh.Close())

	// now simulate gomtree -T testdata/test.tar -f testdata/test.mtree
	fh, err = os.Open(testPath)
	require.NoError(t, err)
	defer fh.Close()

	dh, err := ParseSpec(fh)
	require.NoErrorf(t, err, "parse spec %s", fh.Name())

	res, err := Compare(tdh, dh, append(DefaultKeywords, "sha1"))
	require.NoError(t, err, "compare tar DirectoryHierarchy")
	if !assert.Empty(t, res) {
		pprintInodeDeltas(t, res)
	}
}

// This test checks how gomtree handles archives that were created
// with multiple directories, i.e, archives created with something like:
// `tar -cvf some.tar dir1 dir2 dir3 dir4/dir5 dir6` ... etc.
// The testdata of collection.tar resemble such an archive. the `collection` folder
// is the contents of `collection.tar` extracted
//
//gocyclo:ignore
func TestArchiveCreation(t *testing.T) {
	fh, err := os.Open("./testdata/collection.tar")
	require.NoError(t, err)
	defer fh.Close()

	str := NewTarStreamer(fh, nil, []Keyword{"sha1"})

	n, err := io.Copy(io.Discard, str)
	require.NoError(t, err, "read full tar stream")
	require.NotZero(t, n, "tar stream should be non-empty")
	require.NoError(t, str.Close(), "close tar stream")

	// get DirectoryHierarchy struct from walking the tar archive
	tdh, err := str.Hierarchy()
	require.NoError(t, err, "TarStreamer Hierarchy")
	require.NotNil(t, tdh, "TarStreamer Hierarchy")

	// Test the tar manifest against the actual directory
	dir := "./testdata/collection"
	res, err := Check(dir, tdh, []Keyword{"sha1"}, nil)
	require.NoErrorf(t, err, "check %s", dir)
	if !assert.Empty(t, res, "check against tar DirectoryHierarchy") {
		pprintInodeDeltas(t, res)
	}

	// Test the tar manifest against itself
	res, err = Compare(tdh, tdh, []Keyword{"sha1"})
	require.NoErrorf(t, err, "compare tar against itself")
	if !assert.Empty(t, res, "compare tar against itself") {
		pprintInodeDeltas(t, res)
	}

	// Validate the directory manifest against the archive
	dh, err := Walk(dir, nil, []Keyword{"sha1"}, nil)
	require.NoErrorf(t, err, "walk %s", dir)

	res, err = Compare(tdh, dh, []Keyword{"sha1"})
	require.NoErrorf(t, err, "compare tar against %s walk", dir)
	if !assert.Emptyf(t, res, "compare tar against %s walk", dir) {
		pprintInodeDeltas(t, res)
	}
}

// Now test a tar file that was created with just the path to a file. In this
// test case, the traversal and creation of "placeholder" directories are
// evaluated. Also, The fact that this archive contains a single entry, yet the
// entry is associated with a file that has parent directories, means that the
// "." directory should be the lowest sub-directory under which `file` is contained.
//
//gocyclo:ignore
func TestTreeTraversal(t *testing.T) {
	fh, err := os.Open("./testdata/traversal.tar")
	require.NoError(t, err)
	defer fh.Close()

	str := NewTarStreamer(fh, nil, DefaultTarKeywords)

	n, err := io.Copy(io.Discard, str)
	require.NoError(t, err, "read full tar stream")
	require.NotZero(t, n, "tar stream should be non-empty")
	require.NoError(t, str.Close(), "close tar stream")

	tdh, err := str.Hierarchy()
	require.NoError(t, err, "TarStreamer Hierarchy")
	require.NotNil(t, tdh, "TarStreamer Hierarchy")

	res, err := Compare(tdh, tdh, []Keyword{"sha1"})
	require.NoErrorf(t, err, "compare tar against itself")
	if !assert.Empty(t, res, "compare tar against itself") {
		pprintInodeDeltas(t, res)
	}

	res, err = Check("./testdata/.", tdh, []Keyword{"sha1"}, nil)
	require.NoError(t, err, "check testdata dir")

	// The top-level "." directory will contain contents of some extra files.
	// This test was originally written with the pre-Compare Check code in mind
	// (i.e., only file *modifications* were counted as errors).
	res = slices.DeleteFunc(res, func(delta InodeDelta) bool {
		skip := delta.Type() == Extra
		if skip {
			t.Logf("ignoring extra entry for %q", delta.Path())
		}
		return skip
	})
	if !assert.Emptyf(t, res, "compare %s against testdata walk", fh.Name()) {
		pprintInodeDeltas(t, res)
	}

	// Now test an archive that requires placeholder directories, i.e, there
	// are no headers in the archive that are associated with the actual
	// directory name.
	fh, err = os.Open("./testdata/singlefile.tar")
	require.NoError(t, err)
	defer fh.Close()

	str = NewTarStreamer(fh, nil, DefaultTarKeywords)

	n, err = io.Copy(io.Discard, str)
	require.NoError(t, err, "read full tar stream")
	require.NotZero(t, n, "tar stream should be non-empty")
	require.NoError(t, str.Close(), "close tar stream")

	tdh, err = str.Hierarchy()
	require.NoError(t, err, "TarStreamer Hierarchy")
	require.NotNil(t, tdh, "TarStreamer Hierarchy")

	// The top-level "." directory will contain contents of some extra files.
	// This test was originally written with the pre-Compare Check code in mind
	// (i.e., only file *modifications* were counted as errors).
	res = slices.DeleteFunc(res, func(delta InodeDelta) bool {
		skip := delta.Type() == Extra
		if skip {
			t.Logf("ignoring extra entry for %q", delta.Path())
		}
		return skip
	})
	if !assert.Emptyf(t, res, "compare %s against testdata walk", fh.Name()) {
		pprintInodeDeltas(t, res)
	}
}

func TestHardlinks(t *testing.T) {
	fh, err := os.Open("./testdata/hardlinks.tar")
	require.NoError(t, err)
	defer fh.Close()

	str := NewTarStreamer(fh, nil, append(DefaultTarKeywords, "nlink"))

	n, err := io.Copy(io.Discard, str)
	require.NoError(t, err, "read full tar stream")
	require.NotZero(t, n, "tar stream should be non-empty")
	require.NoError(t, str.Close(), "close tar stream")

	tdh, err := str.Hierarchy()
	require.NoError(t, err, "TarStreamer Hierarchy")
	require.NotNil(t, tdh, "TarStreamer Hierarchy")

	foundnlink := false
	for _, e := range tdh.Entries {
		if e.Type == RelativeType {
			for _, kv := range e.Keywords {
				if KeyVal(kv).Keyword() == "nlink" {
					foundnlink = true
					assert.Equalf(t, "3", KeyVal(kv).Value(), "expected 3 hardlinks for %s", e.Name)
				}
			}
		}
	}
	require.True(t, foundnlink, "there should be an nlink entry")
}

type fakeFile struct {
	Name, Body string
	Mode       int64
	Type       byte
	Sec, Nsec  int64
	Xattrs     map[string]string
}

func makeTarStream(ff []fakeFile) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Create a new tar archive.
	tw := tar.NewWriter(buf)

	// Add some files to the archive.
	for _, file := range ff {
		hdr := &tar.Header{
			Name:       file.Name,
			Uid:        syscall.Getuid(),
			Gid:        syscall.Getgid(),
			Mode:       file.Mode,
			Typeflag:   file.Type,
			Size:       int64(len(file.Body)),
			ModTime:    time.Unix(file.Sec, file.Nsec),
			AccessTime: time.Unix(file.Sec, file.Nsec),
			ChangeTime: time.Unix(file.Sec, file.Nsec),
			Xattrs:     file.Xattrs,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if len(file.Body) > 0 {
			if _, err := tw.Write([]byte(file.Body)); err != nil {
				return nil, err
			}
		}
	}
	// Make sure to check the error on Close.
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func TestArchiveExcludeNonDirectory(t *testing.T) {
	fh, err := os.Open("./testdata/collection.tar")
	require.NoError(t, err)
	defer fh.Close()

	str := NewTarStreamer(fh, []ExcludeFunc{ExcludeNonDirectories}, []Keyword{"type"})

	n, err := io.Copy(io.Discard, str)
	require.NoError(t, err, "read full tar stream")
	require.NotZero(t, n, "tar stream should be non-empty")
	require.NoError(t, str.Close(), "close tar stream")

	tdh, err := str.Hierarchy()
	require.NoError(t, err, "TarStreamer Hierarchy")
	require.NotNil(t, tdh, "TarStreamer Hierarchy")

	for i := range tdh.Entries {
		for _, keyval := range tdh.Entries[i].AllKeys() {
			if tdh.Entries[i].Type == FullType || tdh.Entries[i].Type == RelativeType {
				if keyval.Keyword() == "type" && keyval.Value() != "dir" {
					t.Errorf("expected only directories, but %q is a %q", tdh.Entries[i].Name, keyval.Value())
				}
			}
		}
	}
}
