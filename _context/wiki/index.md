# ibm-common-service-operator Wiki

Entry point operator for installing IBM Cloud Pak foundational services (CPfs). Acts as a meta operator and the sole intended source of configuration for all CPfs components.

## Contents

| File | What's in it |
|------|-------------|
| [project.md](project.md) | Project overview, goals, key modules, architecture notes |
| [preferences.md](preferences.md) | Working standards, coding style, AI collaboration preferences |

## Quick orientation

- This operator is the **entry point** for CPfs installation — Cloud Paks install this to get all other CPfs operators.
- It installs and manages **ODLM** (`operand-deployment-lifecycle-manager`) as its primary operand.
- The `CommonService` CR is the **sole intended configuration source** for all CPfs services — config here propagates downstream.
- There is a key distinction between a **master/configurable** `CommonService` CR and **non-configurable** ones (which can only set CPU/memory sizing).
- Also contains certificate rotation controllers moved out of `ibm-cert-manager-operator` to simplify that operator.
