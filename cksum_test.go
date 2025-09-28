package mtree

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	checkFile        = "./testdata/source.mtree"
	checkSum  uint32 = 1048442895
	checkSize        = 9110
)

// testing that the cksum function matches that of cksum(1) utility (silly POSIX crc32)
func TestCksum(t *testing.T) {
	fh, err := os.Open(checkFile)
	require.NoError(t, err)
	defer fh.Close()

	sum, i, err := cksum(fh)
	require.NoError(t, err)
	assert.Equal(t, checkSize, i, "checksum size mismatch")
	assert.Equal(t, checkSum, sum, "checksum mismatch")
}
