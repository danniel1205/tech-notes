# LLMs for Systems People: Core Concepts & Reading Guide

This guide breaks down the core concepts in the foundational paper **"Attention Is All You Need" (Vaswani et al., 2017)** through the lens of a systems / Kubernetes engineer.

---

## 1. The Fundamental Problem: How Computers Read Text

At its core, a language model does one basic job: **Predict the next word** given the previous words.
For example: given `"The cloud engineer deployed a Kubernetes ..."`, predict `"cluster"`.

To predict the next word accurately, the model must understand the context and relationships between words:

- In *"The bank of the river"*, "bank" means land.
- In *"The bank approved the loan"*, "bank" means a financial institution.
- In *"The server crashed because it ran out of memory"*, what does *"it"* refer to? The server.

### The Old Approach: Single-Threaded Stream Processing

Before Transformers, sequence processing worked like a single-threaded sequential loop:

- You read word 1, update an internal memory buffer.
- You read word 2, update the memory buffer with word 2.
- ...
- You read word 100, update the memory buffer.

**Why this was a massive systems bottleneck:**

1. **Zero Hardware Parallelism:** To process word 100, you *must* wait for word 99 to finish.
   Modern GPUs have thousands of compute cores. A sequential loop leaves 99% of GPU cores idle!
2. **Lossy Memory Buffer:** By the time you reach word 500, the early context has been compressed
   and diluted across hundreds of updates. The model "forgets" what happened at word 1.

### The Transformer Breakthrough: Parallel Direct Lookups

In 2017, the paper **"Attention Is All You Need"** proposed: **Throw away the sequential loop entirely.**

- Take all words in your prompt at once (e.g., all 1,000 words).
- Put them into memory simultaneously.
- Let **every word directly inspect every other word in parallel** using a matrix multiplication.

This mechanism is called **Attention**.
Because all pairs of words can be checked at the same time, this computation perfectly matches
how GPUs work: thousands of cores crunching massive parallel matrix operations.

---

## 2. From Text to Numbers: Embeddings & Positions

Before an LLM can perform inference on hardware, input text must be converted into numerical coordinates that GPU tensor cores can process.

```text
[Raw Text: "what is quantum computing"]
            │
            ▼ (Runs on Host CPU)
   [ 1. Tokenizer (BPE) ] ──> Integer IDs: [10919, 374, 18585, 9662]
            │
            ▼ (Transferred over PCIe to GPU VRAM)
   [ 2. Embedding Matrix ] ──> 4 Float Vectors of size d=4096 (x_1, x_2, x_3, x_4)
            │
            ▼
   [ Layer 1 GPU Assembly Line ]
```

### A. Tokenization

* **What it is:** A deterministic program that slices raw text strings into discrete integer IDs based on a fixed vocabulary dictionary (e.g., vocabulary size `V = 128,000` in Llama 3).
* **Why it is needed:**
  * Computers and GPUs run linear algebra (matrix multiplications and dot products). They cannot perform math directly on raw character strings or arbitrary byte sequences.
  * If we mapped individual characters (`'w'`, `'h'`, `'a'`, `'t'`), sequence lengths would be 4–5× longer, causing GPU attention memory and computation to explode quadratically.
  * If we mapped whole words (`"quantum"`, `"computing"`), the dictionary would be infinite (typos, slang, new words, foreign languages).
  * **Subword tokenization (Byte Pair Encoding / BPE)** achieves optimal compression: common words stay whole, while rare or compound words are sliced into small sub-chunks.
  * *Kubernetes analogy:* Raw text is like unparsed YAML text with comments and whitespace. Tokenization is `kubectl`
    parsing and validating text into strongly typed internal enums and integers before the API Server processes it.
* **How Token IDs get generated:**
  1. **Pre-training setup:** The BPE algorithm scans terabytes of text, starts with 256 individual byte values,
     and iteratively merges the most frequently occurring adjacent pairs until it reaches exactly `V` entries.
  2. **Inference runtime:** Runs entirely on the **Host CPU**. The CPU takes `"what is quantum computing"`,
     matches it deterministically against its BPE table, and generates an array of integers:
     `[10919, 374, 18585, 9662]`. These 4 integers are then copied over PCIe to GPU VRAM.

### B. Embedding Matrix (W_E)

* **What it is:** A 2D lookup table stored directly in GPU VRAM of size `[V, d]` (e.g., `128,000 × 4,096` floats).
* **Why it is needed:**
  * An integer ID like `18585` is just a scalar index—it carries zero semantic geometry.
    Token ID `18585` is not mathematically "twice" ID `9292`, and you cannot calculate an angle or dot product between two plain integers.
  * To perform vector math, every token needs coordinates in a high-dimensional space (`d = 4096`). In this space, words with related concepts point in similar directions.
  * *Kubernetes analogy:* A Token ID is like a Pod's UID string. The Embedding Vector is the deserialized `v1.Pod`
    object spec containing CPU/memory requests, labels, tolerations, and affinities that the K8s scheduler needs to make placement calculations.
* **How it is generated:**
  * **Before training:** Initialized with random float coordinates.
  * **During training:** Through backpropagation, gradients nudge the coordinates of every row so that words appearing in similar contexts cluster meaningfully.
  * **During inference:** The matrix is **100% frozen and read-only**. Looking up an embedding on the GPU is an `O(1)` memory fetch: row `18585` yields 4,096 floats.

### C. Positional Encoding

* **Why it is needed:** Because attention computes all pairs simultaneously without an inherent sequence order, it is *permutation-invariant* (shuffling words yields the same raw attention pool).
* **How order is injected:**
  * In the 2017 paper, sinusoidal signals (sine and cosine frequencies) are added to the embedding vectors before feeding them into the layers.
  * *Modern update:* Modern LLMs (such as Llama 3) use **RoPE (Rotary Position Embedding)**, rotating query
    and key vectors in complex space during attention rather than adding fixed positional vectors up front.

---

## 3. The Core Engine: Query, Key, Value (Q, K, V)

The heart of the paper is the **Scaled Dot-Product Attention**.

### The Systems / K8s Analogy: Service Discovery & Label Selectors

Think of attention like querying an in-memory key-value service catalog:

- **Query (Q):** *"What am I looking for?"* (Like a K8s Label Selector: `app=backend, env=prod`).
- **Key (K):** *"What metadata do I advertise?"* (Like K8s Labels on Pods: `app=backend, env=prod`).
- **Value (V):** *"What payload do I actually hold?"* (Like the Pod IP, port, or payload data).

### The Math (Step by Step)

```text
Attention(Q, K, V) = softmax( (Q * K^T) / sqrt(d_k) ) * V
```

1. **Compute Similarity (`Q * K^T`):** Dot product of Query matrix and Key matrix. If Query i matches Key j, their dot product is large.
2. **Scale by `1 / sqrt(d_k)`:** As dimension d_k grows, dot products grow in magnitude.
   Large values push `softmax` into regions with tiny gradients. Scaling preserves numerical stability.
3. **Normalize with Softmax:** Converts scores across the sequence into probabilities that sum to 1.0 (the "attention weights").
4. **Weighted Sum (`* V`):** Multiply the attention weights by Values V. Each token gathers a blend of information from relevant tokens.

---

## 4. Multi-Head Attention (MHA)

Instead of performing one single attention calculation with dimension d_model, the model splits Q, K, V into h separate heads (e.g., 8 heads in the paper, 32 in Llama-3-8B).

- **Why?** Different heads specialize in different relationships (e.g., one head tracks grammatical subject-verb agreement, another tracks pronoun references, another tracks temporal order).
- **Implementation:** Project inputs with linear weight matrices (`W_Q, W_K, W_V`), run attention independently per head in parallel, concatenate results, and project back through `W_O`.

---

## 5. Feed-Forward Networks (FFN / MLP)

After attention, every token vector passes through an identical Feed-Forward Network:

```text
FFN(x) = max(0, x * W_1 + b_1) * W_2 + b_2
```

- In the original paper, the intermediate dimension expands to `4 * d_model` before projecting back down.
- **Systems takeaway:** FFNs contain roughly **two-thirds of the model's total parameters**! While attention decides *where to look*, the FFN acts as the dense memory/knowledge retrieval bank.

---

## 6. Architecture: Encoder-Decoder vs. Decoder-Only

The original 2017 paper was built for machine translation (English -> German), using an **Encoder-Decoder**:

- **Encoder:** Reads the entire input prompt bidirectionally.
- **Decoder:** Generates output text token-by-token, using:
  1. *Masked Self-Attention:* Prevents token i from peeking at future tokens i+1 (causal masking).
  2. *Cross-Attention:* Queries from the decoder look up Keys and Values from the encoder's output.

### Modern Generative LLMs (Llama, GPT, Mistral)

- Modern LLMs are **Decoder-Only**.
- There is no separate encoder and no cross-attention.
- Everything runs through masked self-attention blocks stacked sequentially.

---

## 7. The Systems Angle: Why Infra Engineers Care

This is what connects the 2017 paper to modern AI platform architecture and GPU serving:

### A. Prefill vs. Decode (Two Radically Different Phases)

1. **Prefill (Time To First Token / TTFT):**
  - Takes your whole prompt (e.g., 2,000 tokens) and computes Q, K, V for all tokens simultaneously.
  - Highly parallel matrix-matrix multiplication (GEMM).
  - **Bottleneck:** **Compute-bound** (saturates GPU Tensor Cores).
2. **Decode (Inter-Token Latency / ITL):**
  - Generates output one single token at a time.
  - For each new token, the GPU must stream all weights (e.g., 16 GB for an 8B model in FP16) from High Bandwidth Memory (HBM) into SRAM just to do a tiny vector-matrix multiplication (GEMV).
  - **Bottleneck:** **Memory-bandwidth-bound** (GPU compute cores spend most of their time idling waiting for memory transfers).

### B. Why the KV Cache Exists

- In autoregressive decoding, to compute token N, you need the Keys and Values of all tokens 1 .. N-1.
- **Without KV Cache:** You would have to recompute all prior K and V matrices from scratch at every step (O(N^2) redundant compute!).
- **With KV Cache:** Save previously computed K and V tensors in GPU VRAM. For step N, compute only Q_N, K_N, V_N, append K_N, V_N to the cache, and run attention.
- **Infra Trade-off:** Saves compute, but eats huge amounts of GPU VRAM. As context lengths and concurrent users grow, **KV cache memory consumption limits your max batch size and serving capacity**.

---

## 8. End-to-End Walkthrough: From Text to Generated Tokens

This walkthrough chains together all the concepts into a concrete systems trace, showing how data moves across hardware boundaries.

### Part 1: Before the Model is Ready for Inference

Before an LLM can perform inference on hardware, input text must be converted into numerical coordinates that GPU tensor cores can process.

```text
[Raw Text: "what is quantum computing"]
            │
            ▼ (Runs on Host CPU)
   [ 1. Tokenizer (BPE) ] ──> Integer IDs: [10919, 374, 18585, 9662]
            │
            ▼ (Transferred over PCIe to GPU VRAM)
   [ 2. Embedding Matrix ] ──> 4 Float Vectors of size d=4096 (x_1, x_2, x_3, x_4)
            │
            ▼
   [ Layer 1 GPU Assembly Line ]
```

### A. Tokenization

* **What it is:** A deterministic program that slices raw text strings into discrete integer IDs based on a fixed vocabulary dictionary (e.g., vocabulary size `V = 128,000` in Llama 3).
* **Why it is needed:**
  * Computers and GPUs run linear algebra (matrix multiplications and dot products). They cannot perform math directly on raw character strings or arbitrary byte sequences.
  * If we mapped individual characters (`'w'`, `'h'`, `'a'`, `'t'`), sequence lengths would be 4–5× longer, causing GPU attention memory and computation to explode quadratically.
  * If we mapped whole words (`"quantum"`, `"computing"`), the dictionary would be infinite (typos, slang, new words, foreign languages).
  * **Subword tokenization (Byte Pair Encoding / BPE)** achieves optimal compression: common words stay whole, while rare or compound words are sliced into small sub-chunks.
  * *Kubernetes analogy:* Raw text is like unparsed YAML text with comments and whitespace. Tokenization is
    `kubectl` parsing and validating text into strongly typed internal enums and integers before the API Server processes it.
* **How Token IDs get generated:**
  1. **Pre-training setup:** The BPE algorithm scans terabytes of text, starts with 256 individual byte values,
     and iteratively merges the most frequently occurring adjacent pairs until it reaches exactly `V` entries.
  2. **Inference runtime:** Runs entirely on the **Host CPU**. The CPU takes `"what is quantum computing"`,
     matches it deterministically against its BPE table, and generates an array of integers:
     `[10919, 374, 18585, 9662]`. These 4 integers are then copied over PCIe to GPU VRAM.

### B. Embedding Matrix (W_E)

* **What it is:** A 2D lookup table stored directly in GPU VRAM of size `[V, d]` (e.g., `128,000 × 4,096` floats).
* **Why it is needed:**
  * An integer ID like `18585` is just a scalar index—it carries zero semantic geometry.
    Token ID `18585` is not mathematically "twice" ID `9292`, and you cannot calculate an angle or dot product between two plain integers.
  * To perform vector math, every token needs coordinates in a high-dimensional space (`d = 4096`). In this space, words with related concepts point in similar directions.
  * *Kubernetes analogy:* A Token ID is like a Pod's UID string. The Embedding Vector is the deserialized `v1.Pod`
    object spec containing CPU/memory requests, labels, tolerations, and affinities that the K8s scheduler needs to make placement calculations.
* **How it is generated:**
  * **Before training:** Initialized with random float coordinates.
  * **During training:** Through backpropagation, gradients nudge the coordinates of every row so that words appearing in similar contexts cluster meaningfully.
  * **During inference:** The matrix is **100% frozen and read-only**. Looking up an embedding on the GPU is an `O(1)` memory fetch: row `18585` yields 4,096 floats.

### C. Positional Encoding

* **Why it is needed:** Because attention computes all pairs simultaneously without an inherent sequence order, it is *permutation-invariant* (shuffling words yields the same raw attention pool).
* **How order is injected:**
  * In the 2017 paper, sinusoidal signals (sine and cosine frequencies) are added to the embedding vectors before feeding them into the layers.
  * *Modern update:* Modern LLMs (such as Llama 3) use **RoPE (Rotary Position Embedding)**, rotating query
    and key vectors in complex space during attention rather than adding fixed positional vectors up front.

### D. Host the model

In production AI infrastructure, the model server and the model weights are decoupled:

```text
┌─────────────────────────────────────────────────────────────┐
│ KUBERNETES POD                                              │
│                                                             │
│  ┌─────────────────────────┐   Mounts PVC   ┌─────────────┐ │
│  │ Container Image         │ ─────────────> │ PVC / Volume│ │
│  │ (The Model Server)      │  /models/...   │ (The Model) │ │
│  │                         │                │             │ │
│  │ • vLLM / Triton / Ollama│                │ • Weights   │ │
│  │ • CUDA Kernels & PyTorch│                │ • Embedding │ │
│  │ • HTTP / gRPC API       │                │ • Tokenizer │ │
│  └─────────────────────────┘                └─────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

#### The Model Server vs. The Model Artifacts

* **The Model Server (Stateless Container):** Applications like vLLM, Ollama, or Triton. They contain no model weights and no embedding tables; they contain only the execution engine and CUDA kernels.
* **The Model Artifacts (Stateful Files on PVC):**
  * `tokenizer.json`: The BPE vocabulary rules, loaded into **Host CPU RAM**.
  * `*.safetensors`: Binary weight tensors streamed directly into **GPU VRAM** at startup.

#### How Weights and Tables Are Created

1. **The Tokenizer Vocabulary:** Built once before training begins. Byte Pair Encoding (BPE) starts with 256
   byte values and iteratively merges frequent character pairs until vocabulary size `V` (e.g. 128,000) is reached.
2. **The Embedding Matrix (`W_E`):** A 2D table of shape `[128000, 4096]`. Initialized randomly, trained via
   backpropagation, and saved to disk. In FP16, it occupies ~1.05 GB of GPU VRAM as a read-only lookup table.
3. **Layer Projection Weights (`W_Q, W_K, W_V`):**
  * These are learned model parameters (~16.8 million floats each for `4096 × 4096`).
  * **Each layer has its own distinct set:** Layer 0 has its own `W_Q, W_K, W_V`, Layer 1 has its own, up to
    Layer 31. They specialize hierarchically (early layers learn syntax, later layers learn semantics).
  * During inference, they are **100% frozen and read-only** in GPU VRAM.

---

### Part 2: During Inference — Step-by-Step Attention Walkthrough

We trace the prompt: **`"what is quantum computing"`** through the system.

```text
[ Client Request: "what is quantum computing" ]
                     │
                     ▼ (Step 0: Host CPU)
   [ Tokenizer (BPE) ] ──> Integer IDs: [10919, 374, 18585, 9662]
                     │
                     ▼ (PCIe Transfer: 16 bytes)
   [ Embedding Matrix (W_E) ] ──> 4 Vectors: x_1_original .. x_4_original (d=4096 each)
                     │
                     ▼ (Step 2: Layer 1 to Layer 32 Assembly Line)
   [ Room 1: Attention (QKV + W_O) ] ──> Residual Patch #1 (x_after_attn)
                     │
                     ▼
   [ Room 2: FFN (SwiGLU) ] ──────────> Residual Patch #2 (x_layer_final)
                     │
                     ▼
   [ Primed 32 KV Caches ] & [ Final RMSNorm ] ──> x_4_final
                     │
                     ▼ (Step 3: Sampling First Output Token)
   [ W_unembed (lm_head) ] ──> 128k Logits ──> Sample Token 5 ("is")
```

#### Step 0: Converting Text to Token IDs (Host CPU)

* **Who & Where:** The **Host CPU** running the serving engine's tokenizer process.
* **How:**
  1. The CPU receives the string `"what is quantum computing"`.
  2. It matches words against the BPE dictionary in Host RAM.
  3. It outputs 4 integer IDs: `[10919, 374, 18585, 9662]` (`"what"`, `"is"`, `"quantum"`, `"computing"`).
  4. The 16-byte integer array is pushed across the **PCIe bus** into GPU VRAM via DMA.

#### Step 1: Converting Token IDs to Vectors (GPU VRAM)

* **Who & Where:** The **GPU Compute Cores** running an Embedding CUDA kernel at Layer 0.
* **How:**
  1. The GPU reads the 4 integer IDs and performs an `O(1)` memory gather against `W_E` in VRAM.
  2. Rows `10919, 374, 18585, 9662` produce 4 initial vectors: `x_1_original, x_2_original, x_3_original, x_4_original` (each length `d = 4096`).
  3. In FP16, this 2D tensor `[4, 4096]` occupies 32 KB of GPU VRAM.

#### Step 2: Layer 1 Attention (The Prefill Phase)

All 4 prompt vectors enter Layer 1 **simultaneously in parallel** across GPU tensor cores (compute-bound GEMM). And will travel through all 32 layers of LLM in parallel and generate 4 * 32 KV caches simultaneously.

##### 1. Generating Q, K, V Projections

Each initial vector `x_i_original` multiplies by three parameter matrices (`W_Q, W_K, W_V`, each sized `4096 × 4096`):

```text
Q_i = x_i * W_Q   (Query:  "What am I looking for?")
K_i = x_i * W_K   (Key:    "What attributes do I advertise?")
V_i = x_i * W_V   (Value:  "My actual information payload")
```

**Weight Matrices (`W_Q, W_K, W_V`):**
They are model weights stored as 2D floating-point matrices in GPU VRAM (`4096 × 4096 = 16,777,216` parameters each, ~33.5 MB in FP16).
Each layer has its own unique set (Layer 0 to Layer 31) that specializes hierarchically (syntax to semantics) and remains 100% frozen during inference.

**Intuitive Roles (Kubernetes Analogy):**
Query (`Q`) specifies Pod requirements (`nodeSelector: {disk: ssd}, cpu: 4`).
Key (`K`) advertises Node labels (`labels: {disk: ssd}, allocatable_cpu: 32`).
Value (`V`) holds the Node execution environment and IP address (the payload consumed once matched).

**Multi-Head Splitting:**
All three vectors (`Q, K, V`) are split across 32 attention heads (`4096 / 32 = 128` floats per head).
The GPU executes one fast, saturated matrix multiplication (`1 × 4096` times `4096 × 4096`) and reshapes the output into 32 heads in GPU SRAM with zero memory copy overhead.

##### 2. Calculating Attention Scores (The Matchmaker)

For token 4 (`"computing"`):

* Each head computes its own dot products against all preceding prompt tokens (and itself):
  ```text
  Raw_Score(4, 1) = (Q_4_head · K_1_head ["what"])      / sqrt(128)
  Raw_Score(4, 2) = (Q_4_head · K_2_head ["is"])        / sqrt(128)
  Raw_Score(4, 3) = (Q_4_head · K_3_head ["quantum"])   / sqrt(128)
  Raw_Score(4, 4) = (Q_4_head · K_4_head ["computing"]) / sqrt(128)
  ```
* **Causal Mask:** Tokens cannot look ahead. Position 4 inspects only positions 1, 2, 3, and 4 (tokens 5+ do not exist yet).
* **Softmax Normalization:** Converts those 4 raw scores into percentages summing to 100% per head:
  * With `"what"` (`K_1`): 5%
  * With `"is"` (`K_2`): 5%
  * With `"quantum"` (`K_3`): 65% (Massive conceptual match)
  * With `"computing"` (`K_4`): 25%

##### 3. Blending Value Vectors (`Blended_V_4` vs. Original `V_4`)

* Each head blends the 4 prompt Value vectors using the exact percentages calculated above:
  ```text
  Blended_V_4_head = (0.05 * V_1_head ["what"])
                   + (0.05 * V_2_head ["is"])
                   + (0.65 * V_3_head ["quantum"])
                   + (0.25 * V_4_head ["computing"])
  ```

##### 4. Room 1 Output Projection (`W_O`) and Residual Patch #1

* **Why multiply by `W_O` (`4096 × 4096`):**
  * *Synthesizing siloed heads:* Right after concatenation, the 32 heads sit in isolated partitions. `W_O` cross-correlates their separate findings into one cohesive conclusion.
  * *K8s Analogy:* Like a Root Cause Analysis engine correlating alerts from 32 separate Prometheus exporters (Node, Network, Kubelet) into a single actionable incident diagnosis.
  * *Coordinate alignment:* Projects the synthesized result back into the model's primary 4,096-dimensional coordinate space:
    ```text
    Attention_Output_4 = Blended_V_4 * W_O   (4,096 floats)
    ```
* **Residual Patch #1 (`x_4_after_attn = x_4_original + Attention_Output_4`, also called `x_4_mid`):**
  * *Identity preservation (The Git Patch / K8s Overlay):* `Attention_Output_4` contains only relational context (from `"quantum"`).
  * Replacing `x_4` would cause the token to forget it was originally `"computing"`. Adding preserves the base object and applies context as an overlay patch (`kubectl patch`).
  * *Hardware highway:* Creates an uninterrupted direct bypass bus preventing gradients from vanishing across 32 layers during training.

##### 5. Room 2 Feed-Forward Network (FFN / MLP) and Residual Patch #2

`x_4_after_attn` (or `x_4_mid`) does **not** move to Layer 2 yet! It immediately enters Room 2 of Layer 1.

* **Why Room 2 exists (Attention vs. FFN):**
  * *Room 1 (Attention):* Acts like a **Network Switch / API Gateway**. It routes information *between* tokens (connecting `"quantum"` with `"computing"`), but stores no factual knowledge.
  * *Room 2 (FFN):* Acts like a **Factual Database / Knowledge Base**. It operates on `x_4` strictly in isolation (no cross-token communication).
  * Over **65% of the model's parameters** live here, storing static world facts (e.g. quantum physics concepts).
* **The 3 SwiGLU Weight Matrices (Llama 3):**
  * `W_gate [4096 × 14336]`: Decides which concepts to activate.
  * `W_up [4096 × 14336]`: Expands features from 4,096 into 14,336 high-dimensional concept slots.
  * `W_down [14336 × 4096]`: Compresses filtered concepts back down to 4,096 dimensions.
* **Step-by-Step FFN Math:**
  ```text
  gate_vector  = RMSNorm(x_4_after_attn) * W_gate   [1 × 14336]
  up_vector    = RMSNorm(x_4_after_attn) * W_up     [1 × 14336]
  filtered     = SiLU(gate_vector) * up_vector      (element-wise multiply / concept mask)
  FFN_Output_4 = filtered * W_down                  [1 × 4096]
  ```
* **Residual Patch #2 (`x_4_layer1_final = x_4_after_attn + FFN_Output_4`):**
  * Injects the factual world-knowledge lookup on top of the contextualized representation.
  * `x_4_layer1_final` has now completed Layer 1 and is ready to enter **Layer 2**.

##### 6. Saving to the KV Cache and Stacking Through 32 Layers

* `K_1..K_4` and `V_1..V_4` generated in Room 1 of Layer 1 are stored in GPU VRAM so they are never recomputed.
* `x_4_layer1_final` enters **Layer 2**, repeating both Room 1 and Room 2 with Layer 2's unique weights (`W_Q, W_K, W_V, W_O, W_gate, W_up, W_down`).
* **32 Layers = 32 Separate KV Caches:** GPU VRAM maintains 32 independent KV cache buffers because each layer operates at a different level of semantic abstraction.
* **The Infra Memory Math (Why KV Cache dominates VRAM):**
  * `Memory per token = 2 (K & V) × 32 (layers) × 4096 (dims) × 2 bytes (FP16) ≈ 0.5 MB / token`.
  * A 4,000-token context consumes **2.0 GB of VRAM per user session**.
  * 20 concurrent users require **40 GB of VRAM** solely for KV caches, demonstrating why engines like **vLLM (PagedAttention)** are vital to prevent memory fragmentation and OOM crashes.

##### 7. Fate of Prompt Tokens at Layer 32 (Why Only x_4 Proceeds to Step 3)

* **The Fate of `x_1`, `x_2`, `x_3`:**
  Once Layer 32 finishes, activations for tokens 1, 2, and 3 (`x_1_layer32_final`, `x_2_layer32_final`, `x_3_layer32_final`) are discarded from GPU memory.
  They do **not** run through Step 3 (`W_unembed`).
* **Why compute them across 32 layers if they get discarded?**
  * *Populated KV Caches:* Key/Value vectors across all 32 layers (`K_1..K_3`, `V_1..V_3`) are safely stored in VRAM so future tokens generated in Decode can attend back to them.
  * *Contextualized `x_4`:* Through attention across all 32 layers, `x_4` absorbed critical context from tokens 1..3 to form a complete understanding of the prompt.
* **Why skip Step 3 for tokens 1..3?**
  * Multiplying `x_1` by `W_unembed` would predict token 2 (`"is"`), and `x_2` would predict token 3 (`"quantum"`).
  * In serving/inference, we already know the prompt text! Re-predicting known prompt tokens is redundant.
* **Infra Optimization (Slicing `[:, -1, :]`):**
  * Serving engines (vLLM, TensorRT-LLM) explicitly slice the output tensor (`hidden_states[:, -1, :]`), sending only `x_4_final` (`1 × 4096`) into `W_unembed` (`4096 × 128000`).
  * For a 4,000-token prompt, projecting all tokens would generate an ~800 MB logits tensor in VRAM and waste trillions of FLOPs. Slicing down to 1 token generates just a 256 KB logits vector.
* **Inference vs. Training Distinction:**
  * *Inference:* Only the last prompt token is projected to predict the next token.
  * *Training:* *All* tokens pass through `W_unembed` to compute Cross-Entropy Loss at every position simultaneously.

#### Step 3: Producing the First Output Token

At the final layer (`L = 32`), when position 4 finishes Layer 32:

* It passes through Layer 32's Attention (Room 1) and Residual Patch #1 (producing `x_4_after_attn_layer32`).
* It passes through Layer 32's FFN (Room 2) and Residual Patch #2 (producing `x_4_layer32_final`).
* It passes through a final standardizer layer (Final RMSNorm) to stabilize the float values.
* The result is a single vector of 4,096 floats (`x_4_final`).

The vector `x_4_final` for position 4 multiplies by `W_unembed` (`[4096, 128000]`). In the model files on disk, W_unembed is called `lm_head.weight`. It is a giant 2D matrix stored in GPU VRAM:

* It has 4,096 rows.
* It has 128,000 columns (one column for every single token in the vocabulary!).

When you multiply:

`[x_4_final (1 × 4096)] x [ W_unembed (4096 × 128000) ] = [Logits (1 × 128000) ]`

This is what happens step-by-step:

* **128,000 Raw Scores (Logits):** GPU calculates dot products between `x_4_final` and all 128,000 vocabulary columns simultaneously.
  This outputs an array of 128,000 unnormalized numbers called **Logits**.
* **Softmax:** The GPU turns those 128,000 numbers into probabilities summing to 100%:
  * Token "is": 78.4% (Huge winner!)
  * Token "a": 12.1%
  * Token "refers": 4.2%
  * ...
  * Token "banana": 0.00000001%
* **Sampling:** The serving engine picks the winning token ID:
  * It selects Token ID 374 (which maps to `"is"`).
* **De-tokenization & Streaming (GPU → Host CPU):**
  * **Device-to-Host Transfer (PCIe):** The GPU transfers only the 4-byte integer `374` back to Host CPU memory over PCIe. The GPU never stores text strings or dictionaries.
  * **Vocabulary Array Lookup:** The host model runtime performs an O(1) array lookup into `tokenizer.json` (`vocab[374]`), yielding raw bytes `b" is"`.
  * **Byte Stream & UTF-8 Assembly:** The CPU validates UTF-8 byte boundaries (buffering multi-byte characters like emojis if split across tokens) and checks for control tokens like `<|eot_id|>`.
  * **Server-Sent Events (SSE):** The host server flushes the decoded text chunk to the HTTP client stream immediately, while concurrently feeding Token ID `374` back into the GPU for Step 4.

**Note**: First Token is Generated and Streamed!

#### Step 4: The Decode Phase (Generating Token-by-Token)

Now begins the autoregressive loop for generating tokens 5, 6, 7... sequentially:

1. **Single Token Input:** Only the newly generated token (`"is"`, ID 374) enters the GPU as `x_5_original` (`[1 × 4096]`).
2. **Layer 1 Single-Vector Projections:** The GPU computes only one set of projections for position 5: `Q_5, K_5, V_5`.
3. **KV Cache Append:** `K_5` and `V_5` are appended to Layer 1's KV Cache in VRAM (now holding tokens 1..5).
4. **Attention over Historical KV Cache:** `Q_5` computes dot products against `K_1..K_5` (streamed from VRAM) to calculate attention weights, then blends `V_1..V_5` into `Blended_V_5`.
5. **Room 1 Output & Patch #1:** Multiplies by `W_O` and adds residual connection: `x_5_after_attn = x_5_original + Attention_Output_5`.
6. **Room 2 (FFN) & Patch #2:** Passes through Layer 1's FFN (SwiGLU) and adds residual connection: `x_5_layer1_final = x_5_after_attn + FFN_Output_5`.
7. **Stacking Across 32 Layers:** This single vector passes through Layers 2 to 32 sequentially, appending its `K_5` and `V_5` to each layer's cache along the way.
8. **Sampling Token 6:** At Layer 32, Final RMSNorm yields `x_5_final`, which multiplies by `W_unembed` to sample Token 6 (`"a"`).
9. **Loop Termination:** This cycle repeats one token at a time until the model samples `<|eot_id|>`, signaling the host server to close the stream.

---

### Systems Summary (Why Infra Engineers Care)

| Phase | What Happens | Systems Bottleneck | Why? |
|---|---|---|---|
| **Prefill** | Processes prompt (`"what is quantum computing"`) | **Compute-Bound** (TFLOPS) | Full matrix-matrix multiplication (`Batch × Tokens`). GPU tensor cores are fully saturated. |
| **Decode** | Generates tokens 1-by-1 (`"is"`, `"a"`, `"branch"`) | **Memory-Bandwidth Bound** (GB/s) | GPU streams weights + entire KV cache from VRAM for each single token to compute one vector. |
