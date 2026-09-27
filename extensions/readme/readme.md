# Doppler provider for Crossplane

Manage Doppler **Projects**, **Environments**, **Configs**, and **Secrets** using
Kubernetes manifests and Crossplane reconciliation. Built with Upjet v2 against
Doppler Terraform provider 1.21.5.

- Namespaced managed resources for Crossplane v2, plus legacy cluster-scoped APIs.
- Kubernetes Secret-backed Doppler credentials and secret values.
- Project, environment, and config references/selectors.
- Observe-only adoption and explicit management/deletion policies.

**Alpha:** no production release is published. Live Doppler lifecycle testing is
required before production use. CI builds are not acceptance-tested releases.

See the [usage guide](https://github.com/oxlev/provider-doppler/blob/main/README.md)
for building/installing packages, private registry authentication, ProviderConfig,
and a complete example. Start with the namespaced examples and least-privilege
service account credentials. Example deletion policies intentionally orphan
Doppler objects; project deletion can cascade to configs and secrets.
