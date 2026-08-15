# Gemma 4 Disaggregated Serving: Architecture Comparison

This document details the architectural migration from custom sidecar routing (old iteration)
to GKE-native Gateway API and `llm-d` EPP (current iteration), explaining how complexities
were simplified and how requests flow through the new system.

---

## 1. Architectural Comparison Summary

| Feature / Component | Custom Sidecar Routing (`gemma-4-disagg-multi-nodes`) | GKE Native Gateway API (`gemma-4-disagg-multi-nodes-llmd`) |
| :--- | :--- | :--- |
| **HTTP Routing Engine** | Custom FastAPI: [router:L15-106][r_o1], [config:L13-100][r_o2]. | GCP L7 LB: [routing.yaml:L1-18][r_n1], [L51-72][r_n2]. |
| **Routing / Scheduling** | Hardcoded in FastAPI: [config:L48-97][s_o1]. | **Endpoint Picker (EPP)**: [routing.yaml:L74-113][s_n1], [L33-36][s_n2]. |
| **Model Node Tracking** | Headless DNS: [serving-lws.yaml:L1-15][t_o1], [router:L38-47][t_o2]. | GKE **`InferencePool`** in [routing.yaml:L20-37][t_n1] (NEGs). |
| **Health Checking** | Probes: [serving-lws.yaml:L90-97][h_o1], loops: [router:L48-56][h_o2]. | GKE **`HealthCheckPolicy`** in [routing.yaml:L157-174][h_n1] (/health). |
| **P2P Sidechannel Config** | Custom Pod IP env vars: [serving-lws.yaml:L63-69][p_o1], [L240-246][p_o2]. | Headless Services (port 5600): [serving-lws.yaml:L1-19][p_n1], [L175-193][p_n2]. |
| **Deployment Footprint** | 5 files, including custom Docker configmaps and routers. | 3 files, completely using standard, native resources. |

---

## 2. Who is Managing the Complexities Now

By migrating to the Gateway API, we eliminated all custom python sidecar proxies and scripts. The responsibilities are now distributed to robust infrastructure controllers:

```mermaid
graph TD
    A[Complexity: Request Routing] -->|Delegated to| B(Google Cloud L7 Load Balancer)
    C[Complexity: Scheduling & Phase Split] -->|Delegated to| D(llm-d Endpoint Picker Pod - EPP)
    E[Complexity: Pod IP Discovery] -->|Delegated to| F(GKE InferencePool NEG Controller)
    G[Complexity: Load Balancer Health] -->|Delegated to| H(GCP HealthCheckPolicy Prober)
    I[Complexity: Sidechannel Routing] -->|Delegated to| J(Kubernetes Headless Services on port 5600)
```

1. **Google Cloud L7 Load Balancer:** Handles incoming HTTP client requests, SSL termination, and routes requests according to rules defined in `HTTPRoute`.
2. **llm-d Endpoint Picker (EPP):** Contains the core logic for disaggregated serving.
   It queries pod roles (`prefill`/`decode`) and health statistics (Prometheus metrics)
   to decide which exact pod should handle prompt processing and who should handle decoding.
3. **GKE InferencePool NEG Controller:** Dynamically registers running vLLM pods into Google Cloud Network Endpoint Groups (NEGs). The load balancer routes traffic directly to these NEGs.
4. **GCP HealthCheckPolicy Prober:** Automatically queries `/health` on all backend endpoints, stripping out unhealthy instances without needing manual readiness script interventions.

---

## 3. End-to-End Request Flow (llmd Iteration)

The following flow chart illustrates how a client completions request is processed through the GKE-native Gateway API and `llm-d` architecture:

```mermaid
sequenceDiagram
    autonumber
    actor Client as Local Terminal
    participant Socat as socat Proxy Pod (8080:80)
    participant Gateway as GKE Load Balancer (10.128.0.43)
    participant EPP as llm-d EPP (Port 9002)
    participant Prefill as Prefill Leader (10.48.1.7)
    participant Decode as Decode Leader (10.48.5.19)

    Client->>Socat: POST /v1/chat/completions (via localhost:8888 tunnel)
    Socat->>Gateway: Forward HTTP Request to LB IP
    Note over Gateway: Gateway intercepts request and triggers ext-proc callout
    Gateway->>EPP: gRPC ext-proc callout (Request Metadata)
    Note over EPP: EPP evaluates active pools & chooses Prefill Leader for prompt phase
    EPP-->>Gateway: Route to Prefill Leader
    Gateway->>Prefill: Forward request to vLLM api-server (port 8000)

    rect rgb(240, 240, 245)
        Note over Prefill: Prefill runs prompt phase (Warm-up / Engine execution)
        Prefill-->>Decode: Push KV Cache via Nixl over TCP port 5600
        Note over Decode: Decode pulls cache blocks into local GPU memory
    end

    Prefill-->>Gateway: Response: Prompt tokens processed (Initiate Stream)
    Gateway-->>Socat: Stream response back
    Socat-->>Client: Stream print tokens

    Note over Gateway: Subsequent token generations route to Decode
    Gateway->>EPP: gRPC check for decoding phase
    EPP-->>Gateway: Route to Decode Leader
    Gateway->>Decode: Run token generation iterations (port 8000)
    Decode-->>Gateway: Stream tokens
    Gateway-->>Client: Final streamed output
```

---

## 4. Kubernetes Control Plane & Resource Association Flow

This section details how the Kubernetes resources defined in [llmd-routing.yaml](llmd-routing.yaml)
and [gemma-serving-lws.yaml](gemma-serving-lws.yaml) are configured and associated by GKE controllers
to establish the data path.

### Resource Relationship Diagram

The GKE Gateway controller and GKE InferencePool controller dynamically discover resources and translate them into Google Cloud L7 Load Balancer components:

```mermaid
graph TD
    %% Kubernetes Custom Resources
    GW["Gateway (gemma-gateway)<br/>[gke-l7-rilb]"] -->|Triggers creation of| GLB["GCP Regional L7 Load Balancer"]
    Route["HTTPRoute (gemma-4-route)"] -->|Binds to| GW
    Route -->|Routes backendRef to| Pool["InferencePool (gemma-4-pool)"]

    Pool -->|Discovers pods using| Selector["Pod Selector: app=gemma-serving"]
    Pool -->|Registers endpoints into| NEG["Network Endpoint Groups (NEGs)"]
    Pool -->|References for scheduling| EPP["Endpoint Picker (gemma-4-epp)"]

    HCP["HealthCheckPolicy"] -->|Overrides probe config on| Pool
    HCP -->|Configures GCP health check path| HC["/health on port 8000"]

    %% Target Pods representation
    subgraph Pods ["Gemma-4 LWS Pods (app: gemma-serving)"]
        direction TB
        PL["gemma-prefill-0 (Prefill Leader, Rank 0)"]
        PW["gemma-prefill-1 (Prefill Worker, Rank 1)"]
        DL["gemma-decode-0 (Decode Leader, Rank 0)"]
        DW["gemma-decode-1 (Decode Worker, Rank 1)"]
    end

    Selector --> Pods
    NEG -->|Maps to IPs of| Pods

    %% EPP Callback Integration
    GLB -->|gRPC ext-proc callout| EPP
    EPP -->|1. Watches endpoints & roles| Pool
    EPP -->|2. Selects target leader pod| PL
    EPP -->|3. Selects target leader pod| DL

    %% Nixl sidechannel P2P connection
    PL <-->|"P2P KV Transfer (Port 5600)"| DL
    PW <-->|"P2P KV Transfer (Port 5600)"| DW
```

### Detailed Description of Control Plane Interactions (Diagram Edges)

The lines (edges) in the relationship diagram above represent critical integration points that bind
the Kubernetes control plane to the physical GCP Load Balancer routing logic:

1. **`Gateway` $\rightarrow$ `GCP Regional L7 Load Balancer` (Provisioning):**
   GKE's Gateway Controller watches the `Gateway` resource with class `gke-l7-rilb`. It communicates with the Google Cloud Compute API to provision a regional, Envoy-managed Internal Application Load Balancer.

2. **`HTTPRoute` $\rightarrow$ `Gateway` (Binding):**
   The `HTTPRoute` binds its routing rules to the Gateway's listeners. Any HTTP traffic reaching the Load Balancer IP matching the path prefix rules (e.g., `/v1`) is forwarded to the designated backend.

3. **`InferencePool` $\rightarrow$ `NEG` (Endpoint Registration):**
   GKE's InferencePool controller automatically creates GCP **Network Endpoint Groups (NEGs)**.
   The controller watches the pod endpoints matching the selector `app: gemma-serving`. It
   dynamically registers the pod IP addresses and port `8000` as network endpoints in the NEGs,
   enabling the GCP Load Balancer to route traffic directly to the container interfaces
   bypassing Kubernetes kube-proxy.

4. **`HealthCheckPolicy` $\rightarrow$ `InferencePool` (Probe Customization):**
   GKE's Gateway controller translates `HealthCheckPolicy` specifications into GCP load balancer
   backend health checkers. It overrides the default Envoy health check probe to target the path
   `/health` on port `8000`, ensuring the load balancer only routes to active vLLM engines.

5. **`GCP Regional L7 Load Balancer` $\rightarrow$ `EPP` (gRPC ext-proc callout):**
   The load balancer resolves the EPP's address indirectly. The `HTTPRoute` specifies
   `InferencePool` as its backendRef. In turn, the `InferencePool` contains `spec.endpointPickerRef`
   pointing to the `gemma-4-epp` Service. GKE's load balancer controller translates this
   reference chain into an **Envoy External Processing (ext-proc)** filter on the Regional L7 Load
   Balancer, instructing Envoy to pause incoming client HTTP requests and execute gRPC callouts
   to EPP port `9002` for routing decisions.

6. **`EPP` $\rightarrow$ `InferencePool` (State Monitoring):**
   The EPP continuously monitors the InferencePool CRD, querying pod liveness and checking their GKE/LWS annotations to separate `prefill-leader`/`decode-leader` targets from stateless compute worker pods.

7. **`EPP` $\rightarrow$ `Prefill Leader` / `Decode Leader` (Scheduling Decisions):**
   Based on the request state (prefill phase vs subsequent token generation/decode phase),
   the EPP responds to the load balancer's ext-proc callback, instructing the load balancer
   to override the routing header and forward the request to the specific NEG endpoint of the
   chosen leader pod (`gemma-prefill-0` or `gemma-decode-0`).

### Control Plane Interaction & Initialization Sequence

The sequence below illustrates how the GKE control plane initializes when the manifests are applied:

```mermaid
sequenceDiagram
    autonumber
    participant Dev as kubectl apply
    participant K8s as K8s API Server
    participant GGW as GKE Gateway Controller
    participant GIP as GKE InferencePool Controller
    participant EPP as llm-d Endpoint Picker
    participant GCP as Google Cloud (Compute Engine)

    Dev->>K8s: Apply Gateway, HTTPRoute, InferencePool, EPP, HealthCheckPolicy

    K8s->>GGW: Watch: Gateway (gemma-gateway) applied
    GGW->>GCP: Provision Regional Internal L7 Load Balancer (Envoy-based)
    GCP-->>GGW: Load Balancer IP allocated (10.128.0.43)

    K8s->>GIP: Watch: InferencePool (gemma-4-pool) applied
    GIP->>K8s: Select pods matching label "app: gemma-serving"
    K8s-->>GIP: Return Prefill and Decode pod endpoints
    GIP->>GCP: Create Network Endpoint Groups (NEGs) containing Pod IPs
    GIP->>K8s: Associate NEGs with InferencePool status

    K8s->>GGW: Watch: HTTPRoute (gemma-4-route) referencing InferencePool
    GGW->>GCP: Configure Load Balancer URL map to forward /v1 prefix to NEGs

    K8s->>GIP: Watch: HealthCheckPolicy applied
    GIP->>GCP: Update Load Balancer BackendService with custom probe (path=/health, port=8000)

    K8s->>EPP: Start Endpoint Picker Pod
    EPP->>K8s: Watch: InferencePool endpoints and GKE pod roles

    Note over GCP,EPP: Setup complete. Load Balancer is now configured to call EPP via gRPC ext-proc callback on incoming client requests.
```

[r_o1]: ../gemma-4-disagg-multi-nodes/gemma-global-router.yaml#L15-106
[r_o2]: ../gemma-4-disagg-multi-nodes/gemma-router-configmap.yaml#L13-100
[r_n1]: llmd-routing.yaml#L1-L18
[r_n2]: llmd-routing.yaml#L51-L72

[s_o1]: ../gemma-4-disagg-multi-nodes/gemma-router-configmap.yaml#L48-97
[s_n1]: llmd-routing.yaml#L74-L113
[s_n2]: llmd-routing.yaml#L33-L36

[t_o1]: ../gemma-4-disagg-multi-nodes/gemma-serving-lws.yaml#L1-15
[t_o2]: ../gemma-4-disagg-multi-nodes/gemma-global-router.yaml#L38-47
[t_n1]: llmd-routing.yaml#L20-L37

[h_o1]: ../gemma-4-disagg-multi-nodes/gemma-serving-lws.yaml#L90-97
[h_o2]: ../gemma-4-disagg-multi-nodes/gemma-global-router.yaml#L48-56
[h_n1]: llmd-routing.yaml#L157-L174

[p_o1]: ../gemma-4-disagg-multi-nodes/gemma-serving-lws.yaml#L63-69
[p_o2]: ../gemma-4-disagg-multi-nodes/gemma-serving-lws.yaml#L240-246
[p_n1]: gemma-serving-lws.yaml#L1-19
[p_n2]: gemma-serving-lws.yaml#L175-193
