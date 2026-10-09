import { message } from "./en-US.js";
import { copy } from "./en-US.js";
import { el, button } from "./ui.js";
import { kindName, shortId, display } from "./presentation.js";
let sequence = 0;
export const referenceKinds = [
  "outcome",
  "objective",
  "work_item",
  "issue",
  "blocker",
  "evidence",
  "artifact",
  "decision",
  "roadmap",
  "relation",
  "evidence_link",
];
// Every request is scoped; selected references survive pagination, never authorization failures.
export function referencePicker({
  label,
  kinds,
  scope,
  request,
  initial = [],
  multiple = false,
  required = false,
  lifecycle = "",
}) {
  const root = el("div", undefined, "reference-picker"),
    id = `reference-picker-${++sequence}`;
  const heading = el("label", label);
  heading.htmlFor = id;
  const input = el("input");
  input.id = id;
  input.type = "search";
  input.placeholder = copy.text.searchAuthorizedItems;
  input.setAttribute("role", "combobox");
  input.setAttribute("aria-autocomplete", "list");
  input.setAttribute("aria-expanded", "false");
  input.setAttribute("aria-controls", `${id}-results`);
  input.autocomplete = "off";
  const results = el("div", undefined, "reference-results");
  results.id = `${id}-results`;
  results.setAttribute("role", "listbox");
  results.setAttribute("aria-label", label);
  results.hidden = true;
  const selected = el("div", undefined, "reference-selections"),
    status = el("p", undefined, "field-help");
  status.setAttribute("role", "status");
  const more = button(copy.text.moreResults, () => search(true), "quiet");
  more.hidden = true;
  const reload = button(
    copy.text.reloadSearch,
    () => {
      cursor = "";
      search(false);
    },
    "quiet",
  );
  reload.hidden = true;
  let values = new Map(),
    items = [],
    cursor = "",
    run = 0,
    controller,
    timer,
    active = -1,
    disposed = false,
    valid = true;
  const key = (r) => `${r.kind}/${r.id}`;
  const path = `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}/references`;
  function renderSelected() {
    selected.replaceChildren();
    for (const candidate of values.values()) {
      const chip = el("span", undefined, "reference-chip");
      chip.append(
        el(
          "span",
          candidate.title ||
            `${kindName(candidate.ref.kind)} ${shortId(candidate.ref.id)}`,
        ),
        button(
          message(
            "remove8890",
            candidate.title || kindName(candidate.ref.kind),
          ),
          () => {
            values.delete(key(candidate.ref));
            renderSelected();
          },
          "quiet",
        ),
      );
      selected.append(chip);
    }
    input.required = required && !values.size;
    input.setCustomValidity(
      valid
        ? ""
        : copy.text.selectedReferencesAreUnavailableRemoveThemOrSearchAgain,
    );
  }
  function choose(candidate) {
    valid = true;
    if (!multiple) values.clear();
    values.set(key(candidate.ref), candidate);
    input.value = "";
    input.removeAttribute("aria-activedescendant");
    active = -1;
    renderSelected();
    if (!multiple) {
      results.hidden = true;
      more.hidden = true;
      input.setAttribute("aria-expanded", "false");
    } else search(false);
  }
  function render() {
    results.replaceChildren();
    items.forEach((candidate, i) => {
      const option = el("div", undefined, "reference-option");
      option.id = `${id}-option-${i}`;
      option.setAttribute("role", "option");
      option.setAttribute("aria-selected", String(i === active));
      option.append(
        el("strong", candidate.title || kindName(candidate.ref.kind)),
        el(
          "small",
          `${kindName(candidate.ref.kind)} · ${candidate.display_context || copy.text.outcome} · ${display(candidate.lifecycle)} · ${shortId(candidate.ref.id)}`,
        ),
      );
      option.onmousedown = (e) => e.preventDefault();
      option.onclick = () => choose(candidate);
      results.append(option);
    });
    results.hidden = false;
    input.setAttribute("aria-expanded", "true");
    more.hidden = !cursor;
  }
  async function search(append = false) {
    const generation = ++run;
    controller?.abort();
    controller = new AbortController();
    reload.hidden = true;
    input.removeAttribute("aria-activedescendant");
    status.textContent = copy.text.searchingThisOutcome;
    const query = new URLSearchParams({
      query: input.value.trim(),
      limit: "25",
    });
    query.set("kind", kinds.join(","));
    if (lifecycle) query.set("lifecycle", lifecycle);
    if (append && cursor) query.set("cursor", cursor);
    try {
      const page = await request(`${path}?${query}`, {
        signal: controller.signal,
      });
      if (disposed || generation !== run) return;
      items = append ? [...items, ...page.items] : page.items;
      cursor = page.next_cursor || "";
      active = -1;
      render();
      status.textContent = items.length
        ? message(
            "matchingItemsUseArrowKeysAndEnter",
            items.length,
            cursor ? copy.text.moreAvailable : "",
          )
        : copy.text.noMatchingAuthorizedItemsTryAnotherSearch;
    } catch (error) {
      if (disposed || generation !== run || error.name === "AbortError") return;
      results.hidden = true;
      more.hidden = true;
      input.setAttribute("aria-expanded", "false");
      status.textContent =
        error.status === 409 || error.status === 412
          ? copy.text.theOutcomeChangedReloadTheSearchYourSelectionsAre
          : error.message;
      reload.hidden = ![409, 412].includes(error.status);
      if (error.status === 403 || error.status === 401) {
        values.clear();
        valid = false;
        renderSelected();
      }
    }
  }
  input.oninput = () => {
    clearTimeout(timer);
    ++run;
    controller?.abort();
    cursor = "";
    timer = setTimeout(() => search(), 180);
  };
  input.onfocus = () => search();
  input.onkeydown = (e) => {
    if (e.key === "Escape" && !results.hidden) {
      e.stopPropagation();
      input.removeAttribute("aria-activedescendant");
      results.hidden = true;
      more.hidden = true;
      input.setAttribute("aria-expanded", "false");
      ++run;
      controller?.abort();
      return;
    }
    if (["ArrowDown", "ArrowUp"].includes(e.key)) {
      e.preventDefault();
      if (results.hidden) {
        search();
        return;
      }
      active = Math.max(
        0,
        Math.min(items.length - 1, active + (e.key === "ArrowDown" ? 1 : -1)),
      );
      if (items[active]) {
        input.setAttribute("aria-activedescendant", `${id}-option-${active}`);
        for (let i = 0; i < results.children.length; i++)
          results.children[i].setAttribute(
            "aria-selected",
            String(i === active),
          );
        results.children[active].scrollIntoView({ block: "nearest" });
      }
    }
    if (e.key === "Enter" && !results.hidden) {
      e.preventDefault();
      if (items[active]) choose(items[active]);
    }
  };
  for (const candidate of initial.filter(Boolean)) {
    const ref = candidate.ref || candidate;
    if (
      !kinds.includes(ref.kind) ||
      ref.namespace_id !== scope.namespace_id ||
      ref.outcome_id !== scope.outcome_id
    )
      continue;
    values.set(key(ref), { ref, title: candidate.title });
  }
  renderSelected();
  // Resolve preselected IDs on the server as well; never infer authority from cached labels.
  Promise.all(
    [...values.values()].map(async (candidate) => {
      const q = new URLSearchParams({
        kind: candidate.ref.kind,
        id: candidate.ref.id,
        limit: "1",
      });
      try {
        const page = await request(`${path}?${q}`);
        if (disposed) return;
        const found = page.items[0];
        if (!found) {
          values.delete(key(candidate.ref));
          valid = false;
        } else if (values.has(key(candidate.ref)))
          values.set(key(candidate.ref), found);
        renderSelected();
      } catch (error) {
        if (disposed) return;
        values.delete(key(candidate.ref));
        valid = false;
        status.textContent = error.message;
        renderSelected();
      }
    }),
  );
  root.append(heading, input, selected, results, more, reload, status);
  return {
    node: root,
    getCandidates: () => [...values.values()].map((x) => structuredClone(x)),
    get() {
      if (!valid)
        throw Error(copy.text.resolveUnavailableReferencesBeforeSaving);
      const refs = [...values.values()].map((x) => ({ ...x.ref }));
      if (required && !refs.length)
        throw Error(message("choose", label.toLowerCase()));
      return multiple ? refs : refs[0];
    },
    dispose() {
      disposed = true;
      ++run;
      controller?.abort();
      clearTimeout(timer);
    },
  };
}
