import { expect } from "playwright/test";

// Measure actual rendered geometry, including relative type and fixed lengths.
export async function dimensions(page, selectors) {
  await page.evaluate(async (selectors) => {
    // A viewport change can request a font that was not used by the previous
    // layout. Wait for the resolved faces before measuring ch-based headings.
    await Promise.all(
      selectors.map((selector) => {
        const element = document.querySelector(selector);
        if (!element) throw Error(`Missing density audit target: ${selector}`);
        const style = getComputedStyle(element);
        return document.fonts.load(
          `${style.fontStyle} ${style.fontWeight} ${style.fontSize} ${style.fontFamily}`,
          element.textContent || "0",
        );
      }),
    );
    await document.fonts.ready;
  }, selectors);
  return page.evaluate((selectors) => {
    const result = {};
    for (const selector of selectors) {
      const element = document.querySelector(selector);
      if (!element) throw Error(`Missing density audit target: ${selector}`);
      const style = getComputedStyle(element);
      const box = element.getBoundingClientRect();
      result[selector] = {
        width: box.width,
        height: box.height,
        fontSize:
          parseFloat(style.fontSize) *
          (element instanceof SVGTextElement ? element.getScreenCTM().a : 1),
        paddingTop: parseFloat(style.paddingTop),
        paddingLeft: parseFloat(style.paddingLeft),
        fontFamily: style.fontFamily,

      };
    }
    return result;
  }, selectors);
}

// Compare the operational tokens that must remain stable as desktop space
// grows. Container positions/heights remain free to follow their natural flow.
export function expectStableMetrics(actual, reference, properties) {
  for (const [selector, baseline] of Object.entries(reference)) {
    for (const property of properties)
      expect(
        actual[selector][property],
        `${selector} stable ${property}`,
      ).toBeCloseTo(baseline[property], property === "fontSize" ? 1 : 0);
  }
}
