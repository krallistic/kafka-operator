package kube

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	log "github.com/Sirupsen/logrus"
)

var (
	logger = log.WithFields(log.Fields{
		"package": "kube",
	})
)

type Kubernetes struct {
	Client        *k8sclient.Clientset
	MasterHost    string
	DefaultOption metav1.GetOptions
	DeleteOption  metav1.DeleteOptions
}

func New(kubeConfigFile, masterHost string) (*Kubernetes, error) {
	methodLogger := logger.WithFields(log.Fields{"method": "New"})

	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	client, err := NewKubeClient(kubeConfigFile)
	if err != nil {
		methodLogger.WithFields(log.Fields{
			"error":  err,
			"config": kubeConfigFile,
			"client": client,
		}).Error("could not init Kubernetes client")
		return nil, err
	}

	k := &Kubernetes{
		Client:     client,
		MasterHost: masterHost,
	}
	methodLogger.WithFields(log.Fields{
		"config": kubeConfigFile,
		"client": client,
	}).Debug("Initilized kubernetes cLient")

	return k, nil
}

func BuildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

//TODO refactor for config *rest.Config :)
func NewKubeClient(kubeCfgFile string) (*k8sclient.Clientset, error) {

	config, err := BuildConfig(kubeCfgFile)
	if err != nil {
		return nil, err
	}

	//TODO refactor & log errors
	return k8sclient.NewForConfig(config)
}
