.PHONY: all build 

BIN_DIR := ./bin
version := $(shell git rev-parse --short=12 HEAD)
timestamp := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
version := $(or $(version), $(shell cat /app/build-release | tr -d '\n'))


all: build

clean:
	rm -f $(BIN_DIR)/ttc

build: lint
	rm -f $(BIN_DIR)/ttc
	go build -o $(BIN_DIR)/ttc -v -ldflags "-X main.rev=$(version) -X main.bts=$(timestamp)" cmd/ttc/*.go


lint:
	go mod tidy
	golangci-lint run --timeout 10m

test: lint
	go test ./...
