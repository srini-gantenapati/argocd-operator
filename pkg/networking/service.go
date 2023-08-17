package networking

import (
	"context"
	"fmt"

	"github.com/argoproj-labs/argocd-operator/pkg/argoutil"
	"github.com/argoproj-labs/argocd-operator/pkg/mutation"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceRequest objects contain all the required information to produce a service object in return
type ServiceRequest struct {
	Name              string
	InstanceName      string
	InstanceNamespace string
	Component         string
	Labels            map[string]string
	Annotations       map[string]string

	// array of functions to mutate role before returning to requester
	Mutations []mutation.MutateFunc
	Client    interface{}
}

// newService returns a new Service instance for the given ArgoCD.
func newService(name, instanceName, instanceNamespace, component string, labels, annotations map[string]string) *corev1.Service {
	var serviceName string
	if name != "" {
		serviceName = name
	} else {
		serviceName = argoutil.GenerateResourceName(instanceName, component)

	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        serviceName,
			Namespace:   instanceNamespace,
			Labels:      argoutil.MergeMaps(argoutil.LabelsForCluster(instanceName, component), labels),
			Annotations: argoutil.MergeMaps(argoutil.AnnotationsForCluster(instanceName, instanceNamespace), annotations),
		},
	}
}

func CreateService(service *corev1.Service, client ctrlClient.Client) error {
	return client.Create(context.TODO(), service)
}

// UpdateService updates the specified Service using the provided client.
func UpdateService(service *corev1.Service, client ctrlClient.Client) error {
	_, err := GetService(service.Name, service.Namespace, client)
	if err != nil {
		return err
	}

	if err = client.Update(context.TODO(), service); err != nil {
		return err
	}
	return nil
}

func DeleteService(name, namespace string, client ctrlClient.Client) error {
	existingService, err := GetService(name, namespace, client)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		return nil
	}

	if err := client.Delete(context.TODO(), existingService); err != nil {
		return err
	}
	return nil
}

func GetService(name, namespace string, client ctrlClient.Client) (*corev1.Service, error) {
	existingService := &corev1.Service{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, existingService)
	if err != nil {
		return nil, err
	}
	return existingService, nil
}

func ListServices(namespace string, client ctrlClient.Client, listOptions []ctrlClient.ListOption) (*corev1.ServiceList, error) {
	existingServices := &corev1.ServiceList{}
	err := client.List(context.TODO(), existingServices, listOptions...)
	if err != nil {
		return nil, err
	}
	return existingServices, nil
}

func RequestService(request ServiceRequest) (*corev1.Service, error) {
	var (
		mutationErr error
	)
	service := newService(request.Name, request.InstanceName, request.InstanceNamespace, request.Component, request.Labels, request.Annotations)

	if len(request.Mutations) > 0 {
		for _, mutation := range request.Mutations {
			err := mutation(nil, service, request.Client)
			if err != nil {
				mutationErr = err
			}
		}
		if mutationErr != nil {
			return service, fmt.Errorf("RequestService: one or more mutation functions could not be applied: %s", mutationErr)
		}
	}

	return service, nil
}
