---
tags: cluster-api
---
# Explore cluster API for docker infrastructure

## Official page

<https://cluster-api.sigs.k8s.io/user/quick-start.html>

## Steps

### Clone clusterAPI github repo

<https://github.com/kubernetes-sigs/cluster-api>

### Install kubectl

<https://kubernetes.io/docs/tasks/tools/install-kubectl/>

### Install kind

<https://github.com/kubernetes-sigs/kind>

### Install clusterctl

<https://cluster-api.sigs.k8s.io/user/quick-start.html#install-clusterctl>

### Install kustomize

<https://kubernetes-sigs.github.io/kustomize/installation/>

### Generate the infrastructure components yaml

``` bash
kustomize build ~/go/src/github.com/cluster-apitest/infrastructure/docker/config > ~/tech-explore/capd/infrastructure-docker/infrastructore-components.yaml
```

### Create images and manifests in order to use the docker provider

<https://cluster-api.sigs.k8s.io/clusterctl/developers.html#additional-steps-in-order-to-use-the-docker-provider>

``` bash
cd ~/go/src/github.com/cluster-api
```

``` bash
make -C test/infrastructure/docker docker-build REGISTRY=danielguo/k8s-
staging-capi-docker
```

``` bash
make -C test/infrastructure/docker generate-manifests REGISTRY=danielguo/k8s-staging-capi-docker
```

### Create kind cluster

``` bash
cat > kind-cluster-with-extramounts.yaml <<EOF
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
nodes:
  - role: control-plane
    extraMounts:
      - hostPath: /var/run/docker.sock
        containerPath: /var/run/docker.sock
EOF

kind create cluster --config ./kind-cluster-with-extramounts.yaml
kind load docker-image danielguo/k8s-staging-capi-docker/capd-manager-amd64:dev

```

### Additional steps in order to use the docker provider

``` bash
cd ~/go/src/github.com/cluster-api
```

``` bash
# https://cluster-api.sigs.k8s.io/clusterctl/developers.html#create-a-clusterctl-settingsjson-file
cat > cluster-settings.json << EOF
{
  "providers": ["cluster-api","bootstrap-kubeadm","control-plane-kubeadm", "infrastructure-docker"],
  "provider_repos": ["./test/infrastructure/docker"]
}
EOF
```

``` bash
#https://cluster-api.sigs.k8s.io/clusterctl/developers.html#available-providers
cat > ./test/infrastructure/docker/cluster-settings.json << EOF
{
  "name": "infrastructure-docker",
  "config": {
    "componentsFile": "infrastructure-components.yaml",
    "nextVersion": "v0.3.0"
  }
}
EOF
```

``` bash
#https://cluster-api.sigs.k8s.io/clusterctl/developers.html#run-the-local-overrides-hack
cmd/clusterctl/hack/local-overrides.py

Output:
--> clusterctl local overrides generated from local repositories for the cluster-api, bootstrap-kubeadm, control-plane-kubeadm, infrastructure-docker providers.
in order to use them, please run:
--> clusterctl init --core cluster-api:v0.3.0 --bootstrap kubeadm:v0.3.0 --control-plane kubeadm:v0.3.0 --infrastructure docker:v0.3.0
--> please check the documentation for additional steps required for using the docker provider

```

### Initialize the management cluster

``` bash
clusterctl init -i docker -v10
Fetching providers
Installing cert-manager
Waiting for cert-manager to be available...
Installing Provider="cluster-api" Version="v0.3.0" TargetNamespace="capi-system"
Installing Provider="bootstrap-kubeadm" Version="v0.3.0" TargetNamespace="capi-kubeadm-bootstrap-system"
Installing Provider="control-plane-kubeadm" Version="v0.3.0" TargetNamespace="capi-kubeadm-control-plane-system"
Installing Provider="infrastructure-docker" Version="v0.3.0" TargetNamespace="capd-system"

Your management cluster has been initialized successfully!

You can now create your first workload cluster by running the following:

  clusterctl config cluster [name] --kubernetes-version [version] | kubectl apply -f -
```

Now you should have all the cluster API components up and running

``` bash
kubectl get pods -A
NAMESPACE                           NAME                                                             READY   STATUS    RESTARTS   AGE
capd-system                         capd-controller-manager-b765fdf4b-qbmpp                          2/2     Running   0          42s
capi-kubeadm-bootstrap-system       capi-kubeadm-bootstrap-controller-manager-7d5bf99fcd-9fvv7       2/2     Running   0          44s
capi-kubeadm-control-plane-system   capi-kubeadm-control-plane-controller-manager-5b78c89b66-t72bq   2/2     Running   0          43s
capi-system                         capi-controller-manager-548c58995f-gjx85                         2/2     Running   0          45s
capi-webhook-system                 capi-controller-manager-84f48bf9bd-xtrqh                         2/2     Running   0          45s
capi-webhook-system                 capi-kubeadm-bootstrap-controller-manager-545bbb995c-8nvq4       2/2     Running   0          44s
capi-webhook-system                 capi-kubeadm-control-plane-controller-manager-777fc96489-hldct   2/2     Running   0          44s
cert-manager                        cert-manager-69b4f77ffc-6ns6g                                    1/1     Running   0          70s
cert-manager                        cert-manager-cainjector-576978ffc8-klzdk                         1/1     Running   0          70s
cert-manager                        cert-manager-webhook-c67fbc858-th5zw                             1/1     Running   0          70s
kube-system                         coredns-6955765f44-bgskj                                         1/1     Running   0          22h
kube-system                         coredns-6955765f44-q9cg9                                         1/1     Running   0          22h
kube-system                         etcd-kind-control-plane                                          1/1     Running   0          22h
kube-system                         kindnet-wtbvn                                                    1/1     Running   0          22h
kube-system                         kube-apiserver-kind-control-plane                                1/1     Running   0          22h
kube-system                         kube-controller-manager-kind-control-plane                       1/1     Running   0          22h
kube-system                         kube-proxy-fnld8                                                 1/1     Running   0          22h
kube-system                         kube-scheduler-kind-control-plane                                1/1     Running   0          22h
local-path-storage                  local-path-provisioner-7745554f7f-lbzwz                          1/1     Running   0          22h
```

And your single controle plane management cluster is running

``` bash
kubectl get nodes
NAME                 STATUS   ROLES    AGE   VERSION
kind-control-plane   Ready    master   22h   v1.17.0
```

### Create a workload cluster

<https://cluster-api.sigs.k8s.io/user/quick-start.html#create-your-first-workload-cluster>
> The clusterctl config cluster command by default uses cluster templates which are provided by the infrastructure providers.

However, I could not find any `cluster-template.yaml` available under `.cluster-api/overrides/infrastructure-docker/v0.3.0/cluster-template.yaml`. The `cluterctl config cluster <cluster_name>` command will fail with error complaining about the missing `cluster-template.yaml` file

For now, I just copied the sample from https://github.com/vmware-tanzu/tgik/blob/master/episodes/110/capd-v3/infrastructure-docker/v0.3.2/cluster-template.yaml and put it under `.cluster-api/overrides/infrastructure-docker/v0.3.0/cluster-template.yaml`

Then run:

``` bash
clusterctl config cluster test > capi-test.yaml
# Modify the capi-test.yaml if necessary, e.g. the machineDeployment replica. It is 0 by default

kubectl apply -f capi-test.yaml
```

After apply the yaml, you should be able to see the following output

``` bash
~/tech-explore/capd  kubectl get cluster --all-namespaces

NAMESPACE   NAME   PHASE
default     test   Provisioned
 ~/tech-explore/capd  kubectl get kubeadmcontrolplane --all-namespaces

NAMESPACE   NAME   READY   INITIALIZED   REPLICAS   READY REPLICAS   UPDATED REPLICAS   UNAVAILABLE REPLICAS
default     test           true          1                           1                  1
 ~/tech-explore/capd  kubectl get machine -A
NAMESPACE   NAME         PROVIDERID                   PHASE
default     test-jgrph   docker:////test-test-jgrph   Running
 ~/tech-explore/capd  docker ps
CONTAINER ID        IMAGE                          COMMAND                  CREATED              STATUS              PORTS                                  NAMES
94d14d4b0cc9        kindest/node:v1.17.0           "/usr/local/bin/entr…"   About a minute ago   Up 58 seconds       36323/tcp, 127.0.0.1:36323->6443/tcp   test-test-jgrph
9dbd182555fb        kindest/haproxy:2.1.1-alpine   "/docker-entrypoint.…"   About a minute ago   Up About a minute   38337/tcp, 0.0.0.0:38337->6443/tcp     test-lb
bd6eaef3f13c        kindest/node:v1.17.0           "/usr/local/bin/entr…"   10 minutes ago       Up 10 minutes       127.0.0.1:32771->6443/tcp              kind-control-plane
```

(Optional) Just in case you want to scale your workload cluster machine deployment, you could do the following:

``` bash
~/tech-explore/capd  kubectl get machinedeployments -A
NAMESPACE   NAME        PHASE     REPLICAS   AVAILABLE   READY
default     test-md-0   Running
 ~/tech-explore/capd  kubectl scale machinedeployments test-md-0 --replicas=3
machinedeployment.cluster.x-k8s.io/test-md-0 scaled
 ~/tech-explore/capd  kubectl get machine -A
NAMESPACE   NAME                        PROVIDERID                                  PHASE
default     test-jgrph                  docker:////test-test-jgrph                  Running
default     test-md-0-db7cb7668-8mdbz   docker:////test-test-md-0-db7cb7668-8mdbz   Running
default     test-md-0-db7cb7668-dlrvg   docker:////test-test-md-0-db7cb7668-dlrvg   Running
default     test-md-0-db7cb7668-nlt4f   docker:////test-test-md-0-db7cb7668-nlt4f   Running
```

You might have notices that the kubeadmcontrolplane is not ready. That is because you have to deploy CNI. https://cluster-api.sigs.k8s.io/user/quick-start.html#deploy-a-cni-solution

### Get access to a workload cluster

<https://cluster-api.sigs.k8s.io/clusterctl/developers.html#connecting-to-a-workload-cluster-on-docker>

``` bash
kubectl --namespace=default get secret/test-kubeconfig -o jsonpath={.data.value} \
  | base64 --decode \
  > ./test.kubeconfig
```

(Optional) If you are using Mac, the following steps are required.

``` bash
# Point the kubeconfig to the exposed port of the load balancer, rather than the inaccessible container IP.
sed -i -e "s/server:.*/server: https:\/\/$(docker port test-lb 6443/tcp | sed "s/0.0.0.0/127.0.0.1/")/g" ./test.kubeconfig

# Ignore the CA, because it is not signed for 127.0.0.1
sed -i -e "s/certificate-authority-data:.*/insecure-skip-tls-verify: true/g" ./test.kubeconfig

```

### Deploy CNI

<https://cluster-api.sigs.k8s.io/clusterctl/developers.html#connecting-to-a-workload-cluster-on-docker>

### You are all set

``` bash
 ~/tech-explore/capd  kubectl --kubeconfig=./test.kubeconfig get pods -A
NAMESPACE     NAME                                      READY   STATUS    RESTARTS   AGE
kube-system   calico-kube-controllers-77c4b7448-dfpv2   1/1     Running   0          7m26s
kube-system   calico-node-2sljv                         1/1     Running   0          7m27s
kube-system   calico-node-cqrvs                         1/1     Running   0          7m27s
kube-system   calico-node-kbh9r                         1/1     Running   0          7m27s
kube-system   calico-node-nb26v                         1/1     Running   0          7m27s
kube-system   coredns-6955765f44-5p47h                  1/1     Running   0          51m
kube-system   coredns-6955765f44-dr4dk                  1/1     Running   0          51m
kube-system   etcd-test-test-jgrph                      1/1     Running   0          51m
kube-system   kube-apiserver-test-test-jgrph            1/1     Running   0          51m
kube-system   kube-controller-manager-test-test-jgrph   1/1     Running   0          51m
kube-system   kube-proxy-2wfxr                          1/1     Running   0          46m
kube-system   kube-proxy-7dpq5                          1/1     Running   0          51m
kube-system   kube-proxy-x5p6h                          1/1     Running   0          46m
kube-system   kube-proxy-z8nhd                          1/1     Running   0          46m
kube-system   kube-scheduler-test-test-jgrph            1/1     Running   0          51m
```
