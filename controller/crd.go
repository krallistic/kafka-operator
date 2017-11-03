package controller

import (
	"fmt"
	"reflect"
	"time"

	"github.com/krallistic/kafka-operator/spec"
	"github.com/krallistic/kafka-operator/util"

	log "github.com/Sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	//"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/pkg/api"

	"k8s.io/client-go/rest"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"k8s.io/client-go/pkg/api/v1"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

var (
	logger = log.WithFields(log.Fields{
		"package": "controller/crd",
	})
)

const (
//tprShortName = "kafka-cluster"
//tprSuffix    = "incubator.test.com"
//tprFullName  = tprShortName + "." + tprSuffix
//API Name is used in the watch of the API, it defined as tprShorName, removal of -, and suffix s
//tprApiName = "kafkaclusters"
//tprVersion = "v1"

)

type CustomResourceController struct {
	ApiExtensionsClient *apiextensionsclient.Clientset
	DefaultOption       metav1.GetOptions
	crdClient           *rest.RESTClient
}

func New(kubeConfigFile, masterHost string) (*CustomResourceController, error) {
	methodLogger := logger.WithFields(log.Fields{"method": "New"})

	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	config, err := util.BuildConfig(kubeConfigFile)

	apiextensionsclientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		methodLogger.WithFields(log.Fields{
			"error":  err,
			"config": config,
			"client": apiextensionsclientset,
		}).Error("could not init Kubernetes client")
		return nil, err
	}

	crdClient, err := newCRDClient(config)
	if err != nil {
		methodLogger.WithFields(log.Fields{
			"Error":  err,
			"Client": crdClient,
			"Config": config,
		}).Error("Could not initialize CustomResourceDefinition Kafkacluster cLient")
		return nil, err
	}

	k := &CustomResourceController{
		crdClient:           crdClient,
		ApiExtensionsClient: apiextensionsclientset,
	}
	methodLogger.Info("Initilized CustomResourceDefinition Kafkacluster cLient")

	return k, nil
}

func (*CustomResourceController) Watch(client *rest.RESTClient, eventsChannel chan spec.KafkaclusterWatchEvent, signalChannel chan int) {
	methodLogger := logger.WithFields(log.Fields{"method": "Watch"})

	stop := make(chan struct{}, 1)
	source := cache.NewListWatchFromClient(
		client,
		spec.CRDRessourcePlural,
		v1.NamespaceAll,
		fields.Everything())

	store, controller := cache.NewInformer(
		source,

		&spec.Kafkacluster{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		0,

		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				cluster := obj.(*spec.Kafkacluster)
				methodLogger.WithFields(log.Fields{"watchFunction": "ADDED"}).Info(spec.PrintCluster(cluster))
				var event spec.KafkaclusterWatchEvent
				//TODO
				event.Type = "ADDED"
				event.Object = *cluster
				eventsChannel <- event
			},

			UpdateFunc: func(old, new interface{}) {
				oldCluster := old.(*spec.Kafkacluster)
				newCluster := new.(*spec.Kafkacluster)
				methodLogger.WithFields(log.Fields{
					"eventType": "UPDATED",
					"old":       spec.PrintCluster(oldCluster),
					"new":       spec.PrintCluster(newCluster),
				}).Debug("Recieved Update Event")
				var event spec.KafkaclusterWatchEvent
				//TODO refactor this. use old/new in EventChannel
				event.Type = "UPDATED"
				event.Object = *newCluster
				event.OldObject = *oldCluster
				eventsChannel <- event
			},

			DeleteFunc: func(obj interface{}) {
				cluster := obj.(*spec.Kafkacluster)
				var event spec.KafkaclusterWatchEvent
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

func (c *CustomResourceController) MonitorKafkaEvents(eventsChannel chan spec.KafkaclusterWatchEvent, signalChannel chan int) {
	methodLogger := logger.WithFields(log.Fields{"method": "MonitorKafkaEvents"})
	methodLogger.Info("Starting Watch")
	c.Watch(c.crdClient, eventsChannel, signalChannel)
}

func configureConfig(cfg *rest.Config) error {
	scheme := runtime.NewScheme()

	if err := spec.AddToScheme(scheme); err != nil {
		return err
	}

	cfg.GroupVersion = &spec.SchemeGroupVersion
	cfg.APIPath = "/apis"
	cfg.ContentType = runtime.ContentTypeJSON
	cfg.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	return nil
}

func newCRDClient(config *rest.Config) (*rest.RESTClient, error) {

	var cdrconfig *rest.Config
	cdrconfig = config
	configureConfig(cdrconfig)

	crdClient, err := rest.RESTClientFor(cdrconfig)
	if err != nil {
		panic(err)
	}

	return crdClient, nil
}

func (c *CustomResourceController) CreateCustomResourceDefinition() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	methodLogger := logger.WithFields(log.Fields{"method": "CreateCustomResourceDefinition"})

	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: spec.CRDFullName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   spec.CRDGroupName,
			Version: spec.CRDVersion,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: spec.CRDRessourcePlural,
				Kind:   reflect.TypeOf(spec.Kafkacluster{}).Name(),
			},
		},
	}
	_, err := c.ApiExtensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		methodLogger.WithFields(log.Fields{
			"error": err,
			"crd":   crd,
		}).Error("Error while creating CRD")
		return nil, err
	}

	// wait for CRD being established
	methodLogger.Debug("Created CRD, wating till its established")
	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = c.ApiExtensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(spec.CRDFullName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					fmt.Printf("Name conflict: %v\n", cond.Reason)
					methodLogger.WithFields(log.Fields{
						"error":  err,
						"crd":    crd,
						"reason": cond.Reason,
					}).Error("Naming Conflict with created CRD")
				}
			}
		}
		return false, err
	})
	if err != nil {
		deleteErr := c.ApiExtensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(spec.CRDFullName, nil)
		if deleteErr != nil {
			return nil, errors.NewAggregate([]error{err, deleteErr})
		}
		return nil, err
	}
	return crd, nil
}

func (c *CustomResourceController) GetKafkaClusters() ([]spec.Kafkacluster, error) {
	methodLogger := logger.WithFields(log.Fields{"method": "GetKafkaClusters"})

	exampleList := spec.KafkaclusterList{}
	err := c.crdClient.Get().Resource(spec.CRDRessourcePlural).Do().Into(&exampleList)

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

func (c *CustomResourceController) SetKafkaclusterState(cluster spec.Kafkacluster) error {
	methodLogger := logger.WithFields(log.Fields{
		"method":    "SetKafkaclusterState",
		"name":      cluster.ObjectMeta.Name,
		"namespace": cluster.ObjectMeta.Namespace,
	})

	methodLogger.Debug("setting state for cluster")

	var result spec.Kafkacluster
	err := c.crdClient.Put(). //TODO check if PATCH is maybe better
					Resource(spec.CRDRessourcePlural).
					Namespace(cluster.ObjectMeta.Namespace).
					Body(cluster).
					Do().Into(&result)

	if err != nil {
		methodLogger.Error("Cant set state on CRD")
	}

	methodLogger.WithField("result", result).Debug("Set state for CRD")

	return err
}
