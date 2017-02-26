package spec


type KafkaCluster struct {
	APIVersion string `json:"apiVersion"`
	Kind string `json:"kind"`
	Metadata map[string]string `json:"metadata"`
	Spec KafkaClusterSpec `json:"spec"`
}

type KafkaClusterSpec struct {
	//Amount of Broker Nodes
	BrokerCount int32 `json:"brokerCount"`
	Image string `json:"image"`
	Name string `json:"name"`
	
	Brokers ClusterBrokerSpec `json:"brokers"`
	KafkaOptions KafkaOption `json:"kafkaOptions"`
	jmxSidecar bool `json:"jmxSidecar"`
	
	ZookeeperConnect string `json:"zookeeperConnect"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	
}

type ClusterBrokerSpec struct {
	Count int `json:"count"`
	Memory int `json:"memory"`
	DiskSpace int `json:"diskSpace"` //TODO Option to use GB etc
	CPU int `json:"cpu"`
}

type KafkaOption struct {
	LogRetentionHours int `json:"logRetentionHours"`
}