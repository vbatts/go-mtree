#!/bin/bash
set -e

name=$(basename $0)
root="$(dirname $(dirname $(dirname $0)))"
gomtree=$(go run ${root}/test/realpath/main.go ${root}/gomtree)
t=$(mktemp -d /tmp/go-mtree.XXXXXX)

echo "[${name}] Running in ${t}"

## testing comparing two files

pushd ${root}
mkdir -p ${t}/extract
git archive --format=tar HEAD^{tree} . | tar -C ${t}/extract/ -x

${gomtree} -K sha256digest -c -p ${t}/extract/ > ${t}/${name}-1.mtree
rm -rf ${t}/extract/*.go
${gomtree} -K sha256digest -c -p ${t}/extract/ > ${t}/${name}-2.mtree

# this _ought_ to fail because the files are missing now
(! ${gomtree} -f ${t}/${name}-1.mtree -f ${t}/${name}-2.mtree)

popd
rm -rf ${t}
