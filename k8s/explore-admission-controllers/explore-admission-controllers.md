---
tags: k8s-fundamentals
---
# Deep dive admission controllers
## What are admission controllers
- **Gatekeeper** that intercept (authenticated) API requests and may change the request object or deny the request altogether. 
    For example, when a namespace is deleted and subsequently enters the Terminating state, the NamespaceLifecycle admission controller is what prevents any new objects from being created in this namespace. 
- Among the more than 30 admission controllers shipped with Kubernetes, two take a special role because of their nearly limitless flexibility - **`ValidatingAdmissionWebhooks`** and **`MutatingAdmissionWebhooks`**. 
    This approach decouples the admission controller logic from the Kubernetes API server, thus allowing users to implement custom logic to be executed whenever resources are created, updated, or deleted in a Kubernetes cluster.
![admission controller phases](https://i.imgur.com/qlwsvEL.png)

Ref: [k8s-blog](https://kubernetes.io/blog/2019/03/21/a-guide-to-kubernetes-admission-controllers/)

## How to write basic validation webhook from beginning
- https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks
- https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html
- https://medium.com/ovni/writing-a-very-basic-kubernetes-mutating-admission-webhook-398dbbcb63ec
- https://github.com/banzaicloud/admission-webhook-example

## Experiments
### Missions
- Write a validating admission webhook
  - Do not allow the creation of Pod without namespace
- Try build the webhook using ko
### Prerequisites
- Getting MacOS set up with k8s 1.18 with kind
    ```
    brew update
    brew upgrade kind kubectl
    kind create cluster --image kindest/node:v1.18.0
    ```
- Ensure that the admissionregistration.k8s.io/v1 or admissionregistration.k8s.io/v1beta1 API is enabled.
    ```
    kubectl api-versions | grep admissionregistration
    admissionregistration.k8s.io/v1
    admissionregistration.k8s.io/v1beta1
    ```
### Write a webhook server
The code is modified based on [link](https://github.com/kubernetes/kubernetes/tree/ec8c186fe8181f52670716d8d10aa7663e868491/test/images/agnhost/webhook). And has been check in to [github](https://github.com/danniel1205/sample-webhook-server).
* Checkout the code
* Build the docker image
```
docker build --no-cache -f Dockerfile -t sample-webhook-server:v1 --rm=true .
```
* Load into kind
```
kind load docker-image sample-webhook-server:v1
```

### Generate certs and keys
**Note**: You have to change the `"/CN=sample-webhook-server.webhook.svc"` in genkeys.sh to be `"/CN=<service-name>.<namespace>.svc"`
```
cd $GOPATH/src/github.com/danniel1205/sample-webhook-server/
mkdir -p keys
.hacks/genkeys.sh keys

tree keys
keys
├── ca.crt
├── ca.key
├── ca.srl
├── webhook-server-tls.crt
└── webhook-server-tls.key
```
### Create a namespace
```
kubectl create namespace webhook
```
### Create secrets from the keys generated
```
kubectl create secret tls webhook-tls --key=./keys/webhook-server-tls.key --cert=./keys/webhook-server-tls.crt -n webhook
```

## Create the webhook service
```
kubectl apply -f $GOPATH/src/github.com/danniel1205/sample-webhook-server/deploy/01-deployment.yaml
```
```
kubectl get svc,deployment,pod -n webhook
NAME                            TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)   AGE
service/sample-webhook-server   ClusterIP   10.99.16.47   <none>        443/TCP   2m33s

NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/sample-webhook-server   1/1     1            1           2m33s

NAME                                         READY   STATUS    RESTARTS   AGE
pod/sample-webhook-server-6449948fcb-d9pq9   1/1     Running   0          25s
```

## Create ValidatingWebhookConfiguration
**Note**: Update the CABundle in `02-webhook-config.yaml` to be base64 encoded of `keys/ca.crt`
```
kubectl apply -f $GOPATH/src/github.com/danniel1205/sample-webhook-server/deploy/02-webhook-config.yaml
```

## Try to create the test pod
```
kubectl apply -f $GOPATH/src/github.com/danniel1205/sample-webhook-server/deploy/test-pod.yaml

Error from server: error when creating "deploy/test-pod.yaml": admission webhook "sample-webhook-server.example.com" denied the request: the namespace must be specified to create pod
```

## Q&A
- Why validating admission is after mutating admission ?
    The reason is whatever request object a validating webhook sees needs to be the final version that would be persisted to `etcd`