package spec

import (
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

//Main API Object
type Kafkacluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	//Metadata          metav1.ObjectMeta `json:"metadata"`

	Spec  KafkaclusterSpec  `json:"spec"`
	State KafkaclusterState `json:"state,omitempty"`
	Scale KafkaclusterScale `json:"scale,omitempty"`
}

// k8s API List Type
type KafkaclusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	//Metadata        metav1.ListMeta `json:"metadata"`

	Items []Kafkacluster `json:"items"`
}

type KafkaclusterSpec struct {
	//Amount of Broker Nodes
	Image            string            `json:"image"`
	BrokerCount      int32             `json:"brokerCount"`
	Resources        ResourceSpec      `json:"resources"`
	KafkaOptions     KafkaOptions      `json:"kafkaOptions"`
	JmxSidecar       bool              `json:"jmxSidecar"`
	Topics           []KafkaTopicSpec  `json:"topics"`
	ZookeeperConnect string            `json:"zookeeperConnect"`
	NodeSelector     map[string]string `json:"nodeSelector,omitempty"`
	StorageClass     string            `json:"storageClass"` //TODO use k8s type?

	// Toleration time if node is down/unreachable/not ready before moving to a new net
	// Set to 0 to disable moving to all together.
	MinimumGracePeriod int64 `json:"minimumGracePeriod"`

	LeaderImbalanceRatio    float32 `json:"leaderImbalanceRatio"`
	LeaderImbalanceInterval int32   `json:"leaderImbalanceInterval"`
}

//KafkaclusterState Represent State field inside cluster, is used to do insert current state information.
type KafkaclusterState struct {
	Status  string        `json:"status,omitempty"`
	Topics  []string      `json:"topics,omitempty"`
	Brokers []BrokerState `json:"brokers,omitempty"`
}

//BrokerState contains state about brokers
type BrokerState struct {
	ID    string `json:"id,omitempty"`
	State string `json:"state,omitempty"`
}

//KafkaclusterScale represent the `scale` field inside the crd
type KafkaclusterScale struct {
	CurrentScale int32 `json:"currentScale,omitempty"`
	DesiredScale int32 `json:"desiredScale,omitempty"`
}

//TODO refactor to just use native k8s types
type ResourceSpec struct {
	Memory    string `json:"memory"`
	DiskSpace string `json:"diskSpace"`
	CPU       string `json:"cpu"`
}

type KafkaBrokerSpec struct {
	BrokerID int32 `json:"brokerID"`

	ClientPort int32             `json:"clientPort"`
	Topics     map[string]string `json:"topics"`
}

type KafkaTopicSpec struct {
	Name              string `json:"name"`
	Partitions        int32  `json:"partitions"`
	ReplicationFactor int32  `json:"replicationFactor"`
}

type KafkaclusterWatchEvent struct {
	Type      string       `json:"type"`
	Object    Kafkacluster `json:"object"`
	OldObject Kafkacluster `json:"oldObject"`
}

type KafkaOption struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  interface{}
}

//Unsused
type KafkaOptions struct {
	// Default: true
	AutoCreateTopicsEnable *bool `json:"autoCreateTopicsEnable,omitempty"`

	//Enables auto balancing of topic leaders.  Done by a background thread.
	// Default: true
	AutoLeaderRebalanceEnable *bool `json:"autoLeaderRebalanceEnable,omitempty"`

	//Amount of threads for various background tasks
	// Default: 10
	BackgroundThreads *int32 `json:"backgroudThreads,omitempty"`

	//Default compression type for a topic. Can be "gzip", "snappy", "lz4"
	// Default: "gzip"
	CompressionType *string `json:"compressionType,omitempty"`

	// Enables delete topic. Delete topic through the admin tool will have no effect if this config is turned off
	// Default: false
	DeleteTopicEnable *bool `json:"deleteTopicEnable,omitempty"`

	//The frequency with which the partition rebalance check is triggered by the controller
	// Default:300
	LeaderImbalanceCheckIntervalSeconds *int32 `json:"leaderImbalanceCheckIntervalSeconds,omitempty"`

	// The ratio of leader imbalance allowed per broker.
	// The controller would trigger a leader balance if it goes above this value per broker. The value is specified in percentage.
	// Default: 10
	LeaderImbalanceBrokerPercentage *int32 `json:"leaderImbalanceBrokerPercentage,omitempty"`

	//The number of messages accumulated on a log partition before messages are flushed to disk
	// Default: 9223372036854775807
	LogFlushIntervalMessages *int64 `json:"logFlushIntervalMessages,omitempty"`

	// The maximum time in ms that a message in any topic is kept in memory before flushed to disk.
	// If not set, the value in log.flush.scheduler.interval.ms is used
	// Default: null
	LogFlushIntervalMs *int64 `json:"logFlushIntervalMs,omitempty"`

	//The frequency with which we update the persistent record of the last flush which acts as the log recovery point
	// Default: 60000
	LogFlushOffsetCheckpointIntervalMs *int32 `json:"logFlushOffsetCheckpointIntervalMs,omitempty"`

	//The frequency in ms that the log flusher checks whether any log needs to be flushed to disk
	// Default: 9223372036854775807
	LogFlushSchedulerIntervalMs *int64 `json:"LogFlushSchedulerIntervalMs,omitempty"`

	// The maximum size of the log before deleting it
	// Default: -1
	LogRetentionBytes *string `json:"logRetentionBytes,omitempty"`

	// The number of hours to keep a log file before deleting it (in hours), tertiary to log.retention.ms property
	// Default: 168
	LogRetentionHours *int32 `json:"logRetentionHours,omitempty"`

	//The maximum time before a new log segment is rolled out (in hours), secondary to log.roll.ms property
	// Default: 168
	LogRollHours *int32 `json:"logRollHours,omitempty"`

	// The maximum jitter to subtract from logRollTimeMillis (in hours), secondary to log.roll.jitter.ms property
	// Default: 0
	LogRollJitterHours *int32 `json:"logRollJitterHours,omitempty"`

	//The maximum size of a single log file
	// Default: 1073741824
	LogSegmentBytes *int32 `json:"logSegmentBytes,omitempty"`

	// The amount of time to wait before deleting a file from the filesystem
	// Default: 60000
	LogSegmentDeleteDelayMS *int64 `json:"logSegmentDeleteDelayMS,omitempty"`

	// The maximum size of message that the server can receive
	// Default: 1000012
	MessagesMaxBytes *int32 `json:"messagesMaxBytes,omitempty"`

	// When a producer sets acks to "all" (or "-1"), min.insync.replicas specifies the minimum number of replicas that must acknowledge
	// a write for the write to be considered successful.
	// Can be overwritten at topic level
	// Default: 1
	MinInsyncReplicas *int32 `json:"minInsyncReplicas,omitempty"`

	// The number of io threads that the server uses for carrying out network requests
	// Default: 8
	NumIOThreads *int32 `json:"numIOThreads,omitempty"`

	// The number of network threads that the server uses for handling network requests
	// Default: 3
	NumNetworkThreads *int32 `json:"numNetworkThreads,omitempty"`

	//The number of threads per data directory to be used for log recovery at startup and flushing at shutdown
	// Default: 1
	NumRecoveryThreadsPerDataDir *int32 `json:"numRecoveryThreadsPerDataDir,omitempty"`

	// Number of fetcher threads used to replicate messages from a source broker.
	// Increasing this value can increase the degree of I/O parallelism in the follower broker.
	// Default: 1
	NumReplicaFetchers *int32 `json:"numReplicaFetchers,omitempty"`

	// The maximum size for a metadata entry associated with an offset commit.
	// Default: 4096
	OffsetMetadataMaxBytes *int32 `json:"offsetMetadataMaxBytes,omitempty"`

	// The required acks before the commit can be accepted. In general, the default (-1) should not be overridden
	// Default: -1
	// Commented out because of dangerous option
	//OffsetCommitReadRequiredAcks int32 `json:"offsetCommitReadRequiredAcks"`

	//Offset commit will be delayed until all replicas for the offsets topic receive the commit or this timeout is reached.
	// This is similar to the producer request timeout.
	// Default: 5000
	OffsetCommitTimeoutMs *int32 `json:"offsetCommitTimeoutMs,omitempty"`

	// Batch size for reading from the offsets segments when loading offsets into the cache.
	// Default: 5242880
	OffsetLoadBufferSize *int32 `json:"offsetLoadBufferSize,omitempty"`

	// Frequency at which to check for stale offsets
	// Default: 600000
	OffsetRetentionCheckIntervalMs *int64 `json:"offsetRetentionCheckIntervalMs,omitempty"`

	// Log retention window in minutes for offsets topic
	// Default: 1440
	OffsetRetentionMinutes *int32 `json:"offsetRetentionMinutes,omitempty"`

	// Compression codec for the offsets topic - compression may be used to achieve "atomic" commits
	// Default: 0
	//Commented out, wrong doku? int fro compression???
	//OffsetTopicCompressionCodec int32 `json:"offset_topic_compression_coded"`

	// The number of partitions for the offset commit topic (should not change after deployment)
	// Default: 50
	OffsetTopicNumPartitions *int32 `json:"offsetTopicNumPartitions,omitempty"`

	// The replication factor for the offsets topic (set higher to ensure availability).
	// To ensure that the effective replication factor of the offsets topic is the configured value, the number of alive brokers has to be at least the replication factor at the time of the first request for the offsets topic.
	// If not, either the offsets topic creation will fail or it will get a replication factor of min(alive brokers, configured replication factor)
	// Default: 3
	OffsetTopicReplicationFactor *int32 `json:"offsetTopicReplicationFactor,omitempty"`

	// The offsets topic segment bytes should be kept relatively small in order to facilitate faster log compaction and cache loads
	// Default: 104857600
	OffsetTopicSegmentsBytes *int32 `json:"offsetTopicSegmentsBytes,omitempty"`

	// The number of queued requests allowed before blocking the network threads
	// Default: 100
	QueuedMaxRequest *int32 `json:"queuedMaxRequest,omitempty"`

	// Minimum bytes expected for each fetch response. If not enough bytes, wait up to replicaMaxWaitTimeMs
	// Default: 1
	ReplicaFetchMinBytes *int32 `json:"replicaFetchMinBytes,omitempty"`

	//max wait time for each fetcher request issued by follower replicas.
	// This value should always be less than the replica.lag.time.max.ms at all times to prevent frequent shrinking of ISR for low throughput topics
	//Default: 500
	ReplicaFetchWaitMaxMs *int32 `json:"replicaFetchWaitMaxMs,omitempty"`

	//The frequency with which the high watermark is saved out to disk
	// Default: 5000
	ReplicaHighWatermarkCheckpointIntervalMs *int64 `json:"replicaHighWatermarkCheckpointIntervalMs,omitempty"`

	// If a follower hasn't sent any fetch requests or hasn't consumed up to the leaders log end offset for at least this time,
	// the leader will remove the follower from isr
	// Defaut: 10000
	ReplicaLagTimeMaxMs *int64 `json:"replicaLagTimeMaxMs,omitempty"`

	// The socket receive buffer for network requests
	// Default: 65536
	ReplicaSocketReceiveBufferBytes *int32 `json:"replicaSocketReceiveBufferBytes,omitempty"`

	// The socket timeout for network requests. Its value should be at least replica.fetch.wait.max.ms
	// Default: 30000
	ReplicaSocketTimeoutMs *int32 `json:"replicaSocketTimeoutMs,omitempty"`

	// The configuration controls the maximum amount of time the client will wait for the response of a request.
	// If the response is not received before the timeout elapses the client will resend the request if necessary
	// or fail the request if retries are exhausted.
	// Default: 30000
	RequestTimeoutMs *int32 `json:"requestTimeoutMs,omitempty"`

	//Socket Settings? TODO? needed

	// Indicates whether to enable replicas not in the ISR set to be elected as leader as a last resort, even though doing so may result in data loss
	// Default: true
	UncleanLeaderElectionEnable *bool `json:"uncleanLeaderElectionEnable,omitempty"`

	// The max time that the client waits to establish a connection to zookeeper.
	// If not set, the value in zookeeper.session.timeout.ms is used
	// Default: null
	ZookeeperConnectionTimeoutMs *int32 `json:"zookeeperConnectionTimeoutMs,omitempty"`

	// Zookeeper session timeout
	// Default: 6000
	ZookeeperSessionTimeoutMs *int32 `json:"zookeeperSessionTimeoutMs,omitempty"`
}

//ReassigmentConfig
type KafkaReassignmentConfig struct {
	Partition []KafkaPartition `json:"partition"`
	Version   string           `json:"version"`
}

type KafkaPartition struct {
	Partition int32   `json:"partition"`
	Replicas  []int32 `json:"replicas"`
}

type KafkaTopic struct {
	Topic             string           `json:"topic"`
	PartitionFactor   int32            `json:"partition_factor"`
	ReplicationFactor int32            `json:"replication_factor"`
	Partitions        []KafkaPartition `json:"partitions"`
}

//No json needed since internal Event type.
type KafkaclusterEvent struct {
	Type       KafkaEventType
	Cluster    Kafkacluster
	OldCluster Kafkacluster
}

type KafkaEventType int32

const (
	NEW_CLUSTER KafkaEventType = iota + 1
	DELETE_CLUSTER
	UPSIZE_CLUSTER
	DOWNSIZE_CLUSTER
	CHANGE_IMAGE
	CHANGE_BROKER_RESOURCES
	CHANGE_NAME
	CHANGE_ZOOKEEPER_CONNECT
	BROKER_CONFIG_CHANGE
	UNKNOWN_CHANGE
	DOWNSIZE_EVENT
	//Cleanup event which get emmised after a Cluster Delete.
	//Its ensure the deletion of the Statefulset after it has been scaled down.
	CLEANUP_EVENT
	KAKFA_EVENT
	STATE_CHANGE
	SCALE_CHANGE
	ERROR_STATE
)

type KafkaBrokerState string

const (
	EMPTY_BROKER     KafkaBrokerState = "EMPTYING"
	REBALANCE_BROKER KafkaBrokerState = "REBALANCING"
	NORMAL_STATE     KafkaBrokerState = "NORMAL"
)

//convenience functions
func PrintCluster(cluster *Kafkacluster) string {
	return fmt.Sprintf("%s/%s, APIVersion: %s, Kind: %s, Value: %#v", cluster.ObjectMeta.Namespace, cluster.ObjectMeta.Name, cluster.APIVersion, cluster.Kind, cluster)
}

// Required to satisfy Object interface
func (e *Kafkacluster) GetObjectKind() schema.ObjectKind {
	return &e.TypeMeta
}

// Required to satisfy Object interface
func (el *KafkaclusterList) GetObjectKind() schema.ObjectKind {

	return &el.TypeMeta
}

//Shamefull copied over from: https://github.com/kubernetes/client-go/blob/master/examples/third-party-resources/types.go
// The code below is used only to work around a known problem with third-party
// resources and ugorji. If/when these issues are resolved, the code below
// should no longer be required.

// type KafkaclusterListCopy KafkaclusterList
// type KafkaclusterCopy Kafkacluster

// func (e *Kafkacluster) UnmarshalJSON(data []byte) error {
// 	tmp := KafkaclusterCopy{}
// 	err := json.Unmarshal(data, &tmp)
// 	if err != nil {
// 		return err
// 	}
// 	tmp2 := Kafkacluster(tmp)
// 	*e = tmp2
// 	return nil
// }

// func (el *KafkaclusterList) UnmarshalJSON(data []byte) error {
// 	tmp := KafkaclusterListCopy{}
// 	err := json.Unmarshal(data, &tmp)
// 	if err != nil {
// 		return err
// 	}
// 	tmp2 := KafkaclusterList(tmp)
// 	*el = tmp2
// 	return nil
// }

// Deprecated: GetGeneratedDeepCopyFuncs returns the generated funcs, since we aren't registering them.
func GetGeneratedDeepCopyFuncs() []conversion.GeneratedDeepCopyFunc {
	return []conversion.GeneratedDeepCopyFunc{
		{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*Kafkacluster).DeepCopyInto(out.(*Kafkacluster))
			return nil
		}, InType: reflect.TypeOf(&Kafkacluster{})},
		{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*KafkaclusterList).DeepCopyInto(out.(*KafkaclusterList))
			return nil
		}, InType: reflect.TypeOf(&KafkaclusterList{})},
		{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*KafkaclusterSpec).DeepCopyInto(out.(*KafkaclusterSpec))
			return nil
		}, InType: reflect.TypeOf(&KafkaclusterSpec{})},
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Kafkacluster) DeepCopyInto(out *Kafkacluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	//in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.ObjectMeta = in.ObjectMeta

	out.Spec = in.Spec
	return
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, creating a new Kafkacluster.
func (x *Kafkacluster) DeepCopy() *Kafkacluster {
	if x == nil {
		return nil
	}
	out := new(Kafkacluster)
	x.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (x *Kafkacluster) DeepCopyObject() runtime.Object {
	if c := x.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KafkaclusterList) DeepCopyInto(out *KafkaclusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Kafkacluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, creating a new KafkaclusterList.
func (x *KafkaclusterList) DeepCopy() *KafkaclusterList {
	if x == nil {
		return nil
	}
	out := new(KafkaclusterList)
	x.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (x *KafkaclusterList) DeepCopyObject() runtime.Object {
	if c := x.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KafkaclusterSpec) DeepCopyInto(out *KafkaclusterSpec) {
	*out = *in
	return
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, creating a new KafkaclusterSpec.
func (x *KafkaclusterSpec) DeepCopy() *KafkaclusterSpec {
	if x == nil {
		return nil
	}
	out := new(KafkaclusterSpec)
	x.DeepCopyInto(out)
	return out
}
