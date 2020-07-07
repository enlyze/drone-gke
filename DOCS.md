# drone-sops-gke Usage

Use this plugin to deploy Docker images to [Google Container Engine (GKE)][gke].

[gke]: https://cloud.google.com/container-engine/

## API Reference

The following parameters are used to configure this plugin:

### `image`

_**type**_ `string`

_**default**_ `''`

_**description**_ reference to a `drone-sops-gke` Docker image

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: docker.pliro.enlyze.com/enlyze/drone-sops-gke
    # ...
```

### `project`

_**type**_ `string`

_**default**_ `''`

_**description**_ GCP project ID; owner of the GKE cluster

_**notes**_ default inferred from the service account credentials provided via [`google_application_credentials_json`](#service-account-credentials)

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: docker.pliro.enlyze.com/enlyze/drone-sops-gke
    settings:
      project: my-gcp-project
      # ...
```

### `zone`

_**type**_ `string`

_**default**_ `''`

_**description**_ GCP zone where GKE cluster is located

_**notes**_ required if [`region`](#region) is not provided

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: docker.pliro.enlyze.com/enlyze/drone-sops-gke
    settings:
      zone: us-east1-b
      # ...
```

### `region`

_**type**_ `string`

_**default**_ `''`

_**description**_ GCP region where GKE cluster is located

_**notes**_ required if [`zone`](#zone) is not provided

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: docker.pliro.enlyze.com/enlyze/drone-sops-gke
    settings:
      region: us-west-1
      # ...
```

### `cluster_name`

_**type**_ `string`

_**default**_ `''`

_**description**_ name of GKE cluster

_**notes**_ required

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: docker.pliro.enlyze.com/enlyze/drone-sops-gke
    settings:
      cluster_name: prod
      # ...
```

### `wait_deployments`

_**type**_ `[]string`

_**default**_ `[]`

_**description**_ wait for the given deployments using `kubectl rollout status ...`

_**notes**_ deployments can be specified as `"<type>/<name>"` as expected by `kubectl`.  If
just `"<name>"` is given it will be defaulted to `"deployment/<name>"`.

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: docker.pliro.enlyze.com/enlyze/drone-sops-gke
    settings:
      wait_deployments:
      - deployment/app
      - statefulset/memcache
      - nginx
      # ...
```

### `wait_seconds`

_**type**_ `int`

_**default**_ `0`

_**description**_ number of seconds to wait before failing the build

_**notes**_ ignored if `wait_deployments` is not set

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: docker.pliro.enlyze.com/enlyze/drone-sops-gke
    settings:
      wait_seconds: 180
      wait_deployments:
      - deployment/app
      - statefulset/memcache
      - nginx
      # ...
```

### `vars`

_**type**_ `map[string]interface{}`

_**default**_

```go
{
  // from $DRONE_BUILD_NUMBER
  "BUILD_NUMBER": "",
  // from $DRONE_COMMIT
  "COMMIT": "",
  // from $DRONE_BRANCH
  "BRANCH": "",
  // from $DRONE_TAG
  "TAG": "",
  // from `project`
  "project": "",
  // from `zone`
  "zone": "",
  // from `cluster`
  "cluster": "",
  // from `namespace`
  "namespace": "",
}
```

_**description**_ variables to use in [`template`](#template) and [`secret_template`](#secret_template)

_**notes**_ see ["Available vars"](#available-vars) for details

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: docker.pliro.enlyze.com/enlyze/drone-sops-gke
    settings:
      vars:
        app_name: echo
        app_image: gcr.io/google_containers/echoserver:1.4
        env: dev
      # ...
```

### `expand_env_vars`

_**type**_ `bool`

_**default**_ `false`

_**description**_ expand environment variables for values in [`vars`](#vars) for reference

_**notes**_ only available for `vars` of type `string`; use `$${var_to_expand}` instead of `${var_to_expand}` (two `$$`) to escape drone's variable substitution

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: docker.pliro.enlyze.com/enlyze/drone-sops-gke
    environment:
      # ;)
      PS1: 'C:${PWD//\//\\\\}>'
    settings:
      expand_env_vars: true
      vars:
        # CLOUD_SDK_VERSION (set by google/cloud-sdk) will be expanded
        message: "deployed using gcloud v$${CLOUD_SDK_VERSION}"
        # PS1 (set using standard drone `environment` field) will be expanded
        prompt: "cmd.exe $${PS1}"
        # HOSTNAME and PATH (set by shell) will not be expanded; the `hostnames` and `env` vars are set to non-string values
        hostnames:
        - example.com
        - blog.example.com
        - "$${HOSTNAME}"
        env:
          path: "$${PATH}"
      # ...

```

### `kubectl_version`

_**type**_ `string`

_**default**_ `''`

_**description**_ version of kubectl executable to use

_**notes**_ see [Using "extra" `kubectl` versions](#using-extra-kubectl-versions) for details

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: docker.pliro.enlyze.com/enlyze/drone-sops-gke
    settings:
      kubectl_version: "1.14"
      # ...
```

### `dry_run`

_**type**_ `bool`

_**default**_ `false`

_**description**_ do not apply the Kubernetes templates

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: docker.pliro.enlyze.com/enlyze/drone-sops-gke
    settings:
      dry_run: true
      # ...
```

### `verbose`

_**type**_ `bool`

_**default**_ `false`

_**description**_ dump available [`vars`](#vars) and the generated Kubernetes [`template`](#template)

_**notes**_ excludes secrets

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: docker.pliro.enlyze.com/enlyze/drone-sops-gke
    settings:
      verbose: true
      # ...
```

## Service Account Credentials

`drone-sops-gke` requires a Google service account and uses its [JSON credential file][service-account] to authenticate.

This must be passed to the plugin as an environment variable called `GOOGLE_APPLICATION_CREDENTIALS_JSON`. While this is a bit long, it better describes what's actually inside of there compared to the original name (`token`).
```yaml
# .drone.yml
---
kind: pipeline
# ...
environment:
  GOOGLE_APPLICATION_CREDENTIALS_JSON:
    from_secret: GOOGLE_CREDENTIALS
```

If not specified via the `project` setting, the plugin also infers the GCP project from the JSON credentials (`service_account`) and retrieves the GKE cluster credentials.

[service-account]: https://cloud.google.com/storage/docs/authentication#service_accounts

### Setting the JSON credential file

#### GUI

Simply copy the contents of the JSON credentials file and paste it directly in the input field (for example for a secret named `GOOGLE_CREDENTIALS`).

#### CLI

```sh
drone secret add \
  --event push \
  --event pull_request \
  --event tag \
  --event deployment \
  --repository org/repo \
  --name GOOGLE_CREDENTIALS \
  --value @gcp-project-name-key-id.json
```

## Cluster Credentials

The plugin attempts to fetch credentials for authenticating to the cluster via `kubectl`.

If connecting to a [regional cluster](https://cloud.google.com/kubernetes-engine/docs/concepts/multi-zone-and-regional-clusters#regional), you must provide the `region` parameter to the plugin and omit the `zone` parameter.

If connecting to a [zonal cluster](https://cloud.google.com/kubernetes-engine/docs/concepts/multi-zone-and-regional-clusters#multi-zone), you must provide the `zone` parameter to the plugin and omit the `region` parameter.

The `zone` and `region` parameters are mutually exclusive; providing both to the plugin for the same execution will result in an error.

## Available vars

These variables are always available to reference in any manifest, and cannot be overwritten by `vars` or `secrets`:

```json
{
  "BRANCH": "master",
  "BUILD_NUMBER": "12",
  "COMMIT": "4923x0c3380413ec9288e3c0bfbf534b0f18fed1",
  "TAG": "",
  "cluster": "my-gke-cluster",
  "namespace": "",
  "project": "my-gcp-proj",
  "zone": "us-east1-a"
}
```

## Expanding environment variables

It may be desired to reference an environment variable for use in the Kubernetes manifest.
In order to do so in `vars`, the `expand_env_vars` must be set to `true`.

For example when using `drone build promote org/repo 5 production -p IMAGE_VERSION=1.0`, to get `IMAGE_VERSION` in `vars`:

```yml
expand_env_vars: true
vars:
  image: my-image:$${IMAGE_VERSION}
```

The plugin will [expand the environment variable][expand] for the template variable.

To use `$${IMAGE_VERSION}` or `$IMAGE_VERSION`, see the [Drone docs][environment] about preprocessing.
`${IMAGE_VERSION}` will be preprocessed to an empty string.

[expand]: https://golang.org/pkg/os/#ExpandEnv
[environment]: http://docs.drone.io/environment/

## Using "extra" `kubectl` versions

### tl;dr

To run `drone-sops-gke` using a different version of `kubectl` than the default, set `kubectl-version` to the version you'd like to use.

For example, to use the **1.14** version of `kubectl`:

```yml
image: docker.pliro.enlyze.com/enlyze/drone-sops-gke
settings:
  kubectl_version: "1.14"
```

This will configure the plugin to execute `/google-cloud-sdk/bin/kubectl.1.14` instead of `/google-cloud-sdk/bin/kubectl` for all `kubectl` commands.

### Background

Beginning with the [`237.0.0 (2019-03-05)` release of the `gcloud` SDK](https://cloud.google.com/sdk/docs/release-notes#23700_2019-03-05), "extra" `kubectl` versions are installed automatically when `kubectl` is installed via `gcloud components install kubectl`.

These "extra" versions are installed alongside the SDK's default `kubectl` version at `/google-cloud-sdk/bin` and are named using the following pattern:

```sh
kubectl.$clientVersionMajor.$clientVersionMinor
```

To list all of the "extra" `kubectl` versions available within a particular version of `drone-sops-gke`, you can run the following:

```sh
# list "extra" kubectl versions available with docker.pliro.enlyze.com/enlyze/drone-sops-gke
docker run \
  --rm \
  --interactive \
  --tty \
  --entrypoint '' \
  docker.pliro.enlyze.com/enlyze/drone-sops-gke list-extra-kubectl-versions
```

## Example reference usage

Note particularly the `gke:` build step.
```yml
---
kind: pipeline
type: docker
name: default

steps:
  - name: build
    image: golang:1.14
    environment:
      - GOPATH=/drone
      - CGO_ENABLED=0
    commands:
      - go get -t
      - go test -v -cover
      - go build -v -a
    when:
      event:
        - push
        - pull_request

  - name: gcr
    image: plugins/gcr
    registry: us.gcr.io
    repo: us.gcr.io/my-gke-project/my-app
    tag: ${DRONE_COMMIT}
    secrets: [google_credentials]
    when:
      event: push
      branch: master

  - name: gke
    image: docker.pliro.enlyze.com/enlyze/drone-sops-gke
    environment:
      GOOGLE_APPLICATION_CREDENTIALS_JSON:
        from_secret: GOOGLE_CREDENTIALS
    settings:
      cluster: my-gke-cluster
      expand_env_vars: true
      namespace: ${DRONE_BRANCH}
      zone: us-central1-a
      vars:
        app: my-app
        env: dev
        image: gcr.io/my-gke-project/my-app:${DRONE_COMMIT}
        user: $${USER}
    when:
      event: push
      branch: master
```
