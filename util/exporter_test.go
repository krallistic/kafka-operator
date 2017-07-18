package util

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	appsv1Beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"

	"github.com/krallistic/kafka-operator/spec"
)

func TestGenerateExporterDeployment(t *testing.T) {
	util := ClientUtil{}

	spec := spec.Kafkacluster{
		Metadata: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test",
		},
		Spec: spec.KafkaclusterSpec{
			Image:            "testImage",
			BrokerCount:      3,
			JmxSidecar:       false,
			ZookeeperConnect: "testZookeeperConnect",
		},
	}
	replicas := int32(1)

	expectedResult := &appsv1Beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kafka-offset-checker" + "-" + "test-cluster",
			Labels: map[string]string{
				"component": "kafka",
				"name":      "test-cluster",
				"role":      "data",
				"type":      "service",
			},
		},
		Spec: appsv1Beta1.DeploymentSpec{
			Replicas: &replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kafka-offset-checker" + "-" + "test-cluster",
					Annotations: map[string]string{

						prometheusScrapeAnnotation: metricsScrape,
						prometheusPortAnnotation:   metricsPort,
						prometheusPathAnnotation:   metricPath,
					},
					Labels: map[string]string{
						"component": "kafka",
						"name":      "test-cluster",
						"role":      "data",
						"type":      "service",
					},
				},
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
									Value: "test-cluster",
								},
								v1.EnvVar{
									Name:  "ZOOKEEPER_CONNECT",
									Value: "testZookeeperConnect",
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

	result := util.GenerateExporterDeployment(spec)
	if result == nil {
		t.Fatalf("return value should not be nil", result)
	}
	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("results were not equal", result, expectedResult)
	}
}
