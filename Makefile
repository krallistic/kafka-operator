# Makefile for the Docker image
# MAINTAINER: Jakob Karalus

.PHONY: all build container push deploy clean test

TAG ?= v0.0.2
PREFIX ?= krallistic

all: push

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/kafka_operator cmd/operator/main.go

container: build
	docker build -t $(PREFIX)/kafka-operator:$(TAG) .

push: container
	docker push $(PREFIX)/kafka-operator:$(TAG)

deploy: container
	docker build -t $(PREFIX)/kafka-operator:latest .
	docker push $(PREFIX)/kafka-operator:latest
	kubectl apply -f deploy/kafka-operator.yaml

clean:
	rm -f bin/kafka-operator
	kubectl delete -f deploy/kafka-operator.yaml

test: clean
	go test $$(go list ./... | grep -v /vendor/)