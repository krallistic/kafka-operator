FROM alpine:3.6
MAINTAINER  Jakob Karalus <jakob.karalus@gnx.net>

ADD bin/kafka_operator /bin/usr/sbin/kafka_operator

CMD ["/bin/usr/sbin/kafka_operator"]