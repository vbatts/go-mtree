package mtree

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsedKeywords(t *testing.T) {
	for _, test := range []struct {
		name string
		blob string
		set  []Keyword
	}{
		{
			name: "NonHomogenous",
			blob: `
#       machine: bananaboat
#          tree: .git
#          date: Wed Nov 16 14:54:17 2016

# .
/set type=file nlink=1 mode=0664 uid=1000 gid=100
. size=4096 type=dir mode=0755 nlink=8 time=1479326055.423853146
  .COMMIT_EDITMSG.un~ size=1006 mode=0644 time=1479325423.450468662 sha1digest=dead0face
  .TAG_EDITMSG.un~ size=1069 mode=0600 time=1471362316.801317529 sha256digest=dead0face
`,
			set: []Keyword{"type", "nlink", "mode", "uid", "gid", "size", "time", "sha1digest", "sha256digest"},
		},
		{
			name: "xattrs",
			blob: `
#          user: cyphar
#       machine: ryuk
#          tree: xattr
#          date: Fri Sep 29 21:00:41 2017
#      keywords: size,type,uid,gid,mode,link,nlink,time,xattr

# .
/set type=file nlink=1 mode=0664 uid=1000 gid=100 xattr.user.kira=SSdsbCB0YWtlIGEgcG90YXRvIGNoaXAuLi4gYW5kIGVhdCBpdCE=
. size=8 type=dir mode=0755 time=1506666472.255992830
    file size=0 mode=0644 time=1506666472.255992830 xattr.user.something=dGVzdA==
..
`,
			set: []Keyword{"size", "type", "uid", "gid", "mode", "nlink", "time", "xattr"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dh, err := ParseSpec(strings.NewReader(test.blob))
			require.NoError(t, err, "parse spec")
			used := dh.UsedKeywords()
			assert.ElementsMatch(t, used, test.set, "UsedKeywords should contain all keywords used")
		})
	}
}
