# Simple CLI and compact read contracts

## Commands to expose

| User intention           | Target command                                   | Required behavior                                                     |
| ------------------------ | ------------------------------------------------ | --------------------------------------------------------------------- |
| Initialize once          | `wos init`                                       | Installation scope; idempotent; no project data                       |
| Select identity          | `wos profile list`, `wos profile use NAME`       | One persisted default; explicit process pin overrides                 |
| Find Outcome             | `wos outcome list --limit 10`                    | Scoped bounded result with continuation                               |
| Find eligible work       | `wos work available --limit 10`                  | Candidate discovery grants no authority                               |
| Acquire next             | `wos work next`                                  | One explicit WorkItem acquisition; preserve incomplete-search reasons |
| Read assigned obligation | `wos work context`                               | Minimal immutable specification plus relevant live state              |
| Inspect contract         | `wos work show`                                  | Auto-select only one matching contract                                |
| Report activity          | `wos work log add`                               | Typed append; no rewriting past records                               |
| Attach proof/material    | `wos work evidence add`, `wos work artifact add` | Validated exact material; no implied approval                         |
| Inspect obligations      | `wos work status`, `wos work check`              | Derive known blockers, criteria, authority, delivery gates            |
| Deliver once             | `wos work finish`                                | Existing signed freeze/send/receipt/recovery guards                   |
| Resume context           | `wos work history --limit 10`                    | Durable bounded history, available after local cleanup                |

These are the agreed public command families. Examples using material fields must match implemented help before being promoted to the README or Skills. No command should require an agent to open internal documents.

## Existing files and integration

`packages/wos-cli/client.go` owns parsing, dispatch and output/errors. `work_commands_v2.go` owns signed checkout/list/show/recover, with operational discovery, local material, activity recovery and bounded continuation in `operations_resolution_v2.go`, `operations_material_v2.go`, `operations_activity_v2.go`, `operations_query_v2.go` and `operations_draft_query_v2.go`. `lease_commands_v2.go` owns renew/resume/refresh. `return_preflight_v2.go`, `return_send_v2.go` and `return_resolution_v2.go` protect frozen returns and receipt-driven cleanup. `packages/wos-sdk-go/signed_queries.go` and generated command wrappers are the remote interface; do not hand-code domain behavior in the CLI.

## Reference resolution

1. Preserve exact UUID support for all existing calls and recovery after cleanup.
2. Permit exact unique titles and explicit short ID prefixes where implemented. Resolve within the selected Profile's authorized scope; never interpret a short reference as authority.
3. Auto-select a missing contract argument only when exactly one applicable contract exists. Filter execution/review kind before selecting.
4. Return an ambiguity diagnostic with bounded candidates. Never pick the first title match or silently switch Outcomes.
5. A search stopped by its bound must report incompleteness and ask for an exact ID. It cannot prove uniqueness from one page.
6. Resolve exact ID, observed version and destination before freezing any mutating intention. A retry cannot re-resolve the same title to a different entity.

## Read and output budget

Use `--for-agent --output json` for model-facing context and `--limit`/`--cursor` for paged discovery/history. Machine output stays one valid JSON value on stdout; diagnostics go to stderr. Include relevant authority/expiry, selected identifiers, revisions, omissions/truncation and next cursor. Never include tokens, private keys, full signatures or unrelated contract payloads by default.

Context answers “what is my obligation and what currently affects it?” Status answers “what is known to need attention?” Check answers “can this particular draft be prepared/returned under current rules?” Check is not a guarantee of later server acceptance. Avoid inventing a manual `pending[]` business list; technical frozen intentions remain internal recovery state.

The implemented context combines the complete immutable issued obligation, bounded live focal scope, staged material summary, and a newest-first WorkItem activity page of five records (up to 25 with `--limit`). Focal descriptions, dependencies, blockers and decisions declare omissions; these do not truncate issued instructions or criteria. Focal state, attention and activity are separate authoritative reads with independent revisions/evaluation times, not one claimed coherent transaction. A descending cursor is followed through `work history --work ID --outcome REF --order desc --cursor CURSOR`. Ascending history remains the default; the Server indexes both directions and fixes the traversal's upper revision.

Activity descriptions, exact version metadata and inner reference collections are compacted explicitly in context; complete chronological records remain on the separate history query. Historical exact-contract context also restores bounded accepted material after local cleanup by expanding the existing submission and acceptance-fact resources. Verify pinned issuer proof and exact scope/contract/WorkItem/specification/submission digests before showing `accepted_result`. This read never manufactures authority or equates the current Task with its accepted historical delivery.

## Before and after

Before: initialize each repository, locate internal contract files, manually change drafts, carry UUID/version details between commands.

After: initialize installation data once, pin the approved Profile, acquire one WorkItem, read focal context, use validated record commands, check and finish. Existing low-level signed/recovery commands remain compatibility tools rather than the primary tutorial.

## Typed preparation commands

```sh
wos work progress set --summary TEXT --next-action TEXT
wos work result set --summary TEXT --completion auto
wos review decision set --decision approved --reason TEXT
wos review assessment add --criterion ORIGINAL_REF --result met --rationale TEXT
wos review check
wos review finish
```

Pin the approved Profile on each command. Result summary is required for delivery; progress alone is advisory. Set the review decision to approval before adding its assessments. For non-approval, first set `review decision set --decision changes_requested|inconclusive --reason TEXT`, then use `review finding add --criterion REF --description TEXT` (or the mutually exclusive `--requirement REF`) with no criterion assessments. Finding validation requires the review reason to be present. Corrections use `work correction add --finding REF --summary TEXT` after explicit original-review acquisition.

Execution history is durable activity history; `review history` must not be described as complete review-decision history unless that separate query exists. Review context/show retains its exact submitted-source/assigned-case binding.

Draft correction is also CLI-only. `assessment add|set|remove|list` targets an original criterion; `finding add|set|remove|list` targets a reviewer finding (`--finding REF` for set/remove), and `correction add|set|remove|list` targets an issued correction finding. Duplicate `add` fails; `set` explicitly upserts an assessment/response and preserves an existing finding's identity. A finding's original obligation is retained unless an explicit replacement criterion/requirement is selected. Changing a decision does not silently clear assessments. Reviewer staged evidence/artifact can be explicitly removed before signing (`--evidence local:KEY` / `--name KEY`); referenced evidence must first be removed from its assessment. No setter edits accepted server history or frozen intentions.

Local list output is bounded to 64 KiB. Its cursor binds the issued contract, selected Profile, section and draft digest. A changed draft or different section rejects the cursor instead of silently continuing through different material. Context exposes a compact staged view and expansion commands; agents do not read the underlying serialization. Large review contracts expand authenticated original-specification references into a derived view and preserve the issued document exactly.

## Acceptance

Test exact/short/title/omitted selectors, duplicate titles, incomplete pages, zero and multiple contracts, two contract kinds, and post-cleanup server history. Reuse signed failure/recovery tests so command aliases do not weaken return or lease semantics. Assert omitted fields stay omitted at large input sizes; do not infer token savings from byte counts.
