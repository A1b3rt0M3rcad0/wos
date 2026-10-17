# Installation data and Profiles

## Target behavior

`wos init` initializes installation-scoped local data. Ordinary commands work from any directory. There is no additional user-facing `home` concept and no per-project initialization requirement. Agent Skill discovery directories remain runtime configuration; they are separate from WOS operational data.

The installation data defaults are Linux `$XDG_DATA_HOME/wos` (or `~/.local/share/wos`), macOS `~/Library/Application Support/wos`, and Windows `%LOCALAPPDATA%/WOS`. An absolute `WOS_DATA_DIR` overrides them. Never repurpose the shell's `HOME` variable as a temporary task path.

```sh
wos init
wos profile list
wos profile use executor
wos profile show executor
wos --profile executor work context --for-agent --output json
WOS_PROFILE=reviewer wos review list --output json
```

Only the temporary selection commands above change one invocation's Profile. `profile use` persists an interactive default. Explicit `--profile` wins over `WOS_PROFILE`, which wins over the active default. Never silently choose between multiple candidates if none is selected.

## Existing components

- `packages/wos-cli/client.go`: CLI dispatch, current-directory behavior, global options.
- `packages/wos-cli/profiles.go`: `ProjectV2`, `ProfileV2`, sealed destination/identity binding, pending operations.
- `packages/wos-cli/profile_commands_v2.go`: creation, enrollment, list/inspect/recover/remove.
- `packages/wos-cli/profile_workspace.go`: per-Profile project binding.
- `packages/wos-cli/profile_trust_v2.go`: explicit independently approved issuer trust updates.
- `packages/wos-cli/schemas/project-v2.schema.json` and `profile-v2.schema.json`: strict packaged internal schemas.
- `packages/wos-cli/locks_v2_*`, `workspace_v2.go`: protected path operations and CAS writes.

## Operator provisioning

The CLI owns Profile creation; agents do not write a proposal document. A trusted operator supplies exact approved bindings and an authorized enrollment:

```sh
wos profile create executor --server APPROVED_ORIGIN --server-id SERVER_UUID --tenant TENANT_UUID --issuer-fingerprint sha256:APPROVED_DIGEST --enrollment ENROLLMENT_UUID --generate-signing-key --credential-ref env:WOS_EXECUTOR_TOKEN
```

`--outcome UUID` is an optional default. `--signing-key-stdin` is an explicit alternative to generation, and `--token-stdin` an alternative to a protected credential reference; token and key cannot share stdin. For headless hosts without a native keyring, use protected external references: replace key generation with `--private-key-ref env:WOS_EXECUTOR_KEY` alongside `--credential-ref env:WOS_EXECUTOR_TOKEN`; do not combine this reference mode with stdin/generation flags. The external key value is a canonical base64 Ed25519 seed/private key. Actual secret values never belong in an argument. `profile inspect/recover/trust` retain live identity/enrollment/issuer checks.

## Implementation steps

1. Resolve one absolute installation data root before opening operational state. Create owned private directories atomically; reject symlink traversal/unmanaged conflicting content.
2. Add idempotent `init`, persistent active selection, and Profile list/use operations. Serialize updates and use atomic publication; never overwrite a concurrently changed selection.
3. Preserve each Profile's own ServerID/origin, Tenant, credential reference, key reference and contract tenant. Profiles for two servers must not share a mutable installation-level destination.
4. Resolve selection once. Pass the selected value through nested calls, recovery and foreground keepalive. A different process running `profile use` must not redirect an active operation.
5. Keep credential values/private seeds outside command arguments, source files, JSON output and model context. Native keyring/protected external references retain existing permission and enrollment checks.
6. Refuse removal while local contracts, pending enrollment, ambiguous returns, or unknown content remain. Removing a local Profile does not revoke its remote identity or delete external secrets.
7. Add explicit migration rather than automatic upward-directory discovery/import. See [migration](migration.md).

## Acceptance and concurrency cases

- Initialize in directory A, operate in unrelated directory B; B receives no operational files.
- Two named Profiles can bind different servers/Tenants without rewriting one another.
- All selection precedence cases are deterministic; invalid named selection fails rather than falling back.
- A process pinned to executor stays executor while another process changes the active default to reviewer.
- Concurrent selection/creation/removal cannot corrupt index or lose an existing Profile.
- Missing secret/key, changed ServerID, unapproved issuer rotation, stale binding MAC, symlink and case-insensitive name collision fail safely.
- Existing pending intentions remain recoverable against their original identity/destination after migration.

CAW-02 depends on the architecture decision and unblocks contract, multiagent, Skills, and migration acceptance. Exact current platform paths and successful test evidence belong in the branch validation record, not an assumed claim here.
