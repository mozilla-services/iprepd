#!/bin/bash

tag="latest"
if [[ -n "$TRAVIS_TAG" ]]; then
	tag=$TRAVIS_TAG
fi

docker tag iprepd:build mozilla/iprepd:${tag}

docker login -u "$DOCKER_USER" -p "$DOCKER_PASS"

docker push mozilla/iprepd:${tag}
