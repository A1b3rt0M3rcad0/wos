import { chromium } from "playwright";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

// Change Chromium's own page zoom through its Settings API in an isolated
// profile. DPR emulation and CDP pinch scaling do not exercise browser zoom.
export async function nativeZoomBrowser({ width, height }) {
  const directory = await mkdtemp(join(tmpdir(), "wos-native-zoom-"));
  let context;
  try {
    context = await chromium.launchPersistentContext(directory, {
      headless: true,
      // Full Chromium exposes its native Settings UI; headless-shell does not.
      channel: "chromium",
      viewport: null,
      ...(process.env.WOS_TEST_CHROMIUM_PATH
        ? { executablePath: process.env.WOS_TEST_CHROMIUM_PATH }
        : {}),
      args: ["--no-sandbox", "--no-zygote", "--disable-dev-shm-usage", `--window-size=${width},${height}`],
    });
    const settings = await context.newPage();
    await settings.goto("chrome://settings/appearance");
    const page = await context.newPage();
    const session = await context.newCDPSession(page);
    const { windowId } = await session.send("Browser.getWindowForTarget");
    const frame = await page.evaluate(() => ({
      width: outerWidth - innerWidth,
      height: outerHeight - innerHeight,
    }));
    // Resize the native window, accounting for actual browser chrome, rather
    // than overriding CSS viewport metrics with an emulation command.
    await session.send("Browser.setWindowBounds", {
      windowId,
      bounds: { width: width + frame.width, height: height + frame.height },
    });
    await page.waitForFunction(({ width, height }) => innerWidth === width && innerHeight === height, { width, height });
    return {
      page,
      async setZoom(factor) {
        const actual = await settings.evaluate(async (value) => {
          await chrome.settingsPrivate.setDefaultZoom(value);
          return chrome.settingsPrivate.getDefaultZoom();
        }, factor);
        if (Math.abs(actual - factor) > 0.0001)
          throw Error(`Native browser zoom ${actual} does not match ${factor}`);
        await page.waitForFunction(({ factor, width, height }) =>
          Math.abs(devicePixelRatio - factor) < 0.0001 &&
          Math.abs(innerWidth - width / factor) <= 1 &&
          Math.abs(innerHeight - height / factor) <= 1 &&
          visualViewport.scale === 1,
        { factor, width, height });
        return actual;
      },
      async close() {
        try { await context.close(); }
        finally { await rm(directory, { recursive: true, force: true }); }
      },
    };
  } catch (error) {
    await context?.close();
    await rm(directory, { recursive: true, force: true });
    throw error;
  }
}
