# Third-party and earlier-license notices

The [WOS Sustainable Use License](LICENSE) applies only to material lawfully supplied under it. It does not replace or narrow licenses for dependencies, vendored assets, separately licensed source or earlier WOS material. Keep each applicable upstream notice and license text with the material it covers.

## Earlier WOS material

WOS releases and source revisions were previously distributed under Apache-2.0. Those grants remain effective for the material they cover, including that material when retained in a later distribution. [LICENSES/Apache-2.0.txt](LICENSES/Apache-2.0.txt) preserves the original text. New material offered only under the current license is not automatically covered by the earlier grant. Historical release artifacts and their notices must not be rewritten.

## Go dependencies and runtime

The source inventory is [go.mod](https://github.com/A1b3rt0M3rcad0/wos/blob/master/go.mod), [go.sum](https://github.com/A1b3rt0M3rcad0/wos/blob/master/go.sum) and the maintained [dependency guide](https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/dependencies.md). Native/npm distributions include a generated `third-party-notices/` directory with the actual module inventory, available module license texts and Go runtime notices. Container builds retain the same generated notices. The release build's `tools/distribution/notices.sh` determines the inventory for that artifact; this overview is not a replacement for those texts.

## Browser fonts and other supplied material

Instrument Serif and DM Sans retain the SIL Open Font License 1.1 and notices under `packages/wos-api/web/assets/fonts/`. The web build embeds and serves those existing license files. Style packages, owner-supplied source documents and any other material carrying a separate applicable notice retain it; do not overwrite their notices with the current WOS license.

## JavaScript tooling

Build and browser-test dependencies retain the licenses in their own packages. A development lockfile's upstream license metadata describes that dependency, not WOS licensing. Production launchers do not acquire ownership of dependencies merely because they package or invoke them.

## Local product-design skill

`.agents/skills/product-design-and-ux/` preserves Magnus Hedemark's MIT-licensed
skill package from `magnus919/agent-skills`, revision
`496c76b8e9b7d9303b8f7ed941fccf256f2486ec`. Its upstream license and attribution
are retained in that directory's `LICENSE.md` and `SOURCE.md`. It is development
guidance, not a new WOS runtime dependency.

## Combined distributions

Preserve this notice together with `LICENSE`, `TRADEMARKS.md`, `NOTICE.md`, `LICENSES/Apache-2.0.txt` and every applicable generated or upstream notice. An integrator adding components must also carry their required notices. This document neither claims an exhaustive legal audit nor grants rights in material the project is not authorized to license.
