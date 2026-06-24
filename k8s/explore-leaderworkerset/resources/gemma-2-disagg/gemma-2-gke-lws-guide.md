# Step-by-Step Guide: Deploying Gemma on GKE using LeaderWorkerSet & DisaggregatedSet

This guide provides a comprehensive, step-by-step walkthrough to deploy **Gemma 2** (or any compatible open-source LLM like Llama 3) on a Google Kubernetes Engine (GKE) cluster. We will use **vLLM** as the model serving framework, orchestrated by **LeaderWorkerSet (LWS)** and **DisaggregatedSet (DS)** to implement a high-performance **Prefill-Decode Disaggregated serving** architecture.

---

## Architecture Overview

```
                      [ Client Request ]
                              │
                              ▼
                [ Global Router (vLLM Gateway) ]
                (Stateless CPU Pods - Manages Routing)
                              │
             ┌────────────────┴────────────────┐
             ▼ (Forward Prompt)                ▼ (Orchestrate Loop)
     [ Prefill LWS Group ]             [ Decode LWS Group ]
     (1 Leader + N Workers)            (1 Leader + M Workers)
             │                                 │
             ▼                                 ▼
   [ Prefill GPU Workers ] ===============► [ Decode GPU Workers ]
     (Processes input prompt                 (Generates tokens;
      and generates KV-cache)   [RDMA/TCP]    hosts KV-cache)
```

---

## Prerequisites

1.  **Google Cloud Project:** Active GCP project with billing enabled.
2.  **Hugging Face Token:** Access token to download the Gemma model weights (e.g., `google/gemma-2-9b-it`). Accept the model license on Hugging Face before starting.
3.  **gcloud CLI:** Installed and authenticated.
4.  **kubectl & Helm:** Installed on your local machine.

---

## Step 1: Create the GKE Cluster and GPU Node Pools

We will create a GKE cluster with two distinct GPU node pools using **NVIDIA L4 GPUs** (ideal for cost-effective inference serving):
*   **Prefill Node Pool:** Compute-heavy (e.g., L4 GPUs).
*   **Decode Node Pool:** Memory/generation-heavy (e.g., L4 GPUs).

Run the following commands to provision the cluster and node pools:

```bash
# Set variables
PROJECT_ID="your-gcp-project-id"
CLUSTER_NAME="gemma-disaggregated-cluster"
ZONE="us-central1-a"

gcloud config set project $PROJECT_ID

# 1. Create the GKE cluster (standard control plane)
gcloud container clusters create $CLUSTER_NAME \
    --zone $ZONE \
    --cluster-version=latest \
    --machine-type e2-standard-4 \
    --release-channel "regular" \
    --enable-image-streaming \
    --num-nodes 1

# 2. Create the Prefill GPU Node Pool (e.g., 2 nodes with 1 L4 GPU each)
gcloud container node-pools create prefill-pool \
    --cluster $CLUSTER_NAME \
    --zone $ZONE \
    --machine-type g2-standard-4 \
    --accelerator type=nvidia-l4,count=1 \
    --num-nodes 2

# 3. Create the Decode GPU Node Pool (e.g., 2 nodes with 1 L4 GPU each)
gcloud container node-pools create decode-pool \
    --cluster $CLUSTER_NAME \
    --zone $ZONE \
    --machine-type g2-standard-4 \
    --accelerator type=nvidia-l4,count=1 \
    --num-nodes 2

# Get credentials
gcloud container clusters get-credentials $CLUSTER_NAME --zone $ZONE
```

> [!NOTE]
> Ensure GKE automatically installs the NVIDIA GPU device drivers. If they do not install automatically, run:
> `kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/container-engine-accelerators/master/nvidia-driver-installer/cos/daemonset-preloaded.yaml`

---

## Step 2: Install the LeaderWorkerSet Operator

We will install LeaderWorkerSet (LWS) using the official Helm chart. This installs the core `LeaderWorkerSet` CRD and controller.

Run the following command:
```bash
helm install lws oci://registry.k8s.io/lws/charts/lws \
    --version v0.9.0 \
    --namespace lws-system \
    --create-namespace \
    --set-json 'tolerations=[{"key":"nvidia.com/gpu","operator":"Exists","effect":"NoSchedule"}]'
```

Verify that the operator pod is running:
```bash
kubectl get pods -n lws-system
```

---

## Step 3: Create the Hugging Face Secret

Gemma is a gated open-source model. Store your Hugging Face API key in a Kubernetes secret so the containers can authenticate and pull the weights:

```bash
HF_TOKEN="<HF_TOKEN>"

kubectl create secret generic hf-secret \
  --from-literal=token=$HF_TOKEN \
  --dry-run=client -o yaml | kubectl apply -f -
```

---

## Step 4: Deploy the Gemma 2 Prefill and Decode LeaderWorkerSets

Instead of using the unreleased DisaggregatedSet operator, we will deploy the disaggregated serving architecture by manually creating two standard **LeaderWorkerSets** and their corresponding manifest content. 

Check out [gemma-serving-lws.yaml](gemma-serving-lws.yaml) for the full manifest.

Apply the serving configuration:
```bash
kubectl apply -f gemma-serving-lws.yaml
```

---

## Step 5: Deploy the Global Router (vLLM Gateway Proxy)

Now we deploy the stateless **Global Router** that exposes the external client API and manages the scheduling handoff between the Prefill and Decode pods.

Check out [gemma-global-configmap.yaml](gemma-global-configmap.yaml) for the configmap.
Check out [gemma-global-router.yaml](gemma-global-router.yaml) for the full manifest.

Apply the deployment:
```bash
kubectl apply -f gemma-global-configmap.yaml
kubectl apply -f gemma-global-router.yaml
```

Tunnel the router service port to your local workstation:
```bash
kubectl port-forward svc/gemma-router-svc 8080:8080
```

---

## Step 6: Test and Verify the Deployment

Once all the pods are in the `Running` phase, the model weights have been cached, and the port tunnel is active, we can test serving Gemma.

Send an inference request using `curl` to generate tokens:

```bash
curl -s -N http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "google/gemma-2-2b-it",
    "messages": [
      {"role": "user", "content": "What is quantum computing? Explain in 2 sentences."}
    ],
    "stream": true
  }' | python3 -c "
import sys, json
for line in sys.stdin:
    if line.startswith('data: ') and not '[DONE]' in line:
        try:
            chunk = json.loads(line[6:])
            content = chunk['choices'][0]['delta'].get('content', '')
            print(content, end='', flush=True)
        except Exception:
            pass
print()
"
```

You should see Gemma's response stream back in real-time, processed by the Prefill pods, handed over via point-to-point network channels, and generated by the Decode pods.
