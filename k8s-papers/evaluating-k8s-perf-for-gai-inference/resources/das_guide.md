# Dynamic Accelerator Slicer (DAS)

**IMPORTANT**: It is worth to mention that DAS is now basically replaced by DRA.

## What is Dynamic Accelerator Slicer (DAS)

**Dynamic Accelerator Slicer (DAS)** is an operator designed for Kubernetes and
OpenShift that solves a very common problem with GPUs: **resource waste**.

In a standard Kubernetes setup, if a Pod requests a GPU, the scheduler typically
gives that Pod an entire physical GPU. If the application inside the Pod only
actually needs 10% of the GPU's compute power and memory, the remaining 90% is
completely locked up and wasted. Nobody else can use it.

**DAS solves this by offering "just-in-time" fractional GPUs.**

Here is how DAS works internally and why it is so effective, particularly in
the paper you read:

1. **Leverages NVIDIA MIG (Multi-Instance GPU):** Modern NVIDIA GPUs (like the
   A100 or H100) have a hardware feature called MIG, which allows a single
   physical GPU to be partitioned into several smaller, fully isolated "slices."
   Each slice gets its own dedicated compute cores and memory bandwidth.
2. **Dynamic / On-Demand Slicing:** Historically, administrators had to
   statically configure MIG slices (e.g., hardcoding a server to always split
   its GPUs into quarters). This is inflexible. DAS changes this by listening to
   incoming Kubernetes Pod requests. When a Pod asks for a specific "size" of
   GPU, DAS talks to the NVIDIA GPU Operator and carves out a slice of the exact
   requested size *on the fly*.
3. **Ephemeral Slicing:** As soon as the application finishes running and the
   Pod is terminated, DAS destroys the slice and returns the capacity back to
   the main physical GPU pool so it can be re-sliced differently for the next
   workload.
4. **Intelligent Scheduling Gates:** Kubernetes scheduler normally tries to
   schedule pods immediately. DAS uses "Scheduling Gates" to hold the Pod in
   place while the actual GPU hardware is re-partitioned. Once the slice is
   ready, DAS lets the Pod schedule onto that specific node and consume that
   specific slice.

**In the context of the paper:**
The authors used 8 physical GPUs. Without DAS, they could only run exactly 8 ASR
(speech-to-text) jobs at the same time. By using DAS, they dynamically sliced
those 8 GPUs into much smaller instances, allowing them to run **25 jobs
concurrently**. Because the Whisper model is relatively small, it didn't need a
full GPU anyway, resulting in massive efficiency gains and significantly faster
overall completion times.

## Simulating DAS Locally

While you cannot run actual GPU workloads on a Mac Mini, you can still test DAS
scheduling and orchestration using its **Emulated Mode** on a local Kubernetes
in Docker (`kind`) cluster. In this mode, DAS bypasses actual hardware detection
and instead fakes the presence of a MIG-capable GPU, allowing you to observe
how Kubernetes handles fractional GPU requests.

### 1. Start a Local Kind Cluster

First, create a fresh Kubernetes cluster using `kind`:

```bash
kind create cluster --name das-test
```

### 2. Install Prerequisites

Before installing DAS, you need `cert-manager` for webhooks:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.4/cert-manager.yaml
```

Wait for the cert-manager pods to be ready:

```bash
kubectl wait --for=condition=ready pod -l app=cert-manager -n cert-manager --timeout=90s
```

### 3. Deploy the DAS Operator in Emulated Mode

The primary repository for DAS is at `git@github.com:project-kessel/das.git`.
Normally, you use their `Makefile` or `just` tasks to deploy but the critical
component here is enabling the `EMULATED_MODE`.

Clone the DAS repository:

```bash
git clone https://github.com/project-kessel/das.git
cd das
```

Export the necessary environment variable to turn on hardware emulation, and
deploy the operator to your local cluster:

```bash
export EMULATED_MODE=enabled
make deploy
```

### 4. Verify the Synthetic GPU

Once the DAS operator and its daemonsets are running, it will inject a synthetic
GPU capacity into your kind node. Check the node capacity to confirm it registers
a fake MIG-capable GPU:

```bash
kubectl describe node das-test-control-plane | grep -A 5 "Capacity:"
```

You should see an artificial GPU resource registered.

### 5. Test Dynamic Slicing

You can now submit a test Pod that requests a GPU slice:

```yaml
# test-pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: gpu-slice-test
spec:
  containers:
  - name: test-container
    image: ubuntu:latest
    command: [ "sleep", "100000" ]
    resources:
      limits:
        nvidia.com/mig-2g.10gb: 1
```

Apply the pod:

```bash
kubectl apply -f test-pod.yaml
```

Watch the DAS operator in action using `kubectl get events`. You will notice
that DAS uses a **Scheduling Gate** to briefly pause the Pod, fakes the creation
of the `2g.10gb` slice, removes the gate, and then the Kubernetes scheduler
assigns the Pod to that specific slice.
