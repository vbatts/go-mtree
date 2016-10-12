#!/usr/bin/env bash
set -e

# this script is used to update vendored dependencies
#
# Usage:
# vendor.sh revendor all dependencies
# vendor.sh github.com/docker/libkv revendor only the libkv dependency.
# vendor.sh github.com/docker/libkv v0.2.1 vendor only libkv at the specified tag/commit.
# vendor.sh git github.com/docker/libkv v0.2.1 is the same but specifies the VCS for cases where the VCS is something else than git
# vendor.sh git golang.org/x/sys eb2c74142fd19a79b3f237334c7384d5167b1b46 https://github.com/golang/sys.git vendor only golang.org/x/sys downloading from the specified URL

cd "$(dirname "$BASH_SOURCE")/.."
source 'hack/.vendor-helpers.sh'

case $# in
0)
	rm -rf vendor/
	;;
# If user passed arguments to the script
1)
	path="$PWD/hack/vendor.sh"
	if ! cloneGrep="$(grep -E "^clone [^ ]+ $1" "$path")"; then
		echo >&2 "error: failed to find 'clone ... $1' in $path"
		exit 1
	fi
	eval "$cloneGrep"
	clean
	exit 0
	;;
2)
	rm -rf "vendor/src/$1"
	clone git "$1" "$2"
	clean
	exit 0
	;;
[34])
	rm -rf "vendor/src/$2"
	clone "$@"
	clean
	exit 0
	;;
*)
	>&2 echo "error: unexpected parameters"
	exit 1
	;;
esac

# go-mtree
clone git github.com/golang/crypto 4cd25d65a015cc83d41bf3454e6e8d6c116d16da
