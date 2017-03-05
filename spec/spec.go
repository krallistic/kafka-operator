package spec

import (
	"k8s.io/client-go/pkg/api/v1"
)


type KafkaCluster struct {
	APIVersion string `json:"apiVersion"`
	Kind string `json:"kind"`
	Metadata v1.ObjectMeta `json:"metadata"`
	Spec KafkaClusterSpec `json:"spec"`
}



type KafkaClusterSpec struct {
	//Amount of Broker Nodes
	Image string `json:"image"`
	Name string `json:"name"`
	BrokerCount int32 `json:"brokerCount"`
	Brokers []KafkaBrokerSpec `json:"brokers"`
	KafkaOptions KafkaOption `json:"kafkaOptions"`
	jmxSidecar bool `json:"jmxSidecar"`
	
	ZookeeperConnect string `json:"zookeeperConnect"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	
}

type KafkaBrokerSpec struct {
	BrokerID int32 `json:"brokerID"`
	Memory int32 `json:"memory"`
	DiskSpace int32 `json:"diskSpace"` //TODO Option to use GB etc
	CPU int32 `json:"cpu"`
	ClientPort int32 `json:"clientPort"`
	Topics map[string]string `json:"topics"`
}

type KafkaTopicSpec struct {
	Name string `json:"name"`
	Partitions int32 `json:"partitions"`
	ReplicationFactor int32 `json:"replicationFactor"`
}

type KafkaClusterWatchEvent struct {
	Type string `json:"type"`
	Object KafkaCluster `json:"object"`
}

type KafkaOption struct {
	LogRetentionHours int `json:"logRetentionHours"`
}


type KafkaEventType int32

const (
	NEW_CLUSTER KafkaEventType = iota + 1
	DELTE_CLUSTER
	UPSIZE_CLUSTER
	DOWNSIZE_CLUSTER
	CHANGE_IMAGE
	CHANGE_BROKER_RESOURCES
	CHANGE_NAME
	CHANGE_ZOOKEEPER_CONNECT
	UNKNOWN_CHANGE

)
