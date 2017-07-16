package util

import (
	"reflect"
	"testing"

	"github.com/krallistic/kafka-operator/spec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	appsv1Beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func TestCreateStsFromSpec(t *testing.T) {
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
												"creator": "kafka-operator",
												"name":    "test-cluster",
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

	created := util.createStsFromSpec(spec)

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

}
