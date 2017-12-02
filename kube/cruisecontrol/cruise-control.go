package cruisecontrol

import (
	"strings"

	"github.com/krallistic/kafka-operator/kube"
	"github.com/krallistic/kafka-operator/spec"
	util "github.com/krallistic/kafka-operator/util"

	appsv1Beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	cc_deplyomentPrefix = "cruise-control"
	cc_image            = "krallistic/cruise-control" //TODO
	cc_version          = "latest"                    //TODO make version cmd arg

)

func GetCruiseControlName(cluster spec.Kafkacluster) string {
	return cc_deplyomentPrefix + "-" + cluster.ObjectMeta.Name
}

func generateCruiseControlDeployment(cluster spec.Kafkacluster) *appsv1Beta1.Deployment {
	replicas := int32(1)

	objectMeta := metav1.ObjectMeta{
		Name: GetCruiseControlName(cluster),
		Labels: map[string]string{
			"component": "kafka",
			"name":      cluster.ObjectMeta.Name,
			"role":      "data",
			"type":      "cruise-control",
		},
	}

	podObjectMeta := metav1.ObjectMeta{
		Name: GetCruiseControlName(cluster),
		Labels: map[string]string{
			"component": "kafka",
			"name":      cluster.ObjectMeta.Name,
			"role":      "data",
			"type":      "cruise-control",
		},
	}
	brokerList := strings.Join(util.GetBrokerAdressess(cluster), ",")

	deploy := &appsv1Beta1.Deployment{
		ObjectMeta: objectMeta,
		Spec: appsv1Beta1.DeploymentSpec{
			Replicas: &replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: podObjectMeta,
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Name:    "cruise-control",
							Image:   "krallistic/cruise-control:latest",
							Command: []string{"/bin/sh", "./setup-cruise-control.sh", "config/cruisecontrol.properties", "9095"},
							Env: []v1.EnvVar{
								v1.EnvVar{
									Name:  "ZOOKEEPER_CONNECT",
									Value: cluster.Spec.ZookeeperConnect,
								},
								v1.EnvVar{
									Name:  "BOOTSTRAP_BROKER",
									Value: brokerList,
								},
							},
							Ports: []v1.ContainerPort{
								v1.ContainerPort{
									Name:          "rest",
									ContainerPort: 9095,
								},
							},
						},
					},
				},
			},
		},
	}

	return deploy
}

func generateCruiseControlService(cluster spec.Kafkacluster) *v1.Service {
	obejctMeta := metav1.ObjectMeta{
		Name: GetCruiseControlName(cluster),
		Labels: map[string]string{
			"component": "kafka",
			"name":      cluster.ObjectMeta.Name,
			"role":      "data",
			"type":      "cruise-control",
		},
	}

	svc := &v1.Service{
		ObjectMeta: obejctMeta,
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"component": "kafka",
				"name":      cluster.ObjectMeta.Name,
				"role":      "data",
				"type":      "cruise-control",
			},
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name: "rest",
					Port: 9095,
				},
			},
		},
	}

	return svc
}

// Deploys the OffsetMonitor as an extra Pod inside the Cluster
func DeployCruiseControl(cluster spec.Kafkacluster, client kube.Kubernetes) error {
	deployment := generateCruiseControlDeployment(cluster)
	svc := generateCruiseControlService(cluster)

	err := client.CreateOrUpdateDeployment(deployment)
	if err != nil {
		return err
	}
	err = client.CreateOrUpdateService(svc)
	if err != nil {
		return err
	}
	return nil
}

func DeleteCruiseControl(cluster spec.Kafkacluster, client kube.Kubernetes) error {
	deployment := generateCruiseControlDeployment(cluster)
	svc := generateCruiseControlService(cluster)

	err := client.DeleteDeployment(deployment)
	if err != nil {
		return err
	}
	err = client.DeleteService(svc)
	if err != nil {
		return err
	}

	return nil
}
