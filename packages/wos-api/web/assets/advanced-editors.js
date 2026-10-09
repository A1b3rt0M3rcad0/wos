import { copy } from "./en-US.js";
import { el, button, inputField } from "./ui.js";
export function externalContextEditor(initial = {}) {
  const node = el("section"),
    rows = [];
  node.append(
    el("p", copy.text.consumerAddressOnlyTheseValuesNeverGrantAccessOr),
  );
  function add(key = "", value = "") {
    const row = el("div", undefined, "context-edit-row"),
      name = inputField(copy.text.contextKey, {
        value: key,
        required: true,
      }),
      type = inputField(copy.text.valueType, {
        options: [
          ["string", copy.text.text],
          ["number", copy.text.number],
          ["boolean", copy.text.yesNo],
          ["null", copy.text.empty],
        ],
        value: value === null ? "null" : typeof value,
      }),
      text = inputField(copy.text.contextValue, {
        value: value === null ? "" : String(value),
      });
    const model = { row, name, type, text };
    rows.push(model);
    row.append(
      name.node,
      type.node,
      text.node,
      button(
        copy.text.removeContextEntry,
        () => {
          rows.splice(rows.indexOf(model), 1);
          row.remove();
        },
        "quiet",
      ),
    );
    node.insertBefore(row, addButton);
  }
  const addButton = button(copy.text.addContextEntry, () => add(), "quiet");
  node.append(addButton);
  for (const [key, value] of Object.entries(initial)) add(key, value);
  return {
    node,
    get() {
      const result = {};
      for (const { name, type, text } of rows) {
        const key = name.get().trim(),
          value = text.get();
        if (
          !key ||
          key.startsWith("wos.") ||
          key.length > 256 ||
          Object.hasOwn(result, key)
        )
          throw Error(copy.text.contextKeysMustBeUniqueNonemptyAndNotReserved);
        const kind = type.get();
        let scalar = value;
        if (kind === "number") {
          scalar = Number(value);
          if (!value.trim() || !Number.isFinite(scalar))
            throw Error(copy.text.enterAFiniteNumericContextValue);
        } else if (kind === "boolean") {
          if (!["true", "false"].includes(value))
            throw Error(copy.text.useTrueOrFalseForABooleanContextValue);
          scalar = value === "true";
        } else if (kind === "null") scalar = null;
        Object.defineProperty(result, key, {
          value: scalar,
          enumerable: true,
          configurable: true,
        });
      }
      if (
        rows.length > 64 ||
        new TextEncoder().encode(JSON.stringify(result)).length > 8192
      )
        throw Error(copy.text.externalContextExceedsTheSupportedSize);
      return result;
    },
  };
}
export function participantEditor(initial = []) {
  const node = el("section"),
    rows = [];
  node.append(
    el(
      "p",
      copy.text
        .participantsDescribeResponsibilityAssignmentNeverGrantsALeaseContract,
    ),
  );
  function add(value = { kind: "agent" }) {
    const row = el("div", undefined, "participant-edit-row"),
      kind = inputField(copy.text.participantType, {
        options: [
          ["human", copy.text.person],
          ["agent", copy.text.agent],
          ["service", copy.text.service],
          ["automation", copy.text.automation],
          ["external_system", copy.text.externalSystem],
        ],
        value: value.kind,
      }),
      provider = inputField(copy.text.participantProvider, {
        value: value.provider,
        required: true,
      }),
      identity = inputField(copy.text.participantIdentity, {
        value: value.id,
        required: true,
      });
    const model = { row, kind, provider, identity };
    rows.push(model);
    row.append(
      kind.node,
      provider.node,
      identity.node,
      button(
        copy.text.removeParticipant,
        () => {
          rows.splice(rows.indexOf(model), 1);
          row.remove();
        },
        "quiet",
      ),
    );
    node.insertBefore(row, addButton);
  }
  const addButton = button(copy.text.addParticipant, () => add(), "quiet");
  node.append(addButton);
  for (const value of initial) add(value);
  return {
    node,
    get() {
      return rows.map(({ kind, provider, identity }) => ({
        kind: kind.get(),
        provider: provider.get(),
        id: identity.get(),
      }));
    },
  };
}
