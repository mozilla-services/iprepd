build:
	docker-compose -f compose/docker-compose.base.yml build

run:
	CONFIG_VOLUME=$(shell pwd) docker-compose -f compose/docker-compose.base.yml -f compose/docker-compose.run.yml up

test:
	docker-compose -f compose/docker-compose.base.yml -f compose/docker-compose.test.yml run --rm iprepd
	docker-compose -f compose/docker-compose.base.yml -f compose/docker-compose.test.yml down


.PHONY: build run test
