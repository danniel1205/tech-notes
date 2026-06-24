# Step-by-Step Guide: Hosting Gemma 4 (12B IT) on GKE using LWS Multi-Node Parallelism

This guide provides step-by-step instructions to deploy Google's **Gemma 4 (12B IT)** model (multimodal reasoning model, ~24 GiB weight size at BF16) on GKE using **multi-node sharding**.

Because we are using **`g2-standard-4`** VM nodes (which only contain **1x L4 GPU** each), we must shard the model across **2 separate VM machines** (Node A + Node B) to get the required 2 GPUs. We leverage **LeaderWorkerSet (LWS)** with a group size of 2, sharding the model using **Ray** running inside the LWS pods.

---

## Architecture Design

*   **Cluster Nodes:** 4x `g2-standard-4` VM instances (1x L4 GPU each).
*   **LWS Group (`size: 2`):**
    *   **Leader Pod:** Scheduled on Node A. Requests `nvidia.com/gpu: 1` (hosts GPU Rank 0, acts as Ray Head).
    *   **Worker Pod:** Scheduled on Node B. Requests `nvidia.com/gpu: 1` (hosts GPU Rank 1, acts as Ray Worker).
*   **Communication:** NCCL over network (TCP sockets) between Node A and Node B.

```
                      [ Client Request ]
                              │
                              ▼
                    [ Global Router Proxy ]
                       │             │
                       ▼ (Prefill)   ▼ (Decode)
             [ Prefill LWS ]       [ Decode LWS ]
              (TP=2 w/ Ray)         (TP=2 w/ Ray)
              ┌───────────┐         ┌───────────┐
              │Leader Pod │         │Leader Pod │  ◄─── Node A (1x L4 GPU, Ray Head)
              └─────┬─────┘         └─────┬─────┘
                    │ (NCCL)              │ (NCCL)
              ┌─────▼─────┐         ┌─────▼─────┐
              │Worker Pod │         │Worker Pod │  ◄─── Node B (1x L4 GPU, Ray Worker)
              └───────────┘         └───────────┘
```

---

## Step 1: Provision the GKE Cluster with 1-GPU Node Pools

Create the GKE cluster and provision node pools featuring `g2-standard-4` VM instances. Since our LWS group size is 2, each pool must have **at least 2 nodes** so the leader and worker pods can schedule onto separate machines.

```bash
# 1. Create the base GKE Cluster
gcloud container clusters create gemma4-serving-cluster \
    --zone us-central1-a \
    --num-nodes 1 \
    --machine-type e2-standard-4

# 2. Create the Prefill GPU pool (2 nodes of g2-standard-4 = 2x L4 GPUs total)
gcloud container node-pools create prefill-pool \
    --cluster gemma4-serving-cluster \
    --zone us-central1-a \
    --num-nodes 2 \
    --machine-type g2-standard-4 \
    --accelerator type=nvidia-l4,count=1

# 3. Create the Decode GPU pool (2 nodes of g2-standard-4 = 2x L4 GPUs total)
gcloud container node-pools create decode-pool \
    --cluster gemma4-serving-cluster \
    --zone us-central1-a \
    --num-nodes 2 \
    --machine-type g2-standard-4 \
    --accelerator type=nvidia-l4,count=1
```

---

## Step 2: Configure the LWS serving manifest (`gemma-serving-lws.yaml`)

We configure the LWS manifest with `size: 2`. The leader starts Ray head, and the worker joins it.

Apply the LWS manifest:
```bash
kubectl apply -f gemma-serving-lws.yaml
```

---

## Step 3: Deploy the Global Router and Configmap

To coordinate disaggregated serving, deploy the Global Router proxy.

1. Apply the router configmap containing the FastAPI routing script:
   ```bash
   kubectl apply -f gemma-router-configmap.yaml
   ```

2. Apply the global router service and deployment:
   ```bash
   kubectl apply -f gemma-global-router.yaml
   ```

---

## Step 4: Validate the serving stack

1. Wait for the serving stack rollout to complete. Because of the native Kubernetes readiness probes, the router pod will only become `Ready` once both prefill and decode backends are fully initialized and serving:
   ```bash
   kubectl rollout status deployment/gemma-router
   ```

2. Establish a local port-forward to the router service:
   ```bash
   kubectl port-forward svc/gemma-router-svc 8080:8080
   ```

3. Send a completions request:
   ```bash
   curl -s -N http://localhost:8080/v1/chat/completions \
     -H "Content-Type: application/json" \
     -d '{
       "model": "google/gemma-4-12b-it",
       "messages": [
         {"role": "user", "content": "What is quantum computing? Explain in 2 sentences."}
       ],
       "stream": true
     }' | python3 -c 'import sys, json; [print(json.loads(line[6:])["choices"][0]["delta"].get("content", ""), end="", flush=True) for line in sys.stdin if line.startswith("data: ") and "[DONE]" not in line]; print()'
   ```