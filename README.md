# provider-doppler

[![CI](https://github.com/oxlev/provider-doppler/actions/workflows/ci.yml/badge.svg)](https://github.com/oxlev/provider-doppler/actions/workflows/ci.yml)

An experimental Crossplane provider generated with Upjet v2 from
[Doppler's Terraform provider](https://github.com/DopplerHQ/terraform-provider-doppler),
pinned to **1.21.5**. All managed APIs are **v1alpha1**; live acceptance tests
are required before a production release. Development packages are published to
`ghcr.io/oxlev/provider-doppler` after CI passes; see the publishing job summary
for the exact immutable SHA tag.

## Supported resources

| Kind | External name | Terraform import ID |
| --- | --- | --- |
| Project | Doppler-assigned project slug | `project` |
| Environment | Environment slug | `project.environment` |
| Config | Config name | `project.environment.config` |
| Secret | Secret name (for example `API_KEY`) | `project.config.name` |

Namespaced APIs: `secrets.doppler.m.crossplane.io/v1alpha1` (recommended for
Crossplane v2). Legacy cluster APIs: `secrets.doppler.crossplane.io/v1alpha1`.
ProviderConfig APIs use `doppler[.m].crossplane.io/v1beta1`. Namespaced resources
support both namespace-local ProviderConfig and ClusterProviderConfig.
Use a current Crossplane v2 installation; the development template targets 2.2.1+.

References/selectors resolve project slugs, environment slugs, and config names.
Do not put composite Terraform import IDs in `crossplane.io/external-name`:
parent identifiers are supplied through `forProvider` or references. Project
`forProvider.name` is a display name, not its server-assigned slug.

## Quick start

### 1. Install a package

Use `ghcr.io/oxlev/provider-doppler:v0.0.0-sha.<full-commit-SHA>` from a successful
CI publishing job. Version tags (`vX.Y.Z` or prereleases) publish that version.
These are multi-platform packages for amd64 and arm64. There is no floating
`latest` tag and a development build is not a production release.

Alternatively, build and push to a registry you control (requires Docker, Go,
the Crossplane CLI, and registry push access):

```sh
git clone --recurse-submodules https://github.com/oxlev/provider-doppler.git
cd provider-doppler
make build.all PLATFORMS=linux_amd64 VERSION=v0.1.0-alpha.1
# Authenticate to your registry using its recommended login mechanism first.
# Replace REGISTRY/OWNER with your registry and organization.
crossplane xpkg push \
  --package-files _output/xpkg/linux_amd64/provider-doppler-v0.1.0-alpha.1.xpkg \
  REGISTRY/OWNER/provider-doppler:v0.1.0-alpha.1
```

Set `spec.package` in `examples/install.yaml` to that exact image reference, then:

```sh
kubectl apply -f examples/install.yaml
kubectl wait provider.pkg.crossplane.io/provider-doppler \
  --for=condition=Healthy --timeout=5m
```

For private registries, create a `kubernetes.io/dockerconfigjson` Secret in the
Crossplane namespace and add `spec.packagePullSecrets: [{name: registry-creds}]`
to the Provider manifest. Registry credentials are separate from Doppler API
credentials. Build `linux_arm64` instead for ARM clusters. Do not install an
amd64-only package into an ARM cluster.

### 2. Configure authentication

Supply a JSON file containing `{"doppler_token":"..."}` from your secret manager.
Keep it outside the repository with restrictive permissions. In a disposable
cluster using the standard `crossplane-system` namespace:

```sh
kubectl -n crossplane-system create secret generic example-creds \
  --from-file=credentials=/secure/path/doppler-credentials.json
kubectl apply -f examples/namespaced/providerconfig/providerconfig.yaml
# The input file must contain only the raw application secret value.
kubectl -n crossplane-system create secret generic app-input \
  --from-file=api-key=/secure/path/application-api-key
```

Use a namespace-local ProviderConfig by default. The separate
`clusterproviderconfig.yaml` example is an alternative for administrator-managed
shared credentials, not required for this quick start.

### 3. Create and observe resources

```sh
kubectl apply -f examples/namespaced/resources.yaml
kubectl -n crossplane-system get projects.secrets.doppler.m.crossplane.io
kubectl -n crossplane-system get environments.secrets.doppler.m.crossplane.io
kubectl -n crossplane-system get configs.secrets.doppler.m.crossplane.io
kubectl -n crossplane-system get secrets.secrets.doppler.m.crossplane.io
kubectl -n crossplane-system wait secrets.secrets.doppler.m.crossplane.io/crossplane-example-api-key \
  --for=condition=Ready --timeout=5m
```

The example creates a project, a `ci` environment, and an `API_KEY` secret.
It observes the `ci` root config automatically created by Doppler. Kubernetes
object names and Doppler names are deliberately separate. References allow dependency reconciliation without manual ordering.
If reconciliation fails, inspect resource conditions/events with `kubectl describe`;
verify token permissions, reference readiness, and Secret names/keys first.

Deleting the example manifests **orphans** the Doppler resources intentionally.
To clean up the external objects, explicitly enable Delete and remove children
before parents, or delete them manually in the disposable Doppler project.
Never delete a production project merely to clean up an example.

## Credentials and safety

Store JSON `{"doppler_token":"..."}` in a Kubernetes Secret and reference its
`credentials` key from ProviderConfig. Only `source: Secret` is accepted; there
is no ambient environment-token fallback. Use a least-privilege Doppler service
account token for the projects and operations you manage. Config-scoped service
tokens cannot administer projects/environments. Restricted secret values require
an appropriate service account or service token.

See `examples/namespaced/providerconfig/secret.yaml.tmpl` and
`examples/namespaced/providerconfig/providerconfig.yaml`. Never commit populated
credentials. Namespaced ProviderConfig credential lookups are confined to the
managed resource's namespace. ClusterProviderConfig is an administrator-controlled
shared credential boundary; restrict who may create/use it and managed resources.

Secret input uses `valueSecretRef`, not plaintext in the managed resource. Upjet
marks both raw and computed values sensitive and excludes them from ordinary
status fields. Terraform state, provider working directories, Kubernetes Secrets,
and connection details must still be treated as sensitive: use encryption at
rest, restrictive RBAC, and avoid debug logging in production.

The examples in `examples/namespaced/resources.yaml` deliberately omit the
`Delete` management policy. Deleting a Kubernetes object therefore leaves its
Doppler object intact. The generated default is full management, including
Delete. Doppler project/environment deletion can cascade to secrets/configs;
only enable deletion deliberately. Keep management-policy support enabled.

For adoption, set the external-name annotation, populate parent identifiers,
and start with `managementPolicies: [Observe]`. Confirm identity and observed
state before enabling Update or Delete. Doppler automatically creates default
environments/configs: adopt those rather than trying to recreate them. The
example uses a separate `ci` environment and observes its automatic `ci` root config.

### Known live-test limitations

With Terraform provider 1.21.5, reading a nonexistent branch config can return
HTTP 400 (`This token does not have access to requested config`) rather than
404. Terraform therefore fails refresh before Crossplane can create the config.
The provider deliberately does not reinterpret authorization errors as absence.
Use an existing config (initially with `managementPolicies: [Observe]`) or the
root config automatically created with an environment. Branch-config creation
and recovery after its external deletion remain blocked for affected tokens.

Reference resolution waits for parents to be both Ready and Synced before
extracting their external names. A merge-patch resolver avoids a pinned-runtime
bug where incremental server-side apply patches drop previously resolved fields.
Existing resolved references retain normal `IfNotPresent` semantics; clear the
resolved field or use `policy.resolve: Always` when deliberately retargeting one.
See [live-test notes](docs/live-reconciliation.md) for validation and limitations.

## Development

Requires Go matching `go.mod`, Git, GNU Make, curl, unzip, and goimports. Docker
and the Crossplane CLI are additionally required for image/package builds.

```sh
git submodule update --init --recursive
go install golang.org/x/tools/cmd/goimports
export PATH="$(go env GOPATH)/bin:$PATH"
make generate
go test ./...
go vet ./...
go build -o /tmp/provider-doppler ./cmd/provider
# Build runtime images and Crossplane packages (requires Docker):
make build
```

Generation fetches the pinned Terraform schema and documentation, generates both
API scopes, reference resolvers, controllers, deepcopy methods, and CRDs. Edit
`config/` and handwritten ProviderConfig/client code, not `zz_*` files or CRDs.
The Terraform CLI is pinned to MPL-licensed 1.5.7 by the upstream build tooling.
The runtime image includes Terraform and the pinned Doppler plugin and runs as a
non-root user. CI has separate formatting/workflow-lint/vet, generation-drift,
and race-enabled unit-test jobs. After these pass, it builds installable `.xpkg`
artifacts for both amd64 and arm64, available from the Actions run for 14 days.
PR CI needs no credentials and never publishes. Trusted pushes to `main` and
version tags publish multi-platform packages using the job-scoped `GITHUB_TOKEN`
with `packages: write`. The `ci/ghcr-publishing` bootstrap branch may publish SHA
tags only, so this pipeline can be tested before merge. Publishing is followed
by an isolated kind smoke test with no Doppler credentials or live API calls.

GitHub container packages initially default to private even in a public repository.
After the first push, an organization owner must set the package visibility to
public in GHCR package settings for anonymous installation; the smoke job
intentionally checks anonymous pulls. No registry secret is used to mask a
visibility problem. Live Doppler lifecycle testing remains a release gate.

Run the same disposable-cluster smoke test locally (requires kind, kubectl, Helm,
and Docker):

```sh
PROVIDER_PACKAGE=ghcr.io/oxlev/provider-doppler:v0.0.0-sha.<full-commit-SHA> \
  bash scripts/kind-smoke.sh
```

The script uses a private temporary kubeconfig, refuses to reuse existing clusters,
and deletes only its own cluster on exit. It verifies package health, CRDs in both
API scopes, and safe reconciliation failure for missing credentials. It does not
claim to validate create/update/delete in Doppler.

See [CONTRIBUTING.md](CONTRIBUTING.md) for validation and provider conventions.

## Before release

1. Build/install the package in a disposable Crossplane v2 cluster.
2. Test create, drift correction, adoption, Observe-only behavior, orphaning, and
   explicit deletion against a disposable Doppler project in both API scopes.
3. Verify namespace credential isolation, ProviderConfig usage tracking, missing
   Secret behavior, and raw/computed secret handling without value leakage.
4. Test provider restart/recovery and reference ordering; delete children before
   parents and verify no unintended cascading deletion.
5. Review dependency versions (the upstream template pins some prereleases),
   image provenance/checksums, RBAC, and API compatibility before publishing.

## Provenance

Bootstrapped from crossplane/upjet-provider-template commit
`fa46ff9a40b1858ad4f91cda09b4ad695fe71bc0`, using its pinned crossplane/build
submodule. Upstream Apache-2.0 license and generated attribution are preserved.
