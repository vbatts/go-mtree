package mtree

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// UpdateKeywordFunc is the signature for a function that will restore a file's
// attributes. Where path is relative path to the file, and value to be
// restored to.
type UpdateKeywordFunc func(path string, value string) (os.FileInfo, error)

// UpdateKeywordFuncs is the registered list of functions to update file attributes.
// Keyed by the keyword as it would show up in the manifest
var UpdateKeywordFuncs = map[Keyword]UpdateKeywordFunc{
	"mode":     modeUpdateKeywordFunc,
	"time":     timeUpdateKeywordFunc,
	"tar_time": tartimeUpdateKeywordFunc,
	"uid":      uidUpdateKeywordFunc,
	"gid":      gidUpdateKeywordFunc,
}

func uidUpdateKeywordFunc(path, value string) (os.FileInfo, error) {
	uid, err := strconv.Atoi(value)
	if err != nil {
		return nil, err
	}
	if err := os.Lchown(path, uid, -1); err != nil {
		return nil, err
	}
	return os.Lstat(path)
}

func gidUpdateKeywordFunc(path, value string) (os.FileInfo, error) {
	gid, err := strconv.Atoi(value)
	if err != nil {
		return nil, err
	}
	if err := os.Lchown(path, -1, gid); err != nil {
		return nil, err
	}
	return os.Lstat(path)
}

func modeUpdateKeywordFunc(path, value string) (os.FileInfo, error) {
	vmode, err := strconv.ParseInt(value, 8, 32)
	if err != nil {
		return nil, err
	}
	Debugf("path: %q, value: %q, vmode: %o", path, value, vmode)
	if err := os.Chmod(path, os.FileMode(vmode)); err != nil {
		return nil, err
	}
	return os.Lstat(path)
}

// since tar_time will only be second level precision, then when restoring the
// filepath from a tar_time, then compare the seconds first and only Chtimes if
// the seconds value is different.
func tartimeUpdateKeywordFunc(path, value string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}

	v := strings.SplitN(value, ".", 2)
	if len(v) != 2 {
		return nil, fmt.Errorf("expected a number like 1469104727.000000000")
	}
	sec, err := strconv.ParseInt(v[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("expected seconds, but got %q", v[0])
	}

	// if the seconds are the same, don't do anything, because the file might
	// have nanosecond value, and if using tar_time it would zero it out.
	if info.ModTime().Unix() == sec {
		return info, nil
	}

	vtime := time.Unix(sec, 0)
	if err := os.Chtimes(path, vtime, vtime); err != nil {
		return nil, err
	}
	return os.Lstat(path)
}

// this is nano second precision
func timeUpdateKeywordFunc(path, value string) (os.FileInfo, error) {
	v := strings.SplitN(value, ".", 2)
	if len(v) != 2 {
		return nil, fmt.Errorf("expected a number like 1469104727.871937272")
	}
	nsec, err := strconv.ParseInt(v[0]+v[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("expected nano seconds, but got %q", v[0]+v[1])
	}
	Debugf("arg: %q; nsec: %q", v[0]+v[1], nsec)

	vtime := time.Unix(0, nsec)
	if err := os.Chtimes(path, vtime, vtime); err != nil {
		return nil, err
	}
	return os.Lstat(path)
}
