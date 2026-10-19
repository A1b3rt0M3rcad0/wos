import { test, expect, openOutcome, pick, chooseArea, clickWorkspaceTool } from "./ux-fixture.mjs";
import { seedCanvas, provisionCanvasViewer } from "./canvas-fixture.mjs";

const canvas = (page) => page.locator("#operational-canvas");
const panel = (page) => page.locator("#canvas-details");
const base = (scope) => `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`;

async function openCanvas(page, wos, data, credential) {
  await wos.login(page, credential);
  await openOutcome(page, data.outcome.title, { area: null });
  await expect(canvas(page)).toBeVisible();
}
async function select(page, value, { wait = true } = {}) {
  await canvas(page).getByRole("combobox", { name: "Search Outcome Board", exact: true }).fill(value.title);
  await canvas(page).getByRole("option").filter({ hasText: value.title }).first().click();
  if (wait) await expect(panel(page).locator("#detail-title")).toHaveText(value.title);
}
async function openAction(page, action) {
  const primary = panel(page).locator(`.workspace-actions [data-action-id="${action}"]`);
  if (await primary.isVisible()) return primary.click();
  await panel(page).locator('#entity-actions').click();
  await panel(page).locator(`#action-options [data-action-id="${action}"]`).click();
}

// Capture a genuine server result and gate its delivery. The optional failure
// simulates transport loss; lifecycle, permissions and conflicts stay real.
async function holdOnce(page, pattern, { failureMessage, subjectId } = {}) {
  let release, seen, failed, finished;
  let started = false;
  const gate = new Promise((resolve) => { release = resolve; });
  const arrived = new Promise((resolve, reject) => { seen = resolve; failed = reject; });
  const done = new Promise((resolve) => { finished = resolve; });
  await page.route(pattern, async (route) => {
    if (started || subjectId && route.request().postDataJSON()?.subject?.ref?.id !== subjectId)
      return route.continue();
    started = true;
    try {
      const response = await route.fetch();
      seen(response.status());
      await gate;
      await route.fulfill(failureMessage
        ? { status: 500, contentType: "application/json", body: JSON.stringify({ error: { message: failureMessage } }) }
        : { response });
    } catch (error) {
      failed(error);
      throw error;
    } finally {
      finished();
    }
  });
  return { arrived, release, done, get started() { return started; } };
}
async function settle(page, held) {
  held.release();
  await held.done;
  await page.waitForLoadState("networkidle");
}

test("a late form restoration preserves edge keyboard focus and exact authorized removal", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  const relation = await wos.command("add_dependency", {
    scope: data.scope, source_ref: data.ref("work_item", data.tasks.unscoped),
    target_ref: data.ref("work_item", data.tasks.ready), strength: "advisory",
    reason: "A real advisory relationship for keyboard selection",
  });
  await openCanvas(page, wos, data);
  await select(page, data.tasks.ready);
  await openAction(page, "add_dependency");
  await pick(page, "Depends on", data.tasks.unscoped.title);
  await page.getByLabel("Why is this prerequisite required?", { exact: true }).fill("The server must still reject the reverse cycle");
  await panel(page).locator("#submit-command").click();
  await expect(panel(page).locator("#command-error")).toContainText(/cycl/i);
  await expect(panel(page).locator("#submit-command")).toBeEnabled();
  const held = await holdOnce(page, "**/workspace-view", { subjectId: data.tasks.ready.id });
  try {
    await page.keyboard.press("Escape");
    expect(await held.arrived).toBe(200);
    await canvas(page).getByLabel("Connections", { exact: true }).selectOption("all");
    const edge = canvas(page).locator(`[data-edge-key="graph:relation:${relation.id}"]`);
    await edge.focus();
    await expect(edge).toBeFocused();
    const camera = await canvas(page).locator(".canvas-world").evaluate((element) => element.style.transform);
    await settle(page, held);
    await expect(edge).toBeFocused();
    expect(await canvas(page).locator(".canvas-world").evaluate((element) => element.style.transform)).toBe(camera);
    await edge.press("Enter");
    await expect(page).toHaveURL(new RegExp(`item=relation(?:%2F|/)${relation.id}`));
    await expect(panel(page).locator("#detail-kind")).toContainText(/relation/i);
    await expect(panel(page)).toContainText(/advisory/i);
    await openAction(page, "remove_dependency");
    await page.getByLabel("Reason", { exact: true }).fill("The selected advisory context is no longer required");
    await panel(page).locator("#submit-command").click();
    await expect(panel(page).locator("#command-form")).toBeHidden();
    await expect(edge).toHaveCount(0);
    const persisted = (await wos.api(`${base(data.scope)}/relations`)).items.find((item) => item.id === relation.id);
    expect(persisted.lifecycle).toBe("removed");
    expect(persisted.version).toBe(relation.version + 1);
    for (const task of [data.tasks.ready, data.tasks.unscoped])
      expect((await wos.api(`${base(data.scope)}/work-items/${task.id}`)).value.lifecycle).toBe("todo");
  } finally {
    held.release();
    if (held.started) await held.done;
  }
});

test("a late Task focal read cannot reopen a closed inspector or report an obsolete revision error", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  await openCanvas(page, wos, data);
  await select(page, data.tasks.ready);
  const delayedTask = data.tasks.unscoped;
  const held = await holdOnce(page, "**/workspace-view", { subjectId: delayedTask.id });
  try {
    await select(page, delayedTask, { wait: false });
    expect(await held.arrived).toBe(200);
    await panel(page).locator("#close-detail").click();
    await chooseArea(page, "work");
    await expect(page.locator("#content")).toContainText(delayedTask.title);
    await expect(page.locator("#notice")).not.toHaveClass(/error/);
    // A fresh canonical view would now observe a newer revision. Closing the
    // previous selection must invalidate that entire obsolete read chain.
    await wos.command("update_work_item", {
      scope: data.scope, work_item_id: delayedTask.id,
      expected_version: delayedTask.version, title: "Changed after the old selection was closed",
    });
    await settle(page, held);
    await expect(page.locator('#area-tabs [data-area="work"]')).toHaveAttribute("aria-pressed", "true");
    await expect(panel(page)).toBeHidden();
    await expect(page.locator("#detail-dialog")).toBeHidden();
    await expect(page).not.toHaveURL(/item=/);
    await expect(page.locator("#notice")).not.toHaveClass(/error/);
  } finally {
    held.release();
    if (held.started) await held.done;
  }
});

test("a late comparison for Task A cannot replace Task B's draft or introduce an old-target save", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  await openCanvas(page, wos, data);
  const first = data.tasks.ready, second = data.tasks.unscoped;
  await select(page, first);
  await openAction(page, "update_work_item");
  await page.getByLabel("Task title", { exact: true }).fill("Task A local draft");
  const remote = await wos.command("update_work_item", {
    scope: data.scope, work_item_id: first.id,
    expected_version: first.version, title: "Task A changed by another participant",
  });
  const refreshed = page.waitForResponse((response) => response.url().includes(`${base(data.scope)}/continuity?`));
  await clickWorkspaceTool(page, "refresh");
  await refreshed;
  await page.waitForLoadState("networkidle");
  await panel(page).locator("#submit-command").click();
  await expect(panel(page).locator("#command-error")).toContainText("A newer version exists");
  const held = await holdOnce(page, `**/work-items/${first.id}`);
  try {
    await page.getByRole("button", { name: "Compare with latest state", exact: true }).click();
    expect(await held.arrived).toBe(200);
    await panel(page).locator("#close-command").click();
    await expect(panel(page).locator("#detail-title")).toHaveText(remote.title);
    await select(page, second);
    await openAction(page, "update_work_item");
    await page.getByLabel("Task title", { exact: true }).fill("Task B independently edited");
    await settle(page, held);
    await expect(page.getByLabel("Task title", { exact: true })).toHaveValue("Task B independently edited");
    await expect(page.locator("#conflict-review")).toBeEmpty();
    await expect(page.getByRole("button", { name: "Use latest version and save my edits", exact: true })).toHaveCount(0);
    await panel(page).locator("#submit-command").click();
    await expect(panel(page).locator("#command-form")).toBeHidden();
    expect((await wos.api(`${base(data.scope)}/work-items/${first.id}`)).value.title).toBe(remote.title);
    expect((await wos.api(`${base(data.scope)}/work-items/${second.id}`)).value.title).toBe("Task B independently edited");
  } finally {
    held.release();
    if (held.started) await held.done;
  }
});

test("an earlier successful refresh cannot restore private Canvas content after a genuine access denial", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  const viewer = await provisionCanvasViewer(wos);
  await openCanvas(page, wos, data, viewer.token);
  await select(page, data.tasks.ready);
  const held = await holdOnce(page, `**/outcomes/${data.outcome.id}/continuity?*`);
  try {
    await clickWorkspaceTool(page, "refresh");
    expect(await held.arrived).toBe(200);
    const administration = await wos.api(`/namespaces/${wos.namespace}/administration`);
    await wos.api("/security/commands", {
      namespace_id: wos.namespace, expected_namespace_version: administration.namespace_version,
      operation: "set_grant", principal_id: "canvas-viewer", permissions: [],
    });
    const denied = page.waitForResponse((response) => response.url().endsWith("/workspace-view")
      && response.request().method() === "POST"
      && response.request().postDataJSON()?.subject?.ref?.id === data.tasks.unscoped.id
      && response.status() === 403);
    // The previous canvas adapter is intentionally invalidated while refresh
    // is in flight. A keyboard selection still performs a fresh entity read.
    await canvas(page).locator(`[data-canvas-node="work_item:${data.tasks.unscoped.id}"] .canvas-node-main`).focus();
    await page.keyboard.press("Enter");
    expect((await denied).status()).toBe(403);
    await expect(page.locator("#outcome")).toBeHidden();
    await settle(page, held);
    await expect(page.locator("#outcome")).toBeHidden();
    await expect(panel(page)).toBeHidden();
    await expect(page.locator("#detail-content")).toBeEmpty();
    await expect(canvas(page).locator(".workspace-canvas")).toHaveCount(0);
    await expect(page.locator("#actions")).toBeDisabled();
  } finally {
    held.release();
    if (held.started) await held.done;
  }
});

test("an obsolete Task history transport failure cannot replace the current Task or its notice", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  // The coherent inspector loads its recent page automatically. Seed enough
  // real notes to exercise a separately paginated history read.
  for (let index = 0; index < 7; index++)
    await wos.command("append_entity_note", {
      scope: data.scope, subject_ref: data.ref("work_item", data.tasks.ready),
      type: "update", body: `History page belonging to the previous Task ${index}`,
    });
  await openCanvas(page, wos, data);
  await select(page, data.tasks.ready);
  const failureMessage = "Obsolete Task history transport failure";
  const held = await holdOnce(page,
    (url) => url.pathname.endsWith("/workspace-timeline") && url.searchParams.get("entity_id") === data.tasks.ready.id,
    { failureMessage },
  );
  try {
    await panel(page).locator('[data-action-id="load_more_activity"]').click();
    expect(await held.arrived).toBe(200);
    await panel(page).locator("#close-detail").click();
    await select(page, data.tasks.unscoped);
    const currentNotice = await page.locator("#notice").textContent();
    const currentClass = await page.locator("#notice").getAttribute("class");
    await settle(page, held);
    await expect(panel(page).locator("#detail-title")).toHaveText(data.tasks.unscoped.title);
    await expect(panel(page)).not.toContainText("History page belonging to the previous Task");
    await expect(panel(page)).not.toContainText(failureMessage);
    await expect(page.locator("#notice")).toHaveText(currentNotice);
    expect(await page.locator("#notice").getAttribute("class")).toBe(currentClass);
    await expect(page).toHaveURL(new RegExp(data.tasks.unscoped.id));
  } finally {
    held.release();
    if (held.started) await held.done;
  }
});
