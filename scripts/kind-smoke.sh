#!/usr/bin/env bash
# Install an anonymously pullable package in an isolated, disposable control plane.
# No Doppler credentials or outbound Doppler API calls are required.
set -euo pipefail
: "${PROVIDER_PACKAGE:?Set PROVIDER_PACKAGE to the published GHCR package}"
CROSSPLANE_VERSION=${CROSSPLANE_VERSION:-2.2.1}
CLUSTER_NAME=${CLUSTER_NAME:-doppler-smoke-$(date +%s)}
TEST_DIR=$(mktemp -d)
export KUBECONFIG="$TEST_DIR/kubeconfig"
created=false
cleanup() {
  code=$?
  if [[ "$created" == true ]]; then
    if [[ "$code" != 0 ]]; then
      kubectl get providers,providerrevisions,pods -A || true
      kubectl get events -A --sort-by=.lastTimestamp || true
      kubectl -n crossplane-system logs -l pkg.crossplane.io/provider=provider-doppler --tail=100 || true
    fi
    kind delete cluster --name "$CLUSTER_NAME"
  fi
  rm -f "$KUBECONFIG"
  rmdir "$TEST_DIR" 2>/dev/null || true
  exit "$code"
}
trap cleanup EXIT
# Refuse to reuse or delete an existing cluster.
if kind get clusters | grep -Fxq "$CLUSTER_NAME"; then
  echo "Cluster already exists: $CLUSTER_NAME" >&2
  exit 1
fi
created=true
kind create cluster --name "$CLUSTER_NAME" --image kindest/node:v1.35.0 --kubeconfig "$KUBECONFIG" --wait 180s
helm upgrade --install crossplane crossplane \
  --repo https://charts.crossplane.io/stable --version "$CROSSPLANE_VERSION" \
  --namespace crossplane-system --create-namespace --wait --timeout 5m
kubectl apply -f - <<EOF
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-doppler
spec:
  package: ${PROVIDER_PACKAGE}
  packagePullPolicy: Always
EOF
kubectl wait provider.pkg.crossplane.io/provider-doppler --for=condition=Installed --timeout=5m
kubectl wait provider.pkg.crossplane.io/provider-doppler --for=condition=Healthy --timeout=5m
# Activate both API scopes explicitly rather than relying on installation defaults.
kubectl apply -f - <<'EOF'
apiVersion: apiextensions.crossplane.io/v1alpha1
kind: ManagedResourceActivationPolicy
metadata:
  name: doppler-smoke
spec:
  activate:
    - '*.doppler.crossplane.io'
    - '*.doppler.m.crossplane.io'
EOF
for scope in doppler.crossplane.io doppler.m.crossplane.io; do
  for resource in projects environments configs secrets; do
    # CRD creation after activation is asynchronous.
    for ((i=0; i<60; i++)); do
      if kubectl get "crd/${resource}.secrets.${scope}" >/dev/null 2>&1; then break; fi
      sleep 2
    done
    kubectl wait "crd/${resource}.secrets.${scope}" --for=condition=Established --timeout=60s
  done
done
kubectl apply -f - <<'EOF'
apiVersion: doppler.m.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: smoke
  namespace: crossplane-system
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: deliberately-missing
      key: credentials
---
apiVersion: secrets.doppler.m.crossplane.io/v1alpha1
kind: Project
metadata:
  name: missing-credentials-smoke
  namespace: crossplane-system
  annotations:
    crossplane.io/external-name: never-contact-doppler
spec:
  managementPolicies: [Observe]
  forProvider:
    name: never-contact-doppler
  providerConfigRef:
    kind: ProviderConfig
    name: smoke
EOF
# Missing credentials must produce a reconcile error, not a crash or API call.
resource=projects.secrets.doppler.m.crossplane.io/missing-credentials-smoke
kubectl -n crossplane-system wait "$resource" --for=condition=Synced=false --timeout=120s
message=$(kubectl -n crossplane-system get "$resource" -o 'jsonpath={.status.conditions[?(@.type=="Synced")].message}')
if [[ "$message" != *deliberately-missing* || "$message" != *"not found"* ]]; then
  echo "Unexpected reconcile failure: $message" >&2
  exit 1
fi
kubectl wait provider.pkg.crossplane.io/provider-doppler --for=condition=Healthy --timeout=60s
printf 'PASS: public package installed, both API scopes established, missing credentials handled safely.\n'
