# WOS Skills

Portable Agent Skills for bounded WOS coordination, native subagent delegation, continuity and proof review. Runtime APIs and context isolation are consumer responsibilities.

After registry publication:

```sh
npx @a1b3rt0m3rcad0/wos-skill install --agent codex --dry-run
npx @a1b3rt0m3rcad0/wos-skill install --agent codex
npx @a1b3rt0m3rcad0/wos-skill install --agent claude
npx @a1b3rt0m3rcad0/wos-skill install --agent hermes
npx @a1b3rt0m3rcad0/wos-skill install --agent openclaw --project /path/to/openclaw-workspace
npx @a1b3rt0m3rcad0/wos-skill install --agent hermes --global
npx @a1b3rt0m3rcad0/wos-skill install --agent all --global
npx @a1b3rt0m3rcad0/wos-skill install --agent generic --dir /path/to/agent/skills
```

Use a pinned package version for repeatable installation. Before npm publication, `npm install /path/to/wos-skill-0.1.0.tgz` and `npx --no-install wos-skill ...` work offline with the release tarball. Nothing runs during npm installation. Run the explicit installer to copy skill bundles.

`update` installs the invoking package's version; it does not fetch latest. `uninstall` removes only unmodified managed bundles. `list`/`doctor` validate hashes and ownership. Edits and unmanaged collisions fail without overwriting files; preserve/move your edits yourself before retrying. Operations preflight all destinations and use per-bundle staging, but are not an atomic transaction across all directories; retry after an I/O failure. Concurrent installs into the same directory are unsupported. Symlink destinations are rejected.

Project defaults: Claude `.claude/skills`, Codex `.agents/skills`, Hermes `.hermes/skills`, OpenClaw `<workspace>/skills`. Global defaults: `~/.claude/skills`, `~/.agents/skills`, `~/.hermes/skills`, `~/.openclaw/skills`. Hermes project discovery is documented in the upstream version checked; older clients may require the global path. For custom profiles/HERMES_HOME or custom discovery paths, use `--dir` explicitly. Generic runtimes need Agent Skills support or an adapter that reads SKILL.md. Installing files is tested; native execution in all these products is not certified. Restart/reload the runtime if required by its version.

Connect WOS MCP separately: Streamable HTTP `/mcp`, or local `wos mcp stdio` with MCP enabled. Provision an identity with authorized Namespace grants; do not place tokens in prompts, skill files or committed configuration. An installed skill does not grant authority. Consult the WOS agent guide and contracts before sending mutations.

Source references checked on 2026-10-07: [Claude skills](https://github.com/anthropics/skills), [Codex skills](https://developers.openai.com/codex/skills), [Hermes skills](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/skills.md), [OpenClaw skills](https://github.com/openclaw/openclaw/blob/main/docs/tools/skills.md). Preserve these distinctions instead of inventing a universal launch API.

The bundle now includes wos-workspace for API-only wosctl/YAML execution and recovery. Contracts have only expired/revoked/completed terminal causes; no holder release exists after Namespace cutover. Four agent discovery destinations and generic installation remain supported; actual runtime execution certification is separate from installation.
