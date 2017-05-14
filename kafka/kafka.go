package kafka

import (
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/krallistic/kafka-operator/spec"
	log "github.com/Sirupsen/logrus"
)

var (
	logger = log.WithFields(log.Fields{
		"package": "kafka",
	})
)


type KafkaUtil struct {
	KafkaClient sarama.Client
	BrokerList  []string
	ClusterName string
}

func New(brokerList []string, clusterName string) (*KafkaUtil, error) {
	methodLogger := log.WithFields(log.Fields{
		"method" : "new",
		"clusterName": clusterName,
		"brokers": brokerList,
	})
	config := sarama.NewConfig()

	methodLogger.Info("Creating KafkaUtil")

	client, err := sarama.NewClient(brokerList, config)
	if err != nil {
		methodLogger.WithFields(log.Fields{
			"error": err,
		}).Error("Error creating sarama kafka Client")
		return nil ,err
	}

	k := &KafkaUtil{
		KafkaClient: client,
		ClusterName: clusterName,
		BrokerList:  brokerList,
	}

	methodLogger.Info("Initilized Kafka CLient and created KafkaUtil")
	k.ListTopics()
	return k, nil
}

func (k *KafkaUtil) ListTopics() ([]string, error) {
	fmt.Println("Listing KafkaTopics")
	topics, err := k.KafkaClient.Topics()
	if err != nil {
		return nil, err
	}

	for _, t := range topics {
		fmt.Println("Current topic:", t)
	}
	return topics, nil
}

func (k *KafkaUtil) GetPartitions(topic string) ([]int32, error) {
	partitions, err := k.KafkaClient.Partitions(topic)
	if err != nil {
		return nil ,err
	}
	return partitions, nil
}

func (k *KafkaUtil) PrintFullStats() error {
	topics, err := k.ListTopics()
	if err != nil {
		return err
	}
	for _, topic := range topics {
		partitions,  err  := k.GetPartitions(topic)
		if err != nil {
			return err
		}
		fmt.Println("Topic: %s, Partitions %s", topic, partitions)
	}

	return nil
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
