import { test, expect, openOutcome, intent } from "./ux-fixture.mjs";
import { mkdir } from "node:fs/promises";
import { join } from "node:path";

// Documentation captures use an isolated, explicitly illustrative database.
// Every displayed record is created through the public command API.
async function seed(wos) {
  const outcome = await wos.command("create_outcome", {
    namespace_id: wos.namespace,
    title: "Ship a calmer workspace",
    desired_state:
      "People and agents can pick up the same work, understand what matters, and verify the result.",
    priority: "normal",
  });
  const scope = { namespace_id: wos.namespace, outcome_id: outcome.id };
  await wos.command("add_criterion", {
    owner: { ...scope, kind: "outcome", id: outcome.id },
    expected_version: outcome.version,
    title: "People can plan, coordinate, and review a delivery",
    required: true,
    verification_mode: "evidence_review",
  });
  await wos.command("activate_outcome", {
    scope,
    expected_version: outcome.version + 1,
  });
  const objective = await wos.command("create_objective", {
    scope,
    title: "A clear path from intent to evidence",
    priority: "high",
    description: "Keep the plan, the work, and its proof connected.",
    required_for_outcome: true,
  });
  await wos.command("create_objective", {
    scope,
    title: "A workspace that travels across sessions",
    priority: "normal",
    description: "Make context recoverable for people and agents.",
  });
  const tasks = [];
  for (const [title, priority] of [
    ["Map the first-time experience", "high"],
    ["Refine the six workspace areas", "normal"],
    ["Connect the evidence review", "high"],
    ["Review keyboard navigation", "normal"],
    ["Publish the design notes", "low"],
  ])
    tasks.push(
      await wos.command("create_work_item", {
        scope,
        title,
        priority,
        lifecycle: "todo",
        objective_id: objective.id,
        description: "A bounded step toward a calmer, verifiable workspace.",
      }),
    );
  await wos.command("claim_work_item", {
    scope,
    work_item_id: tasks[1].id,
    expected_version: 1,
    ttl_seconds: 900,
  });
  const claim = await wos.command("claim_work_item", {
    scope,
    work_item_id: tasks[0].id,
    expected_version: 1,
    ttl_seconds: 900,
  });
  await wos.command("complete_work_item", {
    scope,
    work_item_id: tasks[0].id,
    expected_version: claim.version,
    claim_id: claim.current_lease.claim_id,
    fencing_token: claim.current_lease.fencing_token,
    reason: "The journey is documented.",
    result_summary: "A reviewed map of the first-time experience.",
  });
  const issue = await wos.command("create_issue", {
    scope,
    title: "The release checklist needs an independent review",
    description:
      "Keep publication on hold until a reviewer confirms the checklist.",
    severity: "major",
  });
  await wos.command("create_blocker", {
    scope,
    blocked_ref: { ...scope, kind: "work_item", id: tasks[4].id },
    cause_ref: { ...scope, kind: "issue", id: issue.id },
    description: "Await an independent review of the checklist",
    propagation: "direct",
  });
  for (const description of [
    "Keyboard navigation observations",
    "First-time experience review",
    "HTTP and MCP coordination notes",
  ])
    await wos.command("register_evidence", {
      scope,
      evidence_type: "test_result",
      description,
      source_ref: {
        provider: "design-example",
        id: description.toLowerCase().replaceAll(" ", "-"),
      },
      captured_at: "2026-10-15T12:00:00Z",
    });
  await wos.command("create_roadmap", {
    scope,
    title: "From first sketch to shared workspace",
    plan_scope: { kind: "outcome", id: outcome.id },
  });
  return scope;
}

async function capture(page, name) {
  const directory = process.env.WOS_DESIGN_SCREENSHOTS;
  if (!directory) return;
  await mkdir(directory, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await expect
    .poll(() =>
      page
        .locator("img")
        .evaluateAll((imgs) =>
          imgs
            .filter((img) => img.checkVisibility())
            .every((img) => img.complete && img.naturalWidth > 0),
        ),
    )
    .toBe(true);
  await page.screenshot({
    path: join(directory, name + ".png"),
    fullPage: true,
    animations: "disabled",
  });
}

test("editorial chapters keep all six areas, real records and illustration assets", async ({
  page,
  wos,
}) => {
  test.setTimeout(120000);
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    // A signed-out namespace discovery is expected to return 401 before login.
    const discovery = message.location().url?.endsWith("/api/v1/namespaces");
    if (
      message.type() === "error" &&
      !(discovery && message.text().includes("401"))
    )
      errors.push(message.text());
  });
  await seed(wos);
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto(wos.url + "/app/");
  await capture(page, "01-login");
  await wos.login(page);
  await openOutcome(page, "Ship a calmer workspace");
  await expect(page.locator("#area-tabs button")).toHaveCount(6);
  for (const [area, title, image, name] of [
    [
      "overview",
      "The whole picture.",
      "14_observer_metrics.svg",
      "02-overview",
    ],
    ["plan", "A plan with purpose.", "01_planner_map.svg", "03-plan"],
    ["work", "Make work move.", "05_coordinator_kanban.svg", "04-work"],
    ["issues", "Clear the way.", "16_blocker_sentinel.svg", "06-issues"],
    [
      "evidence",
      "Proof before conclusions.",
      "06_researcher_books.svg",
      "07-evidence",
    ],
    [
      "activity",
      "Every step, remembered.",
      "09_documenter_notes.svg",
      "08-activity",
    ],
  ]) {
    await page.locator(`[data-area="${area}"]`).click();
    await expect(page.locator("#area-title")).toHaveText(title);
    await expect(page.locator("#area-robot")).toHaveAttribute(
      "src",
      `robots/${image}`,
    );
    if (area !== "overview")
      await expect(page.locator("#content .content-heading h2")).toHaveText(
        {
          plan: "Objectives",
          work: "Tasks",
          issues: "Issues",
          evidence: "Evidence",
          activity: "Activity",
        }[area],
      );
    await capture(page, name);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  }
  await page.locator('[data-area="work"]').click();
  await page.getByRole("button", { name: "Board", exact: true }).click();
  await expect(page.locator(".board")).toBeVisible();
  await capture(page, "05-work-board");
  await page.getByRole("button", { name: "List", exact: true }).click();
  await expect(page.locator(".list-row")).toHaveCount(5);
  await page
    .locator(".list-row")
    .filter({ hasText: "Connect the evidence review" })
    .click();
  await capture(page, "09-task-detail");
  await page.keyboard.press("Escape");
  await intent(page, "New task");
  await capture(page, "10-task-form");
  await page.keyboard.press("Escape");
  await page.locator("#administration").click();
  await expect(page.locator("#admin-dialog")).toBeVisible();
  await capture(page, "11-settings");
  expect(errors).toEqual([]);
});

test("editorial layout reflows, respects reduced motion and survives missing fonts and images", async ({
  page,
  wos,
}) => {
  test.setTimeout(120000);
  await seed(wos);
  await wos.login(page);
  await openOutcome(page, "Ship a calmer workspace");
  await page.emulateMedia({ reducedMotion: "reduce" });
  for (const [width, height] of [
    [1920, 1080],
    [1440, 900],
    [1280, 800],
    [768, 1024],
    [390, 844],
    [360, 740],
    [320, 740],
  ]) {
    await page.setViewportSize({ width, height });
    for (const area of [
      "overview",
      "plan",
      "work",
      "issues",
      "evidence",
      "activity",
    ]) {
      await page.locator(`[data-area="${area}"]`).click();
      await expect(page.locator("#area-title")).toBeVisible();
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
        `${area} at ${width}px`,
      ).toBe(true);
    }
    await page.locator('[data-area="overview"]').click();
    if (width === 390) await capture(page, "12-mobile-overview");
    if (width === 390) {
      await page.locator('[data-area="work"]').click();
      await capture(page, "13-mobile-work");
      await page.locator('[data-area="overview"]').click();
    }
  }
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.evaluate(() => {
    document.documentElement.style.fontSize = "28px";
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.route(/\.(woff2|svg)(\?|$)/, (route) => route.abort());
  await page.reload();
  await expect(page.locator("#area-title")).toBeVisible();
  await page.locator('[data-area="work"]').click();
  await expect(page.locator(".list-row")).toHaveCount(5);
  await intent(page, "New task");
  await expect(page.getByLabel("Task title", { exact: true })).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
});

test("static sign-in keeps its narrative and explains the no-JavaScript fallback", async ({
  browser,
  wos,
}) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  try {
    const page = await context.newPage();
    await page.goto(wos.url + "/app/");
    for (const width of [1440, 390, 320]) {
      await page.setViewportSize({ width, height: 900 });
      await expect(page.getByRole("heading", { level: 1 })).toContainText(
        "Meaningful outcomes.",
      );
      await expect(page.locator("noscript p")).toBeVisible();
      await expect(page.locator("noscript p")).toContainText("Enable JavaScript");
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
    }
  } finally {
    await context.close();
  }
});
