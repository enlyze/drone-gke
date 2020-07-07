# drone-sops-gke

Drone plugin to deploy container images to Kubernetes on Google Container Engine with kubernetes kustomizations and mozilla sops.
For the usage information and a listing of the available options please take a look at [the docs](DOCS.md).

Simplify deploying to Google Kubernetes Engine.
Derive the API endpoints and credentials from the Google credentials and open the yaml file to templatization and customization with each Drone build.

## Links

- Usage [documentation](DOCS.md)
- Sops [repo](https://github.com/mozilla/sops)
- Kustomize.io [homepage](https://kustomize.io)
- Drone.io [builds](https://cloud.drone.io/nytimes/drone-gke)
- Contributing [documentation](CONTRIBUTING.md)

## Releases and versioning


### Kubernetes API

Since the [237.0.0 (2019-03-05) Google Cloud SDK][sdk], the container image contains multiple versions of `kubectl`.
The corresponding client version that matches the cluster version will be used automatically.
This follows the minor release support that [GKE offers](https://cloud.google.com/kubernetes-engine/versioning-and-upgrades).

If you want to use a different version, you can specify the version of `kubectl` used with the [`kubectl_version` parameter][version-parameter].

[sdk]: https://cloud.google.com/sdk/docs/release-notes#23700_2019-03-05
[version-parameter]: DOCS.md#kubectl_version


## Usage

> :warning: For usage within in a `.drone.yml` pipeline, please take a look at [the docs](DOCS.md)

Executing locally from the working directory:

```sh
# Deploy the manifest templates in local-example/
cd local-example/

# Set to the path of your GCP service account JSON-formatted key file
# This must have both, GKE and KMS rights within the GCP project
export SERVICE_ACCOUNT_PATH=/home/my-user/credentials.json

# Set to your cluster
export PLUGIN_CLUSTER_NAME=yyy

# Set to your cluster's zone
export PLUGIN_ZONE=zzz

# the kustomization overlay to use
export PLUGIN_OVERLAY=path/to/overlay-dir

# Set to a namespace within your cluster's
export PLUGIN_NAMESPACE=my-supercool-namespace

# Example variables referenced within .kube.yml
export PLUGIN_VARS="$(cat vars.json)"
# {
#   "app": "echo",
#   "env": "dev",
#   "image": "gcr.io/google_containers/echoserver:1.4"
# }

# Execute the plugin
docker run --rm \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  -e GOOGLE_APPLICATION_CREDENTIALS_JSON="$(cat ${SERVICE_ACCOUNT_PATH})" \
  -e PLUGIN_CLUSTER \
  -e PLUGIN_ZONE \
  -e PLUGIN_NAMESPACE \
  -e PLUGIN_OVERLAY \
  -e PLUGIN_VARS \
  docker.pliro.enlyze.com/enlyze/drone-sops-gke --dry-run --verbose

# Remove --dry-run to deploy
```
