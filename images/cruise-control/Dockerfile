FROM openjdk:8 AS BUILD_IMAGE
ENV APP_HOME=/root/dev/
WORKDIR $APP_HOME
RUN git clone https://github.com/linkedin/cruise-control.git
WORKDIR $APP_HOME/cruise-control
RUN ./gradlew jar copyDependantLibs
RUN chmod +x kafka-cruise-control-start.sh
CMD ["./kafka-cruise-control-start.sh", "config/cruisecontrol.properties", "9095"]