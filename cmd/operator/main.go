package main

import (
	"fmt"
	"os/signal"
	"flag"
	"github.com/krallistic/kafka-operator/util"

	meta_v1 "k8s.io/client-go/pkg/apis/meta/v1"
	"os"
	"syscall"
	"github.com/krallistic/kafka-operator/processor"
)

var (
	version = "0.0.1"
	kubeConfigFile string
	print bool
	masterHost string
	image string
	zookeerConnect string


)


func init() {
	flag.BoolVar(&print, "print", false, "Show basic information and quit - debug")
	flag.StringVar(&kubeConfigFile, "kubeconfig", "/Users/jakobkaralus/.kube/config", "Location of kubecfg file for access to kubernetes master service; --kube_master_url overrides the URL part of this; if neither this nor --kube_master_url are provided, defaults to service account tokens")
	flag.StringVar(&masterHost, "masterhost", "http://127.0.0.1:8080", "Full url to kubernetes api server")
	flag.StringVar(&image, "image", "confluentinc/cp-kafka:latest", "Image to use for Brokers")
	flag.StringVar(&zookeerConnect, "zookeeperConnect", "TODODefaultzKValue?", "Connect String to zK, if no string is give a custom zookeeper ist deployed")

	flag.Parse()
}

func Main() int {
	fmt.Println("Started kafka-operator ")
	if print {
		fmt.Println("Operator Version: ", version)
	}
	fmt.Println("masterHost: ", masterHost)
	fmt.Println("kubeConfigFile Location: ", kubeConfigFile)
	k8sclient, err := util.New(kubeConfigFile, masterHost)
	if err != nil {
		fmt.Println("Error Initilizing kubernetes client: ", err)
	}
	fmt.Println(k8sclient)
	fmt.Println(k8sclient.KubernetesClient.ThirdPartyResources().Get("kafkaCluster", meta_v1.GetOptions{}))
	k8sclient.CreateKubernetesThirdPartyResource()
	fmt.Println(k8sclient.KubernetesClient.ThirdPartyResources().Get("kafkaCluster.operator.com", meta_v1.GetOptions{}))
	fmt.Println(k8sclient.KubernetesClient.ThirdPartyResources().Get("kafka-cluster.operator.com", meta_v1.GetOptions{}))

	fmt.Println(k8sclient.KubernetesClient.ThirdPartyResources().Get("ElasticsearchCluster", meta_v1.GetOptions{}))
	controlChannel := make(chan int) //TODO allows more finegranular Object? maybe a Struct?


	proccessor, err := processor.New(*k8sclient.KubernetesClient, image, *k8sclient)
	proccessor.WatchKafkaEvents(controlChannel)







	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, syscall.SIGINT, syscall.SIGKILL)

	runningLoop: for {
		select {
		case sig :=  <- osSignals:
			fmt.Println("Got Signal from OS shutting Down: ", sig)
			break runningLoop

		}
	}
	fmt.Println("Existing now")
	//TODO Eventually cleanup?


	return 0


}



func main() {
	os.Exit(Main())
}
