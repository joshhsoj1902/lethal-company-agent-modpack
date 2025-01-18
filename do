#!/usr/bin/env bash
set -eu -o pipefail

VERSION="${VERSION:-dev}"

build() {
    rm -rf workdir
    rm -f publish.zip || true
    rsync -lrv --exclude=.git --exclude=workdir --exclude=publish.zip --exclude=.gitignore --exclude=do . workdir
    sed -i '/ \/\//d' workdir/manifest.json

    jq '.version_number = "'${VERSION}'"' workdir/manifest.json > workdir/manifest.json.tmp
    mv workdir/manifest.json.tmp workdir/manifest.json

    cd workdir && zip -r ../publish.zip ./
}

"$@"