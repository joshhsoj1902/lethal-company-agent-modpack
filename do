#!/usr/bin/env bash
set -eu -o pipefail

build() {
    rsync -lrv --exclude=.git --exclude=workdir --exclude=publish.zip . workdir
}

"$@"