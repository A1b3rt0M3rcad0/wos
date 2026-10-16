# Local workspace fonts

- Instrument Serif regular and italic, Latin subset: `@fontsource/instrument-serif@5.3.0`, https://www.npmjs.com/package/@fontsource/instrument-serif
- DM Sans variable weight, normal, Latin subset: `@fontsource-variable/dm-sans@5.3.0`, https://www.npmjs.com/package/@fontsource-variable/dm-sans

These files were taken from npm package archives with npm integrity verification. They are licensed under SIL Open Font License 1.1; retain `instrument-serif-OFL.txt` and `dm-sans-OFL.txt` when redistributing them. The fonts are embedded by the existing Go web asset loader and need no external requests. System fonts handle unsupported glyphs and failed font loads.

The Latin subset matches the official en-US UI. Authored text is preserved, with fallback for other scripts. The exact files and SHA-256 values are recorded in `docs/design/editorial/assets.json` at the repository root.
