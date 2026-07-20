# Step-by-Step Guide: Hosting Gemma 4 (12B IT) on GKE using LWS and Local GPU Parallelism

This guide provides step-by-step instructions to deploy Google's mid-sized
**Gemma 4 (12B IT)** model (multimodal reasoning model, ~24 GiB weight size at
BF16) in a **Disaggregated Serving (DS)** architecture on GKE.

To achieve maximum performance and avoid the complexity of distributed Ray
orchestration, we serve the model using **local Tensor Parallelism (TP=2)**
inside a single pod. We achieve this by provisioning GKE node pools featuring
**`g2-standard-24`** VM instances, each hosting **2x NVIDIA L4 GPUs** (24GB VRAM
each) connected locally.

---

## Architecture Design

* **Cluster Nodes:** 2x `g2-standard-24` VM instances.
  * **Node A (Prefill Nodepool):** Hosts 1x Prefill pod utilizing its 2 local L4 GPUs.
  * **Node B (Decode Nodepool):** Hosts 1x Decode pod utilizing its 2 local L4 GPUs.
* **LWS Group (`size: 1`):**
  * Since all 2 GPUs are hosted locally on a single machine, we run LWS with `size: 1` (Leader Pod only, 0 Worker pods).
  * The Leader Pod requests `nvidia.com/gpu: 2` to gain exclusive access to the VM's GPUs.
* **Communication:** vLLM executes Tensor Parallelism (TP=2) locally using native Python multiprocessing. The KV cache is transferred from Node A to Node B over GKE's network via TCP sockets.

```text
                      [ Client Request ]
                              │
                              ▼
                    [ Global Router Proxy ]
                       │             │
                       ▼ (Prefill)   ▼ (Decode)
             [ Prefill Engine ]    [ Decode Engine ]
               (TP=2 local)          (TP=2 local)
             ┌──────────────┐      ┌──────────────┐
             │ 2x L4 GPUs   │      │ 2x L4 GPUs   │  ◄─── Respective Nodes
             └──────────────┘      └──────────────┘
            (gemma-4-prefill)     (gemma-4-decode)
```

---

## Step 1: Provision the GKE Cluster with 2-GPU Node Pools

Create the GKE cluster and provision dedicated node pools featuring `g2-standard-24` VM instances (2x L4 GPUs per node).

```bash
# 1. Create the base GKE Cluster
gcloud container clusters create gemma4-serving-cluster \
    --zone us-central1-a \
    --num-nodes 1 \
    --machine-type e2-standard-4

# 2. Create the Prefill GPU pool (1 node of g2-standard-24 = 2x L4 GPUs)
gcloud container node-pools create prefill-pool \
    --cluster gemma4-serving-cluster \
    --zone us-central1-a \
    --num-nodes 1 \
    --machine-type g2-standard-24 \
    --accelerator type=nvidia-l4,count=2

# 3. Create the Decode GPU pool (1 node of g2-standard-24 = 2x L4 GPUs)
gcloud container node-pools create decode-pool \
    --cluster gemma4-serving-cluster \
    --zone us-central1-a \
    --num-nodes 1 \
    --machine-type g2-standard-24 \
    --accelerator type=nvidia-l4,count=2
```

---

## Step 2: Configure the LWS serving manifest (`gemma-4-serving-lws.yaml`)

Since all 2 GPUs are local, we set LWS **`size: 1`** (only leader pod, no worker template is defined).

See details in [`gemma-4-serving-lws.yaml`](gemma-4-serving-lws.yaml).

---

## Step 3: Deploy the Global Router and Configmap

To coordinate disaggregated serving, you must deploy the Global Router proxy.
The router intercepts completions requests, overrides `max_tokens = 1` for the
prefill engine to execute prompt processing in the background, and forwards the
full request to the decode engine to generate tokens.

1. Apply the router configmap containing the FastAPI routing script:
   ```bash
   kubectl apply -f gemma-router-configmap.yaml
   ```

2. Apply the global router service and deployment:
   ```bash
   kubectl apply -f gemma-global-router.yaml
   ```

---

## Step 4: Complete a Chat completion (Validating the serving)

1. Establish a local port-forward to the router service:
   ```bash
   kubectl port-forward svc/gemma-router-svc 8080:8080
   ```

2. Send a completions request:
   ```bash
   curl -s -N http://localhost:8080/v1/chat/completions \
     -H "Content-Type: application/json" \
     -d '{
       "model": "google/gemma-4-12b-it",
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

