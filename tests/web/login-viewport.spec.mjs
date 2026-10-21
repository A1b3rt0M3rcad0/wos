import { test, expect } from "./ux-fixture.mjs";
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { readFile, readdir, mkdir, writeFile } from "node:fs/promises";
import { resolve, join } from "node:path";

const root = resolve("../..");
const sha = (data) => createHash("sha256").update(data).digest("hex");
const sizes = [[1280, 720], [1366, 768], [1920, 1080], [2560, 1440], [3840, 2160],
  [2048, 1152, 1.25], [320, 740], [390, 844], [430, 932], [844, 390]];

async function geometry(page) {
  return page.evaluate(() => {
    const box = (selector) => {
      const r = document.querySelector(selector).getBoundingClientRect();
      return { x: r.x, y: r.y, width: r.width, height: r.height, bottom: r.bottom, right: r.right };
    };
    const story = box(".login-story"), panel = box(".login-panel"), header = box(".topbar");
    return { viewport: { width: innerWidth, height: innerHeight, dpr: devicePixelRatio },
      story, panel, header, input: box("#token"), button: box("#login button"),
      composition: { top: Math.min(story.y, panel.y), bottom: Math.max(story.bottom, panel.bottom) },
      document: { width: document.documentElement.scrollWidth, height: document.documentElement.scrollHeight, scrollY } };
  });
}

async function capture(page, wos, name, measured) {
  const output = process.env.WOS_CAPTURE_LOGIN_VIEWPORT;
  if (!output) return;
  const source = execFileSync("git", ["rev-parse", "HEAD"], { cwd: root, encoding: "utf8" }).trim();
  const binaryPath = join(root, "bin/wos");
  const binary = JSON.parse(execFileSync(binaryPath, ["--output", "json", "version"], { encoding: "utf8" })).data;
  expect(binary.commit).toBe(source);
  expect(execFileSync("git", ["status", "--porcelain", "--", "packages", "tests/web", "deploy", "tools"],
    { cwd: root, encoding: "utf8" })).toBe("");
  const assets = join(root, "packages/wos-api/web/assets"), served = {};
  for (const file of (await readdir(assets, { recursive: true })).sort()) {
    if (!/\.(js|css)$/.test(file) && file !== "index.html") continue;
    const response = await page.request.get(`${wos.url}/app/${file}`);
    expect(response.ok()).toBe(true);
    const digest = sha(await response.body());
    expect(digest).toBe(sha(await readFile(join(assets, file))));
    served[file] = digest;
  }
  const directory = join(resolve(output), source.slice(0, 12));
  await mkdir(directory, { recursive: true });
  const png = await page.screenshot({ fullPage: false, animations: "disabled" });
  await writeFile(join(directory, `${name}.png`), png, { flag: "wx" });
  await writeFile(join(directory, `${name}.json`), JSON.stringify({ sourceCommit: source,
    capturedAt: new Date().toISOString(), browser: page.context().browser().version(),
    binary: { ...binary, sha256: sha(await readFile(binaryPath)) },
    servedAssetsSha256: served, testSha256: sha(await readFile(join(root, "tests/web/login-viewport.spec.mjs"))),
    image: `${name}.png`, screenshotSha256: sha(png), pixelSize: { width: png.readUInt32BE(16), height: png.readUInt32BE(20) },
    state: name.includes("expanded") ? "Local access expanded; actual visible viewport after reaching its submit control" : "Initial login, complete composition centered when it fits",
    ...measured }, null, 2) + "\n", { flag: "wx" });
}

for (const [width, height, dpr = 1] of sizes) {
  test.describe(`Login viewport ${width}x${height} DPR ${dpr}`, () => {
    test.use({ viewport: { width, height }, deviceScaleFactor: dpr });
    test("centers the complete composition below the actual header and preserves accessible overflow", async ({ page, wos }) => {
      await page.goto(`${wos.url}/app/`);
      await page.evaluate(() => document.fonts.ready);
      await page.locator(".login-vignette img").evaluate((image) => image.decode());
      await expect(page.locator("#token")).toBeVisible();
      const g = await geometry(page);
      expect(g.document.width).toBeLessThanOrEqual(width);
      expect(g.composition.top).toBeGreaterThanOrEqual(g.header.bottom);
      const fits = g.composition.bottom - g.composition.top + 32 <= height - g.header.bottom;
      if (fits) {
        const center = (g.composition.top + g.composition.bottom) / 2;
        expect(Math.abs(center - (g.header.bottom + height) / 2), "Use the available viewport, not just alignment between columns").toBeLessThanOrEqual(8);
        expect(g.document.height).toBeLessThanOrEqual(height + 1);
        expect(g.input.y).toBeGreaterThanOrEqual(0);
        expect(g.button.bottom).toBeLessThanOrEqual(height);
      }
      await capture(page, wos, `login-${width}x${height}${dpr === 1 ? "" : `-dpr${dpr}`}`, g);
      await page.locator("#token").focus();
      await page.keyboard.press("Tab");
      await expect(page.locator("#login button")).toBeFocused();
      expect(await page.locator("#login button").evaluate((button) => {
        const r = button.getBoundingClientRect();
        return r.height >= 44 && r.top >= 0 && r.bottom <= innerHeight && button.contains(document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2));
      })).toBe(true);
    });
  });
}

for (const viewport of [{ width: 1280, height: 720 }, { width: 390, height: 667 }]) {
  test(`expanded local login stays reachable at ${viewport.width}x${viewport.height}`, async ({ page, wos }) => {
    await page.setViewportSize(viewport);
    await page.goto(`${wos.url}/app/`);
    await page.evaluate(() => document.fonts.ready);
    await page.locator(".entry-local > summary").click();
    const button = page.locator("#local button");
    await button.focus();
    await expect(button).toBeFocused();
    expect(await button.evaluate((element) => {
      const r = element.getBoundingClientRect();
      return r.top >= 0 && r.bottom <= innerHeight && element.contains(document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2));
    })).toBe(true);
    const g = await geometry(page);
    expect(g.document.width).toBeLessThanOrEqual(viewport.width);
    await capture(page, wos, `login-expanded-${viewport.width}x${viewport.height}`, g);
  });
}
