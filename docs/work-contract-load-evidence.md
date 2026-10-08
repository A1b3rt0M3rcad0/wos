# Contract load evidence

Measured source: `6179387b1f95cc0ef3d4e060054a4a5e97e909f4`; go version go1.27.1 linux/amd64. Actual local SQLite and PostgreSQL 18.6, Linux amd64. Full [raw report](work-contract-load-evidence.json).

Local storage/application fixture; one shared Outcome; two renewal/query rounds per consumer; checkpoint density on one active sample task. Not end-to-end network throughput, production SLA, native Windows or Woobe certification.

| Storage | Consumers | Focal p95 ms | Renewal attempt p95 ms | Maximum response bytes | Conflicts/retries |
|---|---:|---:|---:|---:|---:|
| sqlite | 2 | 6.681 | 5.259 | 3685 | 0/0 |
| sqlite | 10 | 75.724 | 102.065 | 3685 | 0/0 |
| sqlite | 50 | 309.236 | 357.091 | 3685 | 0/0 |
| postgres | 2 | 25.410 | 13.440 | 3685 | 2/2 |
| postgres | 10 | 33.195 | 38.242 | 3685 | 60/60 |
| postgres | 50 | 147.977 | 137.625 | 3685 | 1034/1034 |

All 54 combinations completed. Renewal attempt latency is observer timing, while the focal query measurement includes any read retries. Cumulative pool wait is summed across consumers and can exceed wall time. Percentiles use the recorded small sample; this is not capacity certification. No test retries altered CAS versions, execution generations or idempotency keys.

PostgreSQL serializable transactions in one shared Outcome create substantial contention. Keep the guard for correctness; distribute independent execution across Outcomes, use bounded same-intent retries and measure a realistic deployment before optimizing lock granularity. The selected-context response does not grow with unrelated WorkItems in this fixture. SQLite native trace counts statements; PostgreSQL statement counts were unavailable and are explicitly omitted.

Checkpoint density was sampled on one active task, not every task. No Woobe consumer, native Windows, million-checkpoint or distributed production run was available.
