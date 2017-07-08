package util

import (
	"fmt"
	"reflect"
	"testing"

	"k8s.io/client-go/pkg/api/v1"

	"github.com/krallistic/kafka-operator/spec"
)

func TestGenerateKafkaOptions(t *testing.T) {
	util := ClientUtil{}

	topicsCreate := true
	logRetentionBytes := "testLogRetentionTime"
	testClusterSpec := spec.KafkaCluster{
		Spec: spec.KafkaClusterSpec{

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
