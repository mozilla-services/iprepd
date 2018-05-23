#!/usr/bin/env bash

set -eo pipefail

# https://github.com/mozilla-services/Dockerflow/blob/master/docs/version_object.md

COMMIT=${TRAVIS_COMMIT:-`git rev-parse HEAD`}
VERSION=${TRAVIS_TAG:-undefined}
SOURCE=undefined
if [[ ! -z "$TRAVIS_PULL_REQUEST_SLUG" ]]; then
	SOURCE=https://github.com/${TRAVIS_PULL_REQUEST_SLUG}
elif [[ ! -z "$TRAVIS_REPO_SLUG" ]]; then
	SOURCE=https://github.com/${TRAVIS_REPO_SLUG}
fi
BUILD=undefined
if [[ ! -z "$TRAVIS_BUILD_ID" ]]; then
	BUILD=https://travis-ci.org/mozilla-services/iprepd/builds/${TRAVIS_BUILD_ID}
fi

echo $COMMIT
echo $VERSION
echo $SOURCE
echo $BUILD

printf '{"commit": "%s", "version": "%s", "source": "%s", "build": "%s"}\n' \
	"$COMMIT" "$VERSION" "$SOURCE" "$BUILD" > version.json
