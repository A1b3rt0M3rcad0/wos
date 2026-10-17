// Every canvas record is created through the public command API in a fresh
// test server. Visual position never creates a dependency or changes lifecycle.
export async function seedCanvas(wos, { title = "Ship a connected workspace", extraTasks = 0 } = {}) {
  if (!Number.isInteger(extraTasks) || extraTasks < 0 || extraTasks > 300)
    throw Error("Canvas acceptance fixtures are bounded to 300 additional Tasks");
  const { scope, value: outcome } = await wos.outcome(title);
  const ref = (kind, value) => ({ ...scope, kind, id: value.id });
  await wos.command("add_criterion", {
    owner: ref("outcome", outcome), expected_version: outcome.version,
    title: "A person can verify the delivered result", required: true,
    verification_mode: "evidence_review",
  });
  await wos.command("activate_outcome", { scope, expected_version: outcome.version + 1 });
  const objective = await wos.command("create_objective", {
    scope, title: "A clear path from intent to evidence", priority: "high",
    description: "Keep planning and real operational state connected.",
    required_for_outcome: false,
  });
  const child = await wos.command("create_objective", {
    scope, title: "A reviewable delivery", priority: "normal",
    parent_objective_id: objective.id, required_for_outcome: false,
  });
  const tasks = {};
  for (const [key, taskTitle, objectiveId] of [
    ["ready", "Map the first-time experience", child.id],
    ["blocked", "Publish the verified handoff", objective.id],
    ["dependent", "Review the completed delivery", child.id],
    ["advisory", "Check the editorial details", objective.id],
    ["unscoped", "Record the shared decision", null],
  ]) tasks[key] = await wos.command("create_work_item", {
    scope, title: taskTitle, priority: key === "ready" ? "high" : "normal",
    lifecycle: "todo", ...(objectiveId ? { objective_id: objectiveId } : {}),
    description: "A bounded, independently verifiable step.",
    execution_spec: {
      instructions: ["Perform only the scoped task", "Record the actual result"],
      constraints: ["Keep execution and independent review separate"],
      deliverables: ["A reviewable result"],
    },
  });
  const hard = await wos.command("add_dependency", {
    scope, source_ref: ref("work_item", tasks.dependent),
    target_ref: ref("work_item", tasks.ready), strength: "hard",
    reason: "Review requires the actual completed delivery",
  });
  const advisory = await wos.command("add_dependency", {
    scope, source_ref: ref("work_item", tasks.advisory),
    target_ref: ref("work_item", tasks.ready), strength: "advisory",
    reason: "Editorial checking benefits from the delivery but may start earlier",
  });
  const issue = await wos.command("create_issue", {
    scope, title: "Access to the review environment is missing", severity: "major",
    description: "A problem record is separate from its blocking impact.",
    affected_refs: [ref("work_item", tasks.blocked)],
  });
  const blocker = await wos.command("create_blocker", {
    scope, blocked_ref: ref("work_item", tasks.blocked),
    cause_ref: ref("issue", issue), propagation: "direct",
    description: "Wait for verified environment access",
  });
  const evidence = [], evidenceLinks = [];
  for (const stance of ["supports", "contradicts", "context"]) {
    const record = await wos.command("register_evidence", {
      scope, evidence_type: "inspection", description: `${stance}: review observation`,
      source_ref: { provider: "canvas-acceptance", id: stance },
      captured_at: "2026-10-16T12:00:00Z",
    });
    evidence.push(record);
    evidenceLinks.push(await wos.command("create_evidence_link", {
      scope, evidence_id: record.id, target_ref: ref("work_item", tasks.ready),
      stance, rationale: "Observation stance is explicit and does not assess a criterion",
    }));
  }
  const additional = [];
  for (let i = 0; i < extraTasks; i++) additional.push(await wos.command("create_work_item", {
    scope, title: i === extraTasks - 1 ? "Find the off-page verification task" : `Additional task ${String(i + 1).padStart(3, "0")}`,
    priority: "normal", lifecycle: "backlog", objective_id: objective.id,
  }));
  return { scope, outcome, objective, child, tasks, issue, blocker, hard, advisory, evidence, evidenceLinks, additional, ref };
}

export async function provisionCanvasViewer(wos) {
  const snapshot = await wos.api(`/tenants/${wos.tenant}/administration`);
  const grant = await wos.api("/security/commands", {
    tenant_id: wos.tenant, expected_tenant_version: snapshot.tenant_version,
    operation: "set_grant", principal_id: "canvas-viewer", permissions: ["state:read"],
  });
  return wos.api("/security/commands", {
    tenant_id: wos.tenant, expected_tenant_version: grant.result.tenant_version,
    operation: "issue_credential", principal_id: "canvas-viewer",
    actor_ref: { kind: "human", provider: "canvas-acceptance", id: "canvas-viewer" },
    expires_at: new Date(Date.now() + 3600000).toISOString(),
  });
}

export async function seedCanvasRoadmap(wos, canvas, { published = false, active = false } = {}) {
  if (active && !published) throw Error("A Draft cannot be activated");
  const { scope, outcome, tasks, ref } = canvas;
  const created = await wos.command("create_roadmap", {
    scope, plan_scope: { kind: "outcome", id: outcome.id },
    title: "A deliberate path to a verified delivery",
  });
  const path = `/tenants/${scope.tenant_id}/outcomes/${scope.outcome_id}/roadmaps/${created.id}`;
  const read = async () => (await wos.api(path)).value;
  await wos.command("open_roadmap_draft", { scope, roadmap_id: created.id, expected_version: created.version });
  let roadmap = await read();
  await wos.command("replace_roadmap_draft", {
    scope, roadmap_id: created.id, expected_version: roadmap.version,
    expected_draft_version: roadmap.draft.draft_version,
    nodes: [
      { node_key: "discovery", node_type: "phase", title: "Discover and verify", position: 0 },
      { node_key: "build", node_type: "reference", title: "Map the experience", position: 0,
        parent_node_key: "discovery", target_ref: ref("work_item", tasks.ready) },
      { node_key: "handoff", node_type: "reference", title: "Publish a handoff", position: 1,
        parent_node_key: "discovery", target_ref: ref("work_item", tasks.unscoped) },
      { node_key: "reviewed", node_type: "milestone", title: "Delivery reviewed", position: 1 },
    ],
    after_links: [{ node_key: "handoff", after_node_key: "build" }],
  });
  roadmap = await read();
  if (published) {
    await wos.command("publish_roadmap_draft", {
      scope, roadmap_id: created.id, expected_version: roadmap.version,
      expected_draft_version: roadmap.draft.draft_version,
    });
    roadmap = await read();
    if (active) {
      await wos.command("activate_roadmap_revision", {
        scope, roadmap_id: created.id, expected_version: roadmap.version,
        revision_number: roadmap.revisions[0].revision_number,
      });
      roadmap = await read();
    }
  }
  return roadmap;
}
