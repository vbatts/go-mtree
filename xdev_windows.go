//go:build windows

package mtree

import "errors"

// ExcludeMountPoints is not supported on Windows.
func ExcludeMountPoints(root string) (ExcludeFunc, error) {
	return nil, errors.New("mount point detection is not supported on Windows")
}
