# Sample exam

## Create a cluster

- Use `kubeadm` to craete a cluster. 1 control plane node and 3 worker nodes.
- Task is complete if `kubectl get nodes` shows all nodes in a `ready` state.

### Solution

- `kubeadm` on control plane node

``` bash
# switch to sudo
kubeadm init --pod-network-cidr=192.168.0.0/16

# swtich back to regular user
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

- `kubeadm` on worker node

``` bash
# switch to sudo user
kubeadm join xxxxx # The join command is available after kubeadm init from control plane node
```

- Apply CNI on control plane node

``` bash
kubectl apply -f https://docs.projectcalico.org/v3.14/manifests/calico.yaml
```

Nodes will be ready after above step

## Create a Pod

- Create a Pod runs the latest version of the alpine image
- This Pod shuold be configured to sleep 3600 seconds and shuold be created in the mynamespace namespace. Make sure the Pod is automatically restarted if it fails

## Create a Pod with init container

- Configure a Pod runs 2 containers.
- The first container should create the `/data/runfile.txt`
- The second container should only be start once this file has been created
- The second container should run the `sleep 10000` command as its task

## Configure storage

- Create persistent volume that uses local host storage
- This PV should be accessible from all namespaces
- Run a Pod with the name `pv-pod` that uses this persistent volume from `myvol` namespace

## Run a Pod once

- Run a Pod with name `xxazz-pod` under `run-once` namespace by using the alphine image with command `sleep 3600`
- Ensure the task in the Pod runs once and stops after running it once

## Manage updates

- Create a deployment runs `nginx 1.14`
- Enable recording and perform rolling upgrade to latest `nginx` version
- After upgrade, undo the upgrade again back to `nginx 1.14`

## Use label

- Find all k8s objects in all namespaces that have the label `k8s-app` set to the value `kube-dns`

## Use ConfigMaps

- Create a ConfigMap that defines the variable `myuser=mypassword`
- Create a Pod that runs alpine and use this variable from the ConfigMap

## Run parallel Pods

- Createa a solution runs multiple Pods in parallel
- The solution should start `nginx` and ensure that it is started on every node
- If new node is added, the Pod is automatically added to that node as well

## Mark a Node as unavailable

- Mark node `worker-3` as unavailable
- Ensure that all Pods are moved away from the local nodes and started again somewhere else
- After successfully executing this task, make sure `worker-3` can be used again

## Use Maintanance mode

- Put the node `worker-2` in maintenance mode, no new Pods will be scheduled on it
- After successfully executing this task, undo it

## Back up Etcd database

- Create a backup of `Etcd` database
- Write the backup to `/var/exam/etcd-backup`

## Use DNS

- Start a Pod runs `busybox` image
- Use the name `busy33` for this Pod
- Expose this Pod on a cluster IP address
- Configure the Pod and Service such that the DNS name resolution is possible
- Use `nslookup` to look up the names of both
- Write the output of the DNS lookup command to `/var/exam/dnsnames.txt`

## Configure a Node to autostart a Pod

- Configure the node `worker-3` to automatically start a Pod that runs `nginx` using the name `auto-web`
- Put the manifest file in `/etc/kubernetes/manifests`

## Find the Pod with the highest CPU load

- Find the Pod with the highest CPU load and write its name to the file `/var/exam/cpu-pods.txt`

