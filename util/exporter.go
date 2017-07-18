package util

import (
	"fmt"

	"github.com/krallistic/kafka-operator/spec"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1Beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"

	log "github.com/Sirupsen/logrus"

	"k8s.io/client-go/pkg/api/v1"
)

const (
	deplyomentPrefix      = "kafka-offset-checker"
	offsetExporterImage   = "krallistic/kafka_exporter" //TODO
	offsetExporterVersion = "v0.1.0"                    //TODO make version cmd arg

	prometheusScrapeAnnotation = "prometheus.io/scrape"
	prometheusPortAnnotation   = "prometheus.io/port"
	prometheusPathAnnotation   = "prometheus.io/path"

	metricPath    = "/metrics"
	metricsPort   = "8080"
	metricsScrape = "true"
)

func (c *ClientUtil) getOffsetMonitorName(cluster spec.Kafkacluster) string {
	return deplyomentPrefix + "-" + cluster.Metadata.Name
}

func (c *ClientUtil) GenerateExporterDeployment(cluster spec.Kafkacluster) *appsv1Beta1.Deployment {
	replicas := int32(1)

	objectMeta := metav1.ObjectMeta{
		Name: c.getOffsetMonitorName(cluster),
		Labels: map[string]string{
			"component": "kafka",
			"name":      cluster.Metadata.Name,
			"role":      "data",
			"type":      "service",
		},
	}
	podObjectMeta := metav1.ObjectMeta{
		Name: c.getOffsetMonitorName(cluster),
		Annotations: map[string]string{

			prometheusScrapeAnnotation: metricsScrape,
			prometheusPortAnnotation:   metricsPort,
			prometheusPathAnnotation:   metricPath,
		},
		Labels: map[string]string{
			"component": "kafka",
			"name":      cluster.Metadata.Name,
			"role":      "data",
			"type":      "service",
		},
	}

	deploy := &appsv1Beta1.Deployment{
		ObjectMeta: objectMeta,
		Spec: appsv1Beta1.DeploymentSpec{
			Replicas: &replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: podObjectMeta,
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Name:  "offset-exporter",
							Image: offsetExporterImage + ":" + offsetExporterVersion,
							Ports: []v1.ContainerPort{
								v1.ContainerPort{
									Name: "prometheus",
									//TODO configPort
									ContainerPort: 8080,
								},
							},
							Env: []v1.EnvVar{
								v1.EnvVar{
									Name:  "CLUSTER_NAME",
									Value: cluster.Metadata.Name,
								},
								v1.EnvVar{
									Name:  "ZOOKEEPER_CONNECT",
									Value: cluster.Spec.ZookeeperConnect,
								},
								v1.EnvVar{
									Name:  "LISTEN_ADDRESS",
									Value: ":8080",
								},
								v1.EnvVar{
									Name:  "TELEMETRY_PATH",
									Value: "/metrics",
								},
							},
						},
					},
				},
			},
		},
	}

	return deploy

}

// Deploys the OffsetMonitor as an extra Pod inside the Cluster
func (c *ClientUtil) DeployOffsetMonitor(cluster spec.Kafkacluster) error {
	methodLogger := logger.WithFields(log.Fields{
		"method":    "DeployOffsetMonitor",
		"name":      cluster.Metadata.Name,
		"namespace": cluster.Metadata.Namespace,
	})

	deployment, err := c.KubernetesClient.AppsV1beta1().Deployments(cluster.Metadata.Namespace).Get(c.getOffsetMonitorName(cluster), c.DefaultOption)

	if err != nil {
		if !errors.IsNotFound(err) {
			methodLogger.WithFields(log.Fields{
				"error": err,
			}).Error("Cant get Deployment INFO from API")
			return err
		}
	}
	if len(deployment.Name) == 0 {
		//Deployment dosnt exist, creating new.
		methodLogger.Info("Deployment dosnt exist, creating new")

		deploy := c.GenerateExporterDeployment(cluster)

		_, err := c.KubernetesClient.AppsV1beta1().Deployments(cluster.Metadata.Namespace).Create(deploy)
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
func (c *ClientUtil) DeleteOffsetMonitor(cluster spec.Kafkacluster) error {
	var gracePeriod int64
	gracePeriod = 10

	deleteOption := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	deployment, err := c.KubernetesClient.AppsV1beta1Client.Deployments(cluster.Metadata.Namespace).Get(c.getOffsetMonitorName(cluster), c.DefaultOption) //Scaling Replicas down to Zero
	if (len(deployment.Name) == 0) && (err != nil) {
		fmt.Println("Error while getting Deployment,"+
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
