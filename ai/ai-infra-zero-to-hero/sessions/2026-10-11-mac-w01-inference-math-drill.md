# 2026-10-11 · mac · Week 1: Inference Formulas, GPU Microarchitecture, and Interview Drill

## Done

- Implemented and verified `Phase-1:Founations/Week-1:LLMs-for-system-people/lab/llm_math.py` supporting Gemma 4 and chips from Apple M4 Pro to NVIDIA B300, with comprehensive systems-level inline comments.
- Moved weekly lab scripts under each week's folder per Decision #17 in `decisions.md`.
- Documented "The 4 Core Inference Formulas" in Section 9 of `Phase-1:Founations/Week-1:LLMs-for-system-people/readme.md` (weights memory, KV cache per token, max concurrency, decode roofline).
- Conducted capacity planning case study: Gemma 4 (31B) processing 10MB PDFs for 100K concurrent users on NVIDIA B300 GPUs (highlighting KV cache dominance and VRAM limits).
- Deep-dived into GPU microarchitecture: GEMM tiling, VRAM-to-SRAM weight streaming, double buffering, and memory latency hiding.
- Completed Week 1 Interview Drill: Questions 1 (KV cache rationale) and 2 (Prefill compute-bound vs Decode bandwidth-bound).

## Measurements

| What | Setup | Result |
|---|---|---|
| Gemma 4 31B KV Cache Footprint | 60 layers, 16 KV heads, avg head_dim 299 (FP16) | ~1.10 MB / token (~35.0 GB per user at 32k context) |
| B300 Sizing for 100K streams | 100k streams at 32k context (FP16), Gemma 4 31B | 16,667 B300 GPUs (~2,084 8-GPU DGX nodes) |
| Llama 3 8B INT4 on Mac M4 Pro | Ollama local inference | Prefill: 104.34 tok/s; Decode: 45.09 tok/s (~81% bus efficiency) |

## Learned

- Tile weights transfer from VRAM to on-chip SRAM identical times in Prefill and a single Decode step; Prefill hides transfer latency via arithmetic intensity, whereas Decode starves Tensor Cores.
- Double buffering (Ping-Pong buffers in SRAM) allows Tensor Cores to compute on Buffer A while memory controllers (like TMA) fetch the next tile into Buffer B in parallel.
- At production scale, GPU cluster sizing is almost entirely constrained by KV cache VRAM capacity rather than model weights.

## Open questions

- Complete Interview Drill Question 3 (Llama-3-8B KV cache sizing at 8k context) and Question 4 (GQA/MQA systems trade-offs).

## Next steps

- On mac: Complete Week 1 Interview Drill (Questions 3 and 4), then advance to Week 2 (GPU hardware, memory hierarchy, and rooflines).
- On cloudtop: Set up GitHub auth (`gh auth login` + `gh auth setup-git`); request GPU quota (L4 now; 8x H100/A3 for later weeks).
