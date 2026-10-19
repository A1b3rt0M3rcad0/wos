import { expandWorkspaceDetails } from './workspace-action-fixture.mjs';
import { test, expect, openOutcome } from "./ux-fixture.mjs";

const outcomePath = (scope) => `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`;

async function inspect(page, title, area) {
  if (area === "canvas") {
    await page.getByRole("combobox", { name: "Search Outcome Board", exact: true }).fill(title);
    await page.getByRole("option").filter({ hasText: title }).first().click();
  } else {
    await page.getByLabel("Work status", { exact: true }).selectOption("work_items");
    await page.locator("#content").getByRole("button").filter({ hasText: title }).first().click();
  }
  await expect(page.locator("#detail-title")).toHaveText(title);
}

async function openAction(page, name) {
  const direct = page.locator(`#detail-actions [data-action-id="${name}"], #detail-content [data-action-id="${name}"]`).filter({ visible: true });
  if (await direct.count()) await direct.first().click();
  else {
    await page.locator("#entity-actions").click();
    const target = page.locator(`#action-options [data-action-id="${name}"]`);
    if (!(await target.isVisible())) {
      const advanced = page.locator("#action-options details").filter({
        has: page.locator(`[data-action-id="${name}"]`),
      });
      if (await advanced.count() && !(await advanced.evaluate((element) => element.open))) {
        await advanced.locator(":scope > summary").click();
        await expect(advanced).toHaveAttribute("open", "");
      }
    }
    await target.click();
  }
  await expect(page.locator("#command-form")).toBeVisible();
}

async function save(page, name) {
  const response = page.waitForResponse((response) => response.request().method() === "POST"
    && response.url().endsWith(`/commands/${name}`));
  await page.locator("#submit-command").click();
  const result = await response;
  const json = await result.json();
  expect(result.ok(), JSON.stringify(json)).toBe(true);
  await expect(page.locator("#command-form")).not.toBeVisible();
  return json.value;
}

for (const area of ["canvas", "work"]) {
  test(`fresh ${area} inspector preserves real lease authority for explicit completion and release`, async ({ page, wos }) => {
    const { value: outcome, scope } = await wos.outcome(`Explicit lease operations from ${area}`);
    const criterion = await wos.command("add_criterion", { owner: { ...scope, kind: "outcome", id: outcome.id },
      expected_version: outcome.version, title: "An agreed Outcome condition", required: true, verification_mode: "attestation" });
    const current = (await wos.api(outcomePath(scope))).value;
    await wos.command("activate_outcome", { scope, expected_version: current.version });
    const tasks = [];
    for (const operation of ["complete_work_item", "release_work_item"]) {
      tasks.push({ operation, value: await wos.command("create_work_item", { scope,
        title: `${area} ${operation} Task`, description: "Operate using the actual persisted reservation",
        priority: "normal", lifecycle: "todo" }) });
    }
    const writes = [];
    page.on("request", (request) => {
      if (request.method() === "POST" && /\/commands\/(complete_work_item|release_work_item)$/.test(request.url()))
        writes.push({ name: request.url().split("/").at(-1), command: request.postDataJSON().command });
    });
    await wos.login(page);
    await openOutcome(page, outcome.title, { area: area === "canvas" ? null : "work" });
    for (const { operation, value: task } of tasks) {
      await inspect(page, task.title, area);
      await openAction(page, "claim_work_item");
      const claimed = await save(page, "claim_work_item");
      expect(claimed.current_lease.principal_id).toBe("owner");
      // A reload removes any canonical record retained by the reservation form.
      // The next action must hydrate the current real lease from a fresh view.
      await page.reload();
      await expect(page.locator("#workspace-loading")).not.toBeVisible();
      if (area === "work" && await page.locator("#detail-dialog").isVisible()) await page.keyboard.press("Escape");
      await inspect(page, task.title, area);
      await openAction(page, operation);
      if (operation === "complete_work_item") await page.getByLabel("Task result", { exact: true })
        .fill("I recorded the requested external result for explicit completion.");
      await page.getByLabel("Reason", { exact: true }).fill("Use the current holder's actual reservation.");
      const result = await save(page, operation);
      const submitted = writes.findLast((entry) => entry.name === operation).command;
      expect(submitted.claim_id).toBe(claimed.current_lease.claim_id);
      expect(String(submitted.fencing_token)).toBe(String(claimed.current_lease.fencing_token));
      expect(String(submitted.expected_version)).toBe(String(claimed.version));
      expect(result.lifecycle).toBe(operation === "complete_work_item" ? "done" : "todo");
      expect(result.current_lease).toBeUndefined();
      expect(result.current_contract_id).toBeUndefined();
      if (operation === "complete_work_item") expect(result.conclusion.principal_id).toBe("owner");
      if (area === "work") await page.keyboard.press("Escape");
    }
    expect(criterion.criterion_revision).toBe(1);
    expect(writes.map((write) => write.name)).toEqual(["complete_work_item", "release_work_item"]);
  });
}

test("editing Issue, Blocker and Criterion preserves descriptions beyond their inspector preview", async ({ page, wos }) => {
  const { value: outcome, scope } = await wos.outcome("Preserve full authored definitions");
  const long = "Retain this authored definition in full. ".repeat(180).trim();
  const task = await wos.command("create_work_item", { scope, title: "Owner of the long criterion",
    priority: "normal", lifecycle: "backlog" });
  const taskRef = { ...scope, kind: "work_item", id: task.id };
  const issue = await wos.command("create_issue", { scope, title: "Long authored Issue", description: long,
    severity: "minor", affected_refs: [taskRef] });
  const blocker = await wos.command("create_blocker", { scope, blocked_ref: taskRef,
    cause_ref: { ...scope, kind: "issue", id: issue.id }, description: long, propagation: "direct" });
  const criterion = await wos.command("add_criterion", { owner: taskRef, expected_version: task.version,
    title: "A long authored criterion", description: long, required: true, verification_mode: "attestation" });
  await wos.login(page);
  await openOutcome(page, outcome.title, { area: null });
  await inspect(page, issue.title, "canvas");
  await openAction(page, "update_issue");
  await expect(page.getByLabel("Description", { exact: true })).toHaveValue(long);
  await page.getByLabel("Issue title", { exact: true }).fill("A renamed Issue keeps its authored description");
  await save(page, "update_issue");
  expect((await wos.api(`${outcomePath(scope)}/issues/${issue.id}`)).value.description).toBe(long);

  // The relation control identifies the actual caused Blocker instead of
  // relying on its deliberately repeated long description as a search title.
  await inspect(page, "A renamed Issue keeps its authored description", "canvas");
  await expandWorkspaceDetails(page.locator("#detail-content"), {history:false});
  await page.locator(`#detail-content [data-reference-kind="blocker"][data-reference-id="${blocker.id}"]`).first().click();
  await openAction(page, "update_blocker_description");
  await expect(page.getByLabel("Description", { exact: true })).toHaveValue(long);
  await save(page, "update_blocker_description");
  expect((await wos.api(`${outcomePath(scope)}/blockers/${blocker.id}`)).value.description).toBe(long);

  await inspect(page, task.title, "canvas");
  await expandWorkspaceDetails(page.locator("#detail-content"), {history:false});
  await page.locator("#detail-content").getByRole("button", { name: criterion.title, exact: true }).click();
  await openAction(page, "revise_criterion");
  await expect(page.getByLabel("Description", { exact: true })).toHaveValue(long);
  await page.getByLabel("Criterion title", { exact: true }).fill("A revised criterion keeps its full definition");
  await page.getByLabel("Reason", { exact: true }).fill("Clarify the title without changing the authored description.");
  await save(page, "revise_criterion");
  const stored = (await wos.api(`${outcomePath(scope)}/work-items/${task.id}`)).value;
  expect(stored.criteria.items.find((item) => item.id === criterion.id).description).toBe(long);
});

test("criterion preparation targets a required definition beyond the first inspector page", async ({ page, wos }) => {
  const outcome = await wos.command("create_outcome", { namespace_id: wos.namespace,
    title: "Prepare the exact paginated criterion", desired_state: "Every agreed criterion is independently assessed",
    priority: "normal", initial_criteria: Array.from({ length: 21 }, (_, index) => ({
      title: `Agreed condition ${index + 1}`, required: true, verification_mode: "attestation",
    })) });
  const scope = { namespace_id: wos.namespace, outcome_id: outcome.id };
  const owner = { ...scope, kind: "outcome", id: outcome.id };
  const firstPage = await wos.api(`${outcomePath(scope)}/workspace-view`, { subject: { kind: "entity", ref: owner } });
  const omitted = outcome.criteria.items.find((criterion) => !firstPage.criteria.some((item) => item.definition.id === criterion.id));
  expect(omitted).toBeTruthy();
  let version = outcome.version;
  for (const criterion of outcome.criteria.items) {
    await wos.command("record_criterion_assessment", { owner, expected_version: version++,
      criterion_id: criterion.id, criterion_revision: criterion.criterion_revision,
      result: "met", rationale: "Explicitly reviewed this agreed condition" });
  }
  await wos.command("activate_outcome", { scope, expected_version: version });
  await wos.login(page);
  await openOutcome(page, outcome.title, { area: null });
  await inspect(page, outcome.title, "canvas");
  await openAction(page, "achieve_outcome");
  const reason = "Retain my explicit acceptance draft while I assess the changed condition.";
  await page.getByLabel("Reason", { exact: true }).fill(reason);
  const current = (await wos.api(outcomePath(scope))).value;
  const revised = await wos.command("revise_criterion", { owner, expected_version: current.version,
    criterion_id: omitted.id, title: omitted.title, description: "A deliberately revised condition",
    required: true, verification_mode: "attestation" });
  const rejection = page.waitForResponse((response) => response.url().endsWith("/commands/achieve_outcome"));
  await page.locator("#submit-command").click();
  expect((await (await rejection).json()).error.code).toBe("precondition_failed");
  await page.locator("#conflict-review").getByRole("button", { name: "Assess criterion", exact: true }).click();
  await expect(page.locator("#command-select")).toHaveValue("record_criterion_assessment");
  await expect(page.locator("#fields").getByRole("button", { name: "Return to your draft", exact: true })).toBeVisible();
  await expect(page.locator("#fields")).toContainText(omitted.title);
  await page.getByLabel("Rationale", { exact: true }).fill("I independently reviewed the current revised condition.");
  const assessmentResponse = page.waitForResponse((response) => response.url().endsWith("/commands/record_criterion_assessment"));
  await page.locator("#submit-command").click();
  const response = await assessmentResponse;
  const result = await response.json();
  expect(response.ok(), JSON.stringify(result)).toBe(true);
  expect(result.value.criterion_id).toBe(omitted.id);
  expect(String(result.value.criterion_revision)).toBe(String(revised.criterion_revision));
  await expect(page.locator("#command-select")).toHaveValue("achieve_outcome");
  await expect(page.getByLabel("Reason", { exact: true })).toHaveValue(reason);
  expect((await wos.api(outcomePath(scope))).value.lifecycle).toBe("active");
});

test("a delayed preparation lookup cannot replace a newer form after the original closes", async ({ page, wos }) => {
  const { value: outcome, scope } = await wos.outcome("Keep the current form during delayed preparation");
  const owner = { ...scope, kind: "outcome", id: outcome.id };
  const criterion = await wos.command("add_criterion", { owner, expected_version: outcome.version,
    title: "The original activation condition", required: true, verification_mode: "attestation" });
  await wos.login(page);
  await openOutcome(page, outcome.title, { area: null });
  await inspect(page, outcome.title, "canvas");
  await openAction(page, "activate_outcome");
  await page.getByLabel("Reason", { exact: true }).fill("An explicit request to activate the agreed plan.");
  const current = (await wos.api(outcomePath(scope))).value;
  await wos.command("retire_criterion", { owner, expected_version: current.version, criterion_id: criterion.id });
  const rejection = page.waitForResponse((response) => response.url().endsWith("/commands/activate_outcome"));
  await page.locator("#submit-command").click();
  expect((await (await rejection).json()).error.code).toBe("precondition_failed");
  let begin, release;
  const started = new Promise((resolve) => { begin = resolve; });
  const gate = new Promise((resolve) => { release = resolve; });
  let first = true, stalled;
  await page.route("**/workspace-view", async (route) => {
    if (!first) { await route.continue(); return; }
    first = false;
    stalled = route.request();
    begin();
    await gate;
    await route.continue();
  });
  try {
    await page.locator("#conflict-review").getByRole("button", { name: "Add success criterion", exact: true }).click();
    await started;
    await page.locator("#close-command").click();
    await expect(page.locator("#command-form")).not.toBeVisible();
    await inspect(page, outcome.title, "canvas");
    await openAction(page, "create_work_item");
    const draft = "Keep this newer Task draft";
    await page.getByLabel("Task title", { exact: true }).fill(draft);
    release();
    await (await stalled.response()).finished();
    await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    await expect(page.locator("#command-select")).toHaveValue("create_work_item");
    await expect(page.getByLabel("Task title", { exact: true })).toHaveValue(draft);
    await expect(page.locator("#fields")).not.toContainText("Back to draft");
  } finally { release(); }
});

test("privileged v1 revocation hydrates the actual contract without requiring the technical record", async ({ page, wos }) => {
  const { value: outcome, scope } = await wos.outcome("Explicit current contract revocation");
  await wos.command("add_criterion", { owner: { ...scope, kind: "outcome", id: outcome.id },
    expected_version: outcome.version, title: "An agreed condition", required: true, verification_mode: "attestation" });
  const current = (await wos.api(outcomePath(scope))).value;
  await wos.command("activate_outcome", { scope, expected_version: current.version });
  await wos.command("set_namespace_work_protocol", { scope, expected_protocol_version: 1,
    phase: "draining", reason: "Stop legacy acquisition for the explicit migration" });
  await wos.command("set_namespace_work_protocol", { scope, expected_protocol_version: 2,
    phase: "contracts_v1", writers_drained: true, reason: "The external writers have been drained" });
  const task = await wos.command("create_work_item", { scope, title: "An externally reserved contract Task",
    priority: "normal", lifecycle: "todo", execution_spec: { instructions: ["Stay within the acquired definition"] } });
  const acquired = await wos.command("acquire_work_contract", { scope, work_item_id: task.id,
    expected_work_item_version: task.version, ttl_seconds: 900 });
  await wos.login(page);
  await openOutcome(page, outcome.title, { area: null });
  await inspect(page, task.title, "canvas");
  await openAction(page, "revoke_work_contract");
  await page.getByLabel("Reason", { exact: true }).fill("Explicitly end this reserved authority before changing the definition.");
  const response = page.waitForResponse((response) => response.url().endsWith("/commands/revoke_work_contract"));
  await page.locator("#submit-command").click();
  const result = await response;
  const payload = result.request().postDataJSON().command;
  const record = await result.json();
  expect(result.ok(), JSON.stringify(record)).toBe(true);
  expect(payload.contract_id).toBe(acquired.contract.id);
  expect(String(payload.expected_contract_version)).toBe(String(acquired.contract.version));
  expect(record.value.contract.status).toBe("revoked");
  expect(record.value.work_item.current_contract_id).toBeUndefined();
  // Revocation ends authority while preserving recorded execution progress.
  expect(record.value.work_item.lifecycle).toBe(acquired.work_item.lifecycle);
  expect(record.value.work_item.lifecycle).toBe("in_progress");
  expect(record.value.work_item.conclusion).toBeUndefined();
});
