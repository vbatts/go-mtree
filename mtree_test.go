package mtree

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	for _, test := range []struct {
		name   string
		counts map[EntryType]int
		size   int64
	}{
		{
			name: "testdata/source.mtree",
			counts: map[EntryType]int{
				//FullType:     0,
				RelativeType: 45,
				CommentType:  37,
				SpecialType:  7,
				DotDotType:   17,
				BlankType:    34,
			},
			size: 7887,
		},
		{
			name: "testdata/source.casync-mtree",
			counts: map[EntryType]int{
				FullType:     744,
				RelativeType: 56,
			},
			size: 168439,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fh, err := os.Open(test.name)
			require.NoError(t, err)
			defer fh.Close()

			var readSpecBuf bytes.Buffer
			rdr := io.TeeReader(fh, &readSpecBuf)

			dh, err := ParseSpec(rdr)
			require.NoErrorf(t, err, "parse spec %s", test.name)

			defer func() {
				if t.Failed() {
					t.Log(spew.Sdump(dh))
				}
			}()

			gotNums := countTypes(dh)
			assert.Equal(t, test.counts, gotNums, "count of entry types mismatch")

			n, err := dh.WriteTo(io.Discard)
			require.NoError(t, err, "write directory hierarchy representation")
			assert.Equal(t, test.size, n, "output mtree spec should match input size")
			// TODO: Verify that the output is equal to the input.
		})
	}
}

func countTypes(dh *DirectoryHierarchy) map[EntryType]int {
	nT := map[EntryType]int{}
	for i := range dh.Entries {
		typ := dh.Entries[i].Type
		if _, ok := nT[typ]; !ok {
			nT[typ] = 1
		} else {
			nT[typ]++
		}
	}
	return nT
}
