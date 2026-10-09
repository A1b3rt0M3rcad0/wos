# Signed workspace profiles (schema 2)

The administrator supplies the persistent server ID, issuer fingerprint, Namespace, Outcome, credential and authorized enrollment. Agent profiles contain secret references only. Onboarding does not grant permission on the server.

```sh
wosctl init --workspace-schema 2 --server https://wos.example.test \
  --server-id SERVER_UUID --namespace NAMESPACE_UUID --outcome OUTCOME_UUID
wosctl profile create executor_a --token-stdin --generate-signing-key \
  --enrollment ENROLLMENT_UUID --server https://wos.example.test \
  --server-id SERVER_UUID --issuer-fingerprint sha256:APPROVED_DIGEST
wosctl --profile executor_a auth status --output json
wosctl --profile executor_a profile inspect --output json
```

Provide the token through the host's protected standard input. No token or private-key flag accepts a secret value. Generated seeds and imported API credentials go only to the OS keyring. If its native credential service is unavailable, creation fails; configure explicit env or protected mounted references instead. Do not place administrative credentials in an executor profile.

A preprovisioned seed can be imported with `--signing-key-stdin --credential-ref env:WOS_TOKEN` instead of `--generate-signing-key`. The dedicated stdin accepts only a canonical base64 Ed25519 seed. Token and signing seed cannot share stdin in one invocation. Existing keyring entries survive interrupted provisioning; a different entry is rejected rather than overwritten.

For env/mounted secrets, supply a schema-2 profile proposal with `profile create NAME --file proposal.yaml` and the same explicit server approval flags. The generated [profile schema](../packages/wos-cli/schemas/profile-v2.schema.json) describes its fields. Provide an authorized `--enrollment` to register a new key, or an already registered active key whose fingerprint matches the selected private reference. Proposal references never transmit private key bytes to WOS. Env is provided by the host; mounted files must be outside the workspace, canonical regular files without links, and deny other accounts access. Windows uses a restricted DACL check whose native verification remains pending.

Selection is `--profile`, then process `WOS_PROFILE`, then the single profile. Multiple profiles without a selector fail with `profile_required`. `--workspace` takes precedence over `WOS_WORKSPACE`. No shared active-profile file is written. Profile names require exact case; case collisions and Windows reserved names are rejected.

The operational layout is:

```text
.wos/project.yaml
.wos/profiles/executor_a/profile.yaml
.wos/profiles/executor_a/contract/CONTRACT_UUID.yaml
```

Enrollment recovery is `wosctl --profile executor_a profile recover`. The frozen intention is authenticated and persisted before a network mutation. An unknown response remains `sent_unknown`; replay uses the original key and command, including original CAS values. Do not edit pending bytes or substitute another challenge. Server receipts make enrollment replay durable. This enrollment behavior does not yet imply durable work acquisition: that is P08.

`profile inspect` and `auth status` return compact identity, selected public key status and credential restrictions. They are observations; each domain command rechecks current grants, credential policy, scope and authority. `doctor` verifies destination/issuer and local signing-key fingerprint without revealing secret values. It does not acquire or renew work.

`profile remove` refuses pending intentions and nonempty contract directories. It removes only known local profile files and empty directories, preserves shared secrets, and does not revoke a remote credential or contract. A changed API credential or profile destination needs explicit new onboarding; local integrity tags do not silently rebind existing work.

Stable locks live outside the operational workspace. Unix lock files are private, are never unlinked and release on process death. Windows uses global named mutexes across interactive/service sessions and fails closed if unavailable; cross-compilation is not native certification. The file writer preserves editor changes observed before replacement. A noncooperating OS writer can race the final comparison/rename; profiles do not isolate hostile processes under the same OS account.

P08 work acquisition/batches/recovery and P09 sign/send/finish remain in progress. Until those commands are implemented, a schema-2 project rejects legacy substitutions. The project does not yet claim a complete schema-2 operational workflow or native keyring certification.
