import { nativeZoomBrowser } from "./native-browser-zoom.mjs";
import { test, expect, openOutcome } from "./ux-fixture.mjs";
import { mkdir, writeFile, readdir, readFile } from "node:fs/promises";
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { resolve } from "node:path";

const panel = (page) => page.locator("#canvas-details");
async function select(page, title) {
  const search = page.getByRole("combobox", {
    name: "Search Outcome Board",
    exact: true,
  });
  await search.fill(title);
  await page.getByRole("option").filter({ hasText: title }).first().click();
  await expect(panel(page).locator("#detail-title")).toHaveText(title);
}
async function setup(wos) {
  const { value: outcome, scope } = await wos.outcome("Payment integration");
  const objective = await wos.command("create_objective", {
    scope,
    title: "Verified payment API",
    priority: "normal",
    required_for_outcome: true,
  });
  const nested = await wos.command("create_objective", {
    scope,
    title: "Refund support",
    priority: "normal",
    required_for_outcome: false,
    parent_objective_id: objective.id,
  });
  const task = await wos.command("create_work_item", {
    scope,
    title: "Implement payment endpoint",
    priority: "normal",
    lifecycle: "backlog",
    objective_id: objective.id,
  });
  const direct = await wos.command("create_work_item", {
    scope,
    title: "Write release notes",
    priority: "normal",
    lifecycle: "backlog",
  });
  const cancelled = await wos.command("create_work_item", {
    scope,
    title: "Retired experiment",
    priority: "normal",
    lifecycle: "backlog",
    objective_id: objective.id,
  });
  await wos.command("cancel_work_item", {
    scope,
    work_item_id: cancelled.id,
    expected_version: cancelled.version,
    reason: "Removed from agreed scope",
  });
  return { outcome, scope, objective, nested, task, direct };
}
async function capture(page, name) {
  if (!process.env.WOS_CAPTURE_CONTEXTUAL_KANBAN) return;
  const root = resolve("../..");
  const source = execFileSync("git", ["rev-parse", "HEAD"], {
    cwd: root,
    encoding: "utf8",
  }).trim();
  expect(
    execFileSync(
      "git",
      ["status", "--porcelain", "--", "packages", "tests/web", "tools"],
      { cwd: root, encoding: "utf8" },
    ),
  ).toBe("");
  const binaryPath = resolve(root, "bin/wos");
  const binary = JSON.parse(
    execFileSync(binaryPath, ["--output", "json", "version"], {
      encoding: "utf8",
    }),
  ).data;
  expect(binary.commit).toBe(source);
  const output = resolve(
    process.env.WOS_CAPTURE_CONTEXTUAL_KANBAN,
    source.slice(0, 12),
  );
  await mkdir(output, { recursive: true });
  const hash = (value) => createHash("sha256").update(value).digest("hex");
  const assets = {};
  const assetRoot = resolve(root, "packages/wos-api/web/assets");
  for (const file of (await readdir(assetRoot, { recursive: true })).sort()) {
    if (!/\.(js|css)$/.test(file) && file !== "index.html") continue;
    const response = await page.request.get(
      new URL("/app/" + file, page.url()).href,
    );
    expect(response.ok()).toBe(true);
    const digest = hash(await response.body());
    expect(digest).toBe(hash(await readFile(resolve(assetRoot, file))));
    assets[file] = digest;
  }
  const buffer = await page.screenshot({
    fullPage: false,
    animations: "disabled",
  });
  await writeFile(resolve(output, name + ".png"), buffer, { flag: "wx" });
  await writeFile(
    resolve(output, name + ".json"),
    JSON.stringify(
      {
        sourceCommit: source,
        binary: { ...binary, sha256: hash(await readFile(binaryPath)) },
        testSha256: hash(
          await readFile(resolve(root, "tests/web/contextual-kanban.spec.mjs")),
        ),
        viewport: page.viewportSize(),
        browser: page.context().browser().version(),
        state: name,
        screenshotSha256: hash(buffer),
        servedAssetsSha256: assets,
        pixelSize: {
          width: buffer.readUInt32BE(16),
          height: buffer.readUInt32BE(20),
        },
      },
      null,
      2,
    ) + "\n",
    { flag: "wx" },
  );
}

for (const [width, height] of [
  [1024, 768],
  [1280, 720],
  [1366, 768],
  [1920, 1080],
  [2560, 1440],
  [3840, 2160],
  [390, 844],
  [320, 740],
  [430, 932],
]) {
  test(`contextual Kanban preserves Board, hierarchy, cancellation and keyboard at ${width}x${height}`, async ({
    page,
    wos,
  }) => {
    await page.setViewportSize({ width, height });
    const data = await setup(wos);
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    const mutations = [];
    await wos.login(page);
    await openOutcome(page, data.outcome.title, { area: null });
    await select(page, data.outcome.title);
    const disclosures = panel(page).locator(".workspace-compact > details");
    await expect(disclosures).toHaveCount(2);
    await expect(disclosures.nth(0)).not.toHaveAttribute("open");
    await expect(disclosures.nth(1)).not.toHaveAttribute("open");
    await capture(page, `inspector-${width}x${height}`);
    const route = page.url();
    const camera = await page
      .locator(".canvas-world")
      .getAttribute("style")
      .catch(() => null);
    page.on("request", (request) => {
      if (request.method() === "POST" && request.url().includes("/commands/"))
        mutations.push(request.url());
    });
    const trigger = panel(page).getByRole("button", {
      name: "Open Objectives Kanban",
      exact: true,
    });
    await trigger.click();
    const modal = page.locator("#workspace-kanban");
    await expect(modal).toBeVisible();
    await expect(
      modal.getByRole("button", { name: /Verified payment API/ }),
    ).toBeVisible();
    await expect(modal.locator(".workspace-kanban-card")).toHaveCount(1);
    await expect(modal).not.toContainText(data.nested.title);
    await capture(page, `objectives-${width}x${height}`);
    await modal
      .getByRole("button", { name: "Direct Tasks", exact: true })
      .click();
    await expect(
      modal.getByRole("button", { name: /Write release notes/ }),
    ).toBeVisible();
    await modal
      .getByRole("button", { name: "Objectives", exact: true })
      .click();
    await modal.getByRole("button", { name: /Verified payment API/ }).click();
    await expect(modal.locator(".workspace-kanban-inspection")).toBeVisible();
    await modal
      .getByRole("button", { name: "Open Tasks Kanban", exact: true })
      .click();
    await expect(
      modal.getByRole("button", { name: /Implement payment endpoint/ }),
    ).toBeVisible();
    await expect(modal.locator(".workspace-kanban-card")).toHaveCount(1);
    await modal.locator(".workspace-kanban-filters > summary").click();
    await modal
      .getByRole("button", { name: "Show cancelled", exact: true })
      .click();
    await expect(modal.locator("[data-column=cancelled]")).toContainText(
      "Retired experiment",
    );
    await expect(modal.locator("[data-column=accepted]")).not.toContainText(
      "Retired experiment",
    );
    await capture(page, `kanban-${width}x${height}`);
    const geometry = await modal.boundingBox();
    expect(geometry.x).toBeGreaterThanOrEqual(0);
    expect(geometry.y).toBeGreaterThanOrEqual(0);
    expect(geometry.x + geometry.width).toBeLessThanOrEqual(width + 1);
    expect(geometry.y + geometry.height).toBeLessThanOrEqual(height + 1);
    await modal
      .getByRole("button", { name: "Subobjectives", exact: true })
      .click();
    await expect(
      modal.getByRole("button", { name: /Refund support/ }),
    ).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(modal).not.toBeVisible();
    await expect(trigger).toBeFocused();
    expect(page.url()).toBe(route);
    if (camera !== null)
      expect(await page.locator(".canvas-world").getAttribute("style")).toBe(
        camera,
      );
    expect(mutations).toEqual([]);
    expect(errors).toEqual([]);
    await select(page, data.task.title);
    await expect(
      panel(page).getByRole("button", { name: /Open .* Kanban/ }),
    ).toHaveCount(0);
  });
}

test("criteria verification is separate from acceptance and cancelled work remains historical", async ({
  page,
  wos,
}) => {
  const { outcome, scope, task } = await setup(wos);
  await wos.command("add_criterion", {
    owner: { ...scope, kind: "work_item", id: task.id },
    expected_version: task.version,
    title: "The expired token is rejected",
    required: true,
    verification_mode: "attestation",
  });
  await wos.login(page);
  await openOutcome(page, outcome.title, { area: null });
  await select(page, task.title);
  await panel(page)
    .getByText("Details and management", { exact: true })
    .click();
  await panel(page)
    .getByRole("button", { name: "Verify criteria", exact: true })
    .click();
  const modal = page.locator("#workspace-kanban");
  await expect(
    modal.getByRole("heading", { name: "Verify criteria", exact: true }),
  ).toBeVisible();
  await expect(modal).toContainText("Not verified");
  await capture(page, "verify-criteria");
  await modal
    .getByRole("button", { name: "Review this criterion", exact: true })
    .click();
  await expect(modal.locator(".workspace-kanban-inspection")).toContainText(
    "The expired token is rejected",
  );
  await expect(modal.locator(".workspace-kanban-inspection")).not.toContainText(
    "Automatically accepted",
  );
});

test("creation in Objective Kanban preserves parent and returns to the same context", async ({
  page,
  wos,
}) => {
  const data = await setup(wos);
  await wos.login(page);
  await openOutcome(page, data.outcome.title, { area: null });
  await select(page, data.objective.title);
  await panel(page)
    .getByRole("button", { name: "Open Tasks Kanban", exact: true })
    .click();
  const modal = page.locator("#workspace-kanban");
  await modal.getByRole("button", { name: "Add Task", exact: true }).click();
  await expect(modal).not.toBeVisible();
  await expect(page.locator("#command-form")).toBeVisible();
  await expect(page.locator("#fields")).toContainText(data.objective.title);
  await page
    .getByLabel("Task title", { exact: true })
    .fill("Exercise the real refund API");
  const response = page.waitForResponse(
    (response) =>
      response.url().endsWith("/commands/create_work_item") &&
      response.request().method() === "POST",
  );
  await page.locator("#submit-command").click();
  const result = await response;
  expect(result.ok()).toBe(true);
  const value = (await result.json()).value;
  expect(value.objective_id).toBe(data.objective.id);
  expect(value.lifecycle).toBe("backlog");
  await expect(page.locator("#command-form")).not.toBeVisible();
  await panel(page)
    .getByRole("button", { name: "Return to Kanban", exact: true })
    .click();
  await expect(modal).toBeVisible();
  await expect(
    modal
      .locator(".workspace-kanban-card")
      .filter({ hasText: "Exercise the real refund API" }),
  ).toBeVisible();
  await expect(modal.locator(".workspace-kanban-breadcrumb")).toContainText(
    data.objective.title,
  );
});

test("activity history is loaded on demand and material choices retain their meaning", async ({
  page,
  wos,
}) => {
  const data = await setup(wos);
  const reads = [];
  page.on("request", (request) => {
    if (request.url().endsWith("/workspace-view"))
      reads.push(request.postDataJSON());
  });
  await wos.login(page);
  await openOutcome(page, data.outcome.title, { area: null });
  await select(page, data.task.title);
  expect(reads.at(-1).include_activity).toBe(false);
  const initial = reads.length;
  await panel(page).getByText("Activity history", { exact: true }).click();
  await expect
    .poll(() =>
      reads.slice(initial).some((query) => query.include_activity === true),
    )
    .toBe(true);
  await expect(
    panel(page)
      .locator("[data-workspace-facet=activity] .workspace-activity-entry")
      .first(),
  ).toBeVisible();
  await panel(page)
    .getByText("Details and management", { exact: true })
    .click();
  await panel(page)
    .getByRole("button", { name: "Add material", exact: true })
    .click();
  const modal = page.locator("#workspace-kanban");
  await expect(modal).toContainText("does not approve a criterion");
  await expect(modal).toContainText("File upload is not available");
  await capture(page, "add-material");
  await modal
    .getByRole("button", { name: "What was produced — Artifact", exact: true })
    .click();
  await expect(page.locator("#command-form")).toBeVisible();
  await page
    .getByLabel("Artifact name", { exact: true })
    .fill("Payment API pull request");
  await page.getByLabel("Artifact type", { exact: true }).fill("pull_request");
  await page
    .getByLabel("URL", { exact: true })
    .fill("https://example.test/pr/42");
  const response = page.waitForResponse(
    (response) =>
      response.url().endsWith("/commands/register_artifact_in_context") &&
      response.request().method() === "POST",
  );
  await page.locator("#submit-command").click();
  expect((await response).ok()).toBe(true);
  await expect(page.locator("#command-form")).not.toBeVisible();
  await panel(page)
    .getByRole("button", { name: "Return to Kanban", exact: true })
    .click();
  await expect(modal).toBeVisible();
  await expect(modal).toContainText("Add material");
});

test("contextual Kanban remains operable at native browser zoom 125%, 150% and 200%", async ({
  wos,
}) => {
  const data = await setup(wos);
  const browser = await nativeZoomBrowser({ width: 1280, height: 900 });
  try {
    for (const factor of [1.25, 1.5, 2]) {
      await browser.setZoom(factor);
      const page = browser.page;
      await page.context().clearCookies();
      await wos.login(page);
      await openOutcome(page, data.outcome.title, { area: null });
      await select(page, data.objective.title);
      await panel(page)
        .getByRole("button", { name: "Open Tasks Kanban", exact: true })
        .click();
      const modal = page.locator("#workspace-kanban");
      const card = modal
        .locator(".workspace-kanban-card")
        .filter({ hasText: data.task.title });
      await expect(card).toBeVisible();
      await card.click();
      await expect(modal.locator(".workspace-kanban-inspection")).toContainText(
        data.task.title,
      );
      await page.keyboard.press("Escape");
      await expect(modal).toBeVisible();
      await expect(modal.locator(".workspace-kanban-inspection")).toHaveCount(
        0,
      );
      await modal
        .getByRole("button", { name: "Close Kanban", exact: true })
        .click();
      await expect(modal).not.toBeVisible();
    }
  } finally {
    await browser.close();
  }
});
