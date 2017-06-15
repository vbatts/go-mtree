// +build linux

package mtree

import (
	"encoding/base64"
	"os"

	"github.com/vbatts/go-mtree/xattr"
)

func xattrUpdateKeywordFunc(keyword Keyword, path, value string) (os.FileInfo, error) {
	buf, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	if err := xattr.Set(path, keyword.Suffix(), buf); err != nil {
		return nil, err
	}
	return os.Lstat(path)
}
