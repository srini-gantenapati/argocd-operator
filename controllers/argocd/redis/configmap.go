package redis

import (
	"github.com/argoproj-labs/argocd-operator/common"
	"github.com/argoproj-labs/argocd-operator/controllers/argocd/argocdcommon"
	"github.com/argoproj-labs/argocd-operator/pkg/argoutil"
	"github.com/argoproj-labs/argocd-operator/pkg/workloads"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	haproxyCfgKey             = "haproxy.cfg"
	haproxyScriptKey          = "haproxy_init.sh"
	initScriptKey             = "init.sh"
	redisConfKey              = "redis.conf"
	sentinelConfKey           = "sentinel.Conf"
	livenessScriptKey         = "redis_liveness.sh"
	readinessScriptKey        = "redis_readiness.sh"
	sentinelLivenessScriptKey = "sentinel_liveness.sh"
)

// reconcileHAConfigMap will ensure that the Redis HA ConfigMap is present for the given ArgoCD instance
func (rr *RedisReconciler) reconcileHAConfigMap() error {
	cmRequest := workloads.ConfigMapRequest{
		ObjectMeta: argoutil.GetObjMeta(common.ArgoCDRedisHAConfigMapName, rr.Instance.Namespace, rr.Instance.Name, rr.Instance.Namespace, component),
		Data: map[string]string{
			haproxyCfgKey:    rr.getHAProxyConfig(),
			haproxyScriptKey: rr.getHAProxyScript(),
			initScriptKey:    rr.getInitScript(),
			redisConfKey:     rr.getConf(),
			sentinelConfKey:  rr.getSentinelConf(),
		},
	}

	desired, err := workloads.RequestConfigMap(cmRequest)
	if err != nil {
		rr.Logger.Debug("reconcileHAConfigMap: one or more mutations could not be applied")
		return errors.Wrapf(err, "reconcileHAConfigMap: failed to request configMap %s in namespace %s", desired.Name, desired.Namespace)
	}

	if err = controllerutil.SetControllerReference(rr.Instance, desired, rr.Scheme); err != nil {
		rr.Logger.Error(err, "reconcileHAConfigMap: failed to set owner reference for configMap", "name", desired.Name, "namespace", desired.Namespace)
	}

	existing, err := workloads.GetConfigMap(desired.Name, desired.Namespace, rr.Client)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, "reconcileHAConfigMap: failed to retrieve configMap %s in namespace %s", desired.Name, desired.Namespace)
		}

		if err = workloads.CreateConfigMap(desired, rr.Client); err != nil {
			return errors.Wrapf(err, "reconcileHAConfigMap: failed to create configMap %s in namespace %s", desired.Name, desired.Namespace)
		}
		rr.Logger.Info("config map created", "name", desired.Name, "namespace", desired.Namespace)
		return nil
	}
	changed := false

	fieldsToCompare := []argocdcommon.FieldToCompare{
		{Existing: &existing.Labels, Desired: &desired.Labels, ExtraAction: nil},
		{Existing: &existing.Annotations, Desired: &desired.Annotations, ExtraAction: nil},
		{Existing: &existing.Data, Desired: &desired.Data, ExtraAction: nil},
	}

	argocdcommon.UpdateIfChanged(fieldsToCompare, &changed)

	if !changed {
		return nil
	}

	if err = workloads.UpdateConfigMap(existing, rr.Client); err != nil {
		return errors.Wrapf(err, "reconcileHAConfigMap: failed to update configmap %s", existing.Name)
	}

	rr.Logger.Info("configmap updated", "name", existing.Name, "namespace", existing.Namespace)
	return nil
}

// reconcileHAHealthConfigMap will ensure that the Redis HA Health ConfigMap is present for the given ArgoCD.
func (rr *RedisReconciler) reconcileHAHealthConfigMap() error {
	req := workloads.ConfigMapRequest{
		ObjectMeta: argoutil.GetObjMeta(common.ArgoCDRedisHAHealthConfigMapName, rr.Instance.Namespace, rr.Instance.Name, rr.Instance.Namespace, component),
		Data: map[string]string{
			livenessScriptKey:         rr.getLivenessScript(),
			readinessScriptKey:        rr.getReadinessScript(),
			sentinelLivenessScriptKey: rr.getSentinelLivenessScript(),
		},
	}

	desired, err := workloads.RequestConfigMap(req)
	if err != nil {
		rr.Logger.Debug("reconcileHAHealthConfigMap: one or more mutations could not be applied")
		return errors.Wrapf(err, "reconcileHAHealthConfigMap: failed to request configMap %s", desired.Namespace)
	}

	if err = controllerutil.SetControllerReference(rr.Instance, desired, rr.Scheme); err != nil {
		rr.Logger.Error(err, "reconcileHAHealthConfigMap: failed to set owner reference for configMap", "name", desired.Name, "namespace", desired.Namespace)
	}

	existing, err := workloads.GetConfigMap(desired.Name, desired.Namespace, rr.Client)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, "reconcileHAHealthConfigMap: failed to retrieve configMap %s in namespace %s", desired.Name, desired.Namespace)
		}

		if err = workloads.CreateConfigMap(desired, rr.Client); err != nil {
			return errors.Wrapf(err, "reconcileHAHealthConfigMap: failed to create configMap %s in namespace %s", desired.Name, desired.Namespace)
		}
		rr.Logger.Info("configMap created", "name", desired.Name, "namespace", desired.Namespace)
		return nil
	}

	changed := false

	fieldsToCompare := []argocdcommon.FieldToCompare{
		{Existing: &existing.Labels, Desired: &desired.Labels, ExtraAction: nil},
		{Existing: &existing.Annotations, Desired: &desired.Annotations, ExtraAction: nil},
		{Existing: &existing.Data, Desired: &desired.Data, ExtraAction: nil},
	}

	argocdcommon.UpdateIfChanged(fieldsToCompare, &changed)

	if !changed {
		return nil
	}

	if err = workloads.UpdateConfigMap(existing, rr.Client); err != nil {
		return errors.Wrapf(err, "reconcileHAHealthConfigMap: failed to update configmap %s", existing.Name)
	}

	rr.Logger.Info("configmap updated", "name", existing.Name, "namespace", existing.Namespace)
	return nil
}

func (rr *RedisReconciler) deleteConfigMap(name, namespace string) error {
	if err := workloads.DeleteConfigMap(name, namespace, rr.Client); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "deleteConfigMap: failed to delete config map %s", name)
	}
	rr.Logger.Info("config map deleted", "name", name, "namespace", namespace)
	return nil
}
