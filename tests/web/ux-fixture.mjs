import { test as base, expect } from "playwright/test";
import { spawn } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { resolve, join } from "node:path";
import { createServer } from "node:net";
import { once } from "node:events";
import { randomUUID, randomBytes } from "node:crypto";
import { revealMenuAction, expandWorkspaceDetails } from "./workspace-action-fixture.mjs";
export { expect };
// Overview exposes the specialist navigation. Follow real controls rather than
// revealing hidden content through DOM mutation in a test.
export async function openViews(page) {
  const views = page.locator("#outcome-views");
  if (!(await views.isVisible())) await page.locator('#area-tabs [data-area="overview"]').click();
  return views;
}
export async function chooseArea(page, area) {
  const key = area.toLowerCase() === "outcome board" ? "canvas" : area.toLowerCase();
  if (key !== "canvas") {
    const views=await openViews(page);
    // An uncertain/submitting command legitimately blocks leaving the Board.
    if (!(await views.isVisible())) return;
  }
  await page.locator(`#area-tabs [data-area="${key}"]`).click();
}
export async function openWorkspaceTools(page) {
  const tools = page.locator("#outcome-tools");
  if (!(await tools.evaluate((element) => element.open)))
    await tools.locator(":scope > summary").click();
  return tools;
}
export async function clickWorkspaceTool(page, id) {
  if (id === "refresh" && await page.locator("#refresh").isVisible()) {
    await page.locator("#refresh").click();
    return;
  }
  if (id === "view-outcome" && !(await page.locator("#outcome-tools").isVisible())) {
    await reviewOutcome(page);
    return;
  }
  await openWorkspaceTools(page);
  await page.locator(`#${id}`).click();
}
export async function reviewOutcome(page) {
  if (await page.locator("#detail-dialog").isVisible())
    await page.keyboard.press("Escape");
  await chooseArea(page, "overview");
  await page.locator("#review-outcome").click();
  await expect(page.locator("#detail-dialog")).toBeVisible();
  await expandWorkspaceDetails(page.locator("#detail-dialog"), {history:false});
}
export async function outcomeActions(page) {
  await reviewOutcome(page);
  await page.locator("#entity-actions").click();
  await expect(page.locator("#action-dialog")).toBeVisible();
}
export async function openCanvasContext(page) {
  const context = page.locator("#canvas-context");
  if (!(await context.evaluate((element) => element.open)))
    await context.locator(":scope > summary").click();
  return context;
}
export const test = base.extend({
  wos: async ({}, use) => {
    const directory = await mkdtemp(join(tmpdir(), "wos-human-ux-")),
      socket = createServer();
    socket.listen(0, "127.0.0.1");
    await once(socket, "listening");
    const port = socket.address().port;
    await new Promise((done) => socket.close(done));
    const url = `http://127.0.0.1:${port}`,
      namespace = "01a11992-0000-7000-8000-000000000001",
      token = randomBytes(32).toString("base64url");
    let logs = "";
    const child = spawn(resolve("../../bin/wos"), ["server"], {
      env: {
        ...process.env,
        WOS_LISTEN: `127.0.0.1:${port}`,
        WOS_SQLITE_PATH: join(directory, "wos.db"),
        WOS_AUTH_MODE: "api_token",
        WOS_BOOTSTRAP_TOKEN: token,
        WOS_BOOTSTRAP_NAMESPACE_ID: namespace,
        WOS_BOOTSTRAP_NAMESPACE_NAME: "Human acceptance",
        WOS_LOCAL_PRINCIPAL_ID: "owner",
      },
      stdio: ["ignore", "ignore", "pipe"],
    });
    child.stderr.on("data", (b) => (logs = (logs + b.toString()).slice(-8000)));
    async function api(path, body, credential = token) {
      const response = await fetch(url + "/api/v1" + path, {
        method: body ? "POST" : "GET",
        headers: {
          Authorization: `Bearer ${credential}`,
          "Content-Type": "application/json",
          "Idempotency-Key": randomUUID(),
        },
        body: body ? JSON.stringify(body) : undefined,
      });
      const data = await response.json();
      if (!response.ok)
        throw Error(`${response.status}: ${JSON.stringify(data)}`);
      return data;
    }
    async function command(name, command) {
      return (await api("/commands/" + name, { command })).value;
    }
    async function login(page, credential = token) {
      await page.goto(url + "/app/");
      await page
        .getByLabel("Access credential", { exact: true })
        .fill(credential);
      await page.getByRole("button", { name: "Sign in", exact: true }).click();
      await expect(page.locator("#workspace")).toBeVisible();
      await expect(page.locator("#workspace-loading")).not.toBeVisible();
    }
    async function outcome(title = "Acceptance Outcome") {
      const value = await command("create_outcome", {
        namespace_id: namespace,
        title,
        desired_state: "A verified result",
        priority: "normal",
      });
      return {
        value,
        scope: { namespace_id: namespace, outcome_id: value.id },
      };
    }
    try {
      let ready = false;
      for (let i = 0; i < 100; i++) {
        try {
          if ((await fetch(url + "/readyz")).ok) {
            ready = true;
            break;
          }
        } catch {}
        await new Promise((done) => setTimeout(done, 50));
      }
      if (!ready) throw Error(logs);
      await use({ url, namespace, token, api, command, login, outcome });
    } finally {
      if (child.exitCode === null) {
        const done = once(child, "exit");
        child.kill("SIGTERM");
        await done;
      }
      await rm(directory, { recursive: true, force: true });
    }
  },
});
// Existing specialist-view journeys retain their original Overview starting
// point. Canvas journeys pass { area: null } to assert the product default.
export async function openOutcome(
  page,
  title = "Acceptance Outcome",
  { area = "overview" } = {},
) {
  // Outcome navigation is available in the retained specialist views. Leave
  // the focused Board through Views rather than targeting its hidden sidebar.
  if (await page.locator("#outcome").isVisible() &&
      await page.locator('#area-tabs [data-area="canvas"][aria-pressed="true"]').count())
    await chooseArea(page, "overview");
  const switcher = page.locator("#navigation-toggle");
  if (
    (await switcher.isVisible()) &&
    (await switcher.getAttribute("aria-expanded")) === "false"
  )
    await switcher.click();
  await page
    .locator("#outcomes")
    .getByRole("button")
    .filter({ hasText: title })
    .click();
  await expect(page.locator("#title")).toHaveText(title);
  if (area)
    await chooseArea(page, area);
}
export async function intent(page, label) {
  // Modal journeys use actual collection/Overview controls. Global Board
  // actions remain contextual and have independent inline-inspector coverage.
  if (await page.locator('#area-tabs [data-area="canvas"][aria-pressed="true"]').count())
    await chooseArea(page, "overview");
  const add = page.locator("#add-item");
  const quick = page.locator(".summary-quick").getByRole("button", { name: label, exact: true });
  if (await add.isVisible() && (await add.innerText()).replace(/^\+\s*/, "") === label)
    await add.click();
  else if (await quick.isVisible())
    await quick.click();
  else {
    await outcomeActions(page);
    const menuLabel = label === "New task" ? "New task · Backlog" : label;
    const target = page.locator("#action-options").getByRole("button", { name: menuLabel, exact: true, includeHidden: true });
    await revealMenuAction(target);
    await target.click();
  }
  await expect(page.locator("#command-dialog")).toBeVisible();
  await expect(page.locator("#command-select")).not.toBeVisible();
}
export async function save(page) {
  await page.locator("#submit-command").click();
  await expect(page.locator("#command-dialog")).not.toBeVisible();
  await expect(page.locator("#submit-command")).toBeEnabled();
  await expect(page.locator("#notice")).not.toHaveClass(/error/);
}
export async function pick(page, label, query, match = query) {
  const input = page.getByRole("combobox", { name: label, exact: true });
  await input.fill(query);
  const root = page.locator(".reference-picker").filter({ has: input });
  await root.getByRole("option").filter({ hasText: match }).first().click();
}
export async function detail(page, section, title) {
  if (await page.locator("#detail-dialog").isVisible())
    await page.keyboard.press("Escape");
  const area = {
    Tasks: "Work",
    Objectives: "Plan",
    "All roadmaps": "Plan",
    Issues: "Issues",
    Blockers: "Issues",
    Evidence: "Evidence",
    Artifacts: "Evidence",
    Decisions: "Evidence",
  }[section];
  if (area)
    await chooseArea(page, area);
  if (section === "Tasks")
    await page
      .getByLabel("Work status", { exact: true })
      .selectOption("work_items");
  else if (!["Issues", "Evidence"].includes(section))
    await page
      .locator("#tabs")
      .getByRole("button", { name: section, exact: true })
      .click();
  await page
    .locator("#content")
    .getByRole("button")
    .filter({ hasText: title })
    .first()
    .click();
  await expect(page.locator("#detail-dialog")).toBeVisible();
  await expandWorkspaceDetails(page.locator("#detail-dialog"), {history:false});
}
