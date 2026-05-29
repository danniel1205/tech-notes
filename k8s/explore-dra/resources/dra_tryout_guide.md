# Trying Out Dynamic Resource Allocation (DRA)

This guide provides step-by-step instructions to try out Dynamic
Resource Allocation (DRA) in Kubernetes. It covers both a local
**Kind** environment (no cost, API testing) and a Google Cloud
**GKE** environment (real hardware).

---

## Prerequisites (Both Options)

* **Standard Kubernetes Client (`kubectl`)**: v1.34 or later.
* **A Workspace or Cluster Check**: Ensure you are in the correct context (`kubectl config current-context`).

---

## 🛠️ Option A: Local Testing with Kind (Easiest & Free)

Kind (Kubernetes in Docker) is the fastest way to explore the DRA
APIs and lifecycle without provisioning expensive cloud GPUs. Since
Kubernetes 1.34+ (2025/2026), DRA is General Availability and
enabled by default.

### 1. Create a Kind Cluster

Create a cluster using Kubernetes v1.34+ image:

```yaml

# kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  image: kindest/node:v1.34.0  # Use latest v1.34 or v1.35+
- role: worker
  image: kindest/node:v1.34.0
```

Run:
```bash
kind create cluster --config kind-config.yaml
```

### 2. Verify DRA Support

Check if the DRA API objects are available:

```bash
kubectl api-resources | grep -i resource
```
You should see `deviceclasses` and `resourceclaims`.

---

## ☁️ Option B: Google Cloud (GKE) (Real Hardware)

Use this if you have a Google Cloud Project and want to test real hardware slicing.

### 1. Enable GKE API

Ensure the GKE API is enabled:
```bash
gcloud services enable container.googleapis.com
```

### 2. Create a GKE Standard Cluster (v1.34+)

DRA is supported in standard clusters running v1.34 or later.

```bash
gcloud container clusters create dra-cluster \
    --cluster-version=latest \
    --region=us-central1 \
    --num-nodes=1
```

### 3. Create a Node Pool for GPUs (Disabling Default Plugins)

To use DRA, you must disable the default Device Plugin and let the DRA driver take over.

```bash
gcloud container node-pools create gpu-pool-dra \

    --cluster=dra-cluster \
    --region=us-central1 \
    --node-locations=us-central1-a \
    --machine-type=n1-standard-4 \
    --accelerator=type=nvidia-tesla-t4,count=1 \
    --node-labels="nvidia.com/gpu.present=true,gke-no-default-nvidia-gpu-device-plugin=true"
```

Because we configured the nodepool with
`gke-no-default-nvidia-gpu-device-plugin=true` to use DRA, GKE has
disabled the dynamic driver installer completely alongside the legacy
device plugin. We have to manually install Nvidia driver. If not,
the DRA kubelet plugin won't be able to find the driver.

Run:

```bash
kubectl apply -f <https://raw.githubusercontent.com/GoogleCloudPlatform/container-engine-accelerators/master/nvidia-driver-installer/cos/daemonset-preloaded.yaml>

kubectl patch daemonset nvidia-driver-installer -n kube-system --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/affinity", "value": {"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"cloud.google.com/gke-accelerator","operator":"Exists"}]}]}}}}]'
```

### 4. Install NVIDIA DRA Driver (via Helm)

Install the vendor driver to manage slicing.

```bash
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia
helm repo update
helm install nvidia-dra nvidia/nvidia-dra-driver-gpu \
  --namespace kube-system \
  --set "gpuResourcesEnabledOverride=true" \
  --set "nvidiaDriverRoot=/home/kubernetes/bin/nvidia" \
  --set "kubeletPlugin.nodeSelector.cloud\.google\.com/gke-accelerator=nvidia-tesla-t4" \
  --set "kubeletPlugin.tolerations[0].key=nvidia.com/gpu" \
  --set "kubeletPlugin.tolerations[0].operator=Exists" \
  --set "kubeletPlugin.tolerations[0].effect=NoSchedule" \
  --set "controller.affinity=null" \
  --set "controller.tolerations=null"
```

---

## 🧪 Verification: Deploy a Sample Workload

Once your cluster (Kind or GKE) is ready, test the DRA flow using a
simulated or generic device class.

### 1. Define a `DeviceClass` (Optional on GKE)

This tells Kubernetes what kind of devices are available.

```yaml
# device-class.yaml
apiVersion: resource.k8s.io/v1beta1  # Check exact API version for your v1.34+ setup
kind: DeviceClass
metadata:
  name: premium-gpu
spec:
  # In a real setup, this is handled by the driver
  # For Kind local testing, you might need a mock driver
  # (For GKE, the NVIDIA driver creates these)
  criteria:
    # Example CEL filter
    - expression: "device.attributes['model'] == 'a100' && device.attributes['memory'] >= '40Gi'"
```

On a running GKE cluster with the NVIDIA DRA driver, these default
`DeviceClasses` are automatically created for you (no manual creation needed):

```bash
NAME                                            AGE
compute-domain-daemon.nvidia.com                7m33s
compute-domain-default-channel.nvidia.com       7m33s
gpu.nvidia.com                                  7m33s
mig.nvidia.com                                  7m33s
mrdma.google.com                                88m
netdev.google.com                               88m
vfio.gpu.nvidia.com                             7m33s
```

### 2. Create a `ResourceClaimTemplate`

```yaml
# resource-claim-template-t4.yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaimTemplate
metadata:
  name: t4-gpu-claim-template
  namespace: default
spec:
  spec:
    devices:
      requests:
        - name: t4-gpu
          exactly:
            deviceClassName: gpu.nvidia.com
            allocationMode: ExactCount
            count: 1
            selectors:
              - cel:
                  expression: |-
                    device.attributes["gpu.nvidia.com"].productName == "Tesla T4" &&
                    device.capacity["gpu.nvidia.com"].memory.compareTo(quantity("15Gi")) >= 0
```

Apply above resource to the cluster.

### 3. run a Pod tracking the Claim

```yaml
# pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: ai-inference-pod
  namespace: default
spec:
  containers:
  - name: inference-app
    image: nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0-ubuntu22.04  # NVIDIA CUDA verification image
    resources:
      claims:
      - name: requested-t4-gpu
  resourceClaims:
  - name: requested-t4-gpu
    resourceClaimTemplateName: t4-gpu-claim-template                        # Direct mapping (no 'source:' key)
  tolerations:
  - key: "nvidia.com/gpu"
    operator: "Equal"
    value: "present"
    effect: "NoSchedule"
```

Above pod runs a simple vector add test, you should be able to see the
following outcome from the log:

```text
[Vector addition of 50000 elements]
Copy input data from the host memory to the CUDA device
CUDA kernel launch with 196 blocks of 256 threads
Copy output data from the CUDA device to the host memory
Test PASSED
Done
```

---

## ⚠️ Caveats & Troubleshooting running DRA on GKE

Deploying DRA on Google Kubernetes Engine (GKE) standard clusters
(especially versions v1.34 and v1.35+) introduces several GKE-specific
behavioral caveats:

### Silent GKE Node Label Overrides

GKE node-pool controllers restrict custom user-applied labels in
system-managed namespaces (like `nvidia.com/*`) and will silently override
`nvidia.com/gpu.present=true` back to `false` during bootstrapping.

* **Fix:** Do not write selector logic targeting custom `nvidia.com`
  labels. Instead, configure Helm and workloads to target GKE's
  system-native accelerator label:
  `cloud.google.com/gke-accelerator: nvidia-tesla-t4`

### The Driver Bootstrapping Trap

When you use the node label `gke-no-default-nvidia-gpu-device-plugin=true`
to bypass GKE's legacy device plugin, **GKE also completely disables
the automated driver installer DaemonSet**.

* **Symptom:** DRA kubelet-plugin pods remain stuck in `Init:0/1`
  with the error `nvidia-smi not found` or `libnvidia-ml.so.1 not found`
  under `/home/kubernetes/bin/nvidia`.
* **Fix:** You must manually deploy GKE's standalone driver installer
  DaemonSet and patch its NodeAffinity rules so that it triggers the
  driver compilation onto your nodes:

  kubectl apply -f <https://raw.githubusercontent.com/GoogleCloudPlatform/container-engine-accelerators/master/nvidia-driver-installer/cos/daemonset-preloaded.yaml>

  kubectl patch daemonset nvidia-driver-installer -n kube-system --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/affinity", "value": {"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"cloud.google.com/gke-accelerator","operator":"Exists"}]}]}}}}]'

  ```

### Mandatory Workload Tolerations

GKE automatically applies node taints (`nvidia.com/gpu=present:NoSchedule`)
to isolate GPU nodes. Because DRA workloads bypass legacy limit
injection, you **must manually specify the toleration** directly in your
Pod's specifications:

```yaml
spec:
  tolerations:
  - key: "nvidia.com/gpu"
    operator: "Equal"
    value: "present"
    effect: "NoSchedule"
```

### CEL Attribute Matching Mismatch

Older templates match against `device.attributes["gpu.nvidia.com"].model`.
However, GKE's NVIDIA driver publishes its hardware names inside the
**`productName`** field instead of `model`.

* **Symptom:** Pods remain pending with: `CEL runtime error: no such key: model`.
* **Fix:** Target the explicit published properties:

  ```yaml
  expression: device.attributes["gpu.nvidia.com"].productName == "Tesla T4"
  ```



