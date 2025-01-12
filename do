#!/usr/bin/env bash
set -eu -o pipefail

build() {
    rm -rf workdir
    rm -f publish.zip || true
    rsync -lrv --exclude=.git --exclude=workdir --exclude=publish.zip --exclude=.gitignore --exclude=do . workdir
    cd workdir && zip -r ../publish.zip ./
}

"$@"