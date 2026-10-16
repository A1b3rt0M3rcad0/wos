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

Every Map card uses the supplied paper itself as its surface, with live content above it. History, toolbars, contextual details, forms and the accessible List alternative keep their warm editorial surfaces. The mapping covers all rendered Map cards.

| Entity             | Default source asset below `assets/postits/svg/` | Proportion | Content insets: top / right / bottom / left |
| ------------------ | ------------------------------------------------ | ---------- | ------------------------------------------- |
| Outcome            | `wide/pinned/surface.svg`                        | 1000:730   | 22% / 14% / 14% / 14%                       |
| Objective          | `square/tabbed/workspace.svg`                    | 1:1        | 22% / 14% / 14% / 14%                       |
| Task (`work_item`) | `wide/curled/surface.svg`                        | 1000:730   | 16% / 14% / 17% / 14%                       |
| Issue              | `wide/taped/workspace.svg`                       | 1000:730   | 22% / 14% / 14% / 14%                       |
| Blocker            | `wide/torn/workspace.svg`                        | 1000:730   | 16% / 14% / 14% / 14%                       |
| Evidence           | `wide/notebook/surface.svg`                      | 1000:730   | 22% / 14% / 14% / 14%                       |
| Artifact           | `wide/stacked/surface.svg`                       | 1000:730   | 16% / 14% / 14% / 14%                       |
| Decision           | `wide/dogear/workspace.svg`                      | 1000:730   | 16% / 14% / 14% / 14%                       |
| Phase / fallback   | `wide/rounded/workspace.svg`                     | 1000:730   | 16% / 14% / 14% / 14%                       |
| Milestone          | `wide/pinned/surface.svg`                        | 1000:730   | 22% / 14% / 14% / 14%                       |
| Plan reference     | `wide/classic/workspace.svg`                     | 1000:730   | 16% / 14% / 14% / 14%                       |

Selection uses the same format/style's original `selected.svg`, an underlined title and programmatic pressed state. Keyboard focus stays on the semantic controls inside the paper. An outer rectangular frame no longer competes with the original silhouette. Forced-colors mode uses system surfaces and a system selection outline. Paper hue is not operational state. Readiness, blockage, priority, review, and lifecycle remain explicit text/badges derived from persisted state. A red-looking note cannot establish a Blocker, and a taped note cannot confer authority.

The square Objective is an organizational note; the wider Task is a compact executable unit with a visible curled corner. The taped Issue remains distinct from the torn Blocker. Planning notes retain explicit Draft/publication/activation labels; their paper does not imply an operational dependency. Tab, pin, tape, folds and hatching stay outside live content. Shapes do not imply new domain relations.

## Rendering contract

The renderer uses an original SVG in a `.canvas-node-paper` image beneath live semantic HTML. Its `alt=""`, `aria-hidden="true"` and disabled dragging keep the art decorative: the supplied SVG contains original descriptive metadata, including Portuguese, which must not become a repeated accessible label. Titles, status, actions and selection names come from the application.

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

The supplied previews were visually inspected. XML inspection covered all 160 SVGs and SHA-256 comparisons verified byte identity after copying. A browser regression decodes the original image for every mapped card kind and the fallback, compares served bytes with the provenance hashes, and checks source proportions, safe areas, transparent live content, selected variants, keyboard focus and compact Map/List operation. Source safe areas are the owner's recommendations, not measurements of arbitrary authored content.

The existing `chess-style` composition guidance and local `wos-art-style` were consulted. Paper materiality complements the fine grid and typography; the robot illustrations and hand-drawn orange identity remain separate, unchanged assets.

The [canvas stylesheet](../../../../packages/wos-api/web/assets/canvas.css) uses this mapping and the existing typography/palette. Isolated Chromium composition checks cover 1440, 390, 320 and 2560 CSS px, including body scrolling in the desktop inspector and document-flow mobile details. Integrated application captures and real-server workflows are recorded separately in the [gallery](../README.md#screenshots) and [verification record](../validation.md), with their measured scope and remaining manual limits.
