package spec

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"encoding/json"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"fmt"
)


type KafkaCluster struct {
	metav1.TypeMeta `json:",inline"`
	Metadata metav1.ObjectMeta `json:"metadata"`
	//APIVersion string `json:"apiVersion"`
	//Kind string `json:"kind"`
	Spec KafkaClusterSpec `json:"spec"`
}

type KafkaClusterList struct {
	metav1.TypeMeta `json:",inline"`
	Metadata  metav1.ListMeta `json:"metadata"`

	Items []KafkaCluster `json:"items"`
}



type KafkaClusterSpec struct {
	//Amount of Broker Nodes
	Image string `json:"image"`
	BrokerCount int32 `json:"brokerCount"`
	Resources ResourceSpec `json:"resources"`
	KafkaOptions KafkaOption `json:"kafkaOptions"`
	jmxSidecar bool `json:"jmxSidecar"`
	Topics []KafkaTopicSpec `json:"topics"`
	ZookeeperConnect string `json:"zookeeperConnect"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	StorageClass string `json:"storageClass"` //TODO use k8s type?

	// Toleration time if node is down/unreachable/not ready before moving to a new net
	// Set to 0 to disable moving to all together.
	MinimumGracePeriod int32 `json:"minimumGracePeriod"`

	LeaderImbalanceRatio float32 `json:"leaderImbalanceRatio"`
	LeaderImbalanceInterval int32 `json:"leaderImbalanceInterval"`

}

//TODO refactor to just use native k8s types
type ResourceSpec struct {
	Memory string `json:"memory"`
	DiskSpace string `json:"diskSpace"`
	CPU string `json:"cpu"`
}

type KafkaBrokerSpec struct {
	BrokerID int32 `json:"brokerID"`

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
	AutoCreateTopics bool `json:"autoCreateTopics"`
	CompressionType string `json:compressionType`
	
}


//No json needed since internal Event type.
type KafkaClusterEvent struct {
	Type KafkaEventType
	Cluster KafkaCluster
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
	BROKER_CONFIG_CHANGE
	UNKNOWN_CHANGE
	RECONSTILATION_EVENT
	//Cleanup event which get emmised after a Cluster Delete.
	//Its ensure the deletion of the Statefulset after it has been scaled down.
	CLEANUP_EVENT
	KAKFA_EVENT

)


// convenience functions
func PrintCluster(cluster *KafkaCluster) string {
	return fmt.Sprintf("%s/%s, APIVersion: %s, Kind: %s, Value: %#v", cluster.Metadata.Namespace, cluster.Metadata.Name, cluster.APIVersion, cluster.Kind, cluster)
}

// Required to satisfy Object interface
func (e *KafkaCluster) GetObjectKind() schema.ObjectKind {
	return &e.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (e *KafkaCluster) GetObjectMeta() metav1.Object {
	return &e.Metadata
}

// Required to satisfy Object interface
func (el *KafkaClusterList) GetObjectKind() schema.ObjectKind {
	return &el.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (el *KafkaClusterList) GetListMeta() metav1.List {
	return &el.Metadata
}

//Shamefull copied over from: https://github.com/kubernetes/client-go/blob/master/examples/third-party-resources/types.go
// The code below is used only to work around a known problem with third-party
// resources and ugorji. If/when these issues are resolved, the code below
// should no longer be required.

type KafkaClusterListCopy KafkaClusterList
type KafkaClusterCopy KafkaCluster

func (e *KafkaCluster) UnmarshalJSON(data []byte) error {
	tmp := KafkaClusterCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := KafkaCluster(tmp)
	*e = tmp2
	return nil
}

func (el *KafkaClusterList) UnmarshalJSON(data []byte) error {
	tmp := KafkaClusterListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := KafkaClusterList(tmp)
	*el = tmp2
	return nil
}
