# WOS — Auditoria e plano de correção da experiência humana, DX e idioma

**Data:** 09/10/2026  
**Base inspecionada:** `A1b3rt0M3rcad0/wos`, branch `master`, commit `5ea787c947e100a57e9b283bc1131245600888d3`  
**Natureza:** análise do código, contratos, documentação e testes versionados. **Nenhum código foi modificado.**  
**Objetivo:** converter o WOS Web de uma interface de execução de comandos técnicos em um produto de coordenação intuitivo para pessoas, com **inglês como idioma oficial de toda a experiência de produto**, sem comprometer o domínio e os protocolos existentes.

## 1. Veredito executivo

O problema principal não é visual. É a **transposição quase literal da API de comandos para a interface humana**. O backend possui um modelo de domínio mais sólido e rigoroso que a experiência de uso expõe. Hoje, o usuário precisa entender comandos, estados, conceitos de auditoria e campos técnicos antes de conseguir criar e coordenar trabalho.

A correção demanda quatro intervenções coordenadas:

1. **Reprojetar a arquitetura de informação:** substituir o catálogo de entidades e projeções por jornadas orientadas a intenção, mantendo os conceitos distintos do domínio.
2. **Substituir o editor genérico por ações e formulários específicos:** `Create outcome`, `Add objective`, `Create task`, `Define success criteria`, `Create roadmap`, `Report issue`, `Review evidence` e operações posteriores.
3. **Eliminar seletores incompletos e excessivamente amplos:** busca autorizada no servidor, paginação, agrupamento por tipo, contextualização e identificação inequívoca.
4. **Migrar o produto para inglês consistentemente:** interface, estados, validações, ajuda, datas, onboarding, testes e documentação voltada ao usuário. Manter identificadores técnicos existentes como contratos de API.

**Ordem recomendada:** intenção/fluxo humano → mapeamento domínio-operações → arquitetura dos componentes → implementação e testes. Uma reescrita puramente estética ou a tradução literal das listas existentes não resolverá o problema.

## 2. O que foi efetivamente inspecionado

- `packages/wos-api/web/assets/index.html` — shell, navegação, diálogos, login e controles.
- `packages/wos-api/web/assets/app.js` — estado, descoberta, 16 seções, cards, quadro, criação, seleção de referência, formulários e submissão.
- `packages/wos-api/web/assets/presentation.js` — estados, nomes, badges, datas, ícones e traduções.
- `packages/wos-api/web/assets/style.css` — responsividade, modais, formulários e tokens atuais.
- `packages/wos-api/web/web.go` — distribuição da interface embutida e Content Security Policy.
- `packages/wos-api/commands/catalog_generated.go` e `contracts.go` — comandos públicos, schemas e tipagem.
- `packages/wos-core/application/commands.go`, `packages/wos-core/domain/{work_contract,work_item,roadmap}.go` — campos e semântica.
- `packages/wos-api/http/{outcomes,objectives,work_items}.go` — operações REST disponíveis.
- `docs/workspace-ux.md`, `docs/ui-workspace/README.md` — intenção/limitações documentadas.
- ADRs 001, 002, 003, 004, 013, 014, 015, 018, 024 e 026 — invariantes do domínio e contratos assinados.
- `tests/web/{journey,workspace,contracts,signed}.spec.mjs` e logs versionados de 07/10 — cobertura declarada.

**Limitação metodológica:** inspeção estática do repositório e de logs registrados, não uma sessão navegada na aplicação executada em 09/10 com usuários reais. Os problemas assinalados como “confirmados” são observáveis no código. Gravidade e efeito sobre o usuário são avaliação de UX; métricas atuais de conclusão/erros não foram coletadas. Os três testes Playwright de 07/10 passaram **naquele registro**, mas não atestam usabilidade nem equivalem a reexecução desta auditoria.

## 3. Matriz de constatações e correções

| ID | Prioridade | Constatação verificável | Consequência | Correção proposta |
|---|---|---|---|---|
| UX-01 | **P0** | `openCommands()` povoa um único `<select>` a partir de `state.catalog`, com `Ações deste item` e `Outras ações` (`app.js` 1700–1737). | Usuário precisa conhecer a API. | Excluir o seletor do fluxo comum; separar ações contextuais por entidade/intenção; manter console técnico restrito, se necessário. |
| UX-02 | **P0** | 100 comandos registrados em `catalog_generated.go`; 12 têm `_signed_` e são filtrados pelo browser; até **88** operações não assinadas ficam candidatas ao seletor no protocolo legado. | Sobrecarga de opções e ações irrelevantes. | Catálogo para capacidade técnica, registry de **use cases humanos** para apresentação; autorização/estado determinam ações efetivas. |
| UX-03 | **P0** | `field()` monta campos recursivamente a partir do schema de comandos (`app.js` 1738–2051). | Formulários refletem estrutura da API, não a tarefa humana. | Formulários dedicados, defaults corretos, campos essenciais visíveis, avançados sob demanda; mapeamento explícito para DTO/command. |
| UX-04 | **P0** | `field()` aceita entrada `JSON` crua quando não conhece o tipo ou em objetos abertos (`app.js` 1854–1897). | Exige familiaridade com serialização, gera erros e confusão. | Editores especializados para referências, contexto, participantes, evidências, especificação e plano; JSON apenas em Advanced/Developer. |
| UX-05 | **P0** | Seletor de entidade coleta `state.entities` + seções do `snapshot`; `continuity` inicia com `limit=25` (`app.js` 405–430, 1755–1829). | Opções ausentes nas páginas não carregadas; aparência de lista completa quando não é. | **Reference Picker** pesquisável com API server-side, paginação, tipagem e escopo explícitos. |
| UX-06 | **P0** | Seletores nativos listam itens sem busca, agrupamento ou descrições discriminativas (`app.js` 1755–1830). | Difícil distinguir dezenas/centenas de itens ou títulos duplicados. | Combobox acessível com busca, tipo, trilha de contexto, ID curto, estado e disponibilidade. |
| UX-07 | **P0** | A navegação gera 16 seções misturando coleções, status/projeções, referências e histórico (`app.js` 496–518). | Modelo mental ambíguo e hierarquia horizontal inchada. | 5–6 áreas organizadas por finalidade; status viram filtros/visões, não seções independentes. |
| UX-08 | **P1** | `Outcome`, `Objective`, `WorkItem`, `Roadmap`, `Issue` e `Blocker` aparecem de forma técnica/traduzida por “Resultado”, “Trabalho”, “Problema”, “Plano” etc. (`presentation.js`, `app.js`). | O usuário não entende quando usar cada conceito. | Glossário inglês consistente + ajuda contextual com exemplos curtos. |
| UX-09 | **P1** | Formulário de Roadmap utiliza nós, chaves, posições, `after_links`, revisões e schemas técnicos (`app.js` 1645–1737, 1889–2051; domínio Roadmap). | Planejamento, replanejamento e publicação exigem engenharia de estrutura de dados. | Editor visual de fases, marcos e referências; Draft → Review → Publish → Activate visíveis e separados. |
| UX-10 | **P1** | `quickActions()` filtra apenas um subconjunto de transições; `entity-actions` volta a oferecer catálogo genérico (`app.js` 1395–1461, 1624–1644). | O próximo passo não fica claro; ações perigosas/irrelevantes podem ser exibidas. | Determinar affordances por entidade, estado, permissão e protocolo; botões primários contextuais. |
| UX-11 | **P1** | A versão conflitante pede fechar, atualizar e revisar (`app.js` 2083–2150). | Interrupção da tarefa e potencial perda de entrada não enviada. | Reter o rascunho local, mostrar comparação e exigir reconfirmação após obter versão nova — nunca reaplicar silenciosamente. |
| UX-12 | **P1** | Pesquisa/filtros de itens atuam sobre itens carregados; documentação reconhece limitação (`app.js` 816–825, 1068–1172; `docs/workspace-ux.md`). | Resultado de busca pode parecer ausência real quando há itens paginados. | Busca/filtros server-side, paginação coerente; provisoriamente sinalizar escopo parcial de forma proeminente. |
| UX-13 | **P1** | Idioma `pt-BR` no HTML e formatador de data `pt-BR`; labels e mensagens dispersas em múltiplos arquivos. | Inglês/português/termos de API podem coexistir. | Fonte única de strings en-US, `lang="en"`, formatos internacionais e verificador CI. |
| UX-14 | **P1** | `default()` preenche `ttl_seconds=900`, `priority=normal`, `lifecycle=todo` e campos de autoridade/versão; `field()` pode exibir detalhes técnicos. | Defaults importantes sem explicação; erros de interpretação. | Campos técnicos nunca oferecidos como entrada humana comum; tempos em unidades humanas e defaults derivados de políticas do servidor. |
| UX-15 | **P1** | Início exige token e Namespace; `Servidor local` pede UUIDv7 (`index.html` 51–81); não há onboarding conceitual completo. | Curva inicial alta. | Acesso → escolha de Workspace → primeira Outcome → critérios → tarefas. Fluxo técnico local isolado. |
| UX-16 | **P1** | Testes E2E executam ações selecionando IDs como `create_work_item` e `add_criterion` (`tests/web/journey.spec.mjs`). | Validam API/UI técnica, mas não detectam dificuldade de descoberta. | E2E por texto visível/intenção, testes de acessibilidade, tarefas exploratórias cronometradas. |
| UX-17 | **P1** | Navegação do resultado é estado transitório JS; documentação declara que a seleção não é restaurada após reload. | Navegação não compartilhável nem recuperável. | Rotas/URLs profundas de contexto, outcome, seção e detalhe com Back/Forward. |
| UX-18 | **P2** | `app.js` tem 2.456 linhas; `style.css`, 1.850; uso de DOM imperativo e renderização combinada com HTTP/regras de ação. | Evolução de formulários e testes de UI custosa. | Modularização por feature; decidir migração TS/React após definição das jornadas. |
| UX-19 | **P2** | Painéis assinados expõem nomes de versões/leases e comandos `wosctl` sem fluxo de orientação visual (`app.js` 2302–2456). | Dificulta operação humana de acompanhamento/review. | Painel de execução/revisão orientado a etapas com comandos copiáveis e indicação explícita do que só o CLI autorizado pode fazer. |
| UX-20 | **P2** | CSS possui responsividade e `prefers-reduced-motion`, mas não há certificação WCAG no relatório. | Qualidade de acesso por teclado/leitores não comprovada. | Auditoria WCAG 2.2 AA com Playwright/axe, teclado real, foco, mensagens de erro, zoom 200%/400%. |

**Fontes primárias:** [app.js](https://github.com/A1b3rt0M3rcad0/wos/blob/5ea787c947e100a57e9b283bc1131245600888d3/packages/wos-api/web/assets/app.js), [index.html](https://github.com/A1b3rt0M3rcad0/wos/blob/5ea787c947e100a57e9b283bc1131245600888d3/packages/wos-api/web/assets/index.html), [presentation.js](https://github.com/A1b3rt0M3rcad0/wos/blob/5ea787c947e100a57e9b283bc1131245600888d3/packages/wos-api/web/assets/presentation.js), [command catalog](https://github.com/A1b3rt0M3rcad0/wos/blob/5ea787c947e100a57e9b283bc1131245600888d3/packages/wos-api/commands/catalog_generated.go), [workspace UX doc](https://github.com/A1b3rt0M3rcad0/wos/blob/5ea787c947e100a57e9b283bc1131245600888d3/docs/workspace-ux.md).

## 4. Preservar as invariantes do WOS

A interface **não** deve simplificar o modelo de forma destrutiva.

| Entidade | Significado preservado | Nome primário da UI |
|---|---|---|
| `Namespace` | Isolamento, autorização e contexto persistente. | **Workspace** (em telas comuns); Namespace em Advanced/Settings. |
| `Outcome` | Resultado final pretendido com condições verificáveis. | **Outcome**. |
| `Objective` | Condição intermediária verificável, eventualmente hierárquica. | **Objective**. |
| `WorkItem` | Unidade concreta de trabalho operacional. | **Task** (tipo interno continua `work_item`). |
| `SuccessCriterion` | Condição explícita cuja avaliação contribui para certificação. | **Success criterion**. |
| `Roadmap` | Plano persistente, versionado, que **referencia** Objectives/Tasks sem possuí-los. | **Roadmap**. |
| `Issue` | Registro de problema. | **Issue**. |
| `Blocker` | Impacto bloqueante associado a alvo, com causa e resolução próprios. | **Blocker**. |
| `Evidence` / `Artifact` | Prova observada / artefato produzido ou registrado. | **Evidence** / **Artifact**. |
| `WorkContract` | Autoridade exclusiva, duração/fencing, especificação congelada e entrega. | **Work contract**. |
| `ReviewCase` | Revisão própria de material aceito, separada da execução. | **Review**. |

**Invariantes inegociáveis:**

- Task `done` **não** torna Objective `achieved`; Objective `achieved` **não** torna Outcome `achieved`.
- “Issue resolved” não libera automaticamente Blockers; “Blocker resolved” não resolve automaticamente o Issue.
- Roadmap é referencial e versionado. Alterar a ordem visual ou remover referência do draft **não** cancela Task/Objective nem altera transições operacionais.
- Prova/evidência não certifica critério automaticamente; waiver não é `met` e exige autorização/motivo.
- Ações mutáveis respeitam versão esperada, Namespace autorizado, protocolo ativo, idempotência e estado de contrato/lease.
- No protocolo `signed_contracts_v2`, browser sem chave privada **não** assina operações nem simula aceite/review. Mostrar instruções e estado não equivale a executá-los.

Base: [ADR-001](https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/adr/0001-outcome-oriented-domain-model.md), [ADR-003](https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/adr/0003-roadmap-ownership-and-versioning.md), [ADR-004](https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/adr/0004-issues-and-blockers.md), [ADR-013](https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/adr/0013-verifiable-conclusions-and-assessment-history.md), [ADR-024](https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/adr/0024-signed-handoff-and-review.md).

## 5. Arquitetura de informação proposta

### 5.1 Navegação principal

```text
WOS
├─ Workspace Switcher
├─ Outcomes
│  └─ Selected Outcome
│     ├─ Overview      [intent, progress, recommended next actions]
│     ├─ Plan          [Objectives + Roadmap + success criteria]
│     ├─ Work          [Tasks: List / Board; filters by readiness/status]
│     ├─ Issues        [Issues + Blockers, with relationships explicit]
│     ├─ Evidence      [Evidence + Artifacts + Decisions]
│     └─ Activity      [History + audit + related graph]
└─ Workspace Settings [security, credentials, integrations, protocol]
```

O agrupamento é de **experiência**, não de persistência. Não mudar contratos do Domain para adequar a UI. `ready_work`, `blocked_work`, `in_progress_work`, `scheduled_work` etc. passam a filtros no Work; `active_roadmaps` e `active_plan_references` aparecem no Plan; `conclusion_contestations` aparece em Overview/Activity. Um painel `Reviews` pode tornar-se navegação principal se o uso real justificar volume operacional, sem duplicar `ReviewCase` ou `WorkItem`.

### 5.2 Overview orientado a decisão

Acima da dobra:

- Outcome + `desired_state` em linguagem legível e status real.
- Progresso de Tasks, Objectives, success criteria — **denominador sempre visível**, sem confundir waived com met.
- **Next recommended action** (`Define success criteria`, `Add objective`, `Create task`, `Review submitted evidence`, `Activate outcome`, `Inspect blockers`) escolhida a partir das leituras, com fundamento.
- Items needing attention: blockers, expiring contracts, review pending, contested conclusions.
- `Create task` e `Add objective` em posições consistentes; menu de ações complementares apenas quando relevante.
- Empty state específico: **sem outcome**, **outcome sem critérios**, **sem tarefas**, **sem roadmap**, **sem evidência** não podem compartilhar texto genérico.

Os próximos passos são **sugestões da interface**, não comandos emitidos automaticamente nem autonomia decisória do WOS.

### 5.3 Taxonomia de estados no Work

Distinguir claramente:

- **Lifecycle:** `Backlog`, `To do`, `In progress`, `Done`, `Cancelled`.
- **Readiness:** `Ready`, `Waiting for activation`, `Waiting for dependencies`, `Scheduled`, `Blocked`, `Needs attention`.
- **Execution authority:** `Unassigned`, `Contract active`, `Contract expired`, `Awaiting review` etc.

Não usar o título de uma coluna como substituto de estado persistido. Board reflete projeções reais; **drag-and-drop não muda status sem uma operação explícita validada**. Se futuramente houver arraste, ele deve abrir confirmação de comando e respeitar os mesmos guards.

## 6. Criação e edição: jornadas-alvo

### F01 — Criar Outcome (P0)

**Entrada:** `+ New outcome` na navegação e estado vazio.  
**Passo 1 (essencial):** `Outcome name*`, `What does success look like?*` (`desired_state`); `Description` opcional. Contexto Workspace pré-selecionado.  
**Passo 2 (recomendado, não transação implícita):** `Define success criteria` como etapa pós-criação. `Criterion title`, `How will it be verified?`, `Required` com explicação.  
**Passo 3:** `Add objectives` ou `Create tasks`; roadmap opcional, de acordo com caso.  
**Passo 4:** `Review readiness` → `Activate outcome`, somente se o servidor aceitar as precondições.

Ao confirmar o Passo 1, o Outcome **já existe em Draft**; fechar os passos seguintes não deve implicar rollback fictício. Exibir `Draft saved`, checklist de configuração e botão `Continue setup`. Não criar automaticamente critérios, objetivos nem ativação sem intenção explícita.

### F02 — Criar Objective (P0)

`+ Add objective` no Plan ou no detalhe de um Objective. Campos: `Title*`, `Description`, `Parent objective` (picker tipado, opcional), `Required for outcome` (explicar impacto), `Priority` (Advanced). Após criação: `Define success criteria`, `Add task`, `Start objective` de acordo com precondições. Visualizar a hierarquia no Plan; sem confundir Objective com uma Task.

### F03 — Criar Task (P0)

`+ New task` na área Work e em cada Objective. Campos prioritários: `Task title*`, `Description`, `Objective` (opcional), `Priority` (opcional/default), `Initial state` (Backlog/To do com ajuda). Área **Execution instructions** com controles legíveis para as listas `instructions`, `constraints`, `deliverables`, `scope_hints`, `context_refs` de `ExecutionSpec` (opcionais na API). `Schedule` como data legível. `Depends on` com picker tipado na configuração avançada/etapa subsequente. `Success criteria` podem ser definidos após criação. A UI não deve obrigar a preencher toda a especificação se o contrato de criação não obriga.

Após criar: abrir o detalhe com `Edit`, `Define acceptance`, `Add dependency`, `Reserve via contract` ou `Open in CLI` (conforme protocolo), sem abrir outro catálogo de comandos.

### F04 — Planejar Roadmap (P1)

Editor próprio:

1. `Create roadmap` com nome e escopo (Outcome ou Objective).
2. `Add phase`, `Add milestone`, `Reference objective/task` com picker tipado; elementos com títulos claros.
3. Ordenação e hierarquia visual sem expor `node_key`, `position`, `parent_node_key`, `after_links` ao usuário comum; a aplicação preserva chaves estáveis internamente.
4. `Save draft` mantendo `draft_version`, `Review changes` com diff e `reason`.
5. `Publish revision` → `Activate revision` como ações distintas e explícitas.

Mostrar que a revisão publicada é imutável; alterar plano não muda automaticamente Task lifecycle. Dependências operacionais exigem operação explícita, inclusive quando uma publicação contém mudanças de dependências validadas atomicamente pelo domínio.

### F05 — Definir/verificar critérios e evidências (P1)

No detalhe do proprietário (Outcome/Objective/Task): `+ Add success criterion`. Modo de verificação com **opções explicadas**:

- `Attestation` — declaração de avaliação autorizada;
- `Evidence review` — avaliação referenciando evidências registradas;
- `External evaluation` — origem externa com informação do avaliador.

Tela `Assess criterion`: resultado (`Met`, `Not met`, `Inconclusive`, `Waived` quando autorizado), justificativa, evidências escolhidas com picker; versão do critério fixada internamente. Separar `Register evidence` de `Assess criterion` de forma inequívoca. Em `Waived`, explicar que a condição foi dispensada, **não comprovada**.

### F06 — Reportar Issue e Blocker (P1)

Ações distintas: `Report issue` cria problema; `Mark task as blocked` registra impacto em alvo e causa (Issue opcional quando permitido). Oferecer fluxo combinado `Report issue and block work` utilizando o comando composto existente quando apropriado. No detalhe, conectar Issue ↔ Blockers; `Resolve issue` e `Release blocker` como decisões diferentes. Para resolver ambos, usar operação composta explicitamente apresentada, nunca efeito colateral invisível.

### F07 — Execução, contratos e revisão (P1/P2)

No detalhe da Task: `Readiness` + `Current contract` + `Holder` + `Expires` + `Latest submitted result` + `Review status` + `Next action`. Se `signed_contracts_v2`, oferecer instrução copiável para `wosctl`/profile; avisar que a assinatura exige chave no host e autorização independente. Não colocar prompts que simulem assinatura no navegador. Caso pendente: `Awaiting independent review`, `Changes requested` e findings ligados ao material correto.

### F08 — Edição, conflito e resposta incerta (P0/P1)

Edição inline/drawer de campos usuais, com `Save changes`. Preservar draft do usuário durante erro. Em `version_conflict`, mostrar `This item changed while you were editing` + dados atualizados + alterações locais; usuário decide rebase/descartar. Nova intenção recebe nova idempotency key; repetição de resposta incerta **reusa a chave e o payload original**. Não esconder CAS, fencing ou auditoria do backend — esconder apenas a manipulação manual do usuário comum.

## 7. Componentes e controles necessários

### 7.1 Reference Picker (P0)

**Contrato de interação:**

- Combobox `Search tasks, objectives, evidence…` com teclado completo, suporte a leitor de tela, `aria-expanded` e `aria-controls` e tratamento de Escape/Enter/Arrow keys.
- Dados buscados no **servidor** dentro de Namespace e Outcome autorizados, nunca da lista paginada em memória como fonte total.
- Escopo por tipos admissíveis para cada campo (`objective_id` → Objectives, `blocked_ref` → alvos elegíveis, `evidence_ids` → Evidence etc.).
- Cada resultado: `Title`, `Type`, `State`, hierarquia/contexto e short ID; suportar títulos duplicados e ausência de título.
- Debounce controlado, resultados limitados/paginados, estado loading/no results/error e carregamento adicional.
- Valor selecionado continua válido quando item sai da lista; endpoint resolve ref pelo ID; cache é otimização, não autorização.
- Se cursor ficar inválido por revisão ou escopo, oferecer reload explícito preservando seleção existente; respostas assíncronas antigas descartadas.
- Nunca retornar refs de outros Namespaces; validar novamente ao salvar.

**API proposta (aditiva, a ser discutida com contratos):**

```http
GET /api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/references
  ?kinds=objective,work_item
  &query=review
  &limit=20
  &cursor=...
```

Resposta: `items: [{ref, title, kind, lifecycle, short_id?, parent_ref?, eligibility?}]`, `next_cursor`, `outcome_revision` quando aplicável. Chaves e shapes finais devem seguir convenções da API existente. `eligibility` opcional é **orientação**, não substitui guardas transacionais. Usar filtros e índices de busca coerentes em Memory/SQLite/Postgres; não montar uma listagem não limitada em memória para filtrar no frontend. Para conjunto pequeno de enums (priority/severity), seletor normal continua apropriado.

### 7.2 Action Registry humano (P0)

Introduzir configuração de apresentação separada do `Catalog` gerado:

```text
CreateOutcomeAction    -> create_outcome
AddCriterionAction     -> add_criterion
CreateTaskAction       -> create_work_item
CompleteTaskAction     -> complete_work_item OR signed CLI guidance
PublishRoadmapAction   -> publish_roadmap_draft
```

Cada ação contém: identificador de UI, label en-US, owner kinds admissíveis, pré-condições **de exibição**, guardas de protocolo/autoridade, campos humanos, mapeador de payload, confirmação, resultado, tratamento de conflito. `Catalog` permanece para schemas/validadores/operações e MCP/SDK; não é o menu de navegação. Quando existir endpoint de `allowed_actions`/`capabilities`, torná-lo fonte de elegibilidade efetiva. Até lá, frontend deriva visibilidade conservadora e servidor mantém decisão final. Não confiar em ocultar botões como mecanismo de permissão.

### 7.3 Form Engine orientado a domínio (P0/P1)

Criar formulários dedicados por jornada. Cada campo possui `label`, `description`, `required`, `validation`, `default`, `conditionalVisibility`, `serializer`, `errorMap`. Renderizador de JSON Schema pode permanecer para `Developer tools`, mas **não** ser o formulário principal. Exemplos:

- `ttl_seconds` → `Reservation duration` em minutos com limites/política do servidor;
- `expected_version`, `fencing_token`, `claim_id`, `spec_digest`, `criterion_revision` → preenchimento automático/estado protegido;
- `context_refs` → seletor múltiplo de referências, não JSON;
- `instructions/constraints/deliverables` → linhas repetíveis ou editor de itens, não objeto cru;
- `assignee_refs` → seletor tipado de participantes quando suportado pelo backend;
- `ExternalContext` → editor de chave/valor validado em Advanced, jamais apresentado como algo que concede autorização.

Campos condicionais só aparecem quando a escolha tem consequência. Erros próximos ao campo, resumo acessível no topo e mensagem de sucesso vinculada à ação (`Task created`, `Roadmap revision published`), não `Alteração registrada` genérica.

### 7.4 Navegação, estados e histórico

- URLs profundas: Workspace / Outcome / view / item; retorno via browser Back/Forward; compartilhar ref sem conceder acesso.
- Busca global de Outcomes permanece server-side. Busca de Tasks/Objetivos/Evidência precisa ser server-side se a interface promete cobertura integral.
- Carregamento/skeletons e indicadores claros; evitar `no results` antes de terminar a busca.
- Resumo de progresso mostra base completa e motivo de `not applicable`; se não há critérios, incentivar definição em vez de apresentar 0% como falha.
- Drawer só mostra ações possíveis; demais em overflow contextual. `Technical details` recolhido para auditores.
- Temas e dimensões responsivos preservando hierarquia legível em 390px e zoom alto.

## 8. Migração integral para inglês

**Escolha recomendada:** `en-US` como idioma/cultura padrão para a V1, com *catálogo de mensagens* desde agora para evitar novas strings dispersas. Não é necessário entregar múltiplos idiomas nesta etapa.

**Cobertura obrigatória:**

1. `index.html`: `lang="en"`, title, labels, placeholders, botões, mensagens e dicas.
2. `presentation.js`: dicionários de lifecycle, readiness, kinds, prioridades, causas, critérios, ações e formatos `Intl`.
3. `app.js` (ou novos componentes): navegação, títulos, empty states, erros, contratos assinados, dashboard, datas relativas, toasts, telas administrativas.
4. Mensagens do **produto** ligadas à CLI (`wosctl`), manuais do usuário, onboarding, help, screenshots, templates/documentos de uso público. Go API/CLI já usam predominantemente nomes técnicos em inglês: não renomear identifiers wire.
5. Testes Playwright e fixtures de UI: selecionar labels em inglês; remover dependência de texto PT-BR. Contratos que conferem strings de erro internas seguem sem mudança quando wire exigir estabilidade.
6. Auditoria automática: negar `lang="pt-BR"`, `Intl(... "pt-BR")` e strings portuguesas nas superfícies públicas (exceções documentadas para material histórico, exemplos de conteúdo criado pelo usuário e eventuais docs legados).

**Glossário de UI:**

| Interno | Exibir em inglês | Evitar |
|---|---|---|
| `outcome` | Outcome | Result (genérico) |
| `objective` | Objective | Goal/Task indistintamente |
| `work_item` | Task | Work (sem entidade) |
| `desired_state` | Desired outcome | Desired state (campo técnico sem contexto) |
| `criterion` | Success criterion | Rule / Item |
| `evidence_review` | Evidence review | Raw enum |
| `blocked_ref` | Blocked item | Reference ID |
| `cause_ref` | Blocking cause | Cause ref |
| `readiness` | Readiness | Status (se confundível com lifecycle) |
| `achieve_outcome` | Mark outcome as achieved | Complete task |
| `waived` | Waived (not verified) | Met |
| `work_contract` | Work contract | Generic reservation |
| `roadmap` | Roadmap | Task list |
| `namespace` | Workspace (UI comum) | Technical namespace UUID |

**Exemplos de microcopy:**

- `What outcome do you want to achieve?`
- `Describe the result that would count as success.`
- `An outcome stays in Draft until you activate it.`
- `Tasks are complete, but the outcome still needs verification.`
- `Select an objective (optional).`
- `Search all eligible tasks in this outcome.`
- `This task is blocked by an unresolved dependency.`
- `Evidence has been recorded. Assess a success criterion to use it in a conclusion.`
- `A newer version exists. Review changes before saving.`
- `This action requires a signed CLI profile; the browser cannot sign on your behalf.`

Datas: `Intl.DateTimeFormat("en-US", {dateStyle: "medium", timeStyle: "short"})` para apresentação, deixando timezone claro quando relevante. APIs continuam ISO 8601; timestamps/digests/IDs não sofrem conversão destrutiva. Adotar pluralização e acessibilidade corretamente. Conteúdo **do usuário** previamente escrito em português não deve ser traduzido, regravado nem migrado automaticamente.

## 9. Decisão técnica: como implementar o frontend

### Alternativa A — Refatoração em ES Modules sem trocar stack

**Vantagens:** mudança menor no pipeline Go, sem dependência nova em produção, preserva testes existentes. **Limitação:** manutenção de estado, efeitos assíncronos, formulários complexos e controles acessíveis continuará custosa com manipulação imperativa do DOM. Adequada para correção imediata de textos/catálogo, mas pouco escalável como arquitetura final.

### Alternativa B — React + TypeScript + Vite com build estático embutido no binário Go

**Vantagens:** componentes testáveis, tipagem de DTO, React Hook Form + schemas quando apropriado, React Query/TanStack Query ou equivalente para estado do servidor, controle de dialogs/pickers, suporte a bibliotecas maduras de acessibilidade, escalabilidade dos fluxos. **Desvantagens:** migração da UI existente, dependências/build e novos testes de integração/distribuição. Node somente em build; servidor final continua Go.

**Recomendação:** definir e aprovar **primeiro** a arquitetura da experiência e os contratos, depois implementar o novo frontend de longo prazo em React/TypeScript/Vite. Não iniciar por um redesign cosmético nem criar segunda fonte de estado; o backend WOS segue fonte de verdade. Para liberação rápida, os fixes críticos (idioma e remoção do catálogo bruto) podem ser entregues incrementalmente no frontend atual antes da migração integral, contanto que não sejam reimplementados duas vezes sem necessidade.

**Organização de referência:**

```text
packages/wos-api/web/
  src/
    app/                 # shell, routing, authenticated workspace
    features/
      outcomes/
      objectives/
      tasks/
      roadmaps/
      criteria/
      evidence/
      issues/
      reviews/
      administration/
    shared/
      api/               # client, DTO, idempotency, version conflicts
      components/        # Dialog, Combobox, ReferencePicker, ConfirmAction
      design-system/     # tokens, semantics, theme, responsive
      i18n/en-US.ts      # official product strings
      domain-ui/         # action registry, statuses, capability mapper
      utils/
  dist/                  # static build embedded in Go
  web.go
```

`web.go` hoje usa `//go:embed assets/*`, aplica CSP e `no-store`. Se houver migração, atualizar embedding/manifest, conteúdo estático gerado, cache policy, Dockerfile, `web.yml` e pipeline de distribuição para build determinístico (lockfile e versões fixadas). Preservar `script-src 'self'` sem `unsafe-inline` e nenhum token/private key em armazenamento do navegador. Nenhuma mudança de React altera o Domain por si só.

## 10. Contratos/API: mudanças mínimas e isoladas

### 10.1 Exigido para o Reference Picker

Novo endpoint de **pesquisa autorizada e paginada de referências** (contrato do item 7.1). A implementação deve cumprir scoping, grants, limites, cursores consistentes, índices e paridade Memory/SQLite/Postgres. Reusar endpoints atuais somente quando eles entregarem busca completa; não declarar como completo o que é paginado parcialmente. A semântica de cursor e snapshot do ADR-014 deve continuar coerente.

### 10.2 Recomendado para UX state-aware

Projeção de `available_actions` ou `ui_capabilities` **read-only**, derivada dos mesmos guards/aplicação onde possível, para a UI saber quais ações mostrar e por quê. Não tornar frontend fonte de autorização. O endpoint pode ser adiado se o primeiro release consumir estado e permissões existentes com mapeamento conservador; entretanto, não inferir permissão real apenas do lifecycle.

### 10.3 Só quando a experiência exigir

Se o wizard precisar criar múltiplas entidades **atomicamente**, criar um Application command composto específico e idempotente com garantias explícitas. Não agrupar chamadas independentes fingindo rollback em caso de falha. É aceitável lançar fluxo por etapas com Draft persistente e `Continue setup`; é mais barato e consistente com os contratos atuais.

## 11. Erros, concorrência e auditoria

- **400/422**: campo inválido destacado, mensagem legível e foco no primeiro erro. Se backend não devolve `field_path`, proposta aditiva de erro estruturado (`code`, `message`, `field_errors`) sem quebrar clientes.
- **403**: explicar que usuário não possui autorização ou profile exigido; não oferecer botão que induza à assinatura no browser.
- **404**: referência indisponível ou fora do escopo, não reconstruir UUID a partir do título.
- **409/412**: conflitos de versão e cursores; mostrar mudança, preservar draft e exigir nova intenção/revisão.
- **429/5xx/network unknown**: proteger idempotency key e payload congelado na tentativa de recuperação; não duplicar evento.
- **Expiração/revogação de contrato**: refletir estado efetivo; não oferecer retomada sem autoridade.
- **Ação irreversível/impactante**: confirmação específica com entidade, motivo e efeito (arquivar, cancelar, revogar, publicar, ativar, certificar), nunca `Are you sure?` genérico.
- **Security**: CSP existente preservada, autorização no backend, nenhum segredo persistido no frontend, conteúdos de usuário escapados e renderização de links segura.

## 12. Plano de implementação por fases e épicos

### Fase 0 — Baseline e contrato de experiência (obrigatória)

- [ ] Documentar 8 jornadas F01–F08 com mapa de telas, ações, estados e payloads.
- [ ] Inventariar todas as strings visíveis, enums, erros, placeholders, validações, ajuda, telas/CLI/documentação pública.
- [ ] Classificar todos os 100 comandos: primário humano, contextual, avançado/admin, CLI/protocolo assinado, não exposto.
- [ ] Definir glossário inglês e taxonomia de estados.
- [ ] Gravar dataset repetível (vazio, pequeno, 32 TaskItems, 1.000+ entidades, títulos duplicados e múltiplos estados).
- [ ] Registrar baseline observável (tempo para criar Outcome/Objective/Task, taxa de erro e necessidade de ajuda); se indisponível, marcar baseline como **não medida**.

**Entregas:** IA aprovada, wireflows, action matrix, copy deck en-US, critérios de aceitação por jornada.

### Fase 1 — Correções críticas de uso e idioma (P0)

- [ ] Interface 100% inglês nos fluxos públicos; `lang=en`, Intl e dicionário unificado.
- [ ] Remover seletor de 88 opções dos fluxos comuns; reservar command catalog para interface técnica autenticada, se mantida.
- [ ] Criar actions `New outcome`, `Add objective`, `New task` com formulários específicos.
- [ ] Mostrar labels/help de conceito na primeira utilização (não tutorial obrigatório).
- [ ] Remover JSON cru dos formulários de criação comuns.
- [ ] Implementar Reference Picker com pesquisa real e resultados inequívocos.
- [ ] Garantir persistência clara de Outcome Draft durante wizard.
- [ ] Atualizar E2E para verificar interface em inglês e acesso sem command IDs.

**Gate de aceite:** um usuário consegue criar Outcome, Objective e Task sem selecionar qualquer comando da API, preencher UUID ou serializar JSON; nenhum item autorizado que existe fora da primeira página fica irrecuperável pelo picker.

### Fase 2 — Navegação, plano e verificabilidade (P1)

- [ ] Substituir 16 seções por IA centrada em `Overview / Plan / Work / Issues / Evidence / Activity`.
- [ ] Migrar projeções para filtros, mantendo distinção lifecycle/readiness/contract.
- [ ] Criar editor visual de Roadmap com draft, diff, publish e activate explícitos.
- [ ] Criar fluxo de Success criteria + evidências + assessment.
- [ ] Criar fluxo de Issue/Blocker que preserve independência e comandos compostos.
- [ ] Integrar URL/deep-link e History API; manter contexto após reload e Back/Forward.
- [ ] Tratamento de conflito preservando edição local.

**Gate:** roadmap criado/replanejado/publicado sem editar JSON/nós manualmente; controles de conclusão exibem avaliação necessária e não certificam automaticamente.

### Fase 3 — Operação colaborativa e signed protocol (P1/P2)

- [ ] Próxima ação sugerida por estado e autoridade, sem execução autônoma.
- [ ] Visibilidade de contratos, vencimento, entrega, revisão e findings.
- [ ] Handoff para CLI com instrução contextual copiável e profile correto, sem guardar assinatura/chave privada.
- [ ] Busca/filtros completos para Work/Outcomes onde prometer cobertura integral.
- [ ] Administração com permissões em seleção tipada, não entrada de strings arbitrárias.
- [ ] Componentizar frontend e revisar React/TS (ou concluir migração planejada) com testes de componente.

**Gate:** usuário distingue `Task done`, `Outcome achieved`, `Review pending` e `Blocker unresolved` sem examinar payload técnico.

### Fase 4 — Qualidade e distribuição (obrigatória antes de declarar completo)

- [ ] Testes unitários do action mapping, validações, tradução, cursor, idempotency e conflitos.
- [ ] Playwright end-to-end contra servidor real (login, F01–F08, 390px, desktop, 200% zoom, teclado completo).
- [ ] axe-core/WCAG 2.2 AA orientados aos componentes implementados; teste manual com leitor de tela.
- [ ] Paridade dos endpoints novos em Memory/SQLite/Postgres e avaliação com dataset volumoso.
- [ ] CI build e distribuição reproduzível (Go embed, CSP, Docker, release archive, npm CLI) e reexecução dos workflows hospedados.
- [ ] Atualizar capturas e docs de UX em inglês após testes **reais**.
- [ ] Ensaio de regressão de `signed_contracts_v2`, grant/revogação, cursor invalidado, snapshot antigo e replay idempotente.

## 13. Critérios objetivos de aceite e indicadores de melhoria

> Metas propostas para a nova UX; **não** são medições atuais nem resultados garantidos.

| Indicador | Meta de aceite proposta | Como medir |
|---|---|---|
| Compreensão | ≥ 90% identificam corretamente Outcome vs Objective vs Task após breve exploração | Teste moderado com novos usuários |
| Criação de Outcome | ≥ 90% concluem o Draft sem orientação | Teste de tarefa e gravação |
| Criação de Task | ≥ 90% sem abrir Advanced nem campo JSON | Teste de tarefa |
| Seleção de referência | 100% dos candidatos autorizados buscáveis, inclusive fora da página inicial | E2E com 1.000+ referências e nomes duplicados |
| Idioma | 0 strings portuguesas **de interface oficial** nas jornadas cobertas | Script/lint + revisão humana |
| Comandos técnicos | 0 seletores globais de command IDs na UI comum | Inspeção de telas e Playwright |
| Controle de versão | 100% dos cenários de conflito preservam entrada e não fazem mutação silenciosa | E2E CAS/409 |
| Retry incerto | 0 intents duplicadas em cenários de replay/erro 5xx simulado | Testes de idempotência |
| Acessibilidade | Sem violações críticas/serious automatizadas e jornadas de teclado funcionais | axe + testes manuais |
| Mobilidade | Sem overflow horizontal não intencional em 390px e zoom 200% | Playwright visual/manual |
| Protocolos | Mesmos guards e autorizações anteriores; sem operação assinada pelo browser | Go + Playwright + integração |

Na primeira fase, medir baseline antes/depois com amostra repetível e tarefas equivalentes. Não usar apenas a contagem de testes verdes como evidência de boa UX.

## 14. PRs sugeridos (separação por responsabilidade)

1. `ux/english-copy-and-glossary` — en-US, traduções, lint, documentação de copy; **sem domínio**.
2. `ux/human-action-registry` — mapeamento de comandos para intenções e disponibilidade por contexto.
3. `api/authorized-reference-search` — consulta tipada/paginada + índices + paridade e autorização.
4. `ux/reference-picker` — componente acessível + tratamento de cursor/duplicados.
5. `ux/outcome-objective-task-flows` — formulários e Draft onboarding.
6. `ux/navigation-and-deep-links` — arquitetura de informação, URLs e estados de navegação.
7. `ux/roadmap-editor` — editor visual com CAS/diff/publish/activate.
8. `ux/evidence-issues-review` — provas, avaliação, Issue/Blocker, signed read-only + CLI handoff.
9. `ux/a11y-usability-e2e` — testes de experiência, QA e observabilidade.
10. `build/web-client-modernization` — opcional/conforme decisão, Vite/React/TS e Go embed sem afetar headless clients.

Cada PR deve atualizar documentação do fluxo relevante e não alterar wire contracts do WOS sem ADR/compatibility gate.

## 15. Definition of Done final

A mudança está concluída apenas quando:

- [ ] Interface e textos de produto estão em inglês, com nomenclatura coerente e conteúdo do usuário preservado.
- [ ] Usuário cria, edita e navega Outcomes, Objectives, Tasks, Roadmaps, critérios, evidências e bloqueios com ações **contextuais**.
- [ ] Nenhuma operação humana comum exige descobrir um nome de command, UUID, fencing token, schema ou payload JSON.
- [ ] Pickers buscam o conjunto autorizado no servidor e diferenciam itens homônimos.
- [ ] Mudanças de plano e mudanças de estado não são confundidas.
- [ ] Outcome, Objective, Task e Review preservam conclusões, versões, evidências e autoridade independentes.
- [ ] Browser não finge capacidade de assinar contratos ou decisões que exigem profile externo.
- [ ] Conflitos, respostas incertas e snapshots atrasados não causam perda silenciosa ou duplicação de intenção.
- [ ] E2E, testes de API/storage, acessibilidade e build/distribuição foram **reexecutados** na revisão entregue.
- [ ] Usuários representativos conseguem concluir jornadas centrais sem treinamento na estrutura interna do WOS.

## 16. Arquivos prováveis para intervenção

| Área | Arquivos atuais / destino sugerido |
|---|---|
| Shell, idioma e navegação | `packages/wos-api/web/assets/index.html` → novo Shell/Router |
| Estado, comandos, formulários | `packages/wos-api/web/assets/app.js` → `features/*`, `shared/api`, `domain-ui` |
| Traduções, taxonomia e datas | `packages/wos-api/web/assets/presentation.js` → `shared/i18n`, `domain-ui/status` |
| UI design system | `packages/wos-api/web/assets/style.css` → tokens/componentes/responsive |
| Web embedding e CSP | `packages/wos-api/web/web.go` |
| Referências e queries HTTP | `packages/wos-api/http/{handler,continuity,objectives,work_items}.go` e adição de endpoint |
| Consulta autorizada | `packages/wos-core/application/{query,read_authorization,service}.go`, ports e adapters; novas queries sem mudar Domain indevidamente |
| E2E | `tests/web/{journey,workspace,contracts,signed}.spec.mjs`; novos UX specs |
| CI e distribuição | `.github/workflows/web.yml`, `Dockerfile`, `tools/distribution/build.mjs` |
| Documentação | `docs/workspace-ux.md`, `docs/ui-workspace/*`, READMEs públicos |

**Parecer final:** o WOS já diferencia corretamente resultados verificáveis, objetivos, trabalho, evidências, problemas, bloqueios, versões e autoridade no domínio. A UI atual força o humano a operar esses conceitos como se fosse um cliente de API. A proposta é **preservar a sofisticação interna e simplificar radicalmente a superfície de interação**. A prioridade real é **modelo mental + ações específicas + picker completo + inglês**, nessa ordem lógica; migração de framework e acabamento visual vêm a serviço dessas decisões, não como substitutos.

---

### Referências complementares de design

- [WAI-ARIA APG — Combobox pattern](https://www.w3.org/WAI/ARIA/apg/patterns/combobox/): teclado, semântica e acessibilidade de pesquisas/autocomplete.
- [Linear — Creating issues](https://linear.app/docs/creating-issues): criação direta e contextual.
- [Linear — Issue templates](https://linear.app/docs/issue-templates): propriedades defaults e modelos como melhoria de produtividade.
- [WOS — Workspace UX atual](https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/workspace-ux.md): escopo declarado pela implementação de 07/10/2026.

## 17. Matriz de migração do catálogo técnico para jornadas humanas

Esta matriz é a **classificação-alvo da experiência**, não uma declaração de que cada operação está autorizada em todos os estados. O catálogo gerado continua sendo o contrato de comandos; a interface deixa de utilizá-lo como menu. Uma mesma operação pode ter um CTA humano em uma jornada e aparecer em ações avançadas em outra. Em todos os casos, a API é autoridade final.

**Classes de apresentação:**

- **H** — operação humana com ação própria e explicação; visível somente no contexto admissível;
- **C** — ação contextual de manutenção, avaliação ou transição; normalmente em detalhe/overflow;
- **A** — Advanced/Admin; exige intenção especializada, confirmação e permissão;
- **L** — operação de execução/lease do protocolo legado; somente quando o protocolo efetivo permitir;
- **S** — operações do protocolo assinado; executadas via profile/CLI/SDK autorizado; Web apresenta estado e orientação, sem forjar autoridade;
- **I** — integração/configuração operacional, nunca ação primária de criação de trabalho.

| Área | Comandos catalogados | Apresentação recomendada |
|---|---|---|
| Outcome — criação/edição | `create_outcome`, `update_outcome` | **H** — New outcome / Edit outcome |
| Outcome — transições | `activate_outcome`, `achieve_outcome`, `fail_outcome`, `abandon_outcome`, `reopen_outcome`, `archive_outcome`, `unarchive_outcome` | **C** — botões e confirmações do detalhe, com evidências/precondições explícitas |
| Outcome — participantes | `set_outcome_owners` | **A/C** — Manage owners; selecionar ator com contrato tipado |
| Objectives | `create_objective`, `update_objective`, `start_objective`, `achieve_objective`, `cancel_objective`, `reopen_objective`, `set_objective_owners` | **H** criar/editar; **C** transições; **A/C** owners |
| Tasks — criação/edição | `create_work_item`, `update_work_item`, `set_work_item_assignees` | **H** criar/editar; **C** assignees |
| Tasks — estado | `activate_work_item`, `defer_work_item`, `reopen_work_item`, `cancel_work_item`, `administrative_cancel_work_item`, `administrative_complete_work_item` | **C** ações compatíveis com estado; **A** overrides administrativos |
| Critérios | `add_criterion`, `revise_criterion`, `retire_criterion`, `record_criterion_assessment`, `attest_criterion` | **H** criar/avaliar; **C** revisar/desativar; não presumir que atestar substitui o protocolo assinado |
| Roadmaps — criação e edição | `create_roadmap`, `open_roadmap_draft`, `replace_roadmap_draft`, `discard_roadmap_draft` | **H/C** editor visual: Create / Edit draft / Discard draft |
| Roadmaps — publicação/estado | `publish_roadmap_draft`, `activate_roadmap_revision`, `deactivate_roadmap_revision`, `archive_roadmap`, `reopen_roadmap` | **C** Preview → Publish → Activate; histórico/retomada específicos |
| Dependências | `add_dependency`, `remove_dependency` | **C** gerenciar no detalhe da Task; picker tipado e verificação de elegibilidade |
| Issues/Blockers | `create_issue`, `update_issue`, `investigate_issue`, `resolve_issue`, `reopen_issue`, `mark_issue_duplicate`, `mark_issue_wont_fix`, `create_blocker`, `update_blocker_description`, `resolve_blocker`, `cancel_blocker`, `report_issue_with_blocker`, `resolve_issue_and_blockers` | **H** reportar; **C** transições, vínculos e operações compostas; sem efeitos implícitos |
| Evidências | `register_evidence`, `retract_evidence`, `create_evidence_link`, `retract_evidence_link` | **H/C** Register evidence / Link evidence / Retract com contexto |
| Artefatos | `register_artifact`, `withdraw_artifact` | **H/C** Register artifact / Withdraw artifact |
| Decisões | `propose_decision`, `update_decision`, `accept_decision`, `reject_decision`, `supersede_decision` | **H/C** fluxo de decisão com estados explícitos; não misturar com resultado do critério |
| Referências externas | `link_external_reference`, `remove_external_reference`, `replace_external_context` | **A/C** relações e contexto; edição tipada de chave/valor com limites |
| Lease legacy v0 | `claim_work_item`, `renew_work_item_lease`, `reclaim_work_item`, `release_work_item`, `complete_work_item` | **L** operações na Task somente sob protocolo legado compatível |
| Contracts v1 | `acquire_work_contract`, `acquire_next_work_contract`, `renew_work_contract`, `resume_work_contract`, `sync_work_contract`, `submit_work_result`, `finalize_work_contract`, `revoke_work_contract`, `reconcile_expired_work_contracts` | **L** painel contextual de contrato; reconciliação/revogação administrativa em **A** |
| Contracts v2 assinados — execução | `acquire_signed_work_contract`, `acquire_next_signed_work_contract`, `renew_signed_work_contract`, `resume_signed_work_contract`, `return_signed_work` | **S** profile autorizado; painel Web somente leitura + handoff |
| Contracts v2 assinados — revisão | `acquire_signed_review_contract`, `acquire_next_signed_review_contract`, `renew_signed_review_contract`, `resume_signed_review_contract`, `return_signed_review` | **S** profile revisor autorizado e independente |
| Contracts v2 assinados — intervenção | `revoke_signed_contract`, `intervene_signed_review_case` | **S/A** autoridade administrativa assinada; estado e orientação na Web |
| Protocolo/integração | `set_namespace_work_protocol`, `configure_trigger`, `update_trigger`, `set_trigger_enabled`, `redeliver_delivery` | **I/A** Workspace settings / integrations; nunca Quick create |

**Gate obrigatório da implementação:** produzir `command-exposure-inventory.md` gerado ou verificado em teste contra o catálogo real. Todo comando deve possuir uma classificação, e a adição de novo comando deve falhar em CI se não tiver tratamento explícito. Não usar heurísticas por prefixos (`create_`, `update_`, `_signed_`) como substituto dessa classificação. Prefixos podem apoiar migração, mas não determinam comportamento de produto.

## 18. Especificação das telas e estados de interação

### 18.1 Workspace / Outcomes

**Visão vazia:** explicar brevemente o que é Outcome, CTA `Create outcome` e exemplo de resultado verificável. Não pedir UUID nem mostrar o catálogo técnico. O Workspace corrente e a identidade são exibidos em menu próprio.

**Lista de Outcomes:** nome, estado, prioridade, progresso sintético, data de atualização quando disponível, botão contextual; busca e paginação reais. Filtro de estado e arquivados permanece. Abrir uma Outcome restaura a URL selecionada e o foco sem sobrepor um resultado anterior após navegação rápida.

**Resultado após criação:** mostrar `Draft` e itens de configuração incompletos. Não mascarar ausência de critérios como certificação bem-sucedida. `Activate outcome` é uma ação independente, nunca consequência automática de `Create outcome`.

### 18.2 Outcome / Overview

**Hierarquia:** Outcome → propósito → progresso e ações → trabalho e verificações → atividades recentes. Diferença entre `Task progress` e `Outcome verification` evidente. Cada indicador possui numerador/denominador e hyperlink para itens que o compõem.

**Next step:** cálculo de apresentação determinístico, sem tomar decisão de domínio. Exemplo: sem critérios → `Define success criteria`; Outcome Draft configurada → `Review readiness`; revisão pendente → `Review submitted work`; itens bloqueados → `Inspect blockers`; conclusão validável → `Review outcome completion`. A sugestão não autoriza nem executa nada.

### 18.3 Plan — Objectives

Visualização hierárquica dos Objectives com indentação, estado, critérios e relação com a Outcome. `Add objective` abre formulário no contexto correto. Suportar pai opcional, distinção entre ser obrigatório ao resultado e prioridade operacional, e vínculo de novas Tasks ao Objective. Sem seleção arbitrária de referências de outros Outcomes.

### 18.4 Plan — Roadmaps

**Estados visuais:** `No roadmap`, `Draft in progress`, `Published revision`, `Active revision`, `Archived`. Um Roadmap pode existir sem draft; `Published` não significa automaticamente `Active`.

**Editor:** adicionar Phase/Milestone/Reference, edição de título, hierarquia, ordenação; mostrar dependências operacionais como entidade separada. Salvar draft usando versão atual; se houver mutação concorrente, reexibir mudanças antes de tentar novo CAS. A remoção de referência do draft não remove Task/Objective. Publicação mostra diffs e exige motivo conforme contrato.

**Preview da revisão:** componentes publicados e referências com snapshot da época, mesmo que seu estado operacional atual tenha mudado. Evitar substituir dados históricos pelos campos atuais sem identificação de temporalidade.

### 18.5 Work — List/Board

**Lista:** colunas Task, Objective, Lifecycle, Readiness, Priority, Contract/Review quando existir, Updated (quando disponível). Busca tipada e filtros server-side se prometida cobertura global. Páginas preservam cursores e não confundem contagem total com itens efetivamente carregados.

**Board:** colunas derivadas da projeção atual, sem mutação por arraste implícita. Quando há cards demais, carregar sob demanda mantendo total e estados de carga claros. Seleção e filtros consistentes com a lista. Estados `Blocked`, `Waiting dependencies`, `Pending review` não são sinônimos de lifecycle.

**Detalhe da Task:** bloco principal sobre o trabalho humano (descrição, objetivo, critérios, instruções/entregáveis) seguido por operação (readiness, dependências, contrato, evidências, revisão). `Technical details` fica recolhido. A ação primária muda conforme a fase e o protocolo.

### 18.6 Issues

Duas coleções vinculadas mas semanticamente distintas: `Issues` registram o problema, `Blockers` representam impacto. Exibir origem/causa, alvo impedido, abrangência (`direct`/`subtree`) e resolução. Uma Issue resolvida com Blocker ativo deve aparecer como estado legítimo e explicado, não como inconsistência da UI.

### 18.7 Evidence / Decisions

Seções internas `Evidence`, `Artifacts`, `Decisions`. Evidências mostram tipo, fonte/proveniência e vínculos a critérios; decisão possui alternativas/resultado quando aplicável. `Register evidence` não equivale a `Assess criterion`, nem a `Accept decision`. Histórico imutável deve ficar distinguível das projeções atuais.

### 18.8 Activity / Audit

Linha do tempo com ator, evento, alvo, instante, revisão e link para detalhe. Evidenciar autoria de agente/humano sem inferir confiança ou aprovação. JSON técnico somente sob `Show raw event`; manter paginação e ordenação originárias da API.

### 18.9 Workspace settings

Separar `Access & credentials`, `Members/permissions` quando a API fornecer catálogo adequado, `Work protocol`, `Integrations` e `Developer tools`. As configurações de protocolo não podem ser disparadas por engano enquanto o usuário cria uma Task. Recursos dependentes de chave local/profile recebem indicação explícita do ambiente onde são realizados.

## 19. Design do frontend e contratos internos de UI

### 19.1 Separação obrigatória de responsabilidades

```text
Domain/API (fonte de verdade)
  ├─ Authorized queries (scope, grants, cursors, revisions)
  ├─ Command catalog (schemas e compatibilidade técnica)
  └─ Mutations (CAS + idempotency + protocolo)
          │
          ▼
Typed API Client
  ├─ DTOs e validação de responses
  ├─ Route/Scope binding
  ├─ Query keys e cancelamento/descartar stale responses
  └─ Mutation coordinator / frozen intent / conflict handling
          │
          ▼
Human UX Application Layer
  ├─ Action registry explícito
  ├─ Form models por intenção
  ├─ Reference queries e selected-ref resolution
  ├─ Readiness/status presentation (não autoridade)
  ├─ Locale/copy en-US
  └─ Navigation + unsaved-draft policy
          │
          ▼
UI Components / Views
  ├─ Workspace shell / Outcome navigation
  ├─ Outcome/Objective/Task forms
  ├─ Roadmap editor / criteria / evidence
  └─ Tasks board / contracts / review / audit
```

**Obrigação de desenho:** uma decisão de UI que depende de protocolo/autorização deve ter contrato testável. Não implementar três versões distintas de `canCompleteTask` em botões, formulários e menus. O registry de ações e as capacidades retornadas pela API, quando existentes, devem dirigir todas essas superfícies.

### 19.2 Especificação mínima para uma ação humana

```ts
// Tipo ilustrativo, não alteração de API publicada.
type HumanAction = {
  id: string;                      // ex.: task.create
  titleKey: string;                // ex.: actions.task.create
  entityKinds: string[];
  placement: 'primary' | 'contextual' | 'advanced';
  capability: string;              // identidade de permissão/ação
  isApplicable(context: ActionContext): Applicability;
  open(context: ActionContext): FormModel | DialogModel;
  toCommand(input: FormInput, context: ActionContext): FrozenIntent;
  confirmation?: ConfirmationPolicy;
  onSuccess?: NavigationPolicy;
};
```

`Applicability` possui pelo menos `visible`, `enabled` e `reason`; não retorna autorização confiável da Web. `FrozenIntent` armazena nome da operação, payload, idempotency key, escopo e expectativa de revisão para a tentativa atual. Estado de formulário nunca se torna fonte canônica de `OutcomeRevision`.

### 19.3 Componentes reutilizáveis e regras

| Componente | Contrato essencial | Teste obrigatório |
|---|---|---|
| `EntityReferencePicker` | Busca remota tipada, cursor, seleção por ID, nomes duplicados, carregamento | 1.000+ resultados, escopo incorreto, teclado/ARIA |
| `MultiReferencePicker` | Seleção de várias evidências/dependências; chips removíveis e IDs preservados | Não perder seleção fora da página atual |
| `HumanActionMenu` | Somente ações aplicáveis, explicação de indisponibilidade, sem raw catalog | Cada perfil/protocolo/estado suportado |
| `ValidatedForm` | Campos essenciais, ajuda contextual, erro por campo, foco e revisão | Dados preservados em 422/409 |
| `CreateOutcomeFlow` | Draft persistido, passos opcionais, checklist claro | Abandonar setup não apaga Draft |
| `CreateTaskFlow` | Vínculo a Objective, instruções e entregáveis tipados | DTO serializado = contrato esperado |
| `RoadmapDraftEditor` | Nodes estáveis, ordenação, versão, diff | Reorder não edita lifecycle |
| `EvidenceAssessment` | Versão do critério, provas e estado da avaliação | Waived != Met; avaliação explícita |
| `SignedContractStatus` | Separação execução/revisão/autoridade, handoff CLI | Nenhuma assinatura simulada na Web |
| `ConflictReview` | Diff de dados do usuário/servidor, confirmação de nova intenção | Nenhuma perda ou rebase silencioso |
| `OperationNotice` | Sucesso/erro específico, recebimento/correlação quando adequado | Retry reaproveita idempotency key |
| `TechnicalAuditPanel` | Dump técnico sob demanda; copiar ID seguro | Nunca exibe token/chave privada |

### 19.4 Design system mínimo

Tokens semânticos de cores (default/attention/success/error), tipografia de hierarquia estável, espaçamento e contraste AA. Componentes com estados `idle/hover/focus/disabled/loading/error/success`; foco visível sem depender apenas de cor. Ações de destruição têm apresentação distinta da principal. Usar linguagem visual consistente; não transformar cada badge, status ou métrica em destaque cromático concorrente.

O WOS pode manter o estilo atual inspirado em interfaces operacionais compactas. **A decisão de framework e o acabamento devem seguir a usabilidade**, não antecipá-la.

## 20. API de busca de referências: protocolo de implementação

**Objetivo:** eliminar o erro de selecionar entre itens carregados parcialmente. Especificação proposta, sujeita à convenção final do projeto.

**Contrato inicial:**

```http
GET /api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/references
    ?kind=objective,work_item,evidence
    &query=validation
    &limit=20
    &cursor=...
```

Resposta de leitura autorizada e limitada:

```json
{
  "items": [
    {
      "ref": {
        "namespace_id": "<UUIDv7>",
        "outcome_id": "<UUIDv7>",
        "kind": "work_item",
        "id": "<UUIDv7>"
      },
      "title": "Validate integration",
      "lifecycle": "todo",
      "parent_title": "API readiness",
      "display_context": "Objective: API readiness"
    }
  ],
  "next_cursor": null,
  "outcome_revision": 42
}
```

Os `<UUIDv7>` acima são *placeholders ilustrativos*, não IDs válidos. Preservar o shape existente de `EntityRef` (`namespace_id`, `outcome_id`, `kind`, `id`). Não incluir informações não autorizadas. `display_context` é decorativo e pode ser omitido se o custo de join não se justificar.

**Invariantes do endpoint:**

1. Autenticar e autorizar com o mesmo principal/credential/grants da API de leitura do WOS, inclusive após revogação.
2. Validar Namespace/Outcome, tipos solicitados e limite. Tipos permitidos são enumerados; nunca aceitar nome arbitrário de tabela/coluna para montar SQL.
3. Ordenação determinística e cursor opaco associado a escopo, filtros, ordenação e revisão exigida pela semântica escolhida. Invalidar cursor explicitamente quando incompatível.
4. Efetuar busca no banco, não fazer fetch irrestrito para filtrar em JavaScript. Planejar índices PostgreSQL/SQLite e implementação Memory com limite correto e testes de paridade.
5. Responder apenas candidatos elegíveis ao tipo de vínculo, **sem declarar que elegibilidade de leitura equivale a autorização para mutação**.
6. Conferir referência/versão de novo na gravação; exclusão, alterações ou conflito entre busca e submit resultam em erro recuperável.
7. Impedir resultados obsoletos quando usuário muda Namespace/Outcome/query; a UI rejeita respostas de gerações anteriores.
8. Exibir estado de `No matching items` somente quando a consulta autorizada retornou efetivamente vazio, não quando uma das páginas locais está vazia.
9. Dar suporte a `resolve selected ref`: se item selecionado não estiver na página atual, preservá-lo e recuperar título por ID autorizado.
10. Instrumentar latência e taxa de busca vazia sem coletar tokens, material de evidência ou texto sensível por padrão.

**Decisão de escopo:** se a primeira entrega implementar busca somente para `Objective` e `WorkItem`, os demais campos dependentes de referências continuam com comportamento limitado **explicitamente sinalizado** e backlog obrigatório; não anunciar resolução global do UX-05 antes de cobrir também Evidence, Issue, Blocker, Roadmap, relations e objetos compostos relevantes.

## 21. Matriz de testes de aceitação por jornada

| Teste | Preparação | Ação executada por humano | Resultado verificável |
|---|---|---|---|
| E2E-01 | Workspace vazio | Criar Outcome com nome e estado desejado | Outcome em Draft, dados persistidos e identificador oculto no fluxo simples |
| E2E-02 | Outcome Draft | Definir criterion e adicionar Objective | Entidades corretas, avaliação ainda não concluída, sem ativação implícita |
| E2E-03 | Outcome com Objective | Criar Task, vincular Objective e definir deliverables | Task possui `ObjectiveID` e `ExecutionSpec` esperados sem JSON manual |
| E2E-04 | 1.000 Tasks / nomes iguais | Pesquisar e selecionar Task fora da primeira página | Referência correta e inequívoca; registro persiste |
| E2E-05 | 1.000 Evidence / nomes iguais | Selecionar múltiplas evidências em páginas diferentes | Seleção preservada por ID; nenhuma perda por paginação |
| E2E-06 | Roadmap existente | Editar draft, mover fase, publicar e ativar revisão | Histórico correto, revisão imutável, lifecycle das Tasks intocado |
| E2E-07 | Critério e evidência | Registrar evidência sem avaliar criterion | Criterion não muda para Met automaticamente |
| E2E-08 | Critério exigindo evidência | Avaliar criterion com evidência apropriada | Avaliação aponta revisão exata e evidências autorizadas |
| E2E-09 | Issue com Blocker | Resolver somente Issue | Blocker continua presente; não há liberação automática |
| E2E-10 | Issue com Blocker | Executar `Resolve issue and blockers` explicitamente | Resultado composto conforme contrato; histórico auditável |
| E2E-11 | Dois navegadores | Editar a mesma Task com versões distintas | Conflito, rascunho preservado; nova decisão do usuário exigida |
| E2E-12 | Rede com resposta incerta | Repetir tentativa com receipt idempotente | Uma mutação, um efeito, reconciliação de resposta |
| E2E-13 | Protocolo signed v2 | Abrir Task aguardando revisão | Estado correto, próximo passo indica profile, sem botão de assinatura Web |
| E2E-14 | Credencial revogada | Tentar busca, edição e replay | Negação de acesso; nenhum dado protegido recuperado |
| E2E-15 | Navegação rápida | Trocar Outcomes durante requests em andamento | Respostas antigas não contaminam Outcome selecionado |
| E2E-16 | Mobile 390px | Criar Outcome, Task e usar picker/diálogo | Sem overflow e com controles efetivamente acessíveis |
| E2E-17 | Zoom 200%/400% | Navegar, preencher e confirmar | Reflow, foco visível e campos utilizáveis |
| E2E-18 | Apenas teclado/leitor de tela | Abrir menu, pesquisar refs e confirmar ação | Sem keyboard traps, labels/ARIA válidos, anúncios de estado |
| E2E-19 | En-US padrão | Percorrer jornadas e fluxo de erro | `document.lang` correto, nenhuma string PT-BR oficial e datas en-US |
| E2E-20 | Regressão backend | Executar suíte Go e transportes em stores configurados | Guards/CAS/fencing/permissions e compatibilidade preservados |

**Dados de teste:** seeds determinísticas, pelo menos 3 perfis (`planner`, `executor`, `reviewer`) com permissões/protocolos apropriados; Outcomes em Draft/Active/Achieved; Tasks em múltiplos estados; Issues sem Blockers, Blockers com Issue resolvida, contratos ativos/expirados, revisão pendente e cursor inválido. A composição exata deve seguir o provisionamento real do WOS, não contas fictícias com permissões não suportadas.

## 22. Entregas pequenas, independentes e verificáveis

**Sequência sugerida de execução:**

```text
Baseline + Glossary + Action Inventory
       │
       ├──► English migration ──────┐
       │                            │
       ├──► Human Action Registry ──┤
       │                            ├──► Outcome/Objective/Task UX ──► Navigation
       └──► Authorized Ref Search ──┤              │                     │
                  │                 │              └──► Criteria/Evidence ├──► E2E Gate
                  └──► ReferencePicker             └──► Roadmap Editor ──┘
                                                              │
                                        Signed status/Review guidance + Admin
                                                              │
                                                 CI + distribution gate
```

**Lotes e Definition of Done específica:**

1. **UX-FND-01 — Foundation:** inventário completo de strings, comandos e invariantes; glossário aceito; protótipos/wireflows; sem implementação prematura de frameworks.
2. **UX-LOC-01 — en-US:** todas as superfícies Web e mensagens humanas em inglês, formatação de data, screenshots/testes e lint; strings de domínio nunca renomeadas.
3. **UX-ACT-01 — Action registry:** substituir global command picker; cobertura explícita dos 100 comandos; UI possui ações primárias por contexto.
4. **UX-API-01 — Reference search:** endpoint autorizado com cursor e paridade storage; testes escala/permissão/revogação.
5. **UX-CMP-01 — Reference picker:** combobox com seleção inequívoca, estados async/ARIA e preservação por ID.
6. **UX-FRM-01 — Outcomes:** criação, edição, Draft setup, status/activation; prevenção de transição implícita.
7. **UX-FRM-02 — Objectives:** criação, pai opcional, requisito, edição, critérios e detalhe contextual.
8. **UX-FRM-03 — Tasks:** form dedicado, `ExecutionSpec` tipado, vínculo, schedule, readiness e detalhe.
9. **UX-NAV-01 — Information architecture:** 6 áreas, filtros em vez de 16 seções e URLs navegáveis.
10. **UX-PLN-01 — Roadmap editor:** draft/revision/publish/activate, node identity, preview/diff, conflito.
11. **UX-EVD-01 — Verification:** evidence, links, criterion assessments, waivers e conclusões independentes.
12. **UX-ISS-01 — Issues:** cause/target/propagation, resoluções separadas/compostas, relações.
13. **UX-OPS-01 — Contracts:** legacy vs signed, execução/revisão, handoff `wosctl` e trust posture.
14. **UX-ADM-01 — Administration:** autorização/capabilities, forms tipados e separação das ações operacionais comuns.
15. **UX-CAS-01 — Conflict/retry:** preservação do rascunho, frozen intent, idempotência, diff e reconciliação.
16. **UX-QA-01 — Quality:** Playwright orientado à intenção, a11y, stress de pickers, protocolo e CI real.
17. **UX-REL-01 — Distribution:** `go:embed`, paths de assets, CSP, lockfiles, build reproduzível e artefatos Linux/Windows afetados.

**Trabalho paralelo admissível:** strings en-US, especificação das jornadas e contrato do picker podem ser desenvolvidos em paralelo após o baseline. O componente picker depende do contrato de busca. O editor de Roadmap depende de ações/formulários e da política de CAS. Não juntar refatoração de storage, troca de framework e alteração de domínio no mesmo PR funcional.

**Critério de encerramento do programa:** PRs aprovados e integrados, métricas comparadas com baseline, jornadas E2E e testes em stores suportados executados, documentação/capturas atualizadas e nenhuma regressão funcional conhecida da colaboração humano–agente.
