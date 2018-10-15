build: compose/docker-compose.base.yml
	docker-compose -f compose/docker-compose.base.yml build

run: compose/docker-compose.run.yml
	CONFIG_VOLUME=$(shell pwd) docker-compose -f compose/docker-compose.base.yml -f compose/docker-compose.run.yml up

test: compose/docker-compose.test.yml
	docker-compose -f compose/docker-compose.base.yml -f compose/docker-compose.test.yml run iprepd


.PHONY: build run test
