import { readFileSync } from "node:fs";
import { test, expect } from "playwright/test";

const assets = new URL("../../packages/wos-api/web/assets/", import.meta.url);
const scope = { namespace_id: "workspace", outcome_id: "outcome" };
const rules = {
  evidence_link_target_kinds: ["outcome", "objective", "work_item", "issue", "decision"],
  blocker_propagation: { outcome: ["subtree"], objective: ["direct", "subtree"], work_item: ["direct"] },
};
const ref = (kind, id) => ({ ...scope, kind, id });

async function fixture(page, name, options = {}) {
  await page.route("http://human-forms.test/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (/^\/assets\/[a-zA-Z0-9.-]+\.js$/.test(path)) return route.fulfill({
      contentType: "text/javascript", body: readFileSync(new URL(path.slice(8), assets), "utf8"),
    });
    return route.fulfill({ contentType: "text/html", body: '<!doctype html><html lang="en"><title>Typed human forms</title><body><form id="form"><div id="fields"></div><button type="submit">Save intent</button></form></body></html>' });
  });
  await page.goto("http://human-forms.test/");
  await page.evaluate(async ({ name, options, scope, rules }) => {
    const { humanForm } = await import("/assets/human-forms.js");
    const sourceVersion = String(options.entity?.version || "3");
    window.reads = [];
    window.submits = [];
    window.editor = humanForm(name, {
      scope, eligibility: rules, outcomeTitle: "Ship a usable release", ...options,
      request: async (path, options) => {
        if (options?.method && options.method !== "GET") {
          if (options.method !== "POST" || !path.endsWith("/workspace-view")) throw Error("A form must not write records");
          const query = JSON.parse(options.body);
          window.reads.push(path);
          return { subject: query.subject, entity_version: sourceVersion, outcome_revision: "19",
            placement_preview: { availability: "available", old_parent: { ref: { ...scope, kind: "outcome", id: scope.outcome_id }, title: "Ship a usable release" },
              new_parent: { ref: { ...scope, kind: query.placement_preview.parent_id ? "objective" : "outcome", id: query.placement_preview.parent_id || scope.outcome_id }, title: "Ship a usable release" },
              affected_objectives: 0, affected_tasks: 1, blocking_changes: [], coverage: { status: "complete" } } };
        }
        window.reads.push(path);
        return { items: [
          { ref: { ...scope, kind: "objective", id: "objective" }, title: "Installation works", version: 1 },
          { ref: { ...scope, kind: "evidence", id: "evidence" }, title: "Inspection notes", version: 1 },
          { ref: { ...scope, kind: "artifact", id: "artifact" }, title: "Patch", version: 1 },
          { ref: { ...scope, kind: "decision", id: "decision" }, title: "Approved direction", version: 1 },
        ] };
      },
    });
    document.querySelector("#fields").append(window.editor.node);
    document.querySelector("#form").onsubmit = async (event) => {
      event.preventDefault();
      window.submits.push({ command: window.editor.commandName(), value: await window.editor.get() });
    };
  }, { name, options, scope, rules });
}

test("quick creation keeps its named parent, creates planned work and supports a title-only intent", async ({ page }) => {
  await fixture(page, "create_work_item", {
    mode: "quick", entity: { id: "objective", _kind: "objective", title: "Installation works" },
  });
  await expect(page.getByText("Installation works", { exact: true })).toBeVisible();
  await expect(page.getByText("Planned in backlog", { exact: true })).toBeVisible();
  await expect(page.getByRole("combobox", { name: "New placement", exact: true })).toHaveCount(0);
  await page.getByLabel("Task title", { exact: true }).fill("Document the installation command");
  await page.getByLabel("Task title", { exact: true }).press("Enter");
  await expect.poll(() => page.evaluate(() => window.submits.length)).toBe(1);
  const intent = await page.evaluate(() => window.submits[0]);
  expect(intent.command).toBe("create_work_item");
  expect(intent.value).toMatchObject({ scope, title: "Document the installation command", lifecycle: "backlog", objective_id: "objective" });
  expect(intent.value.initial_criteria).toBeUndefined();
  expect(intent.value.execution_spec).toBeUndefined();
  expect(await page.evaluate(() => window.reads)).toEqual([]);
});

test("Objective obligations and criterion review mode are explicit human defaults", async ({ page }) => {
  await fixture(page, "create_objective", {
    mode: "quick", entity: { id: "parent", _kind: "objective", title: "Release the product" },
  });
  await page.getByLabel("Objective title", { exact: true }).fill("Install on a clean machine");
  await expect(page.getByLabel("Required for Outcome", { exact: true })).toBeChecked();
  expect(await page.evaluate(() => window.editor.get())).toMatchObject({ parent_objective_id: "parent", required_for_outcome: true });
  await page.getByLabel("Required for Outcome", { exact: true }).uncheck();
  await page.getByText("Details and success conditions", { exact: true }).click();
  await page.getByRole("button", { name: "Add success condition", exact: true }).click();
  await page.getByLabel("What must be true to consider this item satisfied?", { exact: true }).fill("A fresh environment starts with the documented command");
  await expect(page.getByLabel("How will it be verified?", { exact: true })).toHaveValue("evidence_review");
  const intent = await page.evaluate(() => window.editor.get());
  expect(intent.required_for_outcome).toBe(false);
  expect(intent.initial_criteria).toEqual([{ title: "A fresh environment starts with the documented command", description: "", required: true, verification_mode: "evidence_review" }]);
  expect(intent.lifecycle).toBeUndefined();
});

test("a full Task form makes availability a separate explicit choice", async ({ page }) => {
  await fixture(page, "create_work_item", { entity: { _kind: "outcome", id: "outcome" } });
  await page.getByLabel("Task title", { exact: true }).fill("Run the installation check");
  await expect(page.getByText("Choose an Objective", { exact: true })).toBeVisible();
  const missingObjective = await page.evaluate(async () => {
    try { await window.editor.get(); return "accepted"; } catch (error) { return error.message; }
  });
  expect(missingObjective).toContain("Choose an Objective");
  await expect(page.getByLabel("Task title", { exact: true })).toHaveValue("Run the installation check");
  await page.getByRole("button", { name: "Choose Objective", exact: true }).click();
  const placement = page.getByRole("combobox", { name: "New placement", exact: true });
  await placement.fill("Installation");
  const objectiveOption = page.getByRole("option").first();
  await expect(objectiveOption).toContainText("Installation works");
  await objectiveOption.click();
  expect(await page.evaluate(() => window.editor.get())).toMatchObject({ lifecycle: "backlog", objective_id: "objective" });
  await page.getByText("Advanced", { exact: true }).click();
  await page.getByLabel("Create as available", { exact: true }).check();
  const payload = await page.evaluate(() => window.editor.get());
  expect(payload.lifecycle).toBe("todo");
  expect(payload.claim_id).toBeUndefined();
});

test("a batch plan retains its draft when a Task still needs an Objective", async ({ page }) => {
  await fixture(page, "create_plan_items", {
    entity: { _kind: "outcome", id: "outcome", title: "Ship a usable release" },
  });
  await page.getByLabel("One title per line", { exact: true }).fill("Installation works\nRun the installation check");
  await page.getByRole("button", { name: "Preview plan", exact: true }).click();
  const rows = page.locator(".workspace-plan-row");
  await rows.nth(1).getByLabel("Item type", { exact: true }).selectOption("work_item");

  const invalid = await page.evaluate(async () => {
    try { await window.editor.get(); return "accepted"; } catch (error) { return error.message; }
  });
  expect(invalid).toContain("Every Task needs a parent Objective");
  await expect(rows.nth(0).getByLabel("Title", { exact: true })).toHaveValue("Installation works");
  await expect(rows.nth(1).getByLabel("Title", { exact: true })).toHaveValue("Run the installation check");

  await rows.nth(1).getByLabel("Parent Objective", { exact: true }).selectOption("local:item-1");
  expect(await page.evaluate(() => window.editor.get())).toMatchObject({
    items: [
      { local_key: "item-1", kind: "objective", title: "Installation works" },
      { local_key: "item-2", kind: "work_item", title: "Run the installation check", parent_local_key: "item-1" },
    ],
  });
});

test("replacement setters preserve exact versions and send explicit date, list and parent clears", async ({ page }) => {
  await fixture(page, "set_work_item_not_before", {
    entity: { _kind: "work_item", id: "task", version: "9007199254740993", not_before: "2026-10-20T12:00:00Z" },
  });
  await page.getByLabel("Scheduled start (local time)", { exact: true }).fill("");
  expect(await page.evaluate(() => window.editor.get())).toEqual({ scope, work_item_id: "task", expected_version: "9007199254740993", not_before: null });
  await fixture(page, "set_work_item_execution_spec", { entity: { _kind: "work_item", id: "task", version: "2" } });
  expect(await page.evaluate(() => window.editor.get())).toMatchObject({ expected_version: "2", execution_spec: { instructions: [], constraints: [], deliverables: [], scope_hints: [], context_refs: [] } });
  await fixture(page, "set_work_item_objective", { entity: { _kind: "work_item", id: "task", version: "3" } });
  expect(await page.evaluate(async () => {
    try { await window.editor.get(); return "accepted"; } catch (error) { return error.message; }
  })).toContain("Preview this destination");
  await page.getByRole("button", { name: "Preview move", exact: true }).click();
  await page.getByLabel("I reviewed the new placement and its blocking impacts", { exact: true }).check();
  expect(await page.evaluate(() => window.editor.get())).toEqual({ scope, work_item_id: "task", expected_version: "3", expected_outcome_revision: "19", objective_id: null });
});

test("a documentary note contains authenticated-context inputs rather than a state transition", async ({ page }) => {
  await fixture(page, "append_entity_note", { entity: { _kind: "work_item", id: "task", title: "Run checks", lifecycle: "done" } });
  await page.getByLabel("Update", { exact: true }).fill("I finished the Task <script>window.executed = true</script>");
  const value = await page.evaluate(() => window.editor.get());
  expect(value).toEqual({ scope, subject_ref: ref("work_item", "task"), type: "update", body: "I finished the Task <script>window.executed = true</script>", artifact_refs: [], evidence_refs: [] });
  for (const field of ["lifecycle", "principal_id", "actor_ref", "recorded_at", "contract_id", "expected_version"])
    expect(value[field]).toBeUndefined();
  expect(await page.evaluate(() => window.executed)).toBeUndefined();
  await page.getByLabel("Update", { exact: true }).fill("x".repeat(16385));
  expect(await page.evaluate(async () => {
    try { await window.editor.get(); return "accepted"; } catch (error) { return error.message; }
  })).toContain("16 KiB");
});

test("contextual evidence fixes the criterion and uses one explicit compound intent without assessment", async ({ page }) => {
  await fixture(page, "register_evidence_with_links", {
    entity: { _kind: "objective", id: "objective", title: "Installation works", version: "9007199254740993" },
    criterion: { id: "condition", title: "Fresh machine starts", criterion_revision: "9007199254740995" },
  });
  await page.getByLabel("Description", { exact: true }).fill("Observed the documented command on a fresh machine");
  await page.getByLabel("Source URL", { exact: true }).fill("https://example.test/checks/12");
  await page.getByLabel("Why is this relevant?", { exact: true }).fill("Records the environment and its startup output");
  await expect(page.getByLabel("Evidence stance", { exact: true })).toHaveValue("supports");
  const value = await page.evaluate(() => window.editor.get());
  expect(value.links).toEqual([{ target_ref: ref("objective", "objective"), criterion_id: "condition", criterion_revision: "9007199254740995", target_expected_version: "9007199254740993", stance: "supports", rationale: "Records the environment and its startup output" }]);
  expect(value.evidence.source_ref).toEqual({ provider: "external", uri: "https://example.test/checks/12" });
  expect(value.assessment).toBeUndefined();
  expect(await page.evaluate(() => window.reads)).toEqual([]);
  await page.getByLabel("Evidence source", { exact: true }).selectOption("observation");
  const observed = await page.evaluate(() => window.editor.get());
  expect(observed.evidence.evidence_type).toBe("attestation");
  expect(observed.evidence.source_ref).toBeUndefined();
  expect(observed.observation).toMatchObject({ subject_ref: ref("objective", "objective"), criterion_id: "condition", body: value.evidence.description });
});

test("EvidenceLink search and blocking scope consume server-defined eligibility", async ({ page }) => {
  await fixture(page, "create_evidence_link", { entity: { _kind: "evidence", id: "evidence" } });
  await page.getByRole("combobox", { name: "Related item", exact: true }).focus();
  await expect.poll(() => page.evaluate(() => window.reads.length)).toBeGreaterThan(0);
  expect(await page.evaluate(() => new URL(window.reads[0], location.origin).searchParams.get("kind"))).toBe("outcome,objective,work_item,issue,decision");
  await fixture(page, "create_issue", { entity: { _kind: "work_item", id: "task", title: "Check startup" } });
  await page.getByLabel("Issue title", { exact: true }).fill("Missing environment variable");
  const unblocked = await page.evaluate(() => ({ name: window.editor.commandName(), value: window.editor.get() }));
  expect(unblocked.name).toBe("create_issue");
  expect(await page.evaluate(() => window.editor.get())).toMatchObject({ affected_refs: [ref("work_item", "task")] });
  await page.getByLabel("Prevents continuing?", { exact: true }).check();
  await page.getByLabel("Blocking impact", { exact: true }).fill("The server cannot start");
  expect(await page.getByLabel("Propagation", { exact: true }).locator("option").evaluateAll((options) => options.map((option) => option.value))).toEqual(["direct"]);
  const blocked = await page.evaluate(async () => ({ name: window.editor.commandName(), value: await window.editor.get() }));
  expect(blocked).toMatchObject({ name: "report_issue_with_blocker", value: { blocked_ref: ref("work_item", "task"), propagation: "direct" } });
});

test("contextual Artifact and Decision drafts carry canonical documentary relationships", async ({ page }) => {
  const entity = { _kind: "work_item", id: "task", title: "Implement the endpoint" };
  await fixture(page, "register_artifact_in_context", { entity });
  await page.getByLabel("Artifact name", { exact: true }).fill("Refund implementation");
  await page.getByLabel("Artifact type", { exact: true }).fill("patch");
  await page.getByLabel("URL", { exact: true }).fill("https://example.test/pull/1");
  await page.getByLabel("Relationship to this item", { exact: true }).selectOption("produces");
  expect(await page.evaluate(() => window.editor.get())).toEqual({ scope, subject_ref: ref("work_item", "task"), relationship: "produces", artifact: { name: "Refund implementation", artifact_type: "patch", uri: "https://example.test/pull/1" } });
  await fixture(page, "propose_decision_in_context", { entity });
  await page.getByLabel("Decision title", { exact: true }).fill("Choose the refund policy");
  await page.getByLabel("Proposal", { exact: true }).fill("Allow partial refunds");
  await page.getByLabel("Chosen alternative", { exact: true }).fill("Partial refund");
  await page.getByLabel("Rationale", { exact: true }).fill("Matches the agreed acceptance condition");
  const value = await page.evaluate(() => window.editor.get());
  expect(value.decision).toMatchObject({ title: "Choose the refund policy", proposal: "Allow partial refunds" });
  expect(value.title).toBeUndefined();
  expect(value.target_ref).toBeUndefined();
});

test("batch preview is explicit, editable and cannot silently accept a changed title list", async ({ page }) => {
  await fixture(page, "create_plan_items", { batchKind: "objective", entity: { _kind: "outcome", id: "outcome" } });
  await page.getByLabel("One title per line", { exact: true }).fill("API works\nDocument installation\nRun checks");
  await page.getByRole("button", { name: "Preview plan", exact: true }).click();
  const rows = page.locator(".workspace-plan-row");
  await expect(rows).toHaveCount(3);
  await rows.nth(1).getByLabel("Item type", { exact: true }).selectOption("work_item");
  await rows.nth(1).getByLabel("Parent Objective", { exact: true }).selectOption("local:item-1");
  const value = await page.evaluate(() => window.editor.get());
  expect(value.items[1]).toMatchObject({ kind: "work_item", parent_local_key: "item-1" });
  await page.getByLabel("One title per line", { exact: true }).fill("Changed plan");
  expect(await page.evaluate(async () => {
    try { await window.editor.get(); return "accepted"; } catch (error) { return error.message; }
  })).toContain("Preview it again");
});
