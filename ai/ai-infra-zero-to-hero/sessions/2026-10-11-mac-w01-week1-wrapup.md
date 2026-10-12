# 2026-10-11 · mac · Week 1: GQA Analysis, Prefill vs Decode Mechanics, and Week 1 Complete

## Done

- Finished Week 1 Interview Drill: Question 3 (Llama-3-8B KV cache footprint calculation: 128 KB/token, 1.05 GB for 8k context) and Question 4 (MHA vs MQA vs GQA systems trade-offs).
- Documented Section 10 ("Attention Head Architecture: MHA vs. MQA vs. GQA") in `Phase-1:Founations/Week-1:LLMs-for-system-people/readme.md`.
- Documented Section 11 ("Hardware Execution: Why Prefill is Compute-Bound and Decode is Memory-Bandwidth Bound") in `Phase-1:Founations/Week-1:LLMs-for-system-people/readme.md`.
- Validated all markdown formatting against `md_style.rb` (`make md-linter`), resolving all line-length warnings.
- Updated `PROGRESS.md`: checked off Week 1 Drill (`[x]`), marking Week 1 100% complete.

## Measurements

| What | Setup | Result |
|---|---|---|
| Llama-3-8B KV Footprint per Token | 32 layers, 8 KV heads (GQA), head_dim 128 (FP16) | 131,072 Bytes = 128 KB / token |
| Llama-3-8B User Session Footprint | 8,192 (8k) context window in FP16 | ~1.05 GB VRAM per active user |
| GQA vs MHA Memory Footprint | 8 KV heads vs 32 KV heads | 4x VRAM reduction (128 KB/tok vs 512 KB/tok) |

## Learned

- GQA clusters Query heads into groups sharing Key and Value heads (4:1 in Llama 3, 2:1 in Gemma 2/4), achieving 4x-8x VRAM reduction and 4x-8x higher Pod concurrency with near-zero quality loss.
- Prefill achieves high arithmetic intensity because 10 MB weight tiles in SRAM are multiplied across thousands of tokens (GEMM), hiding memory bus transfer latency behind compute.
- Decode achieves low arithmetic intensity because generating a single token requires streaming all model weights for only one operation per parameter (GEMV), leaving Tensor Cores starved.

## Open questions

- None for Week 1 (all paper, lab, notes, and drill milestones completed).

## Next steps

- On mac: Begin Week 2 (GPU hardware, memory hierarchy, and rooflines); read FlashAttention (2022) paper.
- On cloudtop: Set up GitHub auth (`gh auth login` + `gh auth setup-git`); request GPU quota (L4 now; 8x H100/A3 for later weeks);
  run Week 2 GCE L4 profiling lab (`nvidia-smi dmon`, `dcgmi dmon`, `nsys profile`).
