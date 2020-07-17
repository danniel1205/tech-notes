#!/usr/bin/env bash

set -x

IMG_REGISTRY=danielguo/k8s-staging-capi-docker
MGMT_CLUSTER_NAME=kind-mgmt-cluster
WORKLOAD_CLUSTER_NAME=kind-workload-cluster
WORKLOAD_CLUSTER_KUBECONFIG=kind-workload-cluster.kubeconfig
WORKSPACE="${PWD}"

K8S_SIGS_REPO="$GOPATH"/src/github.com/kubernetes-sigs
CAPI_REPO="$K8S_SIGS_REPO"/cluster-api

function prepare() {
  # checkout cluster api repo
  mkdir -p "$K8S_SIGS_REPO"
  cd "$K8S_SIGS_REPO" || exit
  git clone git@github.com:kubernetes-sigs/cluster-api.git
  cd "$CAPI_REPO" || exit
  git checkout tags/v0.3.7 -b v0.3.7

  # build clusterctl binary
  make clusterctl
  cp bin/clusterctl "${WORKSPACE}"

  # prepare clusterctl-settings.json
  cat > "$CAPI_REPO"/clusterctl-settings.json << EOF
{
  "providers": ["cluster-api","bootstrap-kubeadm","control-plane-kubeadm", "infrastructure-docker"],
  "provider_repos": ["$CAPI_REPO/test/infrastructure/docker"]
}
EOF

  cat > "$CAPI_REPO"/test/infrastructure/docker/clusterctl-settings.json << EOF
{
  "name": "infrastructure-docker",
  "config": {
    "componentsFile": "infrastructure-components.yaml",
    "nextVersion": "v0.3.0"
  }
}
EOF

  # build images
  make -C test/infrastructure/docker docker-build REGISTRY="$IMG_REGISTRY"
  make -C test/infrastructure/docker generate-manifests REGISTRY="$IMG_REGISTRY"

  # run local overlap hack
  cmd/clusterctl/hack/local-overrides.py
}

function kubectl_mgmt_cluster() { kubectl --context "kind-${MGMT_CLUSTER_NAME}" "${@}"; }

function base64d() { base64 -D 2>/dev/null || base64 -d; }

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

#  export KIND_EXPERIMENTAL_DOCKER_NETWORK=bridge  # Skip this step if kind v0.7

  kind create cluster --name "$MGMT_CLUSTER_NAME" --image=kindest/node:v1.18.0 --config "${WORKSPACE}"/kind-cluster-with-extramounts.yaml
  kind load docker-image "$IMG_REGISTRY"/capd-manager-amd64:dev --name "$MGMT_CLUSTER_NAME"

  "${WORKSPACE}"/clusterctl init --core cluster-api:v0.3.0 --bootstrap kubeadm:v0.3.0 --control-plane kubeadm:v0.3.0 --infrastructure docker:v0.3.0 -v10

  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capd-controller-manager -n capd-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-controller-manager -n capi-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-kubeadm-bootstrap-controller-manager -n capi-webhook-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-kubeadm-control-plane-controller-manager -n capi-kubeadm-control-plane-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-controller-manager -n capi-webhook-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-kubeadm-bootstrap-controller-manager -n capi-webhook-system
  kubectl_mgmt_cluster wait --for=condition=Available --timeout=300s deployment/capi-kubeadm-control-plane-controller-manager -n capi-webhook-system
}

function create_workload_cluster() {
  "${WORKSPACE}"/clusterctl config cluster "$WORKLOAD_CLUSTER_NAME" --kubernetes-version="v1.18.0" --control-plane-machine-count=1 --worker-machine-count=3 | kubectl apply -f -

  while ! kubectl_mgmt_cluster get secret "$WORKLOAD_CLUSTER_NAME"-kubeconfig; do
    sleep 5
    echo "Secret is not ready. Sleeping 5s..."
  done

  kubectl_mgmt_cluster get secret "$WORKLOAD_CLUSTER_NAME"-kubeconfig -o jsonpath='{.data.value}' | base64d > "${WORKSPACE}/${WORKLOAD_CLUSTER_KUBECONFIG}"

  ls -lah "${WORKSPACE}/${WORKLOAD_CLUSTER_KUBECONFIG}"

  # Point the kubeconfig to the exposed port of the load balancer, rather than the inaccessible container IP.
  sed -i -e "s/server:.*/server: https:\/\/$(docker port "$WORKLOAD_CLUSTER_NAME"-lb 6443/tcp | sed "s/0.0.0.0/127.0.0.1/")/g" "${WORKSPACE}/${WORKLOAD_CLUSTER_KUBECONFIG}"
  # Ignore the CA, because it is not signed for 127.0.0.1
  sed -i -e "s/certificate-authority-data:.*/insecure-skip-tls-verify: true/g" "${WORKSPACE}/${WORKLOAD_CLUSTER_KUBECONFIG}"

  workload_cluster_kubectl="kubectl --kubeconfig=${WORKSPACE}/${WORKLOAD_CLUSTER_KUBECONFIG}"

  while ! ${workload_cluster_kubectl} get nodes; do
    sleep 5
    echo "Node resource is not available yet. Sleeping 5s..."
  done

  # deploy CNI
  ${workload_cluster_kubectl} apply -f https://docs.projectcalico.org/v3.15/manifests/calico.yaml

  # wait for node to be ready
  for node in $(${workload_cluster_kubectl} get nodes -o jsonpath='{.items[].metadata.name}'); do
    ${workload_cluster_kubectl} wait --for=condition=Ready --timeout=300s node/"${node}"
  done

}

prepare
create_mgmt_cluster
create_workload_cluster