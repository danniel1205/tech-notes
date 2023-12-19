# Explore Helm

## Tasks

- Create a sample helm chart (Helm official [doc](https://helm.sh/docs/chart_template_guide/getting_started/))
  - A sample http server as the app
- Deploy into a kind cluster

## Steps

- Bootstrap a kind cluster

  ``` bash
  kind create cluster
  ```

- Build the sample http server image and load into kind cluster

  ``` bash
  cd sample-http-server
  docker build --no-cache -f Dockerfile -t sample-http-server --rm=true .
  kind load docker-image sample-http-server
  ```

- Using helm command to deploy the sample-http-server app

  ``` bash
  cd helm-charts
  helm install sample-http-server ./sample-http-server -f values-overlay.yaml

  kubectl get deploy -A
  NAMESPACE            NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
  default              sample-http-server-overwritten   1/1     1            1           12s
  kube-system          coredns                          2/2     2            2           10m
  local-path-storage   local-path-provisioner           1/1     1            1           10m
  ```
