package cruisecontrol

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
)

func CallCruiseControl(url string, action string, options map[string]string) (*http.Response, error) {
	methodLogger := log.WithFields(log.Fields{
		"method":         "callCruiseControl",
		"request-url":    url,
		"request_values": options,
		"action":         action,
	})
	methodLogger.Info("Calling Cruise Control")

	ccBasePath := "kafkacruisecontrol"
	optionURL := ""
	for option, value := range options {
		optionURL = optionURL + option + "=" + value + "&&"
	}

	requestURl := url + "/" + ccBasePath + "/" + action + "?" + optionURL
	rsp, err := http.Post(requestURl, "text/plain", nil)
	if err != nil {
		methodLogger.Error("Error while talking to cruise-control")
		return nil, err
	}

	return rsp, nil
}
