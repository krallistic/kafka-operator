package controller

import (
	"github.com/krallistic/kafka-operator/spec"
	"github.com/krallistic/kafka-operator/util"

	log "github.com/Sirupsen/logrus"

	"k8s.io/client-go/kubernetes/scheme"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	//"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"

	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

var (
	logger = log.WithFields(log.Fields{
		"package": "controller/tpr",
	})
)

const (
	tprShortName = "kafka-cluster"
	tprSuffix    = "incubator.test.com"
	tprFullName  = tprShortName + "." + tprSuffix
	//API Name is used in the watch of the API, it defined as tprShorName, removal of -, and suffix s
	tprApiName = "kafkaclusters"
	tprVersion = "v1"
)

type ThirdPartyResourceController struct {
	KubernetesClient *k8sclient.Clientset
	DefaultOption    metav1.GetOptions
	tprClient        *rest.RESTClient
}

func New(kubeConfigFile, masterHost string) (*ThirdPartyResourceController, error) {
	methodLogger := logger.WithFields(log.Fields{"method": "New"})

	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	config, err := util.BuildConfig(kubeConfigFile)


	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	client, err := k8sclient.NewForConfig(config)
	if err != nil {
		methodLogger.WithFields(log.Fields{
			"error" : err,
			"config" : config,
			"client" : client,
		}).Error("could not init Kubernetes client")
		return nil, err
	}

	tprClient, err := newTPRClient(config)
	if err != nil {
		methodLogger.WithFields(log.Fields{
			"Error" : err,
			"Client": tprClient,
			"Config": config,
		}).Error("Could not initialize ThirdPartyRessource KafkaCluster cLient")
		return nil, err
	}

	k := &ThirdPartyResourceController{
		tprClient:        tprClient,
		KubernetesClient: client,
	}
	methodLogger.Info("Initilized ThirdPartyRessource KafkaCluster cLient")

	return k, nil
}

func (*ThirdPartyResourceController) Watch(client *rest.RESTClient, eventsChannel chan spec.KafkaClusterWatchEvent, signalChannel chan int) {
	methodLogger := logger.WithFields(log.Fields{"method": "Watch"})

	stop := make(chan struct{}, 1)
	source := cache.NewListWatchFromClient(
		client,
		tprApiName,
		v1.NamespaceAll,
		fields.Everything())

	store, controller := cache.NewInformer(
		source,

		&spec.KafkaCluster{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		0,

		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				cluster := obj.(*spec.KafkaCluster)
				methodLogger.WithFields(log.Fields{"watchFunction": "ADDED"}).Info(spec.PrintCluster(cluster))
				var event spec.KafkaClusterWatchEvent
				//TODO
				event.Type = "ADDED"
				event.Object = *cluster
				eventsChannel <- event
			},

			UpdateFunc: func(old, new interface{}) {
				oldCluster := old.(*spec.KafkaCluster)
				newCluster := new.(*spec.KafkaCluster)
				methodLogger.WithFields(log.Fields{
					"eventType" : "UPDATED",
					"old": spec.PrintCluster(oldCluster),
					"new": spec.PrintCluster(newCluster),
				}).Debug("Recieved Update Event")
				var event spec.KafkaClusterWatchEvent
				//TODO refactor this. use old/new in EventChannel
				event.Type = "UPDATED"
				event.Object = *newCluster
				eventsChannel <- event
			},

			DeleteFunc: func(obj interface{}) {
				cluster := obj.(*spec.KafkaCluster)
				var event spec.KafkaClusterWatchEvent
				event.Type = "DELETED"
				event.Object = *cluster
				eventsChannel <- event
			},
		})

	// the controller run starts the event processing loop
	go controller.Run(stop)
	methodLogger.Info(store)

	go func() {
		select {
		case <-signalChannel:
			methodLogger.Warn("recieved shutdown signal, stopping informer")
			close(stop)
		}
	}()
}

func (c *ThirdPartyResourceController) MonitorKafkaEvents(eventsChannel chan spec.KafkaClusterWatchEvent, signalChannel chan int) {
	methodLogger := logger.WithFields(log.Fields{"method": "MonitorKafkaEvents"})
	methodLogger.Info("Starting Watch")
	c.Watch(c.tprClient, eventsChannel, signalChannel)
}

func configureClient(config *rest.Config) {
	groupversion := schema.GroupVersion{
		Group:   tprSuffix,
		Version: tprVersion,
	}

	config.GroupVersion = &groupversion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	schemeBuilder := runtime.NewSchemeBuilder(
		func(scheme *runtime.Scheme) error {
			scheme.AddKnownTypes(
				groupversion,
				&spec.KafkaCluster{},
				&spec.KafkaClusterList{},
			)
			return nil
		})
	metav1.AddToGroupVersion(scheme.Scheme, groupversion)
	schemeBuilder.AddToScheme(scheme.Scheme)
}

func newTPRClient(config *rest.Config) (*rest.RESTClient, error) {

	var tprconfig *rest.Config
	tprconfig = config
	configureClient(tprconfig)

	tprclient, err := rest.RESTClientFor(tprconfig)
	if err != nil {
		panic(err)
	}

	return tprclient, nil
}

/// Create a the thirdparty ressource inside the Kubernetes Cluster
func (c *ThirdPartyResourceController) CreateKubernetesThirdPartyResource() error {
	methodLogger := logger.WithFields(log.Fields{"method": "CreateKubernetesThirdPartyResource"})
	tpr, err := c.KubernetesClient.ExtensionsV1beta1Client.ThirdPartyResources().Get(tprFullName, c.DefaultOption)
	if err != nil {
		if errors.IsNotFound(err) {
			methodLogger.WithFields(log.Fields{}).Info("No existing KafkaCluster TPR found, creating")

			tpr := &v1beta1.ThirdPartyResource{
				ObjectMeta: metav1.ObjectMeta{
					Name: tprFullName,
				},
				Versions: []v1beta1.APIVersion{
					{Name: tprVersion},
				},
				Description: "Managed Apache Kafka clusters",
			}
			retVal, err := c.KubernetesClient.ThirdPartyResources().Create(tpr)
			if err != nil {
				methodLogger.WithFields(log.Fields{"response": err}).Error("Error creating ThirdPartyRessource KafkaCluster")
				panic(err)
			}
			methodLogger.WithFields(log.Fields{"response": retVal}).Info("Created KafkaCluster TPR")
		}
	} else {
		methodLogger.Info("ThirdPartyRessource KafkaCluster already exist", tpr)
	}
	return nil
}

func (c *ThirdPartyResourceController) GetKafkaClusters() ([]spec.KafkaCluster, error) {
	methodLogger := logger.WithFields(log.Fields{"method": "GetKafkaClusters"})

	exampleList := spec.KafkaClusterList{}
	err := c.tprClient.Get().Resource(tprApiName).Do().Into(&exampleList)

	if err != nil {
		methodLogger.WithFields(log.Fields{
			"response": exampleList,
			"error":    err,
		}).Error("Error response from API")
		return nil, err
	}
	methodLogger.WithFields(log.Fields{
		"response": exampleList,
	}).Info("KafkaCluster received")

	return exampleList.Items, nil
}