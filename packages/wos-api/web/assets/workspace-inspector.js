// The canvas presents the existing detail and command surfaces inline. Their
// nodes, listeners, form values and pending intentions remain owned by app.js.
export function createWorkspaceInspector({ host, dialogs, onClose } = {}) {
  if (!host || !dialogs?.detail || !dialogs?.command || !dialogs?.action)
    throw new TypeError(
      "An inspector host and all three dialogs are required.",
    );

  const document = host.ownerDocument;
  const surfaces = {
    detail: { dialog: dialogs.detail, className: "detail" },
    form: { dialog: dialogs.command, className: "form" },
    actions: { dialog: dialogs.action, className: "actions" },
  };
  const aliases = {
    detail: "detail",
    details: "detail",
    form: "form",
    command: "form",
    action: "actions",
    actions: "actions",
  };
  let enabled = false,
    disposed = false,
    active = null,
    hasDetails = false;
  const returnTargets = new Map();

  for (const [name, surface] of Object.entries(surfaces)) {
    const slot = document.createElement("div");
    slot.className = `canvas-inspector-slot canvas-inspector-${surface.className}`;
    slot.dataset.surface = name;
    slot.tabIndex = -1;
    slot.hidden = true;
    surface.slot = slot;
    host.append(slot);
  }
  host.hidden = true;

  function surfaceName(name) {
    return name == null ? active : aliases[name];
  }

  function visibleTarget(target) {
    if (!target?.isConnected || typeof target.focus !== "function")
      return false;
    if (target.closest?.("[hidden], dialog:not([open])")) return false;
    return target.getClientRects().length > 0;
  }

  function focus(target) {
    if (visibleTarget(target)) target.focus({ preventScroll: true });
    else if (active) surfaces[active].slot.focus({ preventScroll: true });
  }

  function render() {
    host.hidden = !enabled || !active;
    if (active) host.dataset.surface = active;
    else delete host.dataset.surface;
    for (const [name, { slot }] of Object.entries(surfaces)) {
      slot.hidden = name !== active;
      const heading = slot.querySelector(".dialog-heading h2[id]");
      if (heading) {
        slot.setAttribute("role", "region");
        slot.setAttribute("aria-labelledby", heading.id);
      }
    }
  }

  function show(name) {
    if (!enabled || disposed) return false;
    if (active === "form" && name !== "form") return false;
    const target = document.activeElement;
    // A reference inside the inspector can replace the selected detail, but
    // closing the inspector should still return to the originating canvas node.
    if (
      target &&
      target !== document.body &&
      (name !== "detail" || !host.contains(target))
    )
      returnTargets.set(name, target);
    if (name === "detail") hasDetails = true;
    active = name;
    render();
    surfaces[name].slot.focus({ preventScroll: true });
    return true;
  }

  function closeSurface(name, reason = "close") {
    name = surfaceName(name);
    if (!enabled || !name || !surfaces[name]) return false;
    if (name !== active && !(name === "detail" && hasDetails)) return false;
    const wasActive = name === active;
    if (name === "detail") hasDetails = false;
    if (wasActive) active = name !== "detail" && hasDetails ? "detail" : null;
    render();
    if (wasActive) focus(returnTargets.get(name));
    returnTargets.delete(name);
    onClose?.(name, { reason, nextSurface: active });
    return true;
  }

  function closeAll() {
    if (!enabled || !active) return false;
    const previous = active;
    const target = returnTargets.get("detail") || returnTargets.get(previous);
    active = null;
    hasDetails = false;
    render();
    focus(target);
    returnTargets.clear();
    onClose?.(previous, { reason: "reset", nextSurface: null });
    return true;
  }

  function setEnabled(value) {
    value = Boolean(value);
    if (disposed || value === enabled) return !disposed;
    // app.js prevents leaving an open form. Never remove the native open
    // attribute or close a dialog here: that would dispatch business listeners
    // or leave a modal top-layer/backdrop active while its children are moved.
    if (value && Object.values(surfaces).some(({ dialog }) => dialog.open))
      return false;
    enabled = value;
    for (const { dialog, slot } of Object.values(surfaces)) {
      const source = value ? dialog : slot;
      const target = value ? slot : dialog;
      target.append(...Array.from(source.childNodes));
    }
    active = null;
    hasDetails = false;
    returnTargets.clear();
    render();
    return true;
  }

  function escape(event) {
    if (
      !enabled ||
      !active ||
      event.key !== "Escape" ||
      event.defaultPrevented ||
      event.isComposing ||
      event.repeat
    )
      return;
    const target = event.target;
    // Autocomplete and native select interactions own their first Escape.
    // The inspector is nonmodal, so a nested modal must also retain control.
    if (
      target.closest?.(
        'dialog[open], [aria-modal="true"], [role="listbox"], [role="menu"], select',
      ) ||
      target.closest?.('[role="combobox"][aria-expanded="true"]')
    )
      return;
    event.preventDefault();
    event.stopPropagation();
    closeSurface(active, "escape");
  }
  host.addEventListener("keydown", escape);

  return {
    get enabled() {
      return enabled;
    },
    setEnabled,
    showDetails: () => show("detail"),
    showForm: () => show("form"),
    showActions: () => show("actions"),
    close: closeSurface,
    closeAll,
    isVisible: (name) =>
      enabled && Boolean(active) && active === surfaceName(name),
    dispose() {
      if (disposed) return;
      setEnabled(false);
      disposed = true;
      host.removeEventListener("keydown", escape);
      for (const { slot } of Object.values(surfaces)) slot.remove();
    },
  };
}
