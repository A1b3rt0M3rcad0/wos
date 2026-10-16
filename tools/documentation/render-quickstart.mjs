// Render the actual asciicast output; do not synthesize command results.
import { readFile, writeFile, mkdtemp, rm } from "node:fs/promises";
import { join, resolve, dirname } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";
import { spawn } from "node:child_process";
import { once } from "node:events";
import { createHash } from "node:crypto";
import { chromium } from "../../tests/web/node_modules/playwright/index.mjs";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const source = resolve(
  process.argv[2] || join(root, "examples/quickstart/demo.cast"),
);
const output = resolve(
  process.argv[3] || join(root, "docs/images/quickstart.gif"),
);
const bytes = await readFile(source);
const [header, ...events] = bytes
  .toString("utf8")
  .trim()
  .split("\n")
  .map(JSON.parse);
if (header.version !== 2) throw new Error("Expected asciicast v2");
const recorded = events.at(-1)[0];
const fps = 2,
  duration = Math.ceil(recorded) + 2;
const directory = await mkdtemp(join(tmpdir(), "wos-cast-render-"));
const browser = await chromium.launch({
  ...(process.env.WOS_TEST_CHROMIUM_PATH
    ? { executablePath: process.env.WOS_TEST_CHROMIUM_PATH }
    : {}),
  headless: true,
});
try {
  const page = await browser.newPage({
    viewport: { width: 960, height: 540 },
    deviceScaleFactor: 1,
  });
  await page.setContent(`<!doctype html><html lang="en"><meta charset="utf-8"><style>
  *{box-sizing:border-box}body{margin:0;background:#EFEDE8;color:#292620;font:14px/1.45 monospace}
  header{height:45px;padding:12px 20px;border-bottom:1px solid #D4CEC4;background:#F5F3EF;font:12px/20px system-ui,sans-serif}
  header b{color:#A7441D}pre{margin:0;padding:18px 20px;height:460px;overflow:hidden;white-space:pre-wrap;overflow-wrap:anywhere;font:14px/20px monospace}
  footer{height:35px;padding:7px 20px;border-top:1px solid #D4CEC4;background:#F5F3EF;font:11px/20px system-ui,sans-serif}
  </style><header><b>Local workflow</b> · One trusted operator · Unsigned HTTP</header><pre></pre><footer></footer></html>`);
  let text = "",
    next = 0;
  for (let frame = 0; frame < duration * fps; frame++) {
    const seconds = Math.min((frame + 1) / fps, recorded);
    while (next < events.length && events[next][0] <= seconds) {
      const [, type, value] = events[next++];
      if (type === "o")
        text += value
          .replace(/\r\n/g, "\n")
          .replace(/\r/g, "")
          .replace(/\x1b\[[0-9;]*[A-Za-z]/g, "");
    }
    await page.evaluate(
      ({ text, seconds, recorded }) => {
        const pre = document.querySelector("pre");
        pre.textContent = text;
        pre.scrollTop = pre.scrollHeight;
        document.querySelector("footer").textContent =
          `Actual captured output · ${Math.min(seconds, recorded).toFixed(1)} / ${recorded.toFixed(1)} seconds · No agent runtime or signed independent review`;
      },
      { text, seconds, recorded },
    );
    await page.screenshot({
      path: join(directory, `${String(frame).padStart(4, "0")}.png`),
    });
  }
  const child = spawn(
    "ffmpeg",
    [
      "-hide_banner",
      "-loglevel",
      "error",
      "-y",
      "-framerate",
      String(fps),
      "-i",
      join(directory, "%04d.png"),
      "-filter_complex",
      "[0:v]split[a][b];[a]palettegen=stats_mode=diff[p];[b][p]paletteuse=dither=none",
      "-loop",
      "0",
      output,
    ],
    { stdio: ["ignore", "ignore", "inherit"] },
  );
  const [code] = await once(child, "exit");
  if (code !== 0) throw new Error(`ffmpeg exit ${code}`);
  const hash = (p) => createHash("sha256").update(p).digest("hex");
  const metadata = {
    source: "examples/quickstart/demo.cast",
    source_sha256: hash(bytes),
    renderer: "tools/documentation/render-quickstart.mjs",
    renderer_sha256: hash(await readFile(fileURLToPath(import.meta.url))),
    file: "docs/images/quickstart.gif",
    sha256: hash(await readFile(output)),
    recorded_duration_seconds: recorded,
    rendered_duration_seconds: duration,
    fps,
    pixel_size: { width: 960, height: 540 },
    method:
      "Render real asciicast stdout chronologically; 2fps sampling at the end of each frame interval and a final hold. No invented output. See the source recording provenance for execution scope.",
  };
  await writeFile(
    output.replace(/\.gif$/, ".json"),
    JSON.stringify(metadata, null, 2) + "\n",
  );
  console.log(
    JSON.stringify({
      file: output,
      recorded_duration_seconds: recorded,
      rendered_duration_seconds: duration,
      sha256: metadata.sha256,
    }),
  );
} finally {
  await browser.close();
  await rm(directory, { recursive: true, force: true });
}
