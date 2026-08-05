#!/bin/bash
set -ex

name=$(basename $0)
root="$(dirname $(dirname $(dirname $0)))"
gomtree=$(go run ${root}/test/realpath/main.go ${root}/gomtree)
t=$(mktemp -d /tmp/go-mtree.XXXXXX)

echo "[${name}] Running in ${t}"
spec=${root}/testdata/relative.mtree

# mtree2json with -f: spec → JSON
${gomtree} mtree2json -f ${spec} > ${t}/from_spec.json

# valid JSON (python -m json.tool exits non-zero on invalid JSON)
python3 -m json.tool ${t}/from_spec.json > /dev/null

# expected paths appear
grep -q '"path": "lib/foo"' ${t}/from_spec.json
grep -q '"path": "lib/dir/sub"' ${t}/from_spec.json
grep -q '"path": "ayo"' ${t}/from_spec.json

# keywords appear as key/value pairs
grep -q '"type": "file"' ${t}/from_spec.json
grep -q '"type": "dir"' ${t}/from_spec.json

# -k type: only type keyword in output
${gomtree} mtree2json -f ${spec} -k type > ${t}/type_only.json

python3 -m json.tool ${t}/type_only.json > /dev/null
grep -q '"type"' ${t}/type_only.json
(! grep -q '"size"' ${t}/type_only.json)
(! grep -q '"mode"' ${t}/type_only.json)

# stdin input
${gomtree} mtree2json < ${spec} > ${t}/from_stdin.json
python3 -m json.tool ${t}/from_stdin.json > /dev/null
grep -q '"path": "lib/foo"' ${t}/from_stdin.json

# -p: walk a directory
${gomtree} mtree2json -p ${root} > ${t}/from_walk.json
python3 -m json.tool ${t}/from_walk.json > /dev/null
grep -q '"path": "."' ${t}/from_walk.json

# -R size: size keyword removed from output
${gomtree} mtree2json -f ${spec} -R size > ${t}/no_size.json

python3 -m json.tool ${t}/no_size.json > /dev/null
grep -q '"path": "lib/foo"' ${t}/no_size.json
(! grep -q '"size"' ${t}/no_size.json)

# -R size,time: multiple keywords removed
${gomtree} mtree2json -f ${spec} -R size,time > ${t}/no_size_time.json

(! grep -q '"size"' ${t}/no_size_time.json)
(! grep -q '"time"' ${t}/no_size_time.json)
grep -q '"mode"' ${t}/no_size_time.json

# -f and -p together should fail
(! ${gomtree} mtree2json -f ${spec} -p ${root})

# m2j alias works
${gomtree} m2j -f ${spec} > /dev/null

rm -rf ${t}
