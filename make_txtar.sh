#!/usr/bin/env bash

set -e

tmp=$(mktemp -d farwrite.XXXXX)

if [ -z "${tmp+x}" ] || [ -z "$tmp" ]; then
    echo "Error: \$tmp is not set or is an empty string."
    exit 1
fi

{
    rg --files . \
        | grep -v $tmp/filelist.txt \
        | grep -vE 'farwrite$' \
        | grep -v README.org \
        | grep -v make_txtar.sh \
        | grep -v go.sum \
        | grep -v go.mod \
        | grep -v Makefile \
        | grep -v cmd/main.go \
        | grep -v logger.go \
        # | grep -v farwrite.go \

} | tee $tmp/filelist.txt
tar -cf $tmp/farwrite.tar -T $tmp/filelist.txt
mkdir -p $tmp/farwrite
tar xf $tmp/farwrite.tar -C $tmp/farwrite
rg --files $tmp/farwrite
txtar-c $tmp/farwrite | pbcopy

rm -rf $tmp
