# README validation

This record covers the product explanation and workspace-tour follow-up committed at `8e2e83b` on 2026-10-17, using application source at `c321b95`. That baseline changed documentation, media and its recording helper. The subsequent licensing, distribution and managed Skill-notice changes are qualified separately in [licensing validation](licensing-validation.md); the captured tour keeps its original source provenance.

## Executed checks

| Check | Result and scope |
| --- | --- |
| Product explanation | Reviewed against the implemented architecture and agent workflow. The README explains delivery, assessment and explicit conclusion separately. OaaS is a developer integration pattern: the consuming product owns customer management, billing and external execution. Official agent Skills continue to operate through the CLI. |
| Current-source setup | README and getting-started clone commands target `master`; the agent guide identifies the merged centralized workflow as current source. The published v0.4.2 boundary remains explicit. A read-only GitHub release query still identifies v0.4.2 as the latest published release. No release or distribution was changed. |
| Real browser navigation | Seven actual desktop states pass against an isolated loopback server and SQLite database, seeded through public command APIs. Expected records, the primary Canvas Task and all three Work rows are visible. Visible images decode. Authenticated navigation has no failed same-origin responses, script errors, page-level horizontal overflow or domain mutation requests. |
| State semantics | One implementation Task is Done while the Outcome remains Active. Its version is unchanged during navigation, with no Assessment or Conclusion. The example report is illustrative; no refund implementation, external test execution or signed independent review is demonstrated. |
| Media | Independently decode all 56 GIF frames: 28 seconds, 1120 × 963 pixels, looping, 839,211 bytes. The PNG alternative is 1280 × 1100 pixels. Review the Canvas, Work and pending-criteria frames visually. The GIF holds seven genuine screenshots rather than representing continuous-time navigation. |
| Provenance | Verify hashes of GIF, poster, recorder and shared navigation helper. All 27 served top-level web inputs match checked-out source byte for byte. Preserve the original logo and queue illustration; the asset catalog records current media and labels the terminal recording as historical. |
| GFM layout | Render through the GitHub Markdown API. Inspect the output in Chromium at 1280, 390 and 320 CSS pixels using local styles approximating GitHub. All eight images decode, both detail sections expand, and page width stays within each viewport with details closed and open. Tables and code retain their own scrolling. This is a local layout check, not a claim of exact GitHub page styling. |
| Paths and anchors | Local Markdown/HTML targets in changed guides pass. Incoming README anchors are checked, including the preserved requirements/platforms anchor. |
| Static checks | JavaScript syntax for the recorder, JSON parsing, media/source hashes and Git whitespace checks pass. No dependencies added. |

The rendered preview uses the actual fetched live `master` CI badge SVG, cached locally without modification. Static version badges describe the published baseline; they do not imply publication of current source.

## Reproduce

Follow the [tour guide](images/workspace-tour.md) to capture a new isolated run. Its [machine-readable provenance](images/workspace-tour.json) records the command, source commit, browser, input hashes, stage timestamps and output hashes. Fresh runs contain new sample identifiers and capture times, so their media hashes will differ.

The [asset catalog](images/assets.json) distinguishes original artwork, the byte-identical logo alias, current tour and poster, static baseline badges and the retained terminal demonstration. Earlier terminal-example, container and distribution checks belong to the [2026-10-16 baseline](https://github.com/A1b3rt0M3rcad0/wos/blob/c321b951c23e08af62a32a92c1ac39794b4f0b37/docs/readme-validation.md); they were not repeated as part of this follow-up.

## Limits

The full Go, browser acceptance, race, native-platform, packaging and Compose suites were not rerun locally for these documentation changes. The actual seven-state browser journey and README rendering checks above are the executed scope. New PR checks report their own results separately.

The public image contains the earlier released interface. The tour shows current source with illustrative data, not a signed lifecycle acceptance or commercial OaaS offering. Other browser engines, physical devices, manual screen-reader use, production deployment and 1.0 certification are outside these checks. No release, container publication, deployment or PR merge is performed.
