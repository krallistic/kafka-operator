package kafka

import (
	"fmt"
	"strconv"

	"github.com/krallistic/kafka-operator/kube"
	"github.com/krallistic/kafka-operator/spec"

	appsv1Beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultCPU    = "1"
	defaultDisk   = "100G"
	defaultMemory = "4Gi"
)

func createLabels(cluster spec.Kafkacluster) map[string]string {
	labels := map[string]string{
		"component": "kafka",
		"creator":   "kafka-operator",
		"role":      "data",
		"name":      cluster.ObjectMeta.Name,
	}
	return labels
}

func generateKafkaStatefulset(cluster spec.Kafkacluster) *appsv1Beta1.StatefulSet {

	name := cluster.ObjectMeta.Name
	replicas := cluster.Spec.BrokerCount
	image := cluster.Spec.Image

	storageClass := "standard"
	if cluster.Spec.StorageClass != "" {
		storageClass = cluster.Spec.StorageClass
	}

	//TODO error handling, default value?
	cpus, err := resource.ParseQuantity(cluster.Spec.Resources.CPU)
	if err != nil {
		cpus, _ = resource.ParseQuantity(defaultCPU)
	}

	memory, err := resource.ParseQuantity(cluster.Spec.Resources.Memory)
	if err != nil {
		memory, _ = resource.ParseQuantity(defaultMemory)
	}

	heapsize := int64(float64(memory.ScaledValue(resource.Mega)) * 0.6)
	if heapsize > 4096 {
		heapsize = 4096
	}

	fmt.Println(memory)

	diskSpace, err := resource.ParseQuantity(cluster.Spec.Resources.DiskSpace)
	if err != nil {
		diskSpace, _ = resource.ParseQuantity(defaultDisk)
	}

	options := GenerateKafkaOptions(cluster)

	statefulSet := &appsv1Beta1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: createLabels(cluster),
		},
		Spec: appsv1Beta1.StatefulSetSpec{
			Replicas: &replicas,

			ServiceName: cluster.ObjectMeta.Name,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: createLabels(cluster),
				},
				Spec: v1.PodSpec{
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
								v1.WeightedPodAffinityTerm{
									Weight: 50, //TODO flexible weihgt? anti affinity with zK?
									PodAffinityTerm: v1.PodAffinityTerm{
										Namespaces: []string{cluster.ObjectMeta.Namespace},
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: createLabels(cluster),
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
							Command: []string{"/bin/bash",
								"-c",
								fmt.Sprintf(
									"set -ex\n" +
										"[[ `hostname` =~ -([0-9]+)$ ]] || exit 1\n" +
										"export KUBE_LABEL_kafka_broker_id=${BASH_REMATCH[1]}\n" +
										"/run.sh"),
							},
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
								v1.EnvVar{
									Name:  "KUBE_LABEL_kafka_broker_id",
									Value: "thisshouldbeoverwritten",
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
							Env: options,
							Ports: []v1.ContainerPort{
								v1.ContainerPort{
									Name: "kafka",
									//TODO configPort
									ContainerPort: 9092,
								},
							},
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    cpus,
									v1.ResourceMemory: *GetMaxHeap(cluster),
								},
								Limits: v1.ResourceList{
									v1.ResourceMemory: memory,
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []v1.PersistentVolumeClaim{
				v1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kafka-data",
						Annotations: map[string]string{
							//TODO storagClass field in never Versions.
							"volume.beta.kubernetes.io/storage-class": storageClass,
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

func generateHeadlessService(cluster spec.Kafkacluster) *v1.Service {
	labelSelectors := createLabels(cluster)

	objectMeta := metav1.ObjectMeta{
		Name:        cluster.ObjectMeta.Name,
		Annotations: labelSelectors,
	}

	objectMeta.Labels = map[string]string{
		"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
	}

	service := &v1.Service{
		ObjectMeta: objectMeta,

		Spec: v1.ServiceSpec{
			Selector: labelSelectors,
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name: "broker",
					Port: 9092,
				},
			},
			ClusterIP: "None",
		},
	}

	return service
}

func generateDirectBrokerServices(cluster spec.Kafkacluster) []*v1.Service {
	var services []*v1.Service

	for i := 0; i < int(cluster.Spec.BrokerCount); i++ {
		serviceName := cluster.ObjectMeta.Name + "-broker-" + strconv.Itoa(i)

		labelSelectors := createLabels(cluster)
		labelSelectors["kafka_broker_id"] = strconv.Itoa(i)
		objectMeta := metav1.ObjectMeta{
			Name:        serviceName,
			Namespace:   cluster.ObjectMeta.Namespace,
			Annotations: labelSelectors,
		}

		service := &v1.Service{
			ObjectMeta: objectMeta,
			Spec: v1.ServiceSpec{
				Type:     v1.ServiceTypeClusterIP,
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
		services = append(services, service)
	}

	return services
}

func DeleteCluster(cluster spec.Kafkacluster, client kube.Kubernetes) error {
	cluster.Spec.BrokerCount = 0
	sts := generateKafkaStatefulset(cluster)
	//Downsize Statefulset to 0
	err := client.CreateOrUpdateStatefulSet(sts)
	if err != nil {
		return err
	}
	//Delete Headless SVC
	headlessSVC := generateHeadlessService(cluster)
	err = client.DeleteService(headlessSVC)
	if err != nil {
		return err
	}

	//Delete Direct Broker SVCs
	svcs := generateDirectBrokerServices(cluster)
	for _, svc := range svcs {
		err = client.DeleteService(svc)
		if err != nil {
			return err
		}
	}

	//Force Delete of Statefulset
	err = client.DeleteStatefulset(sts)
	if err != nil {
		return err
	}

	return nil
}

func CreateCluster(cluster spec.Kafkacluster, client kube.Kubernetes) error {
	//Create Headless SVC
	headlessSVC := generateHeadlessService(cluster)
	err := client.CreateOrUpdateService(headlessSVC)
	if err != nil {
		return err
	}

	sts := generateKafkaStatefulset(cluster)
	//Create Broker Cluster
	err = client.CreateOrUpdateStatefulSet(sts)
	if err != nil {
		return err
	}

	//CreateDelete Direct Broker SVCs
	svcs := generateDirectBrokerServices(cluster)
	for _, svc := range svcs {
		err = client.CreateOrUpdateService(svc)
		if err != nil {
			return err
		}
	}

	return nil
}

func UpsizeCluster(cluster spec.Kafkacluster, client kube.Kubernetes) error {
	return nil
}

func DownsizeCluster(cluster spec.Kafkacluster, client kube.Kubernetes) error {
	return nil
}

func UpdateStatus(cluster spec.Kafkacluster, client kube.Kubernetes) error {
	return nil
}
