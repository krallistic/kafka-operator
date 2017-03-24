package main

import (
	"fmt"
	"os/signal"
	"flag"
	"github.com/krallistic/kafka-operator/util"


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
	flag.StringVar(&kubeConfigFile, "kubeconfig", "", "Location of kubecfg file for access to kubernetes master service; --kube_master_url overrides the URL part of this; if neither this nor --kube_master_url are provided, defaults to service account tokens")
	flag.StringVar(&masterHost, "masterhost", "http://localhost:8080", "Full url to kubernetes api server")
	flag.StringVar(&image, "image", "confluentinc/cp-kafka:latest", "Image to use for Brokers")
	//flag.StringVar(&zookeerConnect, "zookeeperConnect", "zk-0.zk-headless.default.svc.cluster.local:2181", "Connect String to zK, if no string is give a custom zookeeper ist deployed")
	flag.Parse()
}

func Main() int {
	fmt.Println("Started kafka-operator ")
	if print {
		fmt.Println("Operator Version: ", version)
		return 0
	}
	fmt.Println("masterHost: ", masterHost)
	fmt.Println("kubeConfigFile Location: ", kubeConfigFile)
	k8sclient, err := util.New(kubeConfigFile, masterHost)
	if err != nil {
		fmt.Println("Error initilizing kubernetes client: ", err)
	}
	fmt.Println(k8sclient)

	k8sclient.CreateKubernetesThirdPartyResource()
	controlChannel := make(chan int) //TODO allows more finegranular Object? maybe a Struct?
	
	processor, err := processor.New(*k8sclient.KubernetesClient, image, *k8sclient, controlChannel)
	processor.WatchKafkaEvents()

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, syscall.SIGINT, syscall.SIGKILL)

	runningLoop: for {
		select {
		case sig :=  <- osSignals:
			fmt.Println("Got Signal from OS shutting Down: ", sig)
			break runningLoop

		}
	}
	fmt.Println("Exiting now")
	//TODO Eventually cleanup?

	return 0
}



func main() {
	os.Exit(Main())
}
