# Agent instructions: AI Infra Zero to Hero

This folder is the **shared memory** for a 26-week AI infra learning plan. The learner works on two machines and uses a different agent on each:

| Machine | Agent | Used for |
|---|---|---|
| `mac` (Mac Mini, Apple Silicon) | Antigravity | Local labs: Ollama / llama.cpp / MLX, agent and MCP development, `kind` |
| `cloudtop` (Linux workstation) | Jetski | Cloud labs: GKE, GCE GPU VMs; also local CPU-only labs |

Neither agent can see the other's chat history. **Everything the other machine needs to know must be written into this folder and pushed to git.**

## Files in this folder

| File | Purpose | Who writes it |
|---|---|---|
| `readme.md` | The 26-week plan: weekly goals, papers, labs, interview questions | Rarely changed; only on the learner's request |
| `AGENTS.md` | These instructions (always-on) | Only on the learner's request |
| `decisions.md` | Why the plan looks the way it does: goals, scope, environments | Append when a new decision is made |
| `PROGRESS.md` | Current state: current week, checklist, measurements, next steps per machine | Update at the end of every session |
| `sessions/` | One log per session, named `YYYY-MM-DD-<machine>-wNN-<topic>.md` | Create a new file at the end of every session; never edit old ones |
| `notes/` | The learner's weekly learning notes (published content) | The learner writes; the agent reviews and suggests |
| `sync.sh` | `./sync.sh start` pulls; `./sync.sh end "<msg>"` commits and pushes this folder only | Run it; don't change it |

## Sync protocol (follow it every time)

### When the learner says "start session" (or starts working on the plan)

1. Run `./sync.sh start` from this folder. If it fails (auth, conflict), stop and help fix it before continuing.
2. Read `PROGRESS.md`, then the **2–3 most recent** files in `sessions/`, then the current week's section in `readme.md`.
3. Detect the machine: `uname` = `Darwin` means `mac`, otherwise `cloudtop`.
4. Give a short briefing (at most 8 lines): where we are, what happened last session and on which machine, what is next on **this** machine, and any open questions carried over.
5. If the next task on the list needs the other machine (e.g. a GKE lab while on the Mac), say so and suggest something that can be done here instead.

### During the session

- Keep a running list of what was done, numbers measured (with units and setup), things learned, and open questions. These go into the session log.
- If a new plan-level decision is made (scope, tools, schedule), record it in `decisions.md` with the date.

### When the learner says "end session"

1. Create `sessions/YYYY-MM-DD-<machine>-wNN-<topic>.md` using the template at the bottom of this file.
2. Update `PROGRESS.md`: tick checklist items, add key measurements, and rewrite the "Next up" section for **both** machines.
3. Show the learner a 3–5 line summary, then run `./sync.sh end "session: <machine> wNN <topic>"`.
4. Confirm the push succeeded. If it failed, say so clearly; an unpushed session is invisible to the other machine.

## How to teach this learner

- **Background:** deep Kubernetes expertise, near-zero AI knowledge. Use K8s analogies (scheduler, kubelet, controllers, PVs) where they genuinely fit.
- **Teach, don't do.** Guide the learner through labs; don't silently write all the code. For calculators and labs, ask the learner to **predict numbers first**, then measure, then explain the gap.
- **Depth:** systems-level ML only (KV cache, prefill vs decode, FLOPs and memory-bandwidth math, quantization). Skip training math and backprop.
- **Interview prep is a first-class goal.** End each week with that week's interview questions from `readme.md`; have the learner answer out loud or in writing; give direct, specific feedback.
- **Notes are the learner's own words.** Help structure and fact-check `notes/`, but don't ghost-write them.
- Keep answers concise. Cite sources for factual claims about fast-moving projects (vLLM, llm-d, GKE features) and flag anything that may be outdated.

## Safety rules (this is a PUBLIC GitHub repo)

- **Never** write employer-internal information here: internal hostnames, internal URLs, internal tool names, credentials, tokens, or pasted internal logs.
- Refer to cloud projects as `$PROJECT_ID`; never write real project IDs, billing accounts or IPs.
- Only `sync.sh` commits, and it stages **this folder only**. Never commit files outside `ai/ai-infra-zero-to-hero/`.
- Markdown must pass the repo linter (`md_style.rb`): lines ≤ 200 chars, blank lines around headings, lists and code blocks.

## Session log template

```markdown
# <YYYY-MM-DD> · <machine> · Week <N>: <topic>

## Done
- ...

## Measurements
| What | Setup | Result |
|---|---|---|

## Learned
- ...

## Open questions
- ...

## Next steps
- On mac: ...
- On cloudtop: ...
```
