package main

import (
	"fmt"
	"os/signal"
	"flag"
	"github.com/krallistic/kafka-operator/util"

	"os"
	"syscall"
	"github.com/krallistic/kafka-operator/processor"

	"net/http"
	log "github.com/Sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version = "0.0.1"
	kubeConfigFile string
	print bool
	masterHost string
	image string
	zookeeperConnect string


	metricListenAddress string
	metricListenPath string
	logger = log.WithFields(log.Fields{
		"package": "main",
	})
)


func init() {
	flag.BoolVar(&print, "print", false, "Show basic information and quit - debug")
	flag.StringVar(&kubeConfigFile, "kubeconfig", "", "Location of kubecfg file for access to kubernetes master service; --kube_master_url overrides the URL part of this; if neither this nor --kube_master_url are provided, defaults to service account tokens")
	flag.StringVar(&masterHost, "masterhost", "http://localhost:8080", "Full url to kubernetes api server")
	flag.StringVar(&image, "image", "confluentinc/cp-kafka:latest", "Image to use for Brokers")
	//flag.StringVar(&zookeerConnect, "zookeeperConnect", "zk-0.zk-headless.default.svc.cluster.local:2181", "Connect String to zK, if no string is give a custom zookeeper ist deployed")
	flag.Parse()

	flag.StringVar(&metricListenAddress, "listen-address", ":9090", "The address to listen on for HTTP requests.")
	flag.StringVar(&metricListenPath, "metric-path", "/metrics", "Path under which the the prometheus metrics can be found")

}

func Main() int {
	logger.WithFields(log.Fields{
		"version": version,
		"masterHost": masterHost,
		"kubeconfig": kubeConfigFile,
		"image": image,
		"metric-listen-address": metricListenAddress,
		"metric-listen-path": metricListenPath,
	}).Info("Started kafka-operator with args")
	if print {
		logger.WithFields(log.Fields{"version": version,}).Print("Operator Version")
		return 0
	}
	k8sclient, err := util.New(kubeConfigFile, masterHost)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
			"configFile": kubeConfigFile,
			"masterHost": masterHost,
		}).Fatal("Error initilizing kubernetes client ")
		return 1
	}

	fmt.Println(k8sclient)

	k8sclient.CreateKubernetesThirdPartyResource()
	controlChannel := make(chan int) //TODO allows more finegranular Object? maybe a Struct?
	
	processor, err := processor.New(*k8sclient.KubernetesClient, image, *k8sclient, controlChannel)
	processor.WatchKafkaEvents()

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, syscall.SIGINT, syscall.SIGKILL)

	http.Handle(metricListenPath, promhttp.Handler())
	logger.Fatal(http.ListenAndServe(metricListenAddress, nil))

	runningLoop: for {
		select {
		case sig :=  <- osSignals:
			logger.WithFields(log.Fields{"signal": sig}).Info("Got Signal from OS shutting Down: ")
			break runningLoop
		}
	}

	logger.Info("Exiting now")
	//TODO Eventually cleanup?

	return 0
}



func main() {
	os.Exit(Main())
}
