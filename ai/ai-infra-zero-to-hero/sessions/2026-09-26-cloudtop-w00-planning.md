# 2026-09-26 · cloudtop · Week 0: Planning

## Done

- Ran a structured interview (`/grill-me`) to define the goal, scope, schedule and environments. All decisions are recorded in `../decisions.md`.
- Ran parallel web research on inference infra, agentic platforms, papers and interview prep, then wrote the 26-week plan (`../readme.md`).
- Checked GPU fallback options for stockouts (not applied to the plan yet; see decision 15).
- Set up this folder as the shared memory between mac (Antigravity) and cloudtop (Jetski).

## Measurements

| What | Setup | Result |
|---|---|---|
| cloudtop hardware | lscpu / free | 8 vCPU Xeon 2.2 GHz (AVX2, no AVX-512/AMX), 31 GB RAM, no GPU; Docker + kind installed |

## Learned

- The cloudtop can run CPU-only local labs (small models via llama.cpp/Ollama, agent dev against remote endpoints, kind), but not MLX, and 8B models will be slow.
- Many plan resources were verified only via search summaries; see Appendix C in `../readme.md` before relying on a flagged claim.

## Open questions

- Where should project code live long-term (separate public repo vs. this folder)?
- Is the cloudtop the only "work" machine, or is there also a separate laptop?

## Next steps

- On mac: clone `tech-notes` (branch `kb/ai-infra`), open `ai/ai-infra-zero-to-hero/` in Antigravity, say "start session", begin Week 1.
- On cloudtop: fix GitHub push auth (`gh auth login`, `gh auth setup-git`); request GPU quota.
