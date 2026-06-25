#!/bin/bash
set -ex

name=$(basename $0)
root="$(dirname $(dirname $(dirname $0)))"
gomtree=$(go run ${root}/test/realpath/main.go ${root}/gomtree)
t=$(mktemp -d /tmp/go-mtree.XXXXXX)

echo "[${name}] Running in ${t}"

spec=${root}/testdata/relative.mtree

## Default: comments (#) and blank lines are both stripped

${gomtree} mutate ${spec} -  > ${t}/default.out
# no comment lines
(! grep -q '^#' ${t}/default.out)
# no blank lines
(! grep -q '^$' ${t}/default.out)
# regular entries are preserved
grep -q '^lib ' ${t}/default.out
grep -q 'foo ' ${t}/default.out

## --keep-comments preserves comment lines

${gomtree} mutate --keep-comments ${spec} - > ${t}/keep_comments.out
grep -q '^#' ${t}/keep_comments.out
# blanks still stripped
(! grep -q '^$' ${t}/keep_comments.out)

## --keep-blank preserves blank lines

${gomtree} mutate --keep-blank ${spec} - > ${t}/keep_blank.out
grep -q '^$' ${t}/keep_blank.out
# comments still stripped
(! grep -q '^#' ${t}/keep_blank.out)

## --strip-prefix removes the named directory and rewrites descendant paths

${gomtree} mutate --strip-prefix lib ${spec} - > ${t}/stripped.out
# the lib directory entry itself is gone
(! grep -q '^lib type=dir' ${t}/stripped.out)
# direct child of lib is reparented and still present
grep -q '^foo ' ${t}/stripped.out
# full-type entries under lib/ have the prefix stripped
grep -q '^dir/sub ' ${t}/stripped.out
grep -q '^dir/sub/file\.txt ' ${t}/stripped.out
# entries not under lib are untouched
grep -q '^ayo ' ${t}/stripped.out

## Output to a separate file (second positional argument)

cp ${spec} ${t}/input.mtree
${gomtree} mutate ${t}/input.mtree ${t}/output.mtree
test -f ${t}/output.mtree
(! grep -q '^#' ${t}/output.mtree)

## In-place: default output rewrites the input file

cp ${spec} ${t}/inplace.mtree
${gomtree} mutate ${t}/inplace.mtree
test -f ${t}/inplace.mtree
(! grep -q '^#' ${t}/inplace.mtree)
grep -q '^lib ' ${t}/inplace.mtree

rm -rf ${t}
