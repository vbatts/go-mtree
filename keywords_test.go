package mtree

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyValRoundtrip(t *testing.T) {
	kv := KeyVal("xattr.security.selinux=dW5jb25maW5lZF91Om9iamVjdF9yOnVzZXJfaG9tZV90OnMwAA==")
	expected := "xattr.security.selinux"
	got := string(kv.Keyword())
	assert.Equalf(t, expected, got, "%q keyword", kv)

	expected = "xattr"
	got = string(kv.Keyword().Prefix())
	assert.Equalf(t, expected, got, "%q keyword prefix", kv)

	expected = "security.selinux"
	got = kv.Keyword().Suffix()
	assert.Equalf(t, expected, got, "%q keyword suffix", kv)

	expected = "dW5jb25maW5lZF91Om9iamVjdF9yOnVzZXJfaG9tZV90OnMwAA=="
	got = kv.Value()
	assert.Equalf(t, expected, got, "%q value", kv)

	expected = "xattr.security.selinux=farts"
	got = string(kv.NewValue("farts"))
	assert.Equal(t, expected, got, "NewValue", kv)

	kv1 := KeyVal(expected)
	kv2 := kv.NewValue("farts")
	assert.Equal(t, kv1, kv2, "NewValue should be equivalent to explicit value")

}

type fakeFileInfo struct {
	mtime time.Time
}

func (ffi fakeFileInfo) Name() string {
	// noop
	return ""
}

func (ffi fakeFileInfo) Size() int64 {
	// noop
	return -1
}

func (ffi fakeFileInfo) Mode() os.FileMode {
	// noop
	return 0
}

func (ffi fakeFileInfo) ModTime() time.Time {
	return ffi.mtime
}

func (ffi fakeFileInfo) IsDir() bool {
	return ffi.Mode().IsDir()
}

func (ffi fakeFileInfo) Sys() interface{} {
	// noop
	return nil
}

func TestKeywordsTimeNano(t *testing.T) {
	// We have to make sure that timeKeywordFunc always returns the correct
	// formatting with regards to the nanotime.

	for _, test := range []struct {
		sec, nsec int64
	}{
		{1234, 123456789},
		{5555, 987654321},
		{1337, 100000000},
		{8888, 999999999},
		{144123582122, 1},
		{857125628319, 0},
	} {
		t.Run(fmt.Sprintf("%d.%9.9d", test.sec, test.nsec), func(t *testing.T) {
			mtime := time.Unix(test.sec, test.nsec)
			expected := KeyVal(fmt.Sprintf("time=%d.%9.9d", test.sec, test.nsec))
			got, err := timeKeywordFunc("", fakeFileInfo{
				mtime: mtime,
			}, nil)
			require.NoErrorf(t, err, "time keyword fn")
			if assert.Len(t, got, 1, "time keyword fn") {
				assert.Equal(t, []KeyVal{expected}, got, "should get matching keyword")
			}
		})
	}
}

func TestKeywordsTimeTar(t *testing.T) {
	// tartimeKeywordFunc always has nsec = 0.

	for _, test := range []struct {
		sec, nsec int64
	}{
		{1234, 123456789},
		{5555, 987654321},
		{1337, 100000000},
		{8888, 999999999},
		{144123582122, 1},
		{857125628319, 0},
	} {
		t.Run(fmt.Sprintf("%d.%9.9d", test.sec, test.nsec), func(t *testing.T) {
			mtime := time.Unix(test.sec, test.nsec)
			expected := KeyVal(fmt.Sprintf("tar_time=%d.%9.9d", test.sec, 0))
			got, err := tartimeKeywordFunc("", fakeFileInfo{
				mtime: mtime,
			}, nil)
			require.NoErrorf(t, err, "tar_time keyword fn")
			if assert.Len(t, got, 1, "tar_time keyword fn") {
				assert.Equal(t, []KeyVal{expected}, got, "should get matching keyword")
			}
		})
	}
}

func TestKeywordSynonym(t *testing.T) {
	for _, test := range []struct {
		give   string
		expect Keyword
	}{
		{give: "time", expect: "time"},
		{give: "md5", expect: "md5digest"},
		{give: "md5digest", expect: "md5digest"},
		{give: "rmd160", expect: "ripemd160digest"},
		{give: "rmd160digest", expect: "ripemd160digest"},
		{give: "ripemd160digest", expect: "ripemd160digest"},
		{give: "sha1", expect: "sha1digest"},
		{give: "sha1digest", expect: "sha1digest"},
		{give: "sha256", expect: "sha256digest"},
		{give: "sha256digest", expect: "sha256digest"},
		{give: "sha384", expect: "sha384digest"},
		{give: "sha384digest", expect: "sha384digest"},
		{give: "sha512", expect: "sha512digest"},
		{give: "sha512digest", expect: "sha512digest"},
		{give: "xattr", expect: "xattr"},
		{give: "xattrs", expect: "xattr"},
	} {
		t.Run(test.give, func(t *testing.T) {
			got := KeywordSynonym(test.give)
			assert.Equal(t, test.expect, got)
		})
	}
}
