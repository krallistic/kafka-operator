package kafka

import (
	"github.com/Shopify/sarama"
	"fmt"
	"github.com/krallistic/kafka-operator/spec"
)

type KafkaUtil struct {
	KafkaClient sarama.Client
	BrokerList []string
	ClusterName string
}


func  New(brokerList []string, clusterName string) (*KafkaUtil, error){
	config := sarama.NewConfig()

	fmt.Println("Creating KafkaUtil for cluster with Brokers: ", clusterName, brokerList )

	client, err := sarama.NewClient(brokerList, config)
	if err != nil {
		fmt.Println("Error creating Kafka Client: ", err)
	}

	k := &KafkaUtil{
		KafkaClient: client,
		ClusterName: clusterName,
		BrokerList: brokerList,
	}

	fmt.Println("Initilized Kafka CLient")
	return k, nil
}

func (k *KafkaUtil) CreateTopic(topicSpec spec.KafkaTopic)
