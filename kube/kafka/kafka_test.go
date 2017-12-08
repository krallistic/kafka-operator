package kafka

import (
	"reflect"
	"testing"

	"github.com/krallistic/kafka-operator/spec"
	"github.com/kylelemons/godebug/pretty"
	appsv1Beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateStsFromSpec(t *testing.T) {

	spec := spec.Kafkacluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test",
		},
		Spec: spec.KafkaclusterSpec{
			Image:            "testImage",
			BrokerCount:      3,
			JmxSidecar:       false,
			ZookeeperConnect: "testZookeeperConnect",
			StorageClass:     "testStorageClass",
		},
	}

	replicas := int32(3)
	expected := &appsv1Beta1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
			Labels: map[string]string{
				"component": "kafka",
				"creator":   "kafka-operator",
				"role":      "data",
				"name":      "test-cluster",
			},
		},
		Spec: appsv1Beta1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "test-cluster",
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"component": "kafka",
						"creator":   "kafka-operator",
						"role":      "data",
						"name":      "test-cluster",
					},
				},
				Spec: v1.PodSpec{
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
								v1.WeightedPodAffinityTerm{
									Weight: 50,
									PodAffinityTerm: v1.PodAffinityTerm{
										Namespaces: []string{"test"},
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: map[string]string{
												"component": "kafka",
												"creator":   "kafka-operator",
												"role":      "data",
												"name":      "test-cluster",
											},
										},
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	created := generateKafkaStatefulset(spec)

	if created == nil {
		t.Fatalf("return value should not be nil", created)
	}
	if !reflect.DeepEqual(created.ObjectMeta, expected.ObjectMeta) || !reflect.DeepEqual(created.Spec.Template.ObjectMeta, expected.Spec.Template.ObjectMeta) {
		t.Fatalf("Different Metadata")
	}
	if *created.Spec.Replicas != *expected.Spec.Replicas {
		t.Fatalf("DifferentAmount of replicas ", *created.Spec.Replicas, *expected.Spec.Replicas)
	}
	if !reflect.DeepEqual(*created.Spec.Template.Spec.Affinity, *expected.Spec.Template.Spec.Affinity) {
		t.Fatalf("Different AntiAffintiy", *expected.Spec.Template.Spec.Affinity, *created.Spec.Template.Spec.Affinity)
	}
	//if !reflect.DeepEqual(created.Spec.VolumeClaimTemplates, expected.Spec.VolumeClaimTemplates) {
	//	t.Fatalf("Different Volume claim Temlplate", *created.Spec.VolumeClaimTemplates, *expected.Spec.VolumeClaimTemplates)
	//}

}

func TestGenerateHeadlessService(t *testing.T) {

	spec := spec.Kafkacluster{
		ObjectMeta: metav1.ObjectMeta{
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

	objectMeta := metav1.ObjectMeta{
		Name: "test-cluster",
		Annotations: map[string]string{
			"component": "kafka",
			"creator":   "kafka-operator",
			"role":      "data",
			"name":      "test-cluster",
		},
	}

	objectMeta.Labels = map[string]string{
		"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
	}

	expectedResult := &v1.Service{
		ObjectMeta: objectMeta,

		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"component": "kafka",
				"creator":   "kafka-operator",
				"role":      "data",
				"name":      "test-cluster",
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

	result := generateHeadlessService(spec)
	if result == nil {
		t.Fatalf("return value should not be nil", result)
	}
	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("results were not equal", result, expectedResult)
	}
}

func TestGenerateDirectBrokerService(t *testing.T) {

	spec := spec.Kafkacluster{
		ObjectMeta: metav1.ObjectMeta{
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
	expectedResult := []*v1.Service{
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-cluster-broker-0",
				Annotations: map[string]string{
					"component":       "kafka",
					"creator":         "kafka-operator",
					"role":            "data",
					"name":            "test-cluster",
					"kafka_broker_id": "0",
				},
				Namespace: "test",
			},
			Spec: v1.ServiceSpec{
				Selector: map[string]string{
					"component":       "kafka",
					"creator":         "kafka-operator",
					"role":            "data",
					"name":            "test-cluster",
					"kafka_broker_id": "0",
				},
				Ports: []v1.ServicePort{
					v1.ServicePort{
						Name: "broker",
						Port: 9092,
					},
				},
				Type: "ClusterIP",
			},
		},
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-cluster-broker-1",
				Annotations: map[string]string{
					"component":       "kafka",
					"creator":         "kafka-operator",
					"role":            "data",
					"name":            "test-cluster",
					"kafka_broker_id": "1",
				},
				Namespace: "test",
			},

			Spec: v1.ServiceSpec{
				Selector: map[string]string{
					"component":       "kafka",
					"creator":         "kafka-operator",
					"role":            "data",
					"name":            "test-cluster",
					"kafka_broker_id": "1",
				},
				Ports: []v1.ServicePort{
					v1.ServicePort{
						Name: "broker",
						Port: 9092,
					},
				},
				Type: "ClusterIP",
			},
		},
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-cluster-broker-2",
				Annotations: map[string]string{
					"component":       "kafka",
					"creator":         "kafka-operator",
					"role":            "data",
					"name":            "test-cluster",
					"kafka_broker_id": "2",
				},
				Namespace: "test",
			},

			Spec: v1.ServiceSpec{
				Selector: map[string]string{
					"component":       "kafka",
					"creator":         "kafka-operator",
					"role":            "data",
					"name":            "test-cluster",
					"kafka_broker_id": "2",
				},
				Ports: []v1.ServicePort{
					v1.ServicePort{
						Name: "broker",
						Port: 9092,
					},
				},
				Type: "ClusterIP",
			},
		},
	}

	result := generateDirectBrokerServices(spec)
	if result == nil {
		t.Fatalf("return value should not be nil", result)
	}

	if diff := pretty.Compare(result, expectedResult); diff != "" {
		t.Errorf("%s: diff: (-got +want)\n%s", diff)
	}
	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("results were not equal", result, expectedResult)
	}

}
