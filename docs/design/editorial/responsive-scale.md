# Responsive composition scale

The approved reference is the existing 1440 × 1000 **CSS-pixel viewport**, recorded in `screenshots.json` for `20-fixed-paper-scrolled.png`. Its header is 76 CSS px high, its sidebar 244 px wide, and its paper tile 120 px. These browser dimensions establish the reference; the uploaded image's raster dimensions alone do not establish CSS size.

Above 1440 CSS px, one reference pixel becomes `clamp(1px, 100vw / 1440, 2.6666666667px)`. Both existing stylesheets express their authored pixel dimensions through `--wos-unit`, with a literal 1 px fallback. The 14 px root follows that unit, coordinating existing rem typography with controls, sidebar, padding, margins, gaps, maximum widths, artwork, grid tiles, borders, and focus indicators. Existing fluid headline clamps use `--wos-vw`: one reference vw is 14.4 reference pixels, so headings stop growing with the rest of the composition at the cap.

| CSS viewport | Component scale | Sidebar | Header | Paper tile |
| --- | --- | --- | --- | --- |
| 1440 × 1000 | 1.00× | 244 px | 76 px | 120 px |
| 1920 × 1080 | 1.33× | 325.33 px | 101.33 px | 160 px |
| 2560 × 1440 | 1.78× | 433.78 px | 135.11 px | 213.33 px |
| 3840 × 2160 | 2.67× | 650.67 px | 202.67 px | 320 px |

The cap also applies beyond 3840 CSS px. This is width-driven sizing in normal document flow, with no zoom, global transform, resize listener, markup changes, new columns, or larger-screen rearrangement. The previous 1600 px padding and board-minimum jumps is replaced by the continuous scale. Existing smaller-screen media queries retain their thresholds and behavior. The donut's viewBox already scales its text, so its internal type stays in original SVG coordinates above the reference to avoid scaling twice.

Different aspect ratios expose different amounts of the same document. The 16:9 desktop sizes all have the same effective 1440 × 810 composition; they naturally show less vertical content than the taller 1440 × 1000 reference. Native scrollbars, selection arrows, and device-pixel border rounding remain browser-rendered.

## Verification

After each viewport change, the geometry helper explicitly loads the resolved font faces before sampling ch-based headings and other dimensions. This provides the same font readiness as the viewport captures.

The `large viewports` browser journey measures rendered positions, dimensions, padding, fonts, and grid column counts for sign-in, overview, metrics, illustration, SVG chart, and Task form. It checks all six areas for page overflow at the four requested resolutions, preserves the five-column Work board and persistent form actions, and verifies the cap at 5120 px. Layout-box comparisons allow less than 3 reference pixels for border and line-layout rounding, with a 10 px allowance for the whole main document's accumulated height and the measured native scrollbar-gutter difference for horizontal dimensions inside a dialog body. Font-size comparisons allow less than 0.1 reference pixels. Intrinsic heading and text-button widths follow browser glyph advances: verify their font sizes, padding, heading line counts, and parent/text containment rather than requiring those widths to multiply exactly. Headless Chromium 149/151 rounds the login's unchanged 12ch cap to 576 px at 103.68 px type and 756 px at 138.24 px type; this is the same proportional font-size growth with different glyph rounding. The regression passes on supplemental Chromium Headless 149 and the capture browser Chromium 154.

An optional local comparison loads the original declarations after the current styles in the same DOM, keeping the same font faces, records, focus, scroll position, and browser. It requires byte-identical viewport PNGs for sign-in and for overview/Task forms at 1440, 1280, 768, 390, and 320 px, including a 28 px root at 768 px. Prepare the two baseline CSS files from pre-change commit `314f71ce4b786ed397286dd854193d79ab884bcb` in a temporary directory, then run:

```sh
WOS_SCALE_BASELINE_DIR=/tmp/wos-scale-reference \
WOS_TEST_CHROMIUM_PATH=/usr/bin/chromium \
npm exec -- playwright test editorial.spec.mjs -g 'large viewports'
```

The six scale captures are actual browser viewports, with scroll distances scaled from the reference's 173 px. The 2K form and board supplement the four overview sizes; the board is scrolled into view. Overview captures clear transient focus and pointer hover, while the form explicitly shows keyboard focus on Initial state. See the [gallery](README.md#responsive-scale) and [capture provenance](screenshots.json).
