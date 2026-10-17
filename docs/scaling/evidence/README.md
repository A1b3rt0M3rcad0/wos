# Scaling evidence and its boundaries

These records cover different boundaries:

- [Signed SQL qualification](qualification/README.md): immutable sources, every
  completed repetition, the retained baseline failure and all raw final case logs.
- [Local validation](../validation.md) and [its receipt](validation.json): test
  counts, source scopes, optional skips and private-log fingerprints.
- The actual deployment smoke below: independent processes, verified TLS,
  restarts, session/issuer identity and idempotent-result continuity.

## Actual public-profile smoke

[`compose-smoke.json`](compose-smoke.json) records an actual Docker Compose run of the small PostgreSQL profile followed by the two-API/two-worker overlay. The executed binary is identified by SHA-256; observed production Go/SQL hashes match commit `aaa447520f9236bd0a6b929fa1aeb7932ec0e246`, the same candidate as the [signed SQL qualification](qualification/README.md). The record checks 444 production Go/SQL input hashes, four deployment input hashes and the exact laboratory harness hash. The binary reports `1.0.0` with commit `aaa4475`; the exact binary metadata and SHA remain part of the record. This laboratory artifact is not a release publication. The [earlier `bdc67dc` smoke](compose-smoke-bdc67dc.json) is retained separately rather than relabeled as the final binary.

The check established a TLS browser session and Outcome on the single complete process, retained them after the transition, routed 32 authenticated requests evenly to both API replicas, checked private worker health and exclusion of product routes, and restarted an API and worker while retaining issuer identity, the browser session and the exact original idempotent result. Worker containers mounted only the database password, without the bootstrap credential or issuer seed.

The temporary test image copied the candidate binary and secret entrypoint into the already available `0.4.2` runtime image. It was not published. This validates the candidate process/profile behavior; it does not establish that the normal production Dockerfile build or registry authentication was tested. The run preserved inherited Docker configuration and proxy selectors, used only a private `BUILDX_CONFIG` directory, and verified TLS through the profile's public local CA with the expected hostname.

An initial attempt exposed ambient outbound proxy routing of dynamically resolved private API IPs. The final run uses Caddy 2.10.2's built-in `network_proxy none` only on that private upstream transport. It preserves ambient proxy variables and does not change TLS verification or a host/network policy. The exact supplied-image configuration was validated before the successful run.

[`compose-smoke-harness.py`](compose-smoke-harness.py) preserves the executed laboratory harness, including this managed environment's `/workspace/wos` path. It is historical qualification evidence, not an installer. Public operator instructions and portable Compose files are in [the deployment guide](../../../deploy/scaling/README.md). The temporary fixture's containers, networks and database/TLS volumes were removed after the run. Private bootstrap credentials and issuer seed are excluded from these records.

This run had no external delivery destination and establishes neither delivery throughput nor production high availability. Signed recovery, receiver deduplication, lease/fencing and concurrent security rules remain separate contract/failure tests. A shared database and ingress are still single points of failure in the reference profile.
