#!/bin/bash
set -ex

name=$(basename $0)
root="$(dirname $(dirname $(dirname $0)))"
gomtree=$(go run ${root}/test/realpath/main.go ${root}/gomtree)
t=$(mktemp -d /tmp/go-mtree.XXXXXX)

setfattr -n user.has.xattrs -v "true" "${t}"  || exit 0

echo "[${name}] Running in ${t}"

## Test that --strict mode will error out in cases that stock gomtree will not.

pushd ${root}
mkdir -p ${t}/root

mkdir -p ${t}/root/foo/bar
echo "valid" >${t}/root/foo/bar/file
echo "#!/bin/false" >${t}/root/binary
chmod 755 ${t}/root/binary
date="2025-09-05T13:05:10"
touch -d "$date.12345" ${t}/root/time
touch ${t}/root/xattr
setfattr -n user.mtree.testing -v "apples and=bananas" ${t}/root/xattr

keywords_notime=type,uid,gid,nlink,link,mode,flags,xattr,size,sha256
keywords=$keywords_notime,time
${gomtree} -c -k "$keywords" -p ${t}/root -f ${t}/root.mtree

# Make sure cp -ar still validates.
dir=${t}/copy; cp -ar ${t}/root ${dir}
${gomtree} -k "$keywords" -p ${dir} -f ${t}/root.mtree
${gomtree} --strict -k "$keywords" -p ${dir} -f ${t}/root.mtree

# Missing files should not validate.
dir=${t}/missing; cp -ar ${t}/root ${dir}
rm ${dir}/binary
(! ${gomtree} --strict -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} --strict -k "$keywords" -p ${dir} -f ${t}/root.mtree)
# "size" will tip off gomtree even in stock mode, so re-do with just "type".
${gomtree} -k type -p ${dir} -f ${t}/root.mtree
(! ${gomtree} --strict -k type -p ${dir} -f ${t}/root.mtree)

# Extra files should not validate.
dir=${t}/extra; cp -ar ${t}/root ${dir}
touch ${dir}/newfile
(! ${gomtree} --strict -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} --strict -k "$keywords" -p ${dir} -f ${t}/root.mtree)
# "size" will tip off gomtree even in stock mode, so re-do with just "type".
${gomtree} -k type -p ${dir} -f ${t}/root.mtree
(! ${gomtree} --strict -k type -p ${dir} -f ${t}/root.mtree)

# Extra keywords that are missing from the original manifest should not
# validate.
dir=${t}/root
${gomtree} -k "$keywords,md5" -p ${dir} -f ${t}/root.mtree
(! ${gomtree} --strict -k "$keywords,md5" -p ${dir} -f ${t}/root.mtree)

# Mismatched keywords should not validate when comparing two manifests.
dir=${t}/root
keywords2=type,uid,gid,nlink,mode,flags,xattr,size,sha384,time
${gomtree} -c -k "$keywords2" -p ${dir} -f ${t}/root.mtree2
${gomtree} -k "$keywords" -f ${t}/root.mtree -f ${t}/root.mtree2
(! ${gomtree} --strict -k "$keywords" -f ${t}/root.mtree -f ${t}/root.mtree2)
${gomtree} -k "$keywords" -f ${t}/root.mtree2 -f ${t}/root.mtree
(! ${gomtree} --strict -k "$keywords" -f ${t}/root.mtree2 -f ${t}/root.mtree)
${gomtree} -k "$keywords2" -f ${t}/root.mtree -f ${t}/root.mtree2
(! ${gomtree} --strict -k "$keywords2" -f ${t}/root.mtree -f ${t}/root.mtree2)
${gomtree} -k "$keywords2" -f ${t}/root.mtree2 -f ${t}/root.mtree
(! ${gomtree} --strict -k "$keywords2" -f ${t}/root.mtree2 -f ${t}/root.mtree)

# Changed xattrs should not validate (even without --strict).
dir=${t}/xattr-change; cp -ar ${t}/root ${dir}
setfattr -n user.mtree.testing -v "different value" ${dir}/xattr
(! ${gomtree} -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} -k "$keywords" -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} --strict -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} --strict -k "$keywords" -p ${dir} -f ${t}/root.mtree)

# Adding xattrs to existing xattr-set files should not validate (even without
# --strict).
dir=${t}/xattr-add1; cp -ar ${t}/root ${dir}
setfattr -n user.mtree.new -v "newxattr" ${dir}/xattr
(! ${gomtree} -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} -k "$keywords" -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} --strict -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} --strict -k "$keywords" -p ${dir} -f ${t}/root.mtree)

# Adding xattrs to unrelated files should not validate (even without --strict).
dir=${t}/xattr-add2; cp -ar ${t}/root ${dir}
setfattr -n user.mtree.new -v "newxattr" ${dir}/binary
(! ${gomtree} -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} -k "$keywords" -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} --strict -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} --strict -k "$keywords" -p ${dir} -f ${t}/root.mtree)

# Removing xattrs should not validate (even without --strict).
dir=${t}/xattr-rm; cp -ar ${t}/root ${dir}
setfattr -x user.mtree.testing ${dir}/xattr
(! ${gomtree} -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} -k "$keywords" -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} --strict -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} --strict -k "$keywords" -p ${dir} -f ${t}/root.mtree)

# time -> tar_time validation should still work even with --strict mode.
dir=${t}/tartime; cp -ar ${t}/root ${dir}
touch -d "$date.00000" ${dir}/time
(! ${gomtree} -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} -k "$keywords" -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} --strict -p ${dir} -f ${t}/root.mtree)
(! ${gomtree} --strict -k "$keywords" -p ${dir} -f ${t}/root.mtree)
${gomtree} -k "$keywords_notime,tar_time" -p ${dir} -f ${t}/root.mtree
${gomtree} --strict -k "$keywords_notime,tar_time" -p ${dir} -f ${t}/root.mtree
