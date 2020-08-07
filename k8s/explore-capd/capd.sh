#!/usr/bin/env bash

set -e

IMG_REGISTRY=danielguo/k8s-staging-capi-docker
MGMT_CLUSTER_NAME=capd-e2e-mgmt
WORKLOAD_CLUSTER_NAME=capd-e2e-wl
WORKLOAD_CLUSTER_KUBECONFIG=${WORKLOAD_CLUSTER_NAME}.kubeconfig
WORKSPACE="${PWD}"

INFRA_DOCKER_VERSION=v0.3.0

K8S_SIGS_REPO="$GOPATH"/src/github.com/kubernetes-sigs
CAPI_REPO="$K8S_SIGS_REPO"/cluster-api

USAGE="$(
  cat <<EOF
usage: ${0} [FLAGS]
  Create a management cluster + workload cluster environment using Kind
  and the Cluster API providre for Docker (CAPD).
  WARNING: If $HOME/.cluster-api/ exists, the content will be replaced.

FLAGS
  -h    show this help and exit
  -u    deploy one management cluster and one workload cluster.
  -d    destroy all CAPD clusters including the kind management cluster.

Examples
  Create e2e environment
        bash capd.sh -u
  Destroys all CAPD clusters including the kind management cluster
        bash capd.sh -d
EOF
)"

function kubectl_mgmt_cluster() { kubectl --context "kind-${MGMT_CLUSTER_NAME}" "${@}"; }

function base64d() { base64 -D 2>/dev/null || base64 -d; }

function healthCheck() {
  if ! go version; then
    echo "go is required"
    exit 1
  fi

  if [[ ! -d $GOPATH ]]; then
    echo "GOPATH is not set"
    exit 1
  fi

  if ! kind version; then
    echo "kind is not installed"
    exit 1
  fi

  if ! kubectl version --client; then
    echo "kubectl is required"
    exit 1
  fi

  if ! kustomize version; then
    echo "kustomize is required"
    exit 1
  fi

  if ! python --version 2>&1 =~ 2\.7; then
    echo "python2 is required"
    exit 1
  fi
}

function prepare() {
  # checkout cluster api repo
  if [[ ! -d ${CAPI_REPO} ]]; then
    mkdir -p "$K8S_SIGS_REPO"
    cd "$K8S_SIGS_REPO" || exit
    git clone git@github.com:kubernetes-sigs/cluster-api.git
  fi
  cd "$CAPI_REPO" || exit
  git reset --hard HEAD
  git checkout master
  git pull --rebase
  set +e
  git branch -D v0.3.7
  set -e
  git checkout tags/v0.3.7 -b v0.3.7

  # build clusterctl binary
  make clusterctl
  cp bin/clusterctl "${WORKSPACE}"

  # prepare clusterctl-settings.json
  cat > "$CAPI_REPO"/clusterctl-settings.json << EOF
{
  "providers": ["cluster-api","bootstrap-kubeadm","control-plane-kubeadm", "infrastructure-docker"],
  "provider_repos": ["${CAPI_REPO}/test/infrastructure/docker"]
}
EOF
  if [[ ! -f "$CAPI_REPO"/clusterctl-settings.json ]]; then
    echo "$CAPI_REPO/clusterctl-settings.json does not exist"
    exit
  fi
  cat > "$CAPI_REPO"/test/infrastructure/docker/clusterctl-settings.json << EOF
{
  "name": "infrastructure-docker",
  "config": {
    "componentsFile": "infrastructure-components.yaml",
    "nextVersion": "${INFRA_DOCKER_VERSION}"
  }
}
EOF
  if [[ ! -f "$CAPI_REPO"/test/infrastructure/docker/clusterctl-settings.json ]]; then
    echo "$CAPI_REPO/test/infrastructure/docker/clusterctl-settings.json does not exist"
    exit
  fi

  # build images
  make -C "${CAPI_REPO}"/test/infrastructure/docker docker-build REGISTRY="$IMG_REGISTRY"
  make -C "${CAPI_REPO}"/test/infrastructure/docker generate-manifests REGISTRY="$IMG_REGISTRY"

  # run local overlap hack
  "${CAPI_REPO}"/cmd/clusterctl/hack/local-overrides.py

  cat > "$HOME"/.cluster-api/clusterctl.yaml << EOF
providers:
  - name: docker
    url: $HOME/.cluster-api/overrides/infrastructure-docker/${INFRA_DOCKER_VERSION}/infrastructure-components.yaml
    type: InfrastructureProvider
EOF
  if [[ ! -f $HOME/.cluster-api/clusterctl.yaml ]]; then
    echo "$HOME/.cluster-api/clusterctl.yaml does not exist"
    exit
  fi
}

function create_mgmt_cluster() {
  # prepare kind cluster yaml
  cat > "${WORKSPACE}"/kind-cluster-with-extramounts.yaml << EOF
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
nodes:
  - role: control-plane
    extraMounts:
      - hostPath: /var/run/docker.sock
        containerPath: /var/run/docker.sock
EOF

  export KIND_EXPERIMENTAL_DOCKER_NETWORK=bridge  # for kind v0.8.x

  kind create cluster --name "$MGMT_CLUSTER_NAME" --image=kindest/node:v1.18.0 --config "${WORKSPACE}"/kind-cluster-with-extramounts.yaml
  kind load docker-image "$IMG_REGISTRY"/capd-manager-amd64:dev --name "$MGMT_CLUSTER_NAME"

  "${WORKSPACE}"/clusterctl init --core cluster-api:v0.3.0 --bootstrap kubeadm:v0.3.0 --control-plane kubeadm:v0.3.0 --infrastructure docker:${INFRA_DOCKER_VERSION} -v10

  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capd-controller-manager -n capd-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-controller-manager -n capi-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-kubeadm-bootstrap-controller-manager -n capi-webhook-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-kubeadm-control-plane-controller-manager -n capi-kubeadm-control-plane-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-controller-manager -n capi-webhook-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-kubeadm-bootstrap-controller-manager -n capi-webhook-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-kubeadm-control-plane-controller-manager -n capi-webhook-system
}

function create_workload_cluster() {
  cat ${WORKSPACE}/simple-cluster-with-md.yaml | sed -e 's~CLUSTER_NAME~'"${WORKLOAD_CLUSTER_NAME}"'~g' | kubectl_mgmt_cluster apply -f -

  while ! kubectl_mgmt_cluster get secret "$WORKLOAD_CLUSTER_NAME"-kubeconfig; do
    sleep 5
    echo "Secret is not ready. Sleeping 5s..."
  done

  kubectl_mgmt_cluster get secret "$WORKLOAD_CLUSTER_NAME"-kubeconfig -o jsonpath='{.data.value}' | base64d > "${WORKSPACE}/${WORKLOAD_CLUSTER_KUBECONFIG}"

  if [[ $(uname) = "Darwin" ]]; then
    echo "Modifying the kubeconfig for Mac OS"
    # Point the kubeconfig to the exposed port of the load balancer, rather than the inaccessible container IP.
    sed -i -e "s/server:.*/server: https:\/\/$(docker port "$WORKLOAD_CLUSTER_NAME"-lb 6443/tcp | sed "s/0.0.0.0/127.0.0.1/")/g" "${WORKSPACE}/${WORKLOAD_CLUSTER_KUBECONFIG}"
    # Ignore the CA, because it is not signed for 127.0.0.1
    sed -i -e "s/certificate-authority-data:.*/insecure-skip-tls-verify: true/g" "${WORKSPACE}/${WORKLOAD_CLUSTER_KUBECONFIG}"
  fi

  workload_cluster_kubectl="kubectl --kubeconfig=${WORKSPACE}/${WORKLOAD_CLUSTER_KUBECONFIG}"

  while ! ${workload_cluster_kubectl} get nodes; do
    sleep 5
    echo "Node resource is not available yet. Sleeping 5s..."
  done

  # deploy CNI
  ${workload_cluster_kubectl} apply -f https://docs.projectcalico.org/v3.15/manifests/calico.yaml

  # wait for node to be ready
  # TODO: Might need to add a timeout here
  for node in $(${workload_cluster_kubectl} get nodes -o jsonpath='{.items[].metadata.name}'); do
    ${workload_cluster_kubectl} wait --for=condition=Ready --timeout=300s node/"${node}"
  done

  echo "The workload cluster ${WORKLOAD_CLUSTER_NAME} has been created successfully"
}

function deleteAllClusters() {
    kind delete cluster --name ${MGMT_CLUSTER_NAME}
    kind delete cluster --name ${WORKLOAD_CLUSTER_NAME}
    rm -rf "${WORKSPACE}"/kind-cluster-with-extramounts.yaml
    rm -rf "${WORKSPACE}"/clusterctl
    rm -rf "${WORKSPACE}/${WORKLOAD_CLUSTER_KUBECONFIG}"*
}

while getopts ":hud" opt; do
  case ${opt} in
    h)
      echo "${USAGE}" && exit 1
      ;;
    u)
      healthCheck
      deleteAllClusters
      prepare
      create_mgmt_cluster
      create_workload_cluster
      ;;
    d)
      deleteAllClusters
      ;;
    \?)
      echo "invalid option: -${OPTARG} ${USAGE}" && exit 1
      ;;
  esac
done