#! /bin/bash

[ -z "$1" ] && echo "Error: git repo url required" && exit 1
repo=$1

set -ex
if [ -f "/makisu-secrets/github-token/github_token" ]; then 
    repo="$(cat /makisu-secrets/github-token/github_token)@$repo"
fi
repo="https://$repo"
git clone "$repo" /tmp/repo

cd /tmp/repo
makisu-client "${@:2}" .
