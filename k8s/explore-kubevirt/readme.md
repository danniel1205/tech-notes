# Explore kubevirt

## Prerequistes

- Have a K8S cluster running on minikube

Or

- Have a K8S cluster running on public cloud
  - Per Kubevirt [official doc](https://kubevirt.io/pages/cloud) mentioned, it would be better to have at least 30Gb disk

## Goals

- Install KubeVirt on K8S
- Bring up a ubuntu vm
- Install Nginx on the vm
- Export the vm as K8S servie, so other Pod could access it

## Lab

This steps are performed against a K8S cluster running on VMWare vSphere
If you want to test it local, I would suggest use minikube. For minikube, please follow the instruction [here](https://kubevirt.io/quickstart_minikube/)

### Deploy KubeVirt Operator

``` bash
export KUBEVIRT_VERSION=$(curl -s https://api.github.com/repos/kubevirt/kubevirt/releases | grep tag_name | grep -v -- - | \
  sort -V | tail -1 | awk -F':' '{print $2}' | sed 's/,//' | xargs)
echo $KUBEVIRT_VERSION

# Apply the kubevirt operator yaml
kubectl create -f https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-operator.yaml

# Check it is running
kubectl get pods -n kubevirt
```

### Check for the Virtualization Extenisions

``` bash
egrep 'svm|vmx' /proc/cpuinfo

# The output is empty on vSphere node, so we need to create ConfigMap so that KubeVirt uses emulation mode

kubectl create configmap kubevirt-config -n kubevirt --from-literal debug.useEmulation=true
```

### Deploy KubeVirt

``` bash
kubectl create -f https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-cr.yaml

# Check it's running
kubectl get pods -n kubevirt

kubevirt      virt-api-7dcc588f45-7q6p6           1/1     Running   0          4m33s
kubevirt      virt-api-7dcc588f45-klmn2           1/1     Running   0          4m33s
kubevirt      virt-controller-55b77c4ffb-48c9s    1/1     Running   0          4m7s
kubevirt      virt-controller-55b77c4ffb-hrsmf    1/1     Running   1          4m7s
kubevirt      virt-handler-8jkjz                  1/1     Running   0          4m7s
kubevirt      virt-operator-748868b7f7-95khg      1/1     Running   0          9m33s
kubevirt      virt-operator-748868b7f7-wzq5s      1/1     Running   1          9m33s
```

### Use KubeVirt

<https://kubevirt.io/labs/kubernetes/lab1>

``` bash
# Create the VM
kubectl apply -f ubuntu-vm.yaml

# Patch VM to run a VM instance
kubectl patch virtualmachine ubuntu --type merge -p '{"spec":{"running":true}}'

# Once you start a VM, an VM instance will be running
ubuntu@control:~/kubevirt$ kubectl get vm,vmis
NAME                                AGE   RUNNING   VOLUME
virtualmachine.kubevirt.io/ubuntu   24m   true

NAME                                        AGE     PHASE       IP             NODENAME
virtualmachineinstance.kubevirt.io/ubuntu   7m20s   Scheduled   192.168.1.12   worker-0

# Patch VM to stop the VM instance
kubectl patch virtualmachine testvm --type merge -p '{"spec":{"running":false}}'

# Once you stop a VM, the VM instance will be stopped and eventually deleted
ubuntu@control:~/kubevirt$ kubectl get vms,vmis
NAME                                AGE   RUNNING   VOLUME
virtualmachine.kubevirt.io/ubuntu   25m   false
```

**Note**: When you delete a running VM instance, it will be rescheduled. That means if the VM instance crahsed, KubeVirt
should automatically bring the VM instance back.

**Failed**: I failed to install nginx inside the vm on vSphere, either virt-operator crashes or the worker node becomes
NotReady. That is probably because I am running it inside a nested vm on vSphere. My next plan is to try it out in Minikube
and see what will happen.

### Setup everyting on minikube

Here is the step by step guide: <https://kubevirt.io/quickstart_minikube/>

After the VM instance is up and running, and the nginx has been installed. You could expose the ubuntu VM as a service.

### Expose VM as a K8S service

You could either using virtctl command line tool `virtctl expose vm ubuntu --name=ubuntu-vm-svc --port=80 --target-port=80`. After that, you will see a service running:

``` yaml
daniel@daniel-All-Series:~/kubevirt$ kubectl get svc -o wide
NAME            TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)   AGE   SELECTOR
kubernetes      ClusterIP   10.96.0.1        <none>        443/TCP   37m   <none>
ubuntu-vm-svc   ClusterIP   10.107.119.207   <none>        80/TCP    14m   kubevirt.io/domain=ubuntu,kubevirt.io/size=small
```

### Access the nginx from a different pod

``` bash
kubectl run pod --image=ubuntu
kubectl exec -it pod -- bash
$ curl ubuntu-vm-svc.default.svc.cluster.local

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
</body>
</html>
```

In theory, if VM instance gets deleted or the VM Pod gets deleted, K8S should bring up a new VM instance and VM Pod.
Its IP might have changed, the Endpoints should be updated automatically, so the Service DNS still could route traffic
to corresponding VM instance.
