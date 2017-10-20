FROM openjdk:8-alpine AS BUILD_IMAGE

ENV APP_HOME=/root/dev/
WORKDIR $APP_HOME

RUN apk --no-cache add gettext git bash
#RUN apt-get update && apt-get install -y gettext git
RUN git clone https://github.com/linkedin/cruise-control.git
WORKDIR $APP_HOME/cruise-control

COPY cruisecontrol.properties.tpl config/cruisecontrol.properties.tpl
COPY setup-cruise-control.sh setup-cruise-control.sh
RUN chmod +x setup-cruise-control.sh
RUN ./gradlew jar copyDependantLibs
RUN chmod +x kafka-cruise-control-start.sh

CMD ["./setup-cruise-control.sh"]