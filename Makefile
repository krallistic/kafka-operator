# Makefile for the Docker image
# MAINTAINER: Jakob Karalus

.PHONY: all build container push clean test

TAG ?= 0.0.1
PREFIX ?= krallistic

all: container

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/operator/main.go

container: build
	docker build -t $(PREFIX)/kafka-operator:$(TAG) .

push:
	docker push $(PREFIX)/kafka-operator:$(TAG)

clean:
	rm -f kafka-operator

test: clean
	go test $$(go list ./... | grep -v /vendor/)