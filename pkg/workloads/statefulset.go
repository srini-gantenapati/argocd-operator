package workloads

import (
	"context"
	"fmt"

	"github.com/argoproj-labs/argocd-operator/pkg/argoutil"
	"github.com/argoproj-labs/argocd-operator/pkg/mutation"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// StatefulRequest objects contain all the required information to produce a stateful object in return
type StatefulSetRequest struct {
	Name         string
	InstanceName string
	Namespace    string
	Component    string
	Labels       map[string]string

	// array of functions to mutate role before returning to requester
	Mutations []mutation.MutateFunc
	Client    interface{}
}

// newStateful returns a new Stateful instance for the given ArgoCD.
func newStatefulSet(name, instanceName, namespace, component string, labels map[string]string) *appsv1.StatefulSet {
	var StatefulSetName string
	if name != "" {
		StatefulSetName = name
	} else {
		StatefulSetName = argoutil.GenerateResourceName(instanceName, component)

	}
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        StatefulSetName,
			Namespace:   namespace,
			Labels:      argoutil.MergeMaps(argoutil.LabelsForCluster(instanceName, component), labels),
		},
	}
}

func CreateStatefulSet(StatefulSet *appsv1.StatefulSet, client ctrlClient.Client) error {
	return client.Create(context.TODO(), StatefulSet)
}

// UpdateStatefulSet updates the specified StatefulSet using the provided client.
func UpdateStatefulSet(StatefulSet *appsv1.StatefulSet, client ctrlClient.Client) error {
	_, err := GetStatefulSet(StatefulSet.Name, StatefulSet.Namespace, client)
	if err != nil {
		return err
	}

	if err = client.Update(context.TODO(), StatefulSet); err != nil {
		return err
	}
	return nil
}

func DeleteStatefulSet(name, namespace string, client ctrlClient.Client) error {
	existingStatefulSet, err := GetStatefulSet(name, namespace, client)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		return nil
	}

	if err := client.Delete(context.TODO(), existingStatefulSet); err != nil {
		return err
	}
	return nil
}

func GetStatefulSet(name, namespace string, client ctrlClient.Client) (*appsv1.StatefulSet, error) {
	existingStatefulSet := &appsv1.StatefulSet{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, existingStatefulSet)
	if err != nil {
		return nil, err
	}
	return existingStatefulSet, nil
}

func ListStatefulSets(namespace string, client ctrlClient.Client, listOptions []ctrlClient.ListOption) (*appsv1.StatefulSetList, error) {
	existingStatefulSets := &appsv1.StatefulSetList{}
	err := client.List(context.TODO(), existingStatefulSets, listOptions...)
	if err != nil {
		return nil, err
	}
	return existingStatefulSets, nil
}

func RequestStatefulSet(request StatefulSetRequest) (*appsv1.StatefulSet, error) {
	var (
		mutationErr error
	)
	StatefulSet := newStatefulSet(request.Name, request.InstanceName, request.Namespace, request.Component, request.Labels)

	if len(request.Mutations) > 0 {
		for _, mutation := range request.Mutations {
			err := mutation(nil, StatefulSet, request.Client)
			if err != nil {
				mutationErr = err
			}
		}
		if mutationErr != nil {
			return StatefulSet, fmt.Errorf("RequestStatefulSet: one or more mutation functions could not be applied: %s", mutationErr)
		}
	}

	return StatefulSet, nil
}