#!/bin/bash
set -e

name=$(basename $0)
root="$(dirname $(dirname $(dirname $0)))"
gomtree=$(go run ${root}/test/realpath/main.go ${root}/gomtree)
t=$(mktemp -d /tmp/go-mtree.XXXXXX)

echo "[${name}] Running in ${t}"

## testing comparing two files

pushd ${root}
mkdir -p ${t}/
touch ${t}/foo

## can not walk a file. We're expecting a directory.
## https://github.com/vbatts/go-mtree/issues/166
(! ${gomtree} -c -K uname,uid,gname,gid,type,nlink,link,mode,flags,xattr,xattrs,size,time,sha256 -p ${t}/foo)

popd
rm -rf ${t}
