#!/usr/bin/env bash

set -e

PROJECT_ROOT=$(pwd)
# CAPD image registry, this script will build CAPD images and load into KIND cluster
CAPD_IMG_REGISTRY=docker.io/danielguo/k8s-staging-cluster-api
# Management cluster name
MGMT_CLUSTER_NAME=capd-mgmt
# Management cluster namespace
MGMT_CLUSTER_NAMESPACE=capd-mgmt-ns
# Workload cluster name
WORKLOAD_CLUSTER_NAME=capd-wl
# Workload cluster namespace
WORKLOAD_CLUSTER_NAMESPACE=capd-wl-ns
# Workload cluster machine deployment replicas
WORKLOAD_CLUSTER_MD_REPLICAS=3
# Workload cluster kubeconfig file name
WORKLOAD_CLUSTER_KUBECONFIG=${WORKLOAD_CLUSTER_NAME}.kubeconfig
WORKSPACE="${PROJECT_ROOT}/workspace"

# CAPD infra version, it is v0.3.99 by default
INFRA_VERSION=v0.3.99
# CAPI release, used to pull a particular tag from Github
CAPI_RELEASE=v0.3.17

K8S_SIGS_REPO="$GOPATH"/src/github.com/kubernetes-sigs
CAPI_REPO="$K8S_SIGS_REPO"/cluster-api
PROVIDER_REPO="${CAPI_REPO}"/test/infrastructure/docker

USAGE="$(
  cat <<EOF
usage: ${0} [FLAGS]
  Create a management cluster + workload cluster environment using Kind
  and the Cluster API providre for Docker (CAPD).
  WARNING: If $HOME/.cluster-api/ exists, the content will be replaced.
FLAGS
  -h    show this help and exit
  -p    get all prerequisites ready.
  -m    deploy one management cluster.
  -w    deploy one workload cluster.
  -d    destroy all CAPD clusters including the kind management cluster.
Examples
  Get all prerequisites ready
        bash capd.sh -p
  Create a management cluster
        bash capd.sh -m
  Create a workload cluster
        bash capd.sh -w
  Destroys all CAPD clusters including the kind management cluster
        bash e2e.sh -d
EOF
)"

function kubectl_mgmt_cluster() { kubectl --context "kind-${MGMT_CLUSTER_NAME}" "${@}"; }

function base64d() { base64 -D 2>/dev/null || base64 -d; }

function base64e() { base64 -w0 2>/dev/null || base64; }

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
  # create workspace if it does not exist
  mkdir -p "${WORKSPACE}"

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
  git branch -D "${CAPI_RELEASE}"
  set -e
  git checkout tags/"${CAPI_RELEASE}" -b "${CAPI_RELEASE}"

  # build clusterctl binary
  make clusterctl
  cp bin/clusterctl "${WORKSPACE}"

  # build artifacts locally
  export PULL_POLICY=IfNotPresent
  make docker-build REGISTRY="${CAPD_IMG_REGISTRY}"
  make generate-manifests REGISTRY="${CAPD_IMG_REGISTRY}"
  make -C "${PROVIDER_REPO}" docker-build REGISTRY="${CAPD_IMG_REGISTRY}"
  make -C "${PROVIDER_REPO}" generate-manifests REGISTRY="${CAPD_IMG_REGISTRY}"

  # prepare clusterctl-settings.json under capi repo
  cat > "$CAPI_REPO"/clusterctl-settings.json << EOF
{
  "providers": ["cluster-api","bootstrap-kubeadm","control-plane-kubeadm", "infrastructure-docker"],
  "provider_repos": ["${PROVIDER_REPO}"]
}
EOF
  if [[ ! -f "${CAPI_REPO}"/clusterctl-settings.json ]]; then
    echo "${CAPI_REPO}/clusterctl-settings.json does not exist"
    exit
  fi

# prepare clusterctl-settings.json under provider repo
# this is for capd only, other providers have this file exist already
  cat > "${PROVIDER_REPO}"/clusterctl-settings.json << EOF
{
  "name": "infrastructure-docker",
  "config": {
    "componentsFile": "infrastructure-components.yaml",
    "nextVersion": "${INFRA_VERSION}"
  }
}
EOF
  if [[ ! -f "${PROVIDER_REPO}"/clusterctl-settings.json ]]; then
    echo "${PROVIDER_REPO}/clusterctl-settings.json does not exist"
    exit
  fi

  # copy metadata.yaml into docker provider repo
  cp "${CAPI_REPO}"/metadata.yaml "${PROVIDER_REPO}"

  # run local overlap hack
  "${CAPI_REPO}"/cmd/clusterctl/hack/create-local-repository.py

#   cat > "$HOME"/.cluster-api/clusterctl.yaml << EOF
# providers:
#   - name: docker
#     url: $HOME/.cluster-api/overrides/infrastructure-docker/${INFRA_VERSION}/infrastructure-components.yaml
#     type: InfrastructureProvider
# EOF
#   if [[ ! -f $HOME/.cluster-api/clusterctl.yaml ]]; then
#     echo "$HOME/.cluster-api/clusterctl.yaml does not exist"
#     exit
#   fi
}

function create_mgmt_cluster() {
  # prepare kind cluster yaml
  cat > "${WORKSPACE}"/kind-cluster-with-extramounts.yaml << EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
    - hostPath: /var/run/docker.sock
      containerPath: /var/run/docker.sock
EOF

  # export KIND_EXPERIMENTAL_DOCKER_NETWORK=bridge  # for kind v0.8.x

  # create kind cluster
  kind create cluster --name "${MGMT_CLUSTER_NAME}" --image=kindest/node:v1.20.2 --config "${WORKSPACE}"/kind-cluster-with-extramounts.yaml
  # load image into kind cluster
  kind load docker-image "${CAPD_IMG_REGISTRY}"/cluster-api-controller-amd64:dev --name "${MGMT_CLUSTER_NAME}"
  kind load docker-image "${CAPD_IMG_REGISTRY}"/kubeadm-bootstrap-controller-amd64:dev --name "${MGMT_CLUSTER_NAME}"
  kind load docker-image "${CAPD_IMG_REGISTRY}"/kubeadm-control-plane-controller-amd64:dev --name "${MGMT_CLUSTER_NAME}"
  kind load docker-image "${CAPD_IMG_REGISTRY}"/capd-manager-amd64:dev --name "${MGMT_CLUSTER_NAME}"

  echo -e "Initiating the management cluster... \U00023f3"
  "${WORKSPACE}"/clusterctl init \
    --core cluster-api:"${INFRA_VERSION}" \
    --bootstrap kubeadm:"${INFRA_VERSION}" \
    --control-plane kubeadm:"${INFRA_VERSION}" \
    --infrastructure docker:"${INFRA_VERSION}" \
    --config ~/.cluster-api/dev-repository/config.yaml

  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capd-controller-manager -n capd-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-controller-manager -n capi-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-kubeadm-bootstrap-controller-manager -n capi-webhook-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-kubeadm-control-plane-controller-manager -n capi-kubeadm-control-plane-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-controller-manager -n capi-webhook-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-kubeadm-bootstrap-controller-manager -n capi-webhook-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-kubeadm-control-plane-controller-manager -n capi-webhook-system

  kubectl_mgmt_cluster create ns ${MGMT_CLUSTER_NAMESPACE}
  # The CAPD management cluster created by using "clusterctl init" does not have the cluster resources for itself:
  # [Cluster, KubeadmControlPlane, MachineDeployment]. You might want to create above dummy resources in the mgmt cluster.
#   cat <<EOF | kubectl apply -f -
# apiVersion: cluster.x-k8s.io/v1alpha3
# kind: Cluster
# metadata:
#   name: ${MGMT_CLUSTER_NAME}
#   namespace: ${MGMT_CLUSTER_NAMESPACE}
# spec:
#   clusterNetwork:
#     services:
#       cidrBlocks: ["10.96.0.0/12"]
#     pods:
#       cidrBlocks: ["192.168.0.0/16"]
#     serviceDomain: "cluster.local"
#   controlPlaneRef:
#     apiVersion: controlplane.cluster.x-k8s.io/v1alpha3
#     kind: KubeadmControlPlane
#     name: ${MGMT_CLUSTER_NAME}-controlplane
#     namespace: ${MGMT_CLUSTER_NAMESPACE}
#   infrastructureRef:
#     apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
#     kind: DockerCluster
#     name: ${MGMT_CLUSTER_NAME}
#     namespace: ${MGMT_CLUSTER_NAMESPACE}
# ---
# apiVersion: "controlplane.cluster.x-k8s.io/v1alpha3"
# kind: KubeadmControlPlane
# metadata:
#   name: ${MGMT_CLUSTER_NAME}-controlplane
#   namespace: ${MGMT_CLUSTER_NAMESPACE}
# spec:
#   replicas: 1
#   version: "v1.18.6"
#   infrastructureTemplate:
#     apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
#     kind: DockerMachineTemplate
#     name: ${MGMT_CLUSTER_NAME}-controlplane
#     namespace: ${MGMT_CLUSTER_NAMESPACE}
#   kubeadmConfigSpec:
#     clusterConfiguration:
#       controllerManager:
#         extraArgs:
#           enable-hostpath-provisioner: "true"
#     initConfiguration:
#       nodeRegistration:
#         kubeletExtraArgs:
#           eviction-hard: nodefs.available<0%,nodefs.inodesFree<0%,imagefs.available<0%
# ---
# apiVersion: cluster.x-k8s.io/v1alpha3
# kind: MachineDeployment
# metadata:
#   labels:
#     cluster.x-k8s.io/cluster-name: ${MGMT_CLUSTER_NAME}
#   name: ${MGMT_CLUSTER_NAME}-md-0
#   namespace: ${MGMT_CLUSTER_NAMESPACE}
# spec:
#   clusterName: ${MGMT_CLUSTER_NAME}
#   replicas: 1
#   selector:
#     matchLabels:
#       cluster.x-k8s.io/cluster-name: ${MGMT_CLUSTER_NAME}
#   template:
#     metadata:
#       labels:
#         cluster.x-k8s.io/cluster-name: ${MGMT_CLUSTER_NAME}
#     spec:
#       bootstrap:
#         configRef:
#           apiVersion: bootstrap.cluster.x-k8s.io/v1alpha3
#           kind: KubeadmConfigTemplate
#           name: ${MGMT_CLUSTER_NAME}-md-0
#       clusterName: ${MGMT_CLUSTER_NAME}
#       infrastructureRef:
#         apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
#         kind: DockerMachineTemplate
#         name: ${MGMT_CLUSTER_NAME}-worker
#         namespace: ${MGMT_CLUSTER_NAMESPACE}
#       version: 'v1.18.6'
# EOF

  # Get the kubeconfig from control plane node and updates the server and change it to insecure-skip-tls-verify
  docker exec ${MGMT_CLUSTER_NAME}-control-plane cat /etc/kubernetes/admin.conf > "${WORKSPACE}/value"
  if [[ $(uname) = "Darwin" ]]; then
    # Point the kubeconfig to the exposed port of the load balancer, rather than the inaccessible container IP.
    sed -i -e "s/server:.*/server: https:\/\/$(docker port "${MGMT_CLUSTER_NAME}-control-plane" 6443/tcp | sed "s/0.0.0.0/127.0.0.1/")/g" "${WORKSPACE}/value"
    # Ignore the CA, because it is not signed for 127.0.0.1
    sed -i -e "s/certificate-authority-data:.*/insecure-skip-tls-verify: true/g" "${WORKSPACE}/value"
  fi
  kubectl_mgmt_cluster create secret generic "${MGMT_CLUSTER_NAME}-kubeconfig" --namespace ${MGMT_CLUSTER_NAMESPACE} --from-file="${WORKSPACE}/value"

  rm -rf "${WORKSPACE}/value"*

  echo -e "The management cluster is created successfully! \U0001f601"
}

function create_workload_cluster() {
  echo -e "Creating the workload cluster... \U00023f3"

  kubectl_mgmt_cluster create ns ${WORKLOAD_CLUSTER_NAMESPACE}
  "${WORKSPACE}"/clusterctl config cluster ${WORKLOAD_CLUSTER_NAME} \
    --target-namespace=${WORKLOAD_CLUSTER_NAMESPACE} --flavor development \
    --kubernetes-version v1.19.11 \
    --control-plane-machine-count=1 \
    --worker-machine-count=${WORKLOAD_CLUSTER_MD_REPLICAS} \
    --config ~/.cluster-api/dev-repository/config.yaml | kubectl_mgmt_cluster apply -f -

  while ! kubectl_mgmt_cluster get secret "${WORKLOAD_CLUSTER_NAME}"-kubeconfig -n ${WORKLOAD_CLUSTER_NAMESPACE}; do
    sleep 5
    echo -e "Secret is not ready. Sleeping 5s... \U00023f3"
  done

  kubectl_mgmt_cluster get secret "$WORKLOAD_CLUSTER_NAME"-kubeconfig -n ${WORKLOAD_CLUSTER_NAMESPACE} -o jsonpath='{.data.value}' | base64d > "${WORKSPACE}/${WORKLOAD_CLUSTER_KUBECONFIG}"

  if [[ $(uname) = "Darwin" ]]; then
    echo -e "Modifying the kubeconfig for Mac OS... \U00023f3"
    # Point the kubeconfig to the exposed port of the load balancer, rather than the inaccessible container IP.
    sed -i -e "s/server:.*/server: https:\/\/$(docker port "${WORKLOAD_CLUSTER_NAME}"-lb 6443/tcp | sed "s/0.0.0.0/127.0.0.1/")/g" "${WORKSPACE}/${WORKLOAD_CLUSTER_KUBECONFIG}"
    # Ignore the CA, because it is not signed for 127.0.0.1
    sed -i -e "s/certificate-authority-data:.*/insecure-skip-tls-verify: true/g" "${WORKSPACE}/${WORKLOAD_CLUSTER_KUBECONFIG}"
    echo -e "Modified the kubeconfig for Mac OS... \U0002705"
  fi

  workload_cluster_kubectl="kubectl --kubeconfig=${WORKSPACE}/${WORKLOAD_CLUSTER_KUBECONFIG}"

  while ! ${workload_cluster_kubectl} get nodes; do
    sleep 5
    echo -e "Node resource is not available yet. Sleeping 5s... \U00023f3"
  done

  # deploy CNI
  echo -e "Deploying Calico in the workload cluster... \U00023f3"
  ${workload_cluster_kubectl} apply -f https://docs.projectcalico.org/v3.18/manifests/calico.yaml

  # wait for node to be ready
  # TODO: Might need to add a timeout here
  for node in $(${workload_cluster_kubectl} get nodes -o jsonpath='{.items[].metadata.name}'); do
    ${workload_cluster_kubectl} wait --for=condition=Ready --timeout=300s node/"${node}"
  done

  echo -e "The workload cluster ${WORKLOAD_CLUSTER_NAME} has been created successfully! \U0001f601"
}

function deleteAllClusters() {
    kind delete cluster --name ${MGMT_CLUSTER_NAME}
    kind delete cluster --name ${WORKLOAD_CLUSTER_NAME}
    rm -rf "${WORKSPACE}"/kind-cluster-with-extramounts.yaml
    rm -rf "${WORKSPACE:?}/${WORKLOAD_CLUSTER_KUBECONFIG}"*
}

while getopts ":hpmdw" opt; do
  case ${opt} in
    h)
      echo "${USAGE}" && exit 1
      ;;
    p)
      healthCheck
      deleteAllClusters
      prepare
      ;;
    m)
      create_mgmt_cluster
      ;;
    w)
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