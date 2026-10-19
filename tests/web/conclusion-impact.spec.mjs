import { test, expect, openOutcome } from "./ux-fixture.mjs";
import { revealMenuAction } from "./workspace-action-fixture.mjs";

const confirmation = "I understand that this open work is not completed by this conclusion";
const panel = (page) => page.locator("#canvas-details");

async function selectRecord(page, title) {
  const search = page.getByRole("combobox", { name: "Search Outcome Board", exact: true });
  await search.fill(title);
  await page.getByRole("option").filter({ hasText: title }).first().click();
  await expect(panel(page).locator("#detail-title")).toHaveText(title);
  await expect(panel(page)).toHaveAttribute("data-surface", "detail");
}

async function openAction(page, name) {
  const direct = panel(page).locator(`[data-action-id="${name}"]`).filter({ visible: true });
  if (await direct.count()) await direct.first().click();
  else {
    await panel(page).locator("#entity-actions").click();
    const target = page.locator(`#action-options [data-action-id="${name}"]`);
    await revealMenuAction(target);
    await target.click();
  }
  await expect(page.locator("#command-form")).toBeVisible();
  await expect(page.locator("#command-select")).not.toBeVisible();
}

for (const viewport of [{ width: 2560, height: 1440 }, { width: 390, height: 844 }]) {
  test(`concluding parents reviews actual open work, rejects stale impact and preserves children at ${viewport.width}px`, async ({ page, wos }) => {
    test.setTimeout(150000);
    await page.setViewportSize(viewport);
    const { scope, value: outcome } = await wos.outcome("Accept only the reviewed parent result");
    const base = `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`;
    const ref = (kind, record) => ({ ...scope, kind, id: record.id });
    const read = async (kind, id) => (await wos.api(kind === "outcome" ? base : `${base}/${kind === "objective" ? "objectives" : "work-items"}/${id}`)).value;
    const viewOf = async (owner) => wos.api(`${base}/workspace-view`, { subject: { kind: "entity", ref: owner } });
    async function meetCriterion(kind, record) {
      const owner = ref(kind, record);
      const criterion = await wos.command("add_criterion", { owner, expected_version: record.version,
        title: `${kind === "outcome" ? "Outcome" : "Objective"} result was independently checked`,
        required: true, verification_mode: "attestation" });
      let current = await read(kind, record.id);
      await wos.command("attest_criterion", { owner, expected_version: current.version,
        criterion_id: criterion.id, criterion_revision: criterion.criterion_revision,
        result: "met", rationale: "An authorized participant reviewed the agreed condition before requesting acceptance." });
      return read(kind, record.id);
    }
    let reviewedOutcome = await meetCriterion("outcome", outcome);
    await wos.command("activate_outcome", { scope, expected_version: reviewedOutcome.version });
    let objective = await wos.command("create_objective", { scope, title: "Accept the reviewed documentation result",
      priority: "normal", required_for_outcome: false });
    objective = await meetCriterion("objective", objective);
    objective = await wos.command("start_objective", { scope, objective_id: objective.id, expected_version: objective.version });
    const child = await wos.command("create_objective", { scope, title: "Optional follow-up remains open",
      priority: "normal", parent_objective_id: objective.id, required_for_outcome: false });
    const originalTask = await wos.command("create_work_item", { scope, title: "Unfinished delivery keeps its own workflow",
      priority: "normal", lifecycle: "backlog", objective_id: objective.id,
      initial_criteria: [{ title: "The child delivery has its own unassessed criterion", required: true, verification_mode: "evidence_review" }] });
    const unchangedChildren = [{ kind: "objective", record: child }, { kind: "work_item", record: originalTask }];
    const errors = [], attempts = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("request", (request) => {
      if (request.method() === "POST" && /\/commands\/achieve_(outcome|objective)$/.test(request.url()))
        attempts.push({ name: request.url().split("/").at(-1), command: request.postDataJSON().command });
    });
    await wos.login(page);
    await openOutcome(page, outcome.title, { area: null });

    // Both conclusion types use a genuine guarded read. A parent conclusion
    // must not manufacture completion, assessment, cancellation or authority
    // on a descendant, including descendants of an already achieved Objective.
    for (const [kind, record] of [["objective", objective], ["outcome", outcome]]) {
      const owner = ref(kind, record), commandName = `achieve_${kind}`;
      await selectRecord(page, record.title);
      const initialView = await viewOf(owner);
      expect(initialView.conclusion_impact.open_objectives).toBeGreaterThan(0);
      expect(initialView.conclusion_impact.open_tasks).toBeGreaterThan(0);
      expect(initialView.coverage.conclusion_impact.status).toBe("complete");
      await openAction(page, commandName);
      const impact = page.locator(".workspace-conclusion-impact");
      await expect(impact.getByRole("heading", { name: "Review conclusion impact", exact: true })).toBeVisible();
      await expect(impact).toContainText(`${initialView.conclusion_impact.open_objectives} open Objectives · ${initialView.conclusion_impact.open_tasks} open Tasks`);
      for (const item of initialView.conclusion_impact.open_items) await expect(impact).toContainText(item.title);
      await expect(impact).toContainText("Open child records keep their current lifecycle");
      await expect(impact).toContainText("Criteria, Evidence and assessments on child records remain unchanged.");
      const acknowledgement = page.getByLabel(confirmation, { exact: true });
      await expect(acknowledgement).not.toBeChecked();
      const reason = `I reviewed the ${kind} criteria and the explicitly listed unfinished work.`;
      await page.getByLabel("Reason", { exact: true }).fill(reason);
      await page.locator("#submit-command").click();
      await expect(page.locator("#command-error")).toHaveText("Review the current impact before confirming this conclusion.");
      expect(attempts.filter((attempt) => attempt.name === commandName)).toHaveLength(0);
      await acknowledgement.check();

      // A different participant adds a real Task after the person acknowledged
      // the displayed snapshot. The owner's version alone cannot catch this.
      const lateTask = await wos.command("create_work_item", { scope,
        title: `${kind === "objective" ? "Objective" : "Outcome"} work added after impact review`,
        lifecycle: "backlog", priority: "normal", ...(kind === "objective" ? { objective_id: objective.id } : {}) });
      unchangedChildren.push({ kind: "work_item", record: lateTask });
      const changedView = await viewOf(owner);
      expect(changedView.entity_version).toBe(initialView.entity_version);
      expect(changedView.outcome_revision).not.toBe(initialView.outcome_revision);
      const rejected = page.waitForResponse((response) => response.request().method() === "POST" && response.url().endsWith(`/commands/${commandName}`));
      await page.locator("#submit-command").click();
      const rejection = await rejected, error = await rejection.json();
      expect(rejection.ok()).toBe(false);
      expect(error.error.code).toBe("precondition_failed");
      expect(error.error.details).toMatchObject({ reason_code: "conclusion_impact_changed",
        subject_ref: owner, field_paths: ["observed_outcome_revision"] });
      expect(error.error.details.remedies).toContainEqual({ action_id: "review_conclusion_impact", subject_ref: owner });
      expect(attempts.at(-1).command.observed_outcome_revision).toBe(initialView.outcome_revision);
      expect(typeof attempts.at(-1).command.observed_outcome_revision).toBe("string");
      await expect(page.getByLabel("Reason", { exact: true })).toHaveValue(reason);
      const unconcluded = await read(kind, record.id);
      expect(unconcluded.lifecycle).toBe(kind === "objective" ? "in_progress" : "active");
      expect(unconcluded.conclusion).toBeUndefined();
      await page.locator("#conflict-review").getByRole("button", { name: "Review current impact", exact: true }).click();
      await expect(impact).toContainText(lateTask.title);
      await expect(page.getByLabel("Reason", { exact: true })).toHaveValue(reason);
      await expect(acknowledgement).not.toBeChecked();
      await expect(page.locator("#conflict-review")).toBeEmpty();
      await page.locator("#submit-command").click();
      await expect(page.locator("#command-error")).toHaveText("Review the current impact before confirming this conclusion.");
      expect(attempts.filter((attempt) => attempt.name === commandName)).toHaveLength(1);

      // Refreshing is a read; only a new explicit acknowledgement and submit
      // can accept this parent's result against the fresh shared revision.
      await acknowledgement.check();
      const accepted = page.waitForResponse((response) => response.request().method() === "POST" && response.url().endsWith(`/commands/${commandName}`));
      await page.locator("#submit-command").click();
      const acceptance = await accepted;
      const result = await acceptance.json();
      expect(acceptance.ok(), JSON.stringify(result)).toBe(true);
      await expect(page.locator("#command-form")).not.toBeVisible();
      expect(attempts.at(-1).command.observed_outcome_revision).toBe(changedView.outcome_revision);
      const concluded = await read(kind, record.id);
      expect(concluded.lifecycle).toBe("achieved");
      expect(concluded.conclusion.reason).toBe(reason);
      for (const descendant of unchangedChildren) {
        const current = await read(descendant.kind, descendant.record.id);
        expect(current.lifecycle).toBe(descendant.record.lifecycle);
        expect(current.version).toBe(descendant.record.version);
        expect(current.criteria).toEqual(descendant.record.criteria);
        expect(current.conclusion).toBeUndefined();
        expect(current.current_lease).toBeUndefined();
        expect(current.current_contract_id).toBeUndefined();
      }
    }
    expect(attempts.map((attempt) => attempt.name)).toEqual([
      "achieve_objective", "achieve_objective", "achieve_outcome", "achieve_outcome",
    ]);
    expect(errors).toEqual([]);
  });
}

test("a late failed conclusion-impact review cannot report an obsolete error or replace another Task's inspector", async ({ page, wos }) => {
  const { scope, value: outcome } = await wos.outcome("Accept the parent only after its current impact was reviewed");
  const base = `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`;
  const owner = { ...scope, kind: "outcome", id: outcome.id };
  const criterion = await wos.command("add_criterion", {
    owner, expected_version: outcome.version,
    title: "The agreed parent result was independently reviewed",
    required: true, verification_mode: "attestation",
  });
  let current = (await wos.api(base)).value;
  await wos.command("attest_criterion", {
    owner, expected_version: current.version,
    criterion_id: criterion.id, criterion_revision: criterion.criterion_revision,
    result: "met", rationale: "An authorized participant checked the explicit parent criterion.",
  });
  current = (await wos.api(base)).value;
  await wos.command("activate_outcome", { scope, expected_version: current.version });
  const task = await wos.command("create_work_item", {
    scope, title: "Inspect this separate Task after closing the old conclusion",
    lifecycle: "backlog", priority: "normal",
  });
  const attempts = [], errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/commands/"))
      attempts.push(request.url().split("/").at(-1));
  });
  await wos.login(page);
  await openOutcome(page, outcome.title, { area: null });
  await selectRecord(page, outcome.title);
  await openAction(page, "achieve_outcome");
  await page.getByLabel("Reason", { exact: true }).fill("Reviewing this parent must not replace the next Task context.");
  await page.getByLabel(confirmation, { exact: true }).check();

  await wos.command("create_work_item", {
    scope, title: "A different participant added this work after acknowledgement",
    lifecycle: "backlog", priority: "normal",
  });
  const rejected = page.waitForResponse((response) => response.request().method() === "POST"
    && response.url().endsWith("/commands/achieve_outcome"));
  await page.locator("#submit-command").click();
  const rejection = await rejected, error = await rejection.json();
  expect(rejection.ok()).toBe(false);
  expect(error.error.details.reason_code).toBe("conclusion_impact_changed");
  await expect(page.locator("#conflict-review").getByRole("button", { name: "Review current impact", exact: true })).toBeVisible();

  const obsoleteMessage = "Obsolete conclusion review failed after the person selected another Task";
  let release, seen, finished, rejectRead;
  let started = false;
  const gate = new Promise((resolve) => { release = resolve; });
  const arrived = new Promise((resolve, reject) => { seen = resolve; rejectRead = reject; });
  const done = new Promise((resolve) => { finished = resolve; });
  const routePattern = "**/workspace-view";
  const handler = async (route) => {
    if (started || route.request().method() !== "POST"
      || route.request().postDataJSON()?.subject?.ref?.id !== outcome.id)
      return route.continue();
    started = true;
    try {
      // Capture a real authorized current read, then simulate a delayed
      // transport failure without changing any persisted state or authority.
      const response = await route.fetch();
      seen(response.status());
      await gate;
      await route.fulfill({ status: 500, contentType: "application/json",
        body: JSON.stringify({ error: { code: "internal", message: obsoleteMessage } }) });
    } catch (failure) {
      rejectRead(failure);
      throw failure;
    } finally { finished(); }
  };
  await page.route(routePattern, handler);
  try {
    await page.locator("#conflict-review").getByRole("button", { name: "Review current impact", exact: true }).click();
    expect(await arrived).toBe(200);
    await panel(page).locator("#close-command").click();
    await expect(page.locator("#command-form")).toBeHidden();
    await selectRecord(page, task.title);
    await expect(panel(page).getByRole("heading", { name: "Operational readiness", exact: true })).toBeVisible();
    await expect(panel(page).getByRole("heading", {
      name: "An active blocking impact must be reviewed before work can continue.", exact: true,
    })).toHaveCount(0);
    await expect(page).toHaveURL(new RegExp(`item=work_item(?:%2F|/)${task.id}`));
    // The optional graph enrichment belongs to this new Task selection. Let
    // its legitimate stale-map notice settle before testing the unrelated,
    // obsolete conclusion read; the old read must preserve that newer notice.
    await expect(page.locator("#notice")).toContainText("The Outcome changed while loading the map. Refresh to view the current state.");
    const selectedURL = page.url();
    const retainedFocus = panel(page).locator("#entity-actions");
    await retainedFocus.focus();
    await expect(retainedFocus).toBeFocused();
    const noticeBefore = await page.locator("#notice").evaluate((node) => ({
      text: node.textContent, className: node.className, hidden: node.hidden,
    }));
    release();
    await done;
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveURL(selectedURL);
    await expect(panel(page)).toHaveAttribute("data-surface", "detail");
    await expect(panel(page).locator("#detail-title")).toHaveText(task.title);
    await expect(page.locator("#command-form")).toBeHidden();
    await expect(page.locator("#command-dialog")).toBeHidden();
    await expect(page.locator("#notice")).not.toContainText(obsoleteMessage);
    expect(await page.locator("#notice").evaluate((node) => ({
      text: node.textContent, className: node.className, hidden: node.hidden,
    }))).toEqual(noticeBefore);
    await expect(retainedFocus).toBeFocused();
    expect(attempts).toEqual(["achieve_outcome"]);
    expect(errors).toEqual([]);
    const persistedOutcome = (await wos.api(base)).value;
    const persistedTask = (await wos.api(`${base}/work-items/${task.id}`)).value;
    expect(persistedOutcome.lifecycle).toBe("active");
    expect(persistedOutcome.conclusion).toBeUndefined();
    expect(persistedTask.version).toBe(task.version);
    expect(persistedTask.lifecycle).toBe("backlog");
  } finally {
    release();
    if (started) await done;
    await page.unroute(routePattern, handler);
  }
});
