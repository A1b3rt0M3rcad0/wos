# Operational Kanban — real-browser captures

> Historical qualification/presentation. The owner's subsequent focused-modal
> follow-up supersedes the presentation below. See [current decisions](../simplification.md)
> and [ADR-041](../../../adr/0041-focused-kanban-modal-and-backlog-planning.md).
> Receipts below retain their recorded source; they do not qualify the new implementation.


These viewport captures show actual public commands and persisted state in an
isolated test installation. Original typography, warm paper, branding and robot
assets are retained. They are not design mockups or fabricated state transitions.

| CSS viewport | First item | Acquired execution | Explicit completion form | Persisted accepted column |
|---|---|---|---|---|
| 1280×720 | [Item](item-1280.png) | [Work](executing-1280.png) | [Form](completion-1280.png) | [Accepted](accepted-1280.png) |
| 2560×1440 | [Item](item-2560.png) | [Work](executing-2560.png) | [Form](completion-2560.png) | [Accepted](accepted-2560.png) |
| 3840×2160 | [Item](item-3840.png) | [Work](executing-3840.png) | [Form](completion-3840.png) | [Accepted](accepted-3840.png) |
| 390×844 | [Item](item-390.png) | [Work](executing-390.png) | [Form](completion-390.png) | [Accepted](accepted-390.png) |

Each image has a neighboring JSON receipt with source, source working-tree status,
measured CSS viewport and Chromium version. The manifest binds the stamped binary,
asset files and image hashes; it does not claim production-environment testing.
No live credentials or customer data are included.

The Kanban is a nonmodal workspace. Selecting a card keeps its lanes mounted and
opens a compact inspector: docked on wide screens, bounded on HD, and a reading
panel on mobile. The operation form is the only native modal. Its editable body
scrolls without the Save footer covering the final field.

[Before workspace mode](../before-workspace-mode/README.md) records the same Task
journey before this change. [Manifest](manifest.json) identifies the exact current
runtime source; later documentation-only commits retain identical runtime assets.

Native 125%, 150%, 200% and 400% browser zoom is verified separately by changing
Chromium's Settings zoom and asserting the effective CSS viewport and DPR. These
four viewport sizes are not substitutes for that zoom test.

[Operate the interface](../README.md) · [Current audit](../workspace-mode-audit.md) · [Qualification](../validation.md)
