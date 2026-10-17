# Recorded signed SQL qualification

Candidate `aaa447520f9236bd0a6b929fa1aeb7932ec0e246` completed all 18 declared cases. Baseline `cba58d6d609377df141d1369dccaaaf3b3e331e3` completed 12 cases; its report is incomplete. These are observed development-host results, not a capacity limit or production SLA.

The sources use exactly the same reviewed fixture and runner hashes. The baseline retains its original production implementation. Failed cases are retained and are never replaced with retries that change the original intent or suppress storage/authorization errors. The declaration, exact source manifests, individual successful metrics and failed JSON test output are included beside this document.

Each repetition has four executing/reviewing consumers and eight accepted journeys; history is zero. Service replicas share one Go process and `GOMAXPROCS=3`, with separate 16-connection SQL pools. PostgreSQL 18.6 has two assigned CPUs, 1 GiB RAM, 512 MiB shared memory and the fixed settings recorded in the declaration. Root/agent builds and tests were quiet during both source runs; this shared host was not exclusively reserved. Each source began with a newly created disposable database. The fixed baseline-then-candidate order is not randomized or a controlled cold-cache experiment, so latency differences should not be treated as isolated causal estimates.

## Matching complete groups

Values are medians of three independent repetitions. Read conflict fractions count all attempts; successful-read quantiles exclude conflicts. An unavailable baseline is shown explicitly.

| Topology | Service replicas | Journey p95 ms, baseline → candidate | Accepted/s, baseline → candidate | Read conflict %, baseline → candidate |
| --- | ---: | ---: | ---: | ---: |
| same-outcome | 1 | 2067.91 → 1616.58 | 2.26 → 3.41 | 28.44 → 0.00 |
| same-outcome | 2 | 1856.62 → 1378.00 | 2.89 → 3.96 | 29.84 → 0.00 |
| separate-outcomes | 1 | 2040.10 → 1803.13 | 2.64 → 3.15 | 4.21 → 0.00 |
| separate-outcomes | 2 | 2143.60 → 1737.59 | 2.53 → 2.98 | 4.11 → 0.00 |
| separate-namespaces | 1 | unavailable → 1747.81 | unavailable → 3.26 | unavailable → 0.00 |
| separate-namespaces | 2 | unavailable → 1759.14 | unavailable → 3.19 | unavailable → 0.00 |

## Every completed repetition

The following table includes all completed repetitions, without selecting the fastest runs. Mutation conflicts are unchanged-intent transaction retries. Read conflicts are reported separately from successful-read p95.

| Source | Topology | Replicas | Repetition | Journey p95 ms | Accepted/s | Read p95 ms | Successful reads | Read conflicts | Mutation conflicts |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| baseline | same-outcome | 1 | 1 | 1618.48 | 3.372 | 9.29 | 227 | 128 | 35 |
| baseline | same-outcome | 1 | 2 | 2067.91 | 2.263 | 18.07 | 347 | 128 | 38 |
| baseline | same-outcome | 1 | 3 | 3098.90 | 2.046 | 15.47 | 392 | 128 | 42 |
| baseline | same-outcome | 2 | 1 | 1888.63 | 3.200 | 9.89 | 252 | 128 | 37 |
| baseline | same-outcome | 2 | 2 | 1368.12 | 2.718 | 11.94 | 320 | 128 | 50 |
| baseline | same-outcome | 2 | 3 | 1856.62 | 2.885 | 9.89 | 331 | 128 | 56 |
| baseline | separate-outcomes | 1 | 1 | 2040.10 | 2.637 | 9.10 | 713 | 32 | 43 |
| baseline | separate-outcomes | 1 | 2 | 2090.83 | 2.632 | 6.61 | 755 | 32 | 47 |
| baseline | separate-outcomes | 1 | 3 | 1655.05 | 2.792 | 6.35 | 714 | 32 | 39 |
| baseline | separate-outcomes | 2 | 1 | 2270.27 | 2.423 | 8.57 | 794 | 32 | 51 |
| baseline | separate-outcomes | 2 | 2 | 1878.35 | 2.790 | 7.65 | 690 | 32 | 35 |
| baseline | separate-outcomes | 2 | 3 | 2143.60 | 2.531 | 7.80 | 756 | 32 | 35 |
| candidate | same-outcome | 1 | 1 | 3213.26 | 2.021 | 14.56 | 974 | 0 | 49 |
| candidate | same-outcome | 1 | 2 | 1616.58 | 3.409 | 6.23 | 660 | 0 | 40 |
| candidate | same-outcome | 1 | 3 | 1349.06 | 4.017 | 5.26 | 608 | 0 | 31 |
| candidate | same-outcome | 2 | 1 | 2329.76 | 2.700 | 11.07 | 773 | 0 | 38 |
| candidate | same-outcome | 2 | 2 | 1378.00 | 4.021 | 4.53 | 607 | 0 | 36 |
| candidate | same-outcome | 2 | 3 | 1320.51 | 3.956 | 4.64 | 616 | 0 | 31 |
| candidate | separate-outcomes | 1 | 1 | 1803.13 | 3.248 | 5.77 | 726 | 0 | 45 |
| candidate | separate-outcomes | 1 | 2 | 1936.61 | 3.153 | 5.97 | 747 | 0 | 48 |
| candidate | separate-outcomes | 1 | 3 | 1744.87 | 3.062 | 5.66 | 769 | 0 | 49 |
| candidate | separate-outcomes | 2 | 1 | 1530.59 | 3.233 | 5.85 | 723 | 0 | 53 |
| candidate | separate-outcomes | 2 | 2 | 1737.59 | 2.983 | 6.75 | 765 | 0 | 41 |
| candidate | separate-outcomes | 2 | 3 | 1887.51 | 2.730 | 6.18 | 844 | 0 | 57 |
| candidate | separate-namespaces | 1 | 1 | 2212.19 | 2.488 | 9.17 | 799 | 0 | 56 |
| candidate | separate-namespaces | 1 | 2 | 1690.86 | 3.400 | 6.51 | 691 | 0 | 41 |
| candidate | separate-namespaces | 1 | 3 | 1747.81 | 3.263 | 6.54 | 722 | 0 | 42 |
| candidate | separate-namespaces | 2 | 1 | 1979.85 | 3.194 | 5.75 | 741 | 0 | 52 |
| candidate | separate-namespaces | 2 | 2 | 1759.14 | 2.883 | 8.71 | 780 | 0 | 49 |
| candidate | separate-namespaces | 2 | 3 | 1666.13 | 3.601 | 6.37 | 656 | 0 | 37 |

## Failure and scope

The baseline stops at its first four-Namespace, one-replica case. Its signed `return_review` operations report `forbidden: credential or namespace permission inactive`; the [exact failed test output](baseline/separate-namespaces-replicas-1-history-0-run-1.jsonl) is retained. No throughput or successful-read tail estimate is assigned to that failed case. A completed candidate matrix does not make the full before/after matrix complete. Ratios in the machine record exist only for matching groups with three completed repetitions on both sources.

Eight journeys per repetition provide an empirical sample, not a reliable production p95/p99 estimate. Zero-history results do not qualify sustained storage growth, large retained histories, external delivery throughput, actual HTTP replicas, autoscaling, failover/PITR, online migration, mixed-version rolling upgrades or production high availability. See the [qualification guide](../../qualification.md) and [actual process/profile smoke](../README.md).

[Earlier attempts](earlier-attempts.json) are retained privately for this development session and summarized with artifact hashes; their local paths are not a durable archive. They are not substituted for these source-bound results. Raw published final test output contains only fixture identifiers and metrics; credentials, private keys, authentication cookies and database connection strings are excluded. [Artifact hashes](artifact-sha256.json) cover the published files.
