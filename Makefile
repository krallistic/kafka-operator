# Makefile for the Docker image
# MAINTAINER: Jakob Karalus

.PHONY: all build container push deploy clean test

TAG ?= v0.2.5-dirty-5
PREFIX ?= krallistic

all: push images

build: test
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/kafka_operator cmd/operator/main.go

container: build
	docker build -t $(PREFIX)/kafka-operator:$(TAG) .
	docker build -t $(PREFIX)/kafka-operator:latest .

image-cc: 
	docker build -t $(PREFIX)/cruise-control:latest images/cruise-control/
	docker push $(PREFIX)/cruise-control:latest


images:
	docker build -t $(PREFIX)/kafka-cc-reporter:latest images/kafka/
	docker push $(PREFIX)/kafka-cc-reporter:latest

push: container
	docker push $(PREFIX)/kafka-operator:$(TAG)
	docker push $(PREFIX)/kafka-operator:latest
	docker push $(PREFIX)/kafka-operator:$(TAG) 

deploy: container
	docker build -t $(PREFIX)/kafka-operator:latest .
	docker push $(PREFIX)/kafka-operator:latest
	docker push $(PREFIX)/kafka-operator:$(TAG) 
	kubectl delete -f deploy/kafka-operator.yaml
	sleep 5
	# export TAG=$(TAG)
	# TAG=$(TAG) envsubst < deploy/kafka-operator.yaml.tpl > deploy/kafka-operator.yaml
	# sed -e "s/\${TAG}/$(TAG)/" template.txt deploy/kafka-operator.yaml.tpl 
	kubectl apply -f deploy/kafka-operator.yaml

clean:
	rm -f bin/kafka-operator
	kubectl delete -f deploy/kafka-operator.yaml

test: 
	go test $$(go list ./... | grep -v /vendor/)