FROM alpine
MAINTAINER  Jakob Karalus <jakob.karalus@gnx.net>

ADD bin/kafka-operator /usr/local/bin

CMD ["/bin/sh", "-c", "/usr/local/bin/kafka-operator"]
