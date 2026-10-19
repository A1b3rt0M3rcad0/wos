import { expandWorkspaceDetails } from './workspace-action-fixture.mjs';
import { randomUUID } from "node:crypto";
import { test, expect, openOutcome, chooseArea } from "./ux-fixture.mjs";
import { revealMenuAction } from "./workspace-action-fixture.mjs";

const panel = (page) => page.locator("#canvas-details");
const base = (scope) => `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`;

async function selectRecord(page, title) {
  const search = page.getByRole("combobox", { name: "Search Outcome Board", exact: true });
  await search.fill(title);
  await page.getByRole("option").filter({ hasText: title }).first().click();
  await expect(panel(page).locator("#detail-title")).toHaveText(title);
  await expect(panel(page)).toHaveAttribute("data-surface", "detail");
  await expandWorkspaceDetails(panel(page));
}

async function action(page, name) {
  const direct = panel(page).locator(`[data-action-id="${name}"]`).filter({ visible: true });
  if (await direct.count()) await direct.first().click();
  else {
    const more = panel(page).locator("#entity-actions");
    if (await more.isVisible()) await more.click();
    else await panel(page).getByRole("button", { name: "More actions", exact: true }).click();
    const target = page.locator(`#action-options [data-action-id="${name}"]`);
    await revealMenuAction(target);
    await target.click();
  }
  await expect(page.locator("#command-form")).toBeVisible();
  await expect(page.locator("#command-select")).not.toBeVisible();
}

async function save(page, name) {
  const response = page.waitForResponse((response) => response.request().method() === "POST"
    && response.url().endsWith(`/commands/${name}`));
  await page.locator("#submit-command").click();
  const result = await response;
  const json = await result.json();
  expect(result.ok(), JSON.stringify(json)).toBe(true);
  await expect(page.locator("#command-form")).not.toBeVisible();
  await expect(page.locator("#submit-command")).toBeEnabled();
  await expect(page.locator("#notice")).not.toHaveClass(/error/);
  return json.value;
}

async function addListItem(page, label, value) {
  const field = page.getByRole("group", { name: label, exact: true });
  await field.getByRole("button", { name: `Add ${label.toLowerCase()} item`, exact: true }).click();
  await field.getByLabel(`${label} item`, { exact: true }).last().fill(value);
}

for (const viewport of [{ width: 2560, height: 1440 }, { width: 390, height: 844 }]) {
  test(`progressive planning, specification, update and criterion Evidence persist at ${viewport.width}px`, async ({ page, browser, wos }) => {
    test.setTimeout(180000);
    await page.setViewportSize(viewport);
    const { value: outcome, scope } = await wos.outcome("Publish reviewed installation instructions");
    const errors = [], mutations = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("request", (request) => {
      if (request.method() === "POST" && request.url().includes("/commands/"))
        mutations.push({ name: request.url().split("/").at(-1), payload: request.postDataJSON().command });
    });
    await wos.login(page);
    await openOutcome(page, outcome.title, { area: null });
    await selectRecord(page, outcome.title);
    await action(page, "create_objective");
    await page.getByLabel("Objective title", { exact: true }).fill("Installation instructions are usable");
    await expect(page.getByLabel("Required for Outcome", { exact: true })).toBeChecked();
    const objective = await save(page, "create_objective");
    expect(objective.required_for_outcome).toBe(true);
    expect(objective.criteria.items || []).toHaveLength(0);

    await selectRecord(page, objective.title);
    await action(page, "create_work_item");
    await expect(page.locator("#fields")).toContainText(objective.title);
    await page.getByLabel("Task title", { exact: true }).fill("Review installation on a clean machine");
    const task = await save(page, "create_work_item");
    expect(task.objective_id).toBe(objective.id);
    expect(task.lifecycle).toBe("backlog");
    expect(task.current_lease).toBeUndefined();
    expect(task.current_contract_id).toBeUndefined();

    await selectRecord(page, task.title);
    await action(page, "set_work_item_execution_spec");
    await addListItem(page, "Instructions", "Run the README installation commands on a clean machine");
    await addListItem(page, "Constraints", "Keep the documented production credential private");
    await addListItem(page, "Deliverables", "A recorded installation transcript");
    await save(page, "set_work_item_execution_spec");

    await selectRecord(page, task.title);
    await expect(panel(page)).toContainText("Run the README installation commands on a clean machine");
    await action(page, "append_entity_note");
    await page.getByLabel("Update", { exact: true }).fill("I finished the Task; this is a participant update awaiting verification.");
    const note = await save(page, "append_entity_note");
    expect(note.subject_ref.id).toBe(task.id);
    expect(note.type).toBe("update");
    expect(note.principal_id).toBe("owner");

    await selectRecord(page, task.title);
    await expect(panel(page)).toContainText("I finished the Task; this is a participant update awaiting verification.");
    await action(page, "add_criterion");
    await page.getByLabel("What must be true to consider this item satisfied?", { exact: true })
      .fill("The documented command starts a usable local workspace");
    await expect(page.getByLabel("How will it be verified?", { exact: true })).toHaveValue("evidence_review");
    const criterion = await save(page, "add_criterion");

    await selectRecord(page, task.title);
    await panel(page).getByRole("button").filter({ hasText: criterion.title }).first().click();
    await expect(panel(page).locator("#detail-title")).toHaveText(criterion.title);
    await action(page, "register_evidence_with_links");
    await expect(page.locator("#fields")).toContainText(criterion.title);
    await page.getByLabel("Evidence source", { exact: true }).selectOption("observation");
    await page.getByLabel("Description", { exact: true }).fill("I started the workspace using only the documented commands.");
    await page.getByLabel("Why is this relevant?", { exact: true }).fill("Records the installation sequence for independent review.");
    await expect(page.getByLabel("Evidence stance", { exact: true })).toHaveValue("supports");
    const evidence = await save(page, "register_evidence_with_links");
    expect(evidence.observation.principal_id).toBe("owner");
    expect(evidence.evidence.source_ref.provider).toBe("wos.entity-note");
    expect(evidence.links[0]).toMatchObject({ target_ref: { ...scope, kind: "work_item", id: task.id }, criterion_id: criterion.id, stance: "supports" });

    const persisted = (await wos.api(`${base(scope)}/work-items/${task.id}`)).value;
    expect(persisted.lifecycle).toBe("backlog");
    expect(persisted.execution_spec.instructions).toEqual(["Run the README installation commands on a clean machine"]);
    expect(persisted.criteria.assessments || []).toHaveLength(0);
    expect(Object.keys(persisted.criteria.current_assessments || {})).toHaveLength(0);
    expect(persisted.current_contract_id).toBeUndefined();
    expect(persisted.current_lease).toBeUndefined();
    expect(mutations.map((entry) => entry.name)).toEqual([
      "create_objective", "create_work_item", "set_work_item_execution_spec", "append_entity_note",
      "add_criterion", "register_evidence_with_links",
    ]);
    expect(mutations.find((entry) => entry.name === "register_evidence_with_links").payload.links[0].criterion_revision).toBe(String(criterion.criterion_revision));

    await page.reload();
    await expect(page.locator("#workspace-loading")).not.toBeVisible();
    await selectRecord(page, task.title);
    await expect(panel(page)).toContainText(criterion.title);
    await expect(panel(page)).toContainText("I finished the Task; this is a participant update awaiting verification.");
    await expect(panel(page)).toContainText(/No assessment|Not assessed|Awaiting assessment|Pending assessment/i);
    const secondContext = await browser.newContext({ viewport });
    try {
      const second = await secondContext.newPage();
      await wos.login(second);
      await openOutcome(second, outcome.title, { area: null });
      await selectRecord(second, task.title);
      await expect(panel(second)).toContainText(criterion.title);
      await expect(panel(second)).toContainText("I finished the Task; this is a participant update awaiting verification.");
      await expect(panel(second)).toContainText(/No assessment|Not assessed|Awaiting assessment|Pending assessment/i);
      await chooseArea(second, "Work");
      await second.getByLabel("Work status", { exact: true }).selectOption("work_items");
      await second.locator("#content").getByRole("button").filter({ hasText: task.title }).first().click();
      await expect(second.locator("#detail-dialog")).toBeVisible();
      await expect(second.locator("#detail-content")).toContainText(criterion.title);
    } finally { await secondContext.close(); }
    expect(errors).toEqual([]);
  });
}

test("one contextual evidence intention preserves its receipt when the request is repeated", async ({ wos }) => {
  const { scope, value: outcome } = await wos.outcome("Atomic contextual evidence receipt");
  const criterion = await wos.command("add_criterion", { owner: { ...scope, kind: "outcome", id: outcome.id },
    expected_version: outcome.version, title: "Reviewed installation report", required: true, verification_mode: "evidence_review" });
  const current = (await wos.api(base(scope))).value;
  const key = randomUUID();
  const command = { scope, evidence: { evidence_type: "attestation", description: "A participant observed the installation" },
    observation: { subject_ref: { ...scope, kind: "outcome", id: outcome.id }, criterion_id: criterion.id, body: "I observed the installation" },
    links: [{ target_ref: { ...scope, kind: "outcome", id: outcome.id }, target_expected_version: String(current.version),
      criterion_id: criterion.id, criterion_revision: String(criterion.criterion_revision), stance: "supports", rationale: "Documents the observed instructions" }] };
  async function send(payload = command, intent = key) {
    const response = await fetch(`${wos.url}/api/v1/commands/register_evidence_with_links`, { method: "POST",
      headers: { Authorization: `Bearer ${wos.token}`, "Content-Type": "application/json", "Idempotency-Key": intent },
      body: JSON.stringify({ command: payload }) });
    return { status: response.status, result: await response.json() };
  }
  const first = await send(), replay = await send();
  expect(first.status).toBe(200);
  expect(replay.status).toBe(200);
  expect(replay.result.idempotent_replay).toBe(true);
  expect(replay.result.value).toEqual(first.result.value);
  const invalid = structuredClone(command);
  invalid.links.push({ target_ref: { ...scope, kind: "blocker", id: randomUUID() }, stance: "context", rationale: "Invalid target" });
  const failed = await send(invalid, randomUUID());
  expect(failed.status).toBeGreaterThanOrEqual(400);
  const state = await wos.api(`${base(scope)}/state`);
  expect(state.evidence).toHaveLength(1);
  expect(state.evidence_links).toHaveLength(1);
  expect(state.outcome.criteria.assessments || []).toHaveLength(0);
});

test("a preparation remedy retains the conclusion draft and never substitutes assessment or automatic acceptance", async ({ page, wos }) => {
  const { value: outcome, scope } = await wos.outcome("Explicit conclusion preparation");
  const owner = { ...scope, kind: "outcome", id: outcome.id };
  const criterion = await wos.command("add_criterion", { owner, expected_version: outcome.version,
    title: "Original agreed condition", required: true, verification_mode: "attestation" });
  let current = (await wos.api(base(scope))).value;
  await wos.command("activate_outcome", { scope, expected_version: current.version });
  current = (await wos.api(base(scope))).value;
  await wos.command("attest_criterion", { owner, expected_version: current.version,
    criterion_id: criterion.id, criterion_revision: criterion.criterion_revision,
    result: "met", rationale: "An authorized participant reviewed the original condition" });
  await wos.login(page);
  await openOutcome(page, outcome.title, { area: null });
  await selectRecord(page, outcome.title);
  await action(page, "achieve_outcome");
  const rationale = "I explicitly reviewed the agreed condition and request acceptance.";
  await page.getByLabel("Reason", { exact: true }).fill(rationale);
  current = (await wos.api(base(scope))).value;
  await wos.command("retire_criterion", { owner, expected_version: current.version,
    criterion_id: criterion.id });
  const rejected = page.waitForResponse((response) => response.url().endsWith("/commands/achieve_outcome"));
  await page.locator("#submit-command").click();
  const rejectedResponse = await rejected;
  const error = await rejectedResponse.json();
  expect(error.error.code).toBe("precondition_failed");
  await expect(page.getByLabel("Reason", { exact: true })).toHaveValue(rationale);
  await expect(page.getByRole("button", { name: "Compare with latest state", exact: true })).not.toBeVisible();
  const preparation = page.locator("#conflict-review");
  await preparation.getByRole("button", { name: "Add success criterion", exact: true }).click();
  await page.getByLabel("What must be true to consider this item satisfied?", { exact: true }).fill("The replacement condition must be independently reviewed");
  const added = page.waitForResponse((response) => response.url().endsWith("/commands/add_criterion"));
  await page.locator("#submit-command").click();
  expect((await added).ok()).toBe(true);
  await expect(page.locator("#command-select")).toHaveValue("achieve_outcome");
  await expect(page.getByLabel("Reason", { exact: true })).toHaveValue(rationale);
  await expect(page.locator("#command-help")).toContainText("Your draft is retained");
  current = (await wos.api(base(scope))).value;
  expect(current.lifecycle).toBe("active");
  expect(current.conclusion).toBeUndefined();
  expect(current.criteria.items.filter((item) => item.status === "active")).toHaveLength(1);
  const nextCriterion = current.criteria.items.find((item) => item.status === "active");
  expect(current.criteria.current_assessments?.[nextCriterion.id]).toBeUndefined();
  const stillPending = page.waitForResponse((response) => response.url().endsWith("/commands/achieve_outcome"));
  await page.locator("#submit-command").click();
  expect((await (await stillPending).json()).error.code).toBe("precondition_failed");
  await expect(page.getByLabel("Reason", { exact: true })).toHaveValue(rationale);
  await expect(page.getByRole("button", { name: "Compare with latest state", exact: true })).not.toBeVisible();
  expect((await wos.api(base(scope))).value.lifecycle).toBe("active");
});

test("a mixed planning preview creates two Objectives and four scoped backlog Tasks in one visible intention", async ({ page, wos }) => {
  const { value: outcome, scope } = await wos.outcome("A complete small operational plan");
  await wos.login(page);
  await openOutcome(page, outcome.title, { area: null });
  await selectRecord(page, outcome.title);
  await action(page, "create_plan_items");
  await page.getByLabel("One title per line", { exact: true }).fill([
    "Installation works", "Documentation is reviewed", "Install in a clean environment",
    "Record the installation transcript", "Review the documented commands", "Review recovery instructions",
  ].join("\n"));
  await page.getByRole("button", { name: "Preview plan", exact: true }).click();
  const rows = page.locator(".workspace-plan-row");
  await expect(rows).toHaveCount(6);
  await rows.nth(0).getByLabel("Item type", { exact: true }).selectOption("objective");
  await rows.nth(1).getByLabel("Item type", { exact: true }).selectOption("objective");
  for (const index of [2, 3, 4, 5]) {
    await rows.nth(index).getByLabel("Item type", { exact: true }).selectOption("work_item");
    await rows.nth(index).getByLabel("Parent Objective", { exact: true }).selectOption(`local:item-${index < 4 ? 1 : 2}`);
  }
  const result = await save(page, "create_plan_items");
  expect(result.items).toHaveLength(6);
  const state = await wos.api(`${base(scope)}/state`);
  expect(state.objectives).toHaveLength(2);
  expect(state.objectives.every((objective) => objective.required_for_outcome)).toBe(true);
  expect(state.work_items).toHaveLength(4);
  expect(state.work_items.every((task) => task.lifecycle === "backlog" && !task.current_lease && !task.current_contract_id)).toBe(true);
  expect(state.work_items.filter((task) => task.objective_id === result.refs["item-1"].id)).toHaveLength(2);
  expect(state.work_items.filter((task) => task.objective_id === result.refs["item-2"].id)).toHaveLength(2);
  await selectRecord(page, "Installation works");
  await expect(panel(page)).toContainText("Install in a clean environment");
  await expect(panel(page)).toContainText("Record the installation transcript");
});
