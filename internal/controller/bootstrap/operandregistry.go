package bootstrap

import (
	"context"
	"fmt"

	utilyaml "github.com/ghodss/yaml"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/common"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/constant"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/v4/api/v1alpha1"
)

type OperandRegistryOption func(*odlm.OperandRegistry) error

// WithUserManagedOverrides applies user-managed overrides to the rendered OperandRegistry spec.
func WithUserManagedOverrides(overrides map[string]bool, resetAll bool) OperandRegistryOption {
	return func(reg *odlm.OperandRegistry) error {
		if reg == nil {
			return fmt.Errorf("operand registry option received nil registry")
		}

		if resetAll {
			for i := range reg.Spec.Operators {
				reg.Spec.Operators[i].UserManaged = false
			}
		}

		for name, value := range overrides {
			if err := common.UpdateOpRegUserManaged(reg, name, value); err != nil {
				return err
			}
		}

		return nil
	}
}

// WithUserManagedOverridesFromConfigs constructs an OperandRegistryOption that applies
// user-managed overrides derived from the CommonService spec. When no overrides are
// provided, all operators are reset to operator-managed (false).
func WithUserManagedOverridesFromConfigs(configs []apiv3.OperatorConfig) OperandRegistryOption {
	overrides := make(map[string]bool, len(configs))
	resetAll := true
	if len(configs) > 0 {
		resetAll = false
	}

	for _, cfg := range configs {
		overrides[cfg.Name] = cfg.UserManaged
	}

	// Allow passing nil map when resetting all.
	if len(overrides) == 0 {
		overrides = nil
	}

	return WithUserManagedOverrides(overrides, resetAll)
}

// buildOperandRegistry renders the desired OperandRegistry spec based on the current bootstrap data.
func (b *Bootstrap) buildOperandRegistry(ctx context.Context, installPlanApproval olmv1alpha1.Approval, options ...OperandRegistryOption) (*odlm.OperandRegistry, error) {
	configMap := &corev1.ConfigMap{}
	if err := b.Client.Get(ctx, types.NamespacedName{
		Name:      constant.IBMCPPCONFIG,
		Namespace: b.CSData.ServicesNs,
	}, configMap); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		klog.Infof("ConfigMap %s not found in namespace %s, using default values", constant.IBMCPPCONFIG, b.CSData.ServicesNs)
		configMap.Data = make(map[string]string)
	}

	var baseReg string
	registries := []string{
		constant.CSV4OpReg,
		constant.MongoDBOpReg,
		constant.IMOpReg,
		constant.IdpConfigUIOpReg,
		constant.PlatformUIOpReg,
		constant.KeyCloakOpReg,
		constant.CommonServicePGOpReg,
		constant.CommonServiceCNPGOpReg,
	}

	if b.SaasEnable {
		baseReg = constant.CSV3SaasOpReg
	} else {
		baseReg = constant.CSV3OpReg
	}

	concatenatedReg, err := constant.ConcatenateRegistries(baseReg, registries, b.CSData, configMap.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to concatenate OperandRegistry: %w", err)
	}

	desired := &odlm.OperandRegistry{}
	if err := utilyaml.Unmarshal([]byte(concatenatedReg), desired); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OperandRegistry template: %w", err)
	}

	desired.SetGroupVersionKind(odlm.GroupVersion.WithKind(constant.OpregKind))

	// enforce namespace/name and labels from bootstrap data
	desired.Namespace = b.CSData.ServicesNs
	if desired.Name == "" {
		desired.Name = constant.MasterCR
	}
	if desired.Labels == nil {
		desired.Labels = map[string]string{}
	}
	desired.Labels[constant.CsManagedLabel] = "true"

	// honour explicit install plan approval overrides
	approvalMode := installPlanApproval
	if approvalMode == "" {
		approvalMode = olmv1alpha1.Approval(b.CSData.ApprovalMode)
	}
	if approvalMode != "" {
		for i := range desired.Spec.Operators {
			desired.Spec.Operators[i].InstallPlanApproval = approvalMode
		}
	}

	for _, opt := range options {
		if opt == nil {
			continue
		}
		if err := opt(desired); err != nil {
			return nil, err
		}
	}

	return desired, nil
}

// ReconcileOperandRegistry ensures the OperandRegistry in the cluster matches the desired spec.
func (b *Bootstrap) ReconcileOperandRegistry(ctx context.Context, desired *odlm.OperandRegistry) error {
	if desired == nil {
		return fmt.Errorf("desired OperandRegistry must not be nil")
	}

	key := types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}
	current := &odlm.OperandRegistry{}
	err := b.Reader.Get(ctx, key, current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if errors.IsNotFound(err) {
		if desired.Annotations == nil {
			desired.Annotations = map[string]string{}
		}
		desired.Annotations["version"] = b.CSData.Version
		desired.TypeMeta = metav1.TypeMeta{Kind: constant.OpregKind, APIVersion: odlm.GroupVersion.String()}
		klog.Infof("Creating OperandRegistry %s/%s", desired.Namespace, desired.Name)
		return b.Client.Create(ctx, desired.DeepCopy())
	}

	desiredCopy := desired.DeepCopy()
	desiredCopy.TypeMeta = metav1.TypeMeta{Kind: constant.OpregKind, APIVersion: odlm.GroupVersion.String()}
	desiredCopy.ResourceVersion = current.ResourceVersion

	if desiredCopy.Annotations == nil {
		desiredCopy.Annotations = map[string]string{}
	}
	desiredCopy.Annotations["version"] = b.CSData.Version

	if equality.Semantic.DeepEqual(current.Spec, desiredCopy.Spec) &&
		equality.Semantic.DeepEqual(current.GetLabels(), desiredCopy.GetLabels()) &&
		equality.Semantic.DeepEqual(current.GetAnnotations(), desiredCopy.GetAnnotations()) {
		klog.V(2).Infof("OperandRegistry %s/%s already up to date", desired.Namespace, desired.Name)
		return nil
	}

	klog.Infof("Updating OperandRegistry %s/%s", desired.Namespace, desired.Name)
	return b.Client.Update(ctx, desiredCopy)
}
