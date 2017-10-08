FROM openjdk:8 AS BUILD_IMAGE
ENV APP_HOME=/root/dev/cruise-control
WORKDIR $APP_HOME
RUN git clone https://github.com/linkedin/cruise-control.git
WORKDIR $APP_HOME/cruise-control
RUN git checkout ff461d1288c76c4ab8c41d21e6303fef06872e04 .
RUN ./gradlew jar

FROM confluentinc/cp-kafka:latest
WORKDIR /root/
COPY --from=BUILD_IMAGE /root/dev/cruise-control/cruise-control/cruise-control-metrics-reporter/build/libs/cruise-control-metrics-reporter.jar /usr/share/java/kafka/
CMD ["/etc/confluent/docker/run"]