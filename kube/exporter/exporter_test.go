package exporter

import (
	"fmt"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	appsv1Beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"

	"github.com/krallistic/kafka-operator/spec"
)

func TestGenerateExporterService(t *testing.T) {

	spec := spec.Kafkacluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test",
		},
		Spec: spec.KafkaclusterSpec{
			Image:            "testImage",
			BrokerCount:      1,
			JmxSidecar:       false,
			ZookeeperConnect: "testZookeeperConnect",
		},
	}
	expectedResult := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kafka-offset-checker" + "-" + "test-cluster",
			Labels: map[string]string{
				"component": "kafka",
				"name":      "test-cluster",
				"role":      "data",
				"type":      "exporter",
			},
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"component": "kafka",
				"name":      "test-cluster",
				"role":      "data",
				"type":      "exporter",
			},
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Port: 8080,
				},
			},
		},
	}

	result := generateExporterService(spec)
	if result == nil {
		t.Fatalf("return value should not be nil", result)
	}
	if !reflect.DeepEqual(result, expectedResult) {
		fmt.Println(result)
		fmt.Println("expected")
		fmt.Println(expectedResult)
		t.Fatalf("results were not equal", result, expectedResult)
	}
}

func TestGenerateExporterDeployment(t *testing.T) {

	spec := spec.Kafkacluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test",
		},
		Spec: spec.KafkaclusterSpec{
			Image:            "testImage",
			BrokerCount:      1,
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
				"type":      "exporter",
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
						"type":      "exporter",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Name:  "offset-exporter",
							Image: offsetExporterImage + ":" + offsetExporterVersion,
							Args: []string{
								"--port=8080",
								"--bootstrap-brokers=" + "test-cluster-0.test-cluster.test.svc.cluster.local:9092",
							},
							Ports: []v1.ContainerPort{
								v1.ContainerPort{
									Name: "prometheus",
									//TODO configPort
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}

	result := generateExporterDeployment(spec)
	if result == nil {
		t.Fatalf("return value should not be nil", result)
	}
	if !reflect.DeepEqual(result, expectedResult) {
		fmt.Println(result)
		fmt.Println("expected")
		fmt.Println(expectedResult)
		t.Fatalf("results were not equal", result, expectedResult)
	}
}
