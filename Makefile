# Makefile for the Docker image
# MAINTAINER: Jakob Karalus

.PHONY: all build container push clean test

TAG ?= v0.0.2
PREFIX ?= krallistic

all: container

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o operator cmd/operator/main.go

container: build
	docker build -t $(PREFIX)/kafka-operator:$(TAG) .

push:
	docker push $(PREFIX)/kafka-operator:$(TAG)

clean:
	rm -f kafka-operator

test: clean
	go test $$(go list ./... | grep -v /vendor/)