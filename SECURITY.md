# Security

## Scope and compatibility

Use verified artifacts from [GitHub Releases](https://github.com/A1b3rt0M3rcad0/wos/releases) or the documented public distribution. This branch may contain changes that are not released. The validated platform scope is Linux amd64 for the server and Linux/Windows amd64 for the native client. See the [compatibility contract](docs/stable/compatibility.md) and [operational acceptance limits](docs/stable/operational-acceptance.md) before deploying or upgrading.

WOS checks authenticated scope, permissions, expected versions, leases, fencing, and configured review policies. It records delivery and assessment facts; it does not execute agents, sandbox their tools, or establish that submitted claims are true. Independent review requires eligible distinct Principals and the corresponding policy or signed acceptance mode. The default local demonstration does not enable every independent-review policy.

The server, database administrators, agent host, and operating system remain trusted. Protect server signing material and agent credentials/keys, verify issuer identity through a trusted channel, and keep backups and pending client intentions under appropriate access controls. Signed records do not protect against a compromised authorized host acting within its authority.

## Deploy and recover safely

> [!WARNING]
> The public bootstrap token in `.env.example` is a localhost demonstration credential. Do not expose that configuration to a shared network. Shared deployments require unique credentials, protected storage, and HTTPS at the ingress.

Keep tokens, private keys, database connection strings, and documentary payloads out of public issues, logs, URLs, and commits. Bootstrap configuration initializes credentials; changing it is not a password reset for an existing database. Use the authorized credential-management procedures in [service operations](docs/operations.md#shared-service).

A timeout or disconnected client does not prove a write failed. Recover the original intent and receipt with the same destination, authenticated scope, payload, idempotency key, and expected version. Do not replace an uncertain operation with a new key or different identity. Back up before forward migrations; rollback uses a separate restored database and must reconcile later security changes and external effects. See [operations](docs/operations.md#backup-restore-and-upgrade) and [agent recovery](docs/agents.md).

## Local agent material

Central profiles use a private OS data directory or an explicit absolute `WOS_DATA_DIR`, independently of the project directory. Secrets remain protected references; internal drafts, pending requests, receipts and activity descriptions can still contain sensitive work. Protect and back up this directory. CLI locks coordinate cooperating processes; they do not isolate hostile processes under the same OS account.

Profile selection is pinned per invocation, but an interactive active profile is shared by later invocations. Concurrent agents should use `--profile` explicitly. Removing a profile retires its selector and preserves historical bytes; it does not revoke remote credentials. Migration requires all old writers stopped: older binaries cannot recognize the new retirement fence.

Typed activity is an authenticated actor declaration. A referenced URI, checksum, or recorded test claim is not independent proof that external execution occurred. Reviewers must inspect the exact material. See [profile migration](docs/signed-workspace-profiles.md) and [activity limits](docs/work-activity.md).

## Report a vulnerability

GitHub private vulnerability reporting is not currently enabled for this repository, and no dedicated security email is published. Open a general [issue](https://github.com/A1b3rt0M3rcad0/wos/issues/new) asking the maintainer to establish a private reporting channel. Keep the issue limited to that request: do not include exploit steps, affected private data, credentials, keys, or other sensitive details. Share those details only after a private channel is agreed.

In that private report, include the affected version or commit, deployment context, impact, and a minimal reproduction using synthetic data. Redact secrets and avoid testing against systems you do not control. This policy does not promise a response or remediation timeline.
