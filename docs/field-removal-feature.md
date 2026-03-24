# Field Removal Feature

## Overview

The Field Removal feature automatically removes fields from OperandConfig when they are removed from CommonService CR. This feature is **always enabled** and uses hash-based comparison to detect resource changes.

## How It Works

### Previous Behavior
- When you add a field to CommonService CR → cs-operator adds it to OperandConfig ✅
- When you remove a field from CommonService CR → cs-operator does **NOT** remove it from OperandConfig ❌

### New Behavior (Always Active)
- When you add a field to CommonService CR → cs-operator adds it to OperandConfig ✅
- When you remove a field from CommonService CR → cs-operator removes it from OperandConfig ✅
- Resources are compared using SHA-256 hash to detect changes ✅

## How It Works Internally

1. **Aggregation Phase**: The operator collects configurations from ALL CommonService CRs in the cluster
2. **Merge Phase**: Configurations are merged using existing merge rules
3. **Removal Phase** (ALWAYS ACTIVE):
   - The operator builds a "desired state" from all CommonService CRs
   - Fields in OperandConfig that are NOT in the desired state are removed
   - Resources are compared using SHA-256 hash values to detect changes
   - This includes both `spec` fields and `resources`

## Edge Cases Handled

### Multiple CommonService CRs
If you have multiple CommonService CRs, a field is only removed from OperandConfig if it's not present in **ANY** CommonService CR.

Example:
- CommonService A has field X
- CommonService B does NOT have field X
- Result: Field X is kept in OperandConfig (because it's in A)

### Size Profiles
The feature respects size profile templates. Fields from base size templates are preserved unless explicitly overridden.

### Base Configuration
Fields that come from the base OperandConfig (not from any CommonService CR) are preserved.

## What Gets Removed

### Spec Fields
Custom resource specifications under `services[].spec` that are no longer in any CommonService CR.

Example:
```yaml
# Before: CommonService has this
services:
  - name: ibm-im-operator
    spec:
      authentication:
        replicas: 3
```

```yaml
# After: You remove the authentication spec from CommonService
# Result: authentication spec is removed from OperandConfig
```

### Resources
Resources under `services[].resources` that are no longer in any CommonService CR.

Example:
```yaml
# Before: CommonService has this
services:
  - name: ibm-im-operator
    resources:
      - apiVersion: v1
        kind: ConfigMap
        name: my-custom-config
```

```yaml
# After: You remove the resource from CommonService
# Result: The ConfigMap resource is removed from OperandConfig
```

## Backward Compatibility

⚠️ **Breaking Change**: Field removal is now always enabled. When you remove a field from CommonService CR, it will be automatically removed from OperandConfig.

**Migration Notes**:
- If you have manually edited OperandConfig, those changes may be removed if they're not in any CommonService CR
- Review your CommonService CRs to ensure all desired configurations are present
- Fields from base configuration templates are preserved

## Use Cases

### Use Case 1: Cleanup After Testing
You added a custom configuration for testing and want to remove it cleanly:
1. Enable `enableFieldRemoval: true`
2. Remove the test configuration from CommonService CR
3. The operator automatically removes it from OperandConfig

### Use Case 2: Configuration Drift Prevention
Ensure OperandConfig stays in sync with CommonService CRs:
1. Enable `enableFieldRemoval: true`
2. All changes to CommonService CRs (additions and removals) are reflected in OperandConfig

### Use Case 3: Multi-Tenant Cleanup
When a tenant's CommonService CR is deleted, their custom configurations are automatically cleaned up from the shared OperandConfig.

## Limitations

1. **Base Configuration**: Fields from the base OperandConfig template are not removed
2. **Manual Changes**: If you manually edit OperandConfig, those changes may be removed if they're not in any CommonService CR
3. **Timing**: Removal happens during reconciliation, so there may be a brief delay

## Troubleshooting

### Field Not Being Removed
Check:
1. Is `enableFieldRemoval: true` in at least one CommonService CR?
2. Is the field present in another CommonService CR?
3. Check operator logs for removal messages: `Removed orphaned CR spec: ...`

### Unexpected Field Removal
Check:
1. Is the field defined in any CommonService CR?
2. Is the field part of the base configuration?
3. Review operator logs for removal activity

## Implementation Details

### Files Modified
- `api/v3/commonservice_types.go`: Added `EnableFieldRemoval` field
- `internal/controller/operandconfig.go`: Added field removal logic
- `config/crd/bases/operator.ibm.com_commonservices.yaml`: Updated CRD

### Key Functions
- `checkIfFieldRemovalEnabled()`: Checks if feature is enabled
- `buildDesiredStateFromAllCRs()`: Aggregates all CommonService configurations
- `removeOrphanedFields()`: Removes fields not in desired state
- `removeOrphanedSpecFields()`: Removes orphaned spec fields
- `removeOrphanedResources()`: Removes orphaned resources

## Testing

To test the feature:

1. Create a CommonService CR with a custom field
2. Verify the field appears in OperandConfig
3. Enable `enableFieldRemoval: true`
4. Remove the custom field from CommonService CR
5. Verify the field is removed from OperandConfig

## Future Enhancements

Potential improvements:
- Per-service field removal control
- Dry-run mode to preview removals
- Removal history/audit log
- Selective field removal (whitelist/blacklist)