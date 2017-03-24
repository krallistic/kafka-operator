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

func (k *KafkaUtil) ListTopics() ([]string, error) {
	fmt.Println("Listing KafkaTopics")
	topics, err := k.KafkaClient.Topics()
	if err != nil {
		return nil, err
	}
//	var t string
//	for t = range topics {
//		fmt.Println("Current topic:", t)
//	}
	return topics, nil
}


func (k *KafkaUtil) CreateTopic(topicSpec spec.KafkaTopicSpec) error {
	fmt.Println("Creating Kafka Topics: ", topicSpec)
	broker, _ := k.KafkaClient.Coordinator("operatorConsumerGroup")
	request := sarama.MetadataRequest{Topics: []string{topicSpec.Name}}
	metadataPartial, err := broker.GetMetadata(&request)
	if err != nil {
		return err
	}

	replicas := []int32{0}
	isr := []int32{0}

	metadataResponse := &sarama.MetadataResponse{}
	metadataResponse.AddBroker(broker.Addr(), broker.ID())

	metadataPartial.AddTopic(topicSpec.Name, sarama.ErrNoError)
	//TODO dynamic partitions
	metadataPartial.AddTopicPartition(topicSpec.Name, 0, broker.ID(), replicas, isr, sarama.ErrNoError)
	metadataPartial.AddTopicPartition(topicSpec.Name, 1, broker.ID(), replicas, isr, sarama.ErrNoError)

	return nil
}
