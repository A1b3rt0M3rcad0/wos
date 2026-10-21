import { readFileSync } from "node:fs";
import { test, expect } from "playwright/test";

const assets = new URL("../../packages/wos-api/web/assets/", import.meta.url);
const scope = { namespace_id: "workspace", outcome_id: "outcome" };

async function fixture(page, kind = "work_item") {
  await page.route("http://placement-preview.test/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (/^\/assets\/[a-zA-Z0-9.-]+\.js$/.test(path)) return route.fulfill({
      contentType: "text/javascript", body: readFileSync(new URL(path.slice(8), assets), "utf8"),
    });
    return route.fulfill({ contentType: "text/html", body: '<!doctype html><html lang="en"><title>Review structural movement</title><body><form id="form"><div id="fields"></div><button type="submit">Save move</button><p id="error" role="alert"></p></form></body></html>' });
  });
  await page.goto("http://placement-preview.test/");
  await page.evaluate(async ({ scope, kind }) => {
    const { structuralMoveEditor } = await import("/assets/structural-form-preview.js");
    const ref = (kind, id) => ({ ...scope, kind, id });
    const header = (id) => ({ ref: ref(id === "outcome" ? "outcome" : "objective", id),
      title: ({ outcome: "Release the system", before: "Current branch", after: "New branch", alternative: "Alternative branch" })[id] });
    const entity = { id: "selected", _kind: kind, title: "Ship the installer", version: "3",
      [kind === "work_item" ? "objective_id" : "parent_objective_id"]: "before" };
    window.reads = [];
    window.submits = [];
    window.editor = structuralMoveEditor({ scope, kind, entity,
      workspaceView: { placement: { parent: header("before") } },
      outcomeTitle: "Release the system",
      request: async (path, options = {}) => {
        window.reads.push({ path, method: options.method || "GET", body: options.body && JSON.parse(options.body) });
        if (!options.method) {
          const query = new URL(path, location.origin).searchParams;
          return { items: ["before", "after", "alternative"].map((id) => header(id))
            .filter((item) => (!query.get("id") || item.ref.id === query.get("id"))
              && (!query.get("query") || item.title.includes(query.get("query")))) };
        }
        if (options.method !== "POST" || !path.endsWith("/workspace-view")) throw Error("A preview may only read the workspace");
        const body = JSON.parse(options.body);
        const response = { subject: { kind: "entity", ref: ref(kind, entity.id) },
          entity_version: "9007199254740993", outcome_revision: "9007199254740995",
          placement_preview: { old_parent: header("before"), new_parent: header(body.placement_preview.parent_id || "outcome"),
            changed: body.placement_preview.parent_id !== "before",
            availability: window.blocked ? "blocked" : "available",
            reason_codes: window.blocked ? ["objective_subtree_in_execution"] : [],
            affected_objectives: kind === "objective" ? 2 : 0, affected_tasks: kind === "objective" ? 8 : 1,
            blocking_changes: [{ ref: ref(kind, entity.id), title: entity.title,
              before: { is_blocked: true, blockers: [{ ref: ref("blocker", "gate"), description: "Production access pending", blocked_ref: ref("objective", "before"), inherited: true }],
                coverage: { status: window.partial ? "partial" : "complete", total_count: window.partial ? 25 : 1, returned_count: 1 } },
              after: { is_blocked: false, blockers: [] } }],
            coverage: { status: window.partial ? "partial" : "complete", returned_count: 1,
              total_count: kind === "objective" ? 10 : 1 } } };
        if (window.foreign) response.subject.ref.namespace_id = "foreign";
        if (window.deferPreview) await new Promise((resolve) => { window.releasePreview = resolve; });
        return response;
      },
    });
    document.querySelector("#fields").append(window.editor.node);
    document.querySelector("#form").onsubmit = (event) => {
      event.preventDefault();
      try {
        window.submits.push(window.editor.get());
        document.querySelector("#error").textContent = "";
      } catch (error) { document.querySelector("#error").textContent = error.message; }
    };
  }, { scope, kind });
}

async function destination(page, title) {
  if (!await page.getByRole("combobox", { name: "New placement", exact: true }).count())
    await page.getByRole("button", { name: "Change placement", exact: true }).click();
  await page.getByRole("combobox", { name: "New placement", exact: true }).fill(title);
  await page.getByRole("option").filter({ hasText: title }).click();
}

test("a move requires a coherent named before/after impact and acknowledgement before emitting exact versions", async ({ page }) => {
  await fixture(page);
  await page.getByRole("button", { name: "Save move", exact: true }).click();
  await expect(page.locator("#error")).toContainText("Preview this destination");
  await destination(page, "New branch");
  await page.getByRole("button", { name: "Preview move", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Observed movement impact", exact: true })).toBeVisible();
  await expect(page.locator(".workspace-move-preview")).toContainText("Current branch");
  await expect(page.locator(".workspace-move-preview")).toContainText("New branch");
  await expect(page.locator(".workspace-move-preview")).toContainText("Production access pending");
  await expect(page.locator(".workspace-move-preview")).toContainText("Before the move");
  await expect(page.locator(".workspace-move-preview")).toContainText("After the move");
  await page.getByRole("button", { name: "Save move", exact: true }).click();
  expect(await page.evaluate(() => window.submits.length)).toBe(0);
  await page.getByLabel("I reviewed the new placement and its blocking impacts", { exact: true }).check();
  await page.getByRole("button", { name: "Save move", exact: true }).click();
  expect(await page.evaluate(() => window.submits)).toEqual([{ objective_id: "after",
    expected_version: "9007199254740993", expected_outcome_revision: "9007199254740995" }]);
  const reads = await page.evaluate(() => window.reads);
  expect(reads.filter((read) => read.method === "POST")).toEqual([{
    path: "/namespaces/workspace/outcomes/outcome/workspace-view", method: "POST",
    body: { subject: { kind: "entity", ref: { ...scope, kind: "work_item", id: "selected" } },
      placement_preview: { parent_id: "after" } },
  }]);
  await destination(page, "Alternative branch");
  await expect(page.getByLabel("I reviewed the new placement and its blocking impacts", { exact: true })).toBeDisabled();
  await page.getByRole("button", { name: "Save move", exact: true }).click();
  expect(await page.evaluate(() => window.submits.length)).toBe(1);
});

test("bounded Objective impact is explicit and a blocked destination never becomes a command", async ({ page }) => {
  await fixture(page, "objective");
  await destination(page, "New branch");
  await page.evaluate(() => { window.partial = true; window.blocked = true; });
  await page.getByRole("button", { name: "Preview move", exact: true }).click();
  await expect(page.locator(".workspace-move-preview")).toContainText("Objectives: 2 · Tasks: 8");
  await expect(page.locator(".workspace-move-preview")).toContainText("bounded preview");
  await expect(page.locator(".workspace-move-preview")).toContainText("Applicable blockers: 25");
  await expect(page.locator(".workspace-move-preview")).toContainText("Only the first recorded blockers are shown");
  await expect(page.locator(".workspace-move-preview")).toContainText(/objective subtree in execution/i);
  await expect(page.getByLabel("I reviewed the new placement and its blocking impacts", { exact: true })).toBeDisabled();
  await page.getByRole("button", { name: "Save move", exact: true }).click();
  expect(await page.evaluate(() => window.submits.length)).toBe(0);
});

test("a late preview for a different destination or scope cannot restore an acknowledged command", async ({ page }) => {
  await fixture(page);
  await destination(page, "New branch");
  await page.evaluate(() => { window.deferPreview = true; });
  await page.getByRole("button", { name: "Preview move", exact: true }).click();
  await expect.poll(() => page.evaluate(() => typeof window.releasePreview)).toBe("function");
  await destination(page, "Alternative branch");
  await page.evaluate(() => { window.releasePreview(); window.deferPreview = false; window.foreign = true; });
  await expect(page.locator(".workspace-move-preview")).toBeEmpty();
  await page.getByRole("button", { name: "Preview move", exact: true }).click();
  await expect(page.getByLabel("I reviewed the new placement and its blocking impacts", { exact: true })).toBeDisabled();
  await page.getByRole("button", { name: "Save move", exact: true }).click();
  expect(await page.evaluate(() => window.submits.length)).toBe(0);
});
