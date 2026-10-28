<!-- ceo-plugin:start -->
## Command Selection Guide
Use automatically when the user requests orchestration via the CEO workflow or when coordination is required.

### How to use ceo-plugin:
In Windows:

| Environment | Exact Execution Syntax |
| :--- | :--- |
| PowerShell | & "$HOME\.ceo-plugin\bin\ceo-plugin.exe" |
| CMD | "%USERPROFILE%\.ceo-plugin\bin\ceo-plugin.exe" |
| Git Bash | "$HOME/.ceo-plugin/bin/ceo-plugin.exe" |

These commands return documentation; they do not create agents or execute the described workflow.

### 1. Primary Lane Commands (Loads Core SKILL.md Only)
| When you need to... | Use command |
| :--- | :--- |
| Load the ceo SKILL instructions | `ceo-plugin ceo` |
| Load the system-architect SKILL instructions | `ceo-plugin system-architect` |
| Load the architect SKILL instructions | `ceo-plugin architect` |
| Load the coder SKILL instructions | `ceo-plugin coder` |
| Load the qa SKILL instructions | `ceo-plugin qa` |
| Load the supervisor SKILL instructions | `ceo-plugin supervisor` |
| Load the mechanical-planner SKILL instructions | `ceo-plugin mechanical-planner` |
| Load the strategic-planner SKILL instructions | `ceo-plugin strategic-planner` |

### 2. CEO — Orchestration & Coordination Subcommands
| When you need to... | Use command |
| :--- | :--- |
| Read the session opening prompt template | `ceo-plugin ceo Template-Prompt-for-Opening-a-Session` |
| Read authority and command rules | `ceo-plugin ceo authority-and-command` |
| Read fast-block and fast-reject handling rules | `ceo-plugin ceo how-to-handle-fast-block-fast-reject` |
| Read lane spawning instructions for Antigravity | `ceo-plugin ceo how-to-spawn-lane-in-antigravity` |
| Read lane spawning instructions for Codex | `ceo-plugin ceo how-to-spawn-lane-in-codex` |
| Read lane and skill coordination rules | `ceo-plugin ceo lane-and-skill-coordination` |
| Read recovery and documentation rules | `ceo-plugin ceo recovery-and-documentation` |
| Read standby rules | `ceo-plugin ceo standby-rules` |
| Read subagent reporting and handoff rules | `ceo-plugin ceo subagent-reporting-and-handoff` |
| Read timer rules | `ceo-plugin ceo timer-rules` |
| Read user intervention rules | `ceo-plugin ceo user-intervention` |

### 3. System Architect — Architecture & SPEC Subcommands
| When you need to... | Use command |
| :--- | :--- |
| Read architect review coordination rules | `ceo-plugin system-architect architect-review-coordination` |
| Read interactive architecture discussion and alignment rules | `ceo-plugin system-architect mode-0-interactive-architecture-discussion` |
| Read specification authoring and architecture rules | `ceo-plugin system-architect mode-1-spec-authoring-and-architecture` |
| Read execution planning and job rules | `ceo-plugin system-architect mode-2-execution-planning-and-jobs` |
| Read architecture reporting and artifact lifecycle rules | `ceo-plugin system-architect reporting-and-artifact-lifecycle` |

### 4. QA — Verification Protocols & Quality Rules
| When you need to... | Use command |
| :--- | :--- |
| Read the backend QA protocol | `ceo-plugin qa backend-qa-protocol` |
| Read end-to-end data flow verification rules | `ceo-plugin qa e2e-data-flow` |
| Read QA evidence and blocker rules | `ceo-plugin qa evidence-and-blockers` |
| Read frontend automation and control sweep rules | `ceo-plugin qa frontend-automation-and-control-sweep` |
| Read the frontend coverage model | `ceo-plugin qa frontend-coverage-model` |
| Read frontend locale and internationalization rules | `ceo-plugin qa frontend-i18n-and-locale` |
| Read frontend runtime and build rules | `ceo-plugin qa frontend-runtime-and-build-rules` |
| Read QA reporting rules | `ceo-plugin qa reporting` |
| Read QA source of truth rules | `ceo-plugin qa source-of-truth-rules` |

### 5. Supervisor — Zero-Trust & Audit Rules
| When you need to... | Use command |
| :--- | :--- |
| Read claim and authority review rules | `ceo-plugin supervisor claim-and-authority` |
| Read evidence and source inspection rules | `ceo-plugin supervisor evidence-and-source-inspection` |
| Read invariant and resubmission rules | `ceo-plugin supervisor invariant-and-resubmission` |
| Read supervisor reporting rules | `ceo-plugin supervisor reporting` |

### 6. Strategic Planner — Plan & Ledger Templates
| When you need to... | Use command |
| :--- | :--- |
| Read the actual status ledger template | `ceo-plugin strategic-planner actual-status-template` |
| Read the benchmark ledger template | `ceo-plugin strategic-planner benchmark-template` |
| Read the evidence ledger template | `ceo-plugin strategic-planner evidence-template` |
| Read the strategic plan template | `ceo-plugin strategic-planner plan-template` |

`architect` shares references with `system-architect`; `mechanical-planner` shares templates with `strategic-planner`.
Valid lane commands/tools/prompts automatically update this guide in a resolved workspace. This is not a file watcher and does not force an existing AI session to reload instructions.
<!-- ceo-plugin:end -->
