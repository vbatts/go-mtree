#!/bin/bash
set -ex

name=$(basename $0)
root="$(dirname $(dirname $(dirname $0)))"
gomtree=$(go run ${root}/test/realpath/main.go ${root}/gomtree)
t=$(mktemp -d /tmp/go-mtree.XXXXXX)

echo "[${name}] Running in ${t}"

## Make sure that comparing manifests with tar_time and time are correctly
## truncated.

pushd ${root}
mkdir -p ${t}/root

date="2025-09-05T13:05:10" # POSIX format for "touch -d".

echo "less than .5" >${t}/root/lowerhalf
touch -d "$date.1000" ${t}/root/lowerhalf
echo "more than .5" >${t}/root/upperhalf
touch -d "$date.8000" ${t}/root/upperhalf
echo "no subsecond" >${t}/root/tartime
touch -d "$date.0000" ${t}/root/tartime

keywords=type,uid,gid,nlink,link,mode,flags,xattr,size,sha256

# Generate regular manifests with time and tar_time.
${gomtree} -c -k ${keywords},time -p ${t}/root -f ${t}/time.mtree
${gomtree} -c -k ${keywords},tar_time -p ${t}/root -f ${t}/tartime.mtree

# Make sure that tar_time truncates the value.
unix="$(date -d ${date} +%s)"
grep -q "lowerhalf.*tar_time=$unix.000000000" ${t}/tartime.mtree
grep -q "upperhalf.*tar_time=$unix.000000000" ${t}/tartime.mtree
grep -q "tartime.*tar_time=$unix.000000000" ${t}/tartime.mtree

# Validation with both manifests should still succeed.
${gomtree} validate -p ${t}/root -f ${t}/time.mtree
${gomtree} validate -p ${t}/root -f ${t}/tartime.mtree
# Manifest comparison should also succeed.
${gomtree} validate -f ${t}/tartime.mtree -f ${t}/time.mtree
${gomtree} validate -f ${t}/time.mtree -f ${t}/tartime.mtree

# Truncate the on-disk timestamps manually.
touch -d "$date.0000" ${t}/root/lowerhalf
touch -d "$date.0000" ${t}/root/upperhalf
touch -d "$date.0000" ${t}/root/tartime

# Only the tar_time manifest should succeed.
(! ${gomtree} validate -p ${t}/root -f ${t}/time.mtree)
${gomtree} validate -p ${t}/root -f ${t}/tartime.mtree
${gomtree} validate -k ${keywords},time -p ${t}/root -f ${t}/tartime.mtree
# ... unless you force the usage of tar_time.
${gomtree} validate -k ${keywords},tar_time -p ${t}/root -f ${t}/time.mtree

# The same goes for if you generate the manifests and compare them instead.
${gomtree} -c -k ${keywords},time -p ${t}/root -f ${t}/time-trunc.mtree
${gomtree} -c -k ${keywords},tar_time -p ${t}/root -f ${t}/tartime-trunc.mtree
# Comparing time with time should fail ...
(! ${gomtree} validate -f ${t}/time.mtree -f ${t}/time-trunc.mtree)
(! ${gomtree} validate -f ${t}/time-trunc.mtree -f ${t}/time.mtree)
# ... tar_time with tar_time should succeed ...
${gomtree} validate -f ${t}/tartime.mtree -f ${t}/tartime-trunc.mtree
${gomtree} validate -f ${t}/tartime-trunc.mtree -f ${t}/tartime.mtree
# ... old tar_time with new time should succeed ...
${gomtree} validate -f ${t}/tartime.mtree -f ${t}/time-trunc.mtree
${gomtree} validate -f ${t}/time-trunc.mtree -f ${t}/tartime.mtree
# ... and new tar_time against old time should succeed.
${gomtree} validate -f ${t}/tartime-trunc.mtree -f ${t}/time.mtree
${gomtree} validate -f ${t}/time.mtree -f ${t}/tartime-trunc.mtree

# Change the timestamp entirely.
touch -d "1997-03-25T13:40:00" ${t}/root/lowerhalf
touch -d "1997-03-25T13:40:00" ${t}/root/upperhalf
touch -d "1997-03-25T13:40:00" ${t}/root/tartime

# Now all validations should fail.
(! ${gomtree} validate -p ${t}/root -f ${t}/time.mtree)
(! ${gomtree} validate -p ${t}/root -f ${t}/tartime.mtree)
(! ${gomtree} validate -k ${keywords},tar_time -p ${t}/root -f ${t}/time.mtree)
(! ${gomtree} validate -k ${keywords},time -p ${t}/root -f ${t}/tartime.mtree)

# Ditto for generating the manifests and comparing them.
${gomtree} -c -k ${keywords},time -p ${t}/root -f ${t}/time-change.mtree
${gomtree} -c -k ${keywords},tar_time -p ${t}/root -f ${t}/tartime-change.mtree

# Try all combinations.
lefts=( ${t}/{tar,}time{,-trunc}.mtree )
rights=( ${t}/{tar,}time-change.mtree )
for left in "${lefts[@]}"; do
	for right in "${rights[@]}"; do
		(! ${gomtree} validate -f ${left} -f ${right})
		(! ${gomtree} validate -f ${right} -f ${left})
	done
done
