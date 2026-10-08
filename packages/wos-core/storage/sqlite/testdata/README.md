The gzip SQLite backup was created by running the original pre-contract server
from commit bcd1714ad7cded666890d0b5c9f711772c4f8d7a (prepared version 0.1.0),
not by synthesizing records with the new adapter. Public HTTP commands created
and activated an Outcome, attested its criterion, and claimed/completed work.
Local identity `legacy-fixture`; no credential or customer data. The JSON file
records immutable identities and the uncompressed SHA-256 checksum.

The migration test opens that backup with new migrations, replays original
claim/attestation receipts using original payloads, explicitly drains/cuts over,
and checks IDs, versions and the historic conclusion. `go run -mod=mod` in the
external consumer test resolves the pinned transitive module dependencies just
as a fresh module consumer must; Domain still imports no transport/storage.
