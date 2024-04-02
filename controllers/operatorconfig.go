package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	v3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

func (r *CommonServiceReconciler) updateOperatorConfig(ctx context.Context, configList []v3.OperatorConfig) (bool, error) {
	klog.Info("Applying OperatorConfig")

	if configList == nil {
		return true, nil
	}

	// TODO: remove when this feature is generalized to all other operators
	for _, c := range configList {
		config := c
		packageName, err := r.fetchPackageNameFromOpReg(ctx, config.Name)
		if err != nil {
			return false, err
		}
		if packageName != "cloud-native-postgresql" {
			return false, errors.New("failed to update OperatorConfig. This feature is only available for cloud-native-postgresql operator")
		}
		if config.Replicas == nil {
			return true, nil
		}
	}

	operatorConfig := &odlm.OperatorConfig{}
	if err := r.Reader.Get(ctx, types.NamespacedName{
		Name:      "test-operator-config",
		Namespace: r.Bootstrap.CSData.ServicesNs,
	}, operatorConfig); err != nil {
		if !apierrors.IsNotFound(err) {
			klog.Errorf("failed to get OperandConfig %s/%s: %v", operatorConfig.GetNamespace(), operatorConfig.GetName(), err)
			return true, err
		}
	}
	replicas := *configList[0].Replicas
	replacer := strings.NewReplacer("placeholder-size", fmt.Sprintf("%d", replicas))
	updatedConfig := replacer.Replace(constant.PostGresOperatorConfig)
	klog.V(2).Infof("OperatorConfig to be applied will be: %v", updatedConfig)

	if err := r.Bootstrap.InstallOrUpdateOperatorConfig(updatedConfig, true); err != nil {
		return false, err
	}
	return false, nil
}

func (r *CommonServiceReconciler) fetchPackageNameFromOpReg(ctx context.Context, name string) (string, error) {
	registry, err := r.GetOperandRegistry(ctx, "common-service", r.CSData.ServicesNs)
	if err != nil {
		return "", err
	}

	for _, r := range registry.Spec.Operators {
		operator := r
		if operator.Name == name {
			return operator.PackageName, nil
		}
	}
	return "", nil
}
