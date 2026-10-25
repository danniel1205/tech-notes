# Progress

Last updated: 2026-10-25 by mac. Current week: **Week 2 (in progress)**.

## Next up

- **On mac:** Deep dive into Week 2 Spec #1 Subsection 3: Bare Metal K8s vs. KubeVirt VM Clusters (VFIO, Re-BAR, vNUMA alignment, GPUDirect RDMA/Storage), then explore Spec #2 (Memory Bandwidth).
- **On cloudtop:** Set up GitHub auth (`gh auth login` + `gh auth setup-git`); request GPU quota (L4 now; 8x H100/A3 for later weeks);
  run Week 2 GCE L4 profiling lab (`nvidia-smi dmon`, `dcgmi dmon`, `nsys profile`).

## Setup checklist

- [x] Plan written (`readme.md`)
- [x] Sync folder created (`AGENTS.md`, `decisions.md`, `PROGRESS.md`, `sessions/`, `sync.sh`)
- [ ] cloudtop: GitHub auth works for push
- [x] mac: `tech-notes` cloned on branch `kb/ai-infra`, folder opened in Antigravity
- [ ] GPU quota requested (L4, A3/H100, optional TPU v6e)

## Weekly checklist

| Week | Topic | Paper | Lab | Note | Drill |
|---|---|---|---|---|---|
| 1 | LLMs for systems people | [x] | [x] | [x] | [x] |
| 2 | GPU hardware and rooflines | [ ] | [ ] | [ ] | [ ] |
| 3 | Quantization and local inference | [ ] | [ ] | [ ] | [ ] |
| 4 | Continuous batching, PagedAttention, vLLM V1 | [ ] | [ ] | [ ] | [ ] |
| 5 | Benchmarking, SLOs, chunked prefill | [ ] | [ ] | [ ] | [ ] |
| 6 | Prefix caching, SGLang, speculative decoding | [ ] | [ ] | [ ] | [ ] |
| 7 | Parallelism and MoE for inference | [ ] | [ ] | [ ] | [ ] |
| 8 | Serving on GKE, cold start | [ ] | [ ] | [ ] | [ ] |
| 9 | Autoscaling, observability, cost per token | [ ] | [ ] | [ ] | [ ] |
| 10 | Gateway API Inference Extension, multi-LoRA | [ ] | [ ] | [ ] | [ ] |
| 11 | Multi-host: LWS, Kueue, DRA | [ ] | [ ] | [ ] | [ ] |
| 12 | Disaggregated P/D, distributed KV cache | [ ] | [ ] | [ ] | [ ] |
| 13 | Agent fundamentals | [ ] | [ ] | [ ] | [ ] |
| 14 | Agent frameworks and runtimes | [ ] | [ ] | [ ] | [ ] |
| 15 | MCP | [ ] | [ ] | [ ] | [ ] |
| 16 | A2A and multi-agent | [ ] | [ ] | [ ] | [ ] |
| 17 | Sandboxed code execution | [ ] | [ ] | [ ] | [ ] |
| 18 | State, memory, durable execution | [ ] | [ ] | [ ] | [ ] |
| 19 | AI gateways, token budgets, caching | [ ] | [ ] | [ ] | [ ] |
| 20 | Multi-tenancy, identity, security | [ ] | [ ] | [ ] | [ ] |
| 21 | Observability and evals for agents | [ ] | [ ] | [ ] | [ ] |
| 22 | Serving agentic workloads | [ ] | [ ] | [ ] | [ ] |
| 23 | Capstone: integration + Go operator | [ ] | [ ] | [ ] | [ ] |
| 24 | Capstone: demo, hardening, write-up | [ ] | [ ] | [ ] | [ ] |
| 25 | Interview: system design reps | [ ] | [ ] | [ ] | [ ] |
| 26 | Interview: deep dives and stories | [ ] | [ ] | [ ] | [ ] |

## Milestones

- [ ] M1 (W3): calculator + local inference report
- [ ] M2 (W7): bench harness + engine comparison
- [ ] M3 (W10): GKE serving platform v1
- [ ] M4 (W12): multi-host + disaggregated serving report
- [ ] M5 (W16): agent runtime with MCP and A2A
- [ ] M6 (W19): sandboxed, durable, budgeted agents
- [ ] M7 (W22): secure, observable platform + agent workload benchmark
- [ ] M8 (W24): capstone demo + architecture doc
- [ ] M9 (W26): interview kit

## Key measurements

| Week | Machine | What | Setup | Result |
|---|---|---|---|---|
| 1 | mac | Embedding Matrix Size | Llama 3 8B (128k vocab x 4096 dims, FP16) | ~1.05 GB in GPU VRAM |
| 1 | mac | Layer Projection Weights | W_Q, W_K, W_V, W_O (each 4096 x 4096, FP16) | ~134 MB per layer (33.5 MB each) |
| 1 | mac | Layer Total Weights | Attention (134 MB) + FFN SwiGLU (268 MB) FP16 | ~402 MB per layer (~13 GB total) |
| 1 | mac | KV Cache Footprint | 2 (K, V) x 32 layers x 4096 dims x 2 bytes | ~0.5 MB per token (2 GB for 4k context) |
| 1 | mac | Llama 3.1 8B INT4 Ollama | Apple M4 Pro (64 GB, 273 GB/s bandwidth) | Prefill: 104.3 tok/s, Decode: 45.1 tok/s (~81% bus eff) |
| 1 | mac | Gemma 4 31B KV Cache | 60 layers, 16 KV heads, avg head_dim 299 (FP16) | ~1.10 MB / token (~35.0 GB per user at 32k context) |
| 1 | mac | Llama 3 8B GQA KV Footprint | 32 layers, 8 KV heads, head_dim 128, FP16 | 128 KB / token (~1.05 GB per user at 8k context) |
| 2 | mac | Single-Token Decode Compute vs Bus Time | Llama-3-8B FP16 (16 GB) on NVIDIA H100 | Compute: 16 us, Bus Transfer: 4,800 us (99.7% ALU idle) |
| 2 | mac | Prompt Prefill Compute vs Bus Time | 4,000 tokens on Llama-3-8B FP16, NVIDIA H100 | Compute: 65 ms, Bus Transfer: 4.8 ms (100% ALU saturation) |
| 2 | mac | PCIe vs Internal HBM Bus Speed | PCIe Gen4 (64 GB/s) vs H100 HBM3 (3,350 GB/s) | PCIe is ~50x slower than internal GPU memory bus |

## Open questions

- Where should the project code live long-term? (see `decisions.md`)
