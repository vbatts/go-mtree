#!/bin/bash
set -ex

name=$(basename $0)
root="$(dirname $(dirname $(dirname $0)))"
gomtree=$(go run ${root}/test/realpath/main.go ${root}/gomtree)
t=$(mktemp -d /tmp/go-mtree.XXXXXX)

echo "[${name}] Running in ${t}"
spec=${root}/testdata/relative.mtree

# -C: dump spec with full paths
${gomtree} -C -f ${spec} > ${t}/dump.out

# entries appear with full paths (no ../ lines)
grep -q '^lib type=dir' ${t}/dump.out
grep -q '^lib/foo ' ${t}/dump.out
grep -q '^lib/dir/sub type=dir' ${t}/dump.out
grep -q '^lib/dir/sub/file\.txt ' ${t}/dump.out
grep -q '^ayo ' ${t}/dump.out

# -C with -k type: only type keyword
${gomtree} -C -f ${spec} -k type > ${t}/type_only.out

grep -q '^lib type=dir$' ${t}/type_only.out
grep -q '^lib/foo type=file$' ${t}/type_only.out
# no extra keywords present
(! grep -q 'size=' ${t}/type_only.out)
(! grep -q 'mode=' ${t}/type_only.out)

# -C with -R size: size removed from output
${gomtree} -C -f ${spec} -R size > ${t}/no_size.out

grep -q '^lib/foo ' ${t}/no_size.out
(! grep -q 'size=' ${t}/no_size.out)

# -C without -f should fail
(! ${gomtree} -C)

rm -rf ${t}
