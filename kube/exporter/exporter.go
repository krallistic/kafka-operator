package exporter

import (
	"strings"

	"github.com/krallistic/kafka-operator/kube"
	"github.com/krallistic/kafka-operator/spec"
	util "github.com/krallistic/kafka-operator/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1Beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"

	"k8s.io/client-go/pkg/api/v1"
)

const (
	deplyomentPrefix      = "kafka-offset-checker"
	offsetExporterImage   = "braedon/prometheus-kafka-consumer-group-exporter" //TODO
	offsetExporterVersion = "0.2.0"                                            //TODO make version cmd arg

	prometheusScrapeAnnotation = "prometheus.io/scrape"
	prometheusPortAnnotation   = "prometheus.io/port"
	prometheusPathAnnotation   = "prometheus.io/path"

	metricPath    = "/metrics"
	metricsPort   = "8080"
	metricsScrape = "true"
)

func getOffsetExporterName(cluster spec.Kafkacluster) string {
	return deplyomentPrefix + "-" + cluster.ObjectMeta.Name
}

func generateExporterLabels(cluster spec.Kafkacluster) map[string]string {
	return map[string]string{
		"component": "kafka",
		"name":      cluster.ObjectMeta.Name,
		"role":      "data",
		"type":      "exporter",
	}
}

func generateExporterService(cluster spec.Kafkacluster) *v1.Service {
	objectMeta := metav1.ObjectMeta{
		Name:   getOffsetExporterName(cluster),
		Labels: generateExporterLabels(cluster),
	}

	svc := &v1.Service{
		ObjectMeta: objectMeta,
		Spec: v1.ServiceSpec{
			Selector: generateExporterLabels(cluster),
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Port: 8080,
				},
			},
		},
	}

	return svc
}

func generateExporterDeployment(cluster spec.Kafkacluster) *appsv1Beta1.Deployment {
	replicas := int32(1)

	objectMeta := metav1.ObjectMeta{
		Name:   getOffsetExporterName(cluster),
		Labels: generateExporterLabels(cluster),
	}
	podObjectMeta := metav1.ObjectMeta{
		Name: getOffsetExporterName(cluster),
		Annotations: map[string]string{

			prometheusScrapeAnnotation: metricsScrape,
			prometheusPortAnnotation:   metricsPort,
			prometheusPathAnnotation:   metricPath,
		},
		Labels: generateExporterLabels(cluster),
	}
	brokerList := strings.Join(util.GetBrokerAdressess(cluster), ",")

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
							//Command: ["python", "-u", "/usr/local/bin/prometheus-kafka-consumer-group-exporter"],
							Args: []string{
								"--port=8080",
								"--bootstrap-brokers=" + brokerList,
							},
							Ports: []v1.ContainerPort{
								v1.ContainerPort{
									Name:          "prometheus",
									ContainerPort: 8080,
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
func DeployOffsetMonitor(cluster spec.Kafkacluster, client kube.Kubernetes) error {
	deployment := generateExporterDeployment(cluster)
	svc := generateExporterService(cluster)

	//Deploy Offset Monitor Deplyoment
	err := client.CreateOrUpdateDeployment(deployment)
	if err != nil {
		return err
	}
	//Deploy Offset Monitor Service
	err = client.CreateOrUpdateService(svc)

	return err
}

//
//
//
func DeleteOffsetMonitor(cluster spec.Kafkacluster, client kube.Kubernetes) error {
	deployment := generateExporterDeployment(cluster)
	svc := generateExporterService(cluster)

	client.DeleteDeployment(deployment)
	client.DeleteService(svc)

	return nil
}
