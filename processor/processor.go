package processor

import (
	"fmt"
	"reflect"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/krallistic/kafka-operator/controller"
	"github.com/krallistic/kafka-operator/kafka"
	spec "github.com/krallistic/kafka-operator/spec"
	"github.com/krallistic/kafka-operator/util"
	k8sclient "k8s.io/client-go/kubernetes"
)

type Processor struct {
	client          k8sclient.Clientset
	baseBrokerImage string
	util            util.ClientUtil
	tprController   controller.CustomResourceController
	kafkaClusters   map[string]*spec.Kafkacluster
	watchEvents     chan spec.KafkaclusterWatchEvent
	clusterEvents   chan spec.KafkaclusterEvent
	kafkaClient     map[string]*kafka.KafkaUtil
	control         chan int
	errors          chan error
}

func New(client k8sclient.Clientset, image string, util util.ClientUtil, tprClient controller.CustomResourceController, control chan int) (*Processor, error) {
	p := &Processor{
		client:          client,
		baseBrokerImage: image,
		util:            util,
		kafkaClusters:   make(map[string]*spec.Kafkacluster),
		watchEvents:     make(chan spec.KafkaclusterWatchEvent, 100),
		clusterEvents:   make(chan spec.KafkaclusterEvent, 100),
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
	fmt.Println("Watching")
	return nil
}

//We detect basic change through the event type, beyond that we use the API server to find differences.
//Functions compares the KafkaClusterSpec with the real Pods/Services which are there.
//We do that because otherwise we would have to use a local state to track changes.
func (p *Processor) DetectChangeType(event spec.KafkaclusterWatchEvent) spec.KafkaclusterEvent {
	methodLogger := log.WithFields(log.Fields{
		"method":      "DetectChangeType",
		"clusterName": event.Object.ObjectMeta.Name,
		"eventType":   event.Type,
	})
	methodLogger.Debug("Detecting type of change in Kafka CRD")

	//TODO multiple changes in one Update? right now we only detect one change
	clusterEvent := spec.KafkaclusterEvent{
		Cluster: event.Object,
	}
	if event.Type == "ADDED" {
		clusterEvent.Type = spec.NEW_CLUSTER
		return clusterEvent
	}
	if event.Type == "DELETED" {
		clusterEvent.Type = spec.DELETE_CLUSTER
		return clusterEvent

	}
	//EVENT type must be modfied now.
	oldCluster := event.OldObject
	newCluster := event.Object

	if reflect.DeepEqual(oldCluster, spec.Kafkacluster{}) {
		methodLogger.Error("Got changed type, but either new or old object is nil")
		clusterEvent.Type = spec.ERROR_STATE
		return clusterEvent
	}

	methodLogger = methodLogger.WithFields(log.Fields{
		"oldCluster": oldCluster,
		"newCluster": newCluster,
	})

	clusterEvent.OldCluster = event.OldObject

	if !reflect.DeepEqual(oldCluster.State, newCluster.State) {
		methodLogger.Debug("Cluster State different, doing nothing")
		clusterEvent.Type = spec.STATE_CHANGE
		return clusterEvent
	} else if !reflect.DeepEqual(oldCluster.State, newCluster.State) {
		methodLogger.Debug("Cluster Scale different, doing nothing")
		clusterEvent.Type = spec.SCALE_CHANGE
		return clusterEvent

	} else if oldCluster.Spec.Image != newCluster.Spec.Image {
		clusterEvent.Type = spec.CHANGE_IMAGE
		return clusterEvent
	} else if oldCluster.Spec.BrokerCount < newCluster.Spec.BrokerCount {
		clusterEvent.Type = spec.UPSIZE_CLUSTER
		return clusterEvent
	} else if oldCluster.Spec.BrokerCount > newCluster.Spec.BrokerCount {
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

func (p *Processor) initKafkaClient(cluster spec.Kafkacluster) error {
	methodLogger := log.WithFields(log.Fields{
		"method":            "initKafkaClient",
		"clusterName":       cluster.ObjectMeta.Name,
		"zookeeperConnectL": cluster.Spec.ZookeeperConnect,
	})
	methodLogger.Info("Creating KafkaCLient for cluster")

	client, err := kafka.New(cluster)
	if err != nil {
		internalErrors.Inc()
		return err
	}

	//TODO can metadata.uuid used? check how that changed
	name := p.getClusterUUID(cluster)
	p.kafkaClient[name] = client

	methodLogger.Info("Create KakfaClient for cluser")
	return nil
}

func (p *Processor) getClusterUUID(cluster spec.Kafkacluster) string {
	return cluster.ObjectMeta.Namespace + "-" + cluster.ObjectMeta.Name
}

//Takes in raw Kafka events, lets then detected and the proced to initiate action accoriding to the detected event.
func (p *Processor) processKafkaEvent(currentEvent spec.KafkaclusterEvent) {
	fmt.Println("Recieved Event, proceeding: ", currentEvent)
	methodLogger := log.WithFields(log.Fields{
		"method":                "processKafkaEvent",
		"clusterName":           currentEvent.Cluster.ObjectMeta.Name,
		"KafkaClusterEventType": currentEvent.Type,
	})
	switch currentEvent.Type {
	case spec.NEW_CLUSTER:
		fmt.Println("ADDED")
		clustersTotal.Inc()
		clustersCreated.Inc()
		p.CreateKafkaCluster(currentEvent.Cluster)
		clusterEvent := spec.KafkaclusterEvent{
			Cluster: currentEvent.Cluster,
			Type:    spec.KAKFA_EVENT,
		}

		methodLogger.Info("Init heartbeat type checking...")
		p.sleep30AndSendEvent(clusterEvent)
		break

	case spec.DELETE_CLUSTER:
		methodLogger.Info("Delete Cluster, deleting all Objects ")
		if p.util.DeleteKafkaCluster(currentEvent.Cluster) != nil {
			//Error while deleting, just resubmit event after wait time.
			p.sleep30AndSendEvent(currentEvent)
			break
		}

		go func() {
			time.Sleep(time.Duration(currentEvent.Cluster.Spec.BrokerCount) * time.Minute)
			//TODO dynamic sleep, depending till sts is completely scaled down.
			clusterEvent := spec.KafkaclusterEvent{
				Cluster: currentEvent.Cluster,
				Type:    spec.CLEANUP_EVENT,
			}
			p.clusterEvents <- clusterEvent
		}()
		p.util.DeleteOffsetMonitor(currentEvent.Cluster)
		clustersTotal.Dec()
		clustersDeleted.Inc()
	case spec.CHANGE_IMAGE:
		fmt.Println("Change Image, updating StatefulSet should be enough to trigger a new Image Rollout")
		if p.util.UpdateBrokerImage(currentEvent.Cluster) != nil {
			//Error updating
			internalErrors.Inc()
			p.sleep30AndSendEvent(currentEvent)
			break
		}
		clustersModified.Inc()
	case spec.UPSIZE_CLUSTER:
		methodLogger.Warn("Upsize Cluster, changing StatefulSet with higher Replicas, no Rebalacing")
		p.util.UpsizeBrokerStS(currentEvent.Cluster)
		clustersModified.Inc()
	case spec.UNKNOWN_CHANGE:
		methodLogger.Warn("Unknown (or unsupported) change occured, doing nothing. Maybe manually check the cluster")
		clustersModified.Inc()
	case spec.DOWNSIZE_CLUSTER:
		fmt.Println("Downsize Cluster")
		//TODO remove poor mans casting :P
		//TODO support Downsizing Multiple Brokers
		brokerToDelete := currentEvent.Cluster.Spec.BrokerCount - 0
		methodLogger.Info("Downsizing Broker, deleting Data on Broker: ", brokerToDelete)

		err := p.util.SetBrokerState(currentEvent.Cluster, brokerToDelete, spec.EMPTY_BROKER)
		if err != nil {
			//just re-try delete event
			internalErrors.Inc()
			p.sleep30AndSendEvent(currentEvent)
			break
		}

		err = p.kafkaClient[p.getClusterUUID(currentEvent.Cluster)].RemoveTopicsFromBrokers(currentEvent.Cluster, brokerToDelete)
		if err != nil {
			//just re-try delete event
			internalErrors.Inc()
			p.sleep30AndSendEvent(currentEvent)
			break
		}
		clustersModified.Inc()
	case spec.CHANGE_ZOOKEEPER_CONNECT:
		methodLogger.Warn("Trying to change zookeeper connect, not supported currently")
		clustersModified.Inc()
	case spec.CLEANUP_EVENT:
		fmt.Println("Recieved CleanupEvent, force delete of StatefuleSet.")
		p.util.CleanupKafkaCluster(currentEvent.Cluster)
		clustersModified.Inc()
	case spec.KAKFA_EVENT:
		fmt.Println("Kafka Event, heartbeat etc..")
		p.sleep30AndSendEvent(currentEvent)
	case spec.DOWNSIZE_EVENT:
		methodLogger.Info("Got Downsize Event, checking if all Topics are fully replicated and no topic on to delete cluster")
		//GET CLUSTER TO DELETE
		toDelete, err := p.util.GetBrokersWithState(currentEvent.Cluster, spec.EMPTY_BROKER)
		if err != nil {
			internalErrors.Inc()
			p.sleep30AndSendEvent(currentEvent)
		}
		kafkaClient := p.kafkaClient[p.getClusterUUID(currentEvent.Cluster)]
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
			internalErrors.Inc()
			p.sleep30AndSendEvent(currentEvent)
		}
		//CHECK if all Topics has been moved off
		inSync, err := p.kafkaClient[p.getClusterUUID(currentEvent.Cluster)].AllTopicsInSync()
		if err != nil || !inSync {
			internalErrors.Inc()
			p.sleep30AndSendEvent(currentEvent)
			break
		}

	}
}

func (p *Processor) sleep30AndSendEvent(currentEvent spec.KafkaclusterEvent) {
	p.sleepAndSendEvent(currentEvent, 30)
}

func (p *Processor) sleepAndSendEvent(currentEvent spec.KafkaclusterEvent, seconds int) {
	go func() {
		time.Sleep(time.Second * time.Duration(seconds))
		p.clusterEvents <- currentEvent
	}()
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

// CreateKafkaCluster with the following components: Service, Volumes, StatefulSet.
//Maybe move this also into util
func (p *Processor) CreateKafkaCluster(clusterSpec spec.Kafkacluster) {
	methodLogger := log.WithFields(log.Fields{
		"method":      "CreateKafkaCluster",
		"clusterName": clusterSpec.ObjectMeta.Name,
	})

	err := p.util.CreateBrokerStatefulSet(clusterSpec)
	if err != nil {
		methodLogger.WithField("error", err).Fatal("Cant create statefulset")
	}

	err = p.util.CreateBrokerService(clusterSpec)
	if err != nil {
		methodLogger.WithField("error", err).Fatal("Cant create loadbalacend headless service services")
	}

	err = p.util.CreateDirectBrokerService(clusterSpec)
	if err != nil {
		methodLogger.WithField("error", err).Fatal("Cant create direct broker services")
	}
	//TODO createVolumes here?

	p.initKafkaClient(clusterSpec)

	err = p.util.DeployOffsetMonitor(clusterSpec)
	if err != nil {
		methodLogger.WithField("error", err).Fatal("Cant deploy stats exporter")
	}
}
