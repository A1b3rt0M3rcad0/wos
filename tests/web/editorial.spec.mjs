import { test, expect, openOutcome, intent, save } from "./ux-fixture.mjs";
import { mkdir, readFile } from "node:fs/promises";
import { join } from "node:path";
import { interactionAudit } from "./interaction-audit.mjs";
import {
  dimensions,
  expectProportional,
  expectOriginalPixels,
} from "./responsive-audit.mjs";

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
    fullPage: false,
    animations: "disabled",
  });
}

test("large viewports scale the reference composition without rearranging it", async ({
  page,
  wos,
}) => {
  test.setTimeout(180000);
  await page.emulateMedia({ reducedMotion: "reduce" });
  await seed(wos);
  for (const title of [
    "Keep decisions connected to their evidence",
    "Prepare a shared handoff for people and agents",
  ])
    await wos.command("create_outcome", {
      namespace_id: wos.namespace,
      title,
      desired_state: "People can resume the same shared work.",
      priority: "normal",
    });

  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto(wos.url + "/app/");
  await page.evaluate(() => document.fonts.ready);
  await expectOriginalPixels(page);
  const loginTargets = [
    ".topbar",
    ".brand-logo",
    ".login-layout",
    ".login-story h1",
    ".login-panel",
    "#token",
    "#login button",
  ];
  await page.setViewportSize({ width: 1440, height: 810 });
  const loginReference = await dimensions(page, loginTargets);
  for (const [width, height] of [
    [1920, 1080],
    [2560, 1440],
    [3840, 2160],
  ]) {
    await page.setViewportSize({ width, height });
    expectProportional(
      await dimensions(page, loginTargets),
      loginReference,
      width / 1440,
    );
  }

  await page.setViewportSize({ width: 1440, height: 1000 });
  await wos.login(page);
  await openOutcome(page, "Ship a calmer workspace");
  await page.evaluate(() => document.fonts.ready);
  for (const [width, height, font] of [
    [1440, 1000, ""],
    [1280, 800, ""],
    [768, 1024, "28px"],
    [390, 844, ""],
    [320, 740, ""],
  ]) {
    await page.setViewportSize({ width, height });
    await page.evaluate((font) => {
      document.documentElement.style.fontSize = font;
    }, font);
    await expectOriginalPixels(page);
    await intent(page, "New task");
    await expectOriginalPixels(page);
    await page.keyboard.press("Escape");
  }

  await page.setViewportSize({ width: 1440, height: 810 });
  await page.evaluate(() => scrollTo(0, 0));
  const targets = [
    ".topbar",
    ".brand-logo",
    ".sidebar",
    ".workspace-main",
    "#title",
    "#area-tabs",
    ".area-intro",
    "#area-title",
    ".area-vignette img",
    "#metrics",
    ".metric",
    ".metric-label",
    ".summary-grid",
    ".summary-purpose-card",
    ".summary-purpose",
    ".summary-quick button",
    ".status-chart",
    ".chart-total",
  ];
  const reference = await dimensions(page, targets);
  await intent(page, "New task");
  const formTargets = [
    "#command-dialog",
    "#command-dialog .dialog-heading",
    "#command-dialog .dialog-body",
    "#command-dialog footer",
    '#command-dialog input[type="text"]',
    "#fields select",
    "#submit-command",
  ];
  const formReference = await dimensions(page, formTargets);
  await page.keyboard.press("Escape");

  for (const [width, height, name] of [
    [1440, 1000, "27-scale-reference"],
    [1920, 1080, "28-scale-full-hd"],
    [2560, 1440, "29-scale-2k"],
    [3840, 2160, "30-scale-4k"],
  ]) {
    const factor = width / 1440;
    await page.setViewportSize({ width, height });
    await page.evaluate(() => scrollTo(0, 0));
    // Sidebar height follows the actual viewport; compare the 16:9 matrix.
    if (width > 1440)
      expectProportional(await dimensions(page, targets), reference, factor);
    await page.evaluate((factor) => scrollTo(0, 173 * factor), factor);
    await page.emulateMedia({ reducedMotion: "no-preference" });
    await page.mouse.move(0, 0);
    await page.evaluate(() => document.activeElement?.blur());
    await capture(page, name);
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.evaluate(() => scrollTo(0, 0));
    await intent(page, "New task");
    if (width > 1440)
      expectProportional(
        await dimensions(page, formTargets),
        formReference,
        factor,
      );
    await expectWholeControl(page.getByLabel("Initial state", { exact: true }));
    await expect(page.locator("#submit-command")).toBeInViewport();
    if (width === 2560) await capture(page, "31-scale-2k-task-form");
    await page.keyboard.press("Escape");
    for (const area of [
      "overview",
      "plan",
      "work",
      "issues",
      "evidence",
      "activity",
    ]) {
      await page.locator(`[data-area="${area}"]`).click();
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
        `${area} at ${width}px`,
      ).toBe(true);
    }
    await page.locator('[data-area="work"]').click();
    await page.locator('[data-view="board"]').click();
    expect(
      await page
        .locator(".board")
        .evaluate(
          (element) =>
            getComputedStyle(element).gridTemplateColumns.split(" ").length,
        ),
    ).toBe(5);
    if (width === 2560) {
      await page.locator(".board").scrollIntoViewIfNeeded();
      await page.mouse.move(0, 0);
      await page.evaluate(() => document.activeElement?.blur());
      await capture(page, "32-scale-2k-work-board");
    }
    await page.locator('[data-area="overview"]').click();
  }
  // The upper bound stops component growth even beyond a 4K-wide viewport.
  await page.setViewportSize({ width: 5120, height: 2880 });
  expect(
    await page
      .locator(".sidebar")
      .evaluate((element) => element.getBoundingClientRect().width),
  ).toBeCloseTo((244 * 3840) / 1440, 0);
  expect(
    await page
      .locator("html")
      .evaluate((element) => parseFloat(getComputedStyle(element).fontSize)),
  ).toBeCloseTo((14 * 3840) / 1440, 1);
});

test("buttons keep legible padded states across the workspace, dialogs and touch navigation", async ({
  page,
  browser,
  wos,
}) => {
  test.setTimeout(180000);
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.emulateMedia({ reducedMotion: "reduce" });
  await seed(wos);
  await page.goto(wos.url + "/app/");
  const desktop = interactionAudit(page);
  await desktop.audit();
  await wos.login(page);
  await openOutcome(page, "Ship a calmer workspace");
  await page.evaluate(() => document.fonts.ready);
  for (const area of [
    "overview",
    "plan",
    "work",
    "issues",
    "evidence",
    "activity",
  ]) {
    await page.locator(`[data-area="${area}"]`).click();
    await desktop.audit();
  }
  await page.locator('[data-area="work"]').click();
  await page.evaluate(() => scrollTo(0, 0));
  await page.locator('[data-area="issues"]').hover();
  await expect(page.locator('[data-area="work"]')).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  await expect(page.locator('[data-area="issues"]')).toHaveAttribute(
    "aria-pressed",
    "false",
  );
  expect(
    await page
      .locator('[data-area="issues"]')
      .evaluate((element) => getComputedStyle(element).borderBottomColor),
  ).not.toBe(
    await page
      .locator('[data-area="work"]')
      .evaluate((element) => getComputedStyle(element).borderBottomColor),
  );
  await capture(page, "22-area-tab-hover");
  const board = page.locator('#view-tabs button[data-view="board"]');
  await board.hover();
  await capture(page, "23-view-switch-hover");
  await desktop.selectionTransition("#view-tabs button", () => board.click());
  await desktop.selectionTransition("#view-tabs button", () =>
    page.locator('#view-tabs button[data-view="list"]').click(),
  );
  await board.click();
  await desktop.audit();
  const card = page.locator(".board .item").first();
  await card.hover();
  await capture(page, "24-board-card-hover");
  await card.click();
  await expect(page.locator("#detail-dialog")).toBeVisible();
  await desktop.audit(page.locator("#detail-dialog"));
  await page.keyboard.press("Escape");
  await intent(page, "New task");
  await page.getByText("Execution instructions", { exact: true }).click();
  const add = page.getByRole("button", {
    name: "Add instructions item",
    exact: true,
  });
  await add.click();
  const objective = page.getByRole("combobox", {
    name: "Objective",
    exact: true,
  });
  await objective.fill("A clear path");
  await expect(page.locator("#command-dialog [role=option]")).toHaveCount(1);
  await objective.press("ArrowDown");
  await objective.press("Enter");
  await page
    .locator("#command-dialog")
    .getByText("Advanced", { exact: true })
    .click();
  await desktop.audit(page.locator("#command-dialog"));
  await add.hover();
  await capture(page, "25-form-controls-hover");
  await page.keyboard.press("Escape");
  await page.locator("#administration").click();
  for (const operation of ["set_grant", "issue_credential"]) {
    await page.locator("#admin-operation").selectOption(operation);
    await desktop.audit(page.locator("#admin-dialog"));
  }
  await page.keyboard.press("Escape");
  await page.locator("#actions").click();
  await desktop.audit(page.locator("#action-dialog"));
  await page
    .locator("#action-dialog")
    .getByText("Advanced", { exact: true })
    .click();
  await desktop.audit(page.locator("#action-dialog"));
  await page.keyboard.press("Escape");
  const touch = await browser.newContext({
    viewport: { width: 390, height: 844 },
    isMobile: true,
    hasTouch: true,
    reducedMotion: "reduce",
  });
  try {
    const mobile = await touch.newPage();
    await wos.login(mobile);
    await openOutcome(mobile, "Ship a calmer workspace");
    const audit = interactionAudit(mobile);
    for (const width of [390, 320]) {
      await mobile.setViewportSize({ width, height: 844 });
      await mobile.locator('[data-area="work"]').click();
      await audit.audit();
      const inactive = mobile.locator('[data-area="issues"]');
      await mobile.mouse.move(0, 0);
      const before = await inactive.evaluate(
        (element) => getComputedStyle(element).backgroundColor,
      );
      await inactive.hover();
      expect(
        await inactive.evaluate(
          (element) => getComputedStyle(element).backgroundColor,
        ),
      ).toBe(before);
      await mobile.keyboard.press("Tab");
      await inactive.focus();
      expect(
        await inactive.evaluate(
          (element) => getComputedStyle(element).outlineWidth,
        ),
      ).toBe("3px");
      if (width === 390) await capture(mobile, "26-mobile-tabs-focus");
      expect(
        await mobile.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
    }
    await test.info().attach("interaction-audit", {
      body: JSON.stringify(
        { desktop: desktop.report, touch: audit.report },
        null,
        2,
      ),
      contentType: "application/json",
    });
  } finally {
    await touch.close();
  }
});

test("paper stays fixed behind scrolling content and the editorial sidebar remains usable", async ({
  page,
  wos,
}) => {
  await seed(wos);
  for (const title of [
    "Keep decisions connected to their evidence",
    "Prepare a shared handoff for people and agents",
  ])
    await wos.command("create_outcome", {
      namespace_id: wos.namespace,
      title,
      desired_state: "People can resume the same shared work.",
      priority: "normal",
    });
  await wos.login(page);
  await openOutcome(page, "Ship a calmer workspace");
  await page.evaluate(() => document.fonts.ready);
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 1000 });
    await page.emulateMedia({ reducedMotion: "no-preference" });
    await page.evaluate(() => scrollTo(0, 0));
    const clip = { x: width - 12, y: 700, width: 6, height: 120 };
    const before = await page.screenshot({ clip, animations: "disabled" });
    const shifted = await page.screenshot({
      clip: { ...clip, y: clip.y + 17 },
      animations: "disabled",
    });
    // Prove that this is visible lined paper, not two equal blank margins.
    expect(before.equals(shifted)).toBe(false);
    const headingBefore = await page.locator("#title").boundingBox();
    const sidebarBefore = await page.locator(".sidebar").boundingBox();
    if (width === 1440) await capture(page, "19-fixed-paper-top");
    await page.evaluate(() => scrollBy(0, 173));
    await expect.poll(() => page.evaluate(() => scrollY)).toBe(173);
    const headingAfter = await page.locator("#title").boundingBox();
    expect(headingBefore.y - headingAfter.y).toBe(173);
    expect(
      before.equals(await page.screenshot({ clip, animations: "disabled" })),
    ).toBe(true);
    if (width === 1440) {
      expect((await page.locator(".sidebar").boundingBox()).y).toBe(
        sidebarBefore.y,
      );
      await capture(page, "20-fixed-paper-scrolled");
    }
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.evaluate(() => scrollTo(0, 0));
    const staticBefore = await page.screenshot({
      clip,
      animations: "disabled",
    });
    await page.evaluate(() => scrollBy(0, 173));
    expect(
      staticBefore.equals(
        await page.screenshot({ clip, animations: "disabled" }),
      ),
    ).toBe(false);
  }
  await page.setViewportSize({ width: 390, height: 844 });
  await page.evaluate(() => scrollTo(0, 0));
  await page.locator("#navigation-toggle").click();
  await expect(page.locator("#navigation-toggle")).toHaveAttribute(
    "aria-expanded",
    "true",
  );
  await expect(page.locator("#outcomes .item")).toHaveCount(3);
  const next = page.locator("#outcomes .item").filter({
    hasText: "Prepare a shared handoff",
  });
  await next.focus();
  await capture(page, "21-mobile-outcome-navigation");
  await page.keyboard.press("Enter");
  await expect(page.locator("#title")).toHaveText(
    "Prepare a shared handoff for people and agents",
  );
  await expect(page.locator("#navigation-toggle")).toHaveAttribute(
    "aria-expanded",
    "false",
  );
  // More than one sidebar viewport: the final action remains reachable.
  for (let index = 0; index < 12; index++)
    await wos.command("create_outcome", {
      namespace_id: wos.namespace,
      title: `Shared notebook example ${index + 1}`,
      desired_state: "People can resume the same shared work.",
      priority: "normal",
    });
  await page.setViewportSize({ width: 1440, height: 700 });
  await page.locator("#search").evaluate((form) => form.requestSubmit());
  await expect(page.locator("#outcomes .item")).toHaveCount(15);
  await page.locator("#administration").focus();
  await expect(page.locator("#administration")).toBeInViewport();
  await page.keyboard.press("Enter");
  await expect(page.locator("#admin-dialog")).toBeVisible();
  await page.emulateMedia({ reducedMotion: "no-preference" });
  await page.locator("#admin-operation").selectOption("set_grant");
  const body = page.locator("#admin-dialog .dialog-body");
  await body.evaluate((element) => element.scrollTo(0, 0));
  const bounds = await body.boundingBox();
  const clip = { x: bounds.x + 8, y: bounds.y + 12, width: 6, height: 120 };
  const dialogPaper = await page.screenshot({ clip, animations: "disabled" });
  await body.evaluate((element) => element.scrollBy(0, 137));
  expect(await body.evaluate((element) => element.scrollTop)).toBeGreaterThan(
    0,
  );
  expect(
    dialogPaper.equals(await page.screenshot({ clip, animations: "disabled" })),
  ).toBe(true);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
});

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
  await expect(page.locator("#area-tabs button")).toHaveCount(7);
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
    await expect(page.locator("#area-index")).toContainText(
      area[0].toUpperCase() + area.slice(1),
    );
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
  await expect(page.locator("#detail-dialog")).toBeVisible();
  await expect(page.locator("#detail-title")).toHaveText(
    "Connect the evidence review",
  );
  await capture(page, "09-task-detail");
  await page.keyboard.press("Escape");
  await intent(page, "New task");
  await capture(page, "10-task-form");
  await page.getByText("Execution instructions", { exact: true }).click();
  await page
    .getByRole("button", { name: "Add instructions item", exact: true })
    .click();
  await page
    .getByLabel("Instructions item", { exact: true })
    .fill(
      "Review the shared state, make a bounded change, and attach evidence.",
    );
  await capture(page, "15-expanded-task-form");
  await page.keyboard.press("Escape");
  await page.locator("#actions").click();
  await expect(page.locator("#action-dialog")).toBeVisible();
  await capture(page, "16-action-menu");
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
      await intent(page, "New task");
      await capture(page, "17-mobile-task-form");
      await page.keyboard.press("Escape");
      await page.locator("#administration").click();
      await expect(page.locator("#admin-dialog")).toBeVisible();
      await capture(page, "18-mobile-settings");
      await page.keyboard.press("Escape");
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
      await expect(page.locator("noscript p")).toContainText(
        "Enable JavaScript",
      );
      await expect(page.locator("noscript .product-name")).toHaveText("WOS");
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

test("Woobe colors distinguish interaction copper, primary charcoal and semantic states", async ({
  page,
  wos,
}) => {
  await seed(wos);
  await page.goto(wos.url + "/app/");
  await expect(page.locator("body")).toHaveCSS(
    "background-color",
    "rgb(239, 237, 232)",
  );
  await expect(page.locator(".login-story h1 em")).toHaveCSS(
    "color",
    "rgb(167, 68, 29)",
  );
  const primary = page.locator("#login button");
  await expect(primary).toHaveCSS("background-color", "rgb(41, 38, 32)");
  await expect(primary).toHaveCSS("color", "rgb(255, 253, 249)");
  await primary.hover();
  await expect(primary).toHaveCSS("background-color", "rgb(64, 58, 50)");
  await page.locator("#token").focus();
  await page.keyboard.press("Tab");
  await expect(primary).toBeFocused();
  await expect(primary).toHaveCSS("outline-color", "rgb(167, 68, 29)");
  await wos.login(page);
  await openOutcome(page, "Ship a calmer workspace");
  await expect(page.locator("#outcomes .selected")).toHaveCSS(
    "background-color",
    "rgb(241, 223, 208)",
  );
  await page.locator('#area-tabs [data-area="work"]').click();
  await expect(page.locator(".list-row")).toHaveCount(5);
  await expect(
    page.locator(".list-row .badge").filter({ hasText: /^Done$/ }),
  ).toHaveCSS("color", "rgb(36, 104, 70)");
  await expect(
    page.locator(".list-row .badge").filter({ hasText: /^In progress$/ }),
  ).toHaveCSS("color", "rgb(134, 91, 19)");
  await expect(
    page.locator(".list-row .badge").filter({ hasText: /^Blocked$/ }),
  ).toHaveCSS("color", "rgb(178, 56, 52)");
});

test("all modal families share matte workspace paper on desktop and mobile", async ({
  page,
  wos,
}) => {
  await seed(wos);
  await wos.login(page);
  await openOutcome(page, "Ship a calmer workspace");

  async function paper(id) {
    const dialog = page.locator(id);
    await expect(dialog).toBeVisible();
    await expect(dialog).toHaveCSS("background-color", "rgb(239, 237, 232)");
    await expect(dialog).toHaveCSS("border-top-color", "rgb(212, 206, 196)");
    const surfaces = await dialog.evaluate((root) => {
      const color = (element) => getComputedStyle(element).backgroundColor;
      const colors = (selector) =>
        [...root.querySelectorAll(selector)].map(color);
      return {
        secondary: colors(
          "footer, fieldset, details, .field-context, .detail-section, .signed-protocol-panel, .contract-notice, .conflict-columns section",
        ),
        surface: colors(
          'input:not([type="checkbox"]), select, textarea, pre, .reference-results, .conclusion-card, .relation-card, .roadmap-edit-row, .contract-progress, .contract-delivery, .signed-review-card',
        ),
        white: [...root.querySelectorAll("*")]
          .filter((element) => color(element) === "rgb(255, 255, 255)")
          .map((element) => element.tagName + "." + element.className),
        backdrop: getComputedStyle(root, "::backdrop").backgroundColor,
        focusedInside: root.contains(document.activeElement),
      };
    });
    expect(
      surfaces.secondary.every((color) => color === "rgb(245, 243, 239)"),
    ).toBe(true);
    expect(
      surfaces.surface.every((color) => color === "rgb(255, 253, 249)"),
    ).toBe(true);
    expect(surfaces.white).toEqual([]);
    expect(surfaces.backdrop).toBe("rgba(41, 38, 32, 0.38)");
    expect(surfaces.focusedInside).toBe(true);
  }

  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: width === 1440 ? 1000 : 844 });
    await page.locator('#area-tabs [data-area="work"]').click();
    await page.locator("#actions").click();
    await paper("#action-dialog");
    await page.locator("#action-dialog summary").click();
    await paper("#action-dialog");
    if (width === 390) {
      await page.locator("#action-dialog").evaluate((dialog) => {
        dialog.scrollTop = 0;
      });
      await page.mouse.move(0, 0);
      await capture(page, "33-mobile-action-menu");
    }
    await page.keyboard.press("Escape");

    await intent(page, "New task");
    await page.getByText("Execution instructions", { exact: true }).click();
    await page
      .getByRole("button", { name: "Add instructions item", exact: true })
      .click();
    await paper("#command-dialog");
    await expectWholeControl(
      page.getByLabel("Instructions item", { exact: true }),
    );
    await page.keyboard.press("Escape");

    await page
      .locator(".list-row")
      .filter({ hasText: "Connect the evidence review" })
      .click();
    await paper("#detail-dialog");
    if (width === 390) await capture(page, "34-mobile-task-detail");
    await page.keyboard.press("Escape");

    await page.locator("#administration").click();
    for (const operation of [
      "set_grant",
      "issue_credential",
      "revoke_credential",
      "create_namespace",
    ]) {
      await page.locator("#admin-operation").selectOption(operation);
      await paper("#admin-dialog");
    }
    await page.keyboard.press("Escape");
    await page.locator("#developer-tools").click();
    await paper("#command-dialog");
    await capture(
      page,
      width === 1440 ? "35-technical-command" : "36-mobile-technical-command",
    );
    await page.keyboard.press("Escape");
  }
});

test("WOS uses the supplied hand-drawn logo alone and real browser icons", async ({
  page,
  wos,
}) => {
  await seed(wos);
  await page.goto(wos.url + "/app/");
  await expect(
    page.getByRole("link", { name: "WOS · home", exact: true }),
  ).toBeVisible();
  const logo = page.locator(".brand-logo");
  await expect(page.locator(".brand")).toHaveText("");
  await expect(page.locator(".brand-name, .brand-caption")).toHaveCount(0);
  await expect(logo).toHaveAttribute(
    "src",
    "brand/hand-drawn-chain/Hand-Drawn%20Orange%20Chain%20Icon.svg",
  );
  for (const width of [1440, 390, 320]) {
    await page.setViewportSize({ width, height: 1000 });
    await expect
      .poll(() => logo.evaluate((img) => img.complete && img.naturalWidth > 0))
      .toBe(true);
    const box = await logo.boundingBox();
    expect(box.width).toBeGreaterThanOrEqual(96);
    expect(box.width / box.height).toBeCloseTo(1897 / 632, 2);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  }
  await expect(logo).toHaveCSS("background-color", "rgba(0, 0, 0, 0)");
  const openings = await logo.evaluate((image) => {
    const canvas = document.createElement("canvas");
    canvas.width = image.naturalWidth;
    canvas.height = image.naturalHeight;
    const context = canvas.getContext("2d");
    context.drawImage(image, 0, 0);
    return [
      [450, 450],
      [800, 180],
      [1150, 450],
      [1500, 180],
    ].map(([x, y]) => context.getImageData(x, y, 1, 1).data[3]);
  });
  expect(openings).toEqual([0, 0, 0, 0]);
  const icons = await page
    .locator('link[rel~="icon"], link[rel="apple-touch-icon"]')
    .evaluateAll((links) => links.map((link) => link.href));
  expect(icons.length).toBeGreaterThanOrEqual(3);
  const decoded = await page.evaluate(
    async (urls) =>
      Promise.all(
        urls.map(async (url) => {
          const image = new Image();
          image.src = url;
          await image.decode();
          return image.naturalWidth > 0;
        }),
      ),
    icons,
  );
  expect(decoded.every(Boolean)).toBe(true);
  const favicon = await page.request.get(wos.url + "/app/favicon.svg");
  expect(favicon.ok()).toBe(true);
  expect(await favicon.body()).toEqual(
    await readFile(
      new URL(
        "../../packages/wos-api/web/assets/brand/hand-drawn-chain/Hand-Drawn Orange Chain Icon.svg",
        import.meta.url,
      ),
    ),
  );
  await page.setViewportSize({ width: 1440, height: 1000 });
  await wos.login(page);
  await expect(page.locator("#workspace-loading")).not.toBeVisible();
  await expect(page.locator('img[src^="brand/"]')).toHaveCount(1);
  let release;
  const held = new Promise((resolve) => {
    release = resolve;
  });
  await page.route("**/continuity?**", async (route) => {
    await held;
    await route.continue();
  });
  try {
    await page
      .locator("#outcomes button")
      .filter({ hasText: "Ship a calmer workspace" })
      .click();
    await expect(page.locator("#workspace-loading")).toBeVisible();
    await expect(page.locator("#workspace-loading")).toContainText(
      "Loading outcome",
    );
    await expect(page.locator(".loading-vignette")).toBeVisible();
    await expect(page.locator('img[src^="brand/"]')).toHaveCount(1);
    await capture(page, "14-outcome-loading");
  } finally {
    release();
  }
  await expect(page.locator("#workspace-loading")).not.toBeVisible();
  await expect(page.locator("#title")).toHaveText("Ship a calmer workspace");
  await page.locator('#area-tabs [data-area="overview"]').click();
  await expect(page.locator("#area-index")).toHaveText("01 / Overview");
});

// A focus target must fit inside the scroll region with space for its outline.
// Checking the viewport alone misses clipping by nested form containers/footers.
async function expectWholeControl(control) {
  await control.focus();
  const box = await control.evaluate((element) => {
    const b = element.getBoundingClientRect();
    const body = element.closest(".dialog-body");
    const region = body.getBoundingClientRect();
    const dialog = element.closest("dialog");
    const style = getComputedStyle(element);
    const footer = dialog.querySelector("footer").getBoundingClientRect();
    const hit = document.elementFromPoint(
      b.x + b.width / 2,
      b.y + b.height / 2,
    );
    return {
      top: b.top - region.top,
      bottom: region.bottom - b.bottom,
      left: b.left - region.left,
      right: region.right - b.right,
      height: b.height,
      covered: hit !== element && !element.contains(hit),
      footerOverlap: region.bottom - footer.top,
      extraDialogScroll: dialog.scrollHeight - dialog.clientHeight,
      extraPageWidth: document.documentElement.scrollWidth - innerWidth,
      focusClearance: Math.max(
        1,
        parseFloat(style.outlineWidth) + parseFloat(style.outlineOffset),
      ),
    };
  });
  for (const edge of ["top", "bottom", "left", "right"])
    expect(box[edge], `${edge} border/focus clearance`).toBeGreaterThanOrEqual(
      box.focusClearance - 0.5,
    );
  expect(box.height).toBeGreaterThanOrEqual(44);
  expect(box.covered).toBe(false);
  expect(box.footerOverlap).toBeLessThanOrEqual(1);
  expect(box.extraDialogScroll).toBeLessThanOrEqual(1);
  expect(box.extraPageWidth).toBeLessThanOrEqual(1);
}

test("forms preserve whole controls and save actions through nested editors, compact screens and larger text", async ({
  page,
  wos,
}) => {
  await seed(wos);
  await wos.login(page);
  await openOutcome(page, "Ship a calmer workspace");
  for (const [width, height, font] of [
    [1440, 1000, 14],
    [390, 844, 14],
    [320, 740, 14],
    [740, 390, 14],
    [768, 1024, 28],
  ]) {
    await page.setViewportSize({ width, height });
    await page.evaluate((size) => {
      document.documentElement.style.fontSize = `${size}px`;
    }, font);
    await intent(page, "New task");
    const dialog = page.locator("#command-dialog");
    await page
      .getByLabel("Task title", { exact: true })
      .fill(`Form review ${width}`);
    await dialog.getByLabel("Priority", { exact: true }).selectOption("high");
    const lifecycle = page.getByLabel("Initial state", { exact: true });
    await lifecycle.selectOption("backlog");
    await page.keyboard.press("Tab");
    await expectWholeControl(lifecycle);
    await expect(page.locator("#submit-command")).toBeInViewport();
    await page.getByText("Execution instructions", { exact: true }).click();
    await page
      .getByRole("button", { name: "Add instructions item", exact: true })
      .click();
    const instruction = page.getByLabel("Instructions item", { exact: true });
    await instruction.fill("Keep the selected values and attach scoped proof.");
    await expectWholeControl(instruction);
    const picker = page.getByRole("combobox", {
      name: "Objective",
      exact: true,
    });
    await picker.fill("A clear path");
    await expect(
      dialog
        .getByRole("listbox", { name: "Objective", exact: true })
        .getByRole("option"),
    ).toHaveCount(1);
    await picker.press("ArrowDown");
    await picker.press("Enter");
    await expect(dialog.locator(".reference-chip")).toContainText(
      "A clear path",
    );
    await expectWholeControl(picker);
    await dialog.getByText("Advanced", { exact: true }).click();
    await expectWholeControl(dialog.locator('input[type="datetime-local"]'));
    await expect(lifecycle).toHaveValue("backlog");
    await save(page);
    await expect(page.locator("#notice")).not.toHaveClass(/error/);
    await expect(page.locator("#detail-dialog")).toBeVisible();
    await expect(page.locator("#detail-content")).toContainText(
      "Keep the selected values",
    );
    await page.keyboard.press("Escape");

    await page.locator("#administration").click();
    await expect(page.locator("#admin-dialog")).toBeVisible();
    await page.locator("#admin-operation").selectOption("set_grant");
    await expect(page.locator("#submit-admin")).toBeInViewport();
    await expectWholeControl(page.locator("#admin-operation"));
    const lastPermission = page.locator(".permission-options input").last();
    await lastPermission.focus();
    await lastPermission.check();
    await expect(lastPermission).toBeChecked();
    await expect(page.locator("#submit-admin")).toBeInViewport();
    await page.locator("#admin-operation").selectOption("issue_credential");
    await expectWholeControl(
      page.locator("#admin-dialog").getByLabel("Principal", { exact: true }),
    );
    await page.keyboard.press("Escape");
  }
});
