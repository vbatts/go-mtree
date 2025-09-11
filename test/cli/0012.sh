#!/bin/bash
set -ex

name=$(basename $0)
root="$(dirname $(dirname $(dirname $0)))"
gomtree=$(go run ${root}/test/realpath/main.go ${root}/gomtree)
t=$(mktemp -d /tmp/go-mtree.XXXXXX)

echo "[${name}] Running in ${t}"

## Make sure that comparing manifests with tar_time and time correctly detect
## errors (while also *truncating* timestamps for tar_time comparisons).

pushd ${root}
mkdir -p ${t}/root

date="2025-09-05T13:05:10.12345" # POSIX format for "touch -d".

touch -d "$date" ${t}/root/foo

keywords=type,uid,gid,nlink,link,mode,flags,xattr,size,sha256

# Generate regular manifests with time and tar_time.
${gomtree} -c -k ${keywords},time -p ${t}/root -f ${t}/time.mtree
${gomtree} -c -k ${keywords},tar_time -p ${t}/root -f ${t}/tartime.mtree

# Validation with both manifests should still succeed.
${gomtree} validate -p ${t}/root -f ${t}/time.mtree
${gomtree} validate -p ${t}/root -f ${t}/tartime.mtree
# Manifest comparison should also succeed.
${gomtree} validate -f ${t}/tartime.mtree -f ${t}/time.mtree
${gomtree} validate -f ${t}/time.mtree -f ${t}/tartime.mtree
# Forcefully requesting a different time type should also succeed.
${gomtree} validate -k ${keywords},tar_time -p ${t}/root -f ${t}/time.mtree
${gomtree} validate -k ${keywords},time -p ${t}/root -f ${t}/tartime.mtree

# Change the timestamp.
touch -d "1997-03-25T13:40:00.67890" ${t}/root/foo

# Standard validations should fail.
(! ${gomtree} validate -p ${t}/root -f ${t}/time.mtree)
(! ${gomtree} validate -p ${t}/root -f ${t}/tartime.mtree)

# Forcefully requesting a different time type should also fail.
(! ${gomtree} validate -k ${keywords},tar_time -p ${t}/root -f ${t}/time.mtree)
(! ${gomtree} validate -k ${keywords},time -p ${t}/root -f ${t}/tartime.mtree)

# Ditto if we generate the manifests and compare them.
${gomtree} -c -k ${keywords},time -p ${t}/root -f ${t}/time-change.mtree
${gomtree} -c -k ${keywords},tar_time -p ${t}/root -f ${t}/tartime-change.mtree

# Same time types.
(! ${gomtree} validate -f ${t}/time.mtree -f ${t}/time-change.mtree)
(! ${gomtree} validate -f ${t}/time-change.mtree -f ${t}/time.mtree)
(! ${gomtree} validate -f ${t}/tartime.mtree -f ${t}/tartime-change.mtree)
(! ${gomtree} validate -f ${t}/tartime-change.mtree -f ${t}/tartime.mtree)

# Different time types:
# (old) time <=> (new) tar_time
(! ${gomtree} validate -f ${t}/time.mtree -f ${t}/tartime-change.mtree)
(! ${gomtree} validate -f ${t}/tartime-change.mtree -f ${t}/time.mtree)
# (old) tar_time <=> (new) time
(! ${gomtree} validate -f ${t}/tartime.mtree -f ${t}/time-change.mtree)
(! ${gomtree} validate -f ${t}/time-change.mtree -f ${t}/tartime.mtree)
