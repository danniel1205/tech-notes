# Decisions log

Why the plan in [readme.md](./readme.md) looks the way it does. Append new decisions at the bottom with a date.

## 2026-09-26: Initial planning session (cloudtop, Jetski)

| # | Topic | Decision |
|---|---|---|
| 1 | Target outcome | AI platform engineer with deep **inference** expertise **and agentic platform** expertise. Training infra only as supporting context |
| 2 | Purpose | Learning + preparing for AI infra / agentic platform job interviews. Company-agnostic |
| 3 | Interview formats | System design and domain deep dives first, then project storytelling built on real work. No LeetCode-style coding prep |
| 4 | Timeline | 26 weeks at ~3.5 hrs/week (~91 hrs) |
| 5 | Weekly rhythm | Weekday ~2 hr (30 min paper, 60 min concept, 30 min short lab). Weekend ~1.5–2.5 hr (main lab, note, interview drill) |
| 6 | ML depth | Systems level: transformers, attention/KV cache, prefill vs decode, FLOPs and bandwidth math, quantization. No training math |
| 7 | Agentic angle | Platform builder: running agents in production (runtime, isolation, MCP/tools, state, scaling, safety, observability) |
| 8 | Phases | Weeks 1–12 inference (not compressed), 13–22 agentic platform, 23–24 capstone, 25–26 interview polish |
| 9 | Papers | One paper per week matched to the topic, read for systems insight; 5-bullet summary each |
| 10 | Portfolio | One evolving project: an inference platform that grows into an agentic workflow platform |
| 11 | Environments | GKE = main platform. GCE GPU VM = low-level deep dives. Mac Mini = local dev loop. Cloud labs are driven from the cloudtop |
| 12 | Languages | Python for model, serving and agent code. Go for K8s controllers/operators |
| 13 | Plan format | Week-by-week: goal, paper, resources, labs, environment, deliverable, 3–5 interview questions; phase milestones |
| 14 | Cost | No cost guardrails needed |
| 15 | GPU scarcity | H100/A3 and TPU v6e may be out of stock. Add fallbacks (DWS flex-start, L4/A100, P/D over TCP, llm-d-inference-sim) only when needed |
| 16 | Sync between machines | This folder (public `tech-notes` repo) is the shared memory for mac/Antigravity and cloudtop/Jetski. No employer-internal info |

## Open decisions

- **Where the project code lives.** Options: a separate public repo (portfolio), or inside this folder. Until decided, keep Week 1–3 code under `lab/` in this folder.
