.PHONY: build up enter

build:
	./build.sh

up: build
	docker-compose -f docker-compose.yml up -d

enter:
	docker run --rm -it -v "$$PWD:/app" -w /app golang:1.14.0 bash
