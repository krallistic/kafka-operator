package processor

import (
	k8sclient "k8s.io/client-go/kubernetes"
	spec "github.com/krallistic/kafka-operator/spec"
	"fmt"
	"github.com/krallistic/kafka-operator/util"
)

type Processor struct {
	client k8sclient.Clientset
	baseBrokerImage string
	util util.ClientUtil
	kafkaClusters map[string]*spec.KafkaCluster
}

func New(client k8sclient.Clientset, image string, util util.ClientUtil) (*Processor, error) {
	p := &Processor{
		client:client,
		baseBrokerImage:image,
		util:util,
		kafkaClusters:make(map[string]*spec.KafkaCluster),
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
func (p *Processor) DetectChangeType(event spec.KafkaClusterWatchEvent) spec.KafkaEventType {
	//TODO multiple changes in one Update? right now we only detect one change
	if event.Type == "ADDED" {
		return spec.NEW_CLUSTER
	}
	if event.Type == "DELETED" {
		return spec.DELTE_CLUSTER
	//EVENT type must be modfied now
	} else if p.util.BrokerStatefulSetExist(event.Object.Spec){
		return spec.NEW_CLUSTER
	} else if p.util.BrokerStSImageUpdate(event.Object.Spec) {
		return spec.CHANGE_IMAGE
	} else if p.util.BrokerStSUpsize(event.Object.Spec) {
		return spec.UPSIZE_CLUSTER
	} else if p.util.BrokerStSDownsize(event.Object.Spec) {
		fmt.Println("No Downsizing currently supported, TODO without dataloss?")
		return spec.DOWNSIZE_CLUSTER
	}


	//check IfClusterExist -> NEW_CLUSTER
	//check if Image/TAG same -> Change_IMAGE
	//check if BrokerCount same -> Down/Upsize Cluster

	return spec.UNKNOWN_CHANGE
}

//Takes in raw Kafka events, lets then detected and the proced to initiate action accoriding to the detected event.
func (p *Processor) processKafkaEvent(currentEvent spec.KafkaClusterWatchEvent) {
	fmt.Println("Recieved Event, proceeding: ", currentEvent)
	switch p.DetectChangeType(currentEvent) {
	case spec.NEW_CLUSTER:
		fmt.Println("ADDED")
		p.CreateKafkaCluster(currentEvent.Object)
	case spec.DELTE_CLUSTER:
		fmt.Println("Delete Cluster, deleting all Objects: ", currentEvent.Object, currentEvent.Object.Spec)
		//TODO check if spec is aviable on delete event...
		p.util.DeleteKafkaCluster(currentEvent.Object.Spec)
	case spec.CHANGE_IMAGE:
		fmt.Println("Change Image, updating StatefulSet should be enoguh to trigger a new Image Rollout")
		p.util.UpdateBrokerStS(currentEvent.Object.Spec)
	case spec.UPSIZE_CLUSTER:
		fmt.Println("Upsize Cluster, changing StewtefulSet with higher Replicas, no Rebalacing")
		p.util.UpdateBrokerStS(currentEvent.Object.Spec)
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
func ( p *Processor) WatchKafkaEvents(control chan int) {
	rawEventsChannel, errorChannel := p.util.MonitorKafkaEvents()
	fmt.Println("Watching Kafka Events")
	go func() {
		for {

			select {
			case currentEvent := <- rawEventsChannel:
				p.processKafkaEvent(currentEvent)
			case err := <- errorChannel:
				println("Error Channel", err)
			case <-control:
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
