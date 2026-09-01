# Working Preferences & Standards

## How I use AI

- **Debugging live cluster issues** — the most common use case. Help trace root causes across operators and CRDs.
- **Understanding unfamiliar code** — orientation to areas of the codebase not recently touched.
- **Bug fixes** — targeted, minimal changes. Understand the issue before suggesting a fix.
- **Occasional feature work** — adding new fields to the `CommonService` CR or new configuration propagation logic.

## Communication preferences

- Be direct and technical. No filler phrases ("Great!", "Certainly!", etc.).
- Investigate before answering — never speculate about code you haven't read.
- When explaining unfamiliar code: start with what it *does*, then how it works.
- Flag assumptions and caveats explicitly, especially around the master vs. non-configurable CR distinction.
- If a change could affect downstream Cloud Pak teams or break backwards compatibility, say so upfront.

## Code style & engineering standards

- **Minimal changes.** Produce the smallest diff that solves the problem. No opportunistic refactors.
- **Trace every changed line** back to the stated requirement.
- **Go conventions.** Follow standard Go idioms and the existing style in the file being edited.
- **Backwards compatibility is the top constraint.** A change that breaks a downstream Cloud Pak consumer is worse than not making the change at all.

## What to be careful about

- **Master vs. non-configurable `CommonService` CR.** Always clarify which is in scope before making configuration-related changes.
- **Configuration propagation.** Changes to `CommonService` CR fields must be traced through to wherever they are consumed downstream — a field added but not propagated does nothing.
- **Legacy configuration sources.** Some config still originates outside the `CommonService` CR. Do not assume the CR is the only source without checking.
- **Certificate rotation controllers.** These live here, not in `ibm-cert-manager-operator`. Be aware when tracing cert-related issues.
