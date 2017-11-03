package kube

import (
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1Beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"

	log "github.com/Sirupsen/logrus"
)

func (k *Kubernetes) updateDeployment(deployment *appsv1Beta1.Deployment) error {
	_, err := k.Client.AppsV1beta1().Deployments(deployment.ObjectMeta.Namespace).Update(deployment)
	return err
}

func (k *Kubernetes) createDeployment(deployment *appsv1Beta1.Deployment) error {
	_, err := k.Client.AppsV1beta1().Deployments(deployment.ObjectMeta.Namespace).Create(deployment)
	return err
}

func (k *Kubernetes) deleteDeployment(deployment *appsv1Beta1.Deployment) error {
	var gracePeriod int64
	gracePeriod = 10

	deleteOption := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}
	err := k.Client.AppsV1beta1().Deployments(deployment.ObjectMeta.Namespace).Delete(deployment.ObjectMeta.Name, &deleteOption)
	return err
}

func (k *Kubernetes) deploymentExists(deployment *appsv1Beta1.Deployment) (bool, error) {
	methodLogger := logger.WithFields(log.Fields{
		"method":    "CreateOrUpdateDeployment",
		"name":      deployment.ObjectMeta.Name,
		"namespace": deployment.ObjectMeta.Namespace,
	})
	namespace := deployment.ObjectMeta.Namespace
	depl, err := k.Client.AppsV1beta1().Deployments(namespace).Get(deployment.ObjectMeta.Name, k.DefaultOption)

	if err != nil {
		if errors.IsNotFound(err) {
			methodLogger.Debug("Deployment dosnt exist")
			return false, nil
		} else {
			methodLogger.WithFields(log.Fields{
				"error": err,
			}).Error("Cant get Deployment INFO from API")
			return false, err
		}

	}
	if len(depl.Name) == 0 {
		methodLogger.Debug("Deployment.Name == 0, therefore it dosnt exists")
		return false, nil
	}
	return true, nil
}

// Deploys the given deployment into kubernetes, error is returned if a non recoverable error happens
func (k *Kubernetes) CreateOrUpdateDeployment(deployment *appsv1Beta1.Deployment) error {
	methodLogger := logger.WithFields(log.Fields{
		"method":    "CreateOrUpdateDeployment",
		"name":      deployment.ObjectMeta.Name,
		"namespace": deployment.ObjectMeta.Namespace,
	})

	exists, err := k.deploymentExists(deployment)
	if err != nil {
		methodLogger.WithField("error", err).Error("Error while checking if deployments exists")
		return err
	}
	if exists {
		err = k.createDeployment(deployment)
	} else {
		err = k.updateDeployment(deployment)
	}
	if err != nil {
		methodLogger.WithField("error", err).Error("Error while creating or updating deployment")
	}
	return err
}

func (k *Kubernetes) DeleteDeployment(deployment *appsv1Beta1.Deployment) error {
	methodLogger := logger.WithFields(log.Fields{
		"method":    "DeleteDeployment",
		"name":      deployment.ObjectMeta.Name,
		"namespace": deployment.ObjectMeta.Namespace,
	})
	exists, err := k.deploymentExists(deployment)
	if err != nil {
		methodLogger.WithField("error", err).Error("Error while checking if deployments exists")
		return err
	}
	if exists {
		replicas := int32(0)
		deployment.Spec.Replicas = &replicas
		err = k.updateDeployment(deployment)
		if err != nil {
			methodLogger.WithField("error", err).Warn("Error while scaling deployment down to 0, ignoring since deleting afterwards")
		}

		err = k.deleteDeployment(deployment)
		if err != nil {
			methodLogger.WithField("error", err).Error("Can delete deployment")
			return err
		}
	} else {
		methodLogger.Debug("Trying to delete but Deployment dosnt exist.")
		return nil

	}
	return nil
}
