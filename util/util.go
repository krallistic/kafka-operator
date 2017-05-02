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
	"k8s.io/client-go/pkg/api/resource"
//	"k8s.io/client-go/tools/cache"
//	"github.com/kubernetes/kubernetes/federation/pkg/federation-controller/util"
)

const (
	tprShortName = "kafka-cluster"
	tprSuffix = "incubator.test.com"
	tprFullName = tprShortName + "." + tprSuffix
	tprName = "kafka.operator.com"
	namespace = "default" //TODO flexible NS

	tprEndpoint = "/apis/extensions/v1beta1/thirdpartyresources"
	defaultCPU = "1"
	defaultDisk = "100G"


)

var (
	//TODO make kafkaclusters var
	getEndpoint = fmt.Sprintf("/apis/%s/v1/namespaces/%s/kafkaclusters", tprSuffix,  namespace)
	watchEndpoint = fmt.Sprintf("/apis/%s/v1/watch/namespaces/%s/kafkaclusters", tprSuffix, namespace)
)



type ClientUtil struct {
	KubernetesClient *k8sclient.Clientset
	MasterHost string
	DefaultOption meta_v1.GetOptions

}

func New(kubeConfigFile, masterHost string) (*ClientUtil, error)  {

	client, err := newKubeClient(kubeConfigFile)
	//cache.NewIn

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
		fmt.Println("Using InCluster k8s without a kubeconfig")
		//Depends on k8s env and service account token set.
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
/// Create a the thirdparty ressource inside the Kubernetws Cluster
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
			return err
		}



	} else {
		fmt.Println("TPR already exist")
		//TODO check for correctnes/verison?
	}
	return nil
}


//
func (c *ClientUtil)MonitorKafkaEvents(eventsChannel chan spec.KafkaClusterWatchEvent, errorChannel chan error)  {
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
			fmt.Println("Got Response from WatchEndpoint, parsing now", response)
			fmt.Println("Response Body: ", response.Body)
			decoder := json.NewDecoder(response.Body)
			for {
				var event spec.KafkaClusterWatchEvent
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
}

func (c *ClientUtil) CreateStorage(cluster spec.KafkaClusterSpec) {
	//for every replica create storage image?
	//except for hostPath, emptyDir
	//let Sts create PV?

}

func (c *ClientUtil) CreateDirectBrokerService(cluster spec.KafkaClusterSpec) error {

	brokerCount := cluster.BrokerCount
	fmt.Println("Creating N direkt broker SVCs, ", brokerCount)

	for  i := 0; i < 3 ; i++  {
		//TODO name dependend on cluster metadata
		name := "broker-" + string(i)
		fmt.Println("Creating Direct Broker SVC: ", i, name)
		svc, err := c.KubernetesClient.Services(namespace).Get(name, c.DefaultOption)
		if err != nil {
			return err
		}
		if len(svc.Name) == 0 {
			//Service dosnt exist, creating

			//TODO refactor creation ob object meta out,
			objectMeta := v1.ObjectMeta{
				Name: name,
				Annotations: map[string]string{
					"component": "kafka",
					"name":      name,
					"role": "data",
					"type": "service",
				},
			}
			service := &v1.Service{
				ObjectMeta: objectMeta,
				//TODO label selector
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
				},
			}
			_, err := c.KubernetesClient.Services(namespace).Create(service)
			if err != nil {
				fmt.Println("Error while creating direct service: ", err)
				return err
			}
			fmt.Println(service)

		}

	}



	return nil
}

//TODO refactor, into headless svc and direct svc
func (c *ClientUtil) CreateBrokerService(newSpec spec.KafkaClusterSpec, headless bool) error {
	//Check if already exists?
	name := newSpec.Name
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


//TODO caching of the STS
func (c *ClientUtil) BrokerStatefulSetExist(kafkaClusterSpec spec.KafkaClusterSpec) bool {
	statefulSet, err := c.KubernetesClient.StatefulSets(namespace).Get(kafkaClusterSpec.Name, c.DefaultOption)
	if err != nil ||  len(statefulSet.Name) == 0 {
		return false
	}
	return true
}

func (c *ClientUtil) BrokerStSImageUpdate(kafkaClusterSpec spec.KafkaClusterSpec) bool {
	statefulSet, err := c.KubernetesClient.StatefulSets(namespace).Get(kafkaClusterSpec.Name, c.DefaultOption)
	if err != nil {
		fmt.Println("TODO error?")
	}
	//TODO multiple Containers

	if (len(statefulSet.Spec.ServiceName) == 0) && (statefulSet.Spec.Template.Spec.Containers[0].Image != kafkaClusterSpec.Image) {
		return true
	}
	return false
}

func (c *ClientUtil) BrokerStSUpsize(newSpec spec.KafkaClusterSpec) bool {
	statefulSet, _ := c.KubernetesClient.StatefulSets(namespace).Get(newSpec.Name, c.DefaultOption)
	return *statefulSet.Spec.Replicas < newSpec.BrokerCount
}

func (c *ClientUtil) BrokerStSDownsize(newSpec spec.KafkaClusterSpec) bool {
	statefulSet, _ := c.KubernetesClient.StatefulSets(namespace).Get(newSpec.Name, c.DefaultOption)
	return *statefulSet.Spec.Replicas > newSpec.BrokerCount
}

func (c *ClientUtil) createStsFromSpec(kafkaClusterSpec spec.KafkaClusterSpec) *appsv1Beta1.StatefulSet {

	name := kafkaClusterSpec.Name
	replicas := kafkaClusterSpec.BrokerCount
	image := kafkaClusterSpec.Image

	//TODO error handling, default value?
	cpus, err := resource.ParseQuantity(kafkaClusterSpec.Resources.CPU)
	if err != nil {
		cpus, _ = resource.ParseQuantity(defaultCPU)
	}
	diskSpace, err := resource.ParseQuantity(kafkaClusterSpec.Resources.DiskSpace)
	if err != nil {
		diskSpace, _ = resource.ParseQuantity(defaultDisk)
	}

	statefulSet := &appsv1Beta1.StatefulSet{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"component": "kafka",
				"creator": "kafkaOperator",
				"role":      "data",
				"name": name,
			},
		},
		Spec: appsv1Beta1.StatefulSetSpec{
			Replicas: &replicas,

			ServiceName: kafkaClusterSpec.Name,
			Template: v1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"component": "kafka",
						"creator": "kafkaOperator",
						"role": "data",
						"name": name,
					},
				},
				Spec:v1.PodSpec{
					InitContainers: []v1.Container{
						v1.Container{
							Name: "labeler",
							Image: "devth/k8s-labeler", //TODO fullName, config
							Env: []v1.EnvVar{
								v1.EnvVar{
									Name: "KUBE_NAMESPACE",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								v1.EnvVar{
									Name: "KUBE_LABEL_hostname",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
						},
						v1.Container{
							Name: "zookeeper-ready",
							Image: "busybox", //TODO full Name, config
							Command: []string{"sh", "-c", fmt.Sprintf(
								"until nslookup %s; do echo waiting for myservice; sleep 2; done;",
								kafkaClusterSpec.ZookeeperConnect)},
						},
					},

					Containers: []v1.Container{
						v1.Container{
							Name: "kafka",
							Image: image,
							//TODO String replace operator etc
							Command: []string{"/bin/bash",
									  "-c",
								fmt.Sprintf("export KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://$(hostname).%s.$(NAMESPACE).svc.cluster.local:9092; \n" +
									"set -ex\n" +
									"[[ `hostname` =~ -([0-9]+)$ ]] || exit 1\n" +
									"export KAFKA_BROKER_ID=${BASH_REMATCH[1]}\n" +
									"/etc/confluent/docker/run",name),
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
							},
							Ports: []v1.ContainerPort{
								v1.ContainerPort{
									Name: "kafka",
									ContainerPort: 9092,
								},
							},
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU: cpus,
								},
							},

						},
					},
				},


			},
			VolumeClaimTemplates: []v1.PersistentVolumeClaim{
				v1.PersistentVolumeClaim{
					ObjectMeta: v1.ObjectMeta{
						Name: "kafka-data",
						Annotations: map[string]string{
							//TODO make storageClass Optinal
							//"volume.beta.kubernetes.io/storage-class": "anything",
						},
					},
					Spec: v1.PersistentVolumeClaimSpec{
						AccessModes: []v1.PersistentVolumeAccessMode{
							v1.ReadWriteOnce,
						},
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: diskSpace,
							},
						},
					},
				},
			},
		},
	}
	return statefulSet
}


func (c *ClientUtil) UpsizeBrokerStS(newSpec spec.KafkaClusterSpec) error {

	statefulSet, err := c.KubernetesClient.StatefulSets(namespace).Get(newSpec.Name, c.DefaultOption)
	if err != nil ||  len(statefulSet.Name) == 0 {
		return err
	}
	statefulSet.Spec.Replicas = &newSpec.BrokerCount
	_ ,err = c.KubernetesClient.StatefulSets(namespace).Update(statefulSet)

	if err != nil {
		fmt.Println("Error while updating Broker Count")
	}

	return err
}

func (c *ClientUtil) UpdateBrokerImage(newSpec spec.KafkaClusterSpec) error {
	statefulSet, err := c.KubernetesClient.StatefulSets(namespace).Get(newSpec.Name, c.DefaultOption)
	if err != nil ||  len(statefulSet.Name) == 0 {
		return err
	}
	statefulSet.Spec.Template.Spec.Containers[0].Image = newSpec.Image

	_ ,err = c.KubernetesClient.StatefulSets(namespace).Update(statefulSet)

	if err != nil {
		fmt.Println("Error while updating Broker Count")
		return err
	}

	return nil
}

func (c *ClientUtil) CreatePersistentVolumes(kafkaClusterSpec spec.KafkaClusterSpec) error{
	fmt.Println("Creating Persistent Volumes for KafkaCluster")

	pv, err := c.KubernetesClient.PersistentVolumes().Get("testpv-1", c.DefaultOption)
	if err != nil  {
		return err
	}
	if len(pv.Name) == 0 {
		fmt.Println("PersistentVolume dosnt exist, creating")
		new_pv := v1.PersistentVolume{
			ObjectMeta: v1.ObjectMeta{
				Name:"test-1",
			},
			Spec: v1.PersistentVolumeSpec{
				AccessModes:[]v1.PersistentVolumeAccessMode{
					v1.ReadWriteOnce,
				},
				//Capacity: Reso

			},

		}
		fmt.Println(new_pv)
	}


	return nil

}

func (c *ClientUtil) DeleteKafkaCluster(oldSpec spec.KafkaClusterSpec) error {

	var gracePeriod int64
	gracePeriod = 10
	//var orphan bool
	//orphan = true
	deleteOption := v1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	//Delete Services
	err := c.KubernetesClient.Services(namespace).Delete(oldSpec.Name, &deleteOption)
	if err != nil {
		fmt.Println("Error while deleting Broker Service: ", err)
	}

	statefulSet, err := c.KubernetesClient.StatefulSets(namespace).Get(oldSpec.Name, c.DefaultOption)//Scaling Replicas down to Zero
	if (len(statefulSet.Name) == 0 ) && ( err != nil) {
		fmt.Println("Error while getting StS from k8s: ", err)
	}

	var replicas int32
	replicas = 0
	statefulSet.Spec.Replicas = &replicas

	_, err = c.KubernetesClient.StatefulSets(namespace).Update(statefulSet)
	if err != nil {
		fmt.Println("Error while scaling down Broker Sts: ", err)
	}
	//TODO maybe sleep, yes we need a sleep
	//err =  c.KubernetesClient.StatefulSets(namespace).Delete(oldSpec.Name, &deleteOption)
	//if err != nil {
	//	fmt.Println("Error while deleting sts")
	//}
	//Delete Volumes
	//TODO when volumes are implemented


	//TODO better Error handling
	return err
}


func (c *ClientUtil) CreateBrokerStatefulSet(kafkaClusterSpec spec.KafkaClusterSpec) error {

	//Check if sts with Name already exists
	statefulSet, err := c.KubernetesClient.StatefulSets(namespace).Get(kafkaClusterSpec.Name, c.DefaultOption)


	//TODO dont use really a sts set, instead use just PODs? More control over livetime (aka downscaling which) upscaling etc..but more effort?
	if err != nil {
		fmt.Println("Error get sts")
	}
	if len(statefulSet.Name) == 0 {
		fmt.Println("STS dosnt exist, creating")

		statefulSet := c.createStsFromSpec(kafkaClusterSpec)

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