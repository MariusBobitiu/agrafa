ifneq (,$(wildcard .env.local))
include .env.local
export
endif

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: run build tidy

run:
	go run .

build:
	go build ./...

tidy:
	go mod tidy
