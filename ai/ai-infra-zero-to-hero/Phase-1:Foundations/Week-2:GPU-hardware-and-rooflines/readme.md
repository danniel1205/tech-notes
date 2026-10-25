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
|---|---|---|---|
| **VRAM / HBM Capacity** | High Bandwidth Memory size (e.g. 24 GB, 80 GB, 141 GB, 192 GB) | Node RAM (`node.status.capacity.memory`) | Limits max model weights + KV cache allocatable pool before OOM. |
| **Memory Bandwidth** | Rate at which weights move from HBM to compute cores (e.g. 3,350 GB/s) | Main Memory Bus Throughput (e.g. DDR5 / PCIe bandwidth) | **Directly sets single-stream Decode tokens/second.** |
| **Streaming Multiprocessors (SMs)** | Independent hardware compute clusters on the die (e.g. 132 SMs on H100) | Physical CPU Socket / Core count | Determines hardware concurrency and thread block scheduling slots. |
| **Tensor Cores** | Specialized matrix-multiplication hardware execution units inside each SM | Vector math accelerators (AVX-512 / AMX) | Provides raw TFLOPS throughput for Prefill and batched matrix math. |
| **Dense vs Sparse TFLOPS** | Dense = standard math; Sparse = 2:4 structured sparsity (skipping zeros) | Raw vs overcommitted compute capacity | Most serving engines run **Dense**; 2x sparse marketing numbers require 2:4 pruned weights. |
| **NVLink Bandwidth** | Ultra-high-speed GPU-to-GPU interconnect bus (e.g. 900 GB/s on H100) | Dedicated multi-hundred Gbps backend network fabric (RoCE / InfiniBand) | Determines Tensor Parallelism (TP) efficiency across GPUs within a node. |
| **Thermal Design Power (TDP)** | Maximum power draw under full compute load (e.g. 350W - 700W per GPU) | Node power & cooling allocation in rack | Sizing cluster power distribution units (PDUs) and thermal ceilings. |

---

## 3. Comparison of Common Cloud GPUs

| Spec | NVIDIA L4 | NVIDIA A100 SXM | NVIDIA H100 SXM | NVIDIA H200 SXM | NVIDIA B200 SXM |
|---|---|---|---|---|---|
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
