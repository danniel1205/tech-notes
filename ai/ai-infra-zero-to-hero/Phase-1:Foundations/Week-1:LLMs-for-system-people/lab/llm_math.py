#!/usr/bin/env python3
"""
LLM Inference Capacity & Arithmetic Calculator.

Computes:
1. Model Weight Memory (GB) across precisions (FP16, FP8, INT4)
2. KV Cache Footprint per Token (Bytes/token) taking GQA/MQA into account
3. Max Concurrent Sequences (Batch Capacity) for a given GPU
4. Forward FLOPs per Token
5. Decode Tokens/sec Roofline based on hardware memory bandwidth
"""

import argparse
import sys
from dataclasses import dataclass
from typing import Optional, Dict


@dataclass
class ModelConfig:
    name: str
    params_billion: float
    num_layers: int
    hidden_size: int
    num_attention_heads: int
    num_kv_heads: int
    head_dim: int
    vocab_size: int


@dataclass
class HardwareConfig:
    name: str
    vram_gb: float
    bandwidth_gbs: float
    peak_tflops_bf16: float


# Standard preset models
PRESET_MODELS: Dict[str, ModelConfig] = {
    "llama3-8b": ModelConfig(
        name="Llama-3-8B",
        params_billion=8.03,
        num_layers=32,
        hidden_size=4096,
        num_attention_heads=32,
        num_kv_heads=8,  # GQA: 8 KV heads vs 32 query heads
        head_dim=128,
        vocab_size=128256,
    ),
    "llama3-70b": ModelConfig(
        name="Llama-3-70B",
        params_billion=70.6,
        num_layers=80,
        hidden_size=8192,
        num_attention_heads=64,
        num_kv_heads=8,  # GQA: 8 KV heads vs 64 query heads
        head_dim=128,
        vocab_size=128256,
    ),
    "mistral-7b": ModelConfig(
        name="Mistral-7B-v0.1",
        params_billion=7.24,
        num_layers=32,
        hidden_size=4096,
        num_attention_heads=32,
        num_kv_heads=8,  # GQA: 8 KV heads
        head_dim=128,
        vocab_size=32000,
    ),
    "gemma2-27b": ModelConfig(
        name="Gemma-2-27B",
        params_billion=27.2,
        num_layers=46,
        hidden_size=4608,
        num_attention_heads=32,
        num_kv_heads=16,  # GQA: 16 KV heads vs 32 query heads
        head_dim=128,
        vocab_size=256000,
    ),
    "gemma4-31b": ModelConfig(
        name="Gemma-4-31B",
        params_billion=30.7,
        num_layers=60,
        hidden_size=5376,
        num_attention_heads=32,
        num_kv_heads=16,  # GQA: 16 KV heads
        head_dim=299,  # Heterogeneous avg: 5 sliding layers (256) to 1 global layer (512)
        vocab_size=262144,
    ),
}

# Standard preset hardware
PRESET_HARDWARE: Dict[str, HardwareConfig] = {
    "m4-pro": HardwareConfig(
        name="Apple M4 Pro (Unified Memory)",
        vram_gb=64.0,
        bandwidth_gbs=273.0,
        peak_tflops_bf16=27.0,
    ),
    "m4-max": HardwareConfig(
        name="Apple M4 Max (Unified Memory)",
        vram_gb=128.0,
        bandwidth_gbs=546.0,
        peak_tflops_bf16=54.0,
    ),
    "l4": HardwareConfig(
        name="NVIDIA L4 (GCE)",
        vram_gb=24.0,
        bandwidth_gbs=300.0,
        peak_tflops_bf16=121.0,
    ),
    "a100-80gb": HardwareConfig(
        name="NVIDIA A100 SXM 80GB",
        vram_gb=80.0,
        bandwidth_gbs=2039.0,
        peak_tflops_bf16=312.0,
    ),
    "h100": HardwareConfig(
        name="NVIDIA H100 SXM 80GB",
        vram_gb=80.0,
        bandwidth_gbs=3350.0,
        peak_tflops_bf16=989.0,
    ),
    "h200": HardwareConfig(
        name="NVIDIA H200 SXM 141GB",
        vram_gb=141.0,
        bandwidth_gbs=4800.0,
        peak_tflops_bf16=989.0,
    ),
    "b200": HardwareConfig(
        name="NVIDIA B200 SXM 192GB",
        vram_gb=192.0,
        bandwidth_gbs=8000.0,
        peak_tflops_bf16=2250.0,
    ),
    "b300": HardwareConfig(
        name="NVIDIA B300 SXM 288GB (Blackwell Ultra)",
        vram_gb=288.0,
        bandwidth_gbs=9000.0,
        peak_tflops_bf16=2500.0,
    ),
}

PRECISION_BYTES = {
    "fp32": 4.0,
    "fp16": 2.0,
    "bf16": 2.0,
    "fp8": 1.0,
    "int4": 0.5,
}


def calculate_weight_memory_gb(params_billion: float, precision: str) -> float:
    """
    Calculate model weights in memory (GB).
    
    Why:
    Each parameter is a floating-point or integer number in memory.
    The memory footprint is simply the parameter count multiplied by the
    number of bytes each parameter occupies (e.g., FP16 = 2 bytes, FP8 = 1 byte).
    """
    bytes_per_param = PRECISION_BYTES[precision.lower()]
    raw_gb = params_billion * bytes_per_param
    
    # Why INT4 has overhead:
    # Sub-byte formats like INT4 (0.5 bytes/param) cannot be stored in isolation.
    # Production quantization methods (AWQ, GPTQ) group weights (e.g., groups of 64 or 128)
    # and store FP16 scale factors and zero-point offsets for each group, adding ~10-15% overhead.
    if precision.lower() == "int4":
        raw_gb *= 1.15
    return raw_gb


def calculate_kv_cache_bytes_per_token(
    num_layers: int,
    num_kv_heads: int,
    head_dim: int,
    precision: str = "fp16",
) -> int:
    """
    Calculate memory required for 1 token's KV cache across all layers.
    
    Formula: 2 * num_layers * num_kv_heads * head_dim * bytes_per_element
    
    Why each term is multiplied:
    1. Factor of 2:
       Attention requires saving BOTH the Key vector (K) and the Value vector (V)
       for every past token.
    2. num_layers:
       Every transformer layer has its own unique projection weights (W_K, W_V)
       and maintains its own separate, independent KV cache buffer in VRAM.
    3. num_kv_heads * head_dim:
       In Multi-Head Attention, each head has width `head_dim` (e.g., 128 floats).
       Multiplying by `num_kv_heads` gives the total count of floating-point numbers
       contained inside one Key (or Value) vector for that layer.
    4. bytes_per_elem:
       Converts the count of floats into physical bytes in VRAM (FP16 = 2 bytes, FP8 = 1 byte).
    """
    bytes_per_elem = PRECISION_BYTES[precision.lower()]
    return int(2 * num_layers * num_kv_heads * head_dim * bytes_per_elem)


def calculate_flops_per_token(params_billion: float) -> float:
    """
    Compute forward pass FLOPs per token.
    
    Why 2 * parameters:
    Matrix multiplication (GEMM) calculates dot products of the form y = W * x.
    Every single parameter multiplication involves:
    1. One multiplication (W_ij * x_j)
    2. One addition / accumulation (+ sum)
    This operation is called a Multiply-Accumulate (MAC), which equals exactly 2 FLOPs.
    In the forward pass (inference), every model weight is used once per token,
    yielding approximately 2 * Parameter_Count FLOPs per generated token.
    """
    return 2.0 * params_billion * 1e9


def calculate_profile(
    model: ModelConfig,
    hardware: HardwareConfig,
    precision: str = "fp16",
    kv_precision: str = "fp16",
    context_length: int = 4096,
    overhead_gb: float = 2.0,
) -> dict:
    """Calculate the complete systems inference profile."""
    # 1. Weights Memory:
    # Physical footprint of model parameters permanently resident in GPU VRAM.
    weight_gb = calculate_weight_memory_gb(model.params_billion, precision)

    # 2. KV Cache per Token & per User:
    # Memory cost to remember 1 token across all layers.
    kv_bytes_per_tok = calculate_kv_cache_bytes_per_token(
        model.num_layers,
        model.num_kv_heads,
        model.head_dim,
        kv_precision,
    )
    # Why (kv_bytes_per_tok * context_length):
    # This is the memory footprint of ONE active user/stream across their entire conversation
    # history (Prompt tokens + Generated tokens). Like a Pod's memory request in Kubernetes.
    kv_mb_per_user = (kv_bytes_per_tok * context_length) / (1024 * 1024)
    kv_gb_per_user = kv_mb_per_user / 1024

    # 3. Allocatable GPU capacity:
    # Kubernetes analogy: Allocatable Node Memory = Capacity - System Reserve - Pod Baseline.
    # Total VRAM minus static weights minus 2GB runtime scratchpad (activation tensors & CUDA runtime).
    allocatable_vram = hardware.vram_gb - weight_gb - overhead_gb
    
    # Why allocatable_vram / kv_gb_per_user:
    # How many concurrent user streams can be active before the GPU exhausts memory (OOM).
    if allocatable_vram > 0:
        max_concurrent_sequences = int(allocatable_vram / kv_gb_per_user)
    else:
        max_concurrent_sequences = 0

    # 4. Forward FLOPs per token:
    flops_per_tok = calculate_flops_per_token(model.params_billion)

    # 5. Decode Roofline Speed (Tokens/sec):
    # Why Bandwidth / Weights:
    # In decode (batch size 1), to output 1 token, the GPU compute cores must stream
    # all model weights from VRAM across the memory bus once.
    # Time_Per_Token = Model_Weight_GB / Memory_Bandwidth_GBs.
    # Inverting time gives Tokens_Per_Second = Memory_Bandwidth_GBs / Model_Weight_GB.
    roofline_peak_tok_s = hardware.bandwidth_gbs / weight_gb
    
    # Why multiply by 0.75:
    # Real-world memory efficiency is ~70-80% due to kernel launch latency,
    # memory controller page arbitration, and concurrent KV cache traffic.
    roofline_real_tok_s = roofline_peak_tok_s * 0.75

    return {
        "model": model.name,
        "hardware": hardware.name,
        "precision": precision.upper(),
        "kv_precision": kv_precision.upper(),
        "context_length": context_length,
        "weight_gb": weight_gb,
        "kv_bytes_per_token": kv_bytes_per_tok,
        "kv_gb_per_user": kv_gb_per_user,
        "allocatable_kv_vram_gb": max(0.0, allocatable_vram),
        "max_concurrent_sequences": max_concurrent_sequences,
        "flops_per_token": flops_per_tok,
        "roofline_peak_tok_s": roofline_peak_tok_s,
        "roofline_real_tok_s": roofline_real_tok_s,
    }


def print_report(res: dict):
    """Print an ASCII systems report."""
    print("=" * 72)
    print(f" LLM INFERENCE PROFILE: {res['model']} on {res['hardware']}")
    print("=" * 72)
    print(f" Precision:             {res['precision']} weights / {res['kv_precision']} KV cache")
    print(f" Target Context Window: {res['context_length']:,} tokens")
    print("-" * 72)
    print(" 1. VRAM & FOOTPRINT")
    print(f"    • Model Weights:    {res['weight_gb']:.2f} GB")
    print(f"    • KV Cache / Token: {res['kv_bytes_per_token']:,} Bytes ({res['kv_bytes_per_token'] / 1024:.1f} KB)")
    print(f"    • KV Cache / User:  {res['kv_gb_per_user']:.3f} GB (at {res['context_length']:,} tokens)")
    print("-" * 72)
    print(" 2. GPU CONCURRENCY & CAPACITY")
    print(f"    • Available KV RAM: {res['allocatable_kv_vram_gb']:.2f} GB (Total - Weights - 2GB reserved)")
    if res['allocatable_kv_vram_gb'] <= 0:
        print("    • Max Concurrency:  0 (MODEL DOES NOT FIT IN GPU VRAM!)")
    else:
        print(f"    • Max Concurrency:  {res['max_concurrent_sequences']} concurrent active streams")
    print("-" * 72)
    print(" 3. DECODE SPEED (MEMORY-BANDWIDTH BOUND)")
    print(f"    • Theoretical Peak: {res['roofline_peak_tok_s']:.1f} tokens/sec")
    print(f"    • Expected (~75%):  {res['roofline_real_tok_s']:.1f} tokens/sec")
    print("-" * 72)
    print(" 4. COMPUTE REQUIREMENTS")
    print(f"    • Forward FLOPs:    {res['flops_per_token'] / 1e9:.2f} GFLOPs / token")
    print("=" * 72)


def main():
    parser = argparse.ArgumentParser(description="LLM Inference Arithmetic & Capacity Calculator")
    parser.add_argument("--model", type=str, default="llama3-8b", choices=list(PRESET_MODELS.keys()))
    parser.add_argument("--gpu", type=str, default="m4-pro", choices=list(PRESET_HARDWARE.keys()))
    parser.add_argument("--precision", type=str, default="int4", choices=list(PRECISION_BYTES.keys()))
    parser.add_argument("--kv-precision", type=str, default="fp16", choices=list(PRECISION_BYTES.keys()))
    parser.add_argument("--context", type=int, default=4096, help="Target context length in tokens")
    parser.add_argument("--overhead", type=float, default=2.0, help="VRAM runtime scratchpad reserve (GB)")

    args = parser.parse_args()

    model = PRESET_MODELS[args.model]
    hardware = PRESET_HARDWARE[args.gpu]

    res = calculate_profile(
        model=model,
        hardware=hardware,
        precision=args.precision,
        kv_precision=args.kv_precision,
        context_length=args.context,
        overhead_gb=args.overhead,
    )
    print_report(res)


if __name__ == "__main__":
    main()
