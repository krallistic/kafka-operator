package util

import (
	"fmt"

	//TODO cleanup dependencies
	//"crypto/tls"
	"github.com/krallistic/kafka-operator/spec"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/apimachinery/pkg/api/resource"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	appsv1Beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"

	log "github.com/Sirupsen/logrus"
	//	"k8s.io/client-go/tools/cache"
)

const (
	tprName     = "kafka.operator.com"
	tprEndpoint = "/apis/extensions/v1beta1/thirdpartyresources"
	defaultCPU  = "1"
	defaultDisk = "100G"
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

func EnrichSpecWithLogger(logger *log.Entry, cluster spec.KafkaCluster) *log.Entry {
	return logger.WithFields(log.Fields{"clusterName": cluster.Metadata.Name, "namespace": cluster.Metadata.Name})
}

func New(kubeConfigFile, masterHost string) (*ClientUtil, error) {
	methodLogger := logger.WithFields(log.Fields{"method": "New"})

	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	client, err := NewKubeClient(kubeConfigFile)
	if err != nil {
		methodLogger.WithFields(log.Fields{
			"error" : err,
			"config" : kubeConfigFile,
			"client" : client,
		}).Error("could not init Kubernetes client")
		return nil, err
	}

	k := &ClientUtil{
		KubernetesClient: client,
		MasterHost:       masterHost,
	}
	methodLogger.WithFields(log.Fields{
		"config" : kubeConfigFile,
		"client" : client,
	}).Debug("Initilized kubernetes cLient")

	return k, nil
}

func BuildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

//TODO refactor for config *rest.Config :)
func NewKubeClient(kubeCfgFile string) (*k8sclient.Clientset, error) {

	config, err := BuildConfig(kubeCfgFile)
	if err != nil {
		return nil, err
	}

	//TODO refactor & log errors
	return k8sclient.NewForConfig(config)
}

func (c *ClientUtil) CreateStorage(cluster spec.KafkaClusterSpec) {
	//for every replica create storage image?
	//except for hostPath, emptyDir
	//let Sts create PV?

}

func (c *ClientUtil) CreateDirectBrokerService(cluster spec.KafkaCluster) error {

	brokerCount := cluster.Spec.BrokerCount
	fmt.Println("Creating N direkt broker SVCs, ", brokerCount)

	for i := 0; i < 3; i++ {
		//TODO name dependend on cluster metadata
		name := "broker-" + string(i)
		fmt.Println("Creating Direct Broker SVC: ", i, name)
		svc, err := c.KubernetesClient.Services(cluster.Metadata.Namespace).Get(name, c.DefaultOption)
		if err != nil {
			return err
		}
		if len(svc.Name) == 0 {
			//Service dosnt exist, creating

			//TODO refactor creation ob object meta out,
			objectMeta := metav1.ObjectMeta{
				Name: name,
				Annotations: map[string]string{
					"component": "kafka",
					"name":      name,
					"role":      "data",
					"type":      "service",
				},
			}
			service := &v1.Service{
				ObjectMeta: objectMeta,
				//TODO label selector
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"component": "kafka",
						"creator":   "kafkaOperator",
						"role":      "data",
						"name":      name,
					},
					Ports: []v1.ServicePort{
						v1.ServicePort{
							Name: "broker",
							Port: 9092,
						},
					},
				},
			}
			_, err := c.KubernetesClient.Services(cluster.Metadata.Namespace).Create(service)
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
func (c *ClientUtil) CreateBrokerService(cluster spec.KafkaCluster, headless bool) error {
	//Check if already exists?
	name := cluster.Metadata.Name
	svc, err := c.KubernetesClient.Services(cluster.Metadata.Namespace).Get(name, c.DefaultOption)
	if err != nil {
		fmt.Println("error while talking to k8s api: ", err)
		//TODO better error handling, global retry module?

	}
	if len(svc.Name) == 0 {
		//Service dosnt exist, creating new.
		fmt.Println("Service dosnt exist, creating new")

		objectMeta := metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"component": "kafka",
				"name":      name,
				"role":      "data",
				"type":      "service",
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
					"creator":   "kafkaOperator",
					"role":      "data",
					"name":      name,
				},
				Ports: []v1.ServicePort{
					v1.ServicePort{
						Name: "broker",
						Port: 9092,
					},
				},
				ClusterIP: "None",
			},
		}
		_, err := c.KubernetesClient.Services(cluster.Metadata.Namespace).Create(service)
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
func (c *ClientUtil) BrokerStatefulSetExist(cluster spec.KafkaCluster) bool {

	statefulSet, err := c.KubernetesClient.StatefulSets(cluster.Metadata.Namespace).Get(cluster.Metadata.Name, c.DefaultOption)
	if err != nil || len(statefulSet.Name) == 0 {
		return false
	}
	return true
}

func (c *ClientUtil) BrokerStSImageUpdate(cluster spec.KafkaCluster) bool {
	kafkaClusterSpec := cluster.Spec

	statefulSet, err := c.KubernetesClient.StatefulSets(cluster.Metadata.Namespace).Get(cluster.Metadata.Name, c.DefaultOption)
	if err != nil {
		fmt.Println("TODO error?")
	}
	//TODO multiple Containers

	if (len(statefulSet.Spec.ServiceName) == 0) && (statefulSet.Spec.Template.Spec.Containers[0].Image != kafkaClusterSpec.Image) {
		return true
	}
	return false
}

func (c *ClientUtil) BrokerStSUpsize(cluster spec.KafkaCluster) bool {
	statefulSet, _ := c.KubernetesClient.StatefulSets(cluster.Metadata.Namespace).Get(cluster.Metadata.Name, c.DefaultOption)
	return *statefulSet.Spec.Replicas < cluster.Spec.BrokerCount
}

func (c *ClientUtil) BrokerStSDownsize(cluster spec.KafkaCluster) bool {
	statefulSet, _ := c.KubernetesClient.StatefulSets(cluster.Metadata.Namespace).Get(cluster.Metadata.Name, c.DefaultOption)
	return *statefulSet.Spec.Replicas > cluster.Spec.BrokerCount
}

func (c *ClientUtil) createStsFromSpec(cluster spec.KafkaCluster) *appsv1Beta1.StatefulSet {
	methodLogger := logger.WithFields(log.Fields{"method": "createStsFromSpec"})
	methodLogger = EnrichSpecWithLogger(methodLogger, cluster)

	name := cluster.Metadata.Name
	replicas := cluster.Spec.BrokerCount
	image := cluster.Spec.Image

	//TODO error handling, default value?
	cpus, err := resource.ParseQuantity(cluster.Spec.Resources.CPU)
	if err != nil {
		cpus, _ = resource.ParseQuantity(defaultCPU)
	}
	diskSpace, err := resource.ParseQuantity(cluster.Spec.Resources.DiskSpace)
	if err != nil {
		diskSpace, _ = resource.ParseQuantity(defaultDisk)
	}

	statefulSet := &appsv1Beta1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"component": "kafka",
				"creator":   "kafkaOperator",
				"role":      "data",
				"name":      name,
			},
		},
		Spec: appsv1Beta1.StatefulSetSpec{
			Replicas: &replicas,

			ServiceName: cluster.Metadata.Name,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"component": "kafka",
						"creator":   "kafkaOperator",
						"role":      "data",
						"name":      name,
					},
				},
				Spec: v1.PodSpec{
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
								v1.WeightedPodAffinityTerm{
									Weight: 50, //TODO flexible weihgt? anti affinity with zK?
									PodAffinityTerm: v1.PodAffinityTerm{
										Namespaces: []string{cluster.Metadata.Namespace},
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: map[string]string{
												"creator": "kafkaOperator",
												"name":    name,
											},
										},
										TopologyKey: "kubernetes.io/hostname", //TODO topologieKey defined somehwere in k8s?
									},
								},
							},
						},
					},
					Tolerations: []v1.Toleration{
						v1.Toleration{
							Key:               "node.alpha.kubernetes.io/unreachable",
							Operator:          v1.TolerationOpExists,
							Effect:            v1.TaintEffectNoExecute,
							TolerationSeconds: &cluster.Spec.MinimumGracePeriod,
						},
						v1.Toleration{
							Key:               "node.alpha.kubernetes.io/notReady",
							Operator:          v1.TolerationOpExists,
							Effect:            v1.TaintEffectNoExecute,
							TolerationSeconds: &cluster.Spec.MinimumGracePeriod,
						},
					},
					InitContainers: []v1.Container{
						v1.Container{
							Name:  "labeler",
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
							Name:  "zookeeper-ready",
							Image: "busybox", //TODO full Name, config
							Command: []string{"sh", "-c", fmt.Sprintf(
								"until nslookup %s; do echo waiting for myservice; sleep 2; done;",
								cluster.Spec.ZookeeperConnect)},
						},
					},

					Containers: []v1.Container{
						v1.Container{
							Name:  "kafka",
							Image: image,
							//TODO String replace operator etc
							Command: []string{"/bin/bash",
								"-c",
								fmt.Sprintf("export KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://$(hostname).%s.$(NAMESPACE).svc.cluster.local:9092; \n"+
									"set -ex\n"+
									"[[ `hostname` =~ -([0-9]+)$ ]] || exit 1\n"+
									"export KAFKA_BROKER_ID=${BASH_REMATCH[1]}\n"+
									"/etc/confluent/docker/run", name),
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
									Value: cluster.Spec.ZookeeperConnect,
								},
							},
							Ports: []v1.ContainerPort{
								v1.ContainerPort{
									Name:          "kafka",
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
					ObjectMeta: metav1.ObjectMeta{
						Name:        "kafka-data",
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

func (c *ClientUtil) UpsizeBrokerStS(cluster spec.KafkaCluster) error {

	statefulSet, err := c.KubernetesClient.StatefulSets(cluster.Metadata.Namespace).Get(cluster.Metadata.Name, c.DefaultOption)
	if err != nil || len(statefulSet.Name) == 0 {
		return err
	}
	statefulSet.Spec.Replicas = &cluster.Spec.BrokerCount
	_, err = c.KubernetesClient.StatefulSets(cluster.Metadata.Namespace).Update(statefulSet)

	if err != nil {
		fmt.Println("Error while updating Broker Count")
	}

	return err
}

func (c *ClientUtil) UpdateBrokerImage(cluster spec.KafkaCluster) error {
	statefulSet, err := c.KubernetesClient.StatefulSets(cluster.Metadata.Namespace).Get(cluster.Metadata.Name, c.DefaultOption)
	if err != nil || len(statefulSet.Name) == 0 {
		return err
	}
	statefulSet.Spec.Template.Spec.Containers[0].Image = cluster.Spec.Image

	_, err = c.KubernetesClient.StatefulSets(cluster.Metadata.Namespace).Update(statefulSet)

	if err != nil {
		fmt.Println("Error while updating Broker Count")
		return err
	}

	return nil
}

func (c *ClientUtil) CreatePersistentVolumes(cluster spec.KafkaCluster) error {
	fmt.Println("Creating Persistent Volumes for KafkaCluster")

	pv, err := c.KubernetesClient.PersistentVolumes().Get("testpv-1", c.DefaultOption)
	if err != nil {
		return err
	}
	if len(pv.Name) == 0 {
		fmt.Println("PersistentVolume dosnt exist, creating")
		new_pv := v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-1",
			},
			Spec: v1.PersistentVolumeSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{
					v1.ReadWriteOnce,
				},
				//Capacity: Reso

			},
		}
		fmt.Println(new_pv)
	}

	return nil

}

func (c *ClientUtil) DeleteKafkaCluster(cluster spec.KafkaCluster) error {

	var gracePeriod int64
	gracePeriod = 10
	//var orphan bool
	//orphan = true
	deleteOption := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	//Delete Services
	err := c.KubernetesClient.Services(cluster.Metadata.Namespace).Delete(cluster.Metadata.Name, &deleteOption)
	if err != nil {
		fmt.Println("Error while deleting Broker Service: ", err)
	}

	statefulSet, err := c.KubernetesClient.StatefulSets(cluster.Metadata.Namespace).Get(cluster.Metadata.Name, c.DefaultOption) //Scaling Replicas down to Zero
	if (len(statefulSet.Name) == 0) && (err != nil) {
		fmt.Println("Error while getting StS from k8s: ", err)
	}

	var replicas int32
	replicas = 0
	statefulSet.Spec.Replicas = &replicas

	_, err = c.KubernetesClient.StatefulSets(cluster.Metadata.Namespace).Update(statefulSet)
	if err != nil {
		fmt.Println("Error while scaling down Broker Sts: ", err)
	}
	//Delete Volumes
	//TODO when volumes are implemented

	//TODO better Error handling
	return err
}

func (c *ClientUtil) CreateBrokerStatefulSet(cluster spec.KafkaCluster) error {

	//Check if sts with Name already exists
	statefulSet, err := c.KubernetesClient.StatefulSets(cluster.Metadata.Namespace).Get(cluster.Metadata.Name, c.DefaultOption)

	if err != nil {
		fmt.Println("Error get sts")
	}
	if len(statefulSet.Name) == 0 {
		fmt.Println("STS dosnt exist, creating")

		statefulSet := c.createStsFromSpec(cluster)

		fmt.Println(statefulSet)
		_, err := c.KubernetesClient.StatefulSets(cluster.Metadata.Namespace).Create(statefulSet)
		if err != nil {
			fmt.Println("Error while creating StatefulSet: ", err)
			return err
		}
	} else {
		fmt.Println("STS already exist.", statefulSet)
	}
	return nil
}
