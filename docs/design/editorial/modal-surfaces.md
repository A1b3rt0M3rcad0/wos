# Matte workspace modals

The owner requested a color-only continuation of the workspace paper across every modal. The existing four native `<dialog>` roots cover action selection, human/technical command forms, entity details, and workspace administration. No modal markup or application JavaScript changes are needed.

| Role | Existing token | Coverage |
| --- | --- | --- |
| Main paper | `--wos-canvas` | All four modal roots, visible behind transparent headings and scrolling bodies |
| Secondary paper | `--wos-workspace` | Persistent footers, fieldsets, disclosures, context, detail sections, protocol panels, notices, error backgrounds, and conflict comparison sections |
| Raised reading surface | `--wos-surface` | Fields, including read-only fields; reference results; technical JSON; conclusion/relation cards; existing roadmap and contract/review cards |
| Neutral rules | `--wos-border` | Modal outline, field borders, footer rules, reference boundaries, card borders, and existing dividers |

`editorial.css` scopes the control-line alias and secondary surface rules to dialogs. Selection and hover still use the existing interaction tokens; semantic text and status rails retain their meaning. The token definitions in `colors.css`, typefaces, artwork, backdrop, grid image/attachment, and layout declarations are unchanged. Detail drawers retain their existing transparent scrolling grid and ungridded outer surface.

## Verification

The browser palette journey opens all four roots at **1440 × 1000** and **390 × 844**, expands the action menu and Task instructions, visits all four administration operations, and opens the technical command form. It checks main and nested paper colors, absence of pure-white element backgrounds, the original backdrop color, and focus inside the active dialog. Expanded instruction controls also retain their complete borders and accessible scroll position.

A separate local comparison swaps the current stylesheet for the pre-change `e4a44de` stylesheet through a same-origin CSS route on the same DOM, with fonts loaded and reduced motion enabled. All four modal roots and their descendants retain exactly equal rectangles, scroll metrics, display/position, typography, padding/margins/gaps, border widths, grid columns, overflow, and background image/size/attachment at widths **320, 390, 1440, and 2560 CSS px**: 16 identical comparisons. The [measurement record](modal-geometry.json) includes element counts; this one-off inspection is separate from the committed browser-test count.

The existing complete browser suite additionally covers keyboard focus/restoration, real saves, conflict recovery, uncertain retries, permissions, contract and signed-review details, button states, accessibility checks, fixed-paper scrolling, persistent actions, and responsive geometry. Its nested-form matrix includes 320 px, mobile landscape, and doubled root text; its scale matrix includes Full HD, 2K, and 4K. See [validation](validation.md) for executed results and limits.

Affected captures are refreshed in the [gallery](README.md#detail-forms-and-settings), including expanded forms, the 2K form, and mobile settings. The added mobile action/detail and desktop/mobile technical-command captures show the remaining modal families. [Capture provenance](screenshots.json) distinguishes the modal refresh from retained workspace-page captures.

To reproduce the modal palette check after building the native server, run from `tests/web`:

```sh
npx playwright test editorial.spec.mjs -g 'all modal families'
```
