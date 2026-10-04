# 2026-10-04 · mac · Week 1: Inference Deep Dive

## Done

- Walked through the complete end-to-end inference lifecycle using the prompt "what is quantum computing".
- Detailed hardware boundaries: Host CPU tokenization (BPE) -> PCIe 16-byte transfer -> GPU VRAM Embedding Matrix lookup (W_E).
- Analyzed Layer 1 Attention math: parameter matrices (W_Q, W_K, W_V), why V also has 32 heads (128 floats each), and GPU single-GEMM optimization.
- Clarified why V_4 is immutable in the KV Cache while Blended_V_4 is an ephemeral scratchpad vector.
- Clarified why W_O is needed to cross-correlate 32 siloed head outputs and align coordinates, and why residual addition (x + Attention_Output) prevents identity loss and vanishing gradients.
- Detailed why all prompt tokens travel through all 32 layers in parallel during Prefill, priming 32 distinct KV cache buffers in GPU VRAM.
- Detailed LM Head un-embedding (W_unembed / lm_head.weight) converting the final layer vector into 128k logits to sample the first token.
- Updated Phase-1:Founations/Week-1:LLMs-for-system-people/readme.md with Section 2 enhancements and a new Section 8 containing the full end-to-end walkthrough.

## Measurements

| What | Setup | Result |
|---|---|---|
| Embedding Matrix Size | Llama 3 8B (128k vocab × 4096 dims, FP16) | ~1.05 GB in GPU VRAM |
| Attention Weights per Layer | W_Q, W_K, W_V, W_O (each 4096 × 4096, FP16) | ~134 MB per layer (33.5 MB each) |
| KV Cache Footprint | 2 (K, V) × 32 layers × 4096 dims × 2 bytes | ~0.5 MB per token (2 GB for 4k context) |

## Learned

- The model server binary (vLLM/Triton) is stateless; model weights and tokenizer dictionary reside on mounted storage/PVC.
- Every layer in a transformer has its own unique, independent set of projection weights (W_Q, W_K, W_V, W_O) and its own distinct KV cache buffer in GPU VRAM.
- Prefill is compute-bound (saturating tensor cores with 2D matrix multiplications over all tokens), while Decode is memory-bandwidth-bound (streaming weights and KV caches for a single vector).

## Open questions

- Deep dive into Step 4 (Decode phase loop and KV cache streaming) and Room 2 (Feed-Forward Network / FFN internals).
- Local lab: install Ollama on Mac Mini (Apple Silicon), benchmark an ~8B model, and record baseline tokens/s.

## Next steps

- On mac: Cover Decode phase step-by-step and FFN internals; install Ollama, benchmark 8B model tok/s; begin lab/calculators/llm_math.py.
- On cloudtop: Set up GitHub auth (gh auth login + gh auth setup-git); request GPU quota (L4).
