# Summary of Gemma 4 Disaggregated Serving (TP=2) Deployment on GKE

This document summarizes the architecture, configurations, and workarounds implemented to successfully host **Gemma 4 (12B IT)** using disaggregated serving and **Tensor Parallelism (TP=2)** across multiple GKE nodes.

---

## 1. Request Flow & Multi-Node Architecture

In this deployment, both the Prefill and Decode clusters are configured as **LeaderWorkerSets (LWS)** of size 2, sharding the model using **TP=2**. Because each GPU node in the pool hosts 1x NVIDIA L4 GPU, the two tensor parallel workers reside on separate physical VM nodes:

```
                      [ Client Request ]
                              │
                              ▼
                    [ Global Router Proxy ]
                       │              │
                       ▼ (Prefill)    ▼ (Decode)
             [ Prefill LWS ]        [ Decode LWS ]
              ┌───────────┐          ┌───────────┐
  Rank 0 ──►  │Leader Pod │  ──NCCL─►│Leader Pod │   ◄─── Node A (L4 GPU, Ray Head)
              └─────┬─────┘          └─────┬─────┘
                    │                      │
              ┌─────▼─────┐          ┌─────▼─────┐
  Rank 1 ──►  │Worker Pod │  ──NCCL─►│Worker Pod │   ◄─── Node B (L4 GPU, Ray Worker)
              └───────────┘          └───────────┘
```

1.  **Ray Cluster:** The LWS Leader pod starts the Ray Head. The LWS Worker pod starts the Ray Worker process and joins the leader's GCS registry to form a single virtual Ray execution group of 2 GPUs.
2.  **Disaggregated KV Transfer:**
    *   **TP Rank 0 P2P Channel:** `gemma-prefill-0` (prefill leader) connects to `gemma-decode-0` (decode leader) over TCP port `14579`.
    *   **TP Rank 1 P2P Channel:** `gemma-prefill-0-1` (prefill worker) connects to `gemma-decode-0-1` (decode worker) over TCP port `14580`.

---

## 2. Key Manifest Configurations

The following configurations were applied to [gemma-serving-lws.yaml](file:///usr/local/google/home/daniellguo/git-repos/tech-notes/k8s/explore-leaderworkerset/resources/gemma-4-disagg-multi-nodes/gemma-serving-lws.yaml):

| Parameter | Type | Value / Scope | Purpose |
| :--- | :--- | :--- | :--- |
| `VLLM_USE_MQ_BROADCASTER` | Env Var | `"false"` | Disables shared memory broadcasting. Shared memory cannot cross VM node boundaries. |
| `VLLM_USE_RAY_V2_EXECUTOR_BACKEND` | Env Var | `"0"` | Disables the shared-memory-reliant Ray V2 executor and falls back to Ray V1 socket-based executor. |
| `VLLM_DISABLE_REQUEST_ID_RANDOMIZATION` | Env Var | `"1"` | Bypasses vLLM request ID suffix randomization so prefill and decode use matching IDs. |
| `NCCL_P2P_DISABLE` | Env Var | `"1"` | Disables GPUDirect P2P over network since L4 GPUs reside on separate physical nodes. |
| `NCCL_NET_GDR_LEVEL` | Env Var | `"0"` | Forces NCCL transport over standard TCP sockets instead of RDMA. |
| `NCCL_SOCKET_IFNAME` | Env Var | `"eth0"` | Binds NCCL traffic to default GKE Pod network interfaces. |

---

## 3. Critical Workarounds & Code Patches

During verification, several blockers were identified and resolved via container startup patches:

### 1. MQ Broadcaster Hang Bypass
> [!IMPORTANT]
> **The Issue:**
> vLLM's `GroupCoordinator` inside `parallel_state.py` defaults to creating a shared-memory message queue broadcaster (`MessageQueue`). Since the tensor parallel group spans 2 physical VM nodes, trying to broadcast metadata via shared memory blocks indefinitely.
>
> **The Fix:**
> We patched `/usr/local/lib/python3.12/dist-packages/vllm/distributed/parallel_state.py` to read `VLLM_USE_MQ_BROADCASTER` and fallback to Gloo/NCCL TCP broadcast:
> ```python
> use_message_queue_broadcaster=(os.environ.get("VLLM_USE_MQ_BROADCASTER", "true").lower() == "true")
> ```

### 2. LWS Worker Hostname Resolution
> [!IMPORTANT]
> **The Issue:**
> vLLM's `p2p_nccl_connector.py` calculates peer worker hostnames by replacing `"-0"` (leader suffix) with the TP rank index (e.g. `"-1"`). However, in GKE LeaderWorkerSet, worker pods are named `<lws-name>-0-<rank>` (e.g. `gemma-prefill-0-1`), resulting in resolution failures and connection hangs.
>
> **The Fix:**
> We patched `/usr/local/lib/python3.12/dist-packages/vllm/distributed/kv_transfer/kv_connector/v1/p2p/p2p_nccl_connector.py` to replace `"-0"` with `"-0-{rank}"` instead:
> ```python
> remote_host = ip if self._rank == 0 else ip.replace("-0", f"-0-{self._rank}")
> ```

### 3. Hugging Face Lock Cleanup
> [!NOTE]
> **The Issue:**
> Stale `.lock` files inside GKE host node path caches block Hugging Face downloads on pod recreation. This causes the main engine initialization thread to hang silently inside `AutoTokenizer.from_pretrained()`.
>
> **The Fix:**
> Cleaned up the stale locks from GKE host mounts and copied cached model tokenizer assets directly to skip the network check on startup.

### 4. Ray V2 Executor Backend Fallback
> [!IMPORTANT]
> **The Issue:**
> The new vLLM V1 core defaults to `RayExecutorV2` which instantiates shared-memory message queue broadcasters (`rpc_broadcast_mq`) for worker coordination. Across multiple physical nodes, these queues fail to connect, deadlocking the engine during initialization.
>
> **The Fix:**
> Set environment variable `VLLM_USE_RAY_V2_EXECUTOR_BACKEND="0"` in the container environment variables to fall back to the stable `RayDistributedExecutor` (which coordinates via Ray actors and TCP/NCCL network sockets).

### 5. GCS Dashboard-Free Node Counting Check
> [!IMPORTANT]
> **The Issue:**
> The leader container's wait loop previously used the `/usr/local/bin/ray status` command to wait until the worker joined the Ray cluster. However, this command queries the dashboard server API on port 8265, which requires the dashboard dependencies of `ray[default]`. Since we install standard bare `ray`, the command crashed and returned `0` active nodes, causing the wait loops to hang indefinitely.
>
> **The Fix:**
> Switched to a robust Python GCS registry query (`ray.nodes()`) that checks for alive Ray cluster nodes directly from GCS without relying on the dashboard server.

---

## 4. Verification Output

We validated end-to-end serving by executing a chat completions query through the global proxy router:

```bash
kubectl run curl-test --image=curlimages/curl --restart=Never --rm -i -- \
  -X POST -H "Content-Type: application/json" \
  -d '{"model": "google/gemma-4-12b-it", "messages": [{"role": "user", "content": "What is the capital of France?"}], "max_tokens": 15}' \
  http://gemma-router-svc.default.svc.cluster.local:8080/v1/chat/completions
```

**Response JSON received successfully:**
```json
{
  "id": "chatcmpl-___prefill_addr_gemma-prefill-0.gemma-prefill:14579___decode_addr_gemma-decode-0.gemma-decode:14579_e282541cc97241e4af9b4470580ea45d",
  "object": "chat.completion",
  "created": 1784260760,
  "model": "google/gemma-4-12b-it",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "The capital of France is Paris.",
        "refusal": null
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "total_tokens": 28,
    "completion_tokens": 8
  }
}
```

---

## 5. Automating Readiness via Kubernetes Probes & Workarounds

To automate inference readiness checks natively inside the GKE cluster, the following configuration layers have been added to the manifests:

### 1. Prefill / Decode Leaders (`gemma-serving-lws.yaml`)
Added native `readinessProbe` blocks to monitor vLLM's `/health` endpoint:
```yaml
# Configured under spec.containers[name=prefill-leader / decode-leader]
readinessProbe:
  httpGet:
    path: /health
    port: 8000
  initialDelaySeconds: 30  # Prefill (60 for Decode)
  periodSeconds: 10        # Prefill (15 for Decode)
  timeoutSeconds: 5
  failureThreshold: 20
```

### 2. Service DNS Workaround (`publishNotReadyAddresses`)
* **The Blocker:** By default, CoreDNS only resolves headless service DNS entries (`gemma-decode-0.gemma-decode...`) once the pod is marked `Ready`. This created a circular deadlock: worker pods could not resolve the head IP to join Ray -> Ray cluster could not form -> vLLM could not warm up -> pod could not become `Ready`.
* **The Fix:** Configured `publishNotReadyAddresses: true` on both the prefill and decode headless services. Pod IPs are now published to DNS immediately on creation, allowing Ray nodes to bootstrap successfully.

### 3. Global Router Checks (`gemma-global-router.yaml`)
Modified the router's bash startup loop to query backend readiness `/health` directly via `urllib.request` before spawning the fastapi proxy server. This prevents the proxy from crash-looping during engine weight downloads:
```bash
# Wait until both backend engines respond with 200 OK
PREFILL_READY=$(python3 -c "import urllib.request; print(1 if urllib.request.urlopen('http://$PREFILL_IP:8000/health').getcode() == 200 else 0)" 2>/dev/null)
DECODE_READY=$(python3 -c "import urllib.request; print(1 if urllib.request.urlopen('http://$DECODE_IP:8000/health').getcode() == 200 else 0)" 2>/dev/null)
```
The router's own readiness probe evaluates `/status` on port 8080 and only exposes the ingress once the proxy successfully connects.

---

## 6. Conceptual Deep Dive & FAQ

### Q1: What is the division of labor between Leader and Worker pods in the LWS?
* **Leader Pod (`Rank 0`):** Acts as the cluster coordinator (runs Ray Head/GCS) and the external client gateway (exposes public HTTP ports). It handles tokenizer execution, request batching, and downloads the config files. It also runs GPU calculations for the Rank 0 tensor parallel shards.
* **Worker Pod (`Rank 1`):** Acts as a stateless compute node. It registers itself to the Ray Head and dedicates its GPU resources solely to executing the Rank 1 tensor parallel shards.

### Q2: Do both pods download the complete model weights?
* **Yes.** Because the leader and worker pods run on different physical VM instances in GKE, they have separate local disk partitions (`hostPath` volumes).
* vLLM's model weight sharding happens *in-memory* during engine startup. Both nodes must download the complete `.safetensors` files to local disk so that their respective loading threads can read the layer files and extract their rank's sharded parameters.

### Q3: Why must the Leader and Worker continuously exchange intermediate results during KV Cache generation?
To understand why they need to exchange results constantly to generate the KV Cache, let's look at the mathematical pipeline of a single Layer and see exactly where the **All-Reduce** (exchange) happens.

Here is the step-by-step process of how **Layer 1** and **Layer 2** generate their KV Caches:

#### Step 1: Layer 1 Starts (Initial Prompt Input)
The prompt text enters the model. Let's call the input vector $X$. Both GPUs have a copy of the input $X$.

1. **Calculating KV Cache (Layer 1):**
   * **Leader GPU (Rank 0):** Multiplies input $X$ by its half of the weights matrix to generate the KV Cache for **Heads 1–4**.
   * **Worker GPU (Rank 1):** Multiplies input $X$ by its half of the weights matrix to generate the KV Cache for **Heads 5–8**.
   
   *At this moment, both GPUs have generated their 50% slice of the Layer 1 KV Cache. No exchange has happened yet.*

2. **Calculating the output of Layer 1:**
   To finish Layer 1, the model has to run the Feed-Forward (MLP) layers.
   * Rank 0 calculates its partial output: $Out_0$.
   * Rank 1 calculates its partial output: $Out_1$.
   * **The All-Reduce (Exchange):** Rank 0 and Rank 1 send their outputs to each other and add them together:
     $$Layer_1\_Output = Out_0 + Out_1$$
     
     *Both GPUs now hold the complete, unified $Layer_1\_Output$ vector.*

#### Step 2: Layer 2 Starts (Here is the Dependency!)
To generate the KV Cache for **Layer 2**, the input is no longer the original prompt $X$. **The input is now $Layer_1\_Output$.**

1. **Calculating KV Cache (Layer 2):**
   * **Leader GPU (Rank 0):** Multiplies $Layer_1\_Output$ by its weights to generate the Layer 2 KV Cache for **Heads 1–4**.
   * **Worker GPU (Rank 1):** Multiplies $Layer_1\_Output$ by its weights to generate the Layer 2 KV Cache for **Heads 5–8**.

#### The "Aha!" Moment: Why they must exchange
Look closely at **Step 2**:
To calculate its Layer 2 KV Cache, the Leader GPU (Rank 0) needs the full $Layer_1\_Output$. 

But $Layer_1\_Output$ is composed of:
$$Out_0 \text{ (from Leader)} + Out_1 \text{ (from Worker)}$$

If the Leader and Worker did **not** exchange their intermediate results at the end of Layer 1:
* The Leader would not have $Out_1$.
* The Leader would have to calculate Layer 2 using only $Out_0$ (which is incorrect and incomplete).
* The resulting Layer 2 KV Cache would be completely wrong.

#### Mapping back to the Portrait Painting Analogy:
* **Layer 1 Output:** The unified Face Outline.
* **Layer 2 KV Cache:** The Left Eye (Rank 0) and the Right Eye (Rank 1).
* **The Dependency:** You cannot place the left eye in the correct position (Layer 2 KV Cache) unless you know where the right side of the face outline ended. You must align the outline first.

### Q4: Does either pod ever hold the complete KV Cache?
No, **neither the leader nor the worker will ever have the complete KV Cache.** 

Each pod only holds **its own 50% slice** of the KV Cache. 

Here is how the KV Cache is split and why they never need to merge it:

#### 1. How the KV Cache is Split (By Attention Heads)
In modern AI models, the KV Cache is generated by the **Attention layers**. The model uses multiple "Attention Heads" (like multiple parallel eyes looking at the text). 

For example, Gemma-4-12b has **8 Key/Value Attention Heads**. When we run `TP=2`:
* **Leader Pod (Rank 0):** Calculates and stores the KV Cache only for **Heads 1 to 4** (its 50% portion).
* **Worker Pod (Rank 1):** Calculates and stores the KV Cache only for **Heads 5 to 8** (its 50% portion).

They do not share these caches during the prefill calculation, because Rank 0's GPU only handles calculations for Heads 1–4, and Rank 1's GPU only handles Heads 5–8. 

#### 2. Direct parallel transfer to the Decode Pods
Because we are running the Decode cluster with `TP=2` as well:
* The **Decode Leader Pod** also only needs the KV Cache for **Heads 1 to 4**.
* The **Decode Worker Pod** also only needs the KV Cache for **Heads 5 to 8**.

This creates a perfect match:

```
PREFILL LAYER (TP=2)                        DECODE LAYER (TP=2)

Leader Pod (Rank 0)   ====================> Decode Leader Pod (Rank 0)
[Heads 1-4 Cache]       (Port 14579 TCP)    [Heads 1-4 Cache]

Worker Pod (Rank 1)   ====================> Decode Worker Pod (Rank 1)
[Heads 5-8 Cache]       (Port 14580 TCP)    [Heads 5-8 Cache]
```

At no point does the KV Cache need to be merged into a single complete block. This is the main reason why disaggregated serving is so fast and efficient—we avoid the overhead of copying and combining massive datasets across the network.



### Q5: What networking technologies are used for inter-node communication?
* **NCCL (NVIDIA Collective Communications Library):** The industry-standard software library offering highly optimized collective primitives (All-Reduce, Broadcast) for NVIDIA GPUs.
* **RDMA (Remote Direct Memory Access):** A hardware network capability (utilizing InfiniBand or RoCE) allowing network cards to transfer VRAM data directly between physical servers without CPU/operating system kernel overhead. High-performance production clusters run **NCCL over GPUDirect RDMA** to minimize communication latency.


