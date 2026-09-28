# Live reconciliation findings

Tested against Crossplane 2.4.2 in disposable kind cluster `doppler-test`, with
Doppler Terraform provider 1.21.5. No test credentials belong in this repository.

## Fixed: resolved identifiers disappearing

The pinned crossplane-runtime `APISimpleReferenceResolver` computes a JSON merge
delta but submits it as server-side apply. A subsequent delta omits previously
resolved fields; SSA relinquishes ownership and can delete those fields. This
caused alternating missing project/config values, rapidly increasing generations,
and Terraform errors such as `The argument "config" is required`.

`internal/references.NewResolver` uses an actual JSON merge patch with optimistic
locking, preserves unrelated fields, and does not write on unchanged resolution.
The embedded controller template enables it for both resource scopes. Generated
controllers and resolvers must be regenerated, not hand-edited.

## Fixed: using a parent identifier before creation

Environment and Config external-name annotations are present before their Doppler
objects exist. The default extractor allowed children to attempt creation too
soon (observed as `Invalid environment id`). `ReadyExternalName` returns an
identifier only when the parent is Ready and Synced. Normal Crossplane reference
retry/backoff still applies; a complete chain can take several minutes to settle.
Existing populated references are not rechecked unless their resolve policy is
`Always`.

## Remaining upstream/API limitation: missing branch configs

A direct GET of a nonexistent config under the test project returned HTTP 400:

```
This token does not have access to requested config 'ci_fixed'
```

Terraform's Config read handler only clears state for HTTP 404 or its explicit
not-found error type. Upjet seeds state from the composite external identifier,
so refresh fails before Create. This also affects recovery after failed creation
or out-of-band deletion. This is independent of reference persistence/order.

Do not classify this authorization message as absence or blindly clear state:
it may also mean the token cannot access an existing config. A robust upstream
fix needs an unambiguous missing-resource response or an authoritative existence
check with suitable permissions. No such error suppression is included here.

The quick-start example instead adopts the `ci` root config that Doppler creates
with the environment, using `managementPolicies: [Observe]`. This validates
Config observation, not branch-config creation.

### Plain Terraform reproduction

Using the same token, Terraform 1.5.7 and Doppler provider 1.21.5, a disposable
branch named `ci_tf_probe` was tested under the existing test project's `ci`
environment:

1. `terraform import doppler_config.probe PROJECT.ci.ci_tf_probe` failed during
   refresh with the same access error while the config did not exist.
2. `terraform apply` from empty resource state then successfully created that
   exact config. The plan contained one create and no updates or deletions.
3. Crossplane subsequently adopted the config with `managementPolicies: [Observe]`
   and reached Ready.

This confirms the token can create branch configs and isolates the failure to
reading a nonexistent config, not missing create permissions or references.
Unlike plain Terraform's empty-state create, Upjet synthesizes state from the
external identifier and performs a read before attempting creation.

Minimal configuration (supply `DOPPLER_TOKEN` through a secure environment, not
in this file; use a disposable project/environment and an unused branch name):

```hcl
terraform {
  required_providers {
    doppler = {
      source  = "dopplerhq/doppler"
      version = "1.21.5"
    }
  }
}
provider "doppler" {}
resource "doppler_config" "probe" {
  project     = "YOUR_DISPOSABLE_PROJECT"
  environment = "ci"
  name        = "ci_tf_probe"
  lifecycle { prevent_destroy = true }
}
```

Do not bypass refresh or classify the access error as not-found as a general
workaround. The successful Terraform-created branch is left in place; Crossplane
only observes it, and the Terraform workspace/state is outside the repository
in a permission-restricted temporary directory.

## Live validation

A locally rebuilt controller was layered on the public package runtime and loaded
into kind. A DeploymentRuntimeConfig overrides only the controller image; the
published package and Terraform plugin are unchanged.

- The previously failing Secret reconciled after installing the fixed controller.
- A fresh Project → Environment → observed root Config → Secret chain reached
  Ready when submitted together, without manually sequencing creation.
- Updating the Kubernetes input Secret was reflected in Doppler; verification
  compared values in memory and did not print them.
- A fresh branch-config test still encountered the upstream HTTP 400 limitation
  and was paused to stop retries.
- After a provider Deployment restart (losing its ephemeral Terraform workspaces),
  another input-Secret update propagated to Doppler. The fresh test chain remained
  Ready/Synced and its managed-resource JSON did not expose the test secret value.
- Plain Terraform created a branch successfully; Crossplane observed that branch
  successfully, as detailed above.

All test resources omit Delete. Removing their Kubernetes manifests or the kind
cluster does **not** clean up Doppler projects. Clean up externally only after
explicitly deciding to delete the disposable projects, children before parents.

Unit tests cover incremental reference resolution, stable/no-op resolution,
merge-patch type, optimistic locking, and readiness gating across both scopes.
The full Go test suite and vet pass; targeted resolver/config/client tests also
pass under the race detector. This is not the full production acceptance matrix:
branch lifecycle, deletion, external drift, credential isolation live tests, and
broader restart/recovery scenarios still need validation.
