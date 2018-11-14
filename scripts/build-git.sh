#! /bin/bash

[ -z "$1" ] && echo "Error: git repo url required" && exit 1
repo=$1

set -ex
git clone "$repo" /tmp/repo

cd /tmp/repo
makisu-client "${@:2}" .
