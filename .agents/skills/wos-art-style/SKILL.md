---
name: wos-art-style
description: >-
  Local WOS art direction for creating, selecting, or editing illustrations,
  characters, and assets. Use for project illustration work or when the user
  mentions wos-art-style. Defines monochrome editorial black-ink drawings,
  expressive characters, and transparent backgrounds.
---

# WOS illustration art direction

## Canonical user prompt

Minimalist hand-drawn black ink cartoon illustration, organic slightly imperfect outlines, expressive pen-and-ink linework, subtle cross-hatching and diagonal hatching for shadows, playful character proportions, oversized rounded head, thin expressive limbs, sketchbook aesthetic, monochrome black and white, editorial illustration style, clean silhouette, isolated character, transparent background, no gradients, no colors, no photorealism.

## Project application

- Start with the canonical prompt and add the subject, action, and composition required by the task.
- Preserve slightly imperfect organic outlines, expressive pen linework, and subtle shadow hatching.
- Use playful proportions, oversized rounded heads, and thin expressive limbs.
- Keep a readable silhouette, isolated character, black ink, and transparent background; white may form internal drawing details.
- Do not use gradients, colors, or photorealism in illustrations.
- Request real file transparency when generating or editing images, without a white background or a drawn checkerboard.
- Existing SVGs are in `packages/wos-api/web/assets/robots/`, relative to the repository root. Reuse suitable assets and preserve their originals unless the owner explicitly requests replacement.
- This direction governs illustrations. It does not automatically change interface colors, typography, layout, or product behavior.
- The `chess-style` skill remains available for editorial composition; use this more specific direction for illustrations when combining both.
- By the owner's explicit decision, the interface uses the Woobe light operational palette in `packages/wos-api/web/assets/colors.css`: warm neutrals, charcoal, and copper, with orange reserved for the brand and semantic colors for state. This supersedes the gray/green suggestions in `chess-style` while retaining its editorial composition. See `docs/design/editorial/woobe-palette-source.md` for the supplied specification. Do not recolor robots.
- Create or modify assets only when requested; recording the style does not authorize redesigning existing assets.
- The active brand is the supplied hand-drawn logo from `Hand-Drawn-Orange-Chain-Icon.zip`, installed in `packages/wos-api/web/assets/brand/hand-drawn-chain/`. The owner's latest explicit direction requires only this symbol in the header, without WOS text or a typographic lockup. Preserve the original SVG paths, colors, and 1897 × 632 proportions. The link retains the accessible name WOS · home. Other product-name mentions use Instrument Serif; the previous modular kit remains a reference. See `docs/design/editorial/brand-kit/hand-drawn-chain.md`. The brand does not replace the robots.

## Required local art workflow

This skill is part of the repository; no external package or service is needed to record the art direction. Development agents must consult `AGENTS.md` and this skill **before** creating, selecting, editing, or integrating illustrations, including natural-language requests. In Cursor, `.cursor/rules/wos-art-style.mdc` also guides relevant art files.

### 1. Establish the intent

- Identify the subject, action, expression, framing, and operational context the illustration must communicate.
- First check whether one of the 20 existing robots in `packages/wos-api/web/assets/robots/` fits the intent.
- Treat the owner-supplied robot as the character reference: silhouette, rounded head, proportions, linework, and expressions. Do not replace it with a generic mascot.
- Do not create new assets when the request is only to document or use the art direction.

### 2. Compose the prompt without changing the style

Always begin with the **complete, literal text** under “Canonical user prompt,” without shortening or rewriting its qualifiers. Append only scene-specific parameters, for example:

- **Subject:** the existing WOS robot, preserving its identity.
- **Action:** reviewing task evidence beside a small board.
- **Composition:** isolated figure, complete silhouette inside the frame, room on every side, readable at small sizes.
- **Format:** real alpha transparency; vector SVG only when the resulting file contains actual vectors.

Negative constraints: colors, gradients, glow, 3D rendering, photorealism, geometrically perfect linework, scenery filling the entire image, opaque backgrounds, drawn checkerboards, illegible text, changed robot identity, and clipped elements. These constraints govern illustrations, **not** the interface color system or brand.

### 3. Produce, review, and integrate

1. Preserve the original asset; give an authorized variant its own name, preferably `NN_role_action.ext`, unless the owner explicitly requests replacing an existing file.
2. Check real transparency, a fully visible character, and the absence of residue or background rectangles.
3. Inspect at 64 px, 112 px, and the final screen size, checking hatching, proportions, and expression.
4. Preserve original proportions; avoid `object-fit: cover` when it clips the silhouette.
5. An SVG with an embedded PNG is a raster wrapper. Genuine vector SVGs require actual paths, strokes, and other SVG primitives.
6. Integrate only where needed in `packages/wos-api/web/assets/`: use `alt=""` for decorative artwork and appropriate alternative text for informative artwork.
7. Review affected desktop and mobile screens without changing states, permissions, workflows, or endpoints.

### Acceptance checklist

- [ ] Apply the original prompt literally and preserve the robot's identity.
- [ ] Use monochrome black ink, organic hand-drawn lines, and subtle hatching; no color, gradient, or photorealism.
- [ ] Preserve real transparency and the complete illustration.
- [ ] Describe the file's actual nature accurately, without presenting raster wrappers as vectors.
- [ ] Ensure readability and safe margins at the destination size.
- [ ] Preserve the brand, operational palette, and editorial layout when outside the requested scope.
- [ ] Review accessibility and affected screens.
