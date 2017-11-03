package kube

import (
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1Beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"

	log "github.com/Sirupsen/logrus"
)

func (k *Kubernetes) updateStatefulSet(statefulset *appsv1Beta1.StatefulSet) error {
	_, err := k.Client.AppsV1beta1().StatefulSets(statefulset.ObjectMeta.Namespace).Update(statefulset)

	return err
}

func (k *Kubernetes) deleteStatefulSet(statefulset *appsv1Beta1.StatefulSet) error {
	var gracePeriod int64
	gracePeriod = 10

	deleteOption := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}
	err := k.Client.AppsV1beta1().StatefulSets(statefulset.ObjectMeta.Namespace).Delete(statefulset.ObjectMeta.Name, &deleteOption)

	return err
}
func (k *Kubernetes) createStatefulSet(statefulset *appsv1Beta1.StatefulSet) error {
	_, err := k.Client.AppsV1beta1().StatefulSets(statefulset.ObjectMeta.Namespace).Create(statefulset)
	return err
}

func (k *Kubernetes) statefulsetExists(statefulset *appsv1Beta1.StatefulSet) (bool, error) {
	methodLogger := logger.WithFields(log.Fields{
		"method":    "CreateOrUpdateStatefulSet",
		"name":      statefulset.ObjectMeta.Name,
		"namespace": statefulset.ObjectMeta.Namespace,
	})
	namespace := statefulset.ObjectMeta.Namespace
	sts, err := k.Client.AppsV1beta1().StatefulSets(namespace).Get(statefulset.ObjectMeta.Name, k.DefaultOption)

	if err != nil {
		if errors.IsNotFound(err) {
			methodLogger.Debug("StatefulSet dosnt exist")
			return false, nil
		} else {
			methodLogger.WithFields(log.Fields{
				"error": err,
			}).Error("Cant get StatefulSet INFO from API")
			return false, err
		}

	}
	if len(sts.Name) == 0 {
		methodLogger.Debug("StatefulSet.Name == 0, therefore it dosnt exists")
		return false, nil
	}
	return true, nil
}

// Deploys the given statefulset into kubernetes, error is returned if a non recoverable error happens
func (k *Kubernetes) CreateOrUpdateStatefulSet(statefulset *appsv1Beta1.StatefulSet) error {
	methodLogger := logger.WithFields(log.Fields{
		"method":    "CreateOrUpdateStatefulSet",
		"name":      statefulset.ObjectMeta.Name,
		"namespace": statefulset.ObjectMeta.Namespace,
	})

	exists, err := k.statefulsetExists(statefulset)
	if err != nil {
		methodLogger.WithField("error", err).Error("Error while checking if statefulsets exists")
		return err
	}
	if exists {
		err = k.createStatefulSet(statefulset)
	} else {
		err = k.updateStatefulSet(statefulset)
	}
	if err != nil {
		methodLogger.WithField("error", err).Error("Error while creating or updating statefulset")
	}
	return err
}

func (k *Kubernetes) DeleteStatefulset(statefulset *appsv1Beta1.Statefulset) error {
	methodLogger := logger.WithFields(log.Fields{
		"method":    "DeleteStatefulset",
		"name":      statefulset.ObjectMeta.Name,
		"namespace": statefulset.ObjectMeta.Namespace,
	})
	exists, err := k.statefulsetExists(statefulset)
	if err != nil {
		methodLogger.WithField("error", err).Error("Error while checking if statefulsets exists")
		return err
	}
	if exists {
		replicas := int32(0)
		statefulset.Spec.Replicas = &replicas
		err = k.updateStatefulSet(statefulset)
		if err != nil {
			methodLogger.WithField("error", err).Warn("Error while scaling statefulset down to 0, ignoring since deleting afterwards")
		}

		err = k.deleteStatefulSet(statefulset)
		if err != nil {
			methodLogger.WithField("error", err).Error("Can delete statefulset")
			return err
		}
	} else {
		methodLogger.Debug("Trying to delete but Statefulset dosnt exist.")

	}
	return nil
}
