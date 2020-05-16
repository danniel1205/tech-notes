# Mange cluster nodes

## Lab

### Add a new node to the cluster

Official link is [here](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-join/)

### Cordon a node, drain it and uncordon the node

``` bash
# cordon a node
kubectl cordon worker-0

# drain a node
kubectl drain worker-0

# uncordona  node
kubectl uncordon worker-0
```

### Delete an old node from the cluster

``` bash
# delete a node
kubectl delete nodes worker-0
```

### Backup etcd database

Link: https://discuss.kubernetes.io/t/etcd-backup-ssues/8304

### Configure a node to run static nginx pod

- Create a yaml file under `/etc/kubernetes/manifest` on a particular node

``` yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: static-nginx
  name: static-nginx
spec:
  containers:
  - image: nginx
    name: static-nginx
    resources: {}
  dnsPolicy: ClusterFirst
  restartPolicy: Always
status: {}
```

- Restart kubelet by `systemctl restart kubelet`
- You should be able to see the pod get created by using `kubectl get pods`
