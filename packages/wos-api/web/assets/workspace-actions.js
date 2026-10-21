import { humanActions } from "./actions.js";

export function workspaceActionName(name = "") {
  return name.replace(/([a-z0-9])([A-Z])/g, "$1_$2").toLowerCase();
}

// Server capabilities describe current domain gates. This is only a join with
// presentation metadata; the mutation pipeline still rechecks every condition.
export function workspaceActions(view, fallback = []) {
  if (!view) return fallback;
  const items = new Map(fallback.map((action) => [action.name, action]));
  for (const capability of view.capabilities || []) {
    const name = workspaceActionName(capability.action_id);
    const presentation = humanActions[name];
    if (!presentation) continue;
    if (name === "create_work_item" && capability.target_lifecycle) {
      // The target lifecycle is part of the create intent; preserve both
      // authorized variants and remove any unqualified fallback action.
      items.delete(name);
      const target = capability.target_lifecycle;
      const label = `${presentation.label} · ${target === "todo" ? "To do" : "Backlog"}`;
      items.set(`${name}:${target}`, { name, ...presentation, ...capability, label });
      continue;
    }
    items.set(name, { name, ...presentation, ...capability });
  }
  return [...items.values()];
}

// New projections preserve uint64 values as decimal strings. Legacy commands
// retain their numeric wire format without a float64 round trip.
export function commandForSchema(value, schema) {
  if (value == null) return value;
  if (schema?.type === "integer" && typeof value === "string" && /^[0-9]+$/.test(value)) {
    if (typeof JSON.rawJSON === "function") return JSON.rawJSON(value);
    const numeric = Number(value);
    if (Number.isSafeInteger(numeric)) return numeric;
    throw new Error("This browser cannot safely encode the observed version. Use a current browser or the CLI.");
  }
  if (Array.isArray(value)) return value.map((item) => commandForSchema(item, schema?.items));
  if (typeof value === "object") return Object.fromEntries(Object.entries(value)
    .map(([key, item]) => [key, commandForSchema(item, schema?.properties?.[key])]));
  return value;
}
