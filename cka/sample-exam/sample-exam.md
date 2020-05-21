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

### Solution

- Generate Pod yaml

``` bash
kubectl run alpine --image=alpine --dry-run=client -o yaml > alpine-pod.yaml
```

- Modify the generated yaml

``` yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: alpine
  name: alpine
  namespace: mynamespace
spec:
  containers:
  - image: alpine
    name: alpine
    command: ["sh", "-c", "sleep 3600;"]
    resources: {}
  dnsPolicy: ClusterFirst
  restartPolicy: Always
```

- Create namespace

``` bash
kubectl create ns mynamespace
```

- Create Pod

``` bash
kubectl apply -f alpine-pod.yaml

ubuntu@control:~/sample-exam$ kubectl get pods -n mynamespace
NAME     READY   STATUS    RESTARTS   AGE
alpine   1/1     Running   0          24s
```

## Create a Pod with init container

- Configure a Pod runs 2 containers.
- The first container should create the `/data/runfile.txt`
- The second container should only be start once this file has been created
- The second container should run the `sleep 10000` command as its task

### Solution

- Generate Pod yaml

``` bash
kubectl run init-container-pod --image=alpine --dry-run=client -o yaml > init-container-pod.yaml
```

- Modify the Pod yaml

``` yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: init-container-pod
  name: init-container-pod
spec:
  initContainers:
  - name: first-container
    image: alpine
    command: ["sh", "-c", "mkdir -p /data && touch /data/runfile.txt;"]
  containers:
  - image: alpine
    name: second-container
    command: ["sh", "-c", "sleep 10000"]
    resources: {}
  dnsPolicy: ClusterFirst
  restartPolicy: Always
status: {}
```

- Create the Pod

``` bash
kubectl apply -f init-container-pod.yaml

NAME                 READY   STATUS    RESTARTS   AGE
init-container-pod   1/1     Running   0          2m25s
```

## Configure storage

- Create persistent volume that uses local host storage
- This PV should be accessible from all namespaces
- Run a Pod with the name `pv-pod` that uses this persistent volume from `myvol` namespace

### Solution

- Get the PV yaml
<https://kubernetes.io/docs/concepts/storage/persistent-volumes/>

- Modify to use hostPath
<https://kubernetes.io/docs/concepts/storage/volumes/#hostpath>

- Create PV

``` bash
kubectl apply -f pv.yaml
```

- Create `myvol` namespace

``` bash
kubectl create ns myvol
```

- Get PVC yaml
<https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims>

- Modify and apply the PVC yaml

``` yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc
  namespace: myvol
spec:
  accessModes:
    - ReadWriteOnce
  volumeMode: Filesystem
  resources:
    requests:
      storage: 2Gi
  storageClassName: local-storage
```

``` bash
kubectl apply -f pvc.yaml

ubuntu@control:~/sample-exam$ kubectl get pvc -n myvol
NAME   STATUS   VOLUME   CAPACITY   ACCESS MODES   STORAGECLASS    AGE
pvc    Bound    pv       5Gi        RWO            local-storage   11s
```

- Generate Pod yaml

``` bash
kubectl run pv-pod --image=alpine --dry-run=client -o yaml > pv-pod.yaml
```

- Modify the Pod yaml and apply

``` yaml
apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  labels:
    run: pv-pod
  name: pv-pod
  namespace: myvol
spec:
  volumes:
    - name: pod-vol
      persistentVolumeClaim:
        claimName: pvc
  containers:
  - image: alpine
    name: pv-pod
    command: ["sh", "-c", "sleep 3600;"]
    volumeMounts:
    - mountPath: "/persistent-volume"
      name: pod-vol
  dnsPolicy: ClusterFirst
  restartPolicy: Always
```

## Run a Pod once

- Run a Pod with name `xxazz-pod` under `run-once` namespace by using the alphine image with command `sleep 3600`
- Ensure the task in the Pod runs once and stops after running it once

### Solution

#### Could run as Pod

- Generate Pod yaml
- Modify the Pod yaml and apply

``` yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: xxazz-pod
  name: xxazz-pod
  namespace: run-once
spec:
  containers:
  - image: alpine
    name: xxazz-pod
    command: ["sh", "-c", "sleep 3600;"]
    resources: {}
  dnsPolicy: ClusterFirst
  restartPolicy: Never
```

#### Could run as Job

- Generate Job yaml
- Modify the Job yaml and apply

``` yaml
apiVersion: batch/v1
kind: Job
metadata:
  creationTimestamp: null
  name: run-once-job
  namespace: run-once
spec:
  template:
    metadata:
      creationTimestamp: null
    spec:
      containers:
      - image: alpine
        name: run-once-job
        command: ["sh", "-c", "sleep 3600;"]
        resources: {}
      restartPolicy: Never
status: {}
```

## Manage updates

- Create a deployment runs `nginx 1.14`
- Enable recording and perform rolling upgrade to latest `nginx` version
- After upgrade, undo the upgrade again back to `nginx 1.14`

### Solution

- Generate Deployment yaml

``` bash
kubectl create deployment nginx-deploy --image=nginx:1.14 --dry-run=client -o yaml > nginx-deploy.yaml
```

- Modify the Deployment yaml and apply

``` yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  labels:
    app: nginx-deploy
  name: nginx-deploy
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx-deploy
  strategy: {}
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: nginx-deploy
    spec:
      containers:
      - image: nginx:1.14
        name: nginx
        resources: {}
status: {}
```

- Update the nginx image to latest version

``` bash
kubectl set image deployment/nginx-deploy nginx=nginx:latest --record=true
```

- Rollback the image version upgrade

``` bash
kubectl rollout undo deployment nginx-deploy
```

## Use label

- Find all k8s objects in all namespaces that have the label `k8s-app` set to the value `kube-dns`

``` bash
ubuntu@control:~/sample-exam$ kubectl get all --selector=k8s-app=kube-dns -A
NAMESPACE     NAME                           READY   STATUS    RESTARTS   AGE
kube-system   pod/coredns-66bff467f8-hhdnf   1/1     Running   0          80m
kube-system   pod/coredns-66bff467f8-k8dwc   1/1     Running   0          80m

NAMESPACE     NAME               TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)                  AGE
kube-system   service/kube-dns   ClusterIP   10.96.0.10   <none>        53/UDP,53/TCP,9153/TCP   80m

NAMESPACE     NAME                      READY   UP-TO-DATE   AVAILABLE   AGE
kube-system   deployment.apps/coredns   2/2     2            2           80m

NAMESPACE     NAME                                 DESIRED   CURRENT   READY   AGE
kube-system   replicaset.apps/coredns-66bff467f8   2         2         2       80m
```

## Use ConfigMaps

- Create a ConfigMap that defines the variable `myuser=mypassword`
- Create a Pod that runs alpine and use this variable from the ConfigMap

### Solution

- Generate ConfigMap yaml

``` bash
kubectl create configmap pod-cp --dry-run=client -o yaml > pod-cp.yaml
```

- Modify the ConfigMap yaml and apply

``` yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: pod-cp
data:
  myuser: "mypassword"
```

- Generate the Pod yaml
- Modify the Pod yaml and apply

``` yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: cp-pod
  name: cp-pod
spec:
  containers:
  - image: alpine
    name: cp-pod
    command: ["sh", "-c", "echo $myuser; sleep 3600"]
    env:
      - name: myuser
        valueFrom:
          configMapKeyRef:
            name: pod-cp
            key: myuser
  dnsPolicy: ClusterFirst
  restartPolicy: Always
```

## Run parallel Pods

- Createa a solution runs multiple Pods in parallel
- The solution should start `nginx` and ensure that it is started on every node
- If new node is added, the Pod is automatically added to that node as well

### solution

- Get DaemonSet yaml

<https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/>

- Modify the DaemonSet yaml and apply

``` yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  creationTimestamp: null
  labels:
    app: nginx-deamonset
  name: nginx-deamonset
spec:
  selector:
    matchLabels:
      app: nginx-deamonset
  template:
    metadata:
      labels:
        app: nginx-deamonset
    spec:
      containers:
      - image: nginx
        name: nginx
```

## Mark a Node as unavailable

- Mark node `worker-3` as unavailable
- Ensure that all Pods are moved away from the local nodes and started again somewhere else
- After successfully executing this task, make sure `worker-3` can be used again

### Solution 

- Cordon a node

``` bash
kubectl cordon worker-3
```

- Drain a node

``` bashvkubectl drain work-1 --force --ignore-daemonsets
```

- Uncordon a node

``` bash
kubectl uncordon worker-3
```

**Node**: Maybe we could also taint a node

## Use Maintanance mode

- Put the node `worker-2` in maintenance mode, no new Pods will be scheduled on it
- After successfully executing this task, undo it

### Solution

The same solution as above

## Back up Etcd database

- Create a backup of `Etcd` database
- Write the backup to `/var/exam/etcd-backup`

### Solution

``` bash
# https://discuss.kubernetes.io/t/etcd-backup-ssues/8304
etcdctl snapshot save mysnapshot.db --endpoints=https://127.0.0.1:2379 --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key
```

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

### Solution

- Generate Pod yaml
- Copy the Pod yaml to `/etc/kubernetes/manifests/` under node `worker-3`

## Find the Pod with the highest CPU load

- Find the Pod with the highest CPU load and write its name to the file `/var/exam/cpu-pods.txt`

### Solution

- Get metric-server yaml

<https://github.com/kubernetes-sigs/metrics-server>

- Modify the metric-server yaml and apply

<https://github.com/kubernetes-sigs/metrics-server#configuration>