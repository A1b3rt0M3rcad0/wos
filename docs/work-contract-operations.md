# Contract operations

WOS coordinates durable state and work authority. Agents and humans choose work;
their host executes it. WOS never launches subagents, interprets YAML as a program,
runs an LLM or locks a Git branch/filesystem.

A WorkContract freezes the minimal executable specification and owns one renewable
lease on one WorkItem. Only completion, expiry or explicit administrative revocation
ends it. Losing authority preserves the WorkItem ID and recoverable in_progress state.
Lease CAS is separate from progress CAS; fencing is an exact unsigned decimal string.

For a new agent session, pass authorized Outcome/WorkItem/contract IDs plus essential
intent. Read the frozen spec once, then latest checkpoint, live authority/impediments
and only necessary proof references. Expand bounded history when required; observe
omissions and cursors. This reduces repeated context transfer but does not itself
measure or guarantee lower total token use. Native context isolation and shared-file
coordination belong to Codex, Claude, Hermes, OpenClaw or the other host.

Acquire explicitly → execute externally → checkpoint → submit immutable material →
review exact submission → finalize. Evidence is observation; assessment is acceptance.
A revised submission requires new proof binding. Work completion never automatically
achieves an Objective/Outcome. Pending review may retain explicitly renewed authority;
there is no voluntary holder release. Same-holder takeover rotates execution/fencing
without extending TTL; expiry requires a new eligible acquisition.

`wosctl` offers this flow through restricted YAML and a destination-bound write-ahead
journal. Credentials remain environment references. Refresh/recovery preserve edited
local drafts. Lost replies and post-commit disk failure reconcile the original receipt
or exact payload/key before another intent. A missing retained receipt does not prove
that a transaction failed. Registered local keys identify immutable material and cannot
silently be reused after edits. Refer to [the client guide](../packages/wos-cli/README.md)
and generated schemas for exact examples and exit codes.

The human interface shows contract authority, material progress, submission and review
in WorkItem details. Contract history loads on demand. Revocation requires an explicit
contract/reason and the server's administrative permission. Review uses submission-bound
assessments and the same independent reviewer policy as HTTP/MCP/CLI. A UI button does
not grant authorization.

Benefits: durable handoff without transcript transfer, bounded task authority, exact
replay after uncertain commits, no accidental progress-as-completion, portable instructions.
Tradeoffs: lease renewal/review latency, explicit recovery and conflict handling, host-owned
filesystem and external-effect coordination, deployment-controlled protocol migration.
Improve by measuring contract/context sizes and consumer costs, provisioning distinct
Principals for independent workers, and supervising foreground renewal at the host.

Linux race/protocol/recovery tests are available. Actual Windows crash/lock/reparse and
consumer execution gates must pass on their target environments; compilation and npm
skill installation are not substitutes. Registry publication is reported only after
published artifact identities/checksums are verified.
