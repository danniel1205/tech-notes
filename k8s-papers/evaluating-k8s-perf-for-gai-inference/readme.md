# [Paper] Evaluating Kubernetes Performance for GAI Inference

## Summary

This paper, titled "Evaluating Kubernetes Performance for GenAI Inference:
From Automatic Speech Recognition to LLM Summarization," explores how emerging
Kubernetes-native tools can be combined to efficiently manage complex Generative
AI (GenAI) inference workloads.

To demonstrate this, the authors constructed a multi-stage AI workflow that
processes a dataset of corporate earnings calls. First, the system transcribes
the audio using Automatic Speech Recognition (ASR) via OpenAI's Whisper models
(batch inference). Then, it feeds those transcripts into a Large Language Model
(Qwen3-8B) for summarization (online, real-time inference).

The evaluation focuses on system-level performance, efficiency, and operational
overhead rather than model accuracy. It highlights three specific Kubernetes
components:

1. **Kueue for Batch Scheduling and Queueing**

   **The Problem:** The default Kubernetes scheduler lacks job-level admission
   control and cannot handle complex AI batch jobs efficiently.

   **The Solution:** Kueue introduces a hierarchical queueing model with
   resource sharing, workload priorities, and preemption capabilities to
   efficiently manage large batch jobs.

   **Key Findings:** Kueue provided predictable, deterministic scheduling
   compared to native Kubernetes. By enabling a setup with priorities and
   preemption, Kueue reduced the total makespan (the time to complete all jobs)
   by up to 15%. Furthermore, its admission control logic added negligible
   overhead, finalizing decisions in under 25 milliseconds even at the 99th
   percentile.

2. **Dynamic Accelerator Slicer (DAS) for GPU Optimization**

   **The Problem:** Traditional static GPU partitioning methods lead to resource
   fragmentation, poor utilization, and require service disruptions to
   reconfigure.

   **The Solution:** DAS provides on-demand, just-in-time GPU slicing (using
   NVIDIA MIG) tailored to the specific needs of incoming AI workloads,
   allocating and releasing slices dynamically.

   **Key Findings:** By slicing the 8 GPUs into smaller instances, DAS
   increased the number of concurrently running transcription jobs from 8 to 25.
   While individual jobs ran slightly slower on a slice compared to a full GPU,
   the massive increase in parallel execution reduced the mean job completion
   time by 36% (from 28 minutes down to 18 minutes). It also nearly tripled the
   average Tensor Core and DRAM utilization.

3. **Gateway API Inference Extension (GAIE) and llm-d for Distributed Inference**

   **The Problem:** Standard Kubernetes load balancing distributes traffic
   evenly, which is highly inefficient for LLMs because they rely on Key-Value
   (KV) caches. Sending similar prompts to different replicas wastes GPU cycles
   on redundant computations.

   **The Solution:** GAIE introduces specialized routing primitives, and the
   llm-d framework utilizes a "Precise Prefix-Cache Aware Scheduling" strategy.
   This routes incoming inference requests to the specific backend replica that
   already holds the matching prefix tokens in its cache.

   **Key Findings:** Cache-aware scheduling vastly outperformed random
   scheduling. It reduced the 99th percentile Time to First Token (TTFT) tail
   latency by up to 90% (meaning TTFT was about 6.25 times faster on average).
   Furthermore, end-to-end request latency was reduced by up to 6 seconds at
   the 99th percentile.

**Overall Conclusion:** The paper demonstrates that when these three components
(Kueue, DAS, and GAIE) are used together, they complement core Kubernetes to
create a highly efficient, cohesive platform capable of orchestrating both batch
and online GenAI inference workloads at scale.

## Try-out Kueue

[Guide](resources/kueue_kind_guide.md)

## Try-out DAS

[Guide](resources/das_guide.md)
