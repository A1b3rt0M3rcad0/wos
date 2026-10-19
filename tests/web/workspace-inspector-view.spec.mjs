import { test, expect, openOutcome, chooseArea } from "./ux-fixture.mjs";
import { seedCanvas } from "./canvas-fixture.mjs";

const panel = (page) => page.locator("#canvas-details");
const ref = (scope, kind, id) => ({ ...scope, kind, id });

async function selectBoardRecord(page, title, heading = title) {
  const search = page.getByRole("combobox", { name: "Search Outcome Board", exact: true });
  await search.fill(title);
  await page.getByRole("option").filter({ hasText: title }).first().click();
  await expect(panel(page).locator("#detail-title")).toHaveText(heading);
  await expect(panel(page).locator(".workspace-view")).toBeVisible();
}

async function setup(wos) {
  const data = await seedCanvas(wos, { title: "Understand the recorded work" });
  const task = data.tasks.ready;
  const owner = ref(data.scope, "work_item", task.id);
  const criterion = await wos.command("add_criterion", {
    owner, expected_version: task.version, title: "The documented installation starts a usable workspace",
    required: true, verification_mode: "evidence_review",
  });
  const current = (await wos.api(`/namespaces/${data.scope.namespace_id}/outcomes/${data.scope.outcome_id}/work-items/${task.id}`)).value;
  await wos.command("set_work_item_assignees", {
    scope: data.scope, work_item_id: task.id, expected_version: current.version,
    assignee_refs: [{ kind: "agent", provider: "acceptance", id: "declared-coding-agent" }],
  });
  for (let index = 0; index < 7; index++) await wos.command("append_entity_note", {
    scope: data.scope, subject_ref: owner, type: index === 6 ? "observation" : "update",
    body: `Recorded update ${index + 1}: this is a participant statement, not accepted work.`,
  });
  const artifact = await wos.command("register_artifact_in_context", {
    scope: data.scope, subject_ref: owner, relationship: "produces",
    artifact: { name: "Recorded installation transcript", artifact_type: "transcript", uri: "https://example.test/transcript/1" },
  });
  const decision = await wos.command("propose_decision_in_context", {
    scope: data.scope, subject_ref: owner,
    decision: { title: "Use the existing installation route", proposal: "Keep the reviewed compose entry point", rationale: "Preserve documented startup behavior" },
  });
  return { ...data, criterion, artifact: artifact.artifact, decision: decision.decision };
}

test("one coherent Task inspector exposes the five information groups without granting authority or accepting linked proof", async ({ page, wos }) => {
  const data = await setup(wos);
  const mutations = [], errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/commands/")) mutations.push(request.url());
  });
  await wos.login(page);
  await openOutcome(page, data.outcome.title, { area: null });
  await selectBoardRecord(page, data.tasks.ready.title);
  const view = panel(page).locator(".workspace-view");
  for (const facet of ["work", "people", "attention", "criteria", "activity"])
    await expect(view.locator(`[data-workspace-facet="${facet}"]`)).toBeVisible();
  await expect(view).toContainText("Perform only the scoped task");
  await expect(view).toContainText("A reviewable result");
  await expect(view).toContainText("Declared assignee");
  await expect(view).toContainText("declared-coding-agent");
  await expect(view).not.toContainText("Current recorded authority");
  await expect(view).toContainText("does not grant execution authority");
  const criterion = view.locator(`[data-criterion-id="${data.criterion.id}"]`);
  await expect(criterion).toContainText("Not assessed");
  await expect(criterion).not.toContainText("Met");
  await expect(view).toContainText("Contradicts");
  await expect(view).toContainText("Human observation");
  await expect(view).toContainText("Recorded update 7");
  const more = view.locator('[data-action-id="load_more_activity"]');
  await more.click();
  await expect(view).toContainText("Recorded update 3");
  await expect(view.locator("[data-activity-id]")).toHaveCount(10);
  await view.locator(`[data-criterion-id="${data.criterion.id}"] [data-action-id="open_criterion"]`).click();
  await expect(panel(page).locator("#detail-title")).toHaveText(data.criterion.title);
  await expect(panel(page).locator(".workspace-view")).toContainText("Not assessed");
  await panel(page).locator('.workspace-path [data-reference-kind="work_item"]').click();
  await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.ready.title);
  expect(mutations).toEqual([]);
  expect(errors).toEqual([]);
});

test("an Issue names its affected work without treating the association itself as blocking", async ({ page, wos }) => {
  const data = await setup(wos);
  await wos.login(page);
  await openOutcome(page, data.outcome.title, { area: null });
  await selectBoardRecord(page, data.issue.title);
  await expect(panel(page)).toContainText("Only explicit Blockers record an impediment");
  await expect(panel(page)).toContainText(data.tasks.blocked.title);
});

test("the same typed inspector explains documentary entities and actual blocking in specialist views", async ({ page, wos }) => {
  const data = await setup(wos);
  await wos.login(page);
  await openOutcome(page, data.outcome.title, { area: null });
  await selectBoardRecord(page, data.blocker.description, "Blocker");
  await expect(panel(page)).toContainText("Evidence belongs to the cause or affected work");
  await expect(panel(page)).toContainText(data.issue.title);
  await selectBoardRecord(page, data.artifact.name);
  await expect(panel(page)).toContainText("not an upload or a verification");
  await expect(panel(page).getByRole("link", { name: "https://example.test/transcript/1", exact: true })).toHaveAttribute("rel", "noopener noreferrer");
  await selectBoardRecord(page, data.decision.title);
  await expect(panel(page)).toContainText("do not release a Blocker automatically");
  await expect(panel(page)).toContainText(data.decision.proposal);
  await chooseArea(page, "work");
  await page.locator("#content").getByRole("button").filter({ hasText: data.tasks.ready.title }).first().click();
  await expect(page.locator("#detail-dialog .workspace-view")).toBeVisible();
  await expect(page.locator('#detail-dialog [data-workspace-facet="criteria"]')).toContainText(data.criterion.title);
});

test("the shared inspector retains readable content and reachable keyboard controls across CSS viewports", async ({ page, wos }) => {
  test.setTimeout(180000);
  const data = await setup(wos);
  await wos.login(page);
  await openOutcome(page, data.outcome.title, { area: null });
  for (const viewport of [
    { width: 1280, height: 720 }, { width: 1366, height: 768 },
    { width: 1920, height: 1080 }, { width: 2560, height: 1440 },
    { width: 3840, height: 2160 }, { width: 320, height: 740 }, { width: 430, height: 932 },
  ]) {
    await page.setViewportSize(viewport);
    await selectBoardRecord(page, data.tasks.ready.title);
    const detail = panel(page);
    const geometry = await detail.evaluate((element) => {
      const rect = element.getBoundingClientRect();
      const heading = element.querySelector("#detail-title");
      return { left: rect.left, right: rect.right, top: rect.top, bottom: rect.bottom,
        headingSize: parseFloat(getComputedStyle(heading).fontSize),
        overflow: document.documentElement.scrollWidth - innerWidth };
    });
    expect(geometry.left).toBeGreaterThanOrEqual(-1);
    expect(geometry.right).toBeLessThanOrEqual(viewport.width + 1);
    expect(geometry.bottom).toBeLessThanOrEqual(viewport.height + 1);
    expect(geometry.headingSize).toBeLessThanOrEqual(20);
    expect(geometry.overflow).toBeLessThanOrEqual(0);
    await detail.locator('[data-action-id="more_actions"]').focus();
    await expect(detail.locator('[data-action-id="more_actions"]')).toBeFocused();
    await page.keyboard.press("Enter");
    await expect(detail.locator(".canvas-inspector-actions")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(detail.locator("#detail-title")).toHaveText(data.tasks.ready.title);
    await page.keyboard.press("Escape");
    await expect(detail).not.toBeVisible();
  }
});

test("resolving an Issue releases only the explicitly selected recorded impact", async ({ page, wos }) => {
  const data = await seedCanvas(wos, { title: "Release one recorded impact" });
  const other = await wos.command("create_blocker", {
    scope: data.scope, blocked_ref: data.ref("work_item", data.tasks.advisory),
    cause_ref: data.ref("issue", data.issue), propagation: "direct",
    description: "Keep the second impact active until it has been independently checked",
  });
  await wos.login(page);
  await openOutcome(page, data.outcome.title, { area: null });
  await selectBoardRecord(page, data.issue.title);
  await panel(page).locator('[data-action-id="more_actions"]').click();
  await page.locator('#action-options [data-action-id="resolve_issue_and_blockers"]').click();
  await expect(page.locator("#command-form")).toBeVisible();
  const release = page.getByLabel(data.tasks.blocked.title, { exact: true });
  const retain = page.getByLabel(data.tasks.advisory.title, { exact: true });
  await expect(release).not.toBeChecked();
  await expect(retain).not.toBeChecked();
  await release.check();
  await page.getByLabel("Issue resolution summary", { exact: true }).fill("The first affected environment has verified access; the second remains blocked.");
  await page.getByLabel("Confirm each selected impact is released", { exact: true }).check();
  const response = page.waitForResponse((response) => response.request().method() === "POST"
    && response.url().endsWith("/commands/resolve_issue_and_blockers"));
  await page.locator("#submit-command").click();
  const result = await response;
  expect(result.ok(), await result.text()).toBe(true);
  await expect(page.locator("#command-form")).not.toBeVisible();
  const base = `/namespaces/${data.scope.namespace_id}/outcomes/${data.scope.outcome_id}`;
  const released = (await wos.api(`${base}/blockers/${data.blocker.id}`)).value;
  const retained = (await wos.api(`${base}/blockers/${other.id}`)).value;
  const issue = (await wos.api(`${base}/issues/${data.issue.id}`)).value;
  expect(issue.lifecycle).toBe("resolved");
  expect(released.lifecycle).toBe("resolved");
  expect(retained.lifecycle).toBe("active");
  await selectBoardRecord(page, data.issue.title);
  await expect(panel(page)).toContainText(other.description);
});
