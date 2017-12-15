package util

import (
	"github.com/krallistic/kafka-operator/spec"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/api/core/v1"
	k8sclient "k8s.io/client-go/kubernetes"

	"strconv"

	log "github.com/Sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (

	//TODO move default Options to spec
	defaultCPU      = "1"
	defaultDisk     = "100G"
	defaultMemory   = "4Gi"
	stateAnnotation = "kafka-cluster.incubator/state"
)

var (
	logger = log.WithFields(log.Fields{
		"package": "util",
	})
)

type ClientUtil struct {
	KubernetesClient *k8sclient.Clientset
	MasterHost       string
	DefaultOption    metav1.GetOptions
}

func EnrichSpecWithLogger(logger *log.Entry, cluster spec.Kafkacluster) *log.Entry {
	return logger.WithFields(log.Fields{"clusterName": cluster.ObjectMeta.Name, "namespace": cluster.ObjectMeta.Name})
}

func (c *ClientUtil) createLabels(cluster spec.Kafkacluster) map[string]string {
	labels := map[string]string{
		"component": "kafka",
		"creator":   "kafka-operator",
		"role":      "data",
		"name":      cluster.ObjectMeta.Name,
	}
	return labels
}

func (c *ClientUtil) CreateDirectBrokerService(cluster spec.Kafkacluster) error {
	methodLogger := logger.WithFields(log.Fields{
		"method":      "CreateDirectBrokerService",
		"name":        cluster.ObjectMeta.Name,
		"namespace":   cluster.ObjectMeta.Namespace,
		"brokerCount": cluster.Spec.BrokerCount,
	})

	brokerCount := cluster.Spec.BrokerCount
	methodLogger.Info("Creating direkt broker SVCs")

	for i := 0; i < int(brokerCount); i++ {

		serviceName := cluster.ObjectMeta.Name + "-broker-" + strconv.Itoa(i)
		methodLogger.WithFields(log.Fields{
			"id":           i,
			"service_name": serviceName,
		}).Info("Creating Direct Broker SVC: ")

		svc, err := c.KubernetesClient.Core().Services(cluster.ObjectMeta.Namespace).Get(serviceName, c.DefaultOption)
		if err != nil {
			if !errors.IsNotFound(err) {
				methodLogger.WithFields(log.Fields{
					"error": err,
				}).Error("Cant get Service INFO from API")
				return err
			}
		}
		if len(svc.Name) == 0 {
			//Service dosnt exist, creating

			labelSelectors := c.createLabels(cluster)
			labelSelectors["kafka_broker_id"] = strconv.Itoa(i)
			objectMeta := metav1.ObjectMeta{
				Name:        serviceName,
				Namespace:   cluster.ObjectMeta.Namespace,
				Annotations: labelSelectors,
			}

			service := &v1.Service{
				ObjectMeta: objectMeta,
				Spec: v1.ServiceSpec{
					Type:     v1.ServiceTypeNodePort,
					Selector: labelSelectors,
					Ports: []v1.ServicePort{
						v1.ServicePort{
							Name: "broker",
							Port: 9092,
							//NodePort: 30920,
						},
					},
				},
			}
			_, err := c.KubernetesClient.Core().Services(cluster.ObjectMeta.Namespace).Create(service)
			if err != nil {
				methodLogger.WithFields(log.Fields{
					"error":        err,
					"service_name": serviceName,
				}).Error("Error while creating direct broker service")
				return err
			}
			methodLogger.WithFields(log.Fields{
				"service":      service,
				"service_name": serviceName,
			}).Debug("Created direct Access Service")
		}
	}
	return nil
}

func GetBrokerAdressess(cluster spec.Kafkacluster) []string {
	brokers := make([]string, cluster.Spec.BrokerCount)

	//TODO make governing domain config
	dnsSuffix := cluster.ObjectMeta.Name + "." + cluster.ObjectMeta.Namespace + ".svc.cluster.local"
	port := "9092"

	for i := 0; i < int(cluster.Spec.BrokerCount); i++ {
		hostName := cluster.ObjectMeta.Name + "-" + strconv.Itoa(i)
		brokers[i] = hostName + "." + dnsSuffix + ":" + port
	}

	log.WithFields(log.Fields{
		"method":  "GetBrokerAdressess",
		"cluster": cluster.ObjectMeta.Name,
		"broker":  brokers,
	}).Info("Created Broker Adresses")
	return brokers
}
