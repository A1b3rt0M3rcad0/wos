# Supplied canvas paper artwork

The owner supplied `WOS_Postits_Canvas_Collection.zip` for the operational canvas. Its 160 original SVGs are installed under [the post-it asset directory](../../../../packages/wos-api/web/assets/postits/svg). No artwork was recolored, redrawn, optimized, or converted. The original upload remains available in the development attachments.

## Provenance and format

- Source archive SHA-256: `9b2ce7547e2662742c5fa578fe97575a9afe06c9f5cb557ada489d319fecb1bd`.
- The installed library is 625,344 bytes: 10 styles × 8 colorways × 2 proportions.
- Every SVG contains actual paths/circles, with transparent space outside its paper silhouette. The library has no embedded raster images, scripts, foreign objects, external resources, gradients, filters, or visible text.
- Intrinsic sizes are **800 × 800** (`square`) and **1000 × 730** (`wide`). Proportions and `viewBox` are preserved.
- The 160 raster counterparts and three preview PNGs remain in the supplied archive/extraction. They are not bundled into the application, avoiding 9,034,695 bytes of unused PNG assets.
- [assets.json](assets.json) records the 160 installed paths, original hashes, dimensions, safe areas, and inspected format. [SOURCE_SHA256SUMS.txt](SOURCE_SHA256SUMS.txt) covers the archive and every extracted source file.

The source [README](source/README.pt-BR.md), [manifest](source/manifest.json), and [palette file](source/tokens.css) are preserved byte for byte as source records. Their Portuguese descriptions and examples are not application copy or additional implementation instructions. The source README describes these as original assets created for this request.

## Deliberate use in the workspace

Paper notes represent functional Objectives, Tasks, and Issues on the canvas. They do not replace every surface. Outcome context, Roadmap bands, Evidence, history, toolbars, details, forms, and alternative lists keep the existing warm, ruled editorial surfaces.

| Entity             | Default source asset below `assets/postits/svg/` | Proportion | Content insets: top / right / bottom / left |
| ------------------ | ------------------------------------------------ | ---------- | ------------------------------------------- |
| Objective          | `square/tabbed/workspace.svg`                    | 1:1        | 22% / 14% / 14% / 14%                       |
| Task (`work_item`) | `wide/classic/surface.svg`                       | 1000:730   | 16% / 14% / 14% / 14%                       |
| Issue              | `wide/taped/workspace.svg`                       | 1000:730   | 22% / 14% / 14% / 14%                       |

Selection uses the same format/style's original `selected.svg`, accompanied by a visible focus/selection treatment and programmatic state. Paper hue is not operational state. Readiness, blockage, priority, review, and lifecycle remain explicit text/badges derived from persisted state. A red-looking note cannot establish a Blocker, and a taped note cannot confer authority.

The square Objective is a larger organizational note; the wider Task is a compact executable unit. The restrained tape treatment makes an Issue recognizable without changing its distinction from a real Blocker. Tab, pin, tape, folds and hatching stay outside live content. Shapes are not controls and do not imply new domain relations.

## Rendering contract

Use the original SVG as a decorative image/background with live semantic HTML above it. Keep `alt=""` and/or `aria-hidden="true"` for the artwork; the supplied SVG contains original descriptive metadata, including Portuguese, which must not become a repeated accessible label. Titles, status, actions and selection names come from the application.

Reserve the source aspect ratio and apply the manifest's safe areas. Never stretch a square paper into a wide card, crop it with `cover`, rotate the content to imitate a sketch, or put labels inside the SVG. Use `contain` or a ratio-preserving wrapper. Decorative art receives no pointer events. Keep focus outlines and contextual action targets clear of folds and clip boundaries. Long content opens in the accessible contextual panel rather than overflowing the note.

The canvas's internal zoom is separate from viewport responsiveness. Scale lengths through the existing `--wos-unit`/rem system before applying internal pan/zoom; do not globally zoom the page. On compact screens, the alternative list and contextual panel preserve complete information without requiring precise small-node interactions.

## Palette and text contrast

The source's eight paper fills match existing palette colors. Its `ink` corresponds to `--wos-foreground`; its `brand` corresponds to `--wos-brand-orange`. The source palette file is a provenance record, not another stylesheet to import: [colors.css](../../../../packages/wos-api/web/assets/colors.css) remains authoritative.

Calculated sRGB text contrast on the flat paper fill:

| Paper     | Recommended text |    Ratio |
| --------- | ---------------- | -------: |
| Surface   | Foreground       | 14.845:1 |
| Workspace | Foreground       | 13.609:1 |
| Canvas    | Foreground       | 12.891:1 |
| Selected  | Foreground       | 11.639:1 |
| Grid      | Foreground       |  7.993:1 |
| Copper    | Surface          |  5.927:1 |
| Brand     | Foreground       |  4.705:1 |
| Ink       | Surface          | 14.845:1 |

These ratios describe flat source colors, not complete accessibility certification or all text/control states. The current mapping uses neutral/selected papers; brand orange remains a scarce identity accent. Copper, brand and dark papers remain available originals rather than default node decoration. Focus, hover, small metadata and semantic badges still require their own rendered checks.

## Review evidence and limits

The three supplied previews and the three selected source counterparts were visually inspected. XML inspection covered all 160 SVGs and SHA-256 comparisons verified byte identity after copying. Source safe areas are the owner's recommendations, not measurements of arbitrary application content. Final node content, keyboard/touch targets, hover/focus, long titles, reduced motion, mobile alternatives and 1440/1920/2560/3840 viewport behavior must be verified in the integrated canvas.

The existing `chess-style` composition guidance and local `wos-art-style` were consulted. Paper materiality complements the fine grid and typography; the robot illustrations and hand-drawn orange identity remain separate, unchanged assets.

The new [canvas stylesheet](../../../../packages/wos-api/web/assets/canvas.css) uses this mapping and the existing typography/palette. An isolated Chromium review mounted the actual canvas renderer with illustrative records at 1440, 390, 320 and 2560 CSS px: no page errors or outer horizontal overflow occurred, and visible note content stayed within its safe area. A separate inspector styling fixture verified a 360-unit desktop panel with body scrolling and a full-width document-flow mobile panel, including long note titles. Those are component composition checks; they do not establish the integrated application's mutation, permission or recovery behavior. Product screenshots and end-to-end acceptance remain the integrating branch's responsibility.
