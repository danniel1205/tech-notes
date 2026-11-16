# 2026-11-15 · mac · Week 2: KV Cache Pool Mechanics, MoE Capacity Planning, and Cold-Start Journey

## Done

- Capacity-planned a frontier MoE model (`Muse-Spark-Open-816B-A42B` as open-weight proxy for closed-weights Meta Muse Spark 1.2) on `1x 8x H100` (640 GB VRAM) in INT4.
- Deep-dived KV Cache Pool concurrency: Static Pool Reservation (`0.90` = `576 GB` total budget -> `~107 GB` KV pool), Dynamic Page Leasing (PagedAttention),
  Pool Exhaustion (admission queuing vs. mid-decode preemption/recomputation/CPU swap), and K8s KEDA/HPA + `vllm:num_preemptions_total` alerting + Gateway API Inference Extension (EPP).
- Clarified hardware I/O terms (`NVMe / NIC`, `Direct DMA`, `GDS`, `GPUDirect RDMA`) and traced the K8s Model Weights Cold-Start Journey
  (Object Storage -> Local NVMe PV -> Host CPU RAM or GDS Direct DMA -> GPU HBM VRAM -> KV reservation & CUDA graphs).
- Updated `Phase-1:Foundations/Week-2:GPU-hardware-and-rooflines/readme.md` with the `Muse-Spark-Open` KV pool walkthrough and Subsection 4 (`Model Weights Cold-Start Journey`).

## Measurements

| What | Setup | Result |
|---|---|---|
| `Muse-Spark-Open-816B-A42B` VRAM Sizing | BF16 / FP8 / INT4 total weights (816B total, 42B active) | BF16: ~1,632 GB (3x 8xH100); FP8: ~816 GB (2x 8xH100 or 1x 8xH200); INT4: ~469 GB (1x 8xH100) |
| KV Pool & Concurrency (INT4 on 8x H100) | 640 GB VRAM, `gpu_memory_utilization = 0.90` (576 GB budget) | ~469 GB weights + ~107 GB KV pool (~110k 16-tok blocks); ~17 users @ 100k ctx (~170 @ 10k) |
| Stage 1 Weight Download Time | 469 GB INT4 checkpoint from Object Storage to Local NVMe | ~2.5+ min on 25 Gbps NIC (~6.5 min on 10 Gbps NIC) |

## Learned

- For MoE models, VRAM capacity scales with **Total Parameters (816B)**, while single-stream decode speed scales with **Active Parameters (~42B)**.
- `gpu_memory_utilization = 0.90` caps `Weights + KV Cache`, not KV Cache alone; `vllm:num_preemptions_total` is a "too late" alarm meaning KEDA thresholds must be lowered for Pod cold-start time.
- Standard K8s weight loading bounces through Host CPU RAM / cgroups v2 page cache (risking `OOMKilled`), whereas GDS Direct DMA on bare metal streams NVMe -> HBM VRAM directly.

## Open questions

- None for Spec #1 (VRAM / HBM Capacity).

## Next steps

- On mac: Deep-dive Week 2 Specs #2-#5 (Memory Bandwidth, SMs, Tensor Cores, NVLink), Roofline / Ridge Point math, and FlashAttention paper; run `make md-fix` once Docker is running.
- On cloudtop: Run `./sync.sh start`; set up GitHub auth (`gh auth login` + `gh auth setup-git`) if needed; request GPU quota (L4 now; 8x H100/A3 later);
  run Week 2 GCE L4 profiling lab (`nvidia-smi dmon`, `dcgmi dmon`, `nsys profile`).
