# Testing Kueue on a Local Mac with Kind

This guide will walk you through setting up a local Kubernetes cluster using `kind`, installing Kueue, and running a simple job to see how Kueue
manages resources and queues.

## 1. Prerequisites

Before we start, you need a few command-line tools installed. Since you are on a Mac, you can use Homebrew.
Make sure you have Docker Desktop or OrbStack running.

```bash
# Install kind (Kubernetes IN Docker)
brew install kind

# Install kubectl
brew install kubectl
```

## 2. Create a Kind Cluster

Let's spin up a fresh Kubernetes cluster. This takes about a minute.

```bash
# Create a new kind cluster named "kueue-test"
kind create cluster --name kueue-test
```

Once it's done, verify your cluster is running:

```bash
kubectl cluster-info --context kind-kueue-test
```

## 3. Install Kueue

We will install Kueue using the official release manifests. This will deploy the Kueue controller manager to your cluster.

```bash
# Apply the latest Kueue release manifest
VERSION=v0.16.2 # Or the latest version available
kubectl apply --server-side -f https://github.com/kubernetes-sigs/kueue/releases/download/$VERSION/manifests.yaml
```

Wait for the Kueue components to be ready:

```bash
kubectl wait deploy/kueue-controller-manager -n kueue-system --for=condition=available --timeout=5m
```

## 4. Set Up Kueue Configuration

Kueue uses three main custom resources to manage quotas:

1. **ResourceFlavor**: Defines what kind of nodes/resources you have (e.g., standard CPUs).
2. **ClusterQueue**: A cluster-wide queue that manages quotas for different ResourceFlavors.
3. **LocalQueue**: A queue in a specific namespace that users submit jobs to. It points to a ClusterQueue.

Let's apply a basic setup. Create a file named `kueue-setup.yaml`:

```yaml
---
apiVersion: kueue.x-k8s.io/v1beta1
kind: ResourceFlavor
metadata:
  name: default-flavor
---
apiVersion: kueue.x-k8s.io/v1beta1
kind: ClusterQueue
metadata:
  name: cluster-queue
spec:
  namespaceSelector: {} # Allows all namespaces to use this ClusterQueue
  resourceGroups:
  - coveredResources: ["cpu", "memory", "pods"]
    flavors:
    - name: default-flavor
      resources:
      - name: "cpu"
        nominalQuota: 2 # Maximum 2 CPUs at a time
      - name: "memory"
        nominalQuota: 2Gi
      - name: "pods"
        nominalQuota: 5
---
apiVersion: kueue.x-k8s.io/v1beta1
kind: LocalQueue
metadata:
  namespace: default
  name: user-queue
spec:
  clusterQueue: cluster-queue
```

Apply this configuration:

```bash
kubectl apply -f kueue-setup.yaml
```

## 5. Submit a Sample Job

Now, let's submit a Kubernetes Job and tell Kueue to manage it by pointing it to our `LocalQueue` using a specific label.

Create a file named `sample-job.yaml`:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  generateName: sample-job-
  namespace: default
  labels:
    kueue.x-k8s.io/queue-name: user-queue # This tells Kueue to manage this job
spec:
  parallelism: 3
  completions: 3
  template:
    spec:
      containers:
      - name: dummy-job
        image: gcr.io/k8s-staging-perf-tests/sleep:v0.1.0
        args: ["10s"]
        resources:
          requests:
            cpu: 1 # Note: Our cluster queue only allows 2 CPUs at a time!
            memory: "200Mi"
      restartPolicy: Never
```

Submit the job:

```bash
kubectl create -f sample-job.yaml
```

## 6. Observe Kueue in Action

Because our `ClusterQueue` only has a quota of `2 CPUs`, and our Job requests `3` parallel pods (each wanting `1 CPU`), Kueue will:

1. Allow 2 pods to run immediately.
2. Hold back the 3rd pod until one of the first two finishes.

Watch the job and pods to see this happen:

```bash
# Watch the pods. You should see 2 running and the rest pending/waiting
kubectl get pods -w
```

You can also check the status of the Workload (which Kueue creates to represent the Job):

```bash
kubectl get workloads
# Or describe it to see why it's pending/admitted:
kubectl describe workload --selector=kueue.x-k8s.io/queue-name=user-queue
```

## 7. Cleanup

When you are done experimenting, you can clean up the entire Kind cluster easily:

```bash
kind delete cluster --name kueue-test
```

## 7. FAQ

### What are the differences between LocalQueue and ClusterQueue

A LocalQueue must point to a specific ClusterQueue. You can think of it like an organizational structure:

- Multiple LocalQueues from different namespaces can point to the same ClusterQueue.
- This allows an administrator to define a single ClusterQueue with a central pool of resources (quota), while users from different namespaces
  submit their jobs to their own namespace's LocalQueue to consume from that shared pool.

### For Kueue, how does it know the real time cluster resource usage

Kueue does not actually track real-time resource utilization (like how much CPU/RAM a pod is actively consuming at any given second). Instead,
it tracks real-time resource allocation (based on pod requests) against pre-defined quotas. Kueue is fundamentally "blind" to the true, real-time
physical capacity of your Kubernetes cluster (e.g., it doesn't calculate the sum of Allocatable CPU/Memory across all your nodes).It relies entirely
on the administrator (or an external automation script) to explicitly define the nominalQuota in the ClusterQueue objects. The administrator has to
look at the physical cluster (or their budget for cloud nodes) and manually declare, "I am dedicating a maximum of 100 CPUs and 500Gi of memory to
this ClusterQueue."
