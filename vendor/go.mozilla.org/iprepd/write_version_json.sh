#!/usr/bin/env bash

set -eo pipefail


: "${CIRCLE_SHA1=$(git rev-parse HEAD)}"
: "${CIRCLE_TAG=$(git describe --tags)}"
: "${CIRCLE_PROJECT_USERNAME=mozilla-services}"
: "${CIRCLE_PROJECT_REPONAME=iprepd}"
: "${CIRCLE_BUILD_URL=localdev}"

printf '{"commit":"%s","version":"%s","source":"https://github.com/%s/%s","build":"%s"}\n' \
            "$CIRCLE_SHA1" \
            "$CIRCLE_TAG" \
            "$CIRCLE_PROJECT_USERNAME" \
            "$CIRCLE_PROJECT_REPONAME" \
            "$CIRCLE_BUILD_URL" > version.json
