# Contributing

## Provider conventions

This repository follows the [Upjet provider template](https://github.com/crossplane/upjet-provider-template),
[provider-keycloak](https://github.com/crossplane-contrib/provider-keycloak), and
[provider-upjet-github](https://github.com/crossplane-contrib/provider-upjet-github):

- Separate generation-drift, lint, unit-test, and runtime/package-build CI checks.
- Checked-in generated API types, resolvers, controllers, and CRDs for review.
- Standard Crossplane package metadata and correct `safe-start` capability.
- Resource examples, external-name/import guidance, and Secret-backed credentials.
- Pinned upstream provider version and build submodule; explicit resource allowlist.

We intentionally do not copy cloud-provider credentials, publishing permissions,
comment-triggered privileged tests, or marketplace credentials into pull-request
CI. Both image architectures are built without authenticating to Doppler. Actions
are commit-pinned and workflows use read-only permissions and timeouts.

## Validate a change

```sh
git submodule update --init --recursive
go install golang.org/x/tools/cmd/goimports
export PATH="$(go env GOPATH)/bin:$PATH"
make generate
git diff --check
go test -race -count=1 ./...
go vet ./...
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
make build.all PLATFORMS=linux_amd64 VERSION=v0.0.0-dev
```

Commit generation changes along with handwritten code. Re-running generation
must leave a clean diff. Go dependencies follow `go.mod`; do not install an
unrelated latest goimports version. Builds need sufficient disk for Kubernetes
Go dependencies and Docker layers. ARM image builds on an x86 host also require
QEMU/binfmt emulation (CI configures it).

For new managed resources, review the upstream create/read/update/delete and
import behavior first. Add an explicit external-name mapping and tests, mark
sensitive fields, configure references/selectors, and add examples. Keep new
APIs alpha until lifecycle tests pass. Do not regenerate all upstream resources
merely because they exist in the schema.

## Pull requests and release gates

Describe user-visible changes, API compatibility, test evidence, and any missing
live validation. Never add Doppler tokens, generated Terraform state, kubeconfigs,
or populated Secret manifests. A passing package-build job does not prove live
Doppler reconciliation works.

Before release, run the disposable-project checklist in README.md. Review CRD
changes for compatibility and confirm both image architectures work. Publish only
an explicitly chosen version after review; merging a PR does not publish a package.
Private GHCR packages require Crossplane `packagePullSecrets`. Changing repository
or package visibility to public is a separate owner decision.
