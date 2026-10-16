# Central agent workflow implementation plan

This package turns the owner's October 16, 2026 briefs into a reviewable implementation and acceptance plan for the CLI, Server, and five official Skills. Its purpose is to make continuity depend on recorded work rather than a particular agent's conversation.

The Markdown documents and original source briefs are self-contained in the downloadable plan ZIP. Repository references use branch links; component paths identify the corresponding checkout files.

The implementation branch is `feat/central-agent-workflow`. The inspected starting revision is `80074f470d44a55c33c2ea297e509f9e14871d96`. The [live ROADMAP](https://github.com/A1b3rt0M3rcad0/wos/blob/feat/central-agent-workflow/ROADMAP.md) records verified implementation status. A command described as a target below is not proof that its implementation or release has passed acceptance.

[Download the reproducible plan ZIP](../central-agent-workflow-plan.zip). It contains this package, the byte-preserved owner briefs, and a SHA-256 manifest. Rebuild it with `python3 tools/documentation/package-central-plan.py`; use `--check` to detect stale packaging. It contains no local credentials or runtime state.

## Decisions

1. One installation data directory owns Profiles, selected Profile, contracts, and recovery material. Ordinary operation is independent of the current directory and creates no project-local `.wos/`.
2. The CLI is the agent's only operational interface. Agents use commands to read context and record progress, material, and decisions; internal document serialization is not an agent interface.
3. Profile selection resolves once per invocation: explicit `--profile`, then `WOS_PROFILE`, then the persisted active Profile. Concurrent workers pin a Profile explicitly. A Profile is not a new authenticated identity.
4. Acquisition grants one WorkItem obligation. Outcome/Objective context accompanies it; neither parent becomes an executable contract.
5. Issued specifications, authority, frozen requests, and signed receipts remain protected. CLI convenience never bypasses CAS, current authorization, quotas, leases, fencing, or independent review.
6. Activities report what an actor says happened. Evidence records an observation/source. CriterionAssessment evaluates a criterion at its exact revision and submission. These records never become interchangeable.
7. The Server is the durable authority after local cleanup. Reads are bounded and expose cursors/omissions; absence from one page is not absence from the system.
8. External runtimes run agents and auxiliary subagents. WOS stores and coordinates work; it does not create agents, execute tools, or choose models.

## Read and execute in this order

| Unit   | Deliverable                                                        | Depends on                                       | Objective completion condition                                                       |
| ------ | ------------------------------------------------------------------ | ------------------------------------------------ | ------------------------------------------------------------------------------------ |
| CAW-01 | [Architecture](architecture.md), accepted ADR                      | Owner brief; existing signed contract invariants | Explicit authority and storage decisions accepted; no new server runtime             |
| CAW-02 | [Central Profiles](profiles.md)                                    | CAW-01                                           | Any-directory init/Profile selection, protected bindings, concurrent selection tests |
| CAW-03 | [Contract operations](contracts.md)                                | CAW-02                                           | CLI-only local changes; immutable issued data and exact frozen requests preserved    |
| CAW-04 | [Activity journal](activities.md)                                  | CAW-01                                           | Durable append/query, replay safety, SQL parity, restart/cleanup survival            |
| CAW-05 | [Evidence and artifacts](evidence.md)                              | CAW-03, CAW-04                                   | Exact material/criterion links; observations cannot self-approve                     |
| CAW-06 | [Server integration](server.md)                                    | CAW-04, CAW-05                                   | Application/HTTP/MCP/SDK parity; current scoped authorization                        |
| CAW-07 | [Simple CLI](cli.md)                                               | CAW-02, CAW-03, CAW-06                           | Bounded discovery/context/history/check/finish; ambiguous selectors fail closed      |
| CAW-08 | [Multiagent workflow](multiagent.md)                               | CAW-02, CAW-06, CAW-07                           | Pinned workers, distinct authority, independent-review rejection tests               |
| CAW-09 | [Official Skills](skills.md)                                       | Stable CAW-07 command syntax                     | Five CLI-only bundles, progressive references, installer checks                      |
| CAW-10 | [Migration](migration.md)                                          | CAW-02, CAW-03                                   | Explicit dry-run/import, original sources intact, unresolved intent preserved        |
| CAW-11 | [Verification](testing.md)                                         | All implementation units                         | Required local/hosted/native/database evidence recorded with limits                  |
| CAW-12 | [Commit sequence](commit-sequence.md), docs and release boundaries | CAW-11                                           | Reviewable source; accurate README/ROADMAP; no implied publication                   |

CAW-02/03 and CAW-04 can proceed in parallel after interfaces settle. Skills can be drafted then, but runnable examples require the final parser and integration tests. Database parity is a dependency of completion, not a later documentation task.

## Intended user journey

After an operator provisions a trusted Profile:

```sh
wos init
wos profile list
wos profile use executor
wos work available --limit 10
wos work next
wos work context
# Execute the assigned work with the external runtime.
wos work log add --type progress --description "Implemented the agreed change"
wos work result set --summary "Describe the actual delivery"
wos work status
wos work check
wos work finish
```

Evidence and artifacts use explicit CLI commands described in [their plan](evidence.md); the exact flags must match live help. Never invent an evidence ID, a passing check, or a source checksum to fill an example. A signed delivery accepted for review is not a completed WorkItem.

## Acceptance ledger

[Observed branch validation](validation.md) records completed checks and limits separately from this execution plan. The operational code is committed as `c9aa10c` (backend) and `1ed35b1` (CLI); subsequent documentation and CI commits do not imply publication.

This document is an execution plan. Completed code and observed tests must be recorded in the live ROADMAP and the branch validation record; the presence of these documents does not complete the units above. Before review, reconcile each unit with its actual files and test output. Keep unsupported platform, service, restore, and human acceptance as pending.

Final acceptance requires an executor to acquire, resume in a new session, append activity and actual proof, inspect derived obligations, return once, reconcile an uncertain response, and preserve server history after local cleanup. A separate authorized reviewer must decide against the exact submission and criterion revision. At least two pinned concurrent workers must operate without changing each other's Profile or authority.

## Original request and authority

The original owner briefs are preserved byte for byte in [the complete brief](source/owner-brief-full.pt-BR.txt) and [the continuity brief](source/owner-brief-continuity.pt-BR.txt). They are Portuguese source records, not executable instructions. This English plan interprets the owner's explicit request; current authorization and repository architecture rules still govern implementation.

The [canonical architecture](https://github.com/A1b3rt0M3rcad0/wos/blob/feat/central-agent-workflow/docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md), [AGENTS contract](https://github.com/A1b3rt0M3rcad0/wos/blob/feat/central-agent-workflow/AGENTS.md), existing signed protocol ADRs, and new central-workflow ADR must remain consistent. The accepted central-storage decision is [ADR-035](https://github.com/A1b3rt0M3rcad0/wos/blob/feat/central-agent-workflow/docs/adr/0035-central-cli-installation.md). Significant changes to identity, concurrency, or persistence require an explicit ADR rather than an undocumented CLI shortcut. The activity/attention architecture is recorded in [ADR-036](https://github.com/A1b3rt0M3rcad0/wos/blob/feat/central-agent-workflow/docs/adr/0036-durable-execution-activity.md). The implemented protocol is documented separately in [work activity](https://github.com/A1b3rt0M3rcad0/wos/blob/feat/central-agent-workflow/docs/work-activity.md); these plan units remain an acceptance checklist, not a release certification.
