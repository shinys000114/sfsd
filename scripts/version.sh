#!/bin/bash

if ! git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
    echo "Warning: Not a git repository. Version tag will be 'dev'." >&2
    TAG="dev"
else
    TAG=$(git describe --tags --abbrev=0 2>/dev/null)
    if [ -z "$TAG" ]; then
        TAG="dev"
    fi
fi

BUILD_DATE=${BUILD_DATE:-$(date +%y%m%d%H%M%S)}
VERSION="${TAG}+${BUILD_DATE}"

deb_tag="${TAG#v}"
if [[ "$deb_tag" =~ ^[0-9] ]]; then
    DEB_VERSION="${deb_tag}+${BUILD_DATE}"
else
    DEB_VERSION="0.0.0+${deb_tag}.${BUILD_DATE}"
fi

export TAG
export BUILD_DATE
export VERSION
export DEB_VERSION
