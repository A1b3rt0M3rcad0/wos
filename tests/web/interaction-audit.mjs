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
  async function selectionTransition(selector, activate) {
    const reduced = await page.evaluate(
      () => matchMedia("(prefers-reduced-motion: reduce)").matches,
    );
    await page.emulateMedia({ reducedMotion: "no-preference" });
    // Arm before the real click: measuring only settled endpoints misses ink /
    // paper cross-fades that become unreadable between two accessible states.
    const capture = await page.evaluateHandle((selector) => {
      const controls = [...document.querySelectorAll(selector)];
      const initial = controls.map((node) => [node.className, node.getAttribute("aria-pressed")].join("/"));
      const canvas = document.createElement("canvas");
      canvas.width = canvas.height = 1;
      const context = canvas.getContext("2d", { willReadFrequently: true });
      const rgba = (color) => {
        context.clearRect(0, 0, 1, 1);
        context.fillStyle = color;
        context.fillRect(0, 0, 1, 1);
        return [...context.getImageData(0, 0, 1, 1).data];
      };
      const composite = (front, back) => back.map(
        (value, index) => front[index] * front[3] / 255 + value * (1 - front[3] / 255),
      );
      const luminance = (rgb) => rgb.map((value) => {
        const channel = value / 255;
        return channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4;
      }).reduce((sum, value, index) => sum + value * [0.2126, 0.7152, 0.0722][index], 0);
      const frames = [];
      let resolve, started, animation;
      const finished = new Promise((done) => { resolve = done; });
      const observer = new MutationObserver(() => {
        if (controls.every((node, index) => [node.className, node.getAttribute("aria-pressed")].join("/") === initial[index])) return;
        observer.disconnect();
        clearTimeout(timeout);
        started = performance.now();
        frame(started);
      });
      function frame(now) {
        for (const node of controls) {
          const ancestors = [];
          for (let ancestor = node; ancestor; ancestor = ancestor.parentElement) ancestors.unshift(ancestor);
          let paper = [255, 255, 255];
          for (const ancestor of ancestors) paper = composite(rgba(getComputedStyle(ancestor).backgroundColor), paper);
          const style = getComputedStyle(node), ink = composite(rgba(style.color), paper);
          const light = luminance(ink), dark = luminance(paper);
          frames.push({
            time_ms: now - started,
            label: node.textContent.trim(),
            connected: node.isConnected,
            color: style.color,
            background: style.backgroundColor,
            composited_background: paper,
            contrast: (Math.max(light, dark) + 0.05) / (Math.min(light, dark) + 0.05),
          });
        }
        if (now - started >= 200) resolve({ frames, changed: true });
        else animation = requestAnimationFrame(frame);
      }
      for (const node of controls) observer.observe(node, { attributes: true, attributeFilter: ["class", "aria-pressed"] });
      const timeout = setTimeout(() => { observer.disconnect(); resolve({ frames, changed: false }); }, 2000);
      return { finished, cancel() { observer.disconnect(); clearTimeout(timeout); cancelAnimationFrame(animation); } };
    }, selector);
    try {
      await activate();
      const result = await capture.evaluate((run) => run.finished);
      expect(result.changed, `${selector}: real selection must change`).toBe(true);
      expect(
        result.frames.some((frame) => frame.time_ms > 0 && frame.time_ms < 140),
        `${selector}: observe the transition window, not only its endpoints`,
      ).toBe(true);
      for (const frame of result.frames) {
        expect(frame.connected, `${frame.label}: control remains attached`).toBe(true);
        expect(
          frame.contrast,
          `${frame.label}: selection contrast at ${frame.time_ms.toFixed(1)}ms (${frame.color} on ${frame.background})`,
        ).toBeGreaterThanOrEqual(4.5);
      }
      report.push({ label: "Selection transition", variant: selector, contrast: Math.min(...result.frames.map((frame) => frame.contrast)), samples: result.frames.length, frame_window_ms: 200 });
    } finally {
      await capture.evaluate((run) => run.cancel());
      await capture.dispose();
      await page.emulateMedia({ reducedMotion: reduced ? "reduce" : "no-preference" });
    }
  }
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
      // A fixed header can cover a target that is geometrically in the viewport.
      // Let the browser finish its hit-test scroll before measuring idle paint;
      // the subsequent idle/hover/pressed rectangles remain strictly identical.
      await control.hover();
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
  return { audit, selectionTransition, report };
}
