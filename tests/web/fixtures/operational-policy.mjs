// Current-state characterization data for browser journeys. Seed only through
// existing public commands on the fresh UX test server. This fixture does not
// claim that operational_v2, automatic Objective completion, or unsigned
// delivery/review exists. It does not render or use status-changing card
// controls: lifecycle facts are created through the server's real commands.

const FIXTURE_PROVIDER = "operational-policy-fixture";

function requireTitlePrefix(value) {
  if (typeof value !== "string" || !value.trim())
    throw new TypeError("A non-empty deterministic titlePrefix is required");
  return value.trim();
}

async function issueCredential(wos, {
  principalID,
  actorID,
  permissions,
}) {
  let admin = await wos.api(`/namespaces/${wos.namespace}/administration`);
  let expectedVersion = admin.namespace_version;
  const priorCredentialIDs = new Set(admin.credentials.map((entry) => entry.id));

  if (permissions) {
    const grant = await wos.api("/security/commands", {
      namespace_id: wos.namespace,
      expected_namespace_version: expectedVersion,
      operation: "set_grant",
      principal_id: principalID,
      permissions,
    });
    expectedVersion = grant.result.namespace_version;
  }

  const issued = await wos.api("/security/commands", {
    namespace_id: wos.namespace,
    expected_namespace_version: expectedVersion,
    operation: "issue_credential",
    principal_id: principalID,
    actor_ref: { kind: "human", provider: FIXTURE_PROVIDER, id: actorID },
    expires_at: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  });
  const current = await wos.api(`/namespaces/${wos.namespace}/administration`);
  const credential = current.credentials.find(
    (entry) => entry.principal_id === principalID && !priorCredentialIDs.has(entry.id),
  );
  if (!credential) throw new Error(`Issued fixture credential ${actorID} was not persisted`);
  return {
    principal_id: principalID,
    token: issued.token,
    credential_id: credential.id,
  };
}

// Creates a current-state data set from commands supported by the test server.
// The caller's test harness owns protocol configuration. Each title, rationale,
// source reference, and timestamp is stable; WOS-generated entity IDs and
// credentials remain server-issued.
// The caller can use the returned credential tokens for browser authorization
// characterization, but must never print or persist them.
export async function seedOperationalCharacterization(wos, {
  titlePrefix = "Operational characterization",
} = {}) {
  const prefix = requireTitlePrefix(titlePrefix);
  const { scope, value: outcome } = await wos.outcome(`${prefix}: shared outcome`);
  const ref = (kind, record) => ({ ...scope, kind, id: record.id });
  const base = `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`;
  const read = async (collection, id) =>
    (await wos.api(`${base}/${collection}/${id}`)).value;

  await wos.command("add_criterion", {
    owner: ref("outcome", outcome),
    expected_version: outcome.version,
    title: `${prefix}: outcome condition has been checked`,
    required: true,
    verification_mode: "attestation",
  });
  // Read the actual version rather than deriving it from a fixture-side
  // assumption about version increments.
  const outcomePath = `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`;
  const outcomeSnapshot = (await wos.api(outcomePath)).value;
  const activeOutcome = await wos.command("activate_outcome", {
    scope,
    expected_version: outcomeSnapshot.version,
  });

  const createObjective = (title, parentObjectiveID) => wos.command("create_objective", {
    scope,
    title: `${prefix}: ${title}`,
    priority: "normal",
    required_for_outcome: false,
    ...(parentObjectiveID ? { parent_objective_id: parentObjectiveID } : {}),
  });
  const planned = await createObjective("planned objective");
  let active = await createObjective("in-progress objective");
  const activeCriterion = await wos.command("add_criterion", {
    owner: ref("objective", active),
    expected_version: active.version,
    title: `${prefix}: active objective has a current condition`,
    required: true,
    verification_mode: "attestation",
  });
  active = await read("objectives", active.id);
  active = await wos.command("start_objective", {
    scope,
    objective_id: active.id,
    expected_version: active.version,
  });
  const nestedPlanned = await createObjective("nested planned objective", active.id);

  const historical = await createObjective("historically achieved objective");
  const historicalCriterion = await wos.command("add_criterion", {
    owner: ref("objective", historical),
    expected_version: historical.version,
    title: `${prefix}: historical objective condition`,
    required: true,
    verification_mode: "attestation",
  });
  let historicalCurrent = await read("objectives", historical.id);
  historicalCurrent = await wos.command("start_objective", {
    scope,
    objective_id: historicalCurrent.id,
    expected_version: historicalCurrent.version,
  });
  historicalCurrent = await read("objectives", historical.id);
  await wos.command("attest_criterion", {
    owner: ref("objective", historical),
    criterion_id: historicalCriterion.id,
    criterion_revision: historicalCriterion.criterion_revision,
    expected_version: historicalCurrent.version,
    result: "met",
    rationale: `${prefix}: the existing attestation command records the actual decision.`,
  });
  historicalCurrent = await read("objectives", historical.id);
  const achieved = await wos.command("achieve_objective", {
    scope,
    objective_id: historicalCurrent.id,
    expected_version: historicalCurrent.version,
    reason: "Preserve an actual historical Objective conclusion in the fixture.",
  });

  const cancelled = await createObjective("cancelled objective");
  const cancelledObjective = await wos.command("cancel_objective", {
    scope,
    objective_id: cancelled.id,
    expected_version: cancelled.version,
    reason: "This deterministic characterization item is out of scope.",
  });

  const createTask = (title, objective, lifecycle = "backlog") => wos.command("create_work_item", {
    scope,
    title: `${prefix}: ${title}`,
    description: "A bounded fixture task whose state is persisted by WOS.",
    priority: "normal",
    lifecycle,
    objective_id: objective.id,
    execution_spec: {
      instructions: [`Perform the scoped ${title} characterization.`],
      constraints: ["Do not infer authority from assignment or presentation state."],
      deliverables: ["A current, inspectable result."],
    },
  });
  const tasks = {
    backlog: await createTask("backlog task", planned),
    todo: await createTask("to-do task", active, "todo"),
    nestedBacklog: await createTask("nested backlog task", nestedPlanned),
    inProgress: await createTask("in-progress task", active),
    accepted: await createTask("accepted task with recorded conclusion", active),
    cancelled: await createTask("cancelled task", active),
    hardDependent: await createTask("hard-dependent task", active),
    advisoryDependent: await createTask("advisory-dependent task", active),
    directlyBlocked: await createTask("directly blocked task", active),
  };

  const acceptedCriterion = await wos.command("add_criterion", {
    owner: ref("work_item", tasks.accepted),
    expected_version: tasks.accepted.version,
    title: `${prefix}: accepted task condition`,
    required: true,
    verification_mode: "attestation",
  });
  let accepted = await read("work-items", tasks.accepted.id);
  accepted = await wos.command("activate_work_item", {
    scope,
    work_item_id: accepted.id,
    expected_version: accepted.version,
  });
  accepted = await read("work-items", accepted.id);
  accepted = await wos.command("claim_work_item", {
    scope,
    work_item_id: accepted.id,
    expected_version: accepted.version,
    ttl_seconds: 900,
  });
  accepted = await read("work-items", accepted.id);
  await wos.command("attest_criterion", {
    owner: ref("work_item", accepted),
    criterion_id: acceptedCriterion.id,
    criterion_revision: acceptedCriterion.criterion_revision,
    expected_version: accepted.version,
    result: "met",
    rationale: `${prefix}: the present attestation operation records the criterion decision.`,
  });
  accepted = await read("work-items", accepted.id);
  tasks.accepted = await wos.command("complete_work_item", {
    scope,
    work_item_id: accepted.id,
    expected_version: accepted.version,
    claim_id: accepted.current_lease.claim_id,
    fencing_token: accepted.current_lease.fencing_token,
    result_summary: "The current-state fixture records a real completed result.",
    reason: "The active Task criterion has an explicit current assessment.",
  });

  let inProgress = await read("work-items", tasks.inProgress.id);
  inProgress = await wos.command("activate_work_item", {
    scope,
    work_item_id: inProgress.id,
    expected_version: inProgress.version,
  });
  inProgress = await read("work-items", inProgress.id);
  tasks.inProgress = await wos.command("claim_work_item", {
    scope,
    work_item_id: inProgress.id,
    expected_version: inProgress.version,
    ttl_seconds: 900,
  });
  tasks.cancelled = await wos.command("cancel_work_item", {
    scope,
    work_item_id: tasks.cancelled.id,
    expected_version: tasks.cancelled.version,
    reason: "This deterministic characterization item is out of scope.",
  });

  const dependencies = {
    hard: await wos.command("add_dependency", {
      scope,
      source_ref: ref("work_item", tasks.hardDependent),
      target_ref: ref("work_item", tasks.todo),
      strength: "hard",
      reason: "The prerequisite remains open in this current-state fixture.",
    }),
    advisory: await wos.command("add_dependency", {
      scope,
      source_ref: ref("work_item", tasks.advisoryDependent),
      target_ref: ref("work_item", tasks.accepted),
      strength: "advisory",
      reason: "This completed result is useful context, not a hard gate.",
    }),
  };

  const evidence = await wos.command("register_evidence", {
    scope,
    evidence_type: "test_result",
    description: `${prefix}: a recorded observation linked to the accepted task`,
    source_ref: { provider: FIXTURE_PROVIDER, id: "accepted-task-observation" },
    captured_at: "2026-10-20T12:00:00Z",
  });
  const evidenceLink = await wos.command("create_evidence_link", {
    scope,
    evidence_id: evidence.id,
    target_ref: ref("work_item", tasks.accepted),
    stance: "supports",
    rationale: "Documentary support is not itself a Criterion Assessment.",
  });

  const issues = {
    direct: await wos.command("create_issue", {
      scope,
      objective_id: active.id,
      title: `${prefix}: access problem with direct impact`,
      severity: "major",
      description: "An Issue is an owned problem record, distinct from its Blocker.",
      affected_refs: [ref("work_item", tasks.directlyBlocked)],
    }),
    contextual: await wos.command("create_issue", {
      scope,
      objective_id: planned.id,
      title: `${prefix}: observation without a Blocker`,
      severity: "minor",
      description: "An Issue can be recorded without declaring a Blocker.",
      affected_refs: [ref("objective", planned)],
    }),
  };
  const blockers = {
    direct: await wos.command("create_blocker", {
      scope,
      blocked_ref: ref("work_item", tasks.directlyBlocked),
      cause_ref: ref("issue", issues.direct),
      propagation: "direct",
      description: "Only this explicitly targeted Task receives the direct Blocker.",
    }),
    subtree: await wos.command("create_blocker", {
      scope,
      blocked_ref: ref("objective", active),
      cause_ref: ref("issue", issues.direct),
      propagation: "subtree",
      description: "The Objective Blocker follows structural descendants only.",
    }),
  };

  // The bootstrap Principal and its sibling Credential characterize the
  // existing same-Principal case. The separate reviewer Principal is real
  // test authorization, not proof of signed-protocol independence.
  const siblingCredential = await issueCredential(wos, {
    principalID: "owner",
    actorID: "owner-sibling-credential",
  });
  const secondSiblingCredential = await issueCredential(wos, {
    principalID: "owner",
    actorID: "owner-second-sibling-credential",
  });
  const independentReviewer = await issueCredential(wos, {
    principalID: "operational-policy-reviewer",
    actorID: "operational-policy-reviewer",
    permissions: ["state:read", "assessment:write", "work:write"],
  });

  return {
    scope,
    outcome: activeOutcome,
    objectives: {
      planned,
      inProgress: active,
      nestedPlanned,
      achieved,
      cancelled: cancelledObjective,
      activeCriterion,
      historicalCriterion,
    },
    tasks,
    dependencies,
    issues,
    blockers,
    evidence,
    evidenceLink,
    credentials: {
      samePrincipal: [
        siblingCredential,
        secondSiblingCredential,
      ],
      independentReviewer,
    },
    ref,
    read,
    // The suite's fresh server starts on its configured current protocol. This
    // fixture intentionally does not perform protocol cutover or claim that
    // any profile-v2 capability is available.
  };
}
