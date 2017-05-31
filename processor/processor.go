package processor

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/krallistic/kafka-operator/controller"
	"github.com/krallistic/kafka-operator/kafka"
	spec "github.com/krallistic/kafka-operator/spec"
	"github.com/krallistic/kafka-operator/util"
	k8sclient "k8s.io/client-go/kubernetes"
	"time"
)

type Processor struct {
	client          k8sclient.Clientset
	baseBrokerImage string
	util            util.ClientUtil
	tprController   controller.ThirdPartyResourceController
	kafkaClusters   map[string]*spec.KafkaCluster
	watchEvents     chan spec.KafkaClusterWatchEvent
	clusterEvents   chan spec.KafkaClusterEvent
	kafkaClient     map[string]*kafka.KafkaUtil
	control         chan int
	errors          chan error
}

func New(client k8sclient.Clientset, image string, util util.ClientUtil, tprClient controller.ThirdPartyResourceController, control chan int) (*Processor, error) {
	p := &Processor{
		client:          client,
		baseBrokerImage: image,
		util:            util,
		kafkaClusters:   make(map[string]*spec.KafkaCluster),
		watchEvents:     make(chan spec.KafkaClusterWatchEvent, 100),
		clusterEvents:   make(chan spec.KafkaClusterEvent, 100),
		tprController:   tprClient,
		kafkaClient:     make(map[string]*kafka.KafkaUtil),
		control:         control,
		errors:          make(chan error),
	}
	fmt.Println("Created Processor")
	return p, nil
}

func (p *Processor) Run() error {
	//TODO getListOfAlredyRunningCluster/Refresh
	fmt.Println("Running Processor")
	p.watchKafkaEvents()

	return nil
}

//We detect basic change through the event type, beyond that we use the API server to find differences.
//Functions compares the KafkaClusterSpec with the real Pods/Services which are there.
//We do that because otherwise we would have to use a local state to track changes.
func (p *Processor) DetectChangeType(event spec.KafkaClusterWatchEvent) spec.KafkaClusterEvent {
	fmt.Println("DetectChangeType: ", event)
	methodLogger := log.WithFields(log.Fields{
		"method":       "DetectChangeType",
		"clusterName":  event.Object.Metadata.Name,
		"eventType":	event.Type,
	})
	methodLogger.Info("Detecting type of change in Kafka TPR")

	//TODO multiple changes in one Update? right now we only detect one change
	clusterEvent := spec.KafkaClusterEvent{
		Cluster: event.Object,
	}
	if event.Type == "ADDED" {
		clusterEvent.Type = spec.NEW_CLUSTER
		return clusterEvent
	}
	if event.Type == "DELETED" {
		clusterEvent.Type = spec.DELTE_CLUSTER
		return clusterEvent
		//EVENT type must be modfied now
	}  else if p.util.BrokerStSImageUpdate(event.OldObject, event.Object) {
		clusterEvent.Type = spec.CHANGE_IMAGE
		return clusterEvent
	} else if p.util.BrokerStSUpsize(event.OldObject, event.Object) {
		clusterEvent.Type = spec.UPSIZE_CLUSTER
		return clusterEvent
	} else if p.util.BrokerStSDownsize(event.OldObject, event.Object) {
		clusterEvent.Type = spec.DOWNSIZE_CLUSTER
		return clusterEvent
	} else if p.util.BrokerStatefulSetExist(event.Object) {
		clusterEvent.Type = spec.UNKNOWN_CHANGE
		//TODO change to reconsilation event?
		return clusterEvent
	}

	clusterEvent.Type = spec.UNKNOWN_CHANGE
	return clusterEvent
}

func (p *Processor) initKafkaClient(cluster spec.KafkaCluster) error {
	methodLogger := log.WithFields(log.Fields{
		"method":            "initKafkaClient",
		"clusterName":       cluster.Metadata.Name,
		"zookeeperConnectL": cluster.Spec.ZookeeperConnect,
	})
	methodLogger.Info("Creating KafkaCLient for cluster")

	client, err := kafka.New(cluster)
	if err != nil {
		return err
	}

	//TODO can metadata.uuid used? check how that changed
	name := p.GetClusterUUID(cluster)
	p.kafkaClient[name] = client

	methodLogger.Info("Create KakfaClient for cluser")
	return nil
}

func (p *Processor) GetClusterUUID(cluster spec.KafkaCluster) string{
	return cluster.Metadata.Namespace + "-" + cluster.Metadata.Name
}

//Takes in raw Kafka events, lets then detected and the proced to initiate action accoriding to the detected event.
func (p *Processor) processKafkaEvent(currentEvent spec.KafkaClusterEvent) {
	fmt.Println("Recieved Event, proceeding: ", currentEvent)
	methodLogger := log.WithFields(log.Fields{
		"method":            "processKafkaEvent",
		"clusterName":       currentEvent.Cluster.Metadata.Name,
		"KafkaClusterEventType": currentEvent.Type,
	})
	switch currentEvent.Type {
	case spec.NEW_CLUSTER:
		fmt.Println("ADDED")
		clustersTotal.Inc()
		clustersCreated.Inc()
		p.CreateKafkaCluster(currentEvent.Cluster)
		clusterEvent := spec.KafkaClusterEvent{
			Cluster: currentEvent.Cluster,
			Type:    spec.KAKFA_EVENT,
		}
		methodLogger.Info("Init heartbeat type checking...")
		p.Sleep30AndSendEvent(clusterEvent)
		break

	case spec.DELTE_CLUSTER:
		methodLogger.Info("Delete Cluster, deleting all Objects ")
		if p.util.DeleteKafkaCluster(currentEvent.Cluster) != nil {
			//Error while deleting, just resubmit event after wait time.
			p.Sleep30AndSendEvent(currentEvent)
			break
		}

		go func() {
			time.Sleep(time.Duration(currentEvent.Cluster.Spec.BrokerCount) * time.Minute)
			//TODO dynamic sleep, depending till sts is completely scaled down.
			clusterEvent := spec.KafkaClusterEvent{
				Cluster: currentEvent.Cluster,
				Type:    spec.CLEANUP_EVENT,
			}
			p.clusterEvents <- clusterEvent
		}()
		clustersTotal.Dec()
		clustersDeleted.Inc()
	case spec.CHANGE_IMAGE:
		fmt.Println("Change Image, updating StatefulSet should be enough to trigger a new Image Rollout")
		if p.util.UpdateBrokerImage(currentEvent.Cluster) != nil {
			//Error updating
			p.Sleep30AndSendEvent(currentEvent)
			break
		}
		clustersModified.Inc()
	case spec.UPSIZE_CLUSTER:
		fmt.Println("Upsize Cluster, changing StatefulSet with higher Replicas, no Rebalacing")
		p.util.UpsizeBrokerStS(currentEvent.Cluster)
		clustersModified.Inc()
	case spec.UNKNOWN_CHANGE:
		methodLogger.Warn("Unkown (or unsupported) change occured, doing nothing. Maybe manually check the cluster")
		clustersModified.Inc()
	case spec.DOWNSIZE_CLUSTER:
		fmt.Println("Downsize Cluster")
		//TODO remove poor mans casting :P
		//TODO support Downsizing Multiple Brokers
		brokerToDelete := currentEvent.Cluster.Spec.BrokerCount - 0
		fmt.Println("Downsizing Broker, deleting Data on Broker: ", brokerToDelete)
		p.util.SetBrokerState(currentEvent.Cluster, brokerToDelete, "deleting")
		err := p.kafkaClient[p.GetClusterUUID(currentEvent.Cluster)].RemoveTopicsFromBrokers(currentEvent.Cluster, brokerToDelete)

		states, err := p.util.GetBrokerStates(currentEvent.Cluster)
		if err != nil {
			//just re-try delete event
			p.Sleep30AndSendEvent(currentEvent)
			break
		}
		p.EmptyingBroker(currentEvent.Cluster, states)

		clustersModified.Inc()
	case spec.CHANGE_ZOOKEEPER_CONNECT:
		methodLogger.Warn("Trying to change zookeeper connect, not supported currently")
		clustersModified.Inc()
	case spec.CLEANUP_EVENT:
		fmt.Println("Recieved CleanupEvent, force delete of StatefuleSet.")
		clustersModified.Inc()
	case spec.KAKFA_EVENT:
		fmt.Println("Kafka Event, heartbeat etc..")
		p.Sleep30AndSendEvent(currentEvent)
	case spec.DOWNSIZE_EVENT:
		methodLogger.Info("Got Downsize Event, checking if all Topics are fully replicated and no topic on to delete cluster")
		//GET CLUSTER TO DELETE
		toDelete, err := p.util.GetBrokersWithState(currentEvent.Cluster, spec.EMPTY_BROKER)
		if err != nil {
			p.Sleep30AndSendEvent(currentEvent)
		}
		kafkaClient := p.kafkaClient[p.GetClusterUUID(currentEvent.Cluster)]
		topics, err := kafkaClient.GetTopicsOnBroker(currentEvent.Cluster, int32(toDelete))
		if len(topics) > 0 {
			//Move topics from Broker
			methodLogger.Warn("New Topics found on Broker which should be deleted, moving Topics Off", topics)
			for _, topic := range topics {
				kafkaClient.RemoveTopicFromBrokers(currentEvent.Cluster, toDelete, topic)
			}

			break
		}
		if err != nil {
			p.Sleep30AndSendEvent(currentEvent)
		}
		//CHECK if all Topics has been moved off
		inSync, err := p.kafkaClient[p.GetClusterUUID(currentEvent.Cluster)].AllTopicsInSync()
		if err != nil || !inSync {
			p.Sleep30AndSendEvent(currentEvent)
			break
		}


		//states := p.util.GetPodAnnotations(currentEvent.Cluster)
		//name := currentEvent.Cluster.Metadata.Namespace + "-" + currentEvent.Cluster.Metadata.Name
		//p.kafkaClient[name].PrintFullStats()

	}
}

func (p *Processor) Sleep30AndSendEvent(currentEvent spec.KafkaClusterEvent) {
	p.SleepAndSendEvent(currentEvent, 30)
}

func (p *Processor) SleepAndSendEvent(currentEvent spec.KafkaClusterEvent, seconds int) {
	go func() {
		time.Sleep(time.Second * time.Duration(seconds))
		p.clusterEvents <- currentEvent
	}()
}
func (p *Processor) EmptyingBroker(cluster spec.KafkaCluster, states []string) error {

	for i, state := range states {
		fmt.Println("State, Index: ", state, i)
		if state == "toDelete" {
			// EMPTY Broker,
			// generate Downsize Options
			// Save downsize option and store in k8s
		} else if state == "deleting" {
			//get downsize option from k8s
			//check if downsize done
		} else if state == "deleted" {
			//downsize Broker

		} else {
			//DO nothing?
		}
	}


	return nil
}

//Creates inside a goroutine a watch channel on the KafkaCLuster Endpoint and distibutes the events.
//control chan used for showdown events from outside
func (p *Processor) watchKafkaEvents() {

	p.tprController.MonitorKafkaEvents(p.watchEvents, p.control)
	fmt.Println("Watching Kafka Events")
	go func() {
		for {

			select {
			case currentEvent := <-p.watchEvents:
				classifiedEvent := p.DetectChangeType(currentEvent)
				p.clusterEvents <- classifiedEvent
			case clusterEvent := <-p.clusterEvents:
				p.processKafkaEvent(clusterEvent)
			case err := <-p.errors:
				println("Error Channel", err)
			case <-p.control:
				fmt.Println("Recieved Something on Control Channel, shutting down: ")
				return
			}
		}
	}()
}

//Create the KafkaCluster, with the following components: Service, Volumes, StatefulSet.
//Maybe move this also into util
func (p *Processor) CreateKafkaCluster(clusterSpec spec.KafkaCluster) {
	fmt.Println("CreatingKafkaCluster", clusterSpec)
	fmt.Println("SPEC: ", clusterSpec.Spec)

	suffix := ".cluster.local:9092"
	brokerNames := make([]string, clusterSpec.Spec.BrokerCount)

	headless_SVC_Name := clusterSpec.Metadata.Name
	round_robing_dns := headless_SVC_Name + suffix
	fmt.Println("Headless Service Name: ", headless_SVC_Name, " Should be accessable through LB: ", round_robing_dns)

	var i int32
	for i = 0; i < clusterSpec.Spec.BrokerCount; i++ {
		brokerNames[i] = "kafka-0." + headless_SVC_Name + suffix
		fmt.Println("Broker", i, " ServiceName: ", brokerNames[i])
	}

	//Create Headless Brokersvc
	//TODO better naming
	p.util.CreateBrokerService(clusterSpec, true)

	//TODO createVolumes

	//CREATE Broker sts
	//Currently we extract name out of spec, maybe move to metadata to be more inline with other k8s komponents.
	p.util.CreateBrokerStatefulSet(clusterSpec)

	p.util.CreateDirectBrokerService(clusterSpec)

	p.initKafkaClient(clusterSpec)

}
