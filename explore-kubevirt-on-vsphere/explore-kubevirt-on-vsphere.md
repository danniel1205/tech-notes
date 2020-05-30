# Explore kubevirt on vSphere

## Prerequistes

- Have a K8S cluster running on vSphere
  - Per Kubevirt [official doc](https://kubevirt.io/pages/cloud) mentioned, it would be better to have at least 30Gb disk

## Goals

- Install KubeVirt on K8S running on vSphere
- Bring up a test vm
- Install Nginx on the vm
- Export the Nginx servie, so other

## Lab

### Deploy KubeVirt Operator

``` bash
export KUBEVIRT_VERSION=$(curl -s https://api.github.com/repos/kubevirt/kubevirt/releases | grep tag_name | grep -v -- - | sort -V | tail -1 | awk -F':' '{print $2}' | sed 's/,//' | xargs)
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
kubectl apply -f https://raw.githubusercontent.com/kubevirt/kubevirt.github.io/master/labs/manifests/vm.yaml

# Patch VM to run a VM instance
kubectl patch virtualmachine testvm --type merge -p '{"spec":{"running":true}}'

# Once you start a VM, an VM instance will be running
ubuntu@control:~/kubevirt$ kubectl get vm,vmis
NAME                                AGE   RUNNING   VOLUME
virtualmachine.kubevirt.io/testvm   24m   true

NAME                                        AGE     PHASE       IP             NODENAME
virtualmachineinstance.kubevirt.io/testvm   7m20s   Scheduled   192.168.1.12   worker-0

# Patch VM to stop the VM instance
kubectl patch virtualmachine testvm --type merge -p '{"spec":{"running":false}}'

# Once you stop a VM, the VM instance will be stopped and eventually deleted
ubuntu@control:~/kubevirt$ kubectl get vms,vmis
NAME                                AGE   RUNNING   VOLUME
virtualmachine.kubevirt.io/testvm   25m   false
```

**Note**: When you delete a running VM instance, it will be rescheduled. That means if the VM instance crahsed, KubeVirt should automatically bring the VM instance back.

**Failed**: I failed to install nginx inside the vm, either virt-operator crashes or the worker node becomes NotReady. That is probably because I am running it inside of my VM node (nested vm). My next plan is to try it out in Minikube and see what will happen.
