# Progress

Last updated: 2026-10-08 by mac (transformer internals and architecture deep dive session). Current week: **Week 1 (in progress)**.

## Next up

- **On mac:** Install Ollama, benchmark ~8B model tok/s; implement `calculators/llm_math.py` (predict parameters and tok/s first, then measure); run Week 1 interview drill.
- **On cloudtop:** Set up GitHub auth (`gh auth login` + `gh auth setup-git`) so `sync.sh end` can push. Request GPU quota (L4 now; 8×H100/A3 for weeks 7, 11, 12).

## Setup checklist

- [x] Plan written (`readme.md`)
- [x] Sync folder created (`AGENTS.md`, `decisions.md`, `PROGRESS.md`, `sessions/`, `sync.sh`)
- [ ] cloudtop: GitHub auth works for push
- [x] mac: `tech-notes` cloned on branch `kb/ai-infra`, folder opened in Antigravity
- [ ] GPU quota requested (L4, A3/H100, optional TPU v6e)

## Weekly checklist

| Week | Topic | Paper | Lab | Note | Drill |
|---|---|---|---|---|---|
| 1 | LLMs for systems people | [x] | [ ] | [x] | [ ] |
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
| 1 | mac | Embedding Matrix Size | Llama 3 8B (128k vocab × 4096 dims, FP16) | ~1.05 GB in GPU VRAM |
| 1 | mac | Layer Projection Weights | W_Q, W_K, W_V, W_O (each 4096 × 4096, FP16) | ~134 MB per layer (33.5 MB each) |
| 1 | mac | Layer Total Weights | Attention (134 MB) + FFN SwiGLU (268 MB) FP16 | ~402 MB per layer (~13 GB total) |
| 1 | mac | KV Cache Footprint | 2 (K, V) × 32 layers × 4096 dims × 2 bytes | ~0.5 MB per token (2 GB for 4k context) |

## Open questions

- Where should the project code live long-term? (see `decisions.md`)
