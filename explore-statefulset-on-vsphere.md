---
tags: k8s-fundamentals
---
# StatefulSets
https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/

## What are the differences between StatefulSets and Deployments
https://www.magalix.com/blog/kubernetes-statefulsets-101-state-of-the-pods
https://stackoverflow.com/questions/41583672/kubernetes-deployments-vs-statefulsets
https://medium.com/stakater/k8s-deployments-vs-statefulsets-vs-daemonsets-60582f0c62d4

In short, every replica of a stateful set will have its own state, and each of the pods will be creating its own PVC(Persistent Volume Claim). So a statefulset with 3 replicas will create 3 pods, each having its own Volume, so total 3 PVCs.

## StatefulSet try-out on vShpere
### Create storageclass if needed
To make things easier, I am using dynamci provisioning for my volumes. So, I have my storageclass created before hand.
### (Optional) Create ServiceAccount and Rolebinding
```
apiVersion: v1
kind: ServiceAccount
metadata:
  name: statefulset-sa
---

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: statefulset-cr
rules:
- apiGroups: ['policy']
  resources: ['podsecuritypolicies']
  verbs:     ['use']
  resourceNames:
  - vmware-system-privileged
---

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: statefulset-rb
roleRef:
  kind: ClusterRole
  name: statefulset-cr
  apiGroup: rbac.authorization.k8s.io
subjects:
# Authorize specific service accounts:
- kind: ServiceAccount
  name: statefulset-sa
  namespace: default
```
### Create sample nginx statefulset
```
apiVersion: v1
kind: Service
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  ports:
  - port: 80
    name: web
  clusterIP: None
  selector:
    app: nginx
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  serviceName: "nginx"
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      serviceAccountName: statefulset-sa
      containers:
      - name: nginx
        image: k8s.gcr.io/nginx-slim:0.8
        ports:
        - containerPort: 80
          name: web
        volumeMounts:
        - name: www
          mountPath: /usr/share/nginx/html
  volumeClaimTemplates:
  - metadata:
      name: www
    spec:
      storageClassName: statefulset-storageclass
      accessModes: [ "ReadWriteOnce" ] # vSphere only support ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
```
After apply above yaml, you will see the output like this:
```
root@42143908e772adcf6eec02e7bfc59758 [ ~/statefull-set-test ]# kubectl get statefulset,pod,pv,pvc,storageclass
NAME                   READY   AGE
statefulset.apps/web   2/2     13m

NAME        READY   STATUS    RESTARTS   AGE
pod/web-0   1/1     Running   0          13m
pod/web-1   1/1     Running   0          13m

NAME                                                        CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM               STORAGECLASS         REASON   AGE
persistentvolume/pvc-26f864a9-803f-47e2-ba06-c03426888bd7   1Gi        RWO            Delete           Bound    default/www-web-1   gc-storage-profile            13m
persistentvolume/pvc-a405fce4-81cc-4739-bf43-7aefb3831384   1Gi        RWO            Delete           Bound    default/www-web-0   gc-storage-profile            13m

NAME                              STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS         AGE
persistentvolumeclaim/www-web-0   Bound    pvc-a405fce4-81cc-4739-bf43-7aefb3831384   1Gi        RWO            gc-storage-profile   13m
persistentvolumeclaim/www-web-1   Bound    pvc-26f864a9-803f-47e2-ba06-c03426888bd7   1Gi        RWO            gc-storage-profile   13m

NAME                                             PROVISIONER              RECLAIMPOLICY   VOLUMEBINDINGMODE   ALLOWVOLUMEEXPANSION   AGE
storageclass.storage.k8s.io/gc-storage-profile   csi.vsphere.vmware.com   Delete          Immediate           false                  10h
```

* Two pods, `web-0` and `web-1` got created in order. The creation order is mentioned in the [doc](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#pod-management-policies)
* Two PVs and PVCs got created for each individule pod

### Scale the statefulset
#### Scale up to 3 replicas
```
root@42143908e772adcf6eec02e7bfc59758 [ ~/statefull-set-test ]# kubectl scale statefulset web --replicas=3

root@42143908e772adcf6eec02e7bfc59758 [ ~/statefull-set-test ]# kubectl get pods,pvc,pv
NAME        READY   STATUS    RESTARTS   AGE
pod/kuard   1/1     Running   0          26h
pod/web-0   1/1     Running   0          17h
pod/web-1   1/1     Running   0          17h
pod/web-2   1/1     Running   0          50s

NAME                              STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS         AGE
persistentvolumeclaim/www-web-0   Bound    pvc-a405fce4-81cc-4739-bf43-7aefb3831384   1Gi        RWO            gc-storage-profile   17h
persistentvolumeclaim/www-web-1   Bound    pvc-26f864a9-803f-47e2-ba06-c03426888bd7   1Gi        RWO            gc-storage-profile   17h
persistentvolumeclaim/www-web-2   Bound    pvc-3588dcb0-ce2e-4e1f-9938-4d4b35d5252b   1Gi        RWO            gc-storage-profile   50s

NAME                                                        CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM               STORAGECLASS         REASON   AGE
persistentvolume/pvc-26f864a9-803f-47e2-ba06-c03426888bd7   1Gi        RWO            Delete           Bound    default/www-web-1   gc-storage-profile            17h
persistentvolume/pvc-3588dcb0-ce2e-4e1f-9938-4d4b35d5252b   1Gi        RWO            Delete           Bound    default/www-web-2   gc-storage-profile            45s
persistentvolume/pvc-a405fce4-81cc-4739-bf43-7aefb3831384   1Gi        RWO            Delete           Bound    default/www-web-0   gc-storage-profile            17h

```
#### Scale down to 0 replicas
```
root@42143908e772adcf6eec02e7bfc59758 [ ~/statefull-set-test ]# kubectl scale statefulset web --replicas=0

root@42143908e772adcf6eec02e7bfc59758 [ ~/statefull-set-test ]# kubectl get pods,pvc,pv
NAME        READY   STATUS    RESTARTS   AGE
pod/kuard   1/1     Running   0          26h

NAME                              STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS         AGE
persistentvolumeclaim/www-web-0   Bound    pvc-a405fce4-81cc-4739-bf43-7aefb3831384   1Gi        RWO            gc-storage-profile   17h
persistentvolumeclaim/www-web-1   Bound    pvc-26f864a9-803f-47e2-ba06-c03426888bd7   1Gi        RWO            gc-storage-profile   17h
persistentvolumeclaim/www-web-2   Bound    pvc-3588dcb0-ce2e-4e1f-9938-4d4b35d5252b   1Gi        RWO            gc-storage-profile   3m19s

NAME                                                        CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM               STORAGECLASS         REASON   AGE
persistentvolume/pvc-26f864a9-803f-47e2-ba06-c03426888bd7   1Gi        RWO            Delete           Bound    default/www-web-1   gc-storage-profile            17h
persistentvolume/pvc-3588dcb0-ce2e-4e1f-9938-4d4b35d5252b   1Gi        RWO            Delete           Bound    default/www-web-2   gc-storage-profile            3m14s
persistentvolume/pvc-a405fce4-81cc-4739-bf43-7aefb3831384   1Gi        RWO            Delete           Bound    default/www-web-0   gc-storage-profile            17h
```
* The PVs and PVCs were not deleted automatically. This is also a expected behavior mentioned in [doc](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#limitations)

#### Scale up to 3 replicas again
```
root@42143908e772adcf6eec02e7bfc59758 [ ~/statefull-set-test ]# kubectl scale statefulset web --replicas=3

root@42143908e772adcf6eec02e7bfc59758 [ ~/statefull-set-test ]# kubectl get pods,pvc,pv
NAME        READY   STATUS    RESTARTS   AGE
pod/kuard   1/1     Running   0          26h
pod/web-0   1/1     Running   0          84s
pod/web-1   1/1     Running   0          72s
pod/web-2   1/1     Running   0          14s

NAME                              STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS         AGE
persistentvolumeclaim/www-web-0   Bound    pvc-a405fce4-81cc-4739-bf43-7aefb3831384   1Gi        RWO            gc-storage-profile   17h
persistentvolumeclaim/www-web-1   Bound    pvc-26f864a9-803f-47e2-ba06-c03426888bd7   1Gi        RWO            gc-storage-profile   17h
persistentvolumeclaim/www-web-2   Bound    pvc-3588dcb0-ce2e-4e1f-9938-4d4b35d5252b   1Gi        RWO            gc-storage-profile   7m13s

NAME                                                        CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM               STORAGECLASS         REASON   AGE
persistentvolume/pvc-26f864a9-803f-47e2-ba06-c03426888bd7   1Gi        RWO            Delete           Bound    default/www-web-1   gc-storage-profile            17h
persistentvolume/pvc-3588dcb0-ce2e-4e1f-9938-4d4b35d5252b   1Gi        RWO            Delete           Bound    default/www-web-2   gc-storage-profile            7m8s
persistentvolume/pvc-a405fce4-81cc-4739-bf43-7aefb3831384   1Gi        RWO            Delete           Bound    default/www-web-0   gc-storage-profile            17h
```
* The PVs and PVCs will not be recreated, they are reused

### Access to the pods behind statefulset
```
root@42143908e772adcf6eec02e7bfc59758 [ ~ ]# kubectl get svc nginx -o wide
NAME    TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE   SELECTOR
nginx   ClusterIP   None         <none>        80/TCP    17h   app=nginx

root@42143908e772adcf6eec02e7bfc59758 [ ~ ]# kubectl get endpoints nginx -o wide
NAME    ENDPOINTS                                        AGE
nginx   192.0.160.10:80,192.0.160.11:80,192.0.160.9:80   17h

root@42143908e772adcf6eec02e7bfc59758 [ ~ ]# kubectl get pods -o wide
NAME    READY   STATUS    RESTARTS   AGE   IP             NODE                                                     NOMINATED NODE   READINESS GATES
web-0   1/1     Running   0          29m   192.0.160.9    test-cluster-e2e-script-workers-hzggp-79ff5fb574-2z8nd   <none>           <none>
web-1   1/1     Running   0          29m   192.0.160.10   test-cluster-e2e-script-workers-hzggp-79ff5fb574-2z8nd   <none>           <none>
web-2   1/1     Running   0          28m   192.0.160.11   test-cluster-e2e-script-workers-hzggp-79ff5fb574-2z8nd   <none>           <none>
```
#### Write some dummy data to volume
```
for i in 0 1 2; do kubectl exec web-$i -- sh -c 'echo $(hostname) > /usr/share/nginx/html/index.html'; done

for i in 0 1 2; do kubectl exec -it web-$i -- curl localhost; done
web-0
web-1
web-2
```
#### Create a wrapper LB service to expose the statefulset
https://itnext.io/exposing-statefulsets-in-kubernetes-698730fb92a1
```
apiVersion: v1
kind: Service
metadata:
  name: nginx-lb-svc
  labels:
    app: nginx
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP
  selector:
    app: nginx
```
#### Try access the service from curl
It will randomly send the traffic to the pod in statefulset, and each pod has different state
```
root@423ff1d6680e6d6469f3f001313a0791 [ ~/statefulset-test ]# curl http://192.168.123.3:80
web-0
root@423ff1d6680e6d6469f3f001313a0791 [ ~/statefulset-test ]# curl http://192.168.123.3:80
web-0
root@423ff1d6680e6d6469f3f001313a0791 [ ~/statefulset-test ]# curl http://192.168.123.3:80
web-1
root@423ff1d6680e6d6469f3f001313a0791 [ ~/statefulset-test ]# curl http://192.168.123.3:80
web-2
```