# Release queue

Every distributable change adds a uniquely named JSON file:

```json
{"bump":"patch","summary":"Explain the resulting behavior."}
```

Allowed bumps: patch, minor, major. Highest declared bump wins. Breaking changes use major even before 1.0; behavior additions use minor; corrections use patch. The first prepared release uses initial VERSION (0.1.0). Source package versions are synchronized by release preparation, rather than manually advanced per feature. `node tools/releases/version.mjs plan` is read-only; `prepare` consumes entries and writes versions, notes, release-manifest.json and the generated README release section. The owner may request preparation in the feature PR; otherwise the automatic workflow creates a separate release PR. Preparation is not publication. If preparation is interrupted by an I/O failure, revert only its changes in the release branch and retry; source queues stay on master until the PR merges.

The README block between `<!-- wos-release:start -->` and `<!-- wos-release:end -->` follows `VERSION`; release compatibility tests reject drift. Keep product prose outside that block. GitHub/npm badges query the published versions rather than a manually maintained source snapshot.
