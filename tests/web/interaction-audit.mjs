import { expect } from "./ux-fixture.mjs";

async function appearance(control) {
  return control.evaluate((element) => {
    const style = getComputedStyle(element);
    const canvas = document.createElement("canvas");
    canvas.width = canvas.height = 1;
    const context = canvas.getContext("2d", { willReadFrequently: true });
    const rgba = (color) => {
      context.clearRect(0, 0, 1, 1);
      context.fillStyle = color;
      context.fillRect(0, 0, 1, 1);
      return [...context.getImageData(0, 0, 1, 1).data];
    };
    let background = [255, 255, 255];
    const ancestors = [];
    for (let node = element; node; node = node.parentElement)
      ancestors.unshift(node);
    for (const node of ancestors) {
      const color = rgba(getComputedStyle(node).backgroundColor);
      background = background.map(
        (value, index) =>
          (color[index] * color[3]) / 255 + value * (1 - color[3] / 255),
      );
    }
    const luminance = (rgb) =>
      rgb
        .slice(0, 3)
        .map((value) => {
          const channel = value / 255;
          return channel <= 0.04045
            ? channel / 12.92
            : ((channel + 0.055) / 1.055) ** 2.4;
        })
        .reduce(
          (sum, value, index) => sum + value * [0.2126, 0.7152, 0.0722][index],
          0,
        );
    const ink = luminance(rgba(style.color)),
      paper = luminance(background);
    const bounds = element.getBoundingClientRect();
    return {
      contrast: (Math.max(ink, paper) + 0.05) / (Math.min(ink, paper) + 0.05),
      color: style.color,
      background: style.backgroundColor,
      border: [
        style.borderTopColor,
        style.borderRightColor,
        style.borderBottomColor,
        style.borderLeftColor,
      ],
      padding: [parseFloat(style.paddingLeft), parseFloat(style.paddingRight)],
      bounds: [bounds.x, bounds.y, bounds.width, bounds.height],
    };
  });
}

// Discover real controls instead of maintaining a second list of button variants.
export function interactionAudit(page) {
  const seen = new Set(),
    disabled = new Set(),
    report = [];
  async function audit(root = page.locator("body")) {
    await page.evaluate(() => document.fonts.ready);
    for (const control of await root.locator("button, summary").all()) {
      if (!(await control.isVisible())) continue;
      const info = await control.evaluate((element) => ({
        key: `${innerWidth}/${element.tagName}/${element.id}/${element.className}/${element.textContent}`,
        label: (element.getAttribute("aria-label") || element.textContent)
          .trim()
          .replace(/\s+/g, " ")
          .slice(0, 100),
        button: element.tagName === "BUTTON",
        icon: element.classList.contains("icon-button"),
        variant: `${element.tagName}/${element.className}`,
      }));
      if (seen.has(info.key)) continue;
      seen.add(info.key);
      if (info.button && !(await control.isEnabled())) continue;
      await control.scrollIntoViewIfNeeded();
      await page.mouse.move(0, 0);
      const idle = await appearance(control);
      expect(
        idle.contrast,
        `${info.label}: resting text contrast`,
      ).toBeGreaterThanOrEqual(4.5);
      expect(
        idle.bounds[3],
        `${info.label}: target height`,
      ).toBeGreaterThanOrEqual(44);
      if (!info.icon)
        expect(
          Math.min(...idle.padding),
          `${info.label}: label inset`,
        ).toBeGreaterThanOrEqual(8);
      await control.hover();
      const hover = await appearance(control);
      expect(
        hover.bounds,
        `${info.label}: hover must not move the target`,
      ).toEqual(idle.bounds);
      expect(
        hover.contrast,
        `${info.label}: hover text contrast`,
      ).toBeGreaterThanOrEqual(4.5);
      if (info.button) {
        await page.mouse.down();
        const pressed = await appearance(control);
        expect(pressed.bounds, `${info.label}: pressed geometry`).toEqual(
          idle.bounds,
        );
        expect(
          pressed.contrast,
          `${info.label}: pressed contrast`,
        ).toBeGreaterThanOrEqual(4.5);
        // Release outside the target, so this style audit dispatches no commands.
        await page.mouse.move(0, 0);
        await page.mouse.up();
        await page.evaluate(() => getSelection()?.removeAllRanges());
        if (!disabled.has(info.variant)) {
          disabled.add(info.variant);
          await control.evaluate((element) => {
            element.disabled = true;
          });
          try {
            const before = await appearance(control);
            await control.hover({ force: true });
            const after = await appearance(control);
            expect(
              [after.color, after.background, after.border],
              `${info.label}: disabled hover`,
            ).toEqual([before.color, before.background, before.border]);
          } finally {
            await control.evaluate((element) => {
              element.disabled = false;
            });
          }
        }
      }
      report.push({
        label: info.label,
        variant: info.variant,
        contrast: hover.contrast,
      });
    }
  }
  return { audit, report };
}
