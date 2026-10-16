# Page and component review — 2026-10-16

The owner's feedback identified inconsistent button borders, repeated WOS symbols, limited use of the style's grids, clipped form/select edges, and underdeveloped areas of the page. The approved editorial direction, Woobe colors, Instrument Serif product name, and original robot artwork remain the basis of this revision.

## Review method and scope

The review followed the rendered hierarchy: page → workspace shell → Outcome → area → collection/detail → form → field group → control and its states. Source inspection covered both the dedicated human forms and the schema-based administration/technical forms, plus reference discovery, repeatable lists, roadmap rows, context/participant editors, signed review, conflict recovery, and the shared field renderer.

The `chess-style` references informed the composition: continuous paper, independent graphic and structural grids, typographic scale, small ink scenes, purposeful marginal notes, and readable static states. These are adapted to an operational workspace. The skill's original gray/green suggestions remain superseded by the owner's Woobe palette. Its evidence images are study material and are not embedded in the site.

## Findings from the original implementation

At a 1440 × 1000 viewport, the command dialog was capped at 900 px, while the form could occupy 848 px plus 64 px of dialog padding and 2 px of border. The resulting dialog had **14 px of extra outer scrolling**, in addition to the separately scrolling `#fields`. Its header, general help, fields, and footer therefore used incompatible height budgets. Mobile had a similar 2 px mismatch. Focusing a control could scroll either ancestor, and a select could meet the scroll boundary immediately above the footer.

Field labels inherited a 16 px bottom margin, controls added a 7 px top margin, and the form grid added another gap. Related selects occupied separate full-width rows. The settings dialog scrolled its entire header and save action out of view; its permission checkboxes formed a long single column even on desktop.

At page level, the graphic grid restarted inside the area introduction and disappeared from the actual workspace. The overview used detached sections with little structural connection. The official symbol appeared in the header, workspace selector, and pending-read state; chapter labels repeated the product name rather than identifying the area. Button variants mixed filled rectangles, outlined rectangles, circles, and joined tabs through overlapping component rules.

## Composition and interaction decisions

| Level | Result | Interaction with its children |
| --- | --- | --- |
| Global page | One WOS symbol with live Instrument Serif lettering in the header; browser icons retain the supplied brand | Workspace selection remains a labeled native select; it does not repeat the global symbol |
| Sign-in | Continuous fine grid across the paper, including the outer margins; wider desktop composition | Existing native credential form and static/no-JavaScript narrative remain accessible |
| Workspace | Continuous viewport-fixed 120 px graphic grid behind the page | Structural columns organize content independently; controls retain opaque reading surfaces |
| Sidebar | Serif workspace/Outcome names, decorative sequence numbers, ruled states, registration mark, and an organizer vignette with an italic note | Existing native selection/search and pressed-state semantics remain; mobile collapses its navigation and omits the large vignette; long lists retain their own scroll and reachable settings |
| Area introduction | Number and actual area name, serif headline, relevant robot, italic marginal note, small registration crosses | Decorations remain outside hit areas; a rule connects the introduction to its data |
| Overview | Connected editorial columns and a prominent desired-state passage; recent work spans the adjacent context/attention sections on desktop | The content uses actual counts and records; it becomes one ordered column at the existing breakpoint |
| Collections | Ruled rows and a framed List/Board switch | Row focus stays inside the collection; list overflow no longer clips its focus outline |
| Actions | Primary action filled in charcoal; secondary actions use a bottom rule; textual actions remain light; action menu uses serif rows | Hover, focus, selected, disabled, and pending states retain their existing behavior; labels and dispatch permissions are unchanged |
| Command and settings dialogs | Gridded heading with a small relevant robot, one scrolling body, separate persistent action footer | The form height subtracts the outer border; fields/errors/conflicts/history are inside the body, while close and save remain reachable |
| Detail drawer | Persistent heading and actions around one scrolling content region | Long descriptions, proof, contracts, and history scroll without a footer overlay or negative offsets |
| Fields | Label, control, then helper text; full rectangular borders and copper inset focus on text/select controls | Helpers are connected with `aria-describedby`; checkbox labels stay beside the control; native selection and validation are preserved |
| Related fields | Priority and initial state share a row on desktop; permissions use two columns | Both become one column below 600 px; field getters, selected values, and payloads remain unchanged |
| Nested editors | Disclosures use ruled boundaries, references keep their bounded listbox, and repeatable rows stay within the form body | Expanding instructions/Advanced, adding rows, choosing references, and entering dates share the same scroll context; no added global listener or animation loop |
| Pending read | Existing observer robot beside the live loading message | Visibility follows the real request; no second WOS symbol, artificial delay, or simulated progress |

The main graphic grid is the existing original SVG, scaled consistently across the operational surface. Registration crosses are local CSS geometry. No new illustration, third-party component package, color system, or backend capability was introduced.

## Fixed paper and sidebar follow-up

The owner subsequently asked for the grid to stay stationary while any content moves over it, and for a more expressive `chess-style` sidebar. The page now has one `body::before` layer with `position: fixed`, isolated behind the content and excluded from pointer events. Transparent article backgrounds and a translucent Woobe workspace wash on the sidebar expose this shared sheet. The original 120 px SVG tile stays unmodified. Dialogs and their scroll bodies use viewport-anchored background images with the same tile origin. Opaque field/control surfaces preserve readability.

This fixed behavior is an owner-requested implementation decision, not an assertion about motion in the skill's static reference images. No scroll listener, animation loop, artificial scrolling, or additional library is required. With `prefers-reduced-motion: reduce`, the page grid becomes document-relative and dialog backgrounds scroll locally; the existing animation/transition suppression remains.

The sidebar acts as the notebook's working margin: serif names distinguish authored work from utility labels; numbered entries and rules establish order; the selected Outcome retains its copper rail, tinted surface, and `aria-pressed` state. Indices are decorative and hidden from assistive technology. The original organizer robot and existing “Plan · execute · verify” note form one reserved composition above real settings actions. Mobile retains the native workspace selector and collapse button, horizontal Outcome navigation, and touch-sized settings controls. It omits the larger drawing/note to keep the working area available.

## Verification

The browser regression traverses the form's nested structure at 1440 × 1000, 390 × 844, 320 × 740, landscape 740 × 390, and 768 × 1024 with a doubled 28 px root font. It selects priority/state, expands instructions and Advanced, adds an instruction, selects an authorized Objective with the keyboard, focuses the native date field, and saves a real Task. The resulting detail must retain the authored instruction.

Focused controls are measured against their **scroll ancestor**, including their visible border/focus clearance, rather than only against the viewport. The test checks occlusion, footer overlap, extra outer-dialog scrolling, minimum control height, and page overflow. Settings checks cover its operation select, a permission checkbox at the end of the list, persistent save action, and the credential form's text field. Existing browser journeys cover checkbox submission, roadmap reordering, external context, conflicts, uncertain-response retries, reference authorization/pagination, signed independent review, and keyboard focus restoration.

The fixed-paper regression compares actual screenshot pixels in an unobstructed paper margin before and after 173 px of document scrolling at 1440 and 390 px widths. A shifted sample proves the comparison contains visible grid lines; geometry independently confirms that content moved and the desktop sidebar stayed in place. Reduced-motion checks verify that the paper then moves with the document. A dialog-body scroll also preserves the paper sample. The same journey selects a long mobile Outcome title with the keyboard and opens settings after scrolling a 15-Outcome desktop sidebar.

See [the executed validation record](validation.md) and [updated gallery](README.md#main-pages). Every capture now uses the actual viewport so that the fixed grid, sidebar, backdrop, and action footer represent what the user sees at that scroll position. Automated Chromium checks and visual inspection do not replace manual screen-reader or other-engine acceptance.
