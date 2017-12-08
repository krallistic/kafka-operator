package kube

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	log "github.com/Sirupsen/logrus"
)

func (k *Kubernetes) updateService(service *v1.Service) error {

	_, err := k.Client.Core().Services(service.ObjectMeta.Namespace).Update(service)
	return err
}

func (k *Kubernetes) createService(service *v1.Service) error {
	_, err := k.Client.Core().Services(service.ObjectMeta.Namespace).Create(service)

	return err
}

func (k *Kubernetes) deleteService(service *v1.Service) error {
	var gracePeriod int64
	gracePeriod = 10

	deleteOption := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}
	err := k.Client.Core().Services(service.ObjectMeta.Namespace).Delete(service.ObjectMeta.Name, &deleteOption)
	return err
}

func (k *Kubernetes) serviceExists(service *v1.Service) (bool, error) {
	methodLogger := logger.WithFields(log.Fields{
		"method":    "CreateOrUpdateService",
		"name":      service.ObjectMeta.Name,
		"namespace": service.ObjectMeta.Namespace,
	})
	namespace := service.ObjectMeta.Namespace
	svc, err := k.Client.Core().Services(namespace).Get(service.ObjectMeta.Name, k.DefaultOption)

	if err != nil {
		if errors.IsNotFound(err) {
			methodLogger.Debug("Service dosnt exist")
			return false, nil
		} else {
			methodLogger.WithFields(log.Fields{
				"error": err,
			}).Error("Cant get Service INFO from API")
			return false, err
		}

	}
	if len(svc.Name) == 0 {
		methodLogger.Debug("Service.Name == 0, therefore it dosnt exists")
		return false, nil
	}
	return true, nil
}

// Deploys the given service into kubernetes, error is returned if a non recoverable error happens
func (k *Kubernetes) CreateOrUpdateService(service *v1.Service) error {
	methodLogger := logger.WithFields(log.Fields{
		"method":    "CreateOrUpdateService",
		"name":      service.ObjectMeta.Name,
		"namespace": service.ObjectMeta.Namespace,
	})

	exists, err := k.serviceExists(service)
	if err != nil {
		methodLogger.WithField("error", err).Error("Error while checking if services exists")
		return err
	}
	if exists {
		err = k.createService(service)
	} else {
		err = k.updateService(service)
	}
	if err != nil {
		methodLogger.WithField("error", err).Error("Error while creating or updating service")
	}
	return err
}

func (k *Kubernetes) DeleteService(service *v1.Service) error {
	methodLogger := logger.WithFields(log.Fields{
		"method":    "DeleteService",
		"name":      service.ObjectMeta.Name,
		"namespace": service.ObjectMeta.Namespace,
	})
	exists, err := k.serviceExists(service)
	if err != nil {
		methodLogger.WithField("error", err).Error("Error while checking if services exists")
		return err
	}
	if exists {
		err = k.deleteService(service)
		if err != nil {
			methodLogger.WithField("error", err).Error("Can delete service")
			return err
		}
	} else {
		methodLogger.Debug("Trying to delete but Service dosnt exist.")

	}
	return nil
}
