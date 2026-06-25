//go:build !windows

package mtree

import (
	"os"
	"syscall"
)

// ExcludeMountPoints returns an ExcludeFunc that excludes any directory whose
// device ID differs from the device ID of root, preventing the walk from
// descending across mount points.
func ExcludeMountPoints(root string) (ExcludeFunc, error) {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	rootDev := rootInfo.Sys().(*syscall.Stat_t).Dev
	return func(path string, info os.FileInfo) bool {
		if !info.IsDir() {
			return false
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return false
		}
		return st.Dev != rootDev
	}, nil
}
