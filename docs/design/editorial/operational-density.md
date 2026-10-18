# Stable operational density

The [newer Board gallery](board-communication/README.md) records current
actions, relationships and feedback while retaining these sizing rules.
The density-only verification figures below retain their identified source.

The interface uses bounded operational dimensions across desktop, tablet and
phone viewports. A larger monitor gives the workspace more room for content;
it does not multiply every font, control, illustration and panel.

This follows the owner's correction that the previous interface became too
large on both small and large screens. It supersedes the whole-interface
enlargement described in [Responsive composition scale](responsive-scale.md).
That document and its measurements remain historical evidence of the earlier
requirement, rather than the acceptance criteria for this revision.

## Reference measurements

Woobe is a sizing reference, not a runtime dependency or a replacement design
system. The inspected source is pinned to
[`3dbcb3e1a7559f4a50ebca320e411933d46e8012`](https://github.com/A1b3rt0M3rcad0/woobe/tree/3dbcb3e1a7559f4a50ebca320e411933d46e8012).
Its [shared foundations](https://github.com/A1b3rt0M3rcad0/woobe/blob/3dbcb3e1a7559f4a50ebca320e411933d46e8012/packages/woobe-ui/src/tokens/foundations.ts#L1)
define the following sizes. Pixel equivalents assume the usual 16 px browser
root used by that source; WOS retains its own 14 px root and existing font
families, so copying the numerical rem values would produce different sizes.

| Role | Woobe reference | Adopted WOS dimensions at normal text size |
| --- | --- | --- |
| Body and control text | 14 px, 20 px line height | 14 px root; detail prose 14 px with 1.65 line height |
| Supporting text | 12 px metadata; 13 px secondary labels | Existing contextual label hierarchy retained |
| Main page title | Bounded 30–36 px | 32 px; 28 px at mobile widths |
| Workspace empty-state title | Bounded page-title hierarchy | Same 32/28 px page-title token |
| Editorial section | 24 px | Area title 28 px; 24 px at mobile widths |
| Desired Outcome prose | 16 px reading text; 24 px editorial section | 24 px; 20 px at mobile widths |
| Navigation sidebar | 248 px | 244 px; 220 px at the existing smaller-desktop breakpoint |
| Global workspace header | 54 px | 56 px; 48 px in Outcome Board |
| Primary controls | 44 px; compact alternative 36 px | Existing minimum 44 px interactive targets retained |
| Inspector | Generic 352 px; workspace token bounded 320–360 px | 360 px docked region; overlay capped at 420 px and available width |
| Inspector heading | Network entity heading 14 px | 20 px editorial heading; normal body prose below |
| Map camera controls | Network controls 36 × 36 px | At least 44 × 44 px for existing touch and keyboard operation |

The generic Woobe inspector token is different from its
[Network editor](https://github.com/A1b3rt0M3rcad0/woobe/blob/3dbcb3e1a7559f4a50ebca320e411933d46e8012/packages/woobe-frontend/src/features/network/components/NetworkInspectorShell.tsx#L11),
which explicitly uses 520 px normally and 720 px when expanded. WOS keeps its
smaller existing inspection region instead of copying that editing surface.
Woobe's [Network header](https://github.com/A1b3rt0M3rcad0/woobe/blob/3dbcb3e1a7559f4a50ebca320e411933d46e8012/packages/woobe-frontend/src/pages/NetworkPage.tsx#L969)
uses a 14 px title, a minimum 56 px header and a minimum 32 px context row;
its [entity inspector](https://github.com/A1b3rt0M3rcad0/woobe/blob/3dbcb3e1a7559f4a50ebca320e411933d46e8012/packages/woobe-frontend/src/features/network/components/NetworkNodeInspector.tsx#L118)
uses compact headings, labels and field text. These distinctions guide the
operational hierarchy without removing WOS's serif identity.

## Sizing and interaction boundaries

`--wos-unit` is one CSS pixel. It no longer increases with viewport width.
The page, area and desired-Outcome text sizes use centralized custom properties
in `editorial.css`, with smaller values in the existing mobile media query.
Existing responsive layouts, column counts, navigation and authorizations
remain the responsibility of their original components.

Two local mobile refinements reduce the space occupied by navigation. The
collapsed specialist sidebar places the existing Workspace switcher and
Outcome navigation in two columns above one settings row. Expanding it still
reveals the existing search and Outcome list; switching, creation, settings and
developer controls remain available. The Board's existing heading actions use
three equal columns per row, preserving at least 44 px targets and allowing
labels to wrap. Neither refinement adds a view or changes the work hierarchy.

The mobile Connections filter omits its visible label to give the selected
value more room. Its existing accessible label remains associated with the
native select. The workspace empty-state heading uses the page-title token
instead of retaining a separate 44–64 px display treatment.

A CSS pixel is not a physical screenshot pixel. Device pixel ratio changes
raster resolution; it does not request a larger sidebar or font. Browser zoom
and enlarged text must continue to work normally: zoom changes the available
CSS viewport and visual size, and the interface reflows through its existing
responsive rules. There is no inverse zoom compensation, global transform or
automatic monitor-width enlargement.

The Board camera remains independent of interface density. Zooming, centering,
fitting and moving the map operate on the graph; they do not resize its header,
toolbar or inspector. A long record description uses readable utility text,
while its title retains the smaller editorial heading. Inspection must not
turn authored prose into another display headline or discard its content.

The paper grid, matte color tokens, Instrument Serif and DM Sans, original
robots, transparent branding and supplied post-it artwork retain their
existing identities. Larger viewports expose more usable workspace around
those elements. The Board still owns the viewport without a left sidebar;
Overview and specialist views retain their navigation. Existing inspector
docking, overlay boundaries, scrolling, forms and focus handling must survive
the sizing correction.

## Validation

Source `9d41b3aace7378ba7ed34948fb36db7f32464e47` passes 44 unit checks and
all 106 browser journeys in one clean full run (5.1m), without failures,
retries, skips or flaky results. Native web tests/vet and all 16 release and
compatibility checks pass. Earlier screenshots qualify their recorded source;
the [current gallery](density/README.md) qualifies this correction separately.

- Actual CSS viewports from 320 to 5120 px cover bounded fonts, headers,
  controls, sidebar and form widths, with no horizontal document overflow in
  any specialist view.
- Board checks cover long authored titles and descriptions, readable content,
  stable headers/toolbars, whole controls, scrolling, forms and focus return.
- Existing camera, permissions, persistence and recovery journeys pass.
- Native browser zoom at 125%/150%, enlarged text and separate DPR emulation
  record their actual CSS viewports and pixel ratios.
- Compact login retains credential and local-development access; expanded
  content and enlarged text keep natural scrolling when needed.

## Current gallery

The [27-image gallery and manifest](density/README.md) record the committed
source, binary, browser, served-asset comparisons, CSS viewport, device pixel
ratio and image hashes. Images come directly from the running application with
isolated illustrative data. The mobile rail measures 123–131 px high, while
the Board toolbar starts at 221 px in the two compact density scenarios.
Inspector title/body text stays at 20/14 px across every density scenario.
Short desktop windows and narrow phones use a bounded outer inspection scroller so
the heading and actions cannot squeeze the reading surface into a thin strip.

Local Chromium checks and visual inspection do not establish other-browser,
screen-reader or representative-user acceptance. Hosted CI qualifies the PR
head separately.

| Screen | Capture |
| --- | --- |
| QHD Overview | [2560 × 1440](density/screenshots/overview-2560x1440.png) |
| QHD Board and long Evidence description | [2560 × 1440](density/screenshots/board-2560x1440-inspector.png) |
| Mobile Overview and compact navigation | [390 × 844](density/screenshots/overview-390x844.png) |
| Mobile Board inspection | [390 × 844](density/screenshots/board-390x844-inspector.png) |
| Mobile login | [390 × 844](density/screenshots/login-390x844.png) |
| QHD Task form | [2560 × 1440](density/screenshots/task-form-2560x1440.png) |
