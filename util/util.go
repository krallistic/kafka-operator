package util

import (
	k8sclient "k8s.io/client-go/kubernetes"

	"fmt"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/rest"
	meta_v1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/api/v1"
)

const (
	tprName = "kafka.operator.test"
	namespace = "kafka" //TODO flexible NS

	tprEndpoint = "/apis/extensions/v1beta1/thirdpartyresources"
)

var (
	getEndpoint = fmt.Sprintf("/apis/kafka.operator.test/v1/namespaces/%s/kafkaclusters", namespace)
	watchEndpoint = fmt.Sprintf("/apis/kafka.operator.test/v1/watch/namespaces/%s/kafkaclusters", namespace)
)

type ClientUtil struct {
	KubernetesClient *k8sclient.Clientset
	MasterHost string
	DefaultOption meta_v1.GetOptions
}

func New(kubeConfigFile, masterHost string) (*ClientUtil, error)  {

	client, err := newKubeClient(kubeConfigFile)

	if err != nil {
		fmt.Println("Error, could not Init Kubernetes Client")
		return nil, err
	}

	k := &ClientUtil{
		KubernetesClient: client,
		MasterHost: masterHost,
	}
	fmt.Println("Initilized k8s CLient")
	return k, nil

}


func newKubeClient(kubeCfgFile string) (*k8sclient.Clientset, error) {

	var client *k8sclient.Clientset

	// Should we use in cluster or out of cluster config
	if len(kubeCfgFile) == 0 {
		fmt.Println("Using InCluster k8s config")
		cfg, err := rest.InClusterConfig()

		if err != nil {
			return nil, err
		}

		client, err = k8sclient.NewForConfig(cfg)

		if err != nil {
			return nil, err
		}
	} else {
		fmt.Println("Using OutOfCluster k8s config with kubeConfigFile: %s", kubeCfgFile)
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeCfgFile)

		if err != nil {
			fmt.Println("Got error trying to create client: ", err)
			return nil, err
		}

		client, err = k8sclient.NewForConfig(cfg)

		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

func (c *ClientUtil)CreateKubernetesThirdPartyResource() error  {
	tprResult, _ := c.KubernetesClient.ThirdPartyResources().Get("kafkaCluster", c.DefaultOption)
	if len(tprResult.Name) == 0 {
		fmt.Println("No KafkaCluster TPR found, creating...")

		tpr := &v1beta1.ThirdPartyResource{
			ObjectMeta: v1.ObjectMeta{
				Name: tprName,
				Namespace: namespace,
			},
			Versions: []v1beta1.APIVersion{
				{Name: "v1"},
			},
			Description: "Managed apache kafke clusters",
		}
		fmt.Println("Creating TPR: ", tpr)
		retVal, err := c.KubernetesClient.ThirdPartyResources().Create(tpr)
		fmt.Println("retVal: ", retVal)
		if err != nil {
			fmt.Println("Error creating TPR: ", err)
		}

		//TODO Error checking


	} else {
		fmt.Println("TPR already exist")
		//TODO check for correctnes/verison?
	}
	return nil
}