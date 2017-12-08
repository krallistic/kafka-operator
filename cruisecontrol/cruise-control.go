package cruisecontrol

import (
	"net/http"

	"io/ioutil"

	"errors"

	"github.com/krallistic/kafka-operator/spec"

	cruisecontrol_kube "github.com/krallistic/kafka-operator/kube/cruisecontrol"

	log "github.com/Sirupsen/logrus"
)

type CruiseControlExecuterState int32
type CruiseControlMonitorState int32

type CruiseControlState struct {
	MonitorState      CruiseControlMonitorState
	ExecuterState     CruiseControlExecuterState
	ReplicasToMove    int
	FinshedReplicas   int
	BootstrapProgress int
	ValidPartitions   int
	TotalPartitions   int
	ProposalReady     bool
}

const (
	basePath       = "kafkacruisecontrol"
	statusBasePath = "status"

	removeBrokerAction = "remove_broker"
	addBrokerAction    = "add_broker"

	NO_TASK = iota + 1
	EXECUTION_STARTED
	REPLICA_MOVEMENT_IN_PROGRESS
	LEADER_MOVEMENT_IN_PROGRESS
)

func GetCruiseControlStatus(url string) (string, error) {
	methodLogger := log.WithFields(log.Fields{
		"method": "GetCruiseControlStatus",
	})
	methodLogger.Info("Getting Cruise Control Status")

	requestURl := url + "/" + statusBasePath
	rsp, err := http.Get(requestURl)
	if err != nil {
		return "nil", err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		methodLogger.WithField("response", rsp).Warn("Got non 200 response from CruiseControl while reading state")
		return "nil", errors.New("Non 200 error code from cruise-control while reading state: " + rsp.Status)
	}
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		methodLogger.WithField("error", err).Error("Error while reading response body from cruisecontrol state")
		return "nil", err
	}
	sBody := string(body)

	//TODO parse sBody Respone

	return sBody, nil
}

func DownsizeCluster(cluster spec.Kafkacluster, brokerToDelete string) error {
	methodLogger := log.WithFields(log.Fields{
		"method":    "DownsizeCluster",
		"name":      cluster.ObjectMeta.Name,
		"namespace": cluster.ObjectMeta.Namespace,
	})
	//TODO generate Cluster CruiseControl Service URL
	//

	cruiseControlURL := "http://" + cruisecontrol_kube.GetCruiseControlName(cluster) + "." + cluster.ObjectMeta.Namespace + ".svc.cluster.local:9095"

	options := map[string]string{
		"brokerid": brokerToDelete,
		"dryrun":   "false",
	}

	rsp, err := postCruiseControl(cruiseControlURL, removeBrokerAction, options)
	if err != nil {
		methodLogger.Error("Cant downsize cluster since post to cc failed")
		//TODO do-something?
		return err
	}
	methodLogger.WithField("response", rsp).Info("Initiated Dowsize to cruise control")
	return nil
}

func postCruiseControl(url string, action string, options map[string]string) (*http.Response, error) {
	methodLogger := log.WithFields(log.Fields{
		"method":         "callCruiseControl",
		"request-url":    url,
		"request_values": options,
		"action":         action,
	})
	methodLogger.Info("Calling Cruise Control")

	optionURL := ""
	for option, value := range options {
		optionURL = optionURL + option + "=" + value + "&&"
	}

	requestURl := url + "/" + basePath + "/" + action + "?" + optionURL
	rsp, err := http.Post(requestURl, "text/plain", nil)
	if err != nil {
		methodLogger.WithField("error", err).Error("Error while talking to cruise-control")
		return nil, err
	}
	if rsp.StatusCode != 200 {
		methodLogger.WithField("response", rsp).Warn("Got non 200 response from CruiseControl")
		return nil, errors.New("Non 200 response from cruise-control: " + rsp.Status)
	}

	return rsp, nil
}
