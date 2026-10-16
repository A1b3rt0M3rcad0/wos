# Button and hover review

The owner requested a review of every button family, extending beyond the tightly painted area tabs and the low-contrast List/Board example. The review follows the existing `chess-style` guidance for [links and actions](../../../.agents/skills/chess-style/references/MOTION_AND_INTERACTIONS.md): functional accents, visible focus, stable hit areas, and a usable static state. Woobe colors and the approved editorial typography remain the visual system.

## Findings

The structural stylesheet and editorial theme both defined hover backgrounds. The generic dark button hover could reach an unselected view switch whose text remained dark. Area tabs had horizontal padding of zero, so their colored hover stopped at the label. Other families had similar local exceptions: recent-work entries had zero horizontal padding; compact search, header, metric, reference-removal, and filter controls inherited smaller hit areas. The board's actual `.board .item` buttons retained an older hover translation and shadow; styling an unused `.board-card` selector did not remove it.

## Component inventory and decisions

| Family / renderer | Rest and interaction | Spacing and selection |
| --- | --- | --- |
| Sign-in, submit/save/confirm, creation actions | Charcoal primary and hover/press variants with light text | At least 44 px high; action labels have reserved padding; existing validation/pending/disabled behavior remains |
| Header, sidebar settings, search, refresh/copy, secondary actions | Transparent utility button; soft copper-tinted hover and a fine rule where appropriate | Padded labels; mobile Sign out retains a 44 px target; registration marks stay decorative |
| Six area tabs | Quiet labels, copper selected label/rule, lightly tinted hover | 16 px on each side of the label; small consistent gaps; 48 px height; focus stays inside the scrolling strip |
| List/Board | Transparent unselected choice; charcoal selected choice with light lettering | Padded 44 px choices in a fine frame; hover remains legible on both selected and unselected choices |
| Collection category tabs | Neutral labels, copper selected rule and selected wash | Padded 44 px choices; text/count grouping and pressed-state semantics remain |
| Outcome navigation | Serif titles, numbered entries, soft selection and copper rail | Existing padded rows, native mobile collapse, and independent long-list scrolling remain |
| Overview metrics, context counts, quick actions and recent work | Utility/text-action states; data keeps its own hierarchy | Counts and recent-work entries have internal space around the entire hover surface; metric links retain generous targets |
| Collection rows, criterion rows and board cards | Light tinted hover; board border gains copper | Stable size and position; no hover translation or lifted shadow; text/badge meanings remain |
| Action menu, including technical actions | Ruled serif choices with a subtle tinted hover | 18 px horizontal padding; selected action dispatch and permission checks remain |
| Dialog close, add and other icon buttons | Quiet square target; neutral control wash on hover | At least 44 × 44 px; centered glyph, visible focus, no circular outline |
| Human forms, repeatable lists, schema forms and conflict recovery | Shared primary/utility styles | Add/remove/retry/confirm retain their real actions; existing single-scroll form layout remains |
| Roadmap, context and participant editors; signed contracts/review | Source review confirms these use the shared button renderers and utility variants | Existing workflow tests retain reordering, typed values, authority, and explicit save/review semantics |
| Reference chips and result options | Padded chip removal; tinted option hover; outlined keyboard-active option | Removal has a 44 px target; native combobox/listbox behavior and authorization remain |
| Disclosures and links | Padded disclosure hover; link underline; visible keyboard focus | Native markers and semantic controls remain; decoration does not move or cover labels |

## Shared state implementation

The theme owns interaction styling. Legacy hover rules were removed from `style.css`. Each button resolves its background and foreground from shared state variables. Component variants define idle, hover, and pressed colors; a more specific component selector therefore cannot accidentally keep dark text over an unrelated dark hover background. The same variables retain the selected List/Board and primary creation styles.

Hover rules apply only with a fine pointer and hover capability. Disabled buttons do not enter hover or pressed states. Press feedback changes color without moving the target. Keyboard focus remains independent of hover, with inset outlines inside scrolling navigation/collections. Reduced motion suppresses transitions. Native form controls retain their full borders and existing focus treatment.

## Verification scope

The interaction regression discovers real visible buttons and disclosures in sign-in, every area, list/board, task detail, an expanded Task form with a selected reference, settings operations, and the action menu. It verifies nominal foreground/background contrast of at least 4.5:1 in idle, hover, and pressed states, stable target geometry, label insets, and disabled-hover stability for each encountered button variant. Color calculations composite CSS ancestor backgrounds; they do not certify every pixel of decorative grid lines, illustration content, or nested semantic badges.

A touch-enabled browser context at 390 and 320 px checks compact controls, absence of persistent mouse-hover styling, keyboard focus, and page overflow. Existing regressions exercise the other shared editor workflows, permissions, reference discovery, signed review, form clipping, enlarged text, reduced motion, and the full viewport matrix. Source review and these checks do not imply an exhaustive test of every possible user record or manual accessibility acceptance.

See [the gallery](README.md#button-interaction-states) for actual hover/focus captures and [validation](validation.md) for the executed suite and browser limits.
