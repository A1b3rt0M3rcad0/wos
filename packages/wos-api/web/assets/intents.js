// A dispatched intent is immutable, scoped and replayed byte-for-byte.
export function createIntent(name,command,scope,key=crypto.randomUUID()) {
 const body=JSON.stringify({command});return Object.freeze({name,body,key,scope:Object.freeze({...scope})});
}
export function sameScope(a,b){return a.namespace_id===b.namespace_id&&a.outcome_id===b.outcome_id;}
export function reconcileDraft(local,latest) {
 const next=structuredClone(local);if('expected_version'in next)next.expected_version=latest.version;if('expected_roadmap_version'in next)next.expected_roadmap_version=latest.version;if('expected_draft_version'in next)next.expected_draft_version=latest.draft?.draft_version;
 if('issue_expected_version'in next)next.issue_expected_version=latest.version;
 return next;
}
