package util

import (
	k8sclient "k8s.io/client-go/kubernetes"

	"fmt"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/rest"
	meta_v1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/api/v1"
	appsv1Beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"
	"crypto/tls"
	"github.com/krallistic/kafka-operator/spec"
	"net/http"
	"time"
	"encoding/json"
)

const (
	tprShortName = "kafka-cluster"
	tprSuffix = "incubator.test.com"
	tprFullName = tprShortName + "." + tprSuffix
	tprName = "kafka.operator.com"
	namespace = "default" //TODO flexible NS

	tprEndpoint = "/apis/extensions/v1beta1/thirdpartyresources"
)

var (
	//TODO make kafkaclusters var
	getEndpoint = fmt.Sprintf("/apis/%s/v1/namespaces/%s/kafkaclusters", tprSuffix,  namespace)
	watchEndpoint = fmt.Sprintf("/apis/%s/v1/watch/namespaces/%s/kafkaclusters", tprSuffix, namespace)
)


type KafkaClusterWatchEvent struct {
	Type string `json:"type"`
	Object spec.KafkaCluster `json:"object"`
}


type ClientUtil struct {
	KubernetesClient *k8sclient.Clientset
	MasterHost string
	DefaultOption meta_v1.GetOptions
}

func New(kubeConfigFile, masterHost string) (*ClientUtil, error)  {

	client, err := newKubeClient(kubeConfigFile)

	if err != nil {
		fmt.Println("Error, could not Init Kubernetes Client")
		return nil, err
	}

	k := &ClientUtil{
		KubernetesClient: client,
		MasterHost: masterHost,
	}
	fmt.Println("Initilized k8s CLient")
	return k, nil

}


func newKubeClient(kubeCfgFile string) (*k8sclient.Clientset, error) {

	var client *k8sclient.Clientset

	// Should we use in cluster or out of cluster config
	if len(kubeCfgFile) == 0 {
		fmt.Println("Using InCluster k8s config")
		cfg, err := rest.InClusterConfig()

		if err != nil {
			return nil, err
		}

		client, err = k8sclient.NewForConfig(cfg)

		if err != nil {
			return nil, err
		}
	} else {
		fmt.Println("Using OutOfCluster k8s config with kubeConfigFile: %s", kubeCfgFile)
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeCfgFile)

		if err != nil {
			fmt.Println("Got error trying to create client: ", err)
			return nil, err
		}

		client, err = k8sclient.NewForConfig(cfg)

		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

func (c *ClientUtil) GetKafkaClusters() ([]spec.KafkaCluster, error) {
	//var resp *http.Response
	var err error

	transport := &http.Transport{ TLSClientConfig: &tls.Config{InsecureSkipVerify:true} }

	//We go over the http because go client cant do tpr?
	httpClient := http.Client{Transport: transport}
	response, err := httpClient.Get(c.MasterHost + getEndpoint)
	if err != nil {
		fmt.Println("Error while getting resonse from API: ", err)
		return nil, err
	}
	fmt.Println("GetKafaCluster API Response: ", response)

	return nil, nil
}

func (c *ClientUtil)CreateKubernetesThirdPartyResource() error  {
	tprResult, _ := c.KubernetesClient.ThirdPartyResources().Get("kafkaCluster", c.DefaultOption)
	if len(tprResult.Name) == 0 {
		fmt.Println("No KafkaCluster TPR found, creating...")

		tpr := &v1beta1.ThirdPartyResource{
			ObjectMeta: v1.ObjectMeta{
				Name: tprFullName,
			},
			Versions: []v1beta1.APIVersion{
				{Name: "v1"},
			},
			Description: "Managed apache kafke clusters",
		}
		fmt.Println("Creating TPR: ", tpr)
		retVal, err := c.KubernetesClient.ThirdPartyResources().Create(tpr)
		fmt.Println("retVal: ", retVal)
		if err != nil {
			fmt.Println("Error creating TPR: ", err)
		}

		//TODO Error checking


	} else {
		fmt.Println("TPR already exist")
		//TODO check for correctnes/verison?
	}
	return nil
}

func (c *ClientUtil)MonitorKafkaEvents() (<-chan KafkaClusterWatchEvent, <-chan error) {
	errorChannel := make(chan error, 1)
	eventsChannel := make(chan KafkaClusterWatchEvent)

	go func() {
		for {
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{Transport: tr}
			fmt.Println("Trying API: ", c.MasterHost + watchEndpoint)
			response, err := client.Get(c.MasterHost + watchEndpoint)
			if err != nil {
				fmt.Println("Error reading API:" , err , response)
				errorChannel <- err
				time.Sleep(2 * time.Second)
				continue
			}
			fmt.Println("Got Response from WathEndpoint, parsing now", response)
			decoder := json.NewDecoder(response.Body)
			for {
				var event KafkaClusterWatchEvent
				err = decoder.Decode(&event)
				if err != nil {
					fmt.Println("Error decoding response ", err)
					errorChannel <- err
					break
				}
				fmt.Println("Parsed KafkaWatch Event ", event)
				eventsChannel <- event
			}

			time.Sleep(2 * time.Second)
		}
	}()


	return eventsChannel, errorChannel
}

func (c *ClientUtil) CreateBrokerService(name string, headless bool) error {
	//Check if already exists?
	svc, err := c.KubernetesClient.Services(namespace).Get(name, c.DefaultOption)
	if err != nil {
		fmt.Println("error while talking to k8s api: ", err)
		//TODO better error handling, global retry module?

	}
	if len(svc.Name) == 0 {
		//Service dosnt exist, creating new.
		fmt.Println("Service dosnt exist, creating new")



		objectMeta := v1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"component": "kafka",
				"name":      name,
				"role": "data",
				"type": "service",
			},
		}

		if headless == true {
			objectMeta.Labels = map[string]string{
				"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
			}
			objectMeta.Name = name
		}


		service := &v1.Service{
			ObjectMeta: objectMeta,

			Spec: v1.ServiceSpec{
				Selector: map[string]string{
					"component": "kafka",
					"creator": "kafkaOperator",
					"role":      "data",
					"name": name,
				},
				Ports: []v1.ServicePort{
					v1.ServicePort{
						Name:     "broker",
						Port:     9092,
					},
				},
				ClusterIP: "None",
			},
		}
		_, err := c.KubernetesClient.Services(namespace).Create(service)
		if err != nil {
			fmt.Println("Error while creating Service: ", err)
		}
		fmt.Println(service)
	} else {
		//Service exist
		fmt.Println("Headless Broker SVC already exists: ", svc)
		//TODO maybe check for correct service?
	}


	return nil
}


func (c *ClientUtil) CreateBrokerStatefulSet(kafkaClusterSpec spec.KafkaClusterSpec) error {


	name := kafkaClusterSpec.Name
	replicas := kafkaClusterSpec.Brokers.Count
	image := kafkaClusterSpec.Image

	//Check if sts with Name already exists
	statefulSet, err := c.KubernetesClient.StatefulSets(namespace).Get(name, c.DefaultOption)

	if err != nil {
		fmt.Println("Error get sts")
	}
	if len(statefulSet.Name) == 0 {
		fmt.Println("STS dosnt exist, creating")

		statefulSet := &appsv1Beta1.StatefulSet{
			ObjectMeta: v1.ObjectMeta{
				Name: "kafka",
				Labels: map[string]string{
					"component": "kafka",
					"creator": "kafkaOperator",
					"role":      "data",
					"name": name,
				},
			},
			Spec: appsv1Beta1.StatefulSetSpec{
				Replicas: &replicas,

				ServiceName: kafkaClusterSpec.Name, //TODO variable svc name, or depnedent on soemthing
				Template: v1.PodTemplateSpec{
					ObjectMeta: v1.ObjectMeta{
						Labels: map[string]string{
							"component": "kafka",
							"creator": "kafkaOperator",
							"role": "data",
							"name": name,
						},
						Annotations: map[string]string{
							"pod.beta.kubernetes.io/init-containers": "[ " +
								"]",
						},
					},

					Spec:v1.PodSpec{

						Containers: []v1.Container{
							v1.Container{
								Name: "kafka",
								Image: image,
								//TODO String replace operator etc
								Command: []string{"/bin/bash",
									"-c",
									"export KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://$(hostname).operator.$(NAMESPACE).svc.cluster.local:9092; \n" +
									"set -ex\n" +
									"[[ `hostname` =~ -([0-9]+)$ ]] || exit 1\n" +
									"export KAFKA_BROKER_ID=${BASH_REMATCH[1]}\n" +
									"/etc/confluent/docker/run",
									},
								Env: []v1.EnvVar{
									v1.EnvVar{
										Name: "NAMESPACE",
										ValueFrom: &v1.EnvVarSource{
											FieldRef: &v1.ObjectFieldSelector{
												FieldPath: "metadata.namespace",
											},
										},
									},
									v1.EnvVar{
										Name:  "KAFKA_ZOOKEEPER_CONNECT",
										Value: kafkaClusterSpec.ZookeeperConnect,
									},
									//v1.EnvVar{
									//	Name:  "KAFKA_ADVERTISED_LISTENERS",
									//	Value: "PLAINTEXT://kafka-0." + kafkaClusterSpec.Name + ".default.svc" + ".cluster.local:9092",
									//	//TODO not static, genererate Name, or use ENV VAR?
									//},
									v1.EnvVar{
										Name:  "KAFKA_BROKER_ID",
										Value: "1", //TODO getOrdinal? Can be a String? Hostname?

									},
									v1.EnvVar{
										Name: "TEST",
										Value: "$HOSTNAME.operator.$NAMESPACE.svc.cluster.local:9092",
									},
								},
								Ports: []v1.ContainerPort{
									v1.ContainerPort{
										Name: "kafka",
										ContainerPort: 9092,
									},
								},

							},
						},
					},
				},
			},
		}

		fmt.Println(statefulSet)
		_, err := c.KubernetesClient.StatefulSets(namespace).Create(statefulSet)
		if err != nil {
			fmt.Println("Error while creating StatefulSet: ", err) //TODO what to do with error? If we track State Internally we can do a reconcilidation which would force a recreate
		}
	} else {
		fmt.Println("STS already exist. TODO what to do now?", statefulSet)
	}
	return nil
}