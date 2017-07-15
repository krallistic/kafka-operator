package kafka

import (
	"fmt"

	"github.com/Shopify/sarama"
	log "github.com/Sirupsen/logrus"
	"github.com/krallistic/kafka-operator/spec"
	"github.com/krallistic/kafka-operator/util"
	kazoo "github.com/krallistic/kazoo-go"
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

func New(clusterSpec spec.Kafkacluster) (*KafkaUtil, error) {
	brokerList := util.GetBrokerAdressess(clusterSpec)

	methodLogger := log.WithFields(log.Fields{
		"method":      "new",
		"clusterName": clusterSpec.Metadata.Name,
		"brokers":     brokerList,
	})
	config := sarama.NewConfig()

	methodLogger.Info("Creating KafkaUtil")

	kz, err := kazoo.NewKazooFromConnectionString(clusterSpec.Spec.ZookeeperConnect, nil)
	if err != nil {
		methodLogger.WithFields(log.Fields{
			"error":            err,
			"zookeeperConnect": clusterSpec.Spec.ZookeeperConnect,
		}).Error("Cant create kazoo client")
		return nil, err
	}

	brokers, err := kz.BrokerList()
	if err != nil {
		methodLogger.WithFields(log.Fields{
			"error": err,
		}).Error("Error reading brokers from zk")
		return nil, err
	}

	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		methodLogger.WithFields(log.Fields{
			"error":   err,
			"brokers": brokers,
		}).Error("Error creating sarama kafka Client")
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

func (k *KafkaUtil) GetTopicsOnBroker(cluster spec.Kafkacluster, brokerId int32) ([]string, error) {
	methodLogger := log.WithFields(log.Fields{
		"method":      "GetTopicsOnBroker",
		"clusterName": cluster.Metadata.Name,
	})
	topicConfiguration, err := k.GetTopicConfiguration(cluster)
	if err != nil {
		return nil, err
	}
	topicOnBroker := make([]string, 0)

	for _, topic := range topicConfiguration {
	partitionLoop:
		for _, partition := range topic.Partitions {
			for _, replica := range partition.Replicas {
				if replica == brokerId {
					topicOnBroker = append(topicOnBroker, topic.Topic)
					break partitionLoop
				}
			}
		}
	}
	methodLogger.WithFields(log.Fields{
		"topics": topicOnBroker,
	}).Debug("Topics on Broker")
	return topicOnBroker, nil
}

func (k *KafkaUtil) GetTopicConfiguration(cluster spec.Kafkacluster) ([]spec.KafkaTopic, error) {
	methodLogger := log.WithFields(log.Fields{
		"method":      "GetTopicConfiguration",
		"clusterName": cluster.Metadata.Name,
	})
	topics, err := k.KafkaClient.Topics()
	if err != nil {
		methodLogger.Error("Error Listing Topics")
		return nil, err
	}
	configuration := make([]spec.KafkaTopic, len(topics))
	for i, topic := range topics {

		partitions, err := k.KafkaClient.Partitions(topic)
		if err != nil {
			methodLogger.Error("Error Listing Partitions")
			return nil, err
		}
		t := spec.KafkaTopic{
			Topic:             topic,
			PartitionFactor:   int32(len(partitions)),
			ReplicationFactor: 3,
			Partitions:        make([]spec.KafkaPartition, len(partitions)),
		}
		for j, partition := range partitions {
			replicas, err := k.KafkaClient.Replicas(topic, partition)
			if err != nil {
				methodLogger.Error("Error listing partitions")
				return nil, err
			}
			t.Partitions[j] = spec.KafkaPartition{
				Partition: int32(j),
				Replicas:  replicas,
			}
		}
		configuration[i] = t
	}
	return configuration, nil
}

func (k *KafkaUtil) RemoveTopicFromBrokers(cluster spec.Kafkacluster, brokerToDelete int32, topic string) error {
	methodLogger := log.WithFields(log.Fields{
		"method":        "RemoveTopicFromBrokers",
		"clusterName":   cluster.Metadata.Name,
		"brokerToDelte": brokerToDelete,
		"topic":         topic,
	})

	brokersToDelete := []int32{brokerToDelete}
	err := k.KazooClient.RemoveTopicFromBrokers(topic, brokersToDelete)
	if err != nil {
		methodLogger.Warn("Error removing topic from Broker", err)
		return err
	}
	return nil
}

func (k *KafkaUtil) RemoveTopicsFromBrokers(cluster spec.Kafkacluster, brokerToDelete int32) error {
	methodLogger := log.WithFields(log.Fields{
		"method":        "RemoveTopicsFromBrokers",
		"clusterName":   cluster.Metadata.Name,
		"brokerToDelte": brokerToDelete,
	})
	topics, err := k.KafkaClient.Topics()
	if err != nil {
		methodLogger.Error("Error Listing Topics")
		return err
	}

	//TODO it should be possible to Delete multiple Brokers
	for _, topic := range topics {
		//TODO what do in cases where ReplicationFactor > remaining broker count
		k.RemoveTopicFromBrokers(cluster, brokerToDelete, topic)
	}

	return nil
}

func (k *KafkaUtil) AllTopicsInSync() (bool, error) {
	topics, err := k.KazooClient.Topics()
	if err != nil {
		return false, err
	}
	for _, topic := range topics {
		partitions, err := topic.Partitions()
		if err != nil {
			return false, err
		}
		for _, partition := range partitions {
			underReplicated, err := partition.UnderReplicated()
			if err != nil {
				return false, err
			}
			if underReplicated {
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
