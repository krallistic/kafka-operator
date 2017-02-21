package processor

import (
	k8sclient "k8s.io/client-go/kubernetes"
	kafkaOperatorSpec "github.com/krallistic/kafka-operator/spec"
	"fmt"
)

type Processor struct {
	client k8sclient.Clientset
	baseBrokerImage string
	kafkaClusters map[string]*kafkaOperatorSpec.KafkaCluster
}

func New(client k8sclient.Clientset, image string) (*Processor, error) {
	p := &Processor{
		client:client,
		baseBrokerImage:image,
		kafkaClusters:make(map[string]*kafkaOperatorSpec.KafkaCluster),
	}
	fmt.Println("Created Processor")
	return p, nil
}

func ( p *Processor) Run() error {
	//TODO getListOfAlredyRunningCluster/Refresh
	fmt.Println("Running Processor")
	return nil
}
