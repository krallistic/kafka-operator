package spec


type KafkaCluster struct {
	APIVersion string `json:"apiVersion"`
	Kind string `json:"kind"`
	Metadata map[string]string `json:"metadata"`
	Spec KafkaClusterSpec `json:"spec"`
}

type KafkaClusterSpec struct {
	//Amount of Broker Nodes
	BrokerNodes int32 `json:"brokerNodes"`
	
	ZookeeperConnect string `json:"zookeeperConnect"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}