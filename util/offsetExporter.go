package util

import (
	"github.com/krallistic/kafka-operator/spec"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1Beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"

	"k8s.io/client-go/pkg/api/v1"
)

const (
	deplyomentPrefix = "kafka-offset-checker"
	offsetExporterImage = "krallistic/kafka_offset_exporter" //TODO
	offsetExporterVersion = "latest" //TODO make version cmd arg

	prometheusScrapeAnnotation = "prometheus.io/scrape"
	prometheusPortAnnotation = "prometheus.io/port"
	prometheusPathAnnotation = "prometheus.io/path"

	metricPath = "/metrics"
	metricsPort = "10250"
	metricsScrape = "true"
)

func (c *ClientUtil) getOffsetMonitorName(cluster spec.KafkaCluster) string {
	return deplyomentPrefix + "-" + cluster.Metadata.Name
}

// Deploys the OffsetMonitor as an extra Pod inside the Cluster
func (c *ClientUtil) DeployOffsetMonitor(cluster spec.KafkaCluster) error {
	deployment, err := c.KubernetesClient.AppsV1beta1().Deployments(cluster.Metadata.Namespace).Get(c.getOffsetMonitorName(cluster), c.DefaultOption)

	if err != nil {
		fmt.Println("error while talking to k8s api: ", err)
		//TODO better error handling, global retry module?
		return err
	}
	if len(deployment.Name) == 0 {
		//Deployment dosnt exist, creating new.
		fmt.Println("Deployment dosnt exist, creating new")
		replicas := int32(1)

		objectMeta := metav1.ObjectMeta{
			Name: c.getOffsetMonitorName(cluster),
			Annotations: map[string]string{
				"component": "kafka",
				"name":      cluster.Metadata.Name,
				"role":      "data",
				"type":      "service",
			},
		}

		podObjectMeta := metav1.ObjectMeta{
			Name: c.getOffsetMonitorName(cluster),
			Annotations: map[string]string{
				"component": "kafka",
				"name":      cluster.Metadata.Name,
				"role":      "data",
				"type":      "service",
				prometheusScrapeAnnotation: metricsScrape,
				prometheusPortAnnotation: metricsPort,
				prometheusPathAnnotation: metricPath,
			},
		}
		deploy := appsv1Beta1.Deployment{
			ObjectMeta: objectMeta,
			Spec: appsv1Beta1.DeploymentSpec{
				Replicas: &replicas,
				Template: v1.PodTemplateSpec{
					ObjectMeta: podObjectMeta,
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							v1.Container{
								Name: "offfsetExporter",
								Image: offsetExporterImage + ":" + offsetExporterVersion,
								Ports: []v1.ContainerPort{
									v1.ContainerPort{
										Name: "prometheus",
										//TODO configPort
										ContainerPort: 10250,
									},
								},
								Env: []v1.EnvVar{
									v1.EnvVar{
										//TODO fill in with real
										Name: "clusterName",
										ValueFrom: &v1.EnvVarSource{
											FieldRef: &v1.ObjectFieldSelector{
												FieldPath: "metadata.namespace",
											},
										},
									},
									v1.EnvVar{
										Name:  "zookeeperConnect",
										Value: cluster.Spec.ZookeeperConnect,
									},
								},
							},
						},
					},
				},

			},
		}


		_, err := c.KubernetesClient.AppsV1beta1().Deployments(cluster.Metadata.Namespace).Create(&deploy)
		if err != nil {
			fmt.Println("Error while creating Deployment: ", err)
			return err
		}
	} else {
		//Service exist
		fmt.Println("Deployment already exist: ", deployment)
	}

	return nil
}


//Deletes the offset checker for the given kafka cluster.
// Return error if any problems occurs. (Except if monitor dosnt exist)
//
func (c *ClientUtil) DeleteOffsetMonitor(cluster spec.KafkaCluster) error {
	var gracePeriod int64
	gracePeriod = 10

	deleteOption := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	deployment, err := c.KubernetesClient.AppsV1beta1Client.Deployments(cluster.Metadata.Namespace).Get(c.getOffsetMonitorName(cluster), c.DefaultOption) //Scaling Replicas down to Zero
	if (len(deployment.Name) == 0) && (err != nil) {
		fmt.Println("Error while getting Deployment," +
			" since we want to delete that should be fine: ", err)
		return nil
	}

	var replicas int32
	replicas = 0
	deployment.Spec.Replicas = &replicas

	_, err = c.KubernetesClient.AppsV1beta1Client.Deployments(cluster.Metadata.Namespace).Update(deployment)
	if err != nil {
		fmt.Println("Error while scaling down Broker Sts: ", err)
		return err
	}

	//TODO sleep
	err = c.KubernetesClient.AppsV1beta1Client.Deployments(cluster.Metadata.Namespace).Delete(c.getOffsetMonitorName(cluster), &deleteOption)
	if err != nil {
		fmt.Println("Error while deleting deployment, dont care since we want delete anyway?")
		return err
	}
	return nil
}

