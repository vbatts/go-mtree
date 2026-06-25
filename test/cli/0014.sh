#!/bin/bash
set -ex

name=$(basename $0)
root="$(dirname $(dirname $(dirname $0)))"
gomtree=$(go run ${root}/test/realpath/main.go ${root}/gomtree)
t=$(mktemp -d /tmp/go-mtree.XXXXXX)

echo "[${name}] Running in ${t}"

# Build a small tree to work with
mkdir -p ${t}/root/subdir
echo "hello" > ${t}/root/file1
echo "world" > ${t}/root/file2
echo "sub"   > ${t}/root/subdir/file3

## -R: remove keywords from the current set

# Default manifest includes time=
${gomtree} -c -p ${t}/root > ${t}/with_time.mtree
grep -q 'time=' ${t}/with_time.mtree

# -R time produces a manifest with no time= entries
${gomtree} -c -p ${t}/root -R time > ${t}/no_time.mtree
(! grep -q 'time=' ${t}/no_time.mtree)

# The timeless manifest still validates
${gomtree} -f ${t}/no_time.mtree -p ${t}/root

# -R all strips every keyword; manifest should still be created without error
${gomtree} -c -p ${t}/root -R all > ${t}/no_kw.mtree
(! grep -q 'size=' ${t}/no_kw.mtree)

# Combining -K and -R: add sha256 then remove time
${gomtree} -c -p ${t}/root -K sha256digest -R time > ${t}/sha256_notime.mtree
grep -q 'sha256digest=' ${t}/sha256_notime.mtree
(! grep -q 'time=' ${t}/sha256_notime.mtree)

## -e: don't report extra files

# Create manifest without size: adding a file changes the parent dir's size,
# which would produce a Modified delta for "." unrelated to -e behaviour.
${gomtree} -c -R size -p ${t}/root > ${t}/root.mtree

# Add a file not in the manifest
echo "extra" > ${t}/root/extra_file

# Without -e, strict mode flags the extra file
(! ${gomtree} --strict -R size -p ${t}/root -f ${t}/root.mtree)

# With -e, the extra file is silently ignored
${gomtree} -e -R size -p ${t}/root -f ${t}/root.mtree

rm ${t}/root/extra_file

## -X: exclude paths by fnmatch patterns

echo "skip me" > ${t}/root/skip.txt
echo "skip me too" > ${t}/root/also.log

# Exclude *.txt by basename pattern
printf '*.txt\n' > ${t}/excl.txt
${gomtree} -c -p ${t}/root -X ${t}/excl.txt > ${t}/no_txt.mtree
(! grep -q 'skip\.txt' ${t}/no_txt.mtree)
grep -q 'also\.log' ${t}/no_txt.mtree

# Exclude a directory by basename pattern; its contents must not appear either
printf 'subdir\n' > ${t}/excl_dir.txt
${gomtree} -c -p ${t}/root -X ${t}/excl_dir.txt > ${t}/no_subdir.mtree
(! grep -q 'subdir' ${t}/no_subdir.mtree)
(! grep -q 'file3' ${t}/no_subdir.mtree)

# Comments and blank lines in the exclude file are ignored
printf '# this is a comment\n\n*.log\n\n# another comment\n' > ${t}/excl_comments.txt
${gomtree} -c -p ${t}/root -X ${t}/excl_comments.txt > ${t}/no_log.mtree
(! grep -q 'also\.log' ${t}/no_log.mtree)
grep -q 'skip\.txt' ${t}/no_log.mtree

rm ${t}/root/skip.txt ${t}/root/also.log

## -x: don't descend below mount points
# We cannot create a real mount point without root, so just verify that the
# flag does not break operation on an ordinary single-device tree.
${gomtree} -c -p ${t}/root -x > ${t}/xdev.mtree
${gomtree} -x -p ${t}/root -f ${t}/xdev.mtree

rm -rf ${t}
