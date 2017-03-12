FROM golang:alpine
MAINTAINER  Jakob Karalus <jakob.karalus@gnx.net>


#TODO move go build ouside docker, maybe full travis integration.
RUN mkdir -p /go/src /go/bin && chmod -R 777 /go
ADD . /go/src/github.com/krallistic/kafka-operator/
WORKDIR /go/src/github.com/krallistic/kafka-operator/
RUN go build -o operator cmd/operator/main.go
CMD ["/go/src/github.com/krallistic/kafka-operator/operator"]
