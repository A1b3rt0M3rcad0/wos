import {
  icon,
  display,
  kindName,
  tone,
  badge,
  shortId,
  date,
  donut,
} from "./presentation.js";
const $ = (id) => document.getElementById(id);
const state = {
  namespace: "",
  outcome: null,
  snapshot: null,
  section: "work_items",
  view: "summary",
  generation: 0,
  renderId: 0,
  loaded: new Map(),
  boardLoaded: new Map(),
  selected: null,
  entities: new Map(),
  criterion: null,
  catalog: [],
  next: "",
  pending: null,
};
const labels = {
  blocked_ref: "Item impedido",
  cause_ref: "Causa do impedimento",
  resolution_summary: "Como foi resolvido",
  severity: "Gravidade",
  propagation: "Alcance",
  captured_at: "Data da observação",
  source_ref: "Fonte",
  source_version: "Versão da fonte",
  evidence_id: "Evidência",
  artifact_id: "Artefato",
  base_revision_number: "Revisão de partida",
  active_plan_references: "Trabalho do plano ativo",
  required_criteria_met: "Critérios comprovados",
  required_criteria_waived: "Critérios dispensados",
  work_completion: "Trabalho concluído",
  objective_completion: "Objetivos alcançados",
  required_objectives_completion: "Objetivos obrigatórios alcançados",
  work_items: "Trabalho",
  objectives: "Objetivos",
  ready_work: "Pronto para execução",
  blocked_work: "Bloqueado",
  in_progress_work: "Em execução",
  scheduled_work: "Programado",
  attention_needed_work: "Precisa de atenção",
  issues: "Problemas",
  blockers: "Impedimentos",
  decisions: "Decisões",
  evidence: "Evidências",
  artifacts: "Artefatos",
  active_roadmaps: "Planos ativos",
  conclusion_contestations: "Contestações",
  timeline: "Histórico",
  roadmaps: "Todos os planos",
  graph: "Relações",
  title: "Título",
  description: "Descrição",
  desired_state: "Resultado esperado",
  reason: "Motivo",
  rationale: "Justificativa",
  result_summary: "Resultado do trabalho",
  required: "Obrigatório",
  priority: "Prioridade",
  lifecycle: "Estado",
  verification_mode: "Modo de verificação",
  expected_version: "Versão observada",
  expected_roadmap_version: "Versão do plano",
  expected_draft_version: "Versão do rascunho",
  criterion_revision: "Revisão do critério",
  ttl_seconds: "Duração da reserva (segundos)",
  owner: "Item avaliado",
  scope: "Contexto",
  namespace_id: "Namespace",
  outcome_id: "Resultado",
  objective_id: "Objetivo",
  work_item_id: "Trabalho",
  claim_id: "Reserva",
  fencing_token: "Proteção da reserva",
  criterion_id: "Critério",
  evidence_ids: "Evidências usadas",
  evaluator_ref: "Avaliador externo",
  target_ref: "Item relacionado",
  source_ref: "Origem da relação",
  reference_ref: "Referência do plano",
  parent_objective_id: "Objetivo superior",
  nodes: "Itens do plano",
  after: "Ordenação do plano",
  parent_node_key: "Grupo superior",
  node_key: "Chave do item",
  planned_start: "Início planejado",
  planned_end: "Fim planejado",
  metadata: "Informações adicionais",
  revision_number: "Revisão publicada",
  kind: "Tipo",
  id: "Identificador",
  provider: "Origem",
  result: "Avaliação",
  stance: "Posição da evidência",
  artifact_type: "Tipo de artefato",
  evidence_type: "Tipo de evidência",
  producer_ref: "Produzido por",
  name: "Nome",
  uri: "Endereço",
  chosen_alternative: "Alternativa escolhida",
  alternatives: "Alternativas",
  proposal: "Proposta",
  required_for_outcome: "Obrigatório para o resultado",
};
const choices = {
  severity: ["major", "critical", "minor", "informational"],
  propagation: ["direct", "subtree"],
  evidence_type: [
    "test_result",
    "inspection",
    "measurement",
    "attestation",
    "source",
    "external_evaluation",
  ],
  priority: ["normal", "high", "critical", "low"],
  verification_mode: ["attestation", "evidence_review", "external_evaluation"],
  result: ["met", "not_met", "inconclusive", "waived"],
  stance: ["supports", "contradicts", "neutral"],
  lifecycle: ["todo", "backlog"],
  scope_kind: ["outcome", "objective"],
  node_type: ["reference", "phase", "milestone"],
  kind: ["human", "agent", "service", "automation", "external_system"],
};
const human = (s) =>
  labels[s] ||
  {
    work_item: "Trabalho",
    objective: "Objetivo",
    outcome: "Resultado",
    roadmap: "Plano",
    created_at: "Criado em",
    updated_at: "Atualizado em",
    archived_at: "Arquivado em",
    version: "Versão",
    principal_id: "Participante",
    actor_ref: "Autoria",
    concluded_at: "Concluído em",
    owner_ref: "Item",
    entity_ref: "Item",
    recorded_at: "Registrado em",
    node_type: "Tipo de item",
    content_hash: "Assinatura do conteúdo",
    checksum: "Checksum",
    obligations: "Obrigações",
    assessments: "Avaliações",
    criteria: "Critérios",
    external_context: "Contexto externo",
    assignee_refs: "Responsáveis",
    not_before: "Disponível a partir de",
    reference: "Referência",
  }[s] ||
  display(s);
const verbs = {
  create: "Criar",
  register: "Registrar",
  add: "Adicionar",
  update: "Editar",
  activate: "Ativar",
  start: "Iniciar",
  claim: "Reservar",
  renew: "Renovar reserva de",
  reclaim: "Assumir reserva expirada de",
  release: "Liberar reserva de",
  complete: "Concluir",
  achieve: "Certificar",
  reopen: "Reabrir",
  archive: "Arquivar",
  unarchive: "Desarquivar",
  record: "Registrar",
  publish: "Publicar",
  replace: "Editar",
  resolve: "Resolver",
  retract: "Retirar",
  cancel: "Cancelar",
  remove: "Remover",
  retire: "Desativar",
  supersede: "Substituir",
  investigate: "Investigar",
  open: "Abrir",
  set: "Definir",
  clear: "Limpar",
  delete: "Excluir",
  redeliver: "Reenviar",
};
const actionSubjects = {
  criterion_assessment: "avaliação do critério",
  criterion: "critério",
  criterion_definition: "definição do critério",
  roadmap_draft: "rascunho do plano",
  roadmap_revision: "revisão do plano",
  work_item: "trabalho",
  outcome: "resultado",
  objective: "objetivo",
  evidence_link: "vínculo de evidência",
  evidence: "evidência",
  artifact: "artefato",
  decision: "decisão",
  issue: "problema",
  blocker: "impedimento",
  roadmap: "plano",
  relation: "relação",
  work_item_assignees: "responsáveis pelo trabalho",
  trigger: "sinal de integração",
};
const actionName = (name) => {
  const [verb, ...subject] = name.split("_");
  return `${verbs[verb] || display(verb)} ${actionSubjects[subject.join("_")] || display(subject.join("_"))}`;
};
async function request(path, options = {}) {
  const response = await fetch(path, {
    credentials: "same-origin",
    ...options,
    headers: { "Content-Type": "application/json", ...options.headers },
  });
  const text = await response.text();
  let data;
  try {
    data = JSON.parse(text);
  } catch {
    data = { error: { message: text } };
  }
  if (!response.ok) {
    const error = new Error(
      data.error?.message || `Resposta ${response.status}`,
    );
    error.code = data.error?.code;
    error.status = response.status;
    throw error;
  }
  return data;
}
const { api_prefix: api } = await request("/app/config");
function notice(message, error = false) {
  $("notice").textContent = message;
  $("notice").classList.toggle("error", error);
}
function report(error) {
  notice(
    error.code === "version_conflict" || error.code === "precondition_failed"
      ? "O estado mudou. Atualize e revise a alteração antes de tentar novamente."
      : error.message,
    true,
  );
}
const base = () => `${api}/namespaces/${state.namespace}/outcomes`;
const outcomeBase = () => `${base()}/${state.outcome.id}`;
function el(tag, text, className) {
  const node = document.createElement(tag);
  if (text !== undefined) node.textContent = text;
  if (className) node.className = className;
  return node;
}
async function connect(namespace) {
  state.generation++;
  state.namespace = namespace;
  $("connect").hidden = true;
  $("workspace").hidden = false;
  state.catalog = (await request(`${api}/commands`)).commands;
  await discover();
  $("context-name").textContent =
    $("namespace").selectedOptions[0]?.textContent || "Espaço de trabalho";
}
async function namespaces() {
  const result = await request(`${api}/namespaces`);
  $("namespace").replaceChildren();
  for (const n of result.items) {
    const option = el("option", n.name);
    option.value = n.id;
    $("namespace").append(option);
  }
  if (result.items.length) await connect(result.items[0].id);
}
$("login").onsubmit = async (event) => {
  event.preventDefault();
  try {
    await request("/session", {
      method: "POST",
      body: JSON.stringify({ token: $("token").value }),
    });
    $("token").value = "";
    await namespaces();
  } catch (e) {
    $("token").setCustomValidity(e.message);
    $("token").reportValidity();
    $("token").oninput = () => $("token").setCustomValidity("");
  }
};
$("local").onsubmit = async (event) => {
  event.preventDefault();
  try {
    const id = $("local-ns").value.trim();
    const option = el("option", id);
    option.value = id;
    $("namespace").replaceChildren(option);
    await connect(id);
  } catch (e) {
    report(e);
    $("connect").hidden = false;
  }
};
$("logout").onclick = async () => {
  await request("/session", { method: "DELETE" }).catch(() => {});
  location.reload();
};
$("namespace").onchange = () => {
  state.outcome = null;
  state.snapshot = null;
  $("workspace-loading").hidden = true;
  state.entities.clear();
  state.selected = null;
  $("outcome").hidden = true;
  $("empty").hidden = false;
  connect($("namespace").value).catch(report);
};
$("search").onsubmit = (event) => {
  event.preventDefault();
  discover().catch(report);
};
async function discover(append = false) {
  const generation = state.generation;
  const q = new URLSearchParams({ limit: "25", text: $("text").value });
  if ($("lifecycle").value) q.set("lifecycle", $("lifecycle").value);
  if (!$("archived").checked) q.set("archived", "false");
  if (append && state.next) q.set("cursor", state.next);
  const page = await request(`${base()}?${q}`);
  if (generation !== state.generation) return;
  if (!append) $("outcomes").replaceChildren();
  for (const item of page.items) {
    const button = el("button", undefined, "item");
    button.dataset.outcomeId = item.id;
    button.classList.toggle("selected", state.outcome?.id === item.id);
    button.setAttribute("aria-pressed", String(state.outcome?.id === item.id));
    const copy = el("span", undefined, "nav-item-text");
    copy.append(
      el("strong", item.title),
      el(
        "small",
        `${display(item.lifecycle)}${item.archived_at ? " · Arquivado" : ""}`,
      ),
    );
    button.append(el("span", undefined, "nav-dot"), copy);
    button.onclick = () => openOutcome(item).catch(report);
    $("outcomes").append(button);
  }
  if (!page.items.length && !append)
    $("outcomes").append(el("p", "Nenhum resultado encontrado."));
  state.next = page.next_cursor;
  $("more-outcomes").hidden = !state.next;
}
$("more-outcomes").onclick = () => discover(true).catch(report);
async function openOutcome(outcome) {
  state.generation++;
  document.querySelector(".sidebar").classList.add("navigation-collapsed");
  $("navigation-toggle").setAttribute("aria-expanded", "false");
  state.entities.clear();
  state.loaded.clear();
  state.boardLoaded.clear();
  state.outcome = outcome;
  state.selected = outcome;
  state.criterion = null;
  state.view = "summary";
  state.snapshot = null;
  $("item-search").value = "";
  $("item-priority").value = "";
  $("detail-dialog").close();
  $("command-dialog").close();
  for (const button of $("outcomes").children) {
    button.classList.toggle(
      "selected",
      button.dataset.outcomeId === outcome.id,
    );
    button.setAttribute(
      "aria-pressed",
      String(button.dataset.outcomeId === outcome.id),
    );
  }
  $("empty").hidden = true;
  $("outcome").hidden = true;
  $("workspace-loading").hidden = false;
  const generation = state.generation;
  try {
    await refresh();
  } catch (error) {
    if (generation === state.generation) $("empty").hidden = false;
    throw error;
  } finally {
    if (generation === state.generation) $("workspace-loading").hidden = true;
  }
}
async function refresh() {
  if (!state.outcome) return discover();
  const generation = state.generation;
  const path = outcomeBase();
  const [snapshot, live] = await Promise.all([
    request(`${path}/continuity?limit=25`),
    request(path),
  ]);
  if (generation !== state.generation) return;
  if (
    live.outcome_revision !== undefined &&
    live.outcome_revision !== snapshot.outcome_revision
  ) {
    const changed = new Error(
      "O estado mudou durante a leitura. Atualize para obter uma visualização coerente.",
    );
    changed.code = "precondition_failed";
    throw changed;
  }
  state.snapshot = snapshot;
  state.outcome = live.value || live;
  state.loaded.clear();
  state.boardLoaded.clear();
  state.entities.clear();
  for (const section of Object.values(snapshot.sections))
    for (const item of section) {
      const e = item?.ref ? item : item?.work_item || item?.current;
      if (e?.ref) state.entities.set(`${e.ref.kind}/${e.ref.id}`, e);
    }
  $("empty").hidden = true;
  $("outcome").hidden = false;
  $("title").textContent = snapshot.outcome.title;
  $("state").textContent = display(snapshot.outcome.lifecycle);
  $("state").dataset.lifecycle = snapshot.outcome.lifecycle;
  $("state").className = `badge ${tone(snapshot.outcome.lifecycle)}`;
  $("outcome-key").textContent = shortId(snapshot.outcome.ref.id);
  $("revision").textContent =
    `Revisão ${snapshot.outcome_revision} · ${date(snapshot.evaluated_at)}`;
  renderMetrics();
  renderTabs();
  renderSummary();
  applyView();
  await renderSection();
  notice(
    snapshot.truncated
      ? "Visualização parcial. Há mais itens nas seções; use “Carregar mais”."
      : "Estado compartilhado atualizado.",
  );
}
$("refresh").onclick = () => refresh().catch(report);
$("navigation-toggle").onclick = () => {
  const sidebar = document.querySelector(".sidebar");
  sidebar.classList.toggle("navigation-collapsed");
  $("navigation-toggle").setAttribute(
    "aria-expanded",
    String(!sidebar.classList.contains("navigation-collapsed")),
  );
};
const operationalSections = [
  "backlog_work",
  "waiting_scope_work",
  "waiting_dependencies_work",
  "scheduled_work",
  "ready_work",
  "in_progress_work",
  "blocked_work",
  "attention_needed_work",
  "done_work",
  "cancelled_work",
];
const sectionIcons = {
  work_items: "work_item",
  objectives: "objective",
  issues: "issue",
  blockers: "blocker",
  evidence: "evidence",
  artifacts: "artifact",
  decisions: "decision",
  roadmaps: "roadmap",
  active_roadmaps: "roadmap",
  active_plan_references: "roadmap",
  timeline: "timeline",
  graph: "graph",
  conclusion_contestations: "alert",
};
const sections = [
  "work_items",
  "objectives",
  "ready_work",
  "blocked_work",
  "in_progress_work",
  "issues",
  "blockers",
  "decisions",
  "evidence",
  "artifacts",
  "active_roadmaps",
  "active_plan_references",
  "roadmaps",
  "conclusion_contestations",
  "timeline",
  "graph",
];
function renderTabs() {
  $("tabs").replaceChildren();
  for (const section of sections) {
    const button = el("button", undefined, "quiet");
    button.setAttribute("aria-label", human(section));
    button.append(
      icon(sectionIcons[section] || "work_item"),
      el("span", human(section)),
    );
    if (state.snapshot.counts[section] !== undefined)
      button.append(
        el("span", String(state.snapshot.counts[section]), "tab-count"),
      );
    button.dataset.section = section;
    button.classList.toggle("selected", section === state.section);
    button.setAttribute("aria-pressed", String(section === state.section));
    button.onclick = () => navigateSection(section);
    $("tabs").append(button);
  }
}
function applyView() {
  $("summary-view").hidden = state.view !== "summary";
  $("items-view").hidden = state.view === "summary";
  for (const button of $("view-tabs").children) {
    button.classList.toggle("selected", button.dataset.view === state.view);
    button.setAttribute(
      "aria-pressed",
      String(button.dataset.view === state.view),
    );
  }
  const board = state.view === "board";
  $("tabs").hidden = board;
  $("add-item").textContent =
    board ||
    ["work_items", "ready_work", "blocked_work", "in_progress_work"].includes(
      state.section,
    )
      ? "+ Criar trabalho"
      : `+ ${actionName(sectionCommand())}`;
  $("add-item").hidden = !board && !sectionCommand();
}
function navigateSection(section) {
  state.section = section;
  state.view = "list";
  $("item-search").value = "";
  $("item-priority").value = "";
  applyView();
  renderSection().catch(report);
}
for (const button of $("view-tabs").children) {
  button.prepend(icon(button.dataset.view));
  button.onclick = () => {
    state.view = button.dataset.view;
    applyView();
    renderSection().catch(report);
  };
}
$("item-search").oninput = () => renderSection().catch(report);
$("item-priority").onchange = () => renderSection().catch(report);
const createForSection = {
  work_items: "create_work_item",
  objectives: "create_objective",
  issues: "create_issue",
  blockers: "create_blocker",
  evidence: "register_evidence",
  artifacts: "register_artifact",
  decisions: "create_decision",
  roadmaps: "create_roadmap",
  ready_work: "create_work_item",
  blocked_work: "create_work_item",
  in_progress_work: "create_work_item",
};
function sectionCommand() {
  return createForSection[state.section] || "";
}
function createItem(command) {
  state.selected = { ...state.outcome, _kind: "outcome" };
  state.criterion = null;
  openCommands(command);
}
$("add-item").onclick = () =>
  createItem(state.view === "board" ? "create_work_item" : sectionCommand());
$("empty-create").onclick = () => {
  state.selected = null;
  openCommands("create_outcome");
};
function renderMetrics() {
  $("metrics").replaceChildren();
  for (const key of [
    "work_completion",
    "objective_completion",
    "required_criteria_met",
    "required_objectives_completion",
    "required_criteria_waived",
  ]) {
    const m = state.snapshot.progress[key];
    if (!m) continue;
    const card = el("div", undefined, "metric");
    const label = el("div", undefined, "metric-label");
    label.append(
      icon(
        key.includes("criteria")
          ? "check"
          : key === "work_completion"
            ? "work_item"
            : "objective",
      ),
      el("span", human(key)),
    );
    const value = el(
      "strong",
      m.value === null ? "Não se aplica" : `${Math.round(m.value * 100)}%`,
      m.value === null ? "not-applicable" : "",
    );
    const progress = el("progress");
    progress.max = m.denominator || 1;
    progress.value = m.numerator;
    progress.setAttribute(
      "aria-label",
      `${human(key)}: ${m.numerator} de ${m.denominator}`,
    );
    card.append(
      label,
      value,
      progress,
      el(
        "small",
        `${m.numerator} de ${m.denominator} ${key.includes("criteria") ? "critérios" : "itens"}`,
      ),
    );
    $("metrics").append(card);
  }
}
function summaryCard(title, section) {
  const card = el("section", undefined, "summary-card");
  const heading = el("div", undefined, "card-heading");
  heading.append(el("h2", title));
  if (section) {
    const link = el("button", "Ver todos →", "text-button");
    link.onclick = () => navigateSection(section);
    heading.append(link);
  }
  card.append(heading);
  return card;
}
function renderSummary() {
  const root = $("summary-content");
  root.replaceChildren();
  const s = state.snapshot,
    c = s.counts;
  const purpose = summaryCard("O que queremos alcançar");
  purpose.classList.add("summary-purpose-card");
  purpose.append(
    el(
      "p",
      state.outcome.desired_state ||
        state.outcome.description ||
        state.outcome.title,
      "summary-purpose",
    ),
  );
  const quick = el("div", undefined, "summary-quick");
  for (const [label, command] of [
    ["Adicionar objetivo", "create_objective"],
    ["Criar trabalho", "create_work_item"],
    ["Registrar evidência", "register_evidence"],
  ]) {
    const b = el("button", label, "quiet");
    b.onclick = () => createItem(command);
    quick.append(b);
  }
  purpose.append(quick);
  root.append(purpose);
  const status = summaryCard("Visão geral do trabalho", "work_items");
  const done = c.done_work || 0,
    active = c.in_progress_work || 0,
    blocked = (c.blocked_work || 0) + (c.attention_needed_work || 0),
    total = c.work_items || 0;
  const parts = [
    {
      title: "A fazer / planejado",
      count: Math.max(
        0,
        total - done - active - blocked - (c.cancelled_work || 0),
      ),
      color: "#d6d2c8",
      tone: "",
    },
    { title: "Em execução", count: active, color: "#e97335", tone: "orange" },
    {
      title: "Impedimentos / atenção",
      count: blocked,
      color: "#c75858",
      tone: "red",
    },
    { title: "Concluído", count: done, color: "#339d71", tone: "green" },
    {
      title: "Cancelado",
      count: c.cancelled_work || 0,
      color: "#8e82b3",
      tone: "violet",
    },
  ];
  const body = el("div", undefined, "summary-stats");
  body.append(donut(parts, total));
  const legend = el("div", undefined, "legend");
  for (const part of parts) {
    const row = el("div", undefined, "legend-row");
    row.append(
      el("span", undefined, `legend-dot ${part.tone}`),
      el("span", part.title),
      el("strong", String(part.count)),
    );
    legend.append(row);
  }
  body.append(legend);
  status.append(body);
  root.append(status);
  const proof = summaryCard("Contexto para continuar");
  const counts = el("div", undefined, "summary-counts");
  for (const [key, title] of [
    ["objectives", "Objetivos"],
    ["evidence", "Evidências"],
    ["roadmaps", "Planos"],
  ]) {
    const b = el("button");
    b.append(
      el(
        "strong",
        key === "roadmaps"
          ? String(c.active_roadmaps || 0)
          : String(c[key] || 0),
      ),
      el("span", key === "roadmaps" ? "Planos ativos" : title),
    );
    b.onclick = () => navigateSection(key);
    counts.append(b);
  }
  proof.append(
    counts,
    el(
      "p",
      "Concluir trabalho não certifica o resultado. Revise as obrigações e registre a conclusão explicitamente.",
    ),
  );
  root.append(proof);
  const recent = summaryCard("Trabalho em foco", "work_items");
  recent.classList.add("summary-recent");
  const items = [
    ...(s.sections.ready_work || []),
    ...(s.sections.in_progress_work || []),
    ...(s.sections.blocked_work || []),
  ].slice(0, 3);
  if (!items.length) items.push(...(s.sections.work_items || []).slice(0, 3));
  for (const item of items) recent.append(itemView(item));
  if (!items.length)
    recent.append(el("p", "Seu próximo passo começa com um item de trabalho."));
  root.append(recent);
  if (blocked || c.conclusion_contestations) {
    const alert = summaryCard(
      "Itens que precisam de atenção",
      c.conclusion_contestations ? "conclusion_contestations" : "blocked_work",
    );
    alert.classList.add("summary-alert");
    alert.append(
      el(
        "p",
        `${blocked} itens impedidos ou com reserva para revisar. ${c.conclusion_contestations || 0} contestações registradas.`,
      ),
    );
    root.append(alert);
  }
}
function titleOf(ref) {
  if (!ref) return "Item";
  if (ref.kind === "outcome" && ref.id === state.outcome?.id)
    return state.outcome.title;
  return (
    state.entities.get(`${ref.kind}/${ref.id}`)?.title ||
    `${kindName(ref.kind)} · ${shortId(ref.id)}`
  );
}
function referenceButton(ref) {
  const b = el("button", titleOf(ref), "text-button");
  b.onclick = () => showEntity(ref).catch(report);
  return b;
}
function operational(item) {
  const e = item.work_item || item.current || item;
  return (
    item.state ||
    item.readiness ||
    operationalSections
      .flatMap((key) => state.snapshot.sections[key] || [])
      .find((x) => x.work_item.ref.id === e.ref?.id)?.state
  );
}
function matches(item) {
  const e = item.work_item || item.current || item;
  const query = $("item-search").value.trim().toLocaleLowerCase("pt-BR");
  return (
    (!query ||
      `${e.title || ""} ${item.plan_label || ""} ${shortId(e.ref?.id)} ${display(e.lifecycle || "")}`
        .toLocaleLowerCase("pt-BR")
        .includes(query)) &&
    (!$("item-priority").value || e.priority === $("item-priority").value)
  );
}
function itemView(item, mode = "card") {
  if (item.slot) {
    const card = el("div", undefined, "item");
    const ref = {
      namespace_id: state.namespace,
      outcome_id: state.outcome.id,
      kind: "roadmap",
      id: item.slot.roadmap_id,
    };
    card.append(
      referenceButton(ref),
      el(
        "small",
        `Revisão ${item.slot.revision_number} · ${item.node_count} itens publicados`,
      ),
      badge("active"),
    );
    return card;
  }
  if (item.entity_ref && item.event_type) return timelineView(item);
  if (item.owner_ref && item.kind) {
    const card = el("div", undefined, "item");
    card.append(el("h3", display(item.kind)), referenceButton(item.owner_ref));
    const evidence = item.evidence_id || item.evidence_ref?.id;
    if (evidence)
      card.append(
        referenceButton({
          namespace_id: state.namespace,
          outcome_id: state.outcome.id,
          kind: "evidence",
          id: evidence,
        }),
      );
    card.append(
      el(
        "p",
        "A conclusão permanece no histórico. Revise a nova observação antes de registrar outra decisão.",
      ),
    );
    return card;
  }
  const entity = item.work_item || item.current || item;
  if (!entity.ref) {
    const card = el("div", undefined, "item");
    card.append(readable(item));
    return card;
  }
  state.entities.set(`${entity.ref.kind}/${entity.ref.id}`, entity);
  const status = operational(item);
  const button = el("button", undefined, mode === "list" ? "list-row" : "item");
  button.dataset.entityId = entity.ref.id;
  button.onclick = () => showEntity(entity.ref).catch(report);
  const main = el("span", undefined, "item-main");
  const symbol = el("span", undefined, `entity-symbol ${entity.ref.kind}`);
  symbol.append(icon(entity.ref.kind));
  const copy = el("span");
  copy.append(
    el("span", entity.title || kindName(entity.ref.kind), "item-title"),
    el(
      "span",
      `${shortId(entity.ref.id)} · ${kindName(entity.ref.kind)}${item.plan_label ? ` · ${item.plan_label}` : ""}`,
      "item-subtitle",
    ),
  );
  main.append(symbol, copy);
  button.append(main);
  const lifecycle = status?.display_state || entity.lifecycle;
  const stateCell = el("span", undefined, "item-state");
  stateCell.append(badge(lifecycle));
  if (mode === "list") {
    const priority = el(
      "span",
      entity.priority
        ? `${entity.priority === "low" ? "↓" : entity.priority === "normal" ? "—" : "↑"} ${display(entity.priority)}`
        : "—",
      `item-priority ${entity.priority || ""}`,
    );
    button.append(stateCell, priority);
  } else {
    const footer = el("span", undefined, "board-card-footer");
    footer.append(
      stateCell,
      el(
        "span",
        entity.priority
          ? `${entity.priority === "low" ? "↓" : entity.priority === "normal" ? "—" : "↑"} ${display(entity.priority)}`
          : "",
        `item-priority ${entity.priority || ""}`,
      ),
    );
    button.append(footer);
    if (
      status?.readiness_reasons?.length &&
      [
        "blocked",
        "attention_needed",
        "waiting_dependencies",
        "scheduled",
      ].includes(status.display_state)
    )
      button.append(
        el(
          "small",
          status.readiness_reasons.map(display).join(" · "),
          "board-reason",
        ),
      );
    if (item.plan_label)
      button.append(
        el(
          "small",
          `Plano: ${item.plan_label} · revisão ${item.revision_number}`,
        ),
      );
  }
  return button;
}
function timelineView(item) {
  const row = el("div", undefined, "timeline-entry");
  const marker = el("span", undefined, "timeline-marker");
  marker.append(icon(item.actor_ref?.kind === "agent" ? "work_item" : "user"));
  const body = el("div");
  body.append(
    Object.assign(el("h3", eventName(item.event_type)), {
      title: item.event_type,
    }),
    el(
      "p",
      `${item.actor_ref?.id || item.principal_id} · ${kindName(item.actor_ref?.kind || "human")} · Revisão ${item.outcome_revision}`,
    ),
    referenceButton(item.entity_ref),
    el("time", date(item.recorded_at)),
  );
  row.append(marker, body);
  return row;
}
function eventName(type) {
  const subjects = {
    work_item: "Trabalho",
    objective: "Objetivo",
    outcome: "Resultado",
    roadmap: "Plano",
    evidence: "Evidência",
    issue: "Problema",
    blocker: "Impedimento",
    decision: "Decisão",
    artifact: "Artefato",
    criterion: "Critério",
    relation: "Relação",
  };
  const actions = {
    created: "criado",
    updated: "atualizado",
    completed: "concluído",
    claimed: "reservado",
    achieved: "alcançado",
    activated: "ativado",
    registered: "registrada",
    retracted: "retirada",
    resolved: "resolvido",
    recorded: "registrada",
    published: "publicado",
    reopened: "reaberto",
    archived: "arquivado",
    assessment_recorded: "avaliação registrada",
    draft_opened: "rascunho aberto",
    draft_replaced: "rascunho editado",
    draft_published: "revisão publicada",
  };
  const normalized = type
    .replace(/([a-z])([A-Z])/g, "$1_$2")
    .toLowerCase()
    .replace(/(^wos[._]|[._]v\d+$)/g, "")
    .replaceAll(".", "_");
  const subject = Object.keys(subjects).find((s) =>
    normalized.startsWith(s + "_"),
  );
  if (subject) {
    const action = normalized.slice(subject.length + 1);
    return `${subjects[subject]} · ${actions[action] || display(action)}`;
  }
  return display(normalized);
}
function emptySection(title, command) {
  const box = el("div", undefined, "empty-section");
  box.append(
    icon(sectionIcons[state.section] || "work_item"),
    el("h3", title),
    el("p", "Os registros aparecerão aqui conforme o trabalho avança."),
  );
  if (command) {
    const b = el("button", actionName(command), "quiet");
    b.onclick = () => createItem(command);
    box.append(b);
  }
  return box;
}
async function sectionData(section, cursor = "", append = false) {
  const cached = state.loaded.get(section);
  if (!cursor && cached) return cached;
  let items, next;
  if (section === "timeline") {
    const page = await request(
      `${outcomeBase()}/timeline?limit=25${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
    );
    items = page.items;
    next = page.next_cursor;
  } else if (section === "graph") {
    const page = await request(
      `${outcomeBase()}/graph?depth=3&limit=25${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
    );
    items = page.edges;
    next = page.next_cursor;
  } else if (section === "roadmaps") {
    const page = await request(`${outcomeBase()}/roadmaps`);
    items = (page.items || []).map((x) => ({
      ref: {
        namespace_id: state.namespace,
        outcome_id: state.outcome.id,
        kind: "roadmap",
        id: x.id,
      },
      title: x.title,
      version: x.version,
      lifecycle: x.lifecycle,
    }));
  } else if (!cursor) {
    items = state.snapshot.sections[section] || [];
    next = state.snapshot.section_cursors[section];
  } else {
    const page = await request(
      `${outcomeBase()}/continuity/${section}?limit=25&cursor=${encodeURIComponent(cursor)}`,
    );
    items = page.items;
    next = page.next_cursor;
  }
  return {
    items: append ? [...(cached?.items || []), ...(items || [])] : items || [],
    next,
  };
}
async function renderSection(cursor = "", append = false) {
  if (!state.snapshot || state.view === "summary") return;
  if (state.view === "board") return renderBoard();
  const section = state.section,
    generation = state.generation,
    run = ++state.renderId;
  const data = await sectionData(section, cursor, append);
  if (generation !== state.generation || run !== state.renderId) return;
  state.loaded.set(section, data);
  const content = $("content");
  content.replaceChildren();
  const heading = el("div", undefined, "content-heading");
  const shown = data.items.filter(matches);
  heading.append(
    el("h2", human(section)),
    el(
      "small",
      `${shown.length} itens exibidos${state.snapshot.counts[section] !== undefined ? ` · ${state.snapshot.counts[section]} no total` : ""}`,
    ),
  );
  content.append(heading);
  for (const b of $("tabs").children) {
    b.classList.toggle("selected", b.dataset.section === section);
    b.setAttribute("aria-pressed", String(b.dataset.section === section));
  }
  if ($("item-search").value || $("item-priority").value)
    content.append(
      el(
        "p",
        "Filtro aplicado aos itens carregados nesta visualização.",
        "section-note",
      ),
    );
  if (!shown.length)
    content.append(
      emptySection(
        data.items.length
          ? "Nenhum item corresponde aos filtros"
          : "Ainda não há registros nesta seção.",
        data.items.length ? "" : sectionCommand(),
      ),
    );
  const list = el("div", undefined, "collection");
  let rows = 0;
  for (const item of shown) {
    const entity = item.work_item || item.current || item;
    if (entity.ref) {
      if (!rows) {
        const head = el("div", undefined, "list-head");
        head.append(
          el("span", "Item"),
          el("span", "Estado"),
          el("span", "Prioridade"),
        );
        list.append(head);
        content.append(list);
      }
      list.append(itemView(item, "list"));
      rows++;
    } else if (section === "graph") {
      const ref = item.from || item.source_ref,
        other = item.to || item.target_ref;
      const row = el("div", undefined, "relation-card");
      if (ref && other)
        row.append(
          referenceButton(ref),
          el("span", "→", "relation-arrow"),
          referenceButton(other),
          badge(item.kind || item.type || item.relation_type || "depends_on"),
        );
      else row.append(readable(item));
      content.append(row);
    } else content.append(itemView(item));
  }
  if (data.next) {
    const more = el("button", "Carregar mais", "quiet more-button");
    more.dataset.more = "true";
    more.onclick = () => renderSection(data.next, true).catch(report);
    content.append(more);
  }
}
const boardGroups = [
  {
    title: "Planejado",
    tone: "violet",
    sections: [
      "backlog_work",
      "waiting_scope_work",
      "waiting_dependencies_work",
      "scheduled_work",
    ],
  },
  { title: "Pronto para execução", tone: "", sections: ["ready_work"] },
  { title: "Em execução", tone: "orange", sections: ["in_progress_work"] },
  {
    title: "Impedimentos",
    tone: "red",
    sections: ["blocked_work", "attention_needed_work"],
  },
  {
    title: "Concluído",
    tone: "green",
    sections: ["done_work"],
  },
];
function renderBoard() {
  const content = $("content");
  content.replaceChildren(
    el(
      "p",
      "O quadro reflete disponibilidade e reservas reais. Abra um item para executar uma ação.",
      "board-caption",
    ),
  );
  const board = el("div", undefined, "board");
  const groups = [...boardGroups];
  if (state.snapshot.counts.cancelled_work)
    groups.push({
      title: "Cancelado",
      tone: "violet",
      sections: ["cancelled_work"],
    });
  board.classList.toggle("board-six", groups.length === 6);
  for (const group of groups) {
    const column = el("section", undefined, "board-column");
    column.setAttribute("aria-label", group.title);
    const heading = el("div", undefined, "column-heading");
    const count = group.sections.reduce(
      (n, s) => n + (state.snapshot.counts[s] || 0),
      0,
    );
    heading.append(
      el("span", undefined, `legend-dot ${group.tone}`),
      el("strong", group.title),
      el("span", String(count), "count"),
    );
    column.append(heading);
    const items = group.sections
      .flatMap(
        (s) =>
          state.boardLoaded.get(s)?.items || state.snapshot.sections[s] || [],
      )
      .filter(matches);
    for (const item of items) column.append(itemView(item));
    if (!items.length)
      column.append(
        el(
          "p",
          count
            ? "Nenhum item carregado corresponde aos filtros"
            : "Nenhum item",
          "board-empty",
        ),
      );
    for (const section of group.sections) {
      const cursor =
        state.boardLoaded.get(section)?.next ??
        (state.boardLoaded.has(section)
          ? ""
          : state.snapshot.section_cursors[section]);
      if (cursor) {
        const more = el(
          "button",
          `Carregar mais · ${display(section.replace("_work", ""))}`,
          "quiet more-button",
        );
        more.onclick = async () => {
          const generation = state.generation;
          more.disabled = true;
          try {
            const page = await request(
              `${outcomeBase()}/continuity/${section}?limit=25&cursor=${encodeURIComponent(cursor)}`,
            );
            if (generation !== state.generation) return;
            const previous =
              state.boardLoaded.get(section)?.items ||
              state.snapshot.sections[section] ||
              [];
            state.boardLoaded.set(section, {
              items: [...previous, ...page.items],
              next: page.next_cursor || "",
            });
            if (state.view === "board") renderBoard();
          } catch (e) {
            report(e);
            more.disabled = false;
          }
        };
        column.append(more);
      }
    }
    board.append(column);
  }
  content.append(board);
  if (
    state.snapshot.truncated ||
    $("item-search").value ||
    $("item-priority").value
  )
    content.append(
      el(
        "p",
        "Contagens representam o estado completo. Cartões e filtros incluem os itens já carregados.",
        "section-note",
      ),
    );
}
const plurals = {
  objective: "objectives",
  work_item: "work-items",
  issue: "issues",
  blocker: "blockers",
  decision: "decisions",
  evidence: "evidence",
  artifact: "artifacts",
  roadmap: "roadmaps",
};
function readable(value, key = "", depth = 0) {
  if (value === null || value === undefined) return el("span", "—");
  if (typeof value !== "object")
    return el(
      "span",
      key.endsWith("_at") || key.startsWith("planned_")
        ? date(value)
        : typeof value === "boolean"
          ? value
            ? "Sim"
            : "Não"
          : display(value),
    );
  if (depth > 4)
    return el("span", "Consulte os dados técnicos para mais detalhes.");
  if (value.kind && value.id && value.namespace_id)
    return referenceButton(value);
  if (Array.isArray(value)) {
    const list = el("div");
    if (!value.length) list.append(el("span", "Nenhum registro"));
    for (const v of value) {
      const row = el("div", undefined, "readable-row");
      row.append(readable(v, key, depth + 1));
      list.append(row);
    }
    return list;
  }
  const list = el("dl", undefined, "detail-properties");
  for (const [k, v] of Object.entries(value)) {
    if (
      v === null ||
      v === undefined ||
      [
        "namespace_id",
        "outcome_id",
        "id",
        "scope",
        "content_hash",
        "command_id",
        "event_id",
        "checksum",
      ].includes(k)
    )
      continue;
    list.append(el("dt", human(k)));
    const dd = el("dd");
    dd.append(readable(v, k, depth + 1));
    list.append(dd);
  }
  return list;
}
function conclusionView(conclusion) {
  const card = el("div", undefined, "conclusion-card");
  const status = conclusion.lifecycle_result || "achieved";
  card.append(
    badge(status),
    el("p", conclusion.reason),
    el(
      "small",
      `${conclusion.actor_ref?.id || conclusion.principal_id} · ${date(conclusion.concluded_at)}`,
    ),
  );
  if (conclusion.assessments?.length)
    card.append(
      el(
        "p",
        `${conclusion.assessments.length} avaliações usadas na conclusão.`,
      ),
    );
  if (conclusion.obligations?.required_objective_ids?.length)
    card.append(
      el(
        "p",
        `${conclusion.obligations.required_objective_ids.length} objetivos obrigatórios verificados.`,
      ),
    );
  return card;
}
function planNodes(nodes) {
  const box = el("div");
  for (const [i, n] of [...(nodes || [])]
    .sort((a, b) => a.position - b.position)
    .entries()) {
    const row = el("div", undefined, "plan-node");
    const body = el("div");
    body.append(
      el("strong", n.title),
      el(
        "small",
        `${display(n.node_type)}${n.parent_node_key ? ` · Grupo ${n.parent_node_key}` : ""}`,
      ),
    );
    if (n.target_ref) body.append(referenceButton(n.target_ref));
    if (n.planned_start || n.planned_end)
      body.append(
        el("small", `${date(n.planned_start)} → ${date(n.planned_end)}`),
      );
    row.append(
      el("span", String(i + 1).padStart(2, "0"), "plan-node-number"),
      body,
    );
    box.append(row);
  }
  if (!nodes?.length) box.append(el("p", "Este plano ainda não tem itens."));
  return box;
}
function quickActions(kind, entity) {
  const options = {
    work_item: [
      "claim_work_item",
      "complete_work_item",
      "release_work_item",
      "cancel_work_item",
    ],
    objective: ["start_objective", "achieve_objective"],
    outcome: ["activate_outcome", "achieve_outcome"],
    issue: ["investigate_issue", "resolve_issue"],
    blocker: ["resolve_blocker"],
    roadmap: ["open_roadmap_draft", "publish_roadmap_draft"],
    evidence: ["retract_evidence"],
  };
  const container = el("div", undefined, "quick-actions");
  for (const name of options[kind] || []) {
    if (!state.catalog.some((c) => c.name === name)) continue;
    if (
      kind === "work_item" &&
      (["done", "cancelled"].includes(entity.lifecycle) ||
        (name === "claim_work_item" && entity.lifecycle !== "todo") ||
        (name === "complete_work_item" && entity.lifecycle !== "in_progress") ||
        (name === "release_work_item" && entity.lifecycle !== "in_progress"))
    )
      continue;
    const button = el("button", actionName(name), "quiet");
    button.onclick = () => openCommands(name);
    container.append(button);
  }
  return container;
}
async function showEntity(ref) {
  const generation = state.generation;
  if (
    ref.namespace_id !== state.namespace ||
    ref.outcome_id !== state.outcome.id
  ) {
    notice("Abra o resultado correspondente para consultar este item.", true);
    return;
  }
  const path =
    ref.kind === "outcome"
      ? outcomeBase()
      : `${outcomeBase()}/${plurals[ref.kind]}/${ref.id}`;
  const result = await request(path);
  if (generation !== state.generation) return;
  const entity = result.value || result;
  state.selected = { ...entity, _kind: ref.kind };
  state.criterion = null;
  state.entities.set(`${ref.kind}/${ref.id}`, {
    ref,
    title: entity.title || entity.name || entity.description,
    lifecycle: entity.lifecycle,
  });
  $("detail-kind").textContent = `${kindName(ref.kind)} / ${shortId(ref.id)}`;
  $("detail-title").textContent =
    entity.title || entity.name || entity.description || human(ref.kind);
  const detail = $("detail-content");
  detail.replaceChildren();
  const status = el("div", undefined, "detail-status");
  if (entity.lifecycle) status.append(badge(entity.lifecycle));
  if (entity.priority)
    status.append(
      el("span", `Prioridade ${display(entity.priority)}`, "badge"),
    );
  if (entity.archived_at) status.append(el("span", "Arquivado", "badge"));
  detail.append(status);
  const description =
    entity.desired_state ||
    entity.description ||
    entity.proposal ||
    entity.result_summary;
  if (description) detail.append(el("p", description, "detail-description"));
  const props = el("dl", undefined, "detail-properties");
  for (const key of [
    "created_at",
    "updated_at",
    "captured_at",
    "required_for_outcome",
    "not_before",
    "severity",
    "blocked_ref",
    "cause_ref",
    "resolution_summary",
    "parent_objective_id",
    "verification_mode",
    "uri",
    "source_ref",
    "objective_id",
    "producer_ref",
    "assignee_refs",
  ]) {
    if (entity[key] === undefined || entity[key] === null || entity[key] === "")
      continue;
    const dd = el("dd");
    if (["objective_id", "parent_objective_id"].includes(key))
      dd.append(
        referenceButton({ ...ref, kind: "objective", id: entity[key] }),
      );
    else if (key === "uri") {
      try {
        const u = new URL(entity[key]);
        if (["https:", "http:"].includes(u.protocol)) {
          const a = el("a", entity[key]);
          a.href = u.href;
          a.target = "_blank";
          a.rel = "noopener noreferrer";
          dd.append(a);
        } else dd.append(el("span", entity[key]));
      } catch {
        dd.append(el("span", entity[key]));
      }
    } else dd.append(readable(entity[key], key));
    props.append(el("dt", human(key)), dd);
  }
  detail.append(props);
  if (entity.current_lease) {
    const lease = el("section", undefined, "detail-section");
    lease.append(
      el("h3", "Reserva de execução"),
      el(
        "p",
        `${entity.current_lease.actor_ref?.id || entity.current_lease.principal_id} · expira em ${date(entity.current_lease.expires_at)}`,
      ),
    );
    detail.append(lease);
  }
  const criteria = entity.criteria?.items || [];
  if (criteria.length) {
    const group = el("section", undefined, "detail-section");
    group.append(el("h3", "Critérios de sucesso"));
    for (const criterion of criteria) {
      const button = el("button", undefined, "item criterion-item");
      button.append(
        el("strong", criterion.title),
        el(
          "small",
          `${criterion.required ? "Obrigatório" : "Opcional"} · ${display(criterion.verification_mode)} · revisão ${criterion.criterion_revision}`,
        ),
      );
      const assessment = entity.criteria.current_assessments?.[criterion.id];
      if (assessment) button.append(badge(assessment.result));
      button.append(el("small", "Revisar avaliação →"));
      button.onclick = () => {
        state.criterion = criterion;
        openCommands("record_criterion_assessment");
      };
      group.append(button);
    }
    detail.append(group);
  }
  if (entity.conclusion) {
    const group = el("section", undefined, "detail-section");
    group.append(
      el("h3", "Conclusão atual"),
      conclusionView(entity.conclusion),
    );
    detail.append(group);
  }
  if (entity.conclusion_history?.length) {
    const history = el("details");
    history.append(el("summary", "Conclusões anteriores"));
    for (const c of entity.conclusion_history)
      history.append(conclusionView(c));
    detail.append(history);
  }
  if (ref.kind === "roadmap") {
    if (entity.draft) {
      const draft = el("details");
      draft.open = true;
      draft.append(
        el("summary", `Rascunho · versão ${entity.draft.draft_version}`),
        planNodes(entity.draft.nodes),
      );
      detail.append(draft);
    }
    for (const revision of entity.revisions || []) {
      const history = el("details");
      history.open = revision === entity.revisions.at(-1);
      history.append(
        el(
          "summary",
          `Revisão ${revision.revision_number} · ${revision.reason || "Plano publicado"}`,
        ),
        planNodes(revision.nodes),
      );
      detail.append(history);
    }
  }
  const tech = el("details", undefined, "technical");
  tech.append(
    el("summary", "Dados técnicos e auditoria"),
    el("pre", JSON.stringify(entity, null, 2)),
  );
  detail.append(tech);
  $("detail-actions").querySelector(".quick-actions")?.remove();
  $("detail-actions").prepend(quickActions(ref.kind, entity));
  if (!$("detail-dialog").open) $("detail-dialog").showModal();
}
$("close-detail").onclick = () => $("detail-dialog").close();
$("entity-actions").onclick = () => openCommands();
$("actions").onclick = () => {
  state.selected = { ...state.outcome, _kind: "outcome" };
  state.criterion = null;
  openCommands();
};
$("create-outcome").onclick = () => {
  state.selected = null;
  openCommands("create_outcome");
};
function defaults() {
  const e = state.selected || {},
    kind = e._kind || "outcome";
  const scope = {
    namespace_id: state.namespace,
    outcome_id: state.outcome?.id,
  };
  const ref = { ...scope, kind, id: e.id || state.outcome?.id };
  return {
    title: e.title,
    description: e.description,
    desired_state: e.desired_state,
    nodes: e.draft?.nodes,
    after_links: e.draft?.after_links,
    ...scope,
    scope,
    owner: ref,
    target_ref: ref,
    source_ref: ref,
    expected_version: e.version || state.outcome?.version,
    expected_roadmap_version: e.version,
    expected_draft_version: e.draft?.draft_version,
    plan_scope: { kind: "outcome", id: state.outcome?.id },
    scope_kind: "outcome",
    scope_id: state.outcome?.id,
    priority: "normal",
    lifecycle: "todo",
    ttl_seconds: 900,
    producer_ref: { kind: "human", provider: "wos-web", id: "browser" },
    ...Object.fromEntries(
      [
        ["work_item", "work_item_id"],
        ["objective", "objective_id"],
        ["roadmap", "roadmap_id"],
        ["evidence", "evidence_id"],
        ["artifact", "artifact_id"],
        ["decision", "decision_id"],
        ["issue", "issue_id"],
        ["blocker", "blocker_id"],
      ].map(([k, f]) => [f, kind === k ? e.id : undefined]),
    ),
    claim_id: e.current_lease?.claim_id,
    fencing_token: e.current_lease?.fencing_token,
    criterion_id: state.criterion?.id,
    criterion_revision: state.criterion?.criterion_revision,
    verification_mode: "attestation",
    result: "met",
  };
}
function openCommands(name) {
  $("submit-command").disabled = false;
  $("detail-dialog").close();
  state.pending = null;
  $("retry-command").hidden = true;
  $("command-error").textContent = "";
  $("command-select").replaceChildren();
  const relevant = el("optgroup");
  relevant.label = "Ações deste item";
  const other = el("optgroup");
  other.label = "Outras ações";
  const kind = state.selected?._kind || "outcome";
  for (const d of state.catalog) {
    const option = el("option", actionName(d.name));
    option.value = d.name;
    (d.name.endsWith(`_${kind}`) ||
    (state.criterion && d.name.includes("criterion"))
      ? relevant
      : other
    ).append(option);
  }
  $("command-select").append(relevant, other);
  if (name) $("command-select").value = name;
  else {
    const kind = state.selected?._kind || "outcome";
    const preferred = `update_${kind}`;
    if (state.catalog.some((x) => x.name === preferred))
      $("command-select").value = preferred;
  }
  buildForm();
  $("command-dialog").showModal();
}
function field(name, schema, value, required = false) {
  const wrap = el("div");
  let getter;
  if (
    value !== undefined &&
    value !== null &&
    ([
      "namespace_id",
      "outcome_id",
      "expected_version",
      "expected_roadmap_version",
      "expected_draft_version",
      "claim_id",
      "fencing_token",
    ].includes(name) ||
      (["criterion_id", "criterion_revision"].includes(name) &&
        state.criterion))
  ) {
    return { node: wrap, get: () => value };
  }

  const entityKind = {
    objective_id: "objective",
    parent_objective_id: "objective",
    work_item_id: "work_item",
    evidence_id: "evidence",
    artifact_id: "artifact",
    issue_id: "issue",
    blocker_id: "blocker",
    decision_id: "decision",
    roadmap_id: "roadmap",
  }[name];
  if (
    entityKind &&
    value === state.selected?.id &&
    entityKind === state.selected?._kind &&
    name !== "parent_objective_id"
  )
    return { node: wrap, get: () => value };
  if (entityKind) {
    const label = el("label", human(name)),
      select = el("select");
    select.append(Object.assign(el("option", "Escolher item"), { value: "" }));
    const items = new Map(state.entities);
    for (const section of Object.values(state.snapshot?.sections || {})) {
      for (const entry of section) {
        const item = entry?.ref ? entry : entry?.work_item;
        if (item?.ref) items.set(`${item.ref.kind}/${item.ref.id}`, item);
      }
    }
    for (const item of items.values()) {
      if (item.ref.kind !== entityKind) continue;
      const option = el("option", item.title || human(entityKind));
      option.value = item.ref.id;
      select.append(option);
    }
    if (value && !Array.from(select.options).some((o) => o.value === value)) {
      const option = el("option", state.selected?.title || human(entityKind));
      option.value = value;
      select.append(option);
    }
    select.value = value || "";
    select.required = required;
    label.append(select);
    wrap.append(label);
    return { node: wrap, get: () => select.value || undefined };
  }

  if (
    schema.properties?.namespace_id &&
    schema.properties?.outcome_id &&
    schema.properties?.kind &&
    schema.properties?.id
  ) {
    const label = el("label", human(name));
    const select = el("select");
    const blank = el("option", "Escolher item");
    blank.value = "";
    select.append(blank);
    const items = new Map(state.entities);
    if (state.snapshot) {
      const root = state.snapshot.outcome;
      items.set(`${root.ref.kind}/${root.ref.id}`, root);
      for (const values of Object.values(state.snapshot.sections)) {
        for (const item of values) {
          const entity = item?.ref ? item : item?.work_item;
          if (entity?.ref)
            items.set(`${entity.ref.kind}/${entity.ref.id}`, entity);
        }
      }
    }
    for (const [key, item] of items) {
      const option = el(
        "option",
        `${item.title || kindName(item.ref.kind)} · ${kindName(item.ref.kind)}`,
      );
      option.value = key;
      select.append(option);
    }
    if (value?.id) select.value = `${value.kind}/${value.id}`;
    select.required = required;
    label.append(select);
    wrap.append(label);
    getter = () => (select.value ? items.get(select.value).ref : undefined);
    return { node: wrap, get: getter };
  }
  if (name === "scope" && value?.namespace_id && value?.outcome_id) {
    return { node: wrap, get: () => value };
  }

  if (
    schema.type === "object" &&
    schema.properties &&
    !required &&
    value === undefined
  ) {
    const label = el("label");
    const enabled = el("input");
    enabled.type = "checkbox";
    label.append(enabled, document.createTextNode(` Incluir ${human(name)}`));
    const child = field(name, schema, undefined, true);
    const group = el("fieldset");
    group.append(child.node);
    group.hidden = true;
    group.disabled = true;
    enabled.onchange = () => {
      group.hidden = !enabled.checked;
      group.disabled = !enabled.checked;
    };
    wrap.append(label, group);
    return {
      node: wrap,
      get: () => (enabled.checked ? child.get() : undefined),
    };
  }

  if (!schema.type) {
    const label = el("label", `${human(name)} (JSON)`);
    const input = el("textarea");
    input.value = value === undefined ? "" : JSON.stringify(value);
    input.placeholder = 'Exemplo: "valor", true, ["item"] ou {"chave":"valor"}';
    label.append(input);
    wrap.append(label);
    getter = () => (input.value.trim() ? JSON.parse(input.value) : undefined);
  } else if (schema.type === "object" && !schema.properties) {
    const label = el("label", `${human(name)} (pares de contexto em JSON)`);
    const input = el("textarea");
    input.value = value === undefined ? "{}" : JSON.stringify(value, null, 2);
    input.placeholder = '{"product":"exemplo", "user_id":"pessoa-42"}';
    label.append(input);
    wrap.append(label);
    getter = () => (input.value.trim() ? JSON.parse(input.value) : undefined);
  } else if (schema.type === "object") {
    const group = el("fieldset");
    group.append(el("legend", human(name)));
    const extra = el("details", undefined, "command-extra");
    extra.append(el("summary", "Informações adicionais e proveniência"));
    const secondary = el("div");
    extra.append(secondary);
    const fields = {};
    for (const [key, child] of Object.entries(schema.properties || {})) {
      if (
        [
          "reference_snapshot",
          "published_reference_snapshot",
          "criterion_snapshots",
        ].includes(key)
      )
        continue;
      const rendered = field(
        key,
        child,
        value?.[key],
        schema.required?.includes(key),
      );
      const optionalExtra =
        [
          "metadata",
          "external_context",
          "execution_context",
          "source_version",
          "checksum",
          "producer_ref",
          "assignee_refs",
          "actor_ref",
          "not_before",
          "criterion_refs",
        ].includes(key) && !schema.required?.includes(key);
      (optionalExtra ? secondary : group).append(rendered.node);
      fields[key] = rendered.get;
    }
    if (secondary.children.length) group.append(extra);
    wrap.append(group);
    getter = () => {
      const result = {};
      for (const [key, get] of Object.entries(fields)) {
        const v = get();
        if (v !== undefined) result[key] = v;
      }
      return Object.keys(result).length ? result : undefined;
    };
  } else if (schema.type === "array") {
    const group = el("fieldset");
    group.append(el("legend", human(name)));
    const rows = el("div");
    const getters = [];
    const add = el("button", "Adicionar item", "quiet");
    add.type = "button";
    function append(v) {
      const row = el("div");
      const child = field(
        name === "evidence_ids" ? "evidence_id" : "item",
        schema.items || {},
        v,
        true,
      );
      const remove = el("button", "Remover", "quiet");
      remove.type = "button";
      const item = { get: child.get, active: true };
      getters.push(item);
      remove.onclick = () => {
        item.active = false;
        row.remove();
      };
      row.append(child.node, remove);
      rows.append(row);
    }
    for (const item of value || []) append(item);
    add.onclick = () => append();
    group.append(rows, add);
    wrap.append(group);
    getter = () =>
      getters
        .filter((x) => x.active)
        .map((x) => x.get())
        .filter((x) => x !== undefined);
  } else {
    const label = el("label", human(name));
    let input;
    const options = schema.enum || choices[name];
    if (options) {
      input = el("select");
      if (!required) {
        const blank = el("option", "Não informado");
        blank.value = "";
        input.append(blank);
      }
      for (const choice of options) {
        const option = el("option", display(choice));
        option.value = choice;
        input.append(option);
      }
    } else {
      input = el(
        [
          "description",
          "reason",
          "rationale",
          "result_summary",
          "desired_state",
          "proposal",
        ].includes(name)
          ? "textarea"
          : "input",
      );
      if (input.tagName === "INPUT")
        input.type =
          schema.type === "boolean"
            ? "checkbox"
            : schema.type === "integer" || schema.type === "number"
              ? "number"
              : schema.format === "date-time"
                ? "datetime-local"
                : "text";
      if (schema.pattern) input.pattern = schema.pattern;
      if (schema.type === "integer") {
        input.step = "1";
        input.min = "0";
      }
    }
    if (
      name.startsWith("expected_") ||
      ["namespace_id", "outcome_id"].includes(name)
    )
      input.readOnly = true;
    if (name === "position" && value === undefined) value = 0;
    if (name === "node_type" && value === undefined) value = "reference";
    if (input.type === "checkbox") input.checked = Boolean(value);
    else if (value !== undefined && value !== null)
      input.value =
        schema.format === "date-time"
          ? new Date(
              new Date(value).getTime() -
                new Date(value).getTimezoneOffset() * 60000,
            )
              .toISOString()
              .slice(0, 16)
          : String(value);
    if (required && input.type !== "checkbox") input.required = true;
    label.append(input);
    wrap.append(label);
    getter = () => {
      if (input.type === "checkbox") return input.checked;
      if (input.value === "") return undefined;
      if (schema.type === "integer" || schema.type === "number")
        return Number(input.value);
      if (schema.format === "date-time")
        return new Date(input.value).toISOString();
      return input.value;
    };
  }
  return { node: wrap, get: getter };
}
let formGetter;
function buildForm() {
  const descriptor = state.catalog.find(
    (x) => x.name === $("command-select").value,
  );
  if (!descriptor) return;
  $("command-title").textContent = actionName(descriptor.name);
  $("command-help").textContent =
    descriptor.name === "record_criterion_assessment"
      ? "Revise a prova e registre sua avaliação. Uma evidência, sozinha, não comprova o critério."
      : descriptor.name === "claim_work_item"
        ? "Reserve o trabalho antes de executar. A reserva tem prazo e evita duas execuções concorrentes."
        : "A alteração será registrada para todos os participantes. Revise antes de confirmar.";
  const context = el(
    "div",
    state.criterion
      ? `Critério: ${state.criterion.title} · revisão ${state.criterion.criterion_revision}`
      : state.selected
        ? `${kindName(state.selected._kind || "outcome")}: ${state.selected.title || state.selected.name || state.selected.description || state.outcome?.title || ""}${state.selected.version ? ` · versão ${state.selected.version}` : ""}`
        : "Novo resultado neste contexto",
    "field-context",
  );
  const form = field("Informações", descriptor.schema, defaults(), true);
  $("fields").replaceChildren(context, form.node);
  formGetter = form.get;
  $("command-error").textContent = "";
  state.pending = null;
  $("retry-command").hidden = true;
}

$("command-select").onchange = buildForm;
$("close-command").onclick = () => $("command-dialog").close();
async function submit(retry = false) {
  if (!retry) {
    state.pending = {
      name: $("command-select").value,
      key: crypto.randomUUID(),
      command: formGetter(),
    };
  }
  if (!state.pending) return;
  const pending = state.pending;
  $("submit-command").disabled = true;
  $("retry-command").hidden = true;
  try {
    const result = await request(`${api}/commands/${pending.name}`, {
      method: "POST",
      headers: { "Idempotency-Key": pending.key },
      body: JSON.stringify({ command: pending.command }),
    });
    state.pending = null;
    if (result.value?.desired_state !== undefined) {
      state.outcome = result.value;
      state.selected = result.value;
      if (pending.name === "create_outcome") {
        state.generation++;
        state.view = "summary";
        $("item-search").value = "";
        $("item-priority").value = "";
      }
    }
    const newSection = Object.entries(createForSection).find(
      ([section, command]) => command === pending.name,
    )?.[0];
    if (newSection) {
      state.section = newSection;
      state.view = "list";
      $("item-search").value = "";
      $("item-priority").value = "";
    }
    await discover();
    if (state.outcome) await refresh();
    $("command-dialog").close();
    notice(
      result.result_omitted
        ? "Alteração registrada. Consulte o estado atualizado."
        : "Alteração registrada e persistida.",
    );
  } catch (e) {
    $("command-error").className = "danger";
    $("command-error").textContent =
      e.code === "version_conflict"
        ? "A versão mudou. Feche este formulário, atualize e revise sua intenção."
        : e.message;
    if (e.status === undefined || e.status >= 500) {
      $("retry-command").hidden = false;
      $("command-error").textContent +=
        " A resposta pode ter sido perdida após o registro. Repita a mesma tentativa para recuperar o resultado.";
    }
  } finally {
    $("submit-command").disabled = !$("retry-command").hidden;
  }
}
$("command-form").onsubmit = (event) => {
  event.preventDefault();
  submit().catch(report);
};
$("retry-command").onclick = () => submit(true).catch(report);
namespaces().catch(() => {});

let administration, adminGetter, adminPending;
$("administration").onclick = async () => {
  try {
    administration = await request(
      `${api}/namespaces/${state.namespace}/administration`,
    );
    $("admin-version").textContent =
      `Versão administrativa ${administration.namespace_version}`;
    $("admin-snapshot").replaceChildren(readable(administration));
    $("admin-error").textContent = "";
    $("admin-secret").hidden = true;
    buildAdmin();
    $("admin-dialog").showModal();
  } catch (e) {
    report(e);
  }
};
function buildAdmin() {
  adminPending = null;
  $("retry-admin").hidden = true;
  $("submit-admin").disabled = false;
  const op = $("admin-operation").value;
  const props = {};
  if (op === "create_namespace") props.namespace_name = { type: "string" };
  if (op === "set_grant") {
    props.principal_id = { type: "string" };
    props.permissions = { type: "array", items: { type: "string" } };
  }
  if (op === "issue_credential") {
    props.principal_id = { type: "string" };
    props.actor_ref = {
      type: "object",
      required: ["kind", "provider", "id"],
      properties: {
        kind: { type: "string" },
        provider: { type: "string" },
        id: { type: "string" },
      },
    };
    props.expires_at = { type: "string", format: "date-time" };
  }
  if (op === "revoke_credential") props.credential_id = { type: "string" };
  const rendered = field(
    "Administração",
    { type: "object", properties: props, required: Object.keys(props) },
    {},
  );
  $("admin-fields").replaceChildren(rendered.node);
  adminGetter = rendered.get;
}
$("admin-operation").onchange = buildAdmin;
$("close-admin").onclick = () => {
  $("admin-dialog").close();
  $("admin-token").value = "";
};
async function submitAdmin(retry = false) {
  if (!retry)
    adminPending = {
      key: crypto.randomUUID(),
      intent: {
        namespace_id: state.namespace,
        expected_namespace_version: administration.namespace_version,
        operation: $("admin-operation").value,
        ...adminGetter(),
      },
    };
  if (!adminPending) return;
  $("submit-admin").disabled = true;
  $("retry-admin").hidden = true;
  try {
    const response = await request(`${api}/security/commands`, {
      method: "POST",
      headers: { "Idempotency-Key": adminPending.key },
      body: JSON.stringify(adminPending.intent),
    });
    adminPending = null;
    administration = await request(
      `${api}/namespaces/${state.namespace}/administration`,
    );
    $("admin-version").textContent =
      `Versão administrativa ${administration.namespace_version}`;
    $("admin-snapshot").replaceChildren(readable(administration));
    $("admin-error").textContent = response.result?.token_omitted
      ? "A emissão anterior foi confirmada. A credencial não pode ser recuperada; revogue-a e emita outra."
      : "Alteração registrada.";
    if (response.token) {
      $("admin-token").value = response.token;
      $("admin-secret").hidden = false;
    }
  } catch (e) {
    $("admin-error").textContent = e.message;
    if (e.status === undefined || e.status >= 500) {
      $("retry-admin").hidden = false;
      $("admin-error").textContent +=
        " O registro pode ter ocorrido. Repita a mesma tentativa para consultar o recibo.";
    } else
      $("admin-error").textContent +=
        " Feche e reabra para revisar o estado antes de uma nova intenção.";
  } finally {
    $("submit-admin").disabled = !$("retry-admin").hidden;
  }
}
$("admin-form").onsubmit = (event) => {
  event.preventDefault();
  submitAdmin().catch(report);
};
$("retry-admin").onclick = () => submitAdmin(true).catch(report);

$("view-outcome").onclick = () =>
  showEntity(state.snapshot.outcome.ref).catch(report);
