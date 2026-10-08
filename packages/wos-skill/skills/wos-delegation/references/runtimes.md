# Native runtime delegation

Claude Code: use the native Agent/subagent mechanism available in that client/version; configure instructions, permitted tools and model there. Install project skills in .claude/skills or user skills in ~/.claude/skills. Instructions alone do not configure permissions.

Codex: use native delegation tools when available and authorized. If the tool supports history selection, explicitly choose no inherited conversation for a fresh worker and supply sufficient task constraints. This environment's shared workspace is not a separate filesystem per worker. Install in .agents/skills or ~/.agents/skills. Do not assume every Codex client exposes the same spawn interface.

Hermes: use native delegation only if enabled in the actual runtime/tool catalog. Discover skills in .hermes/skills for supported project-aware versions, ~/.hermes/skills for user scope, or the configured external/profile directory. Do not mistake a Hermes model alone for an agent runtime. An agent's context management and tool authority belong to its host.

OpenClaw: use the configured sessions/subagents tools only if the installed version and agent policy expose them. Workspace skills live in <workspace>/skills, with managed user skills normally ~/.openclaw/skills; profiles may override paths. Session creation does not by itself prove separate worktrees, tool permissions or lack of inherited context.

Other agents: choose --agent generic --dir for the Agent Skills discovery root, or supply a reader adapter for SKILL.md. Delegate sequentially if no native spawn mechanism exists. Never claim all-runtime certification based on successful file installation.

Every worker receives only scope/task IDs plus necessary intent/constraints, fetches current WOS context, acquires its own authorized WorkContract, persists evidence/result, and returns a compact receipt. Separate model contexts can reduce supervisor context but may increase total tokens. Measure both.
