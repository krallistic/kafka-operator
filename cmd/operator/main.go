package main

import (
	"flag"
	"os/signal"

	"os"
	"syscall"

	"net/http"

	log "github.com/Sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/krallistic/kafka-operator/kube"
	"github.com/krallistic/kafka-operator/processor"
	"github.com/krallistic/kafka-operator/util"

	"github.com/krallistic/kafka-operator/controller"
)

var (
	version          = "0.0.1"
	kubeConfigFile   string
	print            bool
	masterHost       string
	image            string
	zookeeperConnect string

	metricListenAddress string
	metricListenPath    string

	logLevel string

	namespace string

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

	flag.StringVar(&logLevel, "log-level", "debug", "log level, one of debug, info, warn, error")

	flag.StringVar(&metricListenAddress, "listen-address", ":9090", "The address to listen on for HTTP requests.")
	flag.StringVar(&metricListenPath, "metric-path", "/metrics", "Path under which the the prometheus metrics can be found")
	flag.StringVar(&namespace, "namespace", "", "Namespace on which the operator listens to CR, if not set then all Namespaces will be used")
	flag.Parse()
}

func Main() int {
	//TODO make cmd-line flag
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.WithField("error", err).Error("Error cant parse log-level, defaulting to info")
		level = log.InfoLevel
	}
	log.SetLevel(level)

	logger.WithFields(log.Fields{
		"version":               version,
		"masterHost":            masterHost,
		"kubeconfig":            kubeConfigFile,
		"image":                 image,
		"metric-listen-address": metricListenAddress,
		"metric-listen-path":    metricListenPath,
	}).Info("Started kafka-operator with args")
	if print {
		logger.WithFields(log.Fields{"version": version}).Print("Operator Version")
		return 0
	}

	//Creating osSignals first so we can exit at any time.
	osSignals := make(chan os.Signal, 2)
	signal.Notify(osSignals, syscall.SIGINT, syscall.SIGKILL, os.Interrupt)

	controlChannel := make(chan int, 2) //TODO allows more finegranular Object? maybe a Struct? Replace with just osSignals?

	go func() {
		for {
			select {
			case sig := <-osSignals:
				logger.WithFields(log.Fields{"signal": sig}).Info("Got Signal from OS shutting Down: ")
				controlChannel <- 1
				//TODO Cleanup
				os.Exit(1)
			}
		}
	}()

	k8sclient, err := util.New(kubeConfigFile, masterHost)
	if err != nil {
		logger.WithFields(log.Fields{
			"error":      err,
			"configFile": kubeConfigFile,
			"masterHost": masterHost,
		}).Fatal("Error initilizing kubernetes client ")
		return 1
	}

	kube, err := kube.New(kubeConfigFile, masterHost)
	if err != nil {
		logger.WithFields(log.Fields{
			"error":      err,
			"configFile": kubeConfigFile,
			"masterHost": masterHost,
		}).Fatal("Error initilizing kubernetes client ")
		return 1
	}

	cdrClient, err := controller.New(kubeConfigFile, masterHost, namespace)
	if err != nil {
		logger.WithFields(log.Fields{
			"error":      err,
			"configFile": kubeConfigFile,
			"masterHost": masterHost,
		}).Fatal("Error initilizing ThirdPartyRessource (KafkaClusters) client ")
		return 1
	}

	cdrClient.CreateCustomResourceDefinition()

	processor, err := processor.New(*k8sclient.KubernetesClient, image, *k8sclient, *cdrClient, controlChannel, *kube)
	processor.Run()

	http.Handle(metricListenPath, promhttp.Handler())
	//Blocking ListenAndServer, so we dont exit
	logger.Fatal(http.ListenAndServe(metricListenAddress, nil))
	logger.Info("Exiting now")

	return 0
}

func main() {
	os.Exit(Main())
}
