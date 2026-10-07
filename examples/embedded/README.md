# Embedded Go host

Run `go run ./examples/embedded` from the repository root. Expected output: `lifecycle=achieved revision=5 certified=true`.

The example imports only public Core packages. The host supplies a UTC clock, UUIDv7 generator, authenticated identity and idempotency keys, then creates an Outcome, adds a required attestation criterion, activates, assesses and explicitly certifies it. Application owns the rules and transaction pipeline. The external-module boundary test copies this program into another Go module and executes the same journey.

This example uses the in-memory adapter, so state ends with the process. For durable embedding choose SQLite/PostgreSQL and implement the host's authentication/authorization policy. For untrusted remote consumers use the standalone server and [Go SDK example](../remote-client); the trusted `NewService` profile does not authenticate network callers.
