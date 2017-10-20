envsubst < config/cruisecontrol.properties.tpl > config/cruisecontrol.properties
cat config/cruisecontrol.properties
echo "Rendered Config"
./kafka-cruise-control-start.sh "$@"