import { test, expect, openOutcome, intent } from "./ux-fixture.mjs";
import { mkdir, readFile } from "node:fs/promises";
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
    await expect(page.locator("#area-index .product-name")).toHaveText("WOS");
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

test("supplied WOS identity stays legible, serves real browser icons and marks only actual loading", async ({
  page,
  wos,
}) => {
  await seed(wos);
  await page.goto(wos.url + "/app/");
  await expect(
    page.getByRole("link", { name: "WOS · home", exact: true }),
  ).toBeVisible();
  const logo = page.locator(".brand-logo");
  const name = page.locator(".brand-name");
  await expect(logo).toHaveAttribute(
    "src",
    "brand/01-marca/wos-symbol-primary.svg",
  );
  await expect(name).toHaveText("WOS");
  for (const width of [1440, 390, 320]) {
    await page.setViewportSize({ width, height: 1000 });
    await expect
      .poll(() => logo.evaluate((img) => img.complete && img.naturalWidth > 0))
      .toBe(true);
    const box = await logo.boundingBox();
    expect(box.width).toBeGreaterThanOrEqual(48);
    expect(box.width / box.height).toBeCloseTo(194 / 82, 1);
    await expect(name).toBeVisible();
    await page.evaluate(() => document.fonts.ready);
    const type = await name.evaluate((element) => {
      const style = getComputedStyle(element);
      return {
        family: style.fontFamily,
        weight: style.fontWeight,
        heading: getComputedStyle(document.querySelector(".login-story h1"))
          .fontFamily,
        loaded: document.fonts.check('36px "Instrument Serif"'),
      };
    });
    expect(type.family).toBe(type.heading);
    expect(type.family).toContain("Instrument Serif");
    expect(type.weight).toBe("400");
    expect(type.loaded).toBe(true);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  }
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
        "../../packages/wos-api/web/assets/brand/03-favicon-app/favicon.svg",
        import.meta.url,
      ),
    ),
  );
  await page.setViewportSize({ width: 1440, height: 1000 });
  await wos.login(page);
  await expect(page.locator("#workspace-loading")).not.toBeVisible();
  await expect(page.locator(".workspace-symbol")).toHaveAttribute(
    "src",
    "brand/04-navegacao/wos-sidebar-light.svg",
  );
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
    await expect(page.locator(".loading-symbol")).toBeVisible();
    await capture(page, "14-outcome-loading");
  } finally {
    release();
  }
  await expect(page.locator("#workspace-loading")).not.toBeVisible();
  await expect(page.locator("#title")).toHaveText("Ship a calmer workspace");
  const chapterName = page.locator("#area-index .product-name");
  await expect(chapterName).toHaveText("WOS");
  expect(
    await chapterName.evaluate((element) => getComputedStyle(element).fontFamily),
  ).toBe(await name.evaluate((element) => getComputedStyle(element).fontFamily));
});
