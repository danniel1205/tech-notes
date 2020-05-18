# Manage pod networking

## Lab

- Create two services
  - myservice should be exposing port 9376 and forward to targetport 80
  - mydb should be exposing port 9377 and forward to port 80

- Create a Pod that will start a busybox container that will sleep for 3600 seconds, but only if the aforesaid services are available

### Steps

- Create a Pod with init containers. The pod will be under init status waiting for the services to be ready

``` yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: busybox
  name: busybox
spec:
  containers:
  - image: busybox
    name: main-container
    command: ['sh', '-c', 'echo The app is running! && sleep 3600']
    resources: {}
  initContainers:
  - name: myservice-check
    image: busybox
    command: ['sh', '-c', "until nslookup myservice; do echo waiting for myservice; sleep 2; done"]
  - name: mydb-check
    image: busybox
    command: ['sh', '-c', "until nslookup mydb; do echo waiting for mydb; sleep 2; done"]
  dnsPolicy: ClusterFirst
  restartPolicy: Always
```

- Create the service

``` yaml
# kubectl create service clusterip myservice --tcp=9376:80 --dry-run=client -o yaml > myservice.yaml
apiVersion: v1
kind: Service
metadata:
  creationTimestamp: null
  labels:
    app: myservice
  name: myservice
spec:
  ports:
  - name: 9376-80
    port: 9376
    protocol: TCP
    targetPort: 80
  selector:
    app: myservice
  type: ClusterIP
status:
  loadBalancer: {}
---
# kubectl create service clusterip myservice --tcp=9377:80 --dry-run=client -o yaml > mydb.yaml
apiVersion: v1
kind: Service
metadata:
  creationTimestamp: null
  labels:
    app: mydb
  name: mydb
spec:
  ports:
  - name: 9377-80
    port: 9377
    protocol: TCP
    targetPort: 80
  selector:
    app: mydb
  type: ClusterIP
status:
  loadBalancer: {}
```

- Check the busybox pod, they should be running now
