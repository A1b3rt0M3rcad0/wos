# Controlled context and cost measurement

Use the same task set, original quality requirements and final integration check
for three strategies: A (one conventional coordinator), B (the selected planner,
executor and reviewer models with external coordination) and C (the same B models
and harness using WOS). B/C must have the same model/harness configuration digest.
Retain failures, retries, corrections, reviews and integration attempts in the
input. An internal Task DONE is insufficient evidence of equivalent quality.

Run `python3 tools/acceptance/context-cost-report.py measurements.json report.json`.
The input is schema 1, with a full `source_commit`, `tasks` (id, outcome_id and
SHA-256 `quality_contract_digest`) and `strategies` keyed by A, B or C. Each
strategy supplies `model_harness_configuration_digest`, `attempts`,
`accepted_tasks`, `accepted_outcomes`, `quality_evidence_ref` and
`integration_evidence_ref`. Each attempt supplies a unique `id`, `task_id` and
`phase` (planning, execution, review, correction, coordination or integration).
Optional usage counts are input_tokens, output_tokens, cached_tokens, tool_calls
and human_interventions. Unknown values are omitted or null.

Costs use decimal strings and one currency, with `cost_evidence` classified as
provider_invoice, provider_usage, documented_rate_estimate or self_reported and
a retained `cost_evidence_ref`. A documented estimate must retain its tariff,
model, cache treatment and date. Self-reported values remain self-reported.
Do not derive token counts from payload bytes or arbitrary token prices from a
subscription fee. Missing usage in any attempt keeps its aggregate unknown.

The report includes total cost and cost per accepted task/Outcome, including
unsuccessful attempts. No accepted task means no cost denominator. Missing A/B/C
controls prevent comparative claims. Retained external evidence must still be
audited; this validator cannot authenticate invoices or quality assessments.

The retained [controlled Codex pilot](audit/controlled-codex-2026-10-09/observations.json)
executed A (single agent/self review), B (fresh executor/reviewer with external
handoff) and C (fresh executor/reviewer with WOS) on one small coding task. All
three passed the same external validator: actual Node tests plus 43 assertions.
Input, quality receipts and the generated report are retained in that directory.

Optional `observed_timing` supplies timezone-bearing `started_at`, `accepted_at`
and `evidence_ref`; elapsed time includes parent/harness orchestration. It is
reported separately from provider usage and billing. Missing timestamps leave
elapsed time unknown. Reversed/naive timestamps and timing without acceptance
are rejected.

This is a functional calibration with fixed A→B→C order, one task per strategy
and no statistical significance. C includes a rejected malformed-evidence
attempt, a new owned fixture and a broker fix/recovery of the exact original
signed return. Its source changed during diagnosis; these wall times are not a
steady-state performance comparison. All failures and recovery steps remain in
the measurement input. Provider model identifiers, tokens/cache/tool counts and
billing are unavailable; cost/token savings remain unknown. No Outcome was
explicitly achieved, so Outcome cost denominators remain empty. Parser fixtures
are synthetic unit data and are not benchmark results.
