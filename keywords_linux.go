package mtree

import (
	"archive/tar"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"
	"syscall"

	"github.com/vbatts/go-mtree/xattr"
)

var (
	unameKeywordFunc = func(path string, info os.FileInfo, r io.Reader) (string, error) {
		if hdr, ok := info.Sys().(*tar.Header); ok {
			return fmt.Sprintf("uname=%s", hdr.Uname), nil
		}

		stat := info.Sys().(*syscall.Stat_t)
		u, err := user.LookupId(fmt.Sprintf("%d", stat.Uid))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("uname=%s", u.Username), nil
	}
	uidKeywordFunc = func(path string, info os.FileInfo, r io.Reader) (string, error) {
		if hdr, ok := info.Sys().(*tar.Header); ok {
			return fmt.Sprintf("uid=%d", hdr.Uid), nil
		}
		stat := info.Sys().(*syscall.Stat_t)
		return fmt.Sprintf("uid=%d", stat.Uid), nil
	}
	gidKeywordFunc = func(path string, info os.FileInfo, r io.Reader) (string, error) {
		if hdr, ok := info.Sys().(*tar.Header); ok {
			return fmt.Sprintf("gid=%d", hdr.Gid), nil
		}
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			return fmt.Sprintf("gid=%d", stat.Gid), nil
		}
		return "", nil
	}
	nlinkKeywordFunc = func(path string, info os.FileInfo, r io.Reader) (string, error) {
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			return fmt.Sprintf("nlink=%d", stat.Nlink), nil
		}
		return "", nil
	}
	xattrKeywordFunc = func(path string, info os.FileInfo, r io.Reader) (string, error) {
		if hdr, ok := info.Sys().(*tar.Header); ok {
			if len(hdr.Xattrs) == 0 {
				return "", nil
			}
			klist := []string{}
			for k, v := range hdr.Xattrs {
				klist = append(klist, fmt.Sprintf("xattr.%s=%x", k, sha1.Sum([]byte(v))))
			}
			return strings.Join(klist, " "), nil
		}

		xlist, err := xattr.List(path)
		if err != nil {
			return "", err
		}
		klist := make([]string, len(xlist))
		for i := range xlist {
			data, err := xattr.Get(path, xlist[i])
			if err != nil {
				return "", err
			}
			klist[i] = fmt.Sprintf("xattr.%s=%x", xlist[i], sha1.Sum(data))
		}
		return strings.Join(klist, " "), nil
	}
)
