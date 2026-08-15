# Bootstrapping llm-d on GKE

This guide outlines the steps required to install the `llm-d` control plane operators and
Gateway API extensions on a GKE cluster. These components must be installed before applying
the `llmd-routing.yaml` manifests.

---

## Installation Prerequisites

* A GKE cluster running Kubernetes **v1.30+**.
* `kubectl` authenticated with cluster-admin privileges.

---

## 1. Enable GKE Gateway API Addon

To populate GKE's native Google-managed GatewayClasses (like `gke-l7-rilb`),
you must enable the Gateway API addon on your cluster:

```bash
# Update cluster to enable Gateway API
gcloud container clusters update gemma4-serving-cluster \
    --zone us-central1-a \
    --gateway-api=standard
```

---

## 2. Install Gateway API Inference Extension CRDs

Install the inference specification schemas (which define `InferencePool` and `InferenceModel` resources):

```bash
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api-inference-extension/releases/download/v1.5.0/manifests.yaml
```

---

## 3. Deploy Gemma 4 serving workloads

Once the Gateway API and Inference Extension CRDs are installed, GKE's managed load balancer controller
(`gke-l7-rilb`) handles the scheduling and endpoint picking natively. You do not
need to install any external Helm charts or operators.

Apply the LeaderWorkerSets and routing rules:

```bash
# Apply serving LeaderWorkerSets
kubectl apply -f gemma-serving-lws.yaml

# Apply InferencePool and HTTPRoute routing rules
kubectl apply -f llmd-routing.yaml
```

---

## 4. GKE Implementation Caveats & Troubleshooting

When implementing this disaggregated serving layout on GKE, ensure the following prerequisites are met:

### 1. Enable Required Cloud APIs

Traffic extensions on GKE Regional Application Load Balancers require the Google Cloud **Network Services API**
(`networkservices.googleapis.com`) to be enabled in your project:

```bash
gcloud services enable networkservices.googleapis.com
```

### 2. Provision Proxy-Only Subnetwork

GKE's Regional Internal Application Load Balancer (`gke-l7-rilb`) requires a dedicated **Proxy-Only Subnet**
in the VPC region to run its Envoy load balancer proxies.

Create it using `gcloud` with a non-overlapping CIDR block (outside `10.128.0.0/9` which is reserved for GKE's auto subnet mode):

```bash
gcloud compute networks subnets create gke-proxy-only-subnet \
    --purpose=REGIONAL_MANAGED_PROXY \
    --role=ACTIVE \
    --region=us-central1 \
    --network=default \
    --range=172.16.128.0/23
```

### 3. Expose Nixl Port in Services

The prefill and decode engines perform a P2P sidechannel connection to coordinate KV Cache block handshakes.
Ensure port **`5600`** is declared in the `ports` block of both prefill and decode headless discovery Services,
otherwise the Kubernetes networking layer will block the handshake.

### 4. Configure HealthCheckPolicy

By default, GKE Gateway health checks probe the root path (`/`) on port 8000. Since vLLM API server
returns `404 Not Found` on `/` and only returns `200 OK` on `/health`, you must attach a GKE-native
`HealthCheckPolicy` to the `InferencePool` to override the path to `/health`.

### 5. Use the llm-d Endpoint Picker Image

Standard Gateway API Inference Extension EPP images do not support disaggregated serving request
phases (sending prompts to prefillers first, and tokens to decoders). You must deploy the custom
`llm-d` image: `ghcr.io/llm-d/llm-d-router-endpoint-picker:v0.9.0`

---

## 5. Testing from Local Terminal

Because GKE's Internal Regional Load Balancer allocates a private IP address dynamically, it is
not directly reachable from a local terminal outside the VPC network.

1. **Retrieve the allocated Gateway IP:**
   Query the Gateway to find the private IP address:
   ```bash
   kubectl get gateway gemma-gateway
   ```
   Or save it directly to an environment variable:
   ```bash
   export GW_IP=$(kubectl get gateway gemma-gateway -o jsonpath='{.status.addresses[0].value}')
   echo "Gateway IP: $GW_IP"
   ```

2. **Deploy a local port-forwarding proxy pod:**
   Deploy a temporary `socat` pod inside the cluster to bridge traffic to the retrieved `$GW_IP`:
   ```bash
   kubectl run gateway-proxy \
       --image=alpine/socat \
       --expose --port=8080 \
       -- tcp-listen:8080,fork,reuseaddr tcp-connect:$GW_IP:80
   ```

3. **Establish a local port-forward tunnel:**
   Use a local port like **`8888`** to avoid conflicts with other local proxy processes (such as port `8080` which is commonly used):
   ```bash
   kubectl port-forward pod/gateway-proxy 8888:8080
   ```

4. **Query the completions endpoint from your local terminal:**
   Open a separate local shell window and execute `curl` with streaming enabled and pipe it to a python parser for clean, human-readable console rendering:
   ```bash
   curl -s -N http://localhost:8888/v1/chat/completions \
     -H "Content-Type: application/json" \
     -d '{
       "model": "google/gemma-4-12b-it",
       "messages": [
         {"role": "user", "content": "Tell me a short 1-sentence story about space."}
       ],
       "max_tokens": 50,
       "temperature": 0.0,
       "stream": true
     }' | python3 -c '
import sys, json
for line in sys.stdin:
    if line.startswith("data: ") and "[DONE]" not in line:
        try:
            chunk = json.loads(line[6:])
            content = chunk["choices"][0]["delta"].get("content", "")
            print(content, end="", flush=True)
        except Exception:
            pass
print()
'
   ```


