# Operate with agents

People, agents, and applications share the state persisted by the service. Agents execute in their own environment; the server records Tasks, authority, deliveries, and assessments. Start with the [illustrated workflow](../README.md#how-it-works) before configuring an integration.

## 1. Install agent instructions

With the `wos` executable available:

```sh
wos skill install --agent codex
wos skill list --agent codex
wos skill doctor --agent codex
```

Use `claude`, `hermes`, or `openclaw` for another discovery layout. `--agent all` installs into all four; `--agent generic --dir /absolute/path/to/skills` selects an explicit directory. `--project /path/to/project`, `--global`, and `--dry-run` control the destination and preview.

The binary includes five skills: coordination, delegation, continuity, review, and workspace operation. Installation is offline and preserves user-modified files. Configure agent execution, MCP, credentials, keys, and protocol activation separately. An agent runtime may need to reload its discovery directories.

## 2. Connect an authorized identity

The server must be running. Each executor or reviewer needs an identity with access to the Namespace and the required operations. A Namespace is the authorization boundary shown as a Workspace in the browser. Follow [shared-service operations](operations.md#shared-service) to configure authentication and permissions.

For signed contracts, an administrator prepares the persistent server identity, Namespace protocol, permissions, and agent-key enrollment. Configure the project and profiles using [signed workspace profiles](signed-workspace-profiles.md). Check the server identity and issuer fingerprint through a trusted channel, and use protected secret references.

An executor and independent reviewer need distinct eligible Principals. Changing only a profile name or key does not establish independence. Skill installation and profile creation do not change the Namespace protocol.

## 3. Acquire and execute work

With an authorized `executor` profile configured:

```sh
wos --profile executor auth status --output json
wos --profile executor doctor --output json
wos --profile executor work checkout --next --count 1 --output json
wos --profile executor work list --output json
wos --profile executor work show CONTRACT_UUID --for-agent --output json
```

Replace `CONTRACT_UUID` with the identifier returned by acquisition. Contract context contains instructions, constraints, criteria, and references. An empty bounded page does not establish that the entire Outcome has no eligible work; respect the scope and cursors in the response.

Execute externally and edit only the permitted execution draft in `.wos/profiles/executor/contract/CONTRACT_UUID.yaml`. Preserve the issued specification and signed intent. Observe the authority's validity; a local copy does not renew an expired acquisition.

```sh
wos --profile executor work finish CONTRACT_UUID --output json
```

The delivery may remain pending review. Successful submission or a passing test does not approve the Task, its criteria, or its Outcome.

## 4. Review a delivery

A reviewer uses their own authorized profile to acquire a review contract:

```sh
wos --profile reviewer review checkout --next --count 1 --output json
wos --profile reviewer review show REVIEW_CONTRACT_UUID --for-agent --output json

# Inspect the submission and complete the permitted review draft.
wos --profile reviewer review finish REVIEW_CONTRACT_UUID --output json
```

Review targets the exact submission and bound criterion revisions. `changes_requested` records a correction request; correction requires a new explicit acquisition and preserves the historical delivery. See [signed returns and review](signed-cli-returns.md).

## 5. Recover an interrupted session or response

```sh
wos --profile executor work recover --all-pending --output json
wos --profile reviewer review recover --all-pending --output json
```

Recovery reconciles original intents and receipts. Keep the original key, destination, payload, and expected version. Preserve drafts and unconfirmed local material. If authority expired or was revoked, follow the current server result and documented recovery; do not substitute another identity to bypass its binding.

When resuming an Outcome, reread current state and outstanding obligations. Task completion does not automatically conclude an Objective or Outcome; evidence and assessments remain distinct records.

## Other integration paths

- [MCP profiles](contracts.md): Streamable HTTP or stdio connections, with execution remaining in the agent's host.
- [HTTP](http.md) and the [Go client](../examples/remote-client): explicit commands, queries, and recovery.
- [Client qualification](client-qualification.md): limits, pagination, idempotency, and receipts.
- [Legacy protocols](../packages/wos-cli/README.md): schema-1 and `contracts_v1` recovery; these do not replace a signed Namespace's authority.
- [Human workspace](workspace-ux.md) and [current screenshots](design/editorial/README.md#main-pages): browser access to the same state.

For development and integration validation, consult [AGENTS.md](../AGENTS.md), the [technical index](README.md#develop-and-understand-the-project), and implementation status in [ROADMAP.md](../ROADMAP.md).
