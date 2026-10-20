# WOS — Plano executivo de implementação: hierarquia, Issues, Blockers, dependências e Kanban contextual

> **Status:** plano aprovado; etapas 01–15 concluídas; etapa 16 é a próxima nesta branch/PR #116. Cada etapa exige seu próprio commit e aceite; o PR permanece Draft. Sem merge, release ou tag.
> **Baseline verificado:** `A1b3rt0M3rcad0/wos`, PR [#116](https://github.com/A1b3rt0M3rcad0/wos/pull/116), branch `fix/workspace-operational-density`, commit `d80d8329e097b7252b65b9c75f49e00d6cdcfa39`, Draft/Open, verificado em 2026-10-20.  
> **Execução autorizada:** branch `fix/workspace-operational-density`, PR Draft #116. **Um commit por etapa**, sem merge, release ou tag até aceite do conjunto.  
> **Nível de confiança:** alto para contratos inspecionados; médio para rotas HTTP, schemas e migrations não inspecionados integralmente. Tais pontos incluem uma etapa obrigatória de inspeção e confirmação antes de escrever código.

## 0. Resumo da decisão arquitetural (normativo)

1. Hierarquia estrutural única: `Outcome → Objective → [Subobjective | Task]`. Todo WorkItem/Task tem **um e somente um Objective direto**, no mesmo Outcome. O campo `scope.outcome_id` continua obrigatório para tenancy/consistência, sem significar vínculo estrutural direto. Subobjectives são Objectives com `parent_objective_id`, não outra classe.
2. Issue possui **exatamente um Objective proprietário** (`objective_id`) e zero ou mais `affected_refs` para seu alcance informativo. Essa propriedade é distinta de relacionamento causal, subobjetivo e dependência. Issues podem afetar Task, Objective, Outcome e demais tipos já aceitos pelo validador sem serem filhas dessas entidades.
3. `AchieveObjective` exige ausência de **Issues pendentes cuja propriedade esteja no Objective**. `open` e `investigating` são pendentes; `resolved` e `wont_fix` são terminais; `duplicate` só satisfaz o gate se o alvo canônico da cadeia estiver efetivamente fechado (`resolved` ou `wont_fix`), com cadeia válida/finita. `wont_fix` requer resolução fundamentada, mas **não comprova critérios**. Gate aplicado ao Objective proprietário; um ancestral só herda os gates de seus Subobjectives quando sua política estrutural de conclusão exige fechar esses Subobjectives. Não impor bloqueio indiscriminado à árvore.
4. Task não tem seu fechamento proibido **apenas** por Issue aberta. Critérios da própria Task, Blockers diretamente aplicáveis/herdados, dependências hard, autoridades e verificações vigentes continuam obrigatórios. `Issue` isolada não produz Blocker.
5. Blocker mantém semântica atual: um `blocked_ref` (Outcome/Objective/WorkItem), causa `cause_ref` ou externa/descritiva e `propagation=direct|subtree` conforme o alvo. Blocker de Task não propaga `is_blocked` por cadeias de dependência; Tasks dependentes hard recebem `hard_dependency_unsatisfied` até pré-requisito concluído. Um Blocker de Objective com subtree propaga somente pela hierarquia estrutural.
6. Dependência `Relation(depends_on)` permanece separada de associação/ownership. `source_ref` depende de `target_ref`; `hard` influencia readiness/gates, `advisory` não. DAG sem ciclos, mesma scope, sem hard dependency da Task em seu próprio Objective; alterações de dependências saindo de Task somente em Backlog (ADR-041). A interface explicará `Depende de` (outgoing) e `Dependem desta` (incoming).
7. Issue ganha `assignee_ref?` **de um responsável único**, distinto de `reported_by`. Atribuição não é lease: atomicidade por expected_version, controle de identidade/autorização, eventos e histórico. Nenhum claim/fencing novo nesta onda; quem assume a Issue não ganha autoridade de execução sobre Tasks.
8. Kanban Issues reutiliza modal, navegação contextual, lanes, inspector, comandos canônicos e linguagem visual da branch. Colunas de apresentação `Open`, `Investigating`, `Closed`; na última, três resultados finais distintos (`resolved`, `wont_fix`, `duplicate`). Drag prepara operação e confirmação; ações do inspector/teclado garantem paridade. Nada de `status setter`, dupla modal ou assinatura feita pelo navegador.
9. Criação de Issue com múltiplos Blockers em uma operação atômica é proposta como extensão pontual do comando existente (`ReportIssueWithBlocker` permanece por compatibilidade). A criação exige avaliação explícita de cada alvo/propagação; resolver Issue não resolve Blockers automaticamente.
10. Onboarding atual, asset pipeline, login, identidade editorial, animações, post-its, navegação existente, autenticação, recuperação de comando incerto, idempotência, permissões e contratos de agente existentes **permanecem intocados salvo integração restrita aos fluxos tratados aqui**.

## 1. Evidência do estado atual — fatos e lacunas

| Área | Confirmado na branch do PR #116 | Lacuna para o comportamento-alvo |
|---|---|---|
| WorkItem | `packages/wos-core/domain/work_item.go`: `ObjectiveID *ID`; `application/planning_creation.go`: pai validado apenas se não-nulo; `application/work_editing.go`: `validateWorkObjectiveChange(nil)` retorna sucesso | Task órfã permitida em criação, batch e movimentação |
| Objective | `domain/objective.go`: `ParentObjectiveID *ID`, `RequiredForOutcome`, lifecycle planned/in_progress/achieved/cancelled | Não há gate de Issues próprias |
| Issue | `domain/issue.go`: scope, title, severity, lifecycle open/investigating/resolved/wont_fix/duplicate, `AffectedRefs`, `ReportedBy`, resolução e duplicata | Não há `objective_id`, `assignee_ref` nem responsabilidade explícita |
| Blocker | `domain/blocker.go`: alvo, causa, propagação, conclusão independente; `application/issues_blockers.go:projectBlockingState` acompanha pais estruturais | Exibição do impacto transitivo de dependências é insuficiente; não confundir com herança de Blocker |
| Issues compostas | `application/issues_blockers_compound.go`: criação Issue + **um** Blocker e resolução Issue + subset explícito de Blockers | Não há criação em lote de impactos validada como ato único |
| Dependências | `domain/relation.go`, `application/dependencies.go`: DAG, same-scope, hard/advisory; `work_contract_guards.go` aplica Backlog planning guard | UX genérica de referência e remoção por relation, sem bulk editor ou impacto transitivo navegável |
| Gate de objetivo | `application/workspace_preconditions.go:checkObjectiveAchievement`: review, Blocker, hard dependency, critérios/prova, strict work | Não consulta Issues; precisa validar no backend de comando e capability/preview |
| UI | `web/assets/workspace-kanban.js` modal contextual, listas `objectives`/`tasks`, drag preparado; `human-forms.js` formulários; `reference-picker.js` pesquisa escopada | Sem lane `issues`; Outcome ainda mostra `Direct Tasks`; opções genéricas para vínculo |
| Persistência | `storage/postgres` e `storage/sqlite`, migrações numeradas; 0033 completion_policy em ambos | Evolução de issues e work_items exigirá migração aditiva + backfill resolvido antes de NOT NULL |
| Contratos | `packages/wos-api/{commands,openapi.yaml,http,mcp}`, `packages/wos-sdk-go`, `packages/wos-cli`; generators existentes `tools/{openapigen,mcpgen,postgresgen}` | Novos comandos/queries precisam paridade HTTP/MCP/CLI/SDK, idempotência e schema atualizado |
| Documentação | `docs/workspace-relationships.md` e documento canônico permitem WorkItem diretamente no Outcome; `docs/adr/0041...` fixa Backlog-only | Atualizar ADR e documentação para remover ambiguidade, preservar política já aceita |

**Arquivos efetivamente inspecionados, entre outros:** `domain/{work_item,objective,issue,blocker,relation,readiness,workspace_eligibility}.go`, `application/{planning_creation,work_editing,dependencies,issues_blockers,issues_blockers_compound,workspace_preconditions,completion_policy,workspace_conclusion,workspace_capabilities,workspace_placement}.go`, `web/assets/{workspace-kanban,human-forms,workspace-form-fields,reference-picker,workspace-plan-editor,canvas-data,workspace-canvas,workspace-view,workspace-entity-context,app}.js`, `docs/{workspace-relationships.md,WOS_Design_Arquitetura_Planejamento_Atualizado.md}`, ADR-041 e migrations 0033. A branch PR #116 foi verificada localmente em `d80d8329e097b7252b65b9c75f49e00d6cdcfa39`. A etapa 01 registra as execuções de baseline desta implementação; resultados anteriores são referenciados apenas como evidência histórica.

### 1.1. Fluxos existentes a preservar e estender

- `CreateObjective` e `CreatePlanItems` criam nós hierárquicos; `CreateWorkItem` cria WorkItem em Backlog e admite initial criteria atômicos. **Extensão:** Objective não-nulo em cada WorkItem e batch.
- `SetObjectiveParent`/`SetWorkItemObjective` têm prévia estrutural, `expected_outcome_revision`, anti-cycle e guard de execução. **Extensão:** impedir destino Outcome/null para Task, preservar recurso null para Objective de topo.
- `AddDependency`/`RemoveDependency` alteram relations tipadas, auditadas. **Extensão:** preparar relações com preview e commit atômico em lote sem repetir a lógica de DAG e contrato.
- `CreateIssue`, `InvestigateIssue`, `ResolveIssue`, `MarkIssueWontFix`, `MarkIssueDuplicate`, `ReopenIssue` são comandos de domínio. **Extensão:** propriedade por Objective, atribuição e gate.
- `CreateBlocker`, `ResolveBlocker`, `CancelBlocker`, `ReportIssueWithBlocker`, `ResolveIssueAndBlockers` já distinguem causa e impacto. **Extensão:** seleção múltipla de impactos com validação total.
- Kanban modal: `createContextualKanban`, `makeFrame`, `load`, `render`, `move`, `showPreparation`, `renderProofChecklist`. **Extensão:** modo `issues` e transições ligadas às capacidades reais, sem copiar framework de modal.

## 2. Onboarding e identidade — invariantes absolutas

A branch contém a UI embedded em `packages/wos-api/web/assets/`; o onboarding/login depende de rotas e estados da aplicação. **Não presumo o caminho de um módulo de onboarding separado sem inspeção específica.** A etapa 01 deve capturar fluxo real, DOM, screenshots, autenticação, formulários, mensagens e estados antes de mudar modelos.

Preservar: primeiro acesso, criação/seleção de namespace e Outcome, login/sessão/permissões, responsividade, teclado/foco, feedback, assets visuais, tipografia, cores, layouts, fluxo de criar primeiro Objective/Task, recuperação pós-reload, estados vazios, mobile, viewport HD 1280×720 e zoom de navegador. Quando fluxo guiado tentar criar Task diretamente sob Outcome, **ajustar só o passo de seleção/criação do Objective obrigatório**, sem redesenhar o restante: a Task só é enviada após Objective válido (um fluxo atômico de `CreatePlanItems` pode incluir Objective+Task). Não criar Objective automático oculto.

Baseline obrigatório: screenshots e teste de jornada de primeiro acesso em desktop/mobile + relatório de comparação sem diferenças não justificadas, e testes de autenticação/permissão. Nenhum ganho de UX justifica quebrar onboarding. Identidade visual deve seguir CSS/tokens existentes (`editorial.css`, `workspace-*.css`), sem framework frontend novo.

## 3. Modelo operacional e invariantes

### 3.1 Tipos existentes versus alterações propostas

**Existente:** `WorkItem.ObjectiveID *ID`, `Objective.ParentObjectiveID *ID`, `Issue.AffectedRefs []EntityRef`, `Issue.ReportedBy ActorRef`, `Blocker.BlockedRef`, `Blocker.CauseRef`, `Relation.SourceRef/TargetRef`, `ReadinessReasonHardDependencyUnsatisfied`. **Proposto:** `WorkItem.ObjectiveID` permanece representado como ponteiro onde migração/compatibilidade exigir, mas **nunca é nil em estado válido novo**; `Issue.ObjectiveID ID` e `Issue.AssigneeRef *ActorRef`. Não renomear WorkItem para Task na API, apenas na apresentação.

**Fonte de verdade:** ownership em `Objective.ParentObjectiveID`, `WorkItem.ObjectiveID`, `Issue.ObjectiveID`; dependências em `Relation`; bloqueio em `Blocker`; resolução em `Issue.Lifecycle`; atribuição em `Issue.AssigneeRef`. Não duplicar edges em uma relation paralela de propriedade.

### 3.2 Matriz exata de conclusão

| Sujeito | Issue pendente no Objective | Blocker aplicável | Dependência hard não satisfeita | Critérios próprios | Regras adicionais |
|---|---|---|---|---|---|
| Task | não é gate isolado | bloqueia onde o gate atual o exige | bloqueia readiness/completion | valida independentemente | authority, lease, independent review, policy existente |
| Objective | **impede `AchieveObjective`** | impede `AchieveObjective` | impede | obrigatório | strict_work do Outcome + obrigações de filhos conforme contrato |
| Outcome | não consulta Issue diretamente como segunda fonte | impede onde aplicável | não criar dependência nova | obrigatório | required objectives e políticas atuais; Issues abertas bloqueiam via seus Objectives obrigatórios, se aplicável |

Issue terminal `wont_fix` é aceita apenas com `resolution_summary` obrigatório; issue `duplicate` segue sua canônica no gate e impede fechamento prematuro. Ao reabrir Issue, o Objective não volta automaticamente de achieved para planned; em Objective já concluído, reabertura exige política explícita: **bloquear `ReopenIssue` enquanto o Objective proprietário estiver achieved, até reabrir o Objective**, ou retornar erro orientado a reabertura. Escolha adotada para evitar conclusão persistida inconsistente. Registro documental histórico pode continuar conforme regras de Outcome terminal, sem mutação operacional incompatível.

### 3.3 Regras de vinculação, resolução e responsabilidade

- Issue sempre pertence a um Objective; em criação a partir de Task, pré-selecionar seu `WorkItem.ObjectiveID`, nunca adivinhar a partir de título. Em criação a partir de Outcome, exigir escolha de Objective antes de submit. `affected_refs` não substitui owner.
- Issue pode ter `affected_refs` múltiplos; selecionar Objective proprietário não adiciona automaticamente uma segunda referência afetada à lista.
- Criar Issue com Blockers: `Issue.ObjectiveID` pertence à mesma scope; cada Blocker aponta causa Issue; um Blocker por alvo/propagação explicitamente confirmado, evitar duplicata ativa; comando falha integralmente em qualquer alvo inválido; revisão do Outcome opcional/obrigatória conforme impacto preview aceito (nesta onda obrigatório para operação múltipla).
- Issue em `open` pode permanecer sem responsável; `StartIssueInvestigation` **novo comando** combina `Open→Investigating` e atribuição ao principal do comando atomicamente. Pode-se alternativamente investigar com atribuição explícita prévia, mas não oferecer ação visual duplicada. `AssignIssue` permite designar/repassar sem alterar lifecycle; `UnassignIssue` exige justificativa quando investigating.
- Encerramento por `ResolveIssue`, `MarkIssueWontFix` e `MarkIssueDuplicate` exige usuário/autorização e `expected_version`; no Kanban, `Closed` abre chooser de resultado. `duplicate_of_issue_id` da mesma scope sem ciclos; fechamento de duplicata não deve ocultar Issue canônica aberta do gate.
- `AssigneeRef` é responsável declarado, **não** concessão implícita para concluir Tasks, adquirir leases ou avaliar critérios. Fonte `ReportedBy` continua imutável.
- `Issue` não é pré-requisito em `Relation(depends_on)`; trabalho corretivo é Task normal associada ao Objective, e vínculo informativo a Issue usa mecanismo documental/contextual tipado existente onde houver suporte; se faltar Relation apropriada, introduzir **`related_work_refs` opcional no Issue** (proposta desta especificação, com integridade same-scope, sem alterar readiness), em vez de sobrecarregar `affected_refs` como vínculo de solução. Preferir primeiro avaliar os `relates_to` existentes e não duplicar fonte de verdade: esta especificação **adota `relates_to` existing** para Task corretiva ↔ Issue, desde que registry aceite endpoint Issue; se a operação de relação pública não suportar escrita ainda, acrescentar comando tipado mínimo que persista Relation, não coluna paralela.

### 3.4 Dependência e alcance transitivo

- Relação `source → target` = Source **depende de** Target. Incoming a uma Task = quem depende dela. Nenhum ciclo em hard **ou advisory** (DAG atual). Não permitir self-dependency nem hard Task→seu Objective (incluindo proteção após move).
- Bloqueio direto `Blocker(Task A)` afeta apenas Task A. Se Task B hard-depends A, B fica `waiting_dependencies` quando A não está done. Se Task C hard-depends B, C aguarda B; não materializar novos Blockers.
- Preview transitivo deve devolver três classes: `direct_blocked`, `hard_waiting`, `advisory_related`; distinguir **efeito atual** de **impacto potencial** (ex.: B já done não perde retrospectivamente conclusão). Exibir caminho e raiz, caminho limitado e paginação; estados calculados no snapshot coeso, não por traversal de cards carregados.
- Incidência dinâmica: se Blocker de A é resolvido, B só fica pronta quando A realmente conclui; resolver Issue não conclui Task. `advisory` nunca bloqueia.
- Qualquer edição de outgoing dependency de Task requer Backlog. Operar se Task em To do via `DeferWorkItem` explicitamente e confirmado; nunca tentar defer automaticamente. Incoming dependency pertence ao outro Source e obedece guard desse Source.

### 3.5 State machine proposta para Kanban Issues

| Coluna | Lifecycle canônico | DnD autorizado | Tratamento |
|---|---|---|---|
| Open | `open` | Open→Investigating via `StartIssueInvestigation` | responsabilização e autorização explícita |
| Investigating | `investigating` | Investigating→Closed via chooser | resolver, won't-fix ou duplicate com motivo; Blockers exigem revisão separada |
| Closed | `resolved`, `wont_fix`, `duplicate` | Closed→Open via `ReopenIssue` autorizado | reabrir exige Objective proprietário não-achieved; nunca resolve Blockers |

Entrada direta `open→resolved/wont_fix/duplicate` pode permanecer disponível por comandos existentes quando regras permitem (report and immediately triage); UI oferece operação no inspector sem lane obrigatória artificial. `investigating→open` não existe atualmente: não inventar status setter; liberar responsável não retorna lifecycle ao estado anterior. `Closed` é projeção e não persistência de um novo lifecycle.

## 4. Backend: contratos, camadas e persistência

### 4.1 Domínio (`packages/wos-core/domain`)

Modificar `work_item.go` (invariante owner), `issue.go` (ObjectiveID, assignee, validação, `Assign`, `Unassign`, `StartInvestigationWithAssignee`), `blocker.go` somente se necessário para identidade de deduplicação de impactos, `readiness.go` **sem** adicionar Issue como motivo de readiness de Task; `relation.go` para relação informativa Issue↔Task apenas se registry atualmente rejeitar `relates_to` nos endpoints. `objective.go` permanece sem arrays derivados de Issues; gate vive na Application com consulta coerente. Adicionar `issue_workflow_test.go`, `issue_owner_test.go`, `objective_issue_gate_test.go` onde couber — **nomes propostos, não existentes**.

### 4.2 Application (`packages/wos-core/application`)

`planning_creation.go`: validar batch com todo WorkItem tendo parent por ID/local key; `insertWorkItemCreation` nunca aceita nil. `work_editing.go`: `validateWorkObjectiveChange` rejeita nil; `workspace_placement.go`: preview `parent_id:null` para Task deve devolver `blocked` com reason code estável, não preview 'allowed'. `work_contract_guards.go`: manter planning/lease/read guards. `issues_blockers.go`: `CreateIssueCommand.ObjectiveID`; validação de owner não terminal; `AssignIssueCommand`, `UnassignIssueCommand`, `StartIssueInvestigationCommand`; idempotent same-state behavior se necessário; bloquear reabertura sob Objective achieved. `issues_blockers_compound.go`: **novo** `ReportIssueWithBlockersCommand` com itens de target/propagation/description, `expected_outcome_revision`, limite configurado dentro envelope atual 256KiB; única UnitOfWork.

`workspace_preconditions.go`: adicionar `requireObjectiveIssuesClosed(ctx, uow, objective.Ref())` antes de `objective.Achieve`; incluindo duplicate-chain canonical resolution. `workspace_capabilities.go`: repetir **exatamente** o mesmo predicate/condition ao preparar Achieve; não fazer check distinto no frontend. `workspace_conclusion.go`: acrescentar contagem/refs de Issues pendentes no preview atual. `workspace_read_types.go`, `workspace_cards.go`, `workspace_frame.go` e `ports/workspace_read.go`: coleções filtradas por Objective e aggregation Issue lanes, resposta coerente revision/cursor. `dependencies.go`: reutilizar `ValidateDependencyAcyclic`; acrescentar `PreviewDependencyChanges` e `ApplyDependencyChanges` **novos** (proposta) com sequência atômica, expected versions/revision, prevenção de ciclos no grafo resultante, guard por Source e permissão, deduplicação e planejamento. Novo arquivo **proposto** `application/dependency_impact.go` para consulta de upstream/downstream transitiva e causas, bounded/coverage, sem persistir nós derivados. Novo `application/issue_assignment.go` caso separar responsabilidades seja coerente com tamanho atual.

### 4.3 Storage (`packages/wos-core/storage`)

Criar migrations pares **propostas** `postgres/migrations/0034_issue_objective_assignment.sql` e `sqlite/migrations/0034_issue_objective_assignment.sql` apenas após verificar HEAD efetivo para evitar conflito de número; 0034 é reserva inicial, não número garantido. Campos `issues.objective_id`, `issues.assignee_ref` (seguir serialização existente de ActorRef), e constraint/index que viabilizem `ListPendingIssuesByObjective`; não gravar `status=closed` nem `is_blocked`. `work_items.objective_id NOT NULL` somente **após** backfill determinístico e validação; SQLite poderá exigir rebuild de tabela com cópia, FK/index/triggers preservados, em migração posterior **proposta** `0035_required_work_item_objective.sql` para ambos os adapters. Não presumir nome de tabela/index sem inspecionar migrations base: etapa de schema discovery identifica nomes reais e ajusta DDL antes de commit.

Estratégia de compatibilidade *expand → backfill → contract*: (a) adicionar coluna nullable Issue, novos leitores tolerantes, índices; (b) rodar auditoria SQL de Issues sem owner e Tasks sem Objective, mapa de ownership manual/importação determinística auditada; não criar Objectives artificiais; (c) preencher mapeamentos dentro de transações; (d) bloquear gravações novas inválidas no domínio imediatamente, estabilizar dados; (e) enrijecer NOT NULL/FK/constraints, preservar backup/restore, checksums de migrations, suporte SQLite/Postgres; (f) revalidar eventos/ETags/cursor. Inexistência de dados órfãos é precondição explícita do contract; se não for possível mapear, a migração para NOT NULL **não roda** e a release permanece bloqueada — sem fallback silencioso.

`memory/adapter.go`, `storage/{sqlite,postgres}/{repositories.go,workspace_read.go,workspace_placement.go}`: ler/escrever propriedade, responsável, Issue views, batch e contagens. Actual SQL query files são definidos pela inspeção de adapter antes de implementação.

### 4.4 Interfaces e transports

`packages/wos-api/commands/{contracts.go,workspace_schema.go,workspace_results.go,catalog_generated.go,commands.openapi.json}`, `packages/wos-api/{http/workspace.go,http/handler.go,mcp/workspace.go,mcp/commands_generated.go,openapi.yaml}`, `packages/wos-sdk-go/{client.go,workspace.go,commands_generated.go}`, `packages/wos-cli/{workspace_commands_v2.go,workspace_inspect.go,workspace_planning.go}`. Gerar mirrors usando generators já existentes (`tools/openapigen`, `tools/mcpgen`), jamais editar saída gerada manualmente. Comandos propostos `AssignIssue`, `UnassignIssue`, `StartIssueInvestigation`, `ReportIssueWithBlockers`, `PreviewDependencyChanges`, `ApplyDependencyChanges`, e query `ListObjectiveIssues` (ou extensão da workspace read existente). Campos `objective_id`, `assignee_ref` no DTO; preservar names `work_item` e version/idempotency. O `CreateIssue` legado passa a exigir `objective_id`: quebra semântica de payload antigo, por isso estratégia de compatibilidade precisa ser explícita; **decisão:** não inferir proprietário a partir de `affected_refs` em writes novos. Rejeitar payload ausente com erro estruturado de migração e exemplo de reparo; na janela expand leitores antigos continuam, mas emitidores são atualizados no mesmo rollout. Sem compatibilidade semântica silenciosa que atribua dono errado.

### 4.5 Operações e resposta de erro

- `objective_issue_pending`: `ErrorCodePreconditionFailed`, `SubjectRef=Objective`, `RelatedRefs` limitada + count + coverage; `Remedies` aponta investigação/inspeção. Resultado não altera versões/eventos/Outcome revision.
- `task_objective_required`: `ErrorCodeInvalidArgument`/`PreconditionFailed` conforme contrato atual, `field_paths=["objective_id"]`. Mesma regra em create, batch, reparent, preview, SDK e CLI.
- `issue_owner_invalid`: Objective inexistente, terminal, fora de scope; criação/transferência não permitida sem validação. **Nesta onda Issue ownership é imutável depois da criação**; realocação exige futuro comando explícito com impact preview — não adicionar setter genérico.
- `issue_duplicate_pending`: aponta canônica ainda aberta. Cadeia com self/cycle/ID ausente ou fora de scope rejeitada conservadoramente.
- `dependency_cycle`, `own_objective_hard_dependency`, `planning_edit_frozen`, `stale_outcome_revision`, `assignee_version_conflict` preservam padrões atuais de erro/versioning; nomes novos são propostos e devem entrar no catálogo de reason codes.
- `IssueAssignee` deve ser identidade validada por autorização existente; permitir atribuição a pessoa/agente referenciado no namespace, não texto livre.
- Operações compostas registram eventos de domínio por entidade alterada **com uma única transação e outcome revision**; falha/cancelamento não deixa meia operação.

## 5. Frontend / UX/UI proposto (sem reescrever a UI)

### 5.1 Navegação

Modificar `packages/wos-api/web/assets/workspace-kanban.js` para `frame.content ∈ {objectives,tasks,issues}` quando `frame.view.subject.ref.kind=objective`; Outcome oferece `objectives` e visão agregada somente-leitura `all_tasks` se necessário — **sem Direct Tasks**. Em cada Objective o seletor de coleção oferece `Subobjectives`, `Tasks`, `Issues`; ação principal contextual `New subobjective`, `New Task`, `Report Issue`. Preservar modal nativo único, camera, filtros, seleção, back, foco, escopo, recuperação após operação.

Criar **somente se reutilização não for suficiente** `web/assets/workspace-issue-kanban.js` com funções `issueLane`, `prepareIssueMove`, `renderIssueCard`; preferir manter renderer em `workspace-kanban.js` com pequenos helpers externos em `kanban-operations.js` (arquivo existente). `workspace-kanban.css`/`workspace-surfaces.css`: apenas modificadores de componentes; não duplicar estilos nem tokens. `human-forms.js`: editor Issue com proprietário fixado no Objective contextual, Severity, Description, assignee, afectados e toggle explicito de impacto; chooser fechamento com razões e referências à Issue canônica; `ResolveIssueAndBlockers` conserva confirmação individual.

### 5.2 Cartões e inspeção

Issue card deve mostrar título truncado com acesso ao título integral, severidade, assignee ou 'Unassigned', contador de Blockers ativos e nota de `affected_refs`, sem confundir reporting actor com owner. Inspector: Summary, Investigation & Responsibility, Affected items / Blockers, linked corrective Tasks, evidence & notes, history; comandos contextuais. Closed lane identifica `Resolved/Won't fix/Duplicate` separadamente. Filtros combináveis: estado, severity, assignee, com/sem Blocker, busca; contagens autoritativas ou rotuladas como carregadas, não total inventado.

### 5.3 Dependências no Canvas e Inspector

`workspace-view.js`/`workspace-entity-context.js`: substituir lista plana por `Depends on` e `Required by`, mostrando relação hard/advisory, lifecycle do alvo, fonte e mecanismo de remover. Comando “Add prerequisite” começa na Source; “Add dependent” cria **relação cujo source é o selecionado dependente externo e target é o item atual**. Gate de planejamento pertence ao Source real; ao modificar incoming de Task em execução, bloquear operação com orientação de Defer no próprio Source (se aplicável). Picker reaproveita `reference-picker.js` mas adiciona filtro contextual, exclusão da própria entidade e notas de elegibilidade sem alterar semântica do picker global.

`workspace-canvas.js`: modo explícito `Connect dependency` com dois cliques ou teclado; preview da seta com legenda e orientação `source depends on target`, escolha hard/advisory, confirmação de comando. Grafo de dependencies oculto/atenuado por padrão, visível mediante seleção ou filtro, nunca confundir com estrutura/layout (`layoutCanvas` mantém só parentKey). Inspector exibe preview de impacto transito de uma Issue/Blocker e seus nós dependentes como **Waiting (hard)**, não com badge **Blocked** duplicado. Graph navigation deve carregar nós ausentes da página atual respeitando paginação/revision.

`workspace-plan-editor.js`: batch editor só disponibiliza `Outcome` como parent para Objective, nunca Task; se alternar tipo Objective→Task e parent virar inválido, exigir correção manual em vez de transformar em órfã. `workspace-form-fields.js:placementField`: exigir Objective quando criando/alterando Task, manter `Outcome` permitido para Objective.

### 5.4 UI/UX e acessibilidade

Drag só prepara ação, realiza fresh capabilities, explicit confirmation; seleção de status “Closed” oferece escolhas, não mutação direta. Keyboard/Touch possuem botões de ação contextual equivalentes sem menu de status nos cards. Nenhum modal empilhado, ESC gerenciado, focus-restoration, scroll/camera preservados. View HD/mobile com inspector acoplado ou painel já existente, sem barras paralelas. Relações coloridas nunca dependem só de cor; badges acessíveis; warning de impacto com texto claro. Não modificar ilustrações, assets/editorial typography e tokens de marca.

## 6. Índice de arquivos por camada (existentes E / novos P)

- **E domínio:** `packages/wos-core/domain/{work_item.go,objective.go,issue.go,blocker.go,relation.go,readiness.go,workspace_eligibility.go}`.
- **E aplicação:** `packages/wos-core/application/{planning_creation.go,work_editing.go,dependencies.go,issues_blockers.go,issues_blockers_compound.go,workspace_preconditions.go,workspace_capabilities.go,workspace_conclusion.go,workspace_placement.go,workspace_read_types.go,workspace_cards.go,workspace_frame.go,completion_policy.go,work_contract_guards.go}`.
- **P aplicação:** `packages/wos-core/application/{dependency_impact.go,issue_assignment.go,issue_completion.go}` (criar somente quando métodos não couberem naturalmente em arquivos existentes).
- **E ports/adapters:** `packages/wos-core/ports/workspace_read.go`; `packages/wos-core/storage/{memory/adapter.go,sqlite/{repositories.go,workspace_read.go,workspace_placement.go},postgres/{repositories.go,workspace_read.go,workspace_placement.go}}`.
- **P migrations:** `packages/wos-core/storage/{postgres,sqlite}/migrations/0034_issue_objective_assignment.sql` e `0035_required_work_item_objective.sql` (número final confirmado no primeiro commit técnico); dados backfill em script versionado **P** `tools/acceptance/issue_ownership_backfill.py` ou Go equivalente, dry-run por default.
- **E API:** `packages/wos-api/{commands/{contracts.go,workspace_schema.go,workspace_results.go,catalog_generated.go,commands.openapi.json},http/{workspace.go,handler.go},mcp/{workspace.go,commands_generated.go},openapi.yaml}`.
- **E clientes:** `packages/wos-sdk-go/{client.go,workspace.go,commands_generated.go}` e `packages/wos-cli/{workspace_commands_v2.go,workspace_planning.go,workspace_inspect.go}`.
- **E UI:** `packages/wos-api/web/assets/{app.js,workspace-kanban.js,kanban-operations.js,human-forms.js,workspace-form-fields.js,workspace-plan-editor.js,reference-picker.js,workspace-canvas.js,workspace-view.js,workspace-entity-context.js,canvas-data.js,workspace-kanban.css,workspace-surfaces.css,editorial.css,en-US.js}`.
- **P testes:** `packages/wos-core/{domain/issue_owner_test.go,domain/issue_assignment_test.go,application/issue_completion_test.go,application/issue_assignment_test.go,application/dependency_impact_test.go}`, `tests/web/{issue-kanban.spec.mjs,issue-objective-gate.spec.mjs,dependency-connection.spec.mjs,onboarding-regression.spec.mjs}`; estender os testes já existentes antes de criar duplicatas.
- **E testes existentes:** `tests/web/{contextual-kanban.spec.mjs,kanban-simplification.spec.mjs,kanban-operations.spec.mjs,progressive-entry.spec.mjs,reference-picker.spec.mjs,structural-move.spec.mjs,structural-preview.spec.mjs,journey.spec.mjs,accessibility.spec.mjs}`, `packages/wos-core/application/{dependencies_test.go,planning_creation_test.go,workspace_placement_test.go}`, adapters `{sqlite,postgres}/*_test.go`.
- **E docs:** `docs/{workspace-relationships.md,WOS_Design_Arquitetura_Planejamento_Atualizado.md,ux/operational-kanban/README.md,adr/README.md}`; **P ADR:** `docs/adr/0042-objective-owned-issues-and-mandatory-work-placement.md` (verificar ordinal livre); **Plano aprovado:** `docs/ux/operational-kanban/issue-and-dependency-plan.md` (esta cópia).

## 7. Protocolo por commit

Cada etapa é **um único commit**, contendo alteração coesa + testes da mudança + documentação afetada. Mensagens no padrão `type(scope): ...`. Nunca agrupar etapas num commit nem fazer commit sem aceitação parcial. Dependências no quadro seguinte são sequenciais e adicionais às referências cruzadas. Para qualquer etapa: (i) HEAD e PR Draft antes/depois, (ii) diff limitado ao escopo, (iii) unit/integration afetados passam, (iv) migração e mirrors gerados limpos quando aplicáveis, (v) logs/screenshots de validação versionados apenas se padrão do PR exigir, (vi) sem regressão do onboarding. Reverter etapa isolada é possível sem alteração de histórico de etapas anteriores.

## 8. Sequência de implementação — 44 commits

### Etapa 01 — Baseline, auditoria de onboarding e invariantes

**Commit sugerido:** `docs(wos): etapa 01 baseline, auditoria de onboarding e invariantes`.  
**Dependência:** baseline PR #116.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `docs/ux/operational-kanban/*; tests/web/{journey,progressive-entry,accessibility}.spec.mjs; app.js`.

**Trabalho:** Capturar SHA, matrizes UX desktop/mobile, primeiro acesso, estado de login/namespace, testes de autenticação e permissões; inventariar todos os fluxos que criam Tasks/Issues.

**Aceite verificável:** Baseline de onboarding anexado, testes existentes reproduzidos ou falhas ambientais classificadas; sem alteração de comportamento.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 02 — ADR normativo e mapa de contratos

**Commit sugerido:** `docs(wos): etapa 02 adr normativo e mapa de contratos`.  
**Dependência:** etapa 01 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `docs/adr/0042-*.md [P]; docs/adr/README.md; docs/workspace-relationships.md`.

**Trabalho:** Registrar decisões deste plano, owner Issue, gate e duplicatas, proibição Task órfã, propagação Blocker vs depends_on, compatibilidade expand/backfill/contract.

**Aceite verificável:** ADR aprovado e casos de verdade listados; nenhuma regra ambígua.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 03 — Inventário de schema e geradores

**Commit sugerido:** `docs(wos): etapa 03 inventário de schema e geradores`.  
**Dependência:** etapa 02 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `storage/{sqlite,postgres}/migrations/*; tools/{postgresgen,openapigen,mcpgen}; wos-api/commands/*`.

**Trabalho:** Inspecionar DDL, FKs, índices, modelos SQL, fixtures, transitórios de API/CLI/MCP; validar ordinal da próxima migration.

**Aceite verificável:** Planilha de DDL real, contratos/generator sources registrados; confirmar nome real de tabela e versão de migração.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 04 — Auditoria de dados órfãos em dry-run

**Commit sugerido:** `test(wos): etapa 04 auditoria de dados órfãos em dry-run`.  
**Dependência:** etapa 03 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `tools/acceptance/issue_ownership_backfill.py [P]; storage adapters`.

**Trabalho:** Desenvolver relatório somente-leitura por Outcome: Tasks objetivo null, Issues sem owner, candidatos existentes e dependências que seriam afetadas.

**Aceite verificável:** Relatório determinístico por escopo e contagem de órfãos sem alterar dados; testes de amostra.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 05 — Invariante domínio Task com Objective

**Commit sugerido:** `feat(wos): etapa 05 invariante domínio task com objective`.  
**Dependência:** etapa 04 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `domain/work_item.go; domain/work_editing_test.go`.

**Trabalho:** Adicionar validação de ownership não-nulo para agregados novos, com caminho de hidratação legado explicitamente transitório durante expand; testes de nil/UUID inválido.

**Aceite verificável:** Task nova sem Objective falha; carga de registro legado continua possível apenas na fase prevista.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 06 — Guard de criação individual

**Commit sugerido:** `feat(wos): etapa 06 guard de criação individual`.  
**Dependência:** etapa 05 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `application/planning_creation.go; application/service.go; planning_creation_test.go`.

**Trabalho:** Rejeitar CreateWorkItem sem Objective; validar owner e escopo, independentemente do transport.

**Aceite verificável:** CreateWorkItem nil falha antes de persistir; versão/revision/eventos invariantes.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 07 — Guard de batch e editor de plano

**Commit sugerido:** `feat(wos): etapa 07 guard de batch e editor de plano`.  
**Dependência:** etapa 06 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `application/planning_creation.go; web/assets/workspace-plan-editor.js`.

**Trabalho:** Validar parent local/existente para toda WorkItem; seletor Outcome exclusivo de Objectives; batch atômico com Objective+Task permitido.

**Aceite verificável:** Todo plano inválido falha atomicamente e UX permite corrigir pai sem perda de conteúdo.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 08 — Guard de reparent e preview

**Commit sugerido:** `feat(wos): etapa 08 guard de reparent e preview`.  
**Dependência:** etapa 07 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `application/work_editing.go; workspace_placement.go; domain/work_item.go`.

**Trabalho:** Rejeitar destino null em SetWorkItemObjective; retornar preview bloqueado; proteger critérios, dependências e estado operacional.

**Aceite verificável:** Preview e comando em paridade; history e versões preservados na rejeição.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 09 — Projeções e navegação sem Direct Tasks

**Commit sugerido:** `feat(wos): etapa 09 projeções e navegação sem direct tasks`.  
**Dependência:** etapa 08 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `web/assets/{canvas-data,workspace-entity-context,workspace-kanban}.js; tests/web/contextual-kanban.spec.mjs`.

**Trabalho:** Retirar seleção Direct Tasks; manter visão agregada sem ownership falso e preservar seleção/camera.

**Aceite verificável:** Task é mostrada sob Objective; Outcome não cria nem sugere Task direta.

**Execução em 2026-10-20:** concluída. O Outcome Kanban contém Objectives e não oferece criação de Tasks órfãs; o Objective Kanban permite alternar entre Subobjectives e Tasks. Registros Task legados sem pai continuam inspecionáveis sem receber uma aresta de ownership inventada. A troca de coleção e a atualização preservam seleção, contexto e câmera. As mudanças de lane seguem exclusivamente por drag-and-drop; os selects permanecem para contexto e filtros.

**Verificação:** `npm run test:unit` (79/79); `contextual-kanban.spec.mjs` (18/18 em HD, Full HD, QHD, 4K, mobile e zoom de navegador); jornada tátil confirma que o cartão não expõe menu/seletor de status. Playwright serviu os assets fonte desta branch contra a API de teste porque o binário local embute assets anteriores; a alteração temporária do fixture foi restaurada.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 10 — Migração expand: Issue owner e assignee

**Commit sugerido:** `feat(wos): etapa 10 migração expand: issue owner e assignee`.  
**Dependência:** etapa 09 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `storage/{postgres,sqlite}/migrations/0034_*.sql [P]; storage/{postgres,sqlite}/repositories.go`.

**Trabalho:** Introduzir `objective_id` nullable temporário, assignee opcional e índice scope/objective/lifecycle conforme schema verificado.

**Aceite verificável:** Upgrade idempotente via runner; rollback/restore e dados anteriores intactos em ambos adapters.

**Execução em 2026-10-20:** concluída como migração expansiva `0034`, mantendo `objective_id` e `assignee_ref_json` nullable durante o backfill e a compatibilidade de leitura. Não reconstrói a tabela `issues` nem altera dados ou FKs de `issue_affected_refs`.

**Verificação:** testes SQLite e PostgreSQL cobrem upgrade de schema 0033, preservação de Issue/referências afetadas, repetição do runner, rollback transacional de DDL e dump/restore. Ambos os testes focados passaram; o gerador PostgreSQL registra o teste espelhado. O schema permanece expand até que o backfill seja verificado nas etapas seguintes.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 11 — Domínio Issue proprietário

**Commit sugerido:** `feat(wos): etapa 11 domínio issue proprietário`.  
**Dependência:** etapa 10 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `domain/issue.go; domain/issue_owner_test.go [P]`.

**Trabalho:** Adicionar ObjectiveID obrigatório na nova construção e serialização; manter leitor legado no ciclo expand; affected_refs independente.

**Aceite verificável:** Issue nova tem owner exato; Objective inconsistente rejeitado; lifecycle legado inalterado.

**Execução em 2026-10-20:** concluída. `Issue.ObjectiveID` é opcional apenas na representação de registros legados durante a expansão do schema. Novas instâncias devem passar por `NewIssueForObjective`/`ValidateForCreation`; a validação geral continua lendo dados legados. O owner estrutural é independente de `affected_refs`; existência e mesmo Outcome serão validados pela Application.

**Verificação:** `go test ./packages/wos-core/domain` passou com Go 1.27.2; `gofmt` e `git diff --check` passaram.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 12 — Comando de criação Issue com Objective

**Commit sugerido:** `feat(wos): etapa 12 comando de criação issue com objective`.  
**Dependência:** etapa 11 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `application/issues_blockers.go; commands/contracts.go`.

**Trabalho:** Exigir owner no CreateIssue, checar Objectivo não terminal e same scope, evolução de DTO; sem autoinferência por affected_refs.

**Aceite verificável:** Issue sem owner rejeitada; nenhuma Task bloqueada por Issue nova isolada.

**Execução em 2026-10-20:** concluída. `CreateIssue` e o comando atômico compatível `ReportIssueWithBlocker` exigem um Objective existente, no mesmo Outcome e não terminal. A Issue retornada e persistida mantém `objective_id`; `affected_refs` permanece independente. O transporte HTTP requer o ID explícito. A rejeição por owner ausente, inexistente ou terminal não grava Issue/Blocker nem avança a revisão.

**Verificação:** testes Application, SQLite e commands passaram; testes PostgreSQL focados passaram para leitura após restart, rollback, migração 0033→0034, replay e dump/restore real; `go build ./packages/wos-api/http` passou. A suíte HTTP não pôde ser executada porque o proxy Go respondeu 403 ao baixar `github.com/segmentio/encoding`; o handler compila. A persistência owner é a base mínima necessária nesta etapa; consulta indexada e listagem por Objective permanecem para a etapa 14.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 13 — Backfill verificado e checkpoint

**Commit sugerido:** `fix(wos): etapa 13 backfill verificado e checkpoint`.  
**Dependência:** etapa 12 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `tools/acceptance/issue_ownership_backfill.py [P]; docs/ux/operational-kanban/*`.

**Trabalho:** Importar mapeamentos explícitos de Issue e Task órfãs; dry-run/replay idempotente; nunca criar Objective-fantasma.

**Aceite verificável:** 0 órfãos por scope antes de contract; mappings auditáveis e nenhum dado sobrescrito incorretamente.

**Execução em 2026-10-20:** a ferramenta existente continua read-only por padrão e agora aceita mapeamento JSON versionado. A prévia valida cada registro sem escrita. A aplicação exige `--apply`, confirmação `--writers-stopped` e recibo novo com permissões privadas; usa transação serializável/`BEGIN IMMEDIATE`, owner explícito, escopo, Objective existente, versão esperada e justificativa por registro. Não infere candidatos, não cria Objectives, não sobrescreve proprietário diferente e falha em lote diante de qualquer conflito. Tasks só podem ser recuperadas se ainda estiverem em Backlog, sem lease/contrato ativo e sem dependência hard do Objective que será seu owner. O replay do mesmo mapeamento é no-op. Mudanças incrementam cada versão, avançam a revisão do Outcome uma vez e registram evento de domínio vinculado ao hash do mapeamento e à identidade ativa do operador. O recibo guarda o mapeamento e o resultado; não contém credenciais.

**Verificação:** auditoria read-only sem alteração, prévia, aplicação, evento e revisão do Outcome, replay, conflito de owner, Task ativa, versão obsoleta e rollback atômico passaram em SQLite; o teste opcional PostgreSQL passou em schema isolado usando transação serializável, inserção de eventos, replay e rejeição de CAS obsoleto. O resultado da auditoria pós-aplicação registra se o checkpoint de órfãos está zerado; isto sozinho ainda não libera a migração contract, que depende das etapas de paridade e validação seguintes.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

O formato e procedimento operacional estão em [`issue-ownership-backfill.md`](issue-ownership-backfill.md).

### Etapa 14 — Repository parity e índices de Issues

**Commit sugerido:** `feat(wos): etapa 14 repository parity e índices de issues`.  
**Dependência:** etapa 13 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `storage/{memory,sqlite,postgres}/*; ports/workspace_read.go`.

**Trabalho:** Persistir owner e assignee e ListPendingIssuesByObjective via consulta completa e mesmo snapshot; desempenho indexado.

**Aceite verificável:** SQLite/Postgres/memory retornam conjunto idêntico com paginação e isolamento.

**Execução em 2026-10-20:** `IssueRepository` agora expõe `ListPendingIssuesByObjective`, com cursor por ID estável, contagem completa independente do cursor e hidratação de `affected_refs` no mesmo snapshot da unidade de trabalho. A projeção considera `open` e `investigating`, filtra pela propriedade estrutural `objective_id` e usa o índice existente de scope/Objective/lifecycle/ID; não infere owner por `affected_refs`. O campo opcional `assignee_ref` e seu JSON foram conectados ao domínio e aos adapters Memory, SQLite e PostgreSQL, mantendo `reported_by` distinto e imutável. A coluna de armazenamento já havia sido criada de forma aditiva na etapa 10; as operações Assign/Unassign permanecem para a etapa 15.

**Verificação:** teste focal de paginação e isolamento passou em Memory e SQLite; o mesmo contrato passou em PostgreSQL real no banco de integração dedicado. Os casos verificam IDs ordenados, cursor, total completo, exclusão de Issues terminais, filtragem de Objective, hidratação de refs e persistência/hidratação de assignee. `gofmt` e `git diff --check` passaram.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 15 — Issue assignment no domínio

**Commit sugerido:** `feat(wos): etapa 15 issue assignment no domínio`.  
**Dependência:** etapa 14 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `domain/issue.go; domain/issue_assignment_test.go [P]`.

**Trabalho:** Adicionar operações Assign/Unassign para o `assignee_ref` já persistido; `reported_by` permanece imutável; estado investigating não cria controle de executor externo.

**Aceite verificável:** Sem criação de lease, concorrência protegida por versão e invariantes validadas.

**Execução em 2026-10-20:** `Issue.Assign` e `Issue.Unassign` agora atualizam apenas o assignee declarativo, sem mudar lifecycle ou `reported_by`. Apenas Issues `open` e `investigating` aceitam essas operações; Issues terminais falham sem mutação. Reatribuir é permitido, e atribuir novamente a mesma identidade ou remover uma atribuição ausente é no-op. Desatribuir uma Issue em investigação exige justificativa não vazia, que a Application registrará no evento de auditoria. A identidade atribuída não concede autoridade de execução nem cria, altera ou libera Task lease.

**Verificação:** testes de domínio cobrem atribuição e reatribuição, no-op, remoção em Issue aberta, justificativa para Issue em investigação, rejeição atômica em estados terminais e inválidos, preservação de lifecycle/reporter e avanço de versão/timestamp apenas quando há mudança.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 16 — Application de atribuição e investigação

**Commit sugerido:** `feat(wos): etapa 16 application de atribuição e investigação`.  
**Dependência:** etapa 15 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `application/issues_blockers.go; issue_assignment.go [P]; tests de aplicação`.

**Trabalho:** Implementar AssignIssue, UnassignIssue, StartIssueInvestigation com atomicidade, permissão e eventos; evitar race para assumir.

**Aceite verificável:** Apenas um vencedor em disputas simultâneas; histórico identifica responsável anterior/novo.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 17 — Policy de reabertura e duplicatas

**Commit sugerido:** `feat(wos): etapa 17 policy de reabertura e duplicatas`.  
**Dependência:** etapa 16 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `application/issues_blockers.go; domain/issue.go`.

**Trabalho:** Reabrir Issue em Objective achieved exige reabrir Objective antes; canonical duplicate chain fechada para gate, self/cycle/no target fail closed.

**Aceite verificável:** Duplicata aberta/canônica aberta não permite bypass do gate; sem alterações retroativas silenciosas.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 18 — Gate backend de Issues

**Commit sugerido:** `feat(wos): etapa 18 gate backend de issues`.  
**Dependência:** etapa 17 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `application/workspace_preconditions.go; issue_completion.go [P]; application/issue_completion_test.go [P]`.

**Trabalho:** Implementar requireObjectiveIssuesClosed no mesmo UoW/guard e reason-code objetivo; aplicar em AchieveObjective.

**Aceite verificável:** Open/investigating impedem Achieve, closed válidas permitem conforme demais gates; sem mutação em falha.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 19 — Gate nas capacidades e preview

**Commit sugerido:** `feat(wos): etapa 19 gate nas capacidades e preview`.  
**Dependência:** etapa 18 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `application/{workspace_capabilities,workspace_conclusion,workspace_read_types}.go`.

**Trabalho:** Usar mesmo predicate Issue gate em affordance e preview; mostrar count, sample refs, cobertura completa/partial explícita.

**Aceite verificável:** Capability 'Achieve' indisponível em exata paridade com comando; não inventa ausência de Issues com página parcial.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 20 — Fechamento de Issue e Blocker independentes

**Commit sugerido:** `feat(wos): etapa 20 fechamento de issue e blocker independentes`.  
**Dependência:** etapa 19 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `application/issues_blockers_compound.go; domain/blocker.go; testes atuais`.

**Trabalho:** Preservar liberação explícita; Issue resolvida não limpa Blocker; escolha de resultado final sem aceitar critérios automaticamente.

**Aceite verificável:** Regressões de resolução Issue/Blocker passam; rastreabilidade causal intacta.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 21 — Create Issue + N Blockers atômico

**Commit sugerido:** `feat(wos): etapa 21 create issue + n blockers atômico`.  
**Dependência:** etapa 20 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `application/issues_blockers_compound.go; ports/UoW e testes`.

**Trabalho:** Novo ReportIssueWithBlockers; validar destino, propagação, unique, versão observada, limite, idempotency e full rollback.

**Aceite verificável:** Um erro em qualquer alvo deixa 0 Issue/Blockers; êxito cria 1 Issue e N Blockers numa revision.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 22 — Consulta canônica de impacto de dependências

**Commit sugerido:** `feat(wos): etapa 22 consulta canônica de impacto de dependências`.  
**Dependência:** etapa 21 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `application/dependency_impact.go [P]; ports/workspace_read.go`.

**Trabalho:** Algoritmo bounded em grafo de hard/advisory: upstream/downstream, caminho, estado atual, coverage, revisão.

**Aceite verificável:** A blocked não 'propaga' Blocker para B; mostra B waiting apenas se hard incompleta; DAG segura.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 23 — Preview e batch edit de dependências

**Commit sugerido:** `feat(wos): etapa 23 preview e batch edit de dependências`.  
**Dependência:** etapa 22 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `application/dependencies.go; application/dependencies_test.go`.

**Trabalho:** Implementar preview + ApplyDependencyChanges atômico, em UoW, DAG sobre grafo final, IDs/removals versionados, guards por todos Sources.

**Aceite verificável:** Ciclos, invalidez/lease/freeze ou version conflitante rejeitam lote completo; paridade single command.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 24 — Contrato de relacionamento correctivo Issue↔Task

**Commit sugerido:** `fix(wos): etapa 24 contrato de relacionamento correctivo issue↔task`.  
**Dependência:** etapa 23 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `domain/relation.go; application/*relation*; tests de relation`.

**Trabalho:** Verificar existing relates_to endpoints; habilitar Issue/Task para relação informativa somente se necessário, sem criar coluna related_work_refs.

**Aceite verificável:** Issue mostra Task corretiva e vice-versa; nenhuma influência indevida em readiness.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 25 — Migração contract Task objective NOT NULL

**Commit sugerido:** `fix(wos): etapa 25 migração contract task objective not null`.  
**Dependência:** etapa 24 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `storage/{postgres,sqlite}/migrations/0035_*.sql [P]; testes upgrade`.

**Trabalho:** Enforce FK+NOT NULL depois de 0 órfãos; rebuild SQLite seguro; preservar outros índices/triggers.

**Aceite verificável:** Fresh install+upgrade+restore passam; schema não aceita WorkItem órfã.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 26 — Migração contract Issue owner NOT NULL

**Commit sugerido:** `fix(wos): etapa 26 migração contract issue owner not null`.  
**Dependência:** etapa 25 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `storage/{postgres,sqlite}/migrations/0035_*.sql [P]; testes migration`.

**Trabalho:** Enforce issue objective_id NOT NULL, FK correta, protegendo delete de Objective proprietário e consultas indexadas.

**Aceite verificável:** Não aceita Issue órfã, upgrade sem perdas; rollback e verificação de checksum.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 27 — Regenerar API/OpenAPI schema

**Commit sugerido:** `feat(wos): etapa 27 regenerar api/openapi schema`.  
**Dependência:** etapa 26 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `wos-api/{commands/*,openapi.yaml}; tools/openapigen`.

**Trabalho:** Novos DTOs, reason codes, exemplos, validação strict nullable, generated zero-diff.

**Aceite verificável:** Schema cobre criação, assignee, gate, compound Blockers e dependency batch; spec validada.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 28 — HTTP parity

**Commit sugerido:** `feat(wos): etapa 28 http parity`.  
**Dependência:** etapa 27 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `wos-api/http/{handler,workspace}.go; testes HTTP`.

**Trabalho:** Expor comandos/queries via roteamento existente, auth/CSRF/idempotency/403/409/412, pagination coerente.

**Aceite verificável:** Testes de request/response e erros sem exposição indevida de outros scopes.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 29 — MCP parity

**Commit sugerido:** `feat(wos): etapa 29 mcp parity`.  
**Dependência:** etapa 28 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `wos-api/mcp/{workspace,commands_generated}.go; tools/mcpgen`.

**Trabalho:** Expor contratos gerados e capability checks sem bypass do domínio; seguir tool naming atual.

**Aceite verificável:** MCP nega Task órfã e Objective com Issue aberta, sem caminhos de contorno.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 30 — SDK Go parity

**Commit sugerido:** `feat(wos): etapa 30 sdk go parity`.  
**Dependência:** etapa 29 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `wos-sdk-go/{client,workspace,commands_generated}.go`.

**Trabalho:** Adicionar typed methods, docs e testes de wire, erro/retry idempotente; evitar quebra silenciosa de tipos.

**Aceite verificável:** Código compila e roundtrip HTTP/MCP e client enfrentam mesmos guards.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 31 — CLI parity

**Commit sugerido:** `feat(wos): etapa 31 cli parity`.  
**Dependência:** etapa 30 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `wos-cli/{workspace_commands_v2,workspace_planning,workspace_inspect}.go; README`.

**Trabalho:** CLI para issue owner/assignee, compound, dependency preview/bulk; mensagens de gate e compatibilidade.

**Aceite verificável:** CLI completo para novos fluxos, sem acesso privilegiado oculto.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 32 — Read model de Kanban Issues

**Commit sugerido:** `feat(wos): etapa 32 read model de kanban issues`.  
**Dependência:** etapa 31 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `application/{workspace_cards,workspace_frame,workspace_read_types}.go; storage/{sqlite,postgres}/workspace_read.go`.

**Trabalho:** Issue lane list por `objective_id` com filtros/pagination/revision; contagens completas vs carregadas, boundaries claros.

**Aceite verificável:** Não misturar Issues de Objetives vizinhos e não confundir affected_refs com owner.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 33 — Seleção de contexto Issues no modal

**Commit sugerido:** `feat(wos): etapa 33 seleção de contexto issues no modal`.  
**Dependência:** etapa 32 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `web/assets/workspace-kanban.js; kanban-operations.js`.

**Trabalho:** Adicionar `issues` a frame, `New Issue` contextual, render das três lanes; manter retorno modal/Board.

**Aceite verificável:** Uma modal, seleção/scroll/camera/foco preservados após navegação e operação.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 34 — Cartão de Issue e filtros

**Commit sugerido:** `feat(wos): etapa 34 cartão de issue e filtros`.  
**Dependência:** etapa 33 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `web/assets/{workspace-kanban,workspace-kanban.css,en-US.js}; tests/web/issue-kanban.spec.mjs [P]`.

**Trabalho:** Card com lifecycle/resultado final, severity, assignee, affected count e Blockers; busca e filtros.

**Aceite verificável:** Desktop/HD/mobile legíveis, sem badge falsamente 'blocked' por Issue.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 35 — Inspector Issue e atribuição

**Commit sugerido:** `feat(wos): etapa 35 inspector issue e atribuição`.  
**Dependência:** etapa 34 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `web/assets/{workspace-view,human-forms,workspace-entity-context}.js`.

**Trabalho:** Mostrar proprietário, responsável, causa/impactos, relacionadas, evidências e eventos; operação assumir/repassar/liberar.

**Aceite verificável:** Owner permanece estável; assignee muda com expected_version e feedback atômico.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 36 — DnD Issue com chooser de fechamento

**Commit sugerido:** `feat(wos): etapa 36 dnd issue com chooser de fechamento`.  
**Dependência:** etapa 35 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `web/assets/{workspace-kanban,kanban-operations,human-forms}.js`.

**Trabalho:** Open→Investigating assume; Investigation→Closed abre resolved/wont_fix/duplicate; closed→open reabre sujeito a gate.

**Aceite verificável:** DnD não persiste até comando confirmado; paridade com inspector e teclado/touch.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 37 — Criação de Issue com impacts múltiplos

**Commit sugerido:** `feat(wos): etapa 37 criação de issue com impacts múltiplos`.  
**Dependência:** etapa 36 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `web/assets/{human-forms,workspace-blocking-form}.js; tests/web/blocker-impact.spec.mjs`.

**Trabalho:** Selecionar multi-alvo, preview de propagação, confirmar cada impact, atomically submit; preserve single-impact flow.

**Aceite verificável:** Fail atomic, target exacto, mensagens claras, sem auto-release.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 38 — UX Dependency source/target

**Commit sugerido:** `feat(wos): etapa 38 ux dependency source/target`.  
**Dependência:** etapa 37 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `web/assets/{workspace-view,workspace-entity-context,reference-picker}.js`.

**Trabalho:** Grupos Depends on/Required by e botões contextuais; relação é tipada e não vira ownership; pesquisa elegibilidade.

**Aceite verificável:** Usuário cria relações nos dois sentidos sem inverter source/target; keyboard equivalente.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 39 — Canvas connect mode

**Commit sugerido:** `feat(wos): etapa 39 canvas connect mode`.  
**Dependência:** etapa 38 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `web/assets/{workspace-canvas,canvas-data}.js; tests/web/dependency-connection.spec.mjs [P]`.

**Trabalho:** Modo conectar dois nós, preview arrow e escolher strength; oculta graph indiscriminado.

**Aceite verificável:** Nada altera posição hierárquica; preview e write validados; source backlogged.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 40 — Editor batch de dependências

**Commit sugerido:** `feat(wos): etapa 40 editor batch de dependências`.  
**Dependência:** etapa 39 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `web/assets/{human-forms,workspace-kanban}.js; application/dependencies.go`.

**Trabalho:** Seleção múltipla, add/remove em rascunho, preview DAG/status/freeze, único confirm e recovery.

**Aceite verificável:** Tudo ou nada e sem perda de draft em conflito; `Defer` sempre explícito.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 41 — Impacto causal e navegação transitiva

**Commit sugerido:** `test(wos): etapa 41 impacto causal e navegação transitiva`.  
**Dependência:** etapa 40 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `web/assets/{workspace-view,workspace-canvas,workspace-entity-context}.js`.

**Trabalho:** Exibir causa-raiz Issue→Blocker→Task diretamente, e Tasks downstream hard como waiting, linhas navegáveis e coverage.

**Aceite verificável:** No misleading inherited Blockers; impacto transito testado com A→B→C e advisory.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 42 — Testes de integração concorrência/storage

**Commit sugerido:** `feat(wos): etapa 42 testes de integração concorrência/storage`.  
**Dependência:** etapa 41 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `storage/{sqlite,postgres}/*test.go; tests/boundary/*`.

**Trabalho:** Races assume Issue; transação compound, migrations upgrade/restore; deadlock/locks, stale version/cursor, auth multi-namespace.

**Aceite verificável:** Go race, SQLite/Postgres reais, fixtures consistentes, revisão coesa.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 43 — E2E UX / onboarding / acessibilidade

**Commit sugerido:** `feat(wos): etapa 43 e2e ux / onboarding / acessibilidade`.  
**Dependência:** etapa 42 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `tests/web/{journey,progressive-entry,contextual-kanban,accessibility,issue-kanban,onboarding-regression}.spec.mjs`.

**Trabalho:** Jornadas completas de primeiro acesso, Objetive+Task, Issue→blocker→responsável→closed, objetivo gate, 1280×720, 390×844, 125/150/200/400% zoom, teclado.

**Aceite verificável:** 0 regressões não justificadas nos fluxos existentes; axe/teclado/screenshot contrast test.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

### Etapa 44 — Documentação, paridade e aceite final

**Commit sugerido:** `docs(wos): etapa 44 documentação, paridade e aceite final`.  
**Dependência:** etapa 43 concluída.  
**Arquivos envolvidos (existentes, salvo marcados [P]):** `README.md; docs/{workspace-relationships.md,WOS_Design_Arquitetura_Planejamento_Atualizado.md,ux/operational-kanban/README.md}; ADR; testes`.

**Trabalho:** Atualizar documentação de propriedade, responsabilidades, gates, cheatsheets CLI/SDK, runbooks de migração/rollout; executar gate completo e auditar head/PR Draft.

**Aceite verificável:** Zero schema drift; checks local/hosted qualificados; plano integral encerrado sem merge/release e onboarding preservado.

**Regra transversal:** testes/contratos relevantes no mesmo commit, sem alterações de onboarding ou mudança não justificada de identidade; diff/revision/event logs consistentes.

## 9. Cenários de testes obrigatórios e evidências

| ID | Given | When | Then |
|---|---|---|---|
| H-01 | Outcome ativo, sem Objective | CreateWorkItem(objective_id=null) | `task_objective_required`, zero write/event/revision |
| H-02 | Objective válido | CreateWorkItem | owner persistido; API/CLI/MCP/SDK retornam a mesma referência |
| H-03 | batch inclui Objective O e Task T(parent_local_key=O) | CreatePlanItems | ambas criadas atomicamente e no mesmo Outcome |
| H-04 | Task backlog sob Objective O | SetWorkItemObjective(null) | rejeição; preview também bloqueado |
| H-05 | Objective tem Task em execução | tentar reparent Task/branch | respeita Backlog/contract/lease e não altera scope |
| I-01 | Issue owner=O, open, com 0 Blockers | CompleteWorkItem sob O | conclusão permitida se todos os outros gates da Task forem satisfeitos |
| I-02 | Issue open sob O | AchieveObjective O | falha `objective_issue_pending`, sem mutação |
| I-03 | Issue investigating sob O | AchieveObjective O | falha idêntica |
| I-04 | todas Issues resolved, critérios satisfeitos | AchieveObjective O | pode avançar se **demais** gates permitirem |
| I-05 | Issue duplicate→canônica open | AchieveObjective | falha para evitar bypass |
| I-06 | Issue duplicate→canônica resolved | AchieveObjective | gate Issue satisfeito, demais gates independentes |
| I-07 | Issue wont_fix com motivo | AchieveObjective | Issue gate satisfeito; prova e critérios continuam validados |
| I-08 | Objective achieved | ReopenIssue sua filha | rejeita até ReopenObjective, não muda conclusão retroativamente |
| I-09 | Dois agentes simultâneos | StartIssueInvestigation mesmo issue/version | um vence; outro VersionConflict, sem dupla atribuição |
| I-10 | Issue own O mas affects Task sob O2 | concluir O2 | não herda gate de ownership de O; impactos explícitos de Blocker permanecem independentes |
| B-01 | Issue causa Blocker direto em Task A | Task B hard→A, C hard→B | A blocked, B/C waiting_dependencies conforme conclusão; nenhum Blocker criado em B/C |
| B-02 | Blocker subtree em Objective O | Task T de O e subobjective | T e descendants blocked; Task em Objective não descendente permanece fora |
| B-03 | Issue resolved com Blocker ativo | consultar/readiness | Blocker permanece até ResolveBlocker explícito |
| B-04 | Criação compound com N alvos, um inválido | submit | rollback total; nenhum Issue/Blocker parcial |
| D-01 | Source Task active | AddDependency out | planejamento proibido até defer explícito elegível |
| D-02 | Task X depende de Y | adicionar Y depende de X | cycle rejeitado sem alteração |
| D-03 | A hard→B e C advisory→A | bloquear B | A espera B; C não espera por advisory |
| D-04 | batch add/remove gera ciclo temporário porém não no grafo final | ApplyDependencyChanges | aplicar somente se grafo final válido, ordem internamente segura |
| D-05 | Context 'Required by A' | escolher B | persistir `source=B,target=A`, sem inverter |
| UI-01 | Canvas com zoom e seleção | abrir/fechar Issues Kanban | câmera, foco e seleção restaurados; uma modal |
| UI-02 | Issue Open | drag Investigating | apenas prepara StartIssueInvestigation; confirmação explícita |
| UI-03 | Issue Investigating | drag Closed | escolher resolved/wont_fix/duplicate; sem status setter direto |
| UI-04 | navegação teclado/mobile | atuar no card | mesmo comando sem exigir drag |
| ON-01 | instalação limpa | primeiro acesso, login e criação de Outcome/Objective/Task | fluxo preservado, Task vinculada validamente |
| ON-02 | usuário antigo com dados | upgrade expand/backfill/contract | ownership consistente, dados/eventos/histórico preservados |
| SEC-01 | outro namespace ou sem permissão | buscar Issue/assignee/dependency | 403/404 seguro; nenhum vazamento por preview/cursor |
| CON-01 | write confirmado, read falhou | retry de refresh | não reenvia write; idempotency key preservada |

### 9.1 Gates de execução exigidos

- `go test ./...`, `go vet ./...`, `go test -race` nos packages de domain/application e suites existentes de boundary; scripts geradores existentes em modo verificação/zero-diff.
- Testes reais de SQLite e PostgreSQL (não aceitar apenas 'skip' por ausência de DSN); migrations fresh/upgrade/backup/restore, testes de concorrência e idempotência.
- JS units da UI, Playwright E2E e snapshots nos breakpoints de referência do PR (HD, QHD, 4K, mobile), zoom nativo, teclado e axe escopado; teste manual de onboarding com checklist e comparativo visual.
- Paridade exata HTTP/MCP/CLI/SDK para `CreateIssue`, criação de WorkItem, atribuição, fechamento, batch dependency, erros e mesmo escopo.
- Nenhum commit gerado neste planejamento; logs de aprovação são tarefa da execução futura. Hosted CI na nova HEAD deve ser observada novamente, sem usar status verde de commit antigo.

## 10. Plano de deploy e compatibilidade

1. Congelar baseline do PR Draft e identificar clientes que enviam `CreateIssue` sem Objective ou `CreateWorkItem` órfão; publicar nova exigência de contrato para integradores antes do rollout.
2. Executar migrations **expand** nos dois adapters, com backups e health-check, sem endurecimento de colunas ainda; migrar readers/writers e clientes coordenadamente.
3. Corrigir dados legados **com mapeamento explícito autorizado**. Script fornece dry-run, exportação CSV do conjunto, hashes, modo apply idempotente e relatório pós-verificação. Não associar Issue arbitrariamente ao primeiro Objective nem inventar Objective-placeholder.
4. Ativar gates de domínio depois que o conjunto estiver consistente; testar paralelismo de antigas e novas versões. A política escolhida **não preserva escrita legada semanticamente inválida**: chamadas antigas devem receber erro claro, com `objective_id` requerido.
5. Contract migration (`NOT NULL`/FK) somente com contagem de órfãos igual a zero e verificação de integridade; realizar fresh install e restore em SQLite/Postgres.
6. Manter PR Draft durante todas as 44 etapas; não alterar README de licenciamento ou fluxos de onboarding fora do estritamente necessário ao novo objetivo obrigatório. Sem merge/release/tag/publicação nesta iniciativa sem autorização posterior.

## 11. Critérios finais e definição de concluído

A implementação só estará concluída quando (1) nenhuma Task/Issue operacional persistida puder existir sem Objective proprietário; (2) todo transport respeitar criação/reparent/estado e frontend não oferecer caminhos inválidos; (3) toda Issue pendente do Objective bloquear `AchieveObjective`, mas não a conclusão de Tasks por si só; (4) Blocker e hard dependency forem exibidos sem colapso semântico e as cadeias transitivas corretas estiverem navegáveis; (5) assignee único de Issue e todos os comandos de resolução forem auditáveis e concorrentes; (6) Kanban Issue trabalhar com 3 lanes, closed outcomes distintos, modal e UX preservados; (7) edição de dependências em ambos os sentidos e em batch obedecer DAG, versão, Backlog-only e commit atômico; (8) SQLite e Postgres passarem migração real/restore, APIs geradas e clientes funcionarem; (9) onboarding primeiro acesso, segurança, acessibilidade, imagem/estrutura visual e fluxos existentes passarem regressão; (10) PR #116 continuar Draft, com 44 commits rastreáveis e CI da HEAD de implementação qualificada.

**Proibição de encerramento por evidência incompleta:** check não executado, PostgreSQL pulado, index incompleto, screenshot antigo, revisão de PR anterior ou teste visual meramente estimado não contam como aceite.

## 12. Links do baseline (código e ADR)

- [PR #116](https://github.com/A1b3rt0M3rcad0/wos/pull/116).
- [WorkItem](https://github.com/A1b3rt0M3rcad0/wos/blob/d80d8329e097b7252b65b9c75f49e00d6cdcfa39/packages/wos-core/domain/work_item.go), [Issue](https://github.com/A1b3rt0M3rcad0/wos/blob/d80d8329e097b7252b65b9c75f49e00d6cdcfa39/packages/wos-core/domain/issue.go), [Blocker](https://github.com/A1b3rt0M3rcad0/wos/blob/d80d8329e097b7252b65b9c75f49e00d6cdcfa39/packages/wos-core/domain/blocker.go), [Relation](https://github.com/A1b3rt0M3rcad0/wos/blob/d80d8329e097b7252b65b9c75f49e00d6cdcfa39/packages/wos-core/domain/relation.go).
- [CreatePlanItems](https://github.com/A1b3rt0M3rcad0/wos/blob/d80d8329e097b7252b65b9c75f49e00d6cdcfa39/packages/wos-core/application/planning_creation.go), [Issue+Blocker commands](https://github.com/A1b3rt0M3rcad0/wos/blob/d80d8329e097b7252b65b9c75f49e00d6cdcfa39/packages/wos-core/application/issues_blockers_compound.go), [Preconditions](https://github.com/A1b3rt0M3rcad0/wos/blob/d80d8329e097b7252b65b9c75f49e00d6cdcfa39/packages/wos-core/application/workspace_preconditions.go).
- [Kanban modal](https://github.com/A1b3rt0M3rcad0/wos/blob/d80d8329e097b7252b65b9c75f49e00d6cdcfa39/packages/wos-api/web/assets/workspace-kanban.js), [Human forms](https://github.com/A1b3rt0M3rcad0/wos/blob/d80d8329e097b7252b65b9c75f49e00d6cdcfa39/packages/wos-api/web/assets/human-forms.js), [ADR-041](https://github.com/A1b3rt0M3rcad0/wos/blob/d80d8329e097b7252b65b9c75f49e00d6cdcfa39/docs/adr/0041-focused-kanban-modal-and-backlog-planning.md).

