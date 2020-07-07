#!/usr/bin/env sh
set -eu

export EXTRA_KUBECTL_VERSIONS=$(echo $(list-extra-kubectl-versions))

# write credentials string into file with correct permissions
CREDENTIALS_PATH="/tmp/credentials.json"
echo "${GOOGLE_APPLICATION_CREDENTIALS_JSON}" > "${CREDENTIALS_PATH}"
chmod 600 "${CREDENTIALS_PATH}"

# let's set this to the global application credentials here
export GOOGLE_APPLICATION_CREDENTIALS="${CREDENTIALS_PATH}"

kustomize build --enable_alpha_plugins templates/overlays/${PLUGIN_OVERLAY} | drone-gke "$@"

# clean that up before we go
rm -f "${CREDENTIALS_PATH}"
