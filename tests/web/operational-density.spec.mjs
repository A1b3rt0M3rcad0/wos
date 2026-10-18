import { test, expect, openOutcome, chooseArea, intent } from "./ux-fixture.mjs";
import { seedCanvas } from "./canvas-fixture.mjs";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { createHash } from "node:crypto";

// A physical monitor does not identify its browser's CSS viewport. The final
// case represents a 2560 x 1440 framebuffer with a 2048 x 1152 CSS viewport;
// it is device-scale emulation, not a claim of native browser zoom coverage.
const scenarios = [
  { width: 1280, height: 720, deviceScaleFactor: 1 },
  { width: 2560, height: 1440, deviceScaleFactor: 1 },
  { width: 3840, height: 2160, deviceScaleFactor: 1 },
  { width: 390, height: 844, deviceScaleFactor: 1 },
  { width: 320, height: 740, deviceScaleFactor: 1 },
  { width: 2048, height: 1152, deviceScaleFactor: 1.25 },
];
const description = "Observed study.adaptation_applied.v1 at aggregate revision 1; this is an operational fact, not an assessment of WOS criteria. A reviewer must inspect the delivery independently.";
const assets = [
  "style.css", "app.js", "canvas.css", "editorial.css", "workspace-entry.css",
  "workspace-canvas.js", "workspace-surfaces.css", "workspace-dialogs.css",
  "colors.css", "index.html", "en-US.js",
];
const sha256 = (value) => createHash("sha256").update(value).digest("hex");
const verifiedAssets = new WeakMap();
const board = (page) => page.locator("#operational-canvas");
const panel = (page) => page.locator("#canvas-details");

async function measured(page) {
  return page.evaluate(() => {
    const measure = (selector) => {
      const element = document.querySelector(selector);
      if (!element || !element.checkVisibility()) return null;
      const rect = element.getBoundingClientRect(), style = getComputedStyle(element);
      return {
        x: rect.x, y: rect.y, width: rect.width, height: rect.height,
        right: rect.right, bottom: rect.bottom,
        fontSize: parseFloat(style.fontSize), fontFamily: style.fontFamily,
        lineHeight: style.lineHeight, clientHeight: element.clientHeight,
        scrollHeight: element.scrollHeight,
      };
    };
    return {
      viewport: {
        width: innerWidth, height: innerHeight, devicePixelRatio,
        visualViewportScale: visualViewport.scale,
      },
      document: {
        width: document.documentElement.scrollWidth,
        height: document.documentElement.scrollHeight, scrollX, scrollY,
      },
      elements: Object.fromEntries([
        "html", ".topbar", ".brand-logo", ".sidebar", "#title", "#area-title",
        ".summary-purpose", "#outcome-views > summary", ".canvas-toolbar",
        ".canvas-toolbar input", ".canvas-viewport", ".canvas-list",
        "#canvas-details", "#detail-title", ".detail-description",
        "#detail-content", "#detail-actions", "#entity-actions",
        "#command-dialog", "#command-dialog .dialog-heading h2", "#submit-command",
        "#token", "#login button", "#empty h2",
      ].map((selector) => [selector, measure(selector)])),
    };
  });
}

async function wholeControl(locator) {
  await expect(locator).toBeVisible();
  expect(await locator.evaluate((element) => {
    const r = element.getBoundingClientRect();
    const hit = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
    return r.left >= -1 && r.top >= -1 && r.right <= innerWidth + 1 &&
      r.bottom <= innerHeight + 1 && element.contains(hit);
  }), "The control is visible, inside the viewport and receives pointer input").toBe(true);
}

async function noHorizontalOverflow(page) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(0);
}

async function boundedBoard(page) {
  await expect.poll(() => page.evaluate(() => ({
    excessX: document.documentElement.scrollWidth - innerWidth,
    excessY: document.documentElement.scrollHeight - innerHeight,
    x: scrollX, y: scrollY,
  }))).toEqual({ excessX: 0, excessY: 0, x: 0, y: 0 });
  await wholeControl(board(page).getByRole("combobox", { name: "Search Outcome Board", exact: true }));
}

async function verifyServedAssets(page, wos) {
  if (!process.env.WOS_CAPTURE_DENSITY) return;
  const entries = await Promise.all(assets.map(async (file) => {
    const [response, source] = await Promise.all([
      page.request.get(`${wos.url}/app/${file}`),
      readFile(new URL(`../../packages/wos-api/web/assets/${file}`, import.meta.url)),
    ]);
    expect(response.ok(), `Capture asset ${file} is served`).toBe(true);
    const actual = sha256(await response.body());
    expect(actual, `Capture refuses stale embedded ${file}`).toBe(sha256(source));
    return [file, actual];
  }));
  verifiedAssets.set(page, Object.fromEntries(entries));
}

async function capture(page, name, state) {
  const directory = process.env.WOS_CAPTURE_DENSITY;
  if (!directory) return;
  expect(verifiedAssets.has(page), "Captured production assets were checked against source").toBe(true);
  await mkdir(directory, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.locator("img").evaluateAll((images) => Promise.all(
    images.filter((image) => image.checkVisibility()).map((image) => image.decode()),
  ));
  await page.mouse.move(0, 0);
  const geometry = await measured(page);
  const png = await page.screenshot({ path: join(directory, `${name}.png`), fullPage: false, animations: "disabled" });
  const pixelSize = { width: png.readUInt32BE(16), height: png.readUInt32BE(20) };
  expect(pixelSize.width).toBe(Math.round(geometry.viewport.width * geometry.viewport.devicePixelRatio));
  expect(pixelSize.height).toBe(Math.round(geometry.viewport.height * geometry.viewport.devicePixelRatio));
  await writeFile(join(directory, `${name}.json`), JSON.stringify({
    state, ...geometry, pixelSize, screenshotSha256: sha256(png),
    captureMethod: "Playwright visible viewport at the recorded device scale; no image resizing",
    servedAssetsSha256: verifiedAssets.get(page), capturedAt: new Date().toISOString(),
  }, null, 2) + "\n");
}

for (const scenario of scenarios) {
  const { width, height, deviceScaleFactor } = scenario;
  const label = `${width}x${height}${deviceScaleFactor === 1 ? "" : `-dpr${deviceScaleFactor}`}`;
  test.describe(`Operational density at ${label}`, () => {
    test.use({ viewport: { width, height }, deviceScaleFactor });

    test("keeps readable controls and bounded inspection without enlarging the whole interface", async ({ page, wos }) => {
      const data = await seedCanvas(wos, { title: "Application operational integration" });
      await wos.command("register_evidence", {
        scope: data.scope, evidence_type: "inspection", description,
        source_ref: { provider: "density-acceptance", id: "adaptation-observation" },
        captured_at: "2026-10-16T12:00:00Z",
      });
      await page.goto(`${wos.url}/app/`);
      await verifyServedAssets(page, wos);
      if (width <= 390) {
        await wholeControl(page.locator("#token"));
        await wholeControl(page.locator("#login button"));
        await noHorizontalOverflow(page);
        await capture(page, `login-${label}`, "Complete credential login with local access collapsed");
      }
      await wos.login(page);
      await openOutcome(page, data.outcome.title);
      await page.evaluate(() => document.fonts.ready);
      await page.evaluate(() => scrollTo(0, 0));
      await noHorizontalOverflow(page);
      const overview = await measured(page);
      expect(overview.elements.html.fontSize).toBeGreaterThanOrEqual(14);
      expect(overview.elements.html.fontSize).toBeLessThanOrEqual(16);
      expect(overview.elements[".topbar"].height).toBeLessThanOrEqual(64);
      expect(overview.elements["#title"].fontSize).toBeLessThanOrEqual(40);
      expect(overview.elements["#area-title"].fontSize).toBeLessThanOrEqual(32);
      expect(overview.elements[".summary-purpose"].fontSize).toBeLessThanOrEqual(28);
      if (width > 760) {
        expect(overview.elements[".sidebar"].width).toBeGreaterThanOrEqual(200);
        expect(overview.elements[".sidebar"].width).toBeLessThanOrEqual(280);
      } else {
        await expect(page.locator(".sidebar")).toHaveClass(/navigation-collapsed/);
        expect(overview.elements[".sidebar"].height).toBeLessThanOrEqual(136);
        await wholeControl(page.locator("#namespace"));
        await wholeControl(page.locator("#navigation-toggle"));
        await wholeControl(page.locator("#administration"));
      }
      expect(overview.viewport.devicePixelRatio).toBe(deviceScaleFactor);
      await capture(page, `overview-${label}`, "Overview with editorial identity and stable operational text/navigation sizes");

      if (width === 2560 || width === 390) {
        await intent(page, "New task");
        const form = page.locator("#command-dialog");
        const initialState = form.getByLabel("Initial state", { exact: true });
        await initialState.scrollIntoViewIfNeeded();
        await wholeControl(initialState);
        await wholeControl(page.locator("#submit-command"));
        expect(await form.locator(".dialog-heading h2").evaluate((element) => parseFloat(getComputedStyle(element).fontSize))).toBeLessThanOrEqual(24.1);
        const fields = await form.locator("input, select, textarea").evaluateAll((elements) => elements
          .filter((element) => element.checkVisibility())
          .map((element) => ({ font: parseFloat(getComputedStyle(element).fontSize), height: element.getBoundingClientRect().height })));
        expect(fields.length).toBeGreaterThan(0);
        for (const field of fields) {
          expect(field.font).toBeLessThanOrEqual(16.1);
          expect(field.height).toBeGreaterThanOrEqual(44);
        }
        await noHorizontalOverflow(page);
        await capture(page, `task-form-${label}`, "Native Task form; whole select borders and reachable submit, without submitting a mutation");
        await page.keyboard.press("Escape");
      }

      const mutations = [];
      page.on("request", (request) => {
        if (request.method() === "POST" && request.url().includes("/commands/")) mutations.push(request.url());
      });
      await chooseArea(page, "canvas");
      await expect(page.locator(".sidebar")).toBeHidden();
      await boundedBoard(page);
      await board(page).getByRole("button", { name: "Map view", exact: true }).click();
      for (const name of ["Zoom in", "Zoom out", "Center view", "Fit view"]) {
        const button = board(page).getByRole("button", { name, exact: true });
        await wholeControl(button);
        const box = await button.boundingBox();
        expect(box.width).toBeGreaterThanOrEqual(44);
        expect(box.height).toBeGreaterThanOrEqual(44);
        expect(box.width).toBeLessThanOrEqual(48);
        expect(box.height).toBeLessThanOrEqual(48);
      }
      if ([1280, 2560, 3840, 390].includes(width))
        await capture(page, `board-${label}-map`, "Outcome Board map with stable chrome and original post-it cards");
      const heading = await page.locator(".outcome-heading").boundingBox();
      const toolbar = await board(page).locator(".canvas-toolbar").boundingBox();
      if (width <= 760)
        expect(toolbar.y, "Compact Board chrome leaves room for the working surface").toBeLessThanOrEqual(320);
      const search = board(page).getByRole("combobox", { name: "Search Outcome Board", exact: true });
      await search.fill("study.adaptation_applied.v1");
      // Search summaries intentionally bound authored prose; match its unique
      // visible prefix and assert the complete source only in record details.
      await board(page).getByRole("option").filter({ hasText: "study.adaptation_applied.v1" }).click();
      await expect(panel(page)).toBeVisible();
      await expect(panel(page).locator("#detail-title")).toHaveText("Evidence");
      await expect(panel(page).locator(".detail-description")).toHaveText(description);
      await expect(panel(page).getByText(description, { exact: true })).toHaveCount(1);
      expect(await page.locator(".outcome-heading").boundingBox()).toEqual(heading);
      expect(await board(page).locator(".canvas-toolbar").boundingBox()).toEqual(toolbar);
      await boundedBoard(page);
      const inspected = await measured(page);
      const inspector = inspected.elements["#canvas-details"];
      const map = inspected.elements[".canvas-viewport"];
      const overlay = inspector.x < map.right;
      expect(inspector.width).toBeLessThanOrEqual(overlay ? 420 : 360);
      expect(inspected.elements["#detail-title"].fontSize).toBeLessThanOrEqual(22);
      expect(inspected.elements[".detail-description"].fontSize).toBeGreaterThanOrEqual(13.5);
      expect(inspected.elements[".detail-description"].fontSize).toBeLessThanOrEqual(16);
      expect(inspected.elements[".detail-description"].fontFamily).toContain("DM Sans");
      expect(inspected.elements["#detail-title"].fontFamily).toContain("Instrument Serif");
      expect(inspected.elements["#detail-content"].clientHeight).toBeGreaterThan(48);
      await capture(page, `board-${label}-inspector`, "Untitled Evidence keeps its long authored description once, in readable operational body text");

      // Exercise the actual scrolling surface before opening the existing action
      // menu, so an on-screen panel cannot hide an unreachable footer control.
      await panel(page).evaluate((element) => {
        for (const surface of element.querySelectorAll(".canvas-inspector-detail, #detail-content, #detail-actions"))
          if (surface.scrollHeight > surface.clientHeight) surface.scrollTop = surface.scrollHeight;
      });
      await panel(page).locator("#entity-actions").scrollIntoViewIfNeeded();
      await wholeControl(panel(page).locator("#entity-actions"));
      await panel(page).locator("#entity-actions").click();
      await expect(panel(page).locator(".canvas-inspector-actions")).toBeVisible();
      await page.keyboard.press("Escape");
      await expect(panel(page).locator(".canvas-inspector-detail")).toBeVisible();
      await wholeControl(panel(page).locator("#close-detail"));
      await panel(page).locator("#close-detail").click();
      await expect(panel(page)).toBeHidden();
      await boundedBoard(page);
      expect(mutations).toEqual([]);
    });
  });
}

test("An empty workspace keeps bounded headings and reachable navigation/settings", async ({ page, wos }) => {
  await wos.login(page);
  await verifyServedAssets(page, wos);
  for (const [width, height] of [[320, 740], [1280, 720], [2560, 1440]]) {
    await page.setViewportSize({ width, height });
    await expect(page.locator("#empty")).toBeVisible();
    await expect(page.locator("#empty h2")).toHaveText("Choose an outcome");
    const size = await page.locator("#empty h2").evaluate((element) => parseFloat(getComputedStyle(element).fontSize));
    expect(size).toBeCloseTo(width <= 760 ? 28 : 32, 1);
    await noHorizontalOverflow(page);
    await wholeControl(page.locator("#namespace"));
    await wholeControl(page.locator("#administration"));
    if (width <= 760) {
      const sidebar = await page.locator(".sidebar").boundingBox();
      expect(sidebar.height).toBeLessThanOrEqual(136);
      await wholeControl(page.locator("#navigation-toggle"));
    }
    await capture(page, `empty-workspace-${width}x${height}`, "Empty authenticated workspace keeps normal-sized headings, workspace switcher and settings");
    await page.locator("#administration").click();
    await expect(page.locator("#admin-dialog")).toBeVisible();
    await wholeControl(page.locator("#close-admin"));
    await page.locator("#close-admin").click();
    await expect(page.locator("#admin-dialog")).toBeHidden();
    await expect(page.locator("#administration")).toBeFocused();
    if (width <= 760) {
      await page.locator("#navigation-toggle").click();
      await expect(page.locator("#navigation-toggle")).toHaveAttribute("aria-expanded", "true");
      await wholeControl(page.locator("#text"));
      await page.locator("#navigation-toggle").click();
      await expect(page.locator("#navigation-toggle")).toHaveAttribute("aria-expanded", "false");
    }
    await noHorizontalOverflow(page);
  }
});
