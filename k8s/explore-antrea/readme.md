# Explore Antrea

## What is Antrea

Antrea is a Kubernetes networking solution intended to be Kubernetes native. It operates at Layer3/4 to provide networking
and security services for a Kubernetes cluster, leveraging Open vSwitch as the networking data plane.

Open vSwitch is a widely adopted high-performance programmable virtual switch; Antrea leverages it to implement Pod networking
and security features. For instance, Open vSwitch enables Antrea to implement Kubernetes Network Policies in a very efficient manner.

![](https://i.imgur.com/v5oo9jB.png)

## Antrea architecture

<https://github.com/vmware-tanzu/antrea/blob/master/docs/architecture.md>

![](https://i.imgur.com/5509JWG.png)

## Deploy Antrea on K8S cluster

<https://github.com/vmware-tanzu/antrea/blob/master/docs/getting-started.md>

This is quite straight forward, just need to do `kubectl apply`. If you have existing CNI installed, you have to completely delete it first.

## Deploy Antrea on Kind

<https://github.com/vmware-tanzu/antrea/blob/master/docs/kind.md>

## Try out the network policy with Antrea

- Create a simple nginx deployment

``` yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  labels:
    app: nginx-deploy
  name: nginx-deploy
spec:
  replicas: 3
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

- Create a simple network policy

``` yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: test-network-policy
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: nginx-deploy
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          run: ubuntu
    ports:
    - protocol: TCP
      port: 80
  egress:
  - {}
```

This network policy isolates all the Pods from `app: nginx-deploy` with the following rules:
```
* Allow the ingress traffic from the Pods with run: ubuntu label
* Allow the ingress traffic from the allowed Pods to 80 port
* Allow all egress traffic
```

- Run a Pod with `run: ubuntu-blocked` label

You shuold see the traffic gets blocked if running curl against any nginx Pod

``` bash
root@ubuntu:/# curl 192.168.1.3
curl: (28) Failed to connect to 192.168.1.3 port 80: Connection timed out
```

- Run a Pod with `run: ubuntu` label

You shuold see the Nginx welcome page if running curl against any nginx Pod

``` bash
root@ubuntu:/# curl 192.168.1.3
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p>If you see this page, the nginx web server is successfully installed and
working. Further configuration is required.</p>

<p>For online documentation and support please refer to
<a href="http://nginx.org/">nginx.org</a>.<br/>
Commercial support is available at
<a href="http://nginx.com/">nginx.com</a>.</p>

<p><em>Thank you for using nginx.</em></p>

```