package processor

import (
	k8sclient "k8s.io/client-go/kubernetes"
	spec "github.com/krallistic/kafka-operator/spec"
	"fmt"
	"github.com/krallistic/kafka-operator/util"
	"time"
)

type Processor struct {
	client k8sclient.Clientset
	baseBrokerImage string
	util util.ClientUtil
	kafkaClusters map[string]*spec.KafkaCluster
	watchEvents chan spec.KafkaClusterWatchEvent
	clusterEvents chan spec.KafkaClusterEvent
	control chan int
	errors chan error
}

func New(client k8sclient.Clientset, image string, util util.ClientUtil, control chan int) (*Processor, error) {
	p := &Processor{
		client:client,
		baseBrokerImage:image,
		util:util,
		kafkaClusters:make(map[string]*spec.KafkaCluster),
		watchEvents: make(chan spec.KafkaClusterWatchEvent),
		clusterEvents: make(chan spec.KafkaClusterEvent),
		control: control,
		errors: make(chan error),
	}
	fmt.Println("Created Processor")
	return p, nil
}

func ( p *Processor) Run() error {
	//TODO getListOfAlredyRunningCluster/Refresh
	fmt.Println("Running Processor")
	return nil
}


//We detect basic change through the event type, beyond that we use the API server to find differences.
//Functions compares the KafkaClusterSpec with the real Pods/Services which are there.
//We do that because otherwise we would have to use a local state to track changes.
func (p *Processor) DetectChangeType(event spec.KafkaClusterWatchEvent) spec.KafkaClusterEvent {
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
	} else if p.util.BrokerStatefulSetExist(event.Object.Spec){
		clusterEvent.Type = spec.NEW_CLUSTER
		return clusterEvent
	} else if p.util.BrokerStSImageUpdate(event.Object.Spec) {
		clusterEvent.Type = spec.CHANGE_IMAGE
		return clusterEvent
	} else if p.util.BrokerStSUpsize(event.Object.Spec) {
		clusterEvent.Type = spec.UPSIZE_CLUSTER
		return clusterEvent
	} else if p.util.BrokerStSDownsize(event.Object.Spec) {
		fmt.Println("No Downsizing currently supported, TODO without dataloss?")
		clusterEvent.Type = spec.DOWNSIZE_CLUSTER
		return clusterEvent
	}

	//check IfClusterExist -> NEW_CLUSTER
	//check if Image/TAG same -> Change_IMAGE
	//check if BrokerCount same -> Down/Upsize Cluster

	clusterEvent.Type = spec.UNKNOWN_CHANGE
	return clusterEvent
}

//Takes in raw Kafka events, lets then detected and the proced to initiate action accoriding to the detected event.
func (p *Processor) processKafkaEvent(currentEvent spec.KafkaClusterEvent) {
	fmt.Println("Recieved Event, proceeding: ", currentEvent)
	switch currentEvent.Type  {
	case spec.NEW_CLUSTER:
		fmt.Println("ADDED")
		p.CreateKafkaCluster(currentEvent.Cluster)
	case spec.DELTE_CLUSTER:
		fmt.Println("Delete Cluster, deleting all Objects: ", currentEvent.Cluster, currentEvent.Cluster.Spec)
		p.util.DeleteKafkaCluster(currentEvent.Cluster.Spec)
		go func() {
			time.Sleep(5 * time.Minute)
			//TODO dynamic sleep, depending till sts is completely scaled down.
			clusterEvent := spec.KafkaClusterEvent{
				Cluster: currentEvent.Cluster,
				Type: spec.CLEANUP_EVENT,
			}
			p.clusterEvents <- clusterEvent
		}()
	case spec.CHANGE_IMAGE:
		fmt.Println("Change Image, updating StatefulSet should be enoguh to trigger a new Image Rollout")
		p.util.UpdateBrokerStS(currentEvent.Cluster.Spec)
	case spec.UPSIZE_CLUSTER:
		fmt.Println("Upsize Cluster, changing StewtefulSet with higher Replicas, no Rebalacing")
		p.util.UpdateBrokerStS(currentEvent.Cluster.Spec)
	case spec.UNKNOWN_CHANGE:
		fmt.Println("Unkown (or unsupported) change occured, doing nothing. Maybe manually check the cluster")
	case spec.DOWNSIZE_CLUSTER:
		fmt.Println("Downsize Cluster")
	case spec.CHANGE_ZOOKEEPER_CONNECT:
		fmt.Println("Trying to change zookeeper connect, not supported currently")
	}
}


//Creates inside a goroutine a watch channel on the KakkaCLuster Endpoint and distibutes the events.
//control chan used for showdown events from outside
func ( p *Processor) WatchKafkaEvents() {
	p.util.MonitorKafkaEvents(p.watchEvents, p.errors)
	fmt.Println("Watching Kafka Events")
	go func() {
		for {

			select {
			case currentEvent := <- p.watchEvents:
				classifiedEvent := p.DetectChangeType(currentEvent)
				p.clusterEvents <- classifiedEvent
			case clusterEvent := <- p.clusterEvents:
				p.processKafkaEvent(clusterEvent)
			case err := <- p.errors:
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

	headless_SVC_Name := clusterSpec.Spec.Name
	round_robing_dns := headless_SVC_Name + suffix
	fmt.Println("Headless Service Name: ", headless_SVC_Name, " Should be accessable through LB: ", round_robing_dns )

	var i int32
	for  i = 0; i < clusterSpec.Spec.BrokerCount; i++ {
		brokerNames[i] = "kafka-0." + headless_SVC_Name + suffix
		fmt.Println("Broker", i , " ServiceName: ", brokerNames[i])
	}

	//Create Headless Brokersvc
	//TODO better naming
	p.util.CreateBrokerService(clusterSpec.Spec, true)

	//TODO createVolumes

	//CREATE Broker sts
	//Currently we extract name out of spec, maybe move to metadata to be more inline with other k8s komponents.
	p.util.CreateBrokerStatefulSet(clusterSpec.Spec)

}
