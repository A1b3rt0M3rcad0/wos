import { test, expect, openOutcome } from "./ux-fixture.mjs";
import { clickRecordAction } from "./workspace-action-fixture.mjs";

test("a subtree Issue previews actual affected work and atomically rejects a stale blocking scope", async ({ page, wos }) => {
  await page.setViewportSize({ width: 1366, height: 768 });
  const { scope, value: outcome } = await wos.outcome("Review the real scope before recording an impediment");
  const base = `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`;
  await wos.command("add_criterion", { owner: { ...scope, kind: "outcome", id: outcome.id },
    expected_version: outcome.version, title: "The actual result is reviewed", required: true, verification_mode: "attestation" });
  const current = (await wos.api(base)).value;
  await wos.command("activate_outcome", { scope, expected_version: current.version });
  const objective = await wos.command("create_objective", { scope, title: "Deliver a reviewable installation guide",
    priority: "normal", required_for_outcome: false });
  const child = await wos.command("create_objective", { scope, title: "Verify the recovery instructions",
    priority: "normal", required_for_outcome: false, parent_objective_id: objective.id });
  const task = await wos.command("create_work_item", { scope, title: "Run the recovery procedure on a clean environment",
    priority: "normal", lifecycle: "backlog", objective_id: child.id });
  const owner = { ...scope, kind: "objective", id: objective.id };
  const queries = [], attempts = [], errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("request", (request) => {
    if (request.method() !== "POST") return;
    if (request.url().endsWith("/commands/report_issue_with_blocker")) attempts.push(request.postDataJSON().command);
    if (request.url().endsWith("/workspace-view") && request.postDataJSON().blocker_preview)
      queries.push(request.postDataJSON());
  });
  await wos.login(page);
  await openOutcome(page, outcome.title, { area: null });
  await page.getByRole("combobox", { name: "Search Outcome Board", exact: true }).fill(objective.title);
  await page.getByRole("option").filter({ hasText: objective.title }).first().click();
  await expect(page.locator("#canvas-details #detail-title")).toHaveText(objective.title);
  await clickRecordAction(page, "Report issue");
  const title = "Recovery environment cannot currently be reached";
  const description = "The independent reviewer cannot access the agreed recovery environment.";
  const impactDescription = "The installation and recovery work must wait for verified access.";
  await page.getByLabel("Issue title", { exact: true }).fill(title);
  await page.getByLabel("Description", { exact: true }).fill(description);
  await page.getByLabel("Prevents continuing?", { exact: true }).check();
  await page.getByLabel("Blocking impact", { exact: true }).fill(impactDescription);
  await page.getByLabel("Propagation", { exact: true }).selectOption("subtree");
  const preview = page.locator(".workspace-blocking-preview");
  await expect(preview).toContainText("Preview the current blocking scope and confirm its impact before saving.");
  const firstResponse = page.waitForResponse((response) => response.request().method() === "POST"
    && response.url().endsWith("/workspace-view") && response.request().postDataJSON().blocker_preview);
  await preview.getByRole("button", { name: "Preview blocking impact", exact: true }).click();
  const firstRead = await firstResponse, firstView = await firstRead.json();
  expect(firstRead.ok()).toBe(true);
  expect(queries.at(-1)).toMatchObject({ subject: { kind: "entity", ref: owner }, blocker_preview: { propagation: "subtree" } });
  expect(firstView.blocker_preview.affected_objectives).toBeGreaterThanOrEqual(2);
  expect(firstView.blocker_preview.affected_tasks).toBe(1);
  expect(firstView.coverage.blocker_preview.status).toBe("complete");
  await expect(preview.getByRole("heading", { name: "Open work affected by this Blocker", exact: true })).toBeVisible();
  for (const record of [objective, child, task]) await expect(preview).toContainText(record.title);
  await expect(preview).toContainText("It does not change ownership, complete work, or assess criteria.");
  const acknowledgement = page.getByLabel("I reviewed the open work affected by this Blocker", { exact: true });
  await expect(acknowledgement).not.toBeChecked();
  await acknowledgement.check();

  const lateTask = await wos.command("create_work_item", { scope, title: "Recovery work added after the scope review",
    priority: "normal", lifecycle: "backlog", objective_id: child.id });
  const rejected = page.waitForResponse((response) => response.request().method() === "POST"
    && response.url().endsWith("/commands/report_issue_with_blocker"));
  await page.locator("#submit-command").click();
  const rejection = await rejected, error = await rejection.json();
  expect(rejection.ok()).toBe(false);
  expect(error.error.code).toBe("precondition_failed");
  expect(error.error.details).toMatchObject({ reason_code: "blocker_impact_changed", subject_ref: owner,
    field_paths: ["observed_outcome_revision"] });
  expect(attempts[0].observed_outcome_revision).toBe(firstView.outcome_revision);
  expect(typeof attempts[0].observed_outcome_revision).toBe("string");
  const rejectedState = await wos.api(`${base}/state`);
  expect(rejectedState.issues || []).toHaveLength(0);
  expect(rejectedState.blockers || []).toHaveLength(0);
  await expect(page.getByLabel("Issue title", { exact: true })).toHaveValue(title);
  await expect(page.getByLabel("Description", { exact: true })).toHaveValue(description);
  await expect(page.getByLabel("Blocking impact", { exact: true })).toHaveValue(impactDescription);

  const nextResponse = page.waitForResponse((response) => response.request().method() === "POST"
    && response.url().endsWith("/workspace-view") && response.request().postDataJSON().blocker_preview);
  await preview.getByRole("button", { name: "Preview blocking impact", exact: true }).click();
  const nextView = await (await nextResponse).json();
  expect(nextView.outcome_revision).not.toBe(firstView.outcome_revision);
  expect(nextView.blocker_preview.affected_tasks).toBe(2);
  await expect(preview).toContainText(lateTask.title);
  await expect(acknowledgement).not.toBeChecked();
  await expect(page.getByLabel("Issue title", { exact: true })).toHaveValue(title);
  expect(attempts).toHaveLength(1);
  await acknowledgement.check();
  const accepted = page.waitForResponse((response) => response.request().method() === "POST"
    && response.url().endsWith("/commands/report_issue_with_blocker"));
  await page.locator("#submit-command").click();
  const acceptance = await accepted, result = await acceptance.json();
  expect(acceptance.ok(), JSON.stringify(result)).toBe(true);
  await expect(page.locator("#command-form")).not.toBeVisible();
  expect(attempts).toHaveLength(2);
  expect(attempts[1].observed_outcome_revision).toBe(nextView.outcome_revision);
  expect(result.value.issue).toMatchObject({ title, lifecycle: "open", affected_refs: [owner] });
  expect(result.value.blocker).toMatchObject({ blocked_ref: owner, propagation: "subtree", lifecycle: "active",
    cause_ref: { ...scope, kind: "issue", id: result.value.issue.id } });
  const state = await wos.api(`${base}/state`);
  expect(state.issues).toHaveLength(1);
  expect(state.blockers).toHaveLength(1);
  for (const original of [task, lateTask]) {
    const unchanged = (await wos.api(`${base}/work-items/${original.id}`)).value;
    expect(unchanged.lifecycle).toBe("backlog");
    expect(unchanged.version).toBe(original.version);
    expect(unchanged.criteria).toEqual(original.criteria);
    expect(unchanged.current_contract_id).toBeUndefined();
    expect(unchanged.current_lease).toBeUndefined();
  }
  expect(errors).toEqual([]);
});
