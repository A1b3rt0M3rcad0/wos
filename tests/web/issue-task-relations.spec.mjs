import { test, expect, openOutcome, pick, save } from "./ux-fixture.mjs";
import { seedCanvas } from "./canvas-fixture.mjs";
import { expandWorkspaceDetails } from "./workspace-action-fixture.mjs";

const board = (page) => page.locator("#operational-canvas");
const inspector = (page) => page.locator("#canvas-details");
const relationships = (page) => inspector(page).locator('[data-workspace-facet="relationships"]');
const relationRow = (page, id) => relationships(page).locator(`[data-issue-task-relation="${id}"]`);
const reference = (host, kind, id) => host.locator(`button[data-reference-kind="${kind}"][data-reference-id="${id}"]`);

async function select(page, record) {
  const search = board(page).getByRole("combobox", { name: "Search Outcome Board", exact: true });
  await search.fill(record.title);
  await board(page).getByRole("option").filter({ hasText: record.title }).first().click();
  await expect(inspector(page).locator("#detail-title")).toHaveText(record.title);
  await expandWorkspaceDetails(inspector(page), { history: false });
}

async function openLinkForm(page, label) {
  await relationships(page).getByRole("button", { name: label, exact: true }).click();
  await expect(page.locator("#command-form")).toBeVisible();
  await expect(page.locator("#command-dialog")).not.toBeVisible();
}

async function loadRelation(wos, scope, issue, task, lifecycle) {
  const page = await wos.api(`/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}/relations`);
  return page.items.find((item) => item.relation_type === "relates_to"
    && (!lifecycle || item.lifecycle === lifecycle)
    && ((item.source_ref.kind === "issue" && item.source_ref.id === issue.id
      && item.target_ref.kind === "work_item" && item.target_ref.id === task.id)
      || (item.source_ref.kind === "work_item" && item.source_ref.id === task.id
        && item.target_ref.kind === "issue" && item.target_ref.id === issue.id)));
}

test("Issue and Task context create and remove the canonical corrective-work link from either side", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  await wos.login(page);
  await openOutcome(page, data.outcome.title, { area: null });

  await select(page, data.issue);
  await expect(relationships(page)).toContainText("This records related corrective work only");
  await openLinkForm(page, "Link corrective Task");
  await expect(page.locator("#command-form .flow-help")).toContainText("does not move or reassign either item");
  await pick(page, "Corrective Task", data.tasks.ready.title);
  await page.locator("#submit-command").click();
  await expect(page.locator("#command-form")).not.toBeVisible({ timeout: 10000 });
  await expect(page.locator("#notice")).not.toHaveClass(/error/);

  let relation = await loadRelation(wos, data.scope, data.issue, data.tasks.ready, "active");
  expect(relation).toBeTruthy();
  expect(relation.source_ref).toMatchObject({ kind: "issue", id: data.issue.id });
  expect(relation.target_ref).toMatchObject({ kind: "work_item", id: data.tasks.ready.id });
  await expect(relationRow(page, relation.id)).toContainText("Corrective Task");
  await reference(relationRow(page, relation.id), "work_item", data.tasks.ready.id).click();
  await expect(inspector(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);
  await expandWorkspaceDetails(inspector(page), { history: false });
  await expect(relationRow(page, relation.id)).toContainText("Related Issue");
  const firstIssueLink = relationRow(page, relation.id).getByRole("button", { name: "Remove link", exact: true });
  await firstIssueLink.click();
  await expect(page.locator("#command-form")).toBeVisible();
  await expect(page.locator("#command-form .flow-help")).toContainText("does not change ownership");
  await page.getByLabel("Reason", { exact: true }).fill("The Issue needs a replacement corrective task");
  await save(page);
  await expect(relationRow(page, relation.id)).toHaveCount(0);
  expect((await loadRelation(wos, data.scope, data.issue, data.tasks.ready, "removed"))?.lifecycle).toBe("removed");

  await openLinkForm(page, "Link Issue");
  await pick(page, "Related Issue", data.issue.title);
  await save(page);
  relation = await loadRelation(wos, data.scope, data.issue, data.tasks.ready, "active");
  expect(relation?.lifecycle).toBe("active");
  await reference(relationRow(page, relation.id), "issue", data.issue.id).click();
  await expect(inspector(page).locator("#detail-title")).toHaveText(data.issue.title);
  await expandWorkspaceDetails(inspector(page), { history: false });
  await expect(relationRow(page, relation.id)).toContainText("Corrective Task");
  await relationRow(page, relation.id).getByRole("button", { name: "Remove link", exact: true }).click();
  await page.getByLabel("Reason", { exact: true }).fill("The corrective work is no longer needed");
  await save(page);
  expect((await loadRelation(wos, data.scope, data.issue, data.tasks.ready, "removed"))?.lifecycle).toBe("removed");

  const issue = (await wos.api(`/namespaces/${data.scope.namespace_id}/outcomes/${data.scope.outcome_id}/issues/${data.issue.id}`)).value;
  const task = (await wos.api(`/namespaces/${data.scope.namespace_id}/outcomes/${data.scope.outcome_id}/work-items/${data.tasks.ready.id}`)).value;
  expect(issue.objective_id).toBe(data.objective.id);
  expect(task.objective_id).toBe(data.child.id);
  expect((await wos.api(`/namespaces/${data.scope.namespace_id}/outcomes/${data.scope.outcome_id}/work-items/${data.tasks.ready.id}/operational-state`)).state.display_state)
    .toBe("ready");
});

test("a stale Issue version is rejected without silently refreshing the relation intent", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  await wos.login(page);
  await openOutcome(page, data.outcome.title, { area: null });
  await select(page, data.issue);
  await openLinkForm(page, "Link corrective Task");
  await pick(page, "Corrective Task", data.tasks.ready.title);

  await wos.command("update_issue", {
    scope: data.scope,
    issue_id: data.issue.id,
    expected_version: data.issue.version,
    description: "Updated after the Issue was opened in the browser",
  });
  await page.locator("#submit-command").click();
  await expect(page.locator("#command-error")).toContainText("A newer version exists");
  await expect(page.locator("#command-form")).toBeVisible();
  await expect(page.locator("#conflict-review")).toContainText("fresh state/definition review");
  await expect(page.locator("#conflict-review").getByRole("button", { name: "Discard my local edits", exact: true })).toBeVisible();
  expect(await loadRelation(wos, data.scope, data.issue, data.tasks.ready)).toBeUndefined();
});
