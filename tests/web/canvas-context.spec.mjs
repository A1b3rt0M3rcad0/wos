import { readFileSync } from "node:fs";
import { test, expect } from "playwright/test";

const assets = new URL("../../packages/wos-api/web/assets/", import.meta.url);
const scope = { namespace_id: "namespace", outcome_id: "outcome" };
const ref = (kind, id) => ({ ...scope, kind, id });

async function fixture(page) {
  await page.route("http://canvas-context.test/**", async (route) => {
    const pathname = new URL(route.request().url()).pathname;
    if (/^\/assets\/[a-zA-Z0-9.-]+\.js$/.test(pathname)) {
      await route.fulfill({
        contentType: "text/javascript",
        body: readFileSync(new URL(pathname.slice("/assets/".length), assets), "utf8"),
      });
    } else {
      await route.fulfill({
        contentType: "text/html",
        body: '<!doctype html><html lang="en"><title>Task context</title><body><p id="canvas">Canvas stays mounted</p><main id="context"></main></body></html>',
      });
    }
  });
  await page.goto("http://canvas-context.test/");
  await page.evaluate(async () => {
    window.renderContext = (await import("/assets/canvas-context.js")).renderCanvasContext;
    window.references = [];
    window.histories = [];
  });
}

test("Task context keeps source text and presents dependencies, Blockers and evidence without accepting work", async ({ page }) => {
  await fixture(page);
  const work = {
    id: "task", scope,
    execution_spec: {
      instructions: ['Keep <script>window.authoredRan = true</script> as authored text'],
      constraints: ["Use the existing API"], deliverables: ["A reviewed patch"],
      context_refs: [ref("objective", "objective")],
    },
  };
  const context = {
    work_item: work,
    outcome: { ref: ref("outcome", "outcome"), title: "Ship the API" },
    objective: { id: "objective", scope, title: "Review the change" },
    outcome_revision: "9007199254740993", evaluated_at: "2026-10-16T12:00:00Z",
    dependencies: [
      { id: "hard", scope, relation_type: "depends_on", strength: "hard", lifecycle: "active", target_ref: ref("work_item", "prerequisite") },
      { id: "advisory", scope, relation_type: "depends_on", strength: "advisory", lifecycle: "active", target_ref: ref("objective", "guide") },
    ],
    blocking: {
      is_blocked: true,
      active_blockers: [{
        inherited: true, inherited_from: ref("objective", "objective"),
        blocker: { id: "blocker", scope, lifecycle: "active", description: "Approval is missing", cause_ref: ref("issue", "issue") },
      }],
    },
    evidence_links: [{ scope, evidence_id: "evidence", target_ref: ref("work_item", "task"), stance: "context", rationale: "Design notes only", lifecycle: "active" }],
  };
  await page.evaluate(({ context, work }) => window.renderContext(document.querySelector("#context"), {
    entity: work, context,
    attention: { status: "awaiting_review", awaiting_review: true, outcome_revision: "9007199254740993", obligations: [{ description: "An independent reviewer must assess this result", status: "awaiting_independent_review" }] },
    onReference: (value) => window.references.push(value),
    onHistory: (value) => window.histories.push(value),
  }), { context, work });
  await expect(page.locator("#context")).toContainText("Revision 9007199254740993");
  await expect(page.getByRole("heading", { name: "Instructions", exact: true })).toBeVisible();
  await expect(page.locator("#context")).toContainText(work.execution_spec.instructions[0]);
  expect(await page.evaluate(() => window.authoredRan)).toBeUndefined();
  await expect(page.getByText("Hard prerequisite", { exact: true })).toBeVisible();
  await expect(page.getByText("Advisory dependency", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Blocking impact", exact: true })).toBeVisible();
  await expect(page.getByText("Blocked: Yes", { exact: true })).toBeVisible();
  await expect(page.getByText("Context", { exact: true })).toBeVisible();
  await expect(page.locator("#context")).toContainText("not an assessment or acceptance decision");
  await expect(page.locator("#context")).toContainText("does not by itself complete the Task");
  await page.getByRole("button", { name: /^Issue / }).click();
  await page.getByRole("button", { name: "Load task history", exact: true }).click();
  expect(await page.evaluate(() => window.references)).toEqual([ref("issue", "issue")]);
  expect(await page.evaluate(() => window.histories)).toEqual([ref("work_item", "task")]);
  await expect(page.locator("#canvas")).toHaveText("Canvas stays mounted");
});

test("bounded summaries declare local and server omissions and replace stale context", async ({ page }) => {
  await fixture(page);
  await page.evaluate(({ scope, target }) => window.renderContext(document.querySelector("#context"), {
    entity: { id: "task", scope },
    context: {
      work_item: { id: "task", scope, execution_spec: { instructions: ["x".repeat(1300)] } },
      dependencies: Array.from({ length: 12 }, (_, i) => ({ id: `dependency-${i}`, relation_type: "depends_on", strength: "advisory", target_ref: target })),
      truncated: true, omitted: { dependencies: 7, assessment_history: 4 }, outcome_revision: "9007199254740993",
    },
    attention: { evidence_search_complete: false, truncated: true, omitted: { evidence_links: 2 } },
  }), { scope, target: ref("work_item", "target") });
  await expect(page.getByText("Advisory dependency", { exact: true })).toHaveCount(10);
  await expect(page.locator("#context")).toContainText("2 additional relations not shown");
  await expect(page.locator("#context")).toContainText("Instructions: 100 characters not shown");
  await expect(page.locator("#context")).toContainText("Context: 7 dependencies omitted by the server");
  await expect(page.locator("#context")).toContainText("Attention: 2 evidence links omitted by the server");
  await expect(page.locator("#context")).toContainText("evidence search is incomplete");
  await page.evaluate((scope) => window.renderContext(document.querySelector("#context"), {
    entity: { id: "task", scope, execution_spec: { instructions: ["Stale instructions"] } },
    context: { work_item: { id: "task", scope }, dependencies: [], evidence_links: [], outcome_revision: 9007199254740992 },
  }), scope);
  await expect(page.locator("#context")).not.toContainText("Stale instructions");
  await expect(page.getByText("Advisory dependency", { exact: true })).toHaveCount(0);
  await expect(page.locator("#context")).toContainText("does not include the complete historical record");
  await expect(page.locator("#context")).toContainText("Exact revision unavailable");
  await expect(page.locator("#context")).not.toContainText("9007199254740992");
});

test("Evidence form submits the canonical context stance through its normal typed picker", async ({ page }) => {
  await fixture(page);
  await page.evaluate(async ({ scope, target }) => {
    const { humanForm } = await import("/assets/human-forms.js");
    window.evidenceForm = humanForm("create_evidence_link", {
      scope, entity: { id: "evidence", _kind: "evidence" },
      request: async () => ({ items: [{ ref: target, title: "Target task", lifecycle: "todo" }] }),
    });
    document.querySelector("#context").append(window.evidenceForm.node);
  }, { scope, target: ref("work_item", "task") });
  await page.getByRole("combobox", { name: "Related item", exact: true }).focus();
  await page.getByRole("option", { name: /^Target task/ }).click();
  await page.getByLabel("Evidence stance", { exact: true }).selectOption("context");
  await page.getByLabel("Rationale", { exact: true }).fill("Useful background, not proof of completion");
  expect(await page.evaluate(() => window.evidenceForm.get())).toEqual({
    scope, evidence_id: "evidence", target_ref: ref("work_item", "task"),
    stance: "context", rationale: "Useful background, not proof of completion",
  });
  expect(await page.getByLabel("Evidence stance", { exact: true }).locator("option").evaluateAll((options) => options.map((option) => option.value))).toEqual(["supports", "contradicts", "context"]);
});
