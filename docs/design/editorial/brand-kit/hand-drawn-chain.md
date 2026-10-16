# Hand-drawn chain identity

The owner supplied `Hand-Drawn-Orange-Chain-Icon.zip` after the matte-modal refinement and requested that the header show only the new logo. This direction supersedes the earlier symbol-plus-WOS wordmark. The home link contains only the supplied symbol; its accessible name remains **WOS · home**. The tagline, session status, sign-out control, and workspace behavior retain their existing roles.

| Original | Integration |
| --- | --- |
| `Hand-Drawn Orange Chain Icon.svg` | Header logo and exact compatibility copy at `/app/favicon.svg`; genuine vector artwork with two paths and a 1897 × 632 viewBox |
| `Hand-Drawn Orange Chain Icon.png` | Transparent 2172 × 724 PNG, also declared as the supplied touch-icon source |
| `Hand-Drawn-Orange-Chain-Icon.ico` | Supplied browser fallback, containing a 256 × 85 frame |

All three files are installed byte for byte under `packages/wos-api/web/assets/brand/hand-drawn-chain/`. Their aspect ratios and colors are preserved, including the SVG's orange `#FC5106` and light detail `#FFEAE1`. The header SVG renders at 120 px wide on desktop and 96 px on compact screens, with automatic proportional height; the existing wide-screen fluid unit scales it. The four internal openings have zero alpha, so the header paper is visible through them. No font, clipping, recoloring, or raster substitution is applied to the logo. The native icon files are rectangular as supplied; they are not presented as newly generated square platform icons.

The operational Woobe color tokens, robot illustrations, Instrument Serif product-name mentions outside the header, and matte modal surfaces remain unchanged. The previous kit's 94 media files remain intact as references. [The asset manifest](hand-drawn-chain-assets.json) records archive and original-file hashes; `.agents/skills/wos-art-style/SKILL.md` records the current owner direction so later work does not restore the superseded header wordmark.

## Verification

The brand browser journey checks the accessible home link, an image-only signature with no WOS/caption text, successful SVG decoding, original aspect ratio, and no page overflow at 1440/390/320 CSS px. It checks zero alpha at the four internal openings and no CSS background on the image. It decodes the declared SVG/ICO/PNG browser assets and compares the compatibility favicon with the supplied SVG bytes. The existing pending-read test retains exactly one on-page brand mark. The scale regression measures the logo together with the rest of the composition.

The complete gallery is refreshed because every page shares the header. See [captures](../README.md#main-pages), [source/capture hashes](../screenshots.json), and [validation](../validation.md). The original artwork files are the source of the identity; no instructions or executable content were included in the archive.
