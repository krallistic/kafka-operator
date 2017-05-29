package kafka

import (
	"fmt"
	"github.com/Shopify/sarama"
	log "github.com/Sirupsen/logrus"
	"github.com/krallistic/kafka-operator/spec"
	kazoo "github.com/krallistic/kazoo-go"
	"github.com/krallistic/kafka-operator/util"
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
	KazooClient *kazoo.Kazoo
}

func New(clusterSpec spec.KafkaCluster) (*KafkaUtil, error) {
	brokerList := util.GetBrokerAdressess(clusterSpec)

	methodLogger := log.WithFields(log.Fields{
		"method":      "new",
		"clusterName": clusterSpec.Metadata.Name,
		"brokers":     brokerList,
	})
	config := sarama.NewConfig()

	methodLogger.Info("Creating KafkaUtil")

	client, err := sarama.NewClient(brokerList, config)
	if err != nil {
		methodLogger.WithFields(log.Fields{
			"error": err,
		}).Error("Error creating sarama kafka Client")
		return nil, err
	}

	kz, err := kazoo.NewKazooFromConnectionString(clusterSpec.Spec.ZookeeperConnect, nil)
	if err != nil {
		methodLogger.WithFields(log.Fields{
			"error": err,
			"zookeeperConnect": clusterSpec.Spec.ZookeeperConnect,
		}).Error("Cant create kazoo client")
		return nil, err
	}

	k := &KafkaUtil{
		KafkaClient: client,
		ClusterName: clusterSpec.Metadata.Name,
		BrokerList:  brokerList,
		KazooClient: kz,
	}

	methodLogger.Info("Initilized Kafka CLient, KazooClient and created KafkaUtil")
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
		return nil, err
	}
	return partitions, nil
}

func (k *KafkaUtil) PrintFullStats() error {
	topics, err := k.ListTopics()
	if err != nil {
		return err
	}
	for _, topic := range topics {
		partitions, err := k.GetPartitions(topic)
		if err != nil {
			return err
		}
		fmt.Println("Topic: %s, Partitions %s", topic, partitions)
	}

	return nil
}

func (k *KafkaUtil) RemoveTopicsFromBrokers(cluster spec.KafkaCluster, brokerToDelete int32) (spec.KafkaReassignmentConfig, error) {
	methodLogger := log.WithFields(log.Fields{
		"method":      "GenerateReassign",
		"clusterName": cluster.Metadata.Name,
	})
	topics, err := k.KafkaClient.Topics()
	if err != nil {
		methodLogger.Error("Error Listing Topics")
		return spec.KafkaReassignmentConfig{}, err
	}

	//TODO it should be possible to Delete multiple Brokers
	brokersToDelete := []int32{brokerToDelete}
	for _, topic := range topics {
		k.KazooClient.RemoveTopicFromBrokers(topic, brokersToDelete)
		partitions, err := k.KafkaClient.Partitions(topic)
		if err != nil {
			methodLogger.Error("Error Listing Partitions")
			return spec.KafkaReassignmentConfig{}, err
		}
		for _, partition := range partitions {
			partition, err := k.KafkaClient.Replicas(topic, partition)
			if err != nil {
				methodLogger.Error("Error listing partitions")
				return spec.KafkaReassignmentConfig{}, err
			}
			fmt.Println(partition)
		}
	}

	return spec.KafkaReassignmentConfig{}, nil
}

func (k *KafkaUtil) AllTopicsInSync() (bool, error) {
	//TODO error checking
	topics, err := k.KazooClient.Topics()
	if err != nil {
		return nil, err
	}
	for _, topic := range topics {
		partitions, err := topic.Partitions()
		if err != nil {
			return nil, err
		}
		for _, partition := range partitions {
			underReplicated, err := partition.UnderReplicated()
			if err != nil {
				return nil, err
			}
			if underReplicated{
				return false, nil
			}
		}
	}
	return true, nil
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
