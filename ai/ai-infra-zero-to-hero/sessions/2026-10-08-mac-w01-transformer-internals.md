# 2026-10-08 · mac · Week 1: Transformer Internals and Architecture Deep Dive

## Done

- Clarified two-room architecture per transformer layer: Room 1 (Attention + Residual Patch #1) and Room 2 (FFN SwiGLU + Residual Patch #2).
- Detailed Step 3 (Token 5 sampling via Final RMSNorm and W_unembed / LM Head) and Step 4 (Autoregressive Decode phase loop with single-token streaming and KV cache reuse).
- Clarified prompt token synchronization: all prompt tokens must be at the same layer simultaneously due to attention dependencies, while Pipeline Parallelism can pipeline across batches.
- Resolved all markdownlint formatting errors across `Phase-1:Founations/Week-1:LLMs-for-system-people/readme.md` and root `readme.md`, passing `make md-linter` cleanly.

## Measurements

| What | Setup | Result |
|---|---|---|
| Linter Status | `make md-linter` (Ruby mdl via docker with md_style.rb) | 0 errors (clean pass) |
| Llama 3 8B Layer Weights | Attention (134 MB) + FFN SwiGLU (3× matrices, ~268 MB) in FP16 | ~402 MB per layer (~13 GB total weights) |
| Prefill vs Decode Arithmetic Intensity | Llama 3 8B on L4 / H100 | Prefill: compute-bound GEMM; Decode: memory-bandwidth bound GEMV |

## Learned

- Within a single prompt, each transformer layer acts as a synchronization barrier because query Q_i depends on keys K_1..K_i produced at that exact layer.
- Decode phase processes only 1 new token vector per step, streaming the entire historical KV cache and all model weights from HBM/VRAM, making it strictly memory-bandwidth bound.
- Residual connections (x_updated = x + sublayer_output) run after both Room 1 (Attention) and Room 2 (FFN), preserving base token identity and gradient flow.

## Open questions

- Local lab: install Ollama on Mac Mini (Apple Silicon), benchmark an ~8B model, and record baseline tokens/s.
- Implement the standalone calculator script `calculators/llm_math.py` and verify predicted vs measured token speeds.

## Next steps

- On mac: Install Ollama, benchmark ~8B model tok/s; implement `calculators/llm_math.py`; complete Week 1 interview drill questions.
- On cloudtop: Set up GitHub auth (`gh auth login` + `gh auth setup-git`); request GPU quota (L4; 8×H100/A3 for later weeks).
