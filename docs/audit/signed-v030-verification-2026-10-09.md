# Signed 0.3.0 verification evidence

The coordinated release verifier actually succeeded on integrated master
`9ee55cf95a8e2c2dff402d3eecc66014cf810135` in
[run 37899227520](https://github.com/A1b3rt0M3rcad0/wos/actions/runs/37899227520).
Core SQL/race/restore/fuzz/load/container gates, browser, packages and native
Linux/Windows checks succeeded. The final verifier installed the offline npm
packages, exercised protected signed enrollment/return/review/correction/recovery,
matched native receipts against the actual client archive hashes and uploaded
`verified-coordinated-release`. Registry publication was explicitly skipped.
Downloaded artifacts were independently checked against all five manifest hashes.

This verified precursor exposed two additional defects during real managed Codex
calibration; they are fixed in the subsequent source commits retained by PR75/76.
Do not substitute precursor receipts for those later binaries. The same-source
release workflow must be rerun for the source intended for publication.

Native precursor receipts used Go1.27.2: Linux passed62 recorded test events and
Windows passed63. Both actually executed their native protected credential store.
Windows had no PostgreSQL fixture; Linux's separate full-disk opt-in test does not
run in the platform job. Main core jobs separately execute actual PostgreSQL and
Linux ENOSPC acceptance. Listed skips are retained in the raw receipts.

Local combined-source race verification before the two pilot fixes passed17
packages and691 passing events (including subtests), using the owned clean
PostgreSQL database and actual restore container. That count is execution evidence,
not a count of unique test functions or a semantic acceptance percentage.
Pilot fixes additionally passed CLI/HTTP/command/Domain race suites and real
SQLite/PostgreSQL atomic-return direct/review cases. Installed final-fix source
`6d098d1c0caf51405c9fc5b73c8de1c09848291d` passed the complete offline distribution
verifier locally, including the newly packaged schemas and compatible metadata.

The [controlled Codex observations](controlled-codex-2026-10-09/observations.json)
retain A/B/C actual original-quality checks, code commits, immutable C recovery,
independent original-revision assessments, Task DONE and empty contract folders.
Provider usage/cost stays unknown. One fixed-order task, parent orchestration and
C source remediation do not establish savings, statistical significance or
steady-state performance. No explicit Outcome achievement is inferred.

T01–T96 have ordered candidate mappings and a reporter binding them to real test
execution. Passing names do not automatically certify every assertion. Bounded
mixed local lock and signed load experiments do not certify every network fault,
million-row history or production SLA. Managed correction rounds and Claude,
Hermes, OpenClaw and Woobe runtime certification remain consumer-specific tests;
installer destinations and automated installed correction are separately tested.
