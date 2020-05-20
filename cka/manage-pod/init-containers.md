# Init containers

Init containers are executed in order prior to containers being started.
If any init container fails, the pod is considered to have failed and is
handled according to its restartPolicy. The name for an init container or
normal container must be unique among all containers.

## Lab

### Goals

- Create pod with two containers:
  - One init container which does `nslookup myservice` to be ready
  - One regular container just echo something

- Check the status of the pod created above
  - It will always under init status

- Create a nginx service named `myservice`
  - Create a nginx deployment
  - Expose the deployment

- Check the status of the pod created at the first step
  - The pod should be running after a few seconds

### Steps

#### Create Pod with init containers

Apply the following yaml file to creat the pod with two containers

``` yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: nginx-pod
  name: nginx-pod
spec:
  initContainers:
  - name: init
    image: busybox
    command: ['sh', '-c', 'until nslookup nginx; do echo waiting for nginx service; sleep 2; done;']
  containers:
  - image: busybox
    name: after-init
    command: ['sh', '-c', 'echo it is running && sleep 3600']
    resources: {}
  dnsPolicy: ClusterFirst
  restartPolicy: Always
status: {}
```

#### Create nginx service

- Create nginx deployment

``` bash
kubectl create deployment nginx --image=nginx --dry-run=client -o yaml > nginx-deployment.yaml
kubectl apply -f nginx-deployment.yaml
```

- Expose nginx deployment as a service

``` base
kubectl expose deployment nginx --port=80 --dry-run=client -o yaml > nginx-service.yaml
kubectl apply -f nginx-service.yaml
```
