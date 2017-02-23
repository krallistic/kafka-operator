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
	
	jmxSidecar bool `json:"jmxSidecar"`
	
	ZookeeperConnect string `json:"zookeeperConnect"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	
}