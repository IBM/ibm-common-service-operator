# Project Overview

## What this project is

`ibm-common-service-operator` is the **entry point operator** for IBM Cloud Pak foundational services (CPfs). It is a meta operator (per the operator-sdk framework) whose primary job is to install and lifecycle-manage `operand-deployment-lifecycle-manager` (ODLM), which in turn installs all other CPfs operators.

It is also the **sole intended source of configuration** for CPfs — configuration placed in the `CommonService` CR propagates to the appropriate downstream CPfs services. Some legacy configuration sources exist outside of the `CommonService` CR, but these are exceptions.

Additionally, it owns a set of **certificate rotation controllers** that were moved out of `ibm-cert-manager-operator` to simplify that operator's scope.

## Main goals

1. **Installation entry point** — provide Cloud Paks with a single operator to install that brings up all of CPfs.
2. **Configuration propagation** — act as the single source of truth for CPfs configuration via the `CommonService` CR.
3. **Backwards compatibility** — changes must not break downstream Cloud Pak consumers.

## Key stakeholders / users

- **IBM Cloud Pak teams** — primary consumers; they install this operator to get CPfs.
- **End customers** — not expected to install this standalone; their exposure to CPfs is typically through a Cloud Pak. When a Cloud Pak service has issues, the root cause may trace back here.

## Important architecture notes

### CommonService CR — master vs. non-configurable
There are two kinds of `CommonService` CRs in a cluster:
- **Master / configurable CR** — the authoritative source of configuration. Changes here propagate downstream.
- **Non-configurable CRs** — read-only in terms of most configuration, but can still set CPU/memory sizing for operands.

This distinction is a common source of confusion and a frequent area of bugs. Always clarify which kind of CR is involved when debugging configuration issues.

### Certificate rotation controllers
Controllers for certificate rotation live here (not in `ibm-cert-manager-operator`) as a deliberate simplification. Be aware of this when tracing certificate-related issues.

### Legacy configuration sources
Some CPfs configuration still comes from sources outside the `CommonService` CR. These are legacy and should not be expanded.

## Key workflows

- **Bug fixes** — most common; often traced from a Cloud Pak team reporting an issue with a CPfs service.
- **Adding new configuration** to the `CommonService` CR — as requested by Cloud Pak teams.
- **Debugging live cluster issues** — primary day-to-day mode of work.
- **Occasional new features** — requested by Cloud Paks.

## Important modules / areas

| Area | Notes |
|------|-------|
| `api/v3/commonservice_types.go` | `CommonService` CRD type definitions — primary configuration surface |
| Certificate rotation controllers | Moved here from `ibm-cert-manager-operator` |
| ODLM installation logic | This operator installs and manages ODLM as its primary operand |
