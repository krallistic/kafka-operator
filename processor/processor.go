package processor

import (
	"reflect"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/krallistic/kafka-operator/controller"
	"github.com/krallistic/kafka-operator/kafka"
	"github.com/krallistic/kafka-operator/kube"
	spec "github.com/krallistic/kafka-operator/spec"

	cruisecontrol_kube "github.com/krallistic/kafka-operator/kube/cruisecontrol"
	exporter_kube "github.com/krallistic/kafka-operator/kube/exporter"
	kafka_kube "github.com/krallistic/kafka-operator/kube/kafka"
)

type Processor struct {
	baseBrokerImage    string
	crdController      controller.CustomResourceController
	kafkaClusters      map[string]*spec.Kafkacluster
	watchEventsChannel chan spec.KafkaclusterWatchEvent
	clusterEvents      chan spec.KafkaclusterEvent
	kafkaClient        map[string]*kafka.KafkaUtil
	control            chan int
	errors             chan error
	kube               kube.Kubernetes
}

func New(image string,
	crdClient controller.CustomResourceController,
	control chan int,
	kube kube.Kubernetes,
) (*Processor, error) {
	p := &Processor{
		baseBrokerImage:    image,
		kafkaClusters:      make(map[string]*spec.Kafkacluster),
		watchEventsChannel: make(chan spec.KafkaclusterWatchEvent, 100),
		clusterEvents:      make(chan spec.KafkaclusterEvent, 100),
		crdController:      crdClient,
		kafkaClient:        make(map[string]*kafka.KafkaUtil),
		control:            control,
		errors:             make(chan error),
		kube:               kube,
	}
	log.Info("Created Processor")
	return p, nil
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

func (p *Processor) Run() error {
	//TODO getListOfAlredyRunningCluster/Refresh
	log.Info("Running Processor")
	p.watchEvents()
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
	} else if !reflect.DeepEqual(oldCluster.Scale, newCluster.Scale) {
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
	} else {
		clusterEvent.Type = spec.UNKNOWN_CHANGE
		//TODO change to reconsilation event?
		methodLogger.Error("Unknown Event found")
		return clusterEvent
	}

	clusterEvent.Type = spec.UNKNOWN_CHANGE
	return clusterEvent
}

func (p *Processor) getClusterUUID(cluster spec.Kafkacluster) string {
	return cluster.ObjectMeta.Namespace + "-" + cluster.ObjectMeta.Name
}

//Takes in raw Kafka events, lets then detected and the proced to initiate action accoriding to the detected event.
func (p *Processor) processEvent(currentEvent spec.KafkaclusterEvent) {
	methodLogger := log.WithFields(log.Fields{
		"method":                "processEvent",
		"clusterName":           currentEvent.Cluster.ObjectMeta.Name,
		"KafkaClusterEventType": currentEvent.Type,
	})
	methodLogger.Debug("Recieved Event, processing")
	switch currentEvent.Type {
	case spec.NEW_CLUSTER:
		methodLogger.WithField("event-type", spec.NEW_CLUSTER).Info("New CRD added, creating cluster")
		p.createKafkaCluster(currentEvent.Cluster)

		clustersTotal.Inc()
		clustersCreated.Inc()

		methodLogger.Info("Init heartbeat type checking...")
		//TODO rename
		clusterEvent := spec.KafkaclusterEvent{
			Cluster: currentEvent.Cluster,
			Type:    spec.KAKFA_EVENT,
		}
		p.sleep30AndSendEvent(clusterEvent)
		break

	case spec.DELETE_CLUSTER:
		methodLogger.WithField("event-type", spec.DELETE_CLUSTER).Info("Delete Cluster, deleting all Objects ")

		p.deleteKafkaCluster(currentEvent.Cluster)

		go func() {
			time.Sleep(time.Duration(currentEvent.Cluster.Spec.BrokerCount) * time.Minute)
			//TODO dynamic sleep, depending till sts is completely scaled down.
			clusterEvent := spec.KafkaclusterEvent{
				Cluster: currentEvent.Cluster,
				Type:    spec.CLEANUP_EVENT,
			}
			p.clusterEvents <- clusterEvent
		}()
		clustersTotal.Dec()
		clustersDeleted.Inc()
	case spec.CHANGE_IMAGE:
		methodLogger.Info("Change Image Event detected, updating StatefulSet to trigger a new rollout")
		methodLogger.Error("Not Implemented Currently")
		clustersModified.Inc()
	case spec.UPSIZE_CLUSTER:
		methodLogger.Warn("Upsize Cluster, changing StatefulSet with higher Replicas, no Rebalacing")
		if kafka_kube.UpsizeCluster(currentEvent.Cluster, p.kube) != nil {
			internalErrors.Inc()
			p.sleep30AndSendEvent(currentEvent)
			break
		}
		clustersModified.Inc()
	case spec.UNKNOWN_CHANGE:
		methodLogger.Warn("Unknown (or unsupported) change occured, doing nothing. Maybe manually check the cluster")
		clustersModified.Inc()
	case spec.DOWNSIZE_CLUSTER:
		//TODO remove poor mans casting :P
		//TODO support Downsizing Multiple Brokers
		brokerToDelete := currentEvent.Cluster.Spec.BrokerCount - 0
		methodLogger.Info("Downsizing Broker, deleting Data on Broker: ", brokerToDelete)

		//TODO INIT CC Rebalance

		//TODO wait till rebalcing complete
		err := kafka_kube.DownsizeCluster(currentEvent.Cluster, p.kube)
		if err != nil {
			//just re-try delete event
			internalErrors.Inc()
			p.sleep30AndSendEvent(currentEvent)
			break
		}
		clustersModified.Inc()
	case spec.CHANGE_ZOOKEEPER_CONNECT:
		methodLogger.Error("Trying to change zookeeper connect, not supported currently")
		clustersModified.Inc()
	case spec.CLEANUP_EVENT:
		methodLogger.Info("Recieved CleanupEvent, force delete of StatefuleSet.")
		//p.util.CleanupKafkaCluster(currentEvent.Cluster)
		clustersModified.Inc()
	case spec.KAKFA_EVENT:
		methodLogger.Debug("Kafka Event, heartbeat etc..")
		p.sleep30AndSendEvent(currentEvent)

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
func (p *Processor) watchEvents() {

	p.crdController.MonitorKafkaEvents(p.watchEventsChannel, p.control)
	log.Debug("Watching Events")
	go func() {
		for {
			select {
			case currentEvent := <-p.watchEventsChannel:
				classifiedEvent := p.DetectChangeType(currentEvent)
				p.clusterEvents <- classifiedEvent
			case clusterEvent := <-p.clusterEvents:
				p.processEvent(clusterEvent)
			case err := <-p.errors:
				log.WithField("error", err).Error("Recieved Error through error channel")
			case ctl := <-p.control:
				log.WithField("control-event", ctl).Warn("Recieved Something on Control Channel, shutting down")
				return
			}
		}
	}()
}

// CreateKafkaCluster with the following components: Service, Volumes, StatefulSet.
//Maybe move this also into util
func (p *Processor) createKafkaCluster(clusterSpec spec.Kafkacluster) {
	methodLogger := log.WithFields(log.Fields{
		"method":      "CreateKafkaCluster",
		"clusterName": clusterSpec.ObjectMeta.Name,
	})

	err := kafka_kube.CreateCluster(clusterSpec, p.kube)
	if err != nil {
		methodLogger.WithField("error", err).Fatal("Cant create statefulset")
	}

	p.initKafkaClient(clusterSpec)

	err = exporter_kube.DeployOffsetMonitor(clusterSpec, p.kube)
	if err != nil {
		methodLogger.WithField("error", err).Fatal("Cant deploy stats exporter")
	}

	err = cruisecontrol_kube.DeployCruiseControl(clusterSpec, p.kube)
	if err != nil {
		methodLogger.WithField("error", err).Fatal("Cant deploy cruise-control exporter")
	}

}

func (p *Processor) deleteKafkaCluster(clusterSpec spec.Kafkacluster) error {
	client := p.kube
	err := exporter_kube.DeleteOffsetMonitor(clusterSpec, client)
	if err != nil {
		return err
	}
	err = cruisecontrol_kube.DeleteCruiseControl(clusterSpec, client)
	if err != nil {
		//Error while deleting, just resubmit event after wait time.
		return err

	}
	err = kafka_kube.DeleteCluster(clusterSpec, client)
	if err != nil {
		//Error while deleting, just resubmit event after wait time.
		return err
	}
	return nil
}
