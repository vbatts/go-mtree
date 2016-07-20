package mtree

import (
	"archive/tar"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/ripemd160"
)

// KeywordFunc is the type of a function called on each file to be included in
// a DirectoryHierarchy, that will produce the string output of the keyword to
// be included for the file entry. Otherwise, empty string.
// io.Reader `r` is to the file stream for the file payload. While this
// function takes an io.Reader, the caller needs to reset it to the beginning
// for each new KeywordFunc
type KeywordFunc func(path string, info os.FileInfo, r io.Reader) (string, error)

// KeyVal is a "keyword=value"
type KeyVal string

// Keyword is the mapping to the available keywords
func (kv KeyVal) Keyword() string {
	if !strings.Contains(string(kv), "=") {
		return ""
	}
	chunks := strings.SplitN(strings.TrimSpace(string(kv)), "=", 2)[0]
	if !strings.Contains(chunks, ".") {
		return chunks
	}
	return strings.SplitN(chunks, ".", 2)[0]
}

// KeywordSuffix is really only used for xattr, as the keyword is a prefix to
// the xattr "namespace.key"
func (kv KeyVal) KeywordSuffix() string {
	if !strings.Contains(string(kv), "=") {
		return ""
	}
	chunks := strings.SplitN(strings.TrimSpace(string(kv)), "=", 2)[0]
	if !strings.Contains(chunks, ".") {
		return ""
	}
	return strings.SplitN(chunks, ".", 2)[1]
}

// Value is the data/value portion of "keyword=value"
func (kv KeyVal) Value() string {
	if !strings.Contains(string(kv), "=") {
		return ""
	}
	return strings.SplitN(strings.TrimSpace(string(kv)), "=", 2)[1]
}

// keywordSelector takes an array of "keyword=value" and filters out that only the set of words
func keywordSelector(keyval, words []string) []string {
	retList := []string{}
	for _, kv := range keyval {
		if inSlice(KeyVal(kv).Keyword(), words) {
			retList = append(retList, kv)
		}
	}
	return retList
}

// NewKeyVals constructs a list of KeyVal from the list of strings, like "keyword=value"
func NewKeyVals(keyvals []string) KeyVals {
	kvs := make(KeyVals, len(keyvals))
	for i := range keyvals {
		kvs[i] = KeyVal(keyvals[i])
	}
	return kvs
}

// KeyVals is a list of KeyVal
type KeyVals []KeyVal

// Has the "keyword" present in the list of KeyVal, and returns the
// corresponding KeyVal, else an empty string.
func (kvs KeyVals) Has(keyword string) KeyVal {
	for i := range kvs {
		if kvs[i].Keyword() == keyword {
			return kvs[i]
		}
	}
	return emptyKV
}

var emptyKV = KeyVal("")

// MergeSet takes the current setKeyVals, and then applies the entryKeyVals
// such that the entry's values win. The union is returned.
func MergeSet(setKeyVals, entryKeyVals []string) KeyVals {
	retList := NewKeyVals(append([]string{}, setKeyVals...))
	eKVs := NewKeyVals(entryKeyVals)
	seenKeywords := []string{}
	for i := range retList {
		word := retList[i].Keyword()
		if ekv := eKVs.Has(word); ekv != emptyKV {
			retList[i] = ekv
		}
		seenKeywords = append(seenKeywords, word)
	}
	for i := range eKVs {
		if !inSlice(eKVs[i].Keyword(), seenKeywords) {
			retList = append(retList, eKVs[i])
		}
	}
	return retList
}

var (
	// DefaultKeywords has the several default keyword producers (uid, gid,
	// mode, nlink, type, size, mtime)
	DefaultKeywords = []string{
		"size",
		"type",
		"uid",
		"gid",
		"mode",
		"link",
		"nlink",
		"time",
	}
	// SetKeywords is the default set of keywords calculated for a `/set` SpecialType
	SetKeywords = []string{
		"uid",
		"gid",
	}
	// KeywordFuncs is the map of all keywords (and the functions to produce them)
	KeywordFuncs = map[string]KeywordFunc{
		"size":            sizeKeywordFunc,                                     // The size, in bytes, of the file
		"type":            typeKeywordFunc,                                     // The type of the file
		"time":            timeKeywordFunc,                                     // The last modification time of the file
		"link":            linkKeywordFunc,                                     // The target of the symbolic link when type=link
		"uid":             uidKeywordFunc,                                      // The file owner as a numeric value
		"gid":             gidKeywordFunc,                                      // The file group as a numeric value
		"nlink":           nlinkKeywordFunc,                                    // The number of hard links the file is expected to have
		"uname":           unameKeywordFunc,                                    // The file owner as a symbolic name
		"mode":            modeKeywordFunc,                                     // The current file's permissions as a numeric (octal) or symbolic value
		"cksum":           cksumKeywordFunc,                                    // The checksum of the file using the default algorithm specified by the cksum(1) utility
		"md5":             hasherKeywordFunc("md5", md5.New),                   // The MD5 message digest of the file
		"md5digest":       hasherKeywordFunc("md5digest", md5.New),             // A synonym for `md5`
		"rmd160":          hasherKeywordFunc("rmd160", ripemd160.New),          // The RIPEMD160 message digest of the file
		"rmd160digest":    hasherKeywordFunc("rmd160digest", ripemd160.New),    // A synonym for `rmd160`
		"ripemd160digest": hasherKeywordFunc("ripemd160digest", ripemd160.New), // A synonym for `rmd160`
		"sha1":            hasherKeywordFunc("sha1", sha1.New),                 // The SHA1 message digest of the file
		"sha1digest":      hasherKeywordFunc("sha1digest", sha1.New),           // A synonym for `sha1`
		"sha256":          hasherKeywordFunc("sha256", sha256.New),             // The SHA256 message digest of the file
		"sha256digest":    hasherKeywordFunc("sha256digest", sha256.New),       // A synonym for `sha256`
		"sha384":          hasherKeywordFunc("sha384", sha512.New384),          // The SHA384 message digest of the file
		"sha384digest":    hasherKeywordFunc("sha384digest", sha512.New384),    // A synonym for `sha384`
		"sha512":          hasherKeywordFunc("sha512", sha512.New),             // The SHA512 message digest of the file
		"sha512digest":    hasherKeywordFunc("sha512digest", sha512.New),       // A synonym for `sha512`

		// This is not an upstreamed keyword, but a needed attribute for file validation.
		// The pattern for this keyword key is prefixed by "xattr." followed by the extended attribute "namespace.key".
		// The keyword value is the SHA1 digest of the extended attribute's value.
		// In this way, the order of the keys does not matter, and the contents of the value is not revealed.
		"xattr": xattrKeywordFunc,
	}
)

var (
	modeKeywordFunc = func(path string, info os.FileInfo, r io.Reader) (string, error) {
		permissions := info.Mode().Perm()
		if os.ModeSetuid&info.Mode() > 0 {
			permissions |= (1 << 11)
		}
		if os.ModeSetgid&info.Mode() > 0 {
			permissions |= (1 << 10)
		}
		if os.ModeSticky&info.Mode() > 0 {
			permissions |= (1 << 9)
		}
		return fmt.Sprintf("mode=%#o", permissions), nil
	}
	sizeKeywordFunc = func(path string, info os.FileInfo, r io.Reader) (string, error) {
		return fmt.Sprintf("size=%d", info.Size()), nil
	}
	cksumKeywordFunc = func(path string, info os.FileInfo, r io.Reader) (string, error) {
		if !info.Mode().IsRegular() {
			return "", nil
		}
		sum, _, err := cksum(r)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("cksum=%d", sum), nil
	}
	hasherKeywordFunc = func(name string, newHash func() hash.Hash) KeywordFunc {
		return func(path string, info os.FileInfo, r io.Reader) (string, error) {
			if !info.Mode().IsRegular() {
				return "", nil
			}
			h := newHash()
			if _, err := io.Copy(h, r); err != nil {
				return "", err
			}
			return fmt.Sprintf("%s=%x", name, h.Sum(nil)), nil
		}
	}
	timeKeywordFunc = func(path string, info os.FileInfo, r io.Reader) (string, error) {
		t := info.ModTime().UnixNano()
		if t == 0 {
			return "time=0.000000000", nil
		}
		return fmt.Sprintf("time=%d.%9.9d", (t / 1e9), (t % (t / 1e9))), nil
	}
	linkKeywordFunc = func(path string, info os.FileInfo, r io.Reader) (string, error) {
		if sys, ok := info.Sys().(*tar.Header); ok {
			if sys.Linkname != "" {
				return fmt.Sprintf("link=%s", sys.Linkname), nil
			}
			return "", nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			str, err := os.Readlink(path)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("link=%s", str), nil
		}
		return "", nil
	}
	typeKeywordFunc = func(path string, info os.FileInfo, r io.Reader) (string, error) {
		if info.Mode().IsDir() {
			return "type=dir", nil
		}
		if info.Mode().IsRegular() {
			return "type=file", nil
		}
		if info.Mode()&os.ModeSocket != 0 {
			return "type=socket", nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "type=link", nil
		}
		if info.Mode()&os.ModeNamedPipe != 0 {
			return "type=fifo", nil
		}
		if info.Mode()&os.ModeDevice != 0 {
			if info.Mode()&os.ModeCharDevice != 0 {
				return "type=char", nil
			}
			return "type=device", nil
		}
		return "", nil
	}
)
