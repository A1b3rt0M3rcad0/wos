# Release queue

Every distributable change adds a uniquely named JSON file:

```json
{"bump":"patch","summary":"Explain the resulting behavior."}
```

Allowed bumps: patch, minor, major. Highest declared bump wins. Breaking changes use major even before 1.0; behavior additions use minor; corrections use patch. The first prepared release uses initial VERSION (0.1.0). Source package versions are synchronized by the release PR, not manually advanced per feature PR. `node tools/releases/version.mjs plan` is read-only; `prepare` consumes entries and writes versions, notes and release-manifest.json. Preparation is not publication. If preparation is interrupted by an I/O failure, revert only its changes in the release branch and retry; source queues stay on master until the PR merges.
