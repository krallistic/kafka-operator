package spec

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"fmt"
)

type KafkaCluster struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ObjectMeta `json:"metadata"`
	//APIVersion string `json:"apiVersion"`
	//Kind string `json:"kind"`
	Spec KafkaClusterSpec `json:"spec"`
}

type KafkaClusterList struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ListMeta `json:"metadata"`

	Items []KafkaCluster `json:"items"`
}

type KafkaClusterSpec struct {
	//Amount of Broker Nodes
	Image            string            `json:"image"`
	BrokerCount      int32             `json:"brokerCount"`
	Resources        ResourceSpec      `json:"resources"`
	KafkaOptions     KafkaOptions       `json:"kafkaOptions"`
	jmxSidecar       bool              `json:"jmxSidecar"`
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

type KafkaClusterWatchEvent struct {
	Type      string       `json:"type"`
	Object    KafkaCluster `json:"object"`
	OldObject KafkaCluster `json:"oldObject"`
}

type KafkaOptions struct {

	// Enables auto create of topics on the broker
	// Default: true
	AutoCreateTopicsEnable  bool   `json:"autoCreateTopicsEnable"`

	//Enables auto balancing of topic leaders.  Done by a background thread.
	// Default: true
	AutoLeaderRebalanceEnable bool `json:"autoLeaderRebalanceEnable"`

	//Amount of threads for various background tasks
	// Default: 10
	BackgroundThreads int32 `json:"backgroudThreads"`

	//Default compression type for a topic. Can be "gzip", "snappy", "lz4"
	// Default: "gzip"
	CompressionType   string `json:"compressionType"`

	//Enables delete topic. Delete topic through the admin tool will have no effect if this config is turned off
	// Default: false
	DeleteTopicEnable bool `json:"deleteTopicEnable"`

	//The frequency with which the partition rebalance check is triggered by the controller
	// Default:300
	LeaderImbalanceCheckIntervalSeconds int32 `json:"leaderImbalanceCheckIntervalSeconds"`

	// The ratio of leader imbalance allowed per broker.
	// The controller would trigger a leader balance if it goes above this value per broker. The value is specified in percentage.
	// Default: 10
	LeaderImbalanceBrokerPercentage int32 `json:"leaderImbalanceBrokerPercentage"`

	//The number of messages accumulated on a log partition before messages are flushed to disk
	// Default: 9223372036854775807
	LogFlushIntervalMessages int64 `json:"logFlushIntervalMessages"`

	// The maximum time in ms that a message in any topic is kept in memory before flushed to disk.
	// If not set, the value in log.flush.scheduler.interval.ms is used
	// Default: null
	LogFlushIntervalMs int64 `json:"logFlushIntervalMs"`

	//The frequency with which we update the persistent record of the last flush which acts as the log recovery point
	// Default: 60000
	LogFlushOffsetCheckpointIntervalMs int32 `json:"logFlushOffsetCheckpointIntervalMs"`

	//The frequency in ms that the log flusher checks whether any log needs to be flushed to disk
	// Default: 9223372036854775807
	LogFlushSchedulerIntervalMs int64 `json:"LogFlushSchedulerIntervalMs"`

	// The maximum size of the log before deleting it
	// Default: -1
	LogRetentionBytes string `json:"logRetentionBytes"`

	// The number of hours to keep a log file before deleting it (in hours), tertiary to log.retention.ms property
	// Default: 168
	LogRetentionHours int32    `json:"logRetentionHours"`

	//The maximum time before a new log segment is rolled out (in hours), secondary to log.roll.ms property
	// Default: 168
	LogRollHours int32 `json:"logRollHours"`

	// The maximum jitter to subtract from logRollTimeMillis (in hours), secondary to log.roll.jitter.ms property
	// Default: 0
	LogRollJitterHours int32 `json:"logRollJitterHours"`

	//The maximum size of a single log file
	// Default: 1073741824
	LogSegmentBytes int32 `json:"logSegmentBytes"`

	// The amount of time to wait before deleting a file from the filesystem
	// Default: 60000
	LogSegmentDeleteDelayMS int64 `json:"logSegmentDeleteDelayMS"`

	// The maximum size of message that the server can receive
	// Default: 1000012
	MessagesMaxBytes int32 `json:"messagesMaxBytes"`

	// When a producer sets acks to "all" (or "-1"), min.insync.replicas specifies the minimum number of replicas that must acknowledge
	// a write for the write to be considered successful.
	// Can be overwritten at topic level
	// Default: 1
	MinInsyncReplicas int32 `json:"minInsyncReplicas"`

	// The number of io threads that the server uses for carrying out network requests
	// Default: 8
	NumIOThreads int32 `json:"numIOThreads"`

	// The number of network threads that the server uses for handling network requests
	// Default: 3
	NumNetworkThreads int32 `json:"numNetworkThreads"`

	//The number of threads per data directory to be used for log recovery at startup and flushing at shutdown
	// Default: 1
	NumRecoveryThreadsPerDataDir int32 `json:"numRecoveryThreadsPerDataDir"`

	// Number of fetcher threads used to replicate messages from a source broker.
	// Increasing this value can increase the degree of I/O parallelism in the follower broker.
	// Default: 1
	NumReplicaFetchers int32 `json:"numReplicaFetchers"`

	// The maximum size for a metadata entry associated with an offset commit.
	// Default: 4096
	OffsetMetadataMaxBytes int32 `json:"offsetMetadataMaxBytes"`

	// The required acks before the commit can be accepted. In general, the default (-1) should not be overridden
	// Default: -1
	// Commented out because of dangerous option
	//OffsetCommitReadRequiredAcks int32 `json:"offsetCommitReadRequiredAcks"`

	//Offset commit will be delayed until all replicas for the offsets topic receive the commit or this timeout is reached.
	// This is similar to the producer request timeout.
	// Default: 5000
	OffsetCommitTimeoutMs int32 `json:"offsetCommitTimeoutMs"`

	// Batch size for reading from the offsets segments when loading offsets into the cache.
	// Default: 5242880
	OffsetLoadBufferSize int32 `json:"offsetLoadBufferSize"`

	// Frequency at which to check for stale offsets
	// Default: 600000
	OffsetRetentionCheckIntervalMs int64 `json:"offsetRetentionCheckIntervalMs"`

	// Log retention window in minutes for offsets topic
	// Default: 1440
	OffsetRetentionMinutes int32 `json:"offsetRetentionMinutes"`

	// Compression codec for the offsets topic - compression may be used to achieve "atomic" commits
	// Default: 0
	//Commented out, wrong doku? int fro compression???
	//OffsetTopicCompressionCodec int32 `json:"offset_topic_compression_coded"`

	// The number of partitions for the offset commit topic (should not change after deployment)
	// Default: 50
	OffsetTopicNumPartitions int32 `json:"offsetTopicNumPartitions"`

	// The replication factor for the offsets topic (set higher to ensure availability).
	// To ensure that the effective replication factor of the offsets topic is the configured value, the number of alive brokers has to be at least the replication factor at the time of the first request for the offsets topic.
	// If not, either the offsets topic creation will fail or it will get a replication factor of min(alive brokers, configured replication factor)
	// Default: 3
	OffsetTopicReplicationFactor int32 `json:"offsetTopicReplicationFactor"`

	// The offsets topic segment bytes should be kept relatively small in order to facilitate faster log compaction and cache loads
	// Default: 104857600
	OffsetTopicSegmentsBytes int32 `json:"offsetTopicSegmentsBytes"`

	// The number of queued requests allowed before blocking the network threads
	// Default: 100
	QueuedMaxRequest int32 `json:"queuedMaxRequest"`

	// Minimum bytes expected for each fetch response. If not enough bytes, wait up to replicaMaxWaitTimeMs
	// Default: 1
	ReplicaFetchMinBytes int32 `json:"replicaFetchMinBytes"`

	//max wait time for each fetcher request issued by follower replicas.
	// This value should always be less than the replica.lag.time.max.ms at all times to prevent frequent shrinking of ISR for low throughput topics
	//Default: 500
	ReplicaFetchWaitMaxMs int32 `json:"replicaFetchWaitMaxMs"`
	
	//The frequency with which the high watermark is saved out to disk
	// Default: 5000
	ReplicaHighWatermarkCheckpointIntervalMs int64 `json:"replicaHighWatermarkCheckpointIntervalMs"`

	// If a follower hasn't sent any fetch requests or hasn't consumed up to the leaders log end offset for at least this time,
	// the leader will remove the follower from isr
	// Defaut: 10000
	ReplicaLagTimeMaxMs int64 `json:"replicaLagTimeMaxMs"`

	// The socket receive buffer for network requests
	// Default: 65536
	ReplicaSocketReceiveBufferBytes int32 `json:"replicaSocketReceiveBufferBytes"`

	// The socket timeout for network requests. Its value should be at least replica.fetch.wait.max.ms
	// Default: 30000
	ReplicaSocketTimeoutMs int32 `json:"replicaSocketTimeoutMs"`

	// The configuration controls the maximum amount of time the client will wait for the response of a request.
	// If the response is not received before the timeout elapses the client will resend the request if necessary
	// or fail the request if retries are exhausted.
	// Default: 30000
	RequestTimeoutMs int32 `json:"requestTimeoutMs"`

	//Socket Settings? TODO? needed

	// Indicates whether to enable replicas not in the ISR set to be elected as leader as a last resort, even though doing so may result in data loss
	// Default: true
	UncleanLeaderElectionEnable bool `json:"uncleanLeaderElectionEnable"`

	// The max time that the client waits to establish a connection to zookeeper.
	// If not set, the value in zookeeper.session.timeout.ms is used
	// Default: null
	ZookeeperConnectionTimeoutMs int32 `json:"zookeeperConnectionTimeoutMs"`

	// Zookeeper session timeout
	// Default: 6000
	ZookeeperSessionTimeoutMs int32 `json:"zookeeperSessionTimeoutMs"`
}

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
type KafkaClusterEvent struct {
	Type       KafkaEventType
	Cluster    KafkaCluster
	OldCluster KafkaCluster
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
	DOWNSIZE_EVENT
	//Cleanup event which get emmised after a Cluster Delete.
	//Its ensure the deletion of the Statefulset after it has been scaled down.
	CLEANUP_EVENT
	KAKFA_EVENT
)

type KafkaBrokerState string

const (
	EMPTY_BROKER     KafkaBrokerState = "EMPTYING"
	REBALANCE_BROKER KafkaBrokerState = "REBALANCING"
	NORMAL_STATE     KafkaBrokerState = "NORMAL"
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
