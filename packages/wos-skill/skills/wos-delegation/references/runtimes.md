# External runtime delegation

WOS does not spawn or execute agents. Use only native delegation exposed by the actual host and authorized by the user/repository. Runtime permissions, model choice, history inheritance, filesystem separation and tool execution belong to that host.

- **Codex:** use its available native delegation mechanism. Where supported, choose fresh/minimal history explicitly and supply necessary intent, permitted paths/tools and stop conditions. A shared workspace is not an isolated filesystem. Skills are discovered in `.agents/skills` or the configured user root.
- **Claude Code:** use the native Agent/subagent mechanism supported by the installed client. Configure permissions and models in the runtime. Skills can be installed into its `.claude/skills` discovery root.
- **Hermes:** delegate only if the actual runtime/tool catalog enables it; a model name alone is not an agent host. Discovery may use `.hermes/skills` or the configured user/profile root.
- **OpenClaw:** use the installed version's exposed sessions/subagent tools under its policy. Workspace/user skill discovery and profile overrides do not prove history or file isolation.
- **Other hosts:** use `wos skill install --agent generic --dir PATH` for an explicit Skill discovery directory. Execute sequentially if no authorized delegation mechanism exists.

Skill discovery roots are unrelated to installation-scoped WOS operational data. Install/update/doctor verify instruction files; they do not provision server access or certify host integration.

Auxiliary helpers share one holder's responsibility; they receive bounded instructions and return checked results. Independent workers acquire different WorkItems under their own pinned approved Profiles/Principals and record progress/proof through the CLI. The holder never silently transfers its contract to another authenticated identity. An independent reviewer must be a distinct Principal under server policy.

Use `--profile NAME` or process-scoped `WOS_PROFILE` in every worker; do not change the global active Profile as concurrent coordination. Return compact IDs/proof/remaining action rather than full transcripts. Measure total usage before claiming token savings.
