# Effective permissions and human action registry

`GET /api/v1/namespaces/{namespace_id}/effective-permissions?outcome_id=…` returns current permission names and `execution_authority: false`. The optional Outcome binds credential-policy scope; namespace metadata can be read before selecting an Outcome. The response is non-cacheable. MCP `wos_effective_permissions` adapts the same Application method.

The query checks authenticated read access, tests the supported permission allowlist through the composed authorizer and checks read access again. Authorization denials omit permission hints; infrastructure failures fail the query. Revoked read access cannot produce a successful hint response. Local embedded composition preserves its explicit trusted normal-command behavior while privileged hints still consult authorization.

`command-exposure.json` records all 100 operations with a class, journey, permission and presentation surface. The Go gate verifies every entry and compares its permission with the existing Application mapping. `actions.js` gives common intents explicit English labels, contexts and lifecycle/material guards. A Task lifecycle alone does not permit completion: legacy operations additionally require the effective protocol, current operational state and matching lease holder. Signed operations are profile-only. Browser hints never replace backend CAS, fencing, independent review, policy or authorization.

Dedicated forms, reference discovery and common-client wiring are a subsequent implementation slice. The English workspace remains compatible while those modules are integrated.
