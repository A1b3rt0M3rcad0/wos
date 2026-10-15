# ADR-030: Durable terminal resolution of a signed return

Status: accepted for implementation in the single operational hardening PR; merge requires owner approval.

A rejected or missing return receipt cannot prove that an in-flight request will
never commit. Local deletion or re-signing therefore remains forbidden. A separate
DSSE intention requests a server decision about the original exact envelope,
payload digest, request ID, idempotency key, Scope, Principal and original CID.
The current credential and signing policy remain authoritative.

Acceptance and resolution acquire the signed protocol Namespace lock and then
the Outcome guard in the same order. Acceptance already committed wins and its
original issuer receipt is returned. Otherwise a new immutable negative decision
and signed resolution receipt are committed atomically. All subsequent returns
consult this table, independent of short idempotency retention. No Task,
assessment, lease, fence or Outcome revision is changed by the negative decision.

Resolution is not a successful return and is stored separately from accepted
operation results. The resolution command itself may have a durable successful
response, for explicit response-loss reconciliation. A second distinct resolution
cannot overwrite the first. Revoked/expired credentials, changed signer authority
or expired leases require existing authorized recovery; this feature grants no
administrative bypass. Signed history is never rewritten.

CLI recovery persists and verifies proof before allowing a new intention. Original
bytes, edited draft and receipt are archived in the confined workspace. A new
return must pass unchanged live CAS, authority and fencing checks. Resolving a
return never claims to undo external execution or close the execution contract.

The original envelope travels alongside the small resolution envelope, whose
signed payload binds its canonical digest, avoiding nested base64 expansion.
All transport size limits remain explicit and bounded.
