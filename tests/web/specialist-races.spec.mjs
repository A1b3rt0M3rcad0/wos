import { test, expect, openOutcome, clickWorkspaceTool } from "./ux-fixture.mjs";

const base = (scope) =>
  `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}`;

// Freeze a genuine, already-completed server read. Only delivery is controlled;
// all state, permissions, revisions and cursor validation remain real.
async function gatedReads(page) {
  const pending = [];
  // Keep one dispatcher installed while reads are in flight. Replacing route
  // registrations mid-flight can itself resolve a Playwright interception.
  await page.route("**/api/v1/**", async (route) => {
    const url = new URL(route.request().url());
    const held = pending.find((entry) => !entry.started && entry.pattern(url));
    if (!held) return route.continue();
    held.started = true;
    try {
      const response = await route.fetch();
      held.seen({
        status: response.status(),
        value: await response.json(),
        url: route.request().url(),
      });
      await held.gate;
      await route.fulfill({ response });
    } catch (error) {
      held.failed(error);
      throw error;
    } finally {
      held.finished();
    }
  });
  return (pattern) => {
    const held = { pattern, started: false };
    held.gate = new Promise((resolve) => {
      held.release = resolve;
    });
    held.arrived = new Promise((resolve, reject) => {
      held.seen = resolve;
      held.failed = reject;
    });
    held.done = new Promise((resolve) => {
      held.finished = resolve;
    });
    pending.push(held);
    return held;
  };
}

async function deliver(page, held, url) {
  const received = page.waitForResponse((response) => response.url() === url);
  held.release();
  await held.done;
  await (await received).finished();
  // Drain the response continuation and its DOM publication before releasing
  // the next barrier. This is a browser frame boundary, not a timing delay.
  await page.evaluate(
    () =>
      new Promise((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(resolve)),
      ),
  );
}

function trackBrowserWrites(page) {
  const writes = [];
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/commands/"))
      writes.push(request.url());
  });
  return writes;
}

function permissionBarrier(hold, scope) {
  return hold(
    (url) =>
      url.pathname.endsWith(
        `/namespaces/${scope.namespace_id}/effective-permissions`,
      ) && url.searchParams.get("outcome_id") === scope.outcome_id,
  );
}

async function openSpecialist(page, wos, outcome, area) {
  await wos.login(page);
  await openOutcome(page, outcome.title, { area });
  // Collection readiness is represented by its rendered heading and records;
  // routine Board feedback no longer creates a second persistent status row.
  await expect(page.locator("#content .content-heading h2")).toBeVisible();
  await expect(page.locator("#notice")).not.toHaveClass(/error/);
  await expect(
    page.locator(`#area-tabs [data-area="${area}"]`),
  ).toHaveAttribute("aria-pressed", "true");
}

test("an old Roadmap list cannot refill a new snapshot's cache while refresh permissions are pending", async ({
  page,
  wos,
}) => {
  const { value: outcome, scope } = await wos.outcome(
    "Refresh keeps the current Roadmap collection",
  );
  const first = await wos.command("create_roadmap", {
    scope,
    title: "The original published-work plan",
    plan_scope: { kind: "outcome", id: outcome.id },
  });
  const writes = trackBrowserWrites(page);
  const hold = await gatedReads(page);
  await openSpecialist(page, wos, outcome, "plan");
  await expect(page.locator("#content .content-heading h2")).toHaveText(
    "Objectives",
  );
  const oldList = hold(
    (url) =>
      url.pathname.endsWith(`${base(scope)}/roadmaps`) ||
      (url.pathname.endsWith(`${base(scope)}/references`) &&
        url.searchParams.get("kind") === "roadmap"),
  );
  let permissions;
  try {
    await page.locator('#tabs [data-section="roadmaps"]').click();
    const old = await oldList.arrived;
    expect(old.status).toBe(200);
    expect(old.value.items.map((item) => item.ref?.id || item.id)).toEqual([
      first.id,
    ]);
    const next = await wos.command("create_roadmap", {
      scope,
      title: "A new plan created before refresh completes",
      plan_scope: { kind: "outcome", id: outcome.id },
    });
    const latest = await wos.api(`${base(scope)}/continuity?limit=25`);
    expect(String(latest.outcome_revision)).not.toBe(
      String(old.value.outcome_revision),
    );
    permissions = permissionBarrier(hold, scope);
    await clickWorkspaceTool(page, "refresh");
    const currentPermissions = await permissions.arrived;
    expect(currentPermissions.status).toBe(200);
    await expect(page.locator("#revision")).toContainText(
      `Revision ${latest.outcome_revision}`,
    );
    // The new snapshot is already installed and its caches cleared. Delivering
    // the old list here must not populate that replacement cache.
    await deliver(page, oldList, old.url);
    await deliver(page, permissions, currentPermissions.url);
    await expect(
      page.locator(`#content [data-entity-id="${next.id}"]`),
    ).toContainText(next.title);
    await expect(page.locator("#content [data-entity-id]")).toHaveCount(2);
    await expect(page.locator("#notice")).not.toHaveClass(/error/);
    await page.locator('#tabs [data-section="objectives"]').click();
    await expect(page.locator("#content .content-heading h2")).toHaveText(
      "Objectives",
    );
    await page.locator('#tabs [data-section="roadmaps"]').click();
    await expect(
      page.locator(`#content [data-entity-id="${next.id}"]`),
    ).toContainText(next.title);
    expect(writes).toEqual([]);
  } finally {
    oldList.release();
    permissions?.release();
    if (oldList.started) await oldList.done;
    if (permissions?.started) await permissions.done;
  }
});

test("a delayed Board page cannot append obsolete records after a same-Outcome refresh", async ({
  page,
  wos,
}) => {
  const { value: outcome, scope } = await wos.outcome(
    "Refresh preserves the Board's current page boundary",
  );
  const tasks = [];
  const objective = await wos.command("create_objective", { scope, title: "Queued work Objective", priority: "normal" });
  for (let index = 0; index < 26; index++)
    tasks.push(
      await wos.command("create_work_item", {
        scope,
        title: `Queued work ${String(index + 1).padStart(2, "0")}`,
        priority: "normal",
        lifecycle: "backlog",
        objective_id: objective.id,
      }),
    );
  const writes = trackBrowserWrites(page);
  const hold = await gatedReads(page);
  await openSpecialist(page, wos, outcome, "work");
  await page.locator('#view-tabs [data-view="board"]').click();
  const board = page.locator("#content .board");
  await expect(board.locator("[data-entity-id]")).toHaveCount(25);
  const oldPage = hold(
    (url) =>
      url.pathname.endsWith(`${base(scope)}/continuity/backlog_work`) &&
      url.searchParams.has("cursor"),
  );
  let permissions;
  try {
    await page
      .locator('.board-column[aria-label="Planned"]')
      .getByRole("button", { name: /Load more.*Backlog/ })
      .click();
    const old = await oldPage.arrived;
    expect(old.status).toBe(200);
    expect(old.value.items).toHaveLength(1);
    const oldSummary = old.value.items[0].work_item;
    const task = tasks.find((item) => item.id === oldSummary.ref.id);
    expect(task).toBeTruthy();
    const updated = await wos.command("update_work_item", {
      scope,
      work_item_id: task.id,
      expected_version: task.version,
      title: "The queued Task changed while its old page was in flight",
    });
    const latest = await wos.api(`${base(scope)}/continuity?limit=25`);
    expect(String(latest.outcome_revision)).not.toBe(
      String(old.value.outcome_revision),
    );
    permissions = permissionBarrier(hold, scope);
    await clickWorkspaceTool(page, "refresh");
    const currentPermissions = await permissions.arrived;
    expect(currentPermissions.status).toBe(200);
    await expect(page.locator("#revision")).toContainText(
      `Revision ${latest.outcome_revision}`,
    );
    await deliver(page, oldPage, old.url);
    await deliver(page, permissions, currentPermissions.url);
    // Refresh returns to the current first page. The late old page must not
    // become a 26th card in the new revision or consume its new cursor.
    await expect(board.locator("[data-entity-id]")).toHaveCount(25);
    await expect(board.locator(`[data-entity-id="${task.id}"]`)).toHaveCount(0);
    await page
      .locator('.board-column[aria-label="Planned"]')
      .getByRole("button", { name: /Load more.*Backlog/ })
      .click();
    await expect(board.locator("[data-entity-id]")).toHaveCount(26);
    await expect(board.locator(`[data-entity-id="${task.id}"]`)).toHaveCount(1);
    await expect(board.locator(`[data-entity-id="${task.id}"]`)).toContainText(
      updated.title,
    );
    await expect(board).not.toContainText(oldSummary.title);
    const ids = await board
      .locator("[data-entity-id]")
      .evaluateAll((items) => items.map((item) => item.dataset.entityId));
    expect(new Set(ids).size).toBe(ids.length);
    await expect(page.locator("#notice")).not.toHaveClass(/error/);
    expect(writes).toEqual([]);
  } finally {
    oldPage.release();
    permissions?.release();
    if (oldPage.started) await oldPage.done;
    if (permissions?.started) await permissions.done;
  }
});
