# Week 2: GPU Hardware, the Memory Hierarchy, and Rooflines

This learning note and reading guide explores GPU hardware microarchitecture through the lens of a systems and Kubernetes platform engineer.

---

## 1. Goal & Syllabus for Week 2

* **Primary Goal:** Learn how to read an NVIDIA GPU datasheet like a Kubernetes node specification, understand the GPU memory hierarchy, master the Roofline Model, and analyze the FlashAttention paper.
* **Paper:** [FlashAttention: Fast and Memory-Efficient Exact Attention with IO-Awareness (Dao et al., 2022)](https://arxiv.org/abs/2205.14135).
* **Core Topics:**
  1. Reading a GPU spec sheet like a K8s node spec (SMs, Tensor Cores, HBM, Memory Bandwidth).
  2. The GPU Memory Hierarchy (Registers -> SRAM -> L2 Cache -> HBM / VRAM).
  3. The Roofline Model (Arithmetic intensity, ridge point, prefill vs decode).
  4. FlashAttention mechanics (IO-awareness, SRAM tiling, online softmax).
  5. MFU (Model FLOPs Utilization) and profiling tools (`nvidia-smi`, `dcgmi`, `nsys`).

---

## 2. Reading a GPU Spec Sheet like a Kubernetes Node Spec

When platform engineers provision Kubernetes worker nodes, they look at node capacity:
`CPU cores`, `Memory (GiB)`, `Network bandwidth (Gbps)`, and `Disk IOPS`.

When deploying LLM workloads on GPU nodes, you read a GPU datasheet with the exact same mental model:

| GPU Spec Field | What it Actually Is | Kubernetes / Systems Equivalent | Why Infra Engineers Care |
| --- | --- | --- | --- |
| **VRAM / HBM Capacity** | High Bandwidth Memory size (e.g. 24 GB, 80 GB, 141 GB, 192 GB) | Node RAM (`node.status.capacity.memory`) | Limits max model weights + KV cache allocatable pool before OOM. |
| **Memory Bandwidth** | Rate at which weights move from HBM to compute cores (e.g. 3,350 GB/s) | Main Memory Bus Throughput (e.g. DDR5 / PCIe bandwidth) | **Directly sets single-stream Decode tokens/second.** |
| **Streaming Multiprocessors (SMs)** | Independent hardware compute clusters on the die (e.g. 132 SMs on H100) | Physical CPU Socket / Core count | Determines hardware concurrency and thread block scheduling slots. |
| **Tensor Cores** | Specialized matrix-multiplication hardware execution units inside each SM | Vector math accelerators (AVX-512 / AMX) | Provides raw TFLOPS throughput for Prefill and batched matrix math. |
| **Dense vs Sparse TFLOPS** | Dense = standard math; Sparse = 2:4 structured sparsity (skipping zeros) | Raw vs overcommitted compute capacity | Most serving engines run **Dense**; 2x sparse marketing numbers require 2:4 pruned weights. |
| **NVLink Bandwidth** | Ultra-high-speed GPU-to-GPU interconnect bus (e.g. 900 GB/s on H100) | Dedicated multi-hundred Gbps backend network fabric (RoCE / InfiniBand) | Determines Tensor Parallelism (TP) efficiency across GPUs within a node. |
| **Thermal Design Power (TDP)** | Maximum power draw under full compute load (e.g. 350W - 700W per GPU) | Node power & cooling allocation in rack | Sizing cluster power distribution units (PDUs) and thermal ceilings. |

---

### Deep Dive — Spec #1: VRAM / HBM Capacity

```text
 ┌────────────────────────────────────────────────────────┐
 │                    NVIDIA H100 GPU                     │
 │  ┌──────────────────────────────────────────────────┐  │
 │  │              Compute Die (Silicon)               │  │
 │  └──────────────────────────────────────────────────┘  │
 │          ▲                               ▲             │
 │          │ (Silicon Interposer - 3,350 GB/s)           │
 │          ▼                               ▼             │
 │  ┌──────────────┐                       ┌───────────┐  │
 │  │ HBM3 Stack 1 │  ... [ 5 Active ] ... │HBM3 Stack5│  │
 │  │   (16 GB)    │      [ 1 Spare  ]     │  (16 GB)  │  │
 │  └──────────────┘                       └───────────┘  │
 │              Total Capacity: 80 GB (5 x 16 GB)         │
 └────────────────────────────────────────────────────────┘
```

#### 1. What HBM Physically Is (and why it is not GDDR)

* **GDDR6 (e.g. NVIDIA L4, consumer GPUs):** Individual memory chips soldered onto the PCB around the GPU die, connected by copper board traces.
* **HBM3 (e.g. NVIDIA H100):** Vertically stacked 3D DRAM dies placed directly on the same silicon substrate (interposer) alongside the GPU compute die.
* **The H100 Physical Layout:** Contains 6 physical stacks of 16 GB HBM3. Five stacks are active (giving 80 GB total), and one stack is reserved for manufacturing yield redundancy.
* **Interposer Bus Width:** Because HBM sits on the silicon interposer, it connects via a massive 5,120-bit wide bus, eliminating PCB trace impedance.

#### 2. The VRAM Allocation Budget in a Pod

When a model runtime (like vLLM or TensorRT-LLM) boots in a Kubernetes Pod, VRAM is partitioned into four distinct buckets:

```text
 Total H100 VRAM: 80 GB
 ├── 1. CUDA Driver & Context Reserve:  ~1.5 - 2.0 GB (Static OS/Driver baseline)
 ├── 2. Model Weights:                  16.0 GB (Llama 3 8B in FP16)
 ├── 3. Dynamic Activation Scratchpad:  ~1.0 - 2.0 GB (Working memory for GEMM)
 └── 4. KV Cache Allocation Pool:       ~60.0 GB (The entire rest of VRAM!)
```

* **The Kubernetes Equivalent:**
  * Buckets 1, 2, and 3 = **System Overhead + Pod Baseline Request**.
  * Bucket 4 (KV Cache Pool) = **Allocatable Capacity for User Sessions**.
* Engines like vLLM pre-allocate Bucket 4 upfront (`gpu_memory_utilization = 0.90`). If model weights grow, the KV cache pool shrinks, directly decreasing maximum user concurrency before queuing.

##### What Happens When Weights Exceed Single-GPU VRAM?

* **The Failure Mode:** The Pod throws `torch.cuda.OutOfMemoryError` at startup during weight loading and enters `CrashLoopBackOff`.
* **Industry Solutions:**
  1. **Multi-GPU Tensor Parallelism (TP):** Shard individual weight matrices across 2, 4, or 8 GPUs
     over high-speed NVLink interconnect (e.g. Llama-3-70B FP16 at 140 GB runs across 4x H100
     with TP=4, using 35 GB weights per GPU).
  2. **Weight Quantization:** Compress weights to FP8 (1 byte/param = ~70 GB) or INT4 AWQ/GPTQ (0.5 bytes/param = ~38 GB), fitting a 70B model onto a single 80 GB GPU.

##### How KV Cache Pool Allocation Affects Concurrency

Walkthrough example: hosting **`Muse-Spark-Open-816B-A42B` (INT4, ~469 GB weights)** on a single **8x H100 node (640 GB total VRAM)** with `gpu_memory_utilization = 0.90`:

* **1. Static Pool Reservation (Boot-Time Slab):**
  * `gpu_memory_utilization = 0.90` (`576 GB`) is the **total VRAM budget** (`Weights + KV Cache Pool`), *not* 576 GB for KV cache alone.
    The remaining 10% (`64 GB`) stays as headroom for CUDA drivers, NCCL, and activation spikes.
  * After loading `~469 GB` of INT4 weights, vLLM statically reserves the remaining **`~107 GB`** in one shot at boot (like Postgres `shared_buffers` or DPDK hugepages) to avoid `cudaMalloc` fragmentation.
  * Because the `107 GB` slab is locked at boot, `nvidia-smi` reports **~90% VRAM used (`576 / 640 GB`)** even with 0 active users.
* **2. Dynamic Page Leasing (PagedAttention Inside the Slab):**
  * vLLM slices that static `~107 GB` pool into fixed-size blocks (16 tokens/block; at `~60 KB/token` hybrid FP8, `1 block = ~1 MB`, yielding `~110,000` free blocks).
  * Like an OS virtual-memory page table, sessions lease blocks **on demand** as tokens grow rather than reserving worst-case `100k` context (`~6 GB`) upfront:
    a `1,000-token` prompt leases only `63` blocks (`~60 MB`), freeing them upon completion.
  * Thus **~17 concurrent users** at `100k` context is the **worst-case floor**; if average requests use `10k` tokens, the same `107 GB` pool supports **~170 concurrent users**.
* **3. Pool Exhaustion (When the 18th 100k-Context Request Arrives):**
  * **Scenario A — Admission Queuing:** If Users 1–17 already hold `~107 GB`, Request #18 waits in vLLM's internal `waiting` queue (`vllm:num_requests_waiting`). Users 1–17 stream normally; #18 sees higher **TTFT**.
  * **Scenario B — Mid-Decode Preemption:** If 20 users admitted at `50k` tokens (`60 GB`) all grow toward `100k` and hit 0 free blocks mid-decode, vLLM avoids deadlock by preempting a victim sequence:
    * **Recomputation (default in vLLM V1):** Drops the victim's KV blocks, appends generated tokens to its prompt, and re-runs Prefill later once VRAM frees up.
    * **CPU Swapping (`--swap-space`):** Copies KV blocks over PCIe to Host CPU RAM and swaps them back later (requires sizing Pod `resources.limits.memory` to avoid Linux `OOMKilled`).
* **4. Who Handles What (Engine vs. Kubernetes + KEDA):**
  * **Why default K8s HPA fails:** Host CPU/RAM stay flat, and GPU VRAM (`nvidia-smi` / DCGM `FB_USED`) stays pinned at ~90% from 0 to 17 users.
  * **Model Server (vLLM inside Pod):** Manages internal admission queuing, dynamic block leasing, and preemption/swapping.
  * **Kubernetes Platform Layer:**
    * **KEDA:** Scales out on `vllm:kv_cache_usage_perc` (e.g. `> 0.75`–`0.80`) and `vllm:num_requests_waiting` *before* preemption starts.
    * **Preemption Alerting (`vllm:num_preemptions_total`):** Treat `rate(vllm:num_preemptions_total[5m]) > 0` in Grafana as a "too late" alarm.
      If it climbs, `469 GB` weight cold-start is outpacing the 20% KV buffer — lower the KEDA threshold (e.g. `0.80 -> 0.70`) and use fast weight caching or warm standby nodes.
    * **Gateway API Inference Extension (EPP):** Replaces blind `kube-proxy` round-robin so Request #18 routes to the newly scaled Replica #2 instead of overloading saturated Replica #1.

#### 3. Bare Metal K8s vs. KubeVirt VM Clusters

In production, how VRAM is exposed depends heavily on whether your Kubernetes cluster runs directly on bare metal or on virtual machines managed by KubeVirt:

* **Bare Metal H100 Cluster:**
  * The NVIDIA GPU Operator deploys the Container Toolkit (`nvidia-ctk`) and device plugin on Linux nodes.
  * Pods request `resources.limits: { nvidia.com/gpu: 1 }`, receiving character device access (`/dev/nvidia0`, `/dev/nvidia-uvm`).
  * Direct DMA: Zero hypervisor translation overhead. Supports GPUDirect Storage (GDS) and GPUDirect RDMA directly from NVMe/NIC into HBM VRAM.
* **KubeVirt VM Cluster (Virtualized GPU via PCIe Passthrough):**
  * **VFIO Binding:** The bare-metal host binds the physical H100 to `vfio-pci`. KubeVirt attaches the PCIe device directly to the QEMU/KVM guest virtual machine.
  * **Resizable BAR (Re-BAR) Requirement:** GPUs map VRAM into CPU host memory via Base Address
    Register 1 (BAR 1). Legacy systems restrict BAR 1 to 256 MB. For 80 GB HBM, Resizable BAR
    must be enabled in BIOS and passed through in QEMU so the guest OS can map large weight buffers over PCIe.
  * **vNUMA Alignment:** Physical H100s are attached to a specific PCIe root complex on Socket 0
    or Socket 1. If KubeVirt pins vCPUs from NUMA Node 1 while the GPU sits on NUMA Node 0, transfers
    cross the high-latency UPI/QPI inter-socket link, degrading prompt ingestion speeds. Strict
    NUMA pinning (`dedicatedCpuPlacement: true`) is essential.

#### 4. Model Weights Cold-Start Journey (Object Storage -> Local PV -> VRAM)

```text
STAGE 1: Remote Object Storage -> Local PV
[S3 / GCS] ──(HTTPS/TLS)──> [Node NIC] ──> [Host CPU RAM (TCP/Page Cache)] ──> [Local NVMe PV]

STAGE 2 (Standard Path — Default): Local PV -> Host RAM -> GPU VRAM
[Local NVMe PV] ──(PCIe DMA)──> [Host CPU RAM (Page Cache + PyTorch Buffer)] ──(PCIe H2D)──> [GPU HBM VRAM]

STAGE 2 (GDS / Direct DMA Path — Bare Metal Optimized):
[Local NVMe PV] ──────────────(Direct Peer-to-Peer PCIe Switch, skips CPU RAM)─────────────> [GPU HBM VRAM]
```

* **Key Hardware Terms:**
  * **NVMe / NIC:** Local PCIe SSDs (`NVMe`) storing cached model shards vs. high-speed RDMA network cards (`NIC`, e.g. 400 Gbps InfiniBand/RoCE) for network pulls and multi-node traffic.
  * **Direct DMA:** Peer-to-peer PCIe transfers allowing an NVMe SSD or NIC to write directly into GPU VRAM without bouncing through Host CPU RAM.
  * **GDS (GPUDirect Storage):** NVIDIA driver (`nvidia-fs` / `cuFile`) enabling Direct DMA from NVMe storage to GPU VRAM (its NIC twin, **GPUDirect RDMA**, does the same from NIC to VRAM).
* **Stage 1 (Object Storage -> Local NVMe PV):**
  * An `initContainer` pulls `.safetensors` shards over HTTPS through the Node NIC and Host CPU RAM onto a Local NVMe PV (`469 GB` over a 25 Gbps NIC takes ~2.5+ min).
  * **K8s Mitigation:** Avoid root-disk `emptyDir` (`DiskPressure` evictions); pre-warm weights on node-local NVMe via a DaemonSet so scaling Pods skip Stage 1.
* **Stage 2 (Local PV -> GPU HBM VRAM):**
  * **Standard Path:** `mmap`/`read()` pulls shards into Host CPU RAM (Page Cache + PyTorch buffers), then `cudaMemcpyAsync` pushes each GPU's TP shard across PCIe (`~32-64 GB/s`) into HBM.
  * **K8s Trap (`OOMKilled`):** Linux cgroups v2 counts Host CPU buffers and page cache charges against `resources.limits.memory`; undersized Pod memory limits cause `OOMKilled` before weights reach VRAM.
  * **GDS Path (Bare Metal):** Streams shards directly from NVMe across the PCIe switch into HBM VRAM, bypassing Host CPU RAM and eliminating host memory spikes.
* **Stage 3 (Post-Load Boot before `Ready`):**
  * After weights land in VRAM, vLLM reserves the static KV cache pool, initializes NCCL NVLink rings, and captures CUDA graphs (~15-45 sec) before passing the K8s `readinessProbe`.

#### 5. Hardware Partitioning (MIG) vs. Software Sharing

* **Software Sharing (Time-Slicing / MPS):** Multiple Pods share the same 80 GB VRAM. However, there is zero memory isolation; if one Pod exhausts memory, it can cause OOM errors for all co-located Pods.
* **MIG (Multi-Instance GPU):** The H100 physically slices its 80 GB VRAM and 132 SMs into isolated
  hardware partitions (e.g. `1g.10gb`, `2g.20gb`, `3g.40gb`, `4g.40gb`). Each slice has dedicated
  memory controllers and hardware bandwidth isolation. An OOM in one slice cannot impact another.

---

## 3. Comparison of Common Cloud GPUs

| Spec | NVIDIA L4 | NVIDIA A100 SXM | NVIDIA H100 SXM | NVIDIA H200 SXM | NVIDIA B200 SXM |
| --- | --- | --- | --- | --- | --- |
| **Architecture** | Ada Lovelace | Ampere | Hopper | Hopper | Blackwell |
| **VRAM (HBM)** | 24 GB GDDR6 | 80 GB HBM2e | 80 GB HBM3 | 141 GB HBM3e | 192 GB HBM3e |
| **Memory Bandwidth** | 300 GB/s | 2,039 GB/s | 3,350 GB/s | 4,800 GB/s | 8,000 GB/s |
| **SM Count** | 58 SMs | 108 SMs | 132 SMs | 132 SMs | 148 SMs |
| **FP16 / BF16 Dense TFLOPS** | 121 TFLOPS | 312 TFLOPS | 989 TFLOPS | 989 TFLOPS | 2,250 TFLOPS |
| **NVLink** | None (PCIe only) | 600 GB/s | 900 GB/s | 900 GB/s | 1,800 GB/s |
| **TDP (Power)** | 72 W | 400 W | 700 W | 700 W | 1,000 W |
| **Primary Workload Fit** | Light serving, audio, vision, INT4 8B models | Standard 8B/70B serving (TP=4) | High-throughput serving, 70B/405B serving | Giant KV caches, long-context 70B models | Extreme throughput, dense large models |

---

## 4. The GPU Memory Hierarchy

Data moves through a steep hierarchy of speed versus capacity:

```text
 Level               Capacity         Bandwidth / Latency      Systems Analogy
 ──────────────────────────────────────────────────────────────────────────────────
 Registers           ~256 KB / SM     ~30,000+ GB/s (<1 ns)    CPU Registers
 SRAM (Shared Mem)   ~228 KB / SM     ~20,000+ GB/s (few ns)   CPU L1/L2 Cache
 L2 Cache (On-Chip)  50 MB (chip)     ~12,000 GB/s (10-20 ns)  OS Page Cache / Linux Buffers
 HBM3 (VRAM)         80 GB (chip)     3,350 GB/s (100-200 ns)  Local NVMe SSD Array
 Host CPU RAM        1 TB - 2 TB      ~100 GB/s (PCIe Gen5)    NFS / Object Storage Mount
```

---

## 5. The Roofline Model: Arithmetic Intensity & Ridge Point

The Roofline Model provides a 2D map explaining whether an algorithm is compute-bound or memory-bound on a given hardware target.

### Key Concepts

* **Arithmetic Intensity (AI):** The ratio of math operations performed per byte of memory read from HBM:

  ```text
  Arithmetic_Intensity = FLOPs / Bytes_Transferred
  ```

* **The Ridge Point (Inflection Point):** The minimum arithmetic intensity required to fully saturate a GPU's compute cores:

  ```text
  Ridge_Point = Peak_TFLOPS / Peak_Memory_Bandwidth_TBps
  ```

  * On an NVIDIA H100 (989 TFLOPS, 3.35 TB/s):
    `Ridge_Point = 989 / 3.35 ≈ 295 FLOPs / byte`.
  * If an operation's arithmetic intensity is **greater than 295 FLOPs/byte**: It sits on the flat roof (**Compute-Bound**).
  * If an operation's arithmetic intensity is **less than 295 FLOPs/byte**: It is trapped on the slope (**Memory-Bandwidth Bound**).

### Where Do LLM Inference Phases Sit?

1. **Prefill Phase (Prompt Ingestion):**
   * Processes thousands of tokens concurrently.
   * Arithmetic Intensity: `~1,000 to 2,000 FLOPs / byte`.
   * **Location:** Far to the right of the ridge point, on the flat ceiling. Throttled by **TFLOPS**.
2. **Decode Phase (Single-Token Autoregressive Generation):**
   * Generates only 1 token per step (batch size = 1).
   * Arithmetic Intensity: `~1 to 2 FLOPs / byte`.
   * **Location:** Far to the left at the bottom of the slope. Throttled by **GB/s Memory Bandwidth**.

---

## 6. Paper Study: FlashAttention (Dao et al., 2022)

* **Paper Title:** *FlashAttention: Fast and Memory-Efficient Exact Attention with IO-Awareness*
* **Core Problem in Standard Attention:**
  * Standard attention computes: `Scores = Q * K^T`, followed by `Probabilities = Softmax(Scores)`, followed by `Output = Probabilities * V`.
  * To compute softmax, standard implementations write an intermediate `[N, N]` attention score matrix out to slow **HBM (VRAM)** and read it back.
  * For a sequence length of N = 4,000 or 16,000, writing and reading an N x N matrix causes massive O(N^2) memory traffic across the memory bus, stalling execution.
* **The FlashAttention Breakthrough:**
  * **IO-Awareness:** Treats memory movement between HBM and on-chip SRAM as the primary optimization objective.
  * **Tiling:** Divides Query, Key, and Value matrices into blocks that fit entirely inside **on-chip SRAM** (~40 MB).
  * **Online Softmax:** Computes softmax incrementally across blocks without ever materializing or writing the giant N x N matrix out to HBM.
  * **Result:** 2x to 4x faster attention speed and O(N) memory footprint instead of O(N^2), computing the mathematically identical exact attention.

---

## 7. Model FLOPs Utilization (MFU)

* **Definition:** The ratio of observed floating-point operations per second to the theoretical peak FLOPS of the hardware:

  ```text
  MFU = Measured_FLOPS / Theoretical_Peak_FLOPS
  ```

* **Why MFU Matters in Production:**
  * In training, an MFU of **45% to 55%** is considered state of the art.
  * In inference Prefill, large prompts achieve **30% to 50% MFU**.
  * In inference single-stream Decode, MFU is often **under 2%** because Tensor Cores are idling waiting on memory bandwidth.
