import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { test, expect } from "playwright/test";

// Measure actual rendered geometry, including relative type and fixed lengths.
export async function dimensions(page, selectors) {
  return page.evaluate((selectors) => {
    const result = {};
    for (const selector of selectors) {
      const element = document.querySelector(selector);
      if (!element) throw Error(`Missing scale audit target: ${selector}`);
      const style = getComputedStyle(element);
      const box = element.getBoundingClientRect();
      const scroller = element.closest(".dialog-body");
      result[selector] = {
        x: box.x,
        y: box.y + scrollY,
        width: box.width,
        height: box.height,
        fontSize:
          parseFloat(style.fontSize) *
          (element instanceof SVGTextElement ? element.getScreenCTM().a : 1),
        paddingTop: parseFloat(style.paddingTop),
        paddingLeft: parseFloat(style.paddingLeft),
        fontFamily: style.fontFamily,
        columns: style.gridTemplateColumns.split(" ").length,
        nativeGutter: scroller
          ? scroller.offsetWidth - scroller.clientWidth
          : 0,
      };
    }
    return result;
  }, selectors);
}

export function expectProportional(actual, reference, factor) {
  for (const [selector, baseline] of Object.entries(reference)) {
    for (const key of [
      "x",
      "y",
      "width",
      "height",
      "fontSize",
      "paddingTop",
      "paddingLeft",
    ]) {
      // Native scrollbar gutters stay in browser pixels; account for that only
      // when checking descendant horizontal dimensions.
      const gutter = ["x", "width"].includes(key)
        ? Math.abs(
            baseline.nativeGutter - actual[selector].nativeGutter / factor,
          )
        : 0;
      // The main document accumulates rounding across all section rules.
      const rounding =
        key === "height" && selector === ".workspace-main" ? 10 : 3;
      expect(
        Math.abs(actual[selector][key] / factor - baseline[key]),
        `${selector} ${key} at ${factor}x`,
      ).toBeLessThan(rounding + gutter);
    }
    expect(actual[selector].fontFamily, selector).toBe(baseline.fontFamily);
    expect(actual[selector].columns, selector).toBe(baseline.columns);
  }
}

// Optional same-DOM comparison against the pre-change CSS. CI needs no archived
// stylesheet copies; the local review can prove pixel identity at smaller widths.
export async function expectOriginalPixels(page) {
  const directory = process.env.WOS_SCALE_BASELINE_DIR;
  if (!directory) return;
  await page.mouse.move(0, 0);
  await page.evaluate(() => document.fonts.ready);
  await page
    .locator("img")
    .evaluateAll((images) =>
      Promise.all(
        images
          .filter((img) => img.checkVisibility())
          .map((img) => img.decode()),
      ),
    );
  const current = await page.screenshot({ animations: "disabled" });
  const original = await Promise.all(
    ["style.css", "editorial.css"].map(async (file) => ({
      file,
      css: await readFile(join(directory, file), "utf8"),
    })),
  );
  // Add the original declarations after the current sheets, retaining loaded
  // font faces and parsing source text directly (CSSOM shorthand serialization
  // is not lossless for mixed variable and border shorthands).
  const pattern = /\/(style|editorial)\.css\?scale-original$/;
  await page.route(pattern, (route) =>
    route.fulfill({
      contentType: "text/css",
      body: original
        .find(({ file }) =>
          new URL(route.request().url()).pathname.endsWith("/" + file),
        )
        .css.replace(/@font-face\s*\{[^}]*\}/g, ""),
    }),
  );
  await page.evaluate(
    (files) =>
      Promise.all(
        files.map(
          (file) =>
            new Promise((resolve, reject) => {
              const link = document.createElement("link");
              link.rel = "stylesheet";
              link.href = file + "?scale-original";
              link.dataset.originalScale = file;
              link.onload = resolve;
              link.onerror = reject;
              document.head.append(link);
            }),
        ),
      ),
    original.map(({ file }) => file),
  );
  await page.evaluate(() => document.fonts.ready);
  try {
    const previous = await page.screenshot({ animations: "disabled" });
    if (!previous.equals(current)) {
      await test
        .info()
        .attach("scale-current", { body: current, contentType: "image/png" });
      await test
        .info()
        .attach("scale-original", { body: previous, contentType: "image/png" });
    }
    expect(
      previous.equals(current),
      `Original pixels at ${JSON.stringify(page.viewportSize())}`,
    ).toBe(true);
  } finally {
    await page.evaluate(() =>
      document
        .querySelectorAll("[data-original-scale]")
        .forEach((link) => link.remove()),
    );
    await page.unroute(pattern);
  }
}
