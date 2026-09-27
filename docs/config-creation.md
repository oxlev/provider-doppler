# Config creation: desired name versus confirmed identity

## Root cause

Doppler Terraform provider 1.21.5 reports an ambiguous HTTP 400 authorization
error when reading some nonexistent configs. Previously, our external-name
adapter synthesized `project.environment.config` before Create, so Upjet seeded
Terraform state with a nonexistent resource ID. Refresh then failed before
creation could run. Plain Terraform creates the same config with the same token
when starting from empty resource state.

## Fix

A new Config supplies its desired Doppler name through `spec.forProvider.name`
(or `initProvider.name`) and omits `crossplane.io/external-name`. Its initial
Terraform ID is empty. Terraform can therefore perform a normal create and the
provider records the confirmed external name from the resulting state.

The Config controller no longer initializes the external name from the Kubernetes
object name. Other resource kinds keep their existing naming behavior.

An explicitly annotated Config still means adoption of an existing identity;
its composite Terraform ID and import format are unchanged. The new name field
is optional in the CRD so old annotated manifests remain valid. For new configs,
a missing/blank desired name produces an actionable reconciliation error.
If desired name and confirmed external name differ, reconciliation fails rather
than attempting a rename or replacement.

This change never interprets an authorization error as not-found, probes with
extra permissions, or bypasses Terraform refresh for an existing identity.

## Creation and adoption examples

New branch (use a namespace-local ProviderConfig and an existing test environment):

```yaml
apiVersion: secrets.doppler.m.crossplane.io/v1alpha1
kind: Config
metadata:
  name: test-branch
  namespace: crossplane-system
spec:
  managementPolicies: [Observe, Create, Update, LateInitialize]
  forProvider:
    name: ci_build
    project: YOUR_DISPOSABLE_PROJECT
    environment: ci
  providerConfigRef:
    kind: ProviderConfig
    name: default
```

For adoption, set `metadata.annotations["crossplane.io/external-name"]` to the
existing config name and start with `managementPolicies: [Observe]`.
`forProvider.name` may be omitted in that case.

## Migration and remaining limitations

- Existing successfully reconciled annotated Configs need no migration.
- Old manifests intended to create a missing Config must switch to the new desired
  name field. Confirm the remote config is actually absent before removing any
  external identity. Never clear an existing resource's identity blindly.
- If a config was created but the controller crashed before persisting its
  identity, another Create can return `Name is already in use`. Confirm the
  remote config, then explicitly adopt it; do not delete it just to unblock a test.
- A config that is externally deleted after creation still has a confirmed
  identity. Its refresh may return Doppler's ambiguous access error; automatic
  recovery remains blocked rather than risking unauthorized adoption/recreation.
- Observe-only Configs require an explicit external name to identify what to
  observe. Renaming a managed config is intentionally unsupported.

## Live validation

Using the existing disposable kind cluster, a local package containing the new
Config CRDs was installed through the loopback address of a test registry sidecar
in Crossplane's pod (no Service or host port exposed), with a kind-loaded controller image selected by DeploymentRuntimeConfig.
The registry and package are local test artifacts, not published releases.

Verified with Terraform 1.5.7, Doppler provider 1.21.5, and Crossplane 2.4.2:

- Namespaced Config creation with a desired name and no external-name annotation
  reached Ready/Synced and recorded the correct external name.
- Cluster-scoped Config creation reached Ready/Synced using the same path.
- A duplicate-name creation failed with `Name is already in use` without adopting
  the existing config. That probe was paused to stop retries.
- Creation with an invalid environment failed; correcting the environment retried
  successfully without deleting/recreating the Kubernetes object.
- A Secret referencing the newly created branch reached Ready.
- After restarting the provider, branch Configs and the older adopted Config
  remained Ready/Synced. An input-Secret update propagated into the new branch,
  verified without displaying its value.

All test resources omitted Delete; nothing was externally deleted. These tests
are not a full deletion, drift, or permission-isolation acceptance suite.
