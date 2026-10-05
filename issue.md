## Summary
When Bring Your Own CA (BYO CA) is enabled, `ibm-common-service-operator` skips deploying the `cs-ca-certificate` Certificate CR, but it continues to create and reconcile `cs-ss-issuer` (self-signed issuer). Furthermore, the current auto-detection logic in `IsBYOCert()` can misidentify a re-installation or leftover `cs-ca-certificate-secret` from a prior installation as a BYO certificate scenario, leading to missing CA certificates. In addition, post-deployment label updaters continue to mutate labels on `cs-ca-certificate` even when BYO CA is enabled.

This enhancement addresses these issues:
1. Skip `cs-ss-issuer` creation/reconciliation when BYO CA is active.
2. Improve BYO CA auto-detection to support both documented BYO CA methods without relying on explicit configuration or misinterpreting re-installations.
3. Skip label mutations (`operator.ibm.com/managedByCsOperator`, `manage-cert-rotation`) on `cs-ca-certificate` when BYO CA is active, adhering to the Kubernetes operator principle of not mutating resources governed by users or external systems.

---

## Background & Problem Details

### 1. `cs-ss-issuer` is created unnecessarily when BYO CA is active
- `cs-ss-issuer` exists solely to bootstrap `cs-ca-certificate`.
- Downstream workload certificates (Keycloak, EDB Postgres, Zen, IM, etc.) reference `cs-ca-issuer`, which consumes `cs-ca-certificate-secret`.
- When BYO CA is active, `cs-ca-certificate` is skipped, rendering `cs-ss-issuer` completely unused.
- However, `DeployCertManagerCR` deploys `constant.CertManagerIssuers` (both `cs-ss-issuer` and `cs-ca-issuer`) unconditionally, recreating `cs-ss-issuer` on every reconcile loop even if deleted.

### 2. Auto-detection misinterprets re-installation as BYO CA
- `IsBYOCert()` currently checks if `cs-ca-certificate-secret` exists and `cs-ca-certificate` Certificate CR does not exist.
- If a cluster is uninstalled/reinstalled, or if `cs-ca-certificate` CR is deleted while `cs-ca-certificate-secret` remains:
  - `IsBYOCert()` returns `true` because the secret exists without the certificate CR.
  - The operator incorrectly flags this as BYO CA and skips creating `cs-ca-certificate`.
  - Cert-manager stops managing and rotating the CA certificate.

### 3. Post-deployment label updaters mutate user-managed `cs-ca-certificate`
- In BYO Method 2, where the customer creates and governs their own `cs-ca-certificate` CR with a custom issuer:
  - `ConfigCertManagerOperandManagedByOperator` attempts to add `operator.ibm.com/managedByCsOperator: "true"`.
  - `UpdateManageCertRotationLabel` attempts to enforce `manage-cert-rotation: "true" | "false"`.
- Mutating user-managed/external PKI resources causes GitOps / IaC configuration drift and improperly claims ownership/rotation management over externally governed resources.

---

## Proposed Solution

### 1. Skip `cs-ss-issuer` when `!deployRootCert` (BYO CA active)
- Only deploy `cs-ca-issuer` when BYO CA is enabled.
- Only deploy `cs-ss-issuer` when `deployRootCert == true`.

### 2. Enhanced Auto-Detection Algorithm
Support both documented BYO CA methods ([IBM Documentation](https://www.ibm.com/docs/en/cloud-paks/foundational-services/4.19.x?topic=manager-bringing-your-own-ca-certificate)) and distinguish leftover secrets during re-installation:

- **BYO Method 1 (Direct Secret `cs-ca-certificate-secret`)**:
  - The user manually creates `cs-ca-certificate-secret` (no `cs-ca-certificate` CR exists).
  - The secret does **not** have the cert-manager annotation `cert-manager.io/issuer-name: cs-ss-issuer`.
  - **Action**: Detected as BYO -> Skip `cs-ca-certificate` and `cs-ss-issuer`.

- **BYO Method 2 (Custom Issuer & `cs-ca-certificate` Certificate CR)**:
  - `cs-ca-certificate` CR exists, but `spec.issuerRef.name != "cs-ss-issuer"`.
  - **Action**: Detected as BYO -> Do not overwrite CR or create `cs-ss-issuer`. Also ensure `shouldAddOwnerReference` ignores custom Certificate CRs to avoid hijacking ownership.

- **Re-installation / Leftover Secret**:
  - `cs-ca-certificate` CR is absent, but `cs-ca-certificate-secret` exists **and** contains `cert-manager.io/issuer-name: cs-ss-issuer`.
  - **Action**: Detected as previously operator-managed CA -> Not BYO -> Re-create `cs-ca-certificate` and `cs-ss-issuer`.

### 3. Skip Label Mutations on `cs-ca-certificate` when BYO CA is Active
- In `ConfigCertManagerOperandManagedByOperator`, exclude `cs-ca-certificate` from label updates when BYO CA is active (`!deployRootCert`).
- In `UpdateManageCertRotationLabel`, return early / skip updating `cs-ca-certificate` when BYO CA is active (`!deployRootCert`).

---

## References
- Upstream / Related Issue: https://github.ibm.com/PrivateCloud-analytics/CPD-Quality/issues/152298
- Documentation: [Bringing your own CA certificate](https://www.ibm.com/docs/en/cloud-paks/foundational-services/4.19.x?topic=manager-bringing-your-own-ca-certificate)

---

## Implementation Plan

### Files to Modify

| File | Functions Affected |
|------|-------------------|
| `internal/controller/bootstrap/init.go` | `IsBYOCert()`, `DeployCertManagerCR()`, `ConfigCertManagerOperandManagedByOperator()`, `UpdateManageCertRotationLabel()` |
| `internal/controller/constant/certmanager.go` | (Optional) Split `CertManagerIssuers` into separate constants for finer control |

---

### Change 1: Skip `cs-ss-issuer` in `DeployCertManagerCR()` when BYO CA is active

**Location**: `internal/controller/bootstrap/init.go`, around line 2036

**Current behaviour**: The loop iterates over `constant.CertManagerIssuers` (which contains both `cs-ss-issuer` and `cs-ca-issuer`) unconditionally, so `cs-ss-issuer` is recreated on every reconcile even when BYO CA is active.

**Change**:
- Always deploy `cs-ca-issuer` (downstream workloads depend on it).
- Only deploy `cs-ss-issuer` when `deployRootCert == true`.
- Split the single issuer loop into two separate deploy calls, gated by `deployRootCert`.

---

### Change 2: Improve `IsBYOCert()` detection logic

**Location**: `internal/controller/bootstrap/init.go`, lines 1913–1943

**Current behaviour**: Returns `true` whenever `cs-ca-certificate-secret` exists and no `cs-ca-certificate` CR is found — this incorrectly flags re-installations with leftover secrets as BYO.

**New three-way detection logic**:

| Scenario | Detection Condition | Result |
|----------|--------------------|---------|
| **BYO Method 1** — user manually created the secret | Secret exists + no Certificate CR + secret annotation `cert-manager.io/issuer-name` is **absent or not** `cs-ss-issuer` | `true` (BYO) |
| **BYO Method 2** — user owns the Certificate CR | Certificate CR exists + `spec.issuerRef.name != "cs-ss-issuer"` | `true` (BYO) |
| **Re-installation / leftover secret** | Secret exists + no Certificate CR + secret annotation `cert-manager.io/issuer-name == cs-ss-issuer` | `false` (operator-managed, recreate) |

Additionally, wherever `shouldAddOwnerReference` is evaluated for `cs-ca-certificate`, skip adding ownerReference when BYO Method 2 is active (Certificate CR exists but is user-governed) to avoid hijacking ownership.

---

### Change 3: Skip label mutations on `cs-ca-certificate` when BYO CA is active

**Locations**:
- `ConfigCertManagerOperandManagedByOperator()` — `internal/controller/bootstrap/init.go`, around line 2316
- `UpdateManageCertRotationLabel()` — `internal/controller/bootstrap/init.go`, around line 2712

**Current behaviour**: Both functions unconditionally write labels (`operator.ibm.com/managedByCsOperator`, `manage-cert-rotation`) onto `cs-ca-certificate`, even when the CR is user-governed.

**Change**:
- Pass `deployRootCert bool` (or re-derive BYO status) into both functions.
- When `!deployRootCert`, skip any label write on `cs-ca-certificate` in both functions.