package util

import (
	"fmt"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/api/core/v1"

	"github.com/krallistic/kafka-operator/spec"
)

func TestGenerateKafkaOptions(t *testing.T) {
	util := ClientUtil{}

	topicsCreate := true
	logRetentionBytes := "testLogRetentionTime"
	testClusterSpec := spec.Kafkacluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test",
		},
		Spec: spec.KafkaclusterSpec{

			KafkaOptions: spec.KafkaOptions{
				AutoCreateTopicsEnable: &topicsCreate,
				LogRetentionBytes:      &logRetentionBytes,
			},
		},
	}

	expectedResult := []v1.EnvVar{
		v1.EnvVar{
			Name:  "AUTO_CREATE_TOPICS_ENABLE",
			Value: "true",
		},
		v1.EnvVar{
			Name:  "LOG_RETENTION_BYTES",
			Value: "testLogRetentionTime",
		},
		v1.EnvVar{
			Name: "NAMESPACE",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		v1.EnvVar{
			Name: "KAFKA_ZOOKEEPER_CONNECT",
		},
		v1.EnvVar{
			Name:  "KAFKA_HEAP_OPTS",
			Value: "-Xmx2577M",
		},
		v1.EnvVar{
			Name:  "KAFKA_METRIC_REPORTERS",
			Value: "com.linkedin.kafka.cruisecontrol.metricsreporter.CruiseControlMetricsReporter",
		},
		v1.EnvVar{
			Name:  "KAFKA_CRUISE_CONTROL_METRICS_REPORTER_BOOTSTRAP_SERVER",
			Value: "test-cluster-0.test-cluster.test.svc.cluster.local:9092",
		},
		v1.EnvVar{
			Name:  "KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR",
			Value: "1",
		},
	}

	result := util.GenerateKafkaOptions(testClusterSpec)
	if result == nil {
		t.Fatalf("return value should not be nil", result)
	}
	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("results were not equal")
	}

	fmt.Println(result)

}
