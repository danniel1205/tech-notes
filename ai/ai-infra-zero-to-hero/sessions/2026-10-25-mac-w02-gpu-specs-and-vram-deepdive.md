# 2026-10-25 · mac · Week 2: GPU Spec Sheets, VRAM Allocation, and Kubernetes Platform Boundaries

## Done

- Recapped Week 1: Why Decode is memory-bandwidth bound (H100 math race: 16 us compute vs 4,800 us transfer = 99.7% ALU idle time).
- Updated Section 11 of `Phase-1:Foundations/Week-1:LLMs-for-system-people/readme.md` with physical hardware hierarchy and the Golden Rule: `Max_Tokens_Per_Sec = Bandwidth_GBs / Weight_GB`.
- Renamed `Phase-1:Founations` to `Phase-1:Foundations` via `git mv` and updated all internal references.
- Initialized Week 2: Created `Phase-1:Foundations/Week-2:GPU-hardware-and-rooflines/readme.md` and `lab/readme.md`.
- Deep-dived into Spec #1 (VRAM / HBM Capacity): 3D stacked dies on silicon interposer, 5 active + 1 spare stack on H100.
- Documented what happens when weights exceed single GPU VRAM (CUDA OOM crash, multi-GPU Tensor Parallelism over NVLink, and Weight Quantization).
- Documented KV cache pool allocation: static pre-allocation at boot (`gpu_memory_utilization = 0.90`), PagedAttention dynamic leasing, and pool exhaustion (queueing vs preemption).
- Clarified the two memory boundaries: Host RAM -> VRAM over PCIe at startup (64 GB/s) vs VRAM -> SRAM over HBM bus during inference (3,350 GB/s).
- Documented Model Server vs. Kubernetes Platform responsibilities: internal vLLM queuing/swapping vs KEDA/HPA on `gpu_cache_usage_factor` and Gateway API Inference Extension routing.
- Ran `make md-fix` and confirmed 0 lint errors across all 149 markdown files.

## Measurements

| What | Setup | Result |
|---|---|---|
| Single-Token Decode Compute vs Bus Time | Llama-3-8B FP16 (16 GB) on NVIDIA H100 | Compute: 16 us, Bus Transfer: 4,800 us (99.7% ALU idle) |
| Prompt Prefill Compute vs Bus Time | 4,000 tokens on Llama-3-8B FP16, NVIDIA H100 | Compute: 65 ms, Bus Transfer: 4.8 ms (100% ALU saturation) |
| PCIe vs Internal HBM Bus Speed | PCIe Gen4 (64 GB/s) vs H100 HBM3 (3,350 GB/s) | PCIe is ~50x slower than internal GPU memory bus |

## Learned

- Model weights (100% of all layers) must reside permanently in GPU VRAM from startup; layer-by-layer streaming happens strictly between VRAM and on-chip SRAM across the 3,350 GB/s bus.
- vLLM pre-allocates the entire KV cache pool (~60 GB on an 80 GB GPU) at boot, causing `nvidia-smi` to show 90% memory usage even with 0 active users.
- Standard Kubernetes cannot autoscale on VRAM %; platform engineers must use KEDA on custom metrics (`vllm:num_requests_waiting` and `vllm:gpu_cache_usage_factor`).

## Open questions

- None for Spec #1 (VRAM allocation and platform boundaries documented).

## Next steps

- On mac: Deep dive into Week 2 Spec #1 Subsection 3: Bare Metal K8s vs. KubeVirt VM Clusters (VFIO, Re-BAR, vNUMA alignment, GPUDirect RDMA/Storage), then proceed to Spec #2 (Memory Bandwidth).
- On cloudtop: Set up GitHub auth (`gh auth login` + `gh auth setup-git`); request GPU quota (L4 now; 8x H100/A3 for later weeks);
  run Week 2 GCE L4 profiling lab (`nvidia-smi dmon`, `dcgmi dmon`, `nsys profile`).
