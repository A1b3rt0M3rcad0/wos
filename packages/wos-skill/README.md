# WOS Skills

The five official Skills teach one operational interface for agents: the native
CLI. They use installation-scoped Profiles, bounded WorkItem context, durable
activity/proof records, independent review, and original-intention recovery.
They do not ask agents to inspect or modify internal operational documents.

| Skill              | Purpose                                                                        |
| ------------------ | ------------------------------------------------------------------------------ |
| `wos-workspace`    | Initialize installation data, pin a Profile and operate through typed commands |
| `wos-coordination` | Discover, acquire, follow and return one WorkItem obligation                   |
| `wos-delegation`   | Separate auxiliary helpers from independently authorized workers               |
| `wos-continuity`   | Resume using compact current state and bounded durable history                 |
| `wos-review`       | Evaluate exact submitted material from an independent authorized Principal     |

These sources describe the centralized workflow on this branch. The published
package may contain an earlier command set; use a coordinated build and its live
help. Skill installation neither provisions credentials nor grants server access,
configures an agent runtime, or starts agents. WOS Server remains separate from
the client that runs these commands.

## Install directly using the WOS Go binary (recommended)

The native server/operator executable contains all five skills and installs them
without Node.js, npm, network access, a running WOS server or an API credential.

```bash
wos skill install --agent codex
wos skill install --agent claude
wos skill install --agent hermes
wos skill install --agent openclaw
wos skill install --agent all --dry-run
wos skill install --agent all
wos skill doctor --agent codex
wos skill update --agent codex
wos skill uninstall --agent codex
```

The native installer reads the canonical skill sources embedded during the Go
build, uses the existing `.wos-skill.json` SHA-256 ownership format, preserves
local modifications and refuses symlink/unmanaged destinations. The optional
Node installer source remains only as a compatibility fixture; Node.js is **not** required
when using `wos skill`.

## Public installation

```sh
npm install -g wosctl@latest
wos skill install --agent codex
```

`wosctl` is the single public npm package. This directory contains the canonical embedded skill source and historical Node installer compatibility fixtures; it is not published as a separate npm package. Native Go owns installation, updates, doctor and removal.

## Run a pinned worker

After an operator provisions an approved Profile:

```sh
wos init
wos profile list
wos --profile executor work next
wos --profile executor work context --for-agent --output json
```

Continue using the same explicit Profile, or a process-scoped `WOS_PROFILE`.
`profile use` sets an interactive default; it must not switch concurrent workers.
Use an explicit contract reference whenever more than one obligation applies.
The Skills explain typed activity, evidence, artifact, result, review and recovery
commands progressively. Execution stays in the external runtime; activities are
declarations, evidence is observation, and assessment is explicit evaluation.
