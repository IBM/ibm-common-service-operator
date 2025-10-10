package bootstrap

import (
	"context"
	"testing"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/constant"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/v4/api/v1alpha1"
)

func buildTestBootstrap(t *testing.T) *Bootstrap {
	t.Helper()

	testScheme := runtime.NewScheme()
	_ = corev1.AddToScheme(testScheme)
	_ = odlm.AddToScheme(testScheme)
	_ = apiv3.AddToScheme(testScheme)

	client := fake.NewClientBuilder().WithScheme(testScheme).Build()

	return &Bootstrap{
		Client: client,
		Reader: client,
		CSData: apiv3.CSData{
			ServicesNs:              "ibm-common-services",
			CPFSNs:                  "ibm-common-services",
			CatalogSourceName:       "ibm-odlm-catalog",
			CatalogSourceNs:         "openshift-marketplace",
			ApprovalMode:            string(olmv1alpha1.ApprovalAutomatic),
			Version:                 "4.14.1",
			ExcludedCatalog:         constant.ExcludedCatalog,
			StatusMonitoredServices: constant.StatusMonitoredServices,
		},
	}
}

func TestInstallOrUpdateOpregCreatesResource(t *testing.T) {
	t.Parallel()

	bs := buildTestBootstrap(t)
	ctx := context.Background()

	if err := bs.InstallOrUpdateOpreg(ctx, ""); err != nil {
		t.Fatalf("InstallOrUpdateOpreg returned error: %v", err)
	}

	opreg := &odlm.OperandRegistry{}
	if err := bs.Client.Get(ctx, types.NamespacedName{Name: constant.MasterCR, Namespace: bs.CSData.ServicesNs}, opreg); err != nil {
		t.Fatalf("expected OperandRegistry to be created: %v", err)
	}

	if len(opreg.Spec.Operators) == 0 {
		t.Fatalf("expected operators to be populated in OperandRegistry")
	}
}

func TestReconcileOperandRegistryUpdatesSpec(t *testing.T) {
	t.Parallel()

	bs := buildTestBootstrap(t)
	ctx := context.Background()

	if err := bs.InstallOrUpdateOpreg(ctx, ""); err != nil {
		t.Fatalf("InstallOrUpdateOpreg returned error: %v", err)
	}

	opreg := &odlm.OperandRegistry{}
	if err := bs.Client.Get(ctx, types.NamespacedName{Name: constant.MasterCR, Namespace: bs.CSData.ServicesNs}, opreg); err != nil {
		t.Fatalf("expected OperandRegistry to exist: %v", err)
	}

	for i := range opreg.Spec.Operators {
		opreg.Spec.Operators[i].InstallPlanApproval = olmv1alpha1.ApprovalManual
	}

	if err := bs.ReconcileOperandRegistry(ctx, opreg); err != nil {
		t.Fatalf("ReconcileOperandRegistry returned error: %v", err)
	}

	refreshed := &odlm.OperandRegistry{}
	if err := bs.Client.Get(ctx, types.NamespacedName{Name: constant.MasterCR, Namespace: bs.CSData.ServicesNs}, refreshed); err != nil {
		t.Fatalf("failed to read OperandRegistry: %v", err)
	}

	for _, operator := range refreshed.Spec.Operators {
		if operator.InstallPlanApproval != olmv1alpha1.ApprovalManual {
			t.Fatalf("expected InstallPlanApproval to be manual, got %s", operator.InstallPlanApproval)
		}
	}
}

func TestInstallOrUpdateOpregWithUserManagedOverrides(t *testing.T) {
	t.Parallel()

	bs := buildTestBootstrap(t)
	ctx := context.Background()

	overrides := []apiv3.OperatorConfig{{Name: "common-service-postgresql", UserManaged: true}}

	if err := bs.InstallOrUpdateOpreg(ctx, "", WithUserManagedOverridesFromConfigs(overrides)); err != nil {
		t.Fatalf("InstallOrUpdateOpreg returned error: %v", err)
	}

	opreg := &odlm.OperandRegistry{}
	if err := bs.Client.Get(ctx, types.NamespacedName{Name: constant.MasterCR, Namespace: bs.CSData.ServicesNs}, opreg); err != nil {
		t.Fatalf("expected OperandRegistry to exist: %v", err)
	}

	found := false
	for _, operator := range opreg.Spec.Operators {
		if operator.Name == "common-service-postgresql" {
			found = true
			if !operator.UserManaged {
				t.Fatalf("expected user managed flag to be true")
			}
			break
		}
	}

	if !found {
		t.Fatalf("expected to find operator common-service-postgresql in registry")
	}

	if err := bs.InstallOrUpdateOpreg(ctx, "", WithUserManagedOverridesFromConfigs(nil)); err != nil {
		t.Fatalf("InstallOrUpdateOpreg returned error: %v", err)
	}

	refreshed := &odlm.OperandRegistry{}
	if err := bs.Client.Get(ctx, types.NamespacedName{Name: constant.MasterCR, Namespace: bs.CSData.ServicesNs}, refreshed); err != nil {
		t.Fatalf("failed to read OperandRegistry: %v", err)
	}

	for _, operator := range refreshed.Spec.Operators {
		if operator.Name == "common-service-postgresql" && operator.UserManaged {
			t.Fatalf("expected user managed flag to be reset to false")
		}
	}
}
