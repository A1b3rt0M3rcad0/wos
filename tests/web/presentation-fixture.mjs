import { seedCanvas, seedCanvasRoadmap } from "./canvas-fixture.mjs";

// Illustration data is created only through the same public commands as real
// participants. Every journey receives a fresh server and SQLite database.
export async function seedPresentation(wos) {
  const data = await seedCanvas(wos, {
    title: "Deliver a workspace people can trust",
    extraTasks: 6,
  });
  const { scope, tasks } = data;
  const base = `/tenants/${scope.tenant_id}/outcomes/${scope.outcome_id}`;
  const read = async (collection, id) => (await wos.api(`${base}/${collection}/${id}`)).value;
  const progress = await wos.command("claim_work_item", {
    scope, work_item_id: tasks.advisory.id, expected_version: tasks.advisory.version, ttl_seconds: 900,
  });
  const claim = await wos.command("claim_work_item", {
    scope, work_item_id: tasks.unscoped.id, expected_version: tasks.unscoped.version, ttl_seconds: 900,
  });
  const completed = await wos.command("complete_work_item", {
    scope, work_item_id: claim.id, expected_version: claim.version,
    claim_id: claim.current_lease.claim_id, fencing_token: claim.current_lease.fencing_token,
    reason: "The bounded delivery was recorded explicitly",
    result_summary: "A reviewable decision log and evidence are available.",
  });
  tasks.advisory = progress;
  tasks.unscoped = completed;
  const artifact = await wos.command("register_artifact", {
    scope, name: "First-time experience guide", artifact_type: "document",
    uri: "https://example.test/deliveries/first-time-experience", media_type: "text/markdown",
  });
  await wos.command("register_artifact", {
    scope, name: "Keyboard review results", artifact_type: "test_report",
    uri: "https://example.test/deliveries/keyboard-review", media_type: "application/json",
  });
  const decision = await wos.command("propose_decision", {
    scope, title: "Keep review independent from delivery",
    proposal: "A separate participant checks the actual delivery against the agreed criteria.",
    alternatives: ["Independent review", "Self-reported completion"],
    chosen_alternative: "Independent review",
    rationale: "Recorded evidence and an explicit review preserve accountability across sessions.",
  });
  const roadmap = await seedCanvasRoadmap(wos, data, { published: true, active: true });
  await wos.command("open_roadmap_draft", {
    scope, roadmap_id: roadmap.id, expected_version: roadmap.version, base_revision_number: 1,
  });
  const draft = await read("roadmaps", roadmap.id);
  await wos.command("replace_roadmap_draft", {
    scope, roadmap_id: draft.id, expected_version: draft.version,
    expected_draft_version: draft.draft.draft_version,
    nodes: [
      ...draft.draft.nodes,
      { node_key: "followup", node_type: "phase", title: "Refine the shared handoff", position: 2 },
    ],
    after_links: draft.draft.after_links,
  });
  // A later contradictory assessment contests the historical conclusion; it
  // never silently reopens the achieved Objective or changes this Outcome.
  let reviewed = await wos.command("create_objective", {
    scope, title: "An independently reviewed handoff", priority: "normal", required_for_outcome: false,
  });
  const owner = { ...scope, kind: "objective", id: reviewed.id };
  const criterion = await wos.command("add_criterion", {
    owner, expected_version: reviewed.version, title: "A reviewer can reproduce the handoff",
    required: true, verification_mode: "attestation",
  });
  reviewed = await read("objectives", reviewed.id);
  await wos.command("attest_criterion", {
    owner, criterion_id: criterion.id, criterion_revision: criterion.criterion_revision,
    expected_version: reviewed.version, result: "met", rationale: "The original handoff was reproduced.",
  });
  reviewed = await read("objectives", reviewed.id);
  reviewed = await wos.command("achieve_objective", {
    scope, objective_id: reviewed.id, expected_version: reviewed.version,
    reason: "The current required criterion was independently checked.",
  });
  await wos.command("record_criterion_assessment", {
    owner, criterion_id: criterion.id, criterion_revision: criterion.criterion_revision,
    expected_version: reviewed.version, result: "not_met",
    rationale: "A later reproduction found a missing handoff instruction; inspect the conclusion without rewriting history.",
  });
  for (const title of ["Keep decisions connected to their evidence", "Prepare the next shared handoff"])
    await wos.outcome(title);
  return { ...data, artifact, decision, roadmap: await read("roadmaps", roadmap.id), reviewed, base, read };
}
