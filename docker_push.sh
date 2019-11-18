#!/bin/bash

tag="latest"
if [[ -n "$CIRCLE_TAG" ]]; then
	tag=$CIRCLE_TAG
fi

docker tag repd:build mozilla/repd:${tag}

docker login -u "$DOCKER_USER" -p "$DOCKER_PASS"

docker push mozilla/repd:${tag}
