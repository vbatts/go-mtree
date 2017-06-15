// +build !linux

package mtree

import "os"

func xattrUpdateKeywordFunc(keyword Keyword, path, value string) (os.FileInfo, error) {
	return os.Lstat(path)
}
