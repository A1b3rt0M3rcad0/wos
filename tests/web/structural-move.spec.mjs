import { test, expect, openOutcome, pick } from "./ux-fixture.mjs";
import { clickRecordAction } from "./workspace-action-fixture.mjs";

test("a Task move previews real inherited blocking, rejects a stale revision and preserves its recorded identity and proof", async ({ page, wos }) => {
  await page.setViewportSize({ width: 1366, height: 768 });
  const { scope, value: outcome } = await wos.outcome("Move work only after reviewing the actual operational impact");
  const base = `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`;
  await wos.command("add_criterion", { owner: { ...scope, kind: "outcome", id: outcome.id },
    expected_version: outcome.version, title: "The delivered result is independently verified", required: true, verification_mode: "attestation" });
  await wos.command("activate_outcome", { scope, expected_version: (await wos.api(base)).value.version });
  const objectives = [];
  for (const title of ["Current delivery branch", "Destination with recorded access restrictions"])
    objectives.push(await wos.command("create_objective", { scope, title, priority: "normal", required_for_outcome: false }));
  const [previous, destination] = objectives;
  let task = await wos.command("create_work_item", { scope, title: "Review the installation transcript",
    priority: "normal", lifecycle: "todo", objective_id: previous.id,
    execution_spec: { instructions: ["Review the real installation transcript"], constraints: ["Do not claim missing verification"], deliverables: ["A recorded assessment"] },
    initial_criteria: [{ title: "The transcript records the actual commands", required: true, verification_mode: "attestation" }] });
  const owner = { ...scope, kind: "work_item", id: task.id };
  const criterion = task.criteria.items[0];
  await wos.command("attest_criterion", { owner, expected_version: task.version, criterion_id: criterion.id,
    criterion_revision: criterion.criterion_revision, result: "not_met", rationale: "The first transcript is missing the final command." });
  task = (await wos.api(`${base}/work-items/${task.id}`)).value;
  const issue = await wos.command("create_issue", { scope, title: "Destination environment access is missing", severity: "major",
    description: "Access is a separate recorded impediment rather than a Task lifecycle change.", affected_refs: [{ ...scope, kind: "objective", id: destination.id }] });
  const createBlocker = (description) => wos.command("create_blocker", { scope,
    blocked_ref: { ...scope, kind: "objective", id: destination.id }, cause_ref: { ...scope, kind: "issue", id: issue.id },
    propagation: "subtree", description });
  const blocker = await createBlocker("Verified destination access is still pending");
  const attempts = [], errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().endsWith("/commands/set_work_item_objective")) attempts.push(request.postDataJSON().command);
  });
  await wos.login(page);
  await openOutcome(page, outcome.title, { area: null });
  await page.getByRole("combobox", { name: "Search Outcome Board", exact: true }).fill(task.title);
  await page.getByRole("option").filter({ hasText: task.title }).first().click();
  await expect(page.locator("#canvas-details #detail-title")).toHaveText(task.title);
  await clickRecordAction(page, "Move Task");
  await page.getByRole("button", { name: "Change placement", exact: true }).click();
  await pick(page, "New placement", destination.title);
  const review = page.getByRole("button", { name: "Preview move", exact: true });
  const acknowledgement = page.getByLabel("I reviewed the new placement and its blocking impacts", { exact: true });
  await expect(acknowledgement).toBeDisabled();
  async function previewMove() {
    const response = page.waitForResponse((response) => response.request().method() === "POST"
      && response.url().endsWith("/workspace-view") && response.request().postDataJSON().placement_preview);
    await review.click();
    const result = await response;
    expect(result.ok()).toBe(true);
    return result.json();
  }
  const initial = await previewMove();
  const preview = page.locator(".workspace-move-preview");
  await expect(preview.getByRole("heading", { name: "Observed movement impact", exact: true })).toBeVisible();
  await expect(preview).toContainText(previous.title);
  await expect(preview).toContainText(destination.title);
  await expect(preview).toContainText(blocker.description);
  await expect(preview).toContainText("Inherited blocker");
  expect(initial.placement_preview).toMatchObject({ old_parent: { ref: { id: previous.id } },
    new_parent: { ref: { id: destination.id } }, availability: "available", affected_tasks: 1,
    readiness_before: { is_blocked: false }, readiness_after: { is_blocked: true } });
  const change = initial.placement_preview.blocking_changes.find((item) => item.ref.id === task.id);
  expect(change.before.blockers).toHaveLength(0);
  expect(change.after.blockers).toContainEqual(expect.objectContaining({ ref: expect.objectContaining({ id: blocker.id }), inherited: true }));
  await expect(acknowledgement).toBeEnabled();
  await expect(acknowledgement).not.toBeChecked();
  await acknowledgement.check();

  // The selected Task itself has not changed, but another participant adds a
  // new inherited impediment after the old placement preview was confirmed.
  const lateBlocker = await createBlocker("A second destination restriction was recorded after review");
  const rejected = page.waitForResponse((response) => response.request().method() === "POST"
    && response.url().endsWith("/commands/set_work_item_objective"));
  await page.locator("#submit-command").click();
  const rejection = await rejected, error = await rejection.json();
  expect(rejection.ok()).toBe(false);
  expect(error.error.code).toBe("precondition_failed");
  expect(error.error.details.reason_code).toBe("structure_preview_changed");
  expect(attempts[0]).toMatchObject({ objective_id: destination.id, expected_outcome_revision: initial.outcome_revision });
  expect(typeof attempts[0].expected_outcome_revision).toBe("string");
  const unmoved = (await wos.api(`${base}/work-items/${task.id}`)).value;
  expect(unmoved.objective_id).toBe(previous.id);
  expect(unmoved.version).toBe(task.version);
  await expect(page.getByRole("combobox", { name: "New placement", exact: true })).toHaveValue("");
  await expect(page.locator(".workspace-structural-editor")).toContainText(destination.title);
  const refreshed = await previewMove();
  expect(refreshed.outcome_revision).not.toBe(initial.outcome_revision);
  await expect(preview).toContainText(lateBlocker.description);
  await expect(acknowledgement).not.toBeChecked();
  expect(attempts).toHaveLength(1);
  await acknowledgement.check();
  const accepted = page.waitForResponse((response) => response.request().method() === "POST"
    && response.url().endsWith("/commands/set_work_item_objective"));
  await page.locator("#submit-command").click();
  const acceptance = await accepted, result = await acceptance.json();
  expect(acceptance.ok(), JSON.stringify(result)).toBe(true);
  await expect(page.locator("#command-form")).not.toBeVisible();
  expect(attempts).toHaveLength(2);
  expect(attempts[1].expected_outcome_revision).toBe(refreshed.outcome_revision);
  const persisted = (await wos.api(`${base}/work-items/${task.id}`)).value;
  expect(persisted.id).toBe(task.id);
  expect(persisted.objective_id).toBe(destination.id);
  expect(persisted.lifecycle).toBe(task.lifecycle);
  expect(persisted.criteria).toEqual(task.criteria);
  expect(persisted.execution_spec).toEqual(task.execution_spec);
  expect(persisted.current_lease).toBeUndefined();
  expect(persisted.current_contract_id).toBeUndefined();
  const finalView = await wos.api(`${base}/workspace-view`, { subject: { kind: "entity", ref: owner },
    placement_preview: { parent_id: previous.id } });
  expect(finalView.placement.parent.ref.id).toBe(destination.id);
  expect(finalView.placement_preview.readiness_before).toMatchObject({ is_blocked: true });
  expect(finalView.placement_preview.readiness_before.display_state).toBe(refreshed.placement_preview.readiness_after.display_state);
  expect(finalView.placement_preview.readiness_after).toMatchObject({ is_blocked: false });
  expect(errors).toEqual([]);
});
