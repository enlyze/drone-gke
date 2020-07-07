#!/usr/bin/env sh
set -eu

export EXTRA_KUBECTL_VERSIONS=$(echo $(list-extra-kubectl-versions))

echo "${PLUGIN_SERVICE_ACCOUNT}" > "/tmp/gcloud.json"
export GOOGLE_APPLICATION_CREDENTIALS="/tmp/gcloud.json"

kustomize build --enable_alpha_plugins templates/overlays/${PLUGIN_OVERLAY} | drone-gke "$@"

echo "== cleanup"
rm -f "/tmp/gcloud.json"
