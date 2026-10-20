# WOS — Plano de implementação da nova experiência de Kanban operacional

**Documento:** Especificação de implementação e critérios de aceite  
**Data de referência:** 20/10/2026  
**Repositório:** [`A1b3rt0M3rcad0/wos`](https://github.com/A1b3rt0M3rcad0/wos)  
**PR de referência:** [#116 — Operational Contextual Kanbans](https://github.com/A1b3rt0M3rcad0/wos/pull/116)  
**Branch examinada:** `fix/workspace-operational-density`  
**HEAD examinado:** `75a56fe43ddd20ff55835fd2ba22ca8af9588d1c`  
**Natureza:** plano de evolução da experiência existente, **não** implementação de código.  
**Idioma da interface:** inglês, conforme a interface e o PR atuais; este documento está em português.

> **Regra de atualização:** antes de executar qualquer etapa, verificar novamente o HEAD da branch e reconciliar este plano com alterações posteriores. Os achados abaixo estão referenciados à versão inspecionada, não são uma afirmação sobre qualquer commit futuro.

---

## 1. Objetivo e resultado esperado

Transformar o Kanban do WOS em uma superfície de **visualização, inspeção e operação de trabalho**, na qual o usuário compreenda imediatamente a situação de cada item e consiga executar uma transição legítima a partir do gesto de arrastar, sem navegar por telas técnicas ou perder o contexto do Board.

**Experiência final desejada:**

1. A pessoa abre o Kanban contextual de um Outcome ou Objective a partir do Canvas.
2. As colunas e os cards permanecem visíveis enquanto ela inspeciona um item no painel contextual.
3. Arrastar um card para outra coluna **prepara uma intenção de negócio**; não altera diretamente um campo `status`.
4. Uma operação simples apresenta confirmação mínima quando necessária; uma operação com requisitos abre um fluxo guiado, com explicação objetiva e remediação no mesmo contexto.
5. Após a confirmação e persistência pelo comando canônico do WOS, o Kanban atualiza sua projeção a partir do servidor e devolve o foco ao card apropriado.
6. Critérios, evidências, artefatos, assessments, responsáveis, autoridade, impedimentos e histórico podem ser consultados ou trabalhados diretamente no inspector, sem criar entidades ou estados artificiais.
7. A identidade editorial do WOS permanece intacta: cores, tipografia, ilustrações, robozinhos, superfícies em papel e densidade visual característica.

**Critério norteador:** reduzir navegações e decisões artificiais **sem simplificar indevidamente as regras de domínio**.

## 2. Escopo e limites explícitos

### 2.1 Incluído

- Evoluir a apresentação contextual do Kanban e sua integração com o Canvas.
- Substituir a **página de detalhe que esconde as colunas** por inspector compacto de consulta e gerenciamento, responsivo e não modal.
- Eliminar o select `Move to…` permanentemente exposto sob cada card, preservando alternativa acessível a drag-and-drop.
- Tornar o arraste um gatilho claro para um **fluxo de preparação/execução de operação**, com feedback, confirmação, remediação, espera e recuperação.
- Definir lanes e transições de **Objective** e **Task** segundo seus ciclos de vida reais, inclusive `backlog` versus `todo`.
- Reorganizar cards, filtros, contadores, toolbar, feedback e estados vazios.
- Integrar ações contextuais de critérios, evidências, artefatos, avaliações, atribuição, dependências, Issues e Blockers.
- Reutilizar a superfície canônica de comandos, forms e suas garantias de autorização, versão, idempotência e recuperação.
- Cobrir mouse, teclado, toque, zoom, viewport curto, screen readers e fluxos com dados parciais.
- Atualizar testes, documentação de UX e capturas de evidência.

### 2.2 Fora de escopo

- Criar `Board`, `Column`, `KanbanCard` ou `KanbanStatus` como novas entidades persistidas.
- Criar API genérica de `set_status` ou permitir arraste atualizar diretamente `lifecycle`.
- Inventar ciclo de submissão/revisão para Tasks que não o possuam.
- Mudar os ciclos de vida de Objective, Task, Outcome, Issue, Blocker, Criterion ou Assessment.
- Executar Agents no navegador, emitir assinaturas, selecionar credenciais externas ou substituir o host autorizado.
- Tornar assignment equivalente a autoridade de execução.
- Fazer Evidence equivaler a Assessment, nem Task done equivaler a Objective achieved ou Outcome accepted.
- Criar Cloud, sistema de uploads, scheduler, filas de revisão parentais ou infraestrutura paralela.
- Refatorar CLI, SDK ou contratos públicos além de adaptações **estritamente aditivas e justificadas**, caso necessárias.
- Redesenhar a identidade visual ou trocar a linguagem editorial do produto.

### 2.3 Pré-condições de execução

- Trabalhar no contexto do PR #116, sem presumir que suas alterações já estejam publicadas em `master`.
- Verificar a branch corrente, o HEAD, o estado dos testes, componentes e contratos de API.
- Ler e aplicar as skills instaladas no repositório:
  - `.agents/skills/laws-of-ux-auditor/SKILL.md`;
  - `.agents/skills/product-design-and-ux/SKILL.md`.
- Consultar os documentos existentes `docs/ux/operational-kanban/`, `docs/ux/contextual-kanban/`, `docs/adr/0039-contextual-kanbans-and-strict-completion.md` e as regras do domínio antes de modificar a UX.
- Não considerar sucesso em testes automatizados equivalente a validação de usabilidade com participantes.

---

## 3. Diagnóstico verificado na branch

| ID | Evidência no código / interface atual | Correção proposta |
|---|---|---|
| K-01 | `workspace-kanban.js` mantém `current().selected`, oculta `.workspace-kanban-columns` e renderiza `.workspace-kanban-inspection` como página inteira. | Inspector contextual com colunas persistentes; detalhe completo somente sob escolha explícita. |
| K-02 | Cada card usa `draggable = true`, mas recebe um select `Move to…` permanente após o card. | Arraste como ação principal; menu contextual/controle acessível como alternativa, não um select abaixo de cada card. |
| K-03 | O drop atual ignora destino igual à lane atual (`item.stage !== stage`). A projeção atual agrega `backlog` e `todo` em `todo`. | Separar visualmente planejamento e disponibilidade da Task e corrigir a elegibilidade das intenções de transição. |
| K-04 | `kanban-operations.js` associa um nome de coluna a uma capability e abre a operação; Verification pode apenas abrir leitura de revisão, sem mudança de fase. | Preparador de transição explícito, tipado por entidade e sem promessas de estados inexistentes. |
| K-05 | `workspaceKanbanStage` é uma projeção do backend: `awaiting_review` pode gerar `verification`; `done`/`achieved` geram `accepted`. | Respeitar fatos reais e distinguir lane apresentada de lifecycle persistido. |
| K-06 | O Kanban inteiro é um `<dialog>` nativo; operações internas reutilizam o host de formulários e manipulam seu conteúdo. | Transformar o Kanban em modo contextual **não modal** dentro da área do Canvas; reservar modal nativo para a operação. |
| K-07 | Informações do item aparecem distribuídas em seção de próxima operação, facetas, registros, auditoria e listas extensas de ações. | Hierarquia operacional por pergunta: o quê, quem, pendências, provas, histórico; uma ação principal contextual. |
| K-08 | Busca e contadores refletem apenas cards carregados e a paginação depende de revisão. | Conservar avisos de cobertura parcial; jamais apresentar contagem carregada como total autoritativo. |
| K-09 | `app.js` integra `createContextualKanban` a `canvas-details` e possui recuperação específica de foco, câmera e operações incertas. | Migrar a apresentação sem duplicar forms, revogar garantias do pipeline nem deslocar a câmera do Canvas. |

**Importante:** o PR já implementou funções reais: drag-and-drop, ações, handoff externo, tratamento de conflito, preservação de revisão, histórico, critérios e testes de jornada. A tarefa é **reorganizar e completar essa experiência**, não substituí-la por um CRUD de status.

---

## 4. Decisão de arquitetura da interface

### 4.1 Superfícies da experiência

Adotar **três superfícies com responsabilidades distintas**:

| Superfície | Papel | Comportamento |
|---|---|---|
| **Canvas / Outcome Board** | Visão espacial principal do Outcome | Continua sendo a entrada padrão, preservando câmera, seleção e hierarquia. |
| **Kanban contextual** | Operar os filhos diretos de um Outcome ou Objective | Passa a ocupar a **área útil do workspace** como modo contextual não modal; evita a sobreposição de um grande modal sobre o Canvas. |
| **Inspector do card** | Ler e gerenciar um item sem abandonar as colunas | Painel contextual não modal, acoplado em telas amplas e sobreposto/ocupando a região de conteúdo conforme viewport. |
| **Modal de operação** | Confirmar comando ou cumprir requisitos da intenção | Único modal nativo ativo durante o comando; reutiliza formulário e pipeline existentes. |

Embora existam quatro componentes na tabela, há **três níveis de interação** para o usuário: Kanban, Inspector e Operação; o Canvas é o espaço de origem e retorno.

**Decisão:** não manter o Kanban como `<dialog>` na arquitetura final. A exigência de abrir um modal de transição ao soltar o card é mais coerente quando o Kanban pertence ao workspace. Evitar um segundo diálogo aninhado e não simular um modal com semântica inacessível.

**Compatibilidade de migração:** conservar, quando viável, a fachada pública `open`, `close`, `reset`, `navigate`, `resume` e `finishOperation` durante o refactor, desacoplando internamente o container visual; adaptar as chamadas em `app.js` de forma incremental. **Não** criar outra implementação paralela de Kanban com estados próprios.

### 4.2 Navegação

- Ao abrir o Kanban de um Outcome, iniciar em **Objectives**; permitir alternar para **Direct Tasks** sem alterar a árvore de domínio.
- Ao abrir o Kanban de um Objective, iniciar em **Tasks**; permitir alternar para **Subobjectives**.
- Preservar a pilha de contexto já existente (`frames`), busca, filtros, scroll horizontal, scroll de lane, seleção e disclosure enquanto a sessão/scope continuar válida.
- Uma Task não abre um Kanban próprio; abre seu inspector. Um Objective pode abrir Kanban de filhos via ação contextual.
- *Back to Canvas* restaura a câmera e a seleção anteriores, sem `fit` ou `pan` silencioso.
- Não inventar esquema de URL ou novo roteador sem verificar o contrato atual de `navigation.js`; compatibilizar o estado contextual com a navegação do Canvas existente.
- Mudança de principal, revogação, escopo ou geração da sessão invalida dados privados e leituras em voo.
- Ao abrir uma operação, a navegação de saída respeita os bloqueios de `state.pending`, `state.submitting` e drafts existentes.

### 4.3 Contrato de layout

- Toolbar única e compacta: **breadcrumb / tipo de Kanban**, busca, filtros, atualizar e botão de retorno ao Canvas; comandos de criação em ação contextual visível, sem rodapé permanente muito alto.
- Lanes ocupam o espaço útil disponível e preservam largura mínima legível; quando necessário, usar scroll horizontal **do Kanban**, não reduzir cards até ficarem ilegíveis.
- Inspector: largura alvo aproximada **400–480 CSS px** em telas amplas, ajustada por densidade real; em HD deve ser sobreposto à área do Kanban sem recalcular/deslocar todas as lanes.
- Em telas de baixa largura, inspector vira painel de leitura que usa a área disponível, com retorno inequívoco ao board; não sobrepor controles essenciais.
- Um scroll dominante por região; evitar múltiplas caixas roláveis sem orientação e rodapés cobrindo o último campo.
- Zoom espacial do Canvas nunca dimensiona controles, fontes ou cards do Kanban.
- Manter os tokens de cores, tipografia, bordas, fundos em papel, ilustrações e estados visuais já existentes; não introduzir outro design system.

---

## 5. Contrato de lanes e domínio (decisão obrigatória)

**Não haverá `setStatus` nem mutação otimista de lane.** Uma lane é uma apresentação do estado/fatos canônicos, calculada de forma consistente com a Application.

### 5.1 Kanban de Objectives

Lanes principais:

1. **Planned** — `Objective.lifecycle = planned`.
2. **In progress** — `Objective.lifecycle = in_progress`.
3. **Achieved** — `Objective.lifecycle = achieved`.

`Cancelled` aparece apenas mediante filtro/visão explícita e **nunca** compõe concluídos.

**Não exibir uma lane `Verification` para Objectives** enquanto o domínio não possuir uma fase real de revisão de Objective. Critérios/assessments são consultados no inspector e integram a operação `AchieveObjective`, sem inventar fila de submissão.

### 5.2 Kanban de Tasks

Lanes:

1. **Backlog** — `WorkItem.lifecycle = backlog`: planejada, ainda não disponibilizada.
2. **To do** — `WorkItem.lifecycle = todo`: disponibilizada; a prontidão para aquisição depende das condições atuais.
3. **In progress** — `WorkItem.lifecycle = in_progress`, salvo projeção de revisão real. Exibir autoridade registrada sem inferir presença online.
4. **Verification** — somente quando houver fato aplicável `display_state = awaiting_review`/review pendente reconhecido pelo backend. Não inventar lifecycle `verification`.
5. **Accepted** — `WorkItem.lifecycle = done`.

`Cancelled` permanece em lane/filtro separado, nunca aceito.

**Por que cinco lanes?** Separar `backlog` de `todo` resolve a confusão operacional observada: `ActivateWorkItem` é uma operação real que não pode ser comunicada como mover um card dentro de uma única coluna visual.

**Compatibilidade de dados:** `WorkspaceCardSummary` já possui `ref.kind`, `lifecycle`, `display_state` e `stage`. Implementar o agrupamento tipado na projeção de apresentação, com testes de equivalência. Antes de alterar `workspaceKanbanStage` ou o campo público `stage`, identificar consumidores e contratos existentes. Preferir não alterar silenciosamente um enum consumido externamente; se necessário, adicionar um campo de leitura **aditivo** e documentado, sem migração e sem novo estado persistido.

### 5.3 Correspondência entre destino e comando real

| Entidade / origem | Intenção de destino | Operação possível | Resultado e regra |
|---|---|---|---|
| Objective `planned` | `In progress` | `StartObjective` | Preparar requisitos e confirmar; servidor decide. |
| Objective `planned` ou `in_progress` | `Achieved` | `AchieveObjective` | Aceitar **diretamente de planned** quando permitido pelo domínio; exige avaliações/conclusão aplicáveis. |
| Task `backlog` | `To do` | `ActivateWorkItem` | Disponibilizar; não adquirir autoridade. |
| Task `todo` | `In progress` | `ClaimWorkItem`, `AcquireWorkContract` ou `AcquireSignedWorkContract` conforme protocolo | Lease humano legítimo ou **handoff ao host autorizado**; assignment não substitui aquisição. |
| Task `backlog` | `In progress` | Primeiro `ActivateWorkItem`, depois aquisição pertinente | Fluxo guiado em duas decisões explícitas; **não** disparar múltiplos comandos em silêncio. |
| Task `in_progress` | `To do` | `ReleaseWorkItem`, **somente** quando autorizado para lease legado | Sem falsa reversão de contratos assinados. |
| Task `in_progress` | `Verification` | Preparação de material, critérios e possível handoff de entrega/revisão | Se não houver protocolo/fato de revisão, abrir fluxo de comprovação **sem mudar a lane**. |
| Task `verification` ou `in_progress` | `Accepted` | `CompleteWorkItem` quando realmente disponível | Verificar assessments, prova, dependências, blockers, autoridade e conclusão. |
| Task com lease expirado | `In progress` | `ReclaimWorkItem` se autorizado | Nova autoridade/fencing; **não** interpretar como mudança automática de lane. |
| Task com lease válido | Manter trabalho | `RenewWorkItemLease` quando autorizado | Estender autoridade sem alterar a lane. |
| Item terminal ou cancelado | Outro destino | Apenas comando explícito de reabertura/recuperação, **se existir e for autorizado** | Jamais permitir mover arbitrariamente por drag. |

**Operação inexistente:** exibir a razão sem executar nada. **Operação bloqueada:** explicar impedimentos e oferecer remediações elegíveis. **Operação de host externo:** mostrar handoff; não fabricar aquisição, entrega, avaliação ou assinatura no navegador.

---

## 6. Contrato de interação do card

### 6.1 Conteúdo mínimo

O card apresenta, nesta ordem:

1. Título e indicador discreto de que pode ser arrastado.
2. Estado projetado com texto — nunca somente cor.
3. Prioridade apenas se relevante.
4. Responsável **declarado**, quando disponível (não confundir com executor autenticado).
5. Até um resumo de critérios, distinguindo atendidos, dispensados e pendentes quando aplicável.
6. Uma pendência principal contextual: blocker, dependência não atendida, prova/assessment pendente ou requisito de autoridade.
7. Menu `…` de ações contextualizadas, incluindo alternativa **Move to…**; ele não ocupa espaço abaixo de todos os cards.

O card não é um relatório nem uma lista extensiva de registros. Evidências completas, critérios detalhados, participantes e histórico pertencem ao inspector.

### 6.2 Comportamentos

| Ação | Comportamento esperado |
|---|---|
| Clique/Enter no card | Abre inspector, mantendo as lanes montadas e seu scroll preservado. |
| Arraste iniciado | Exibe affordance e destaca destinos compatíveis; não dispara comando. |
| Soltar em destino elegível | Abre modal de operação preparada. |
| Soltar em destino com pré-condição faltante | Abre o mesmo fluxo com requisito/explicação/remediação; nenhuma escrita ocorre por esse gesto. |
| Soltar na lane de origem | Cancela arraste e preserva estado; não abre operação redundante. |
| Soltar em destino sem transição semântica | Informa que não há operação correspondente; mantém lane. |
| Cancelar/ESC | Fecha preparação se não houver operação irreversível, pendente ou incerta; restaura foco e scroll. |
| Sucesso confirmado | Refaz leitura coerente e posiciona card segundo o estado retornado; feedback acessível. |
| Falha de comando | Mantém card onde está; preserva conteúdo de formulário e oferece remediação adequada. |
| Menu alternativo / teclado / toque | Permite escolher a mesma intenção sem drag. |

**Não tratar todos os alvos como elegíveis** apenas por serem visualmente próximos. Diferenciar destino `eligible`, `requires_preparation`, `read_only`, `external_handoff` e `unsupported` na UI; toda autorização final permanece no servidor.

### 6.3 Sinalização durante drag

- Ícone/handle visível apenas com contraste adequado, sem tornar o card dependente exclusivamente de um pequeno alvo para operação via mouse.
- Realce de destino com texto/ícone, não só mudança de cor.
- Prévia breve de significado da intenção: `Start objective`, `Make Task available`, `Claim Task`, `Review proof`, `Complete Task`.
- Destinos indisponíveis não parecem aceitáveis; podem oferecer explicação sem executar nada.
- Movimento visual não causa reordenação persistente do Canvas nem alteração de pai/ownership. Drag em Kanban **não** substitui as operações estruturais `SetWorkItemObjective` ou `SetObjectiveParent`.
- Limite de movimento para distinguir clique de arraste; evitar ativação involuntária por clique após drop.

---

## 7. Modal de transição: especificação operacional

### 7.1 Estados de UI

Definir o estado da preparação de forma explícita, separado do estado persistido do trabalho:

- `idle`: nenhum fluxo aberto;
- `preparing`: leitura fresca de item/capacidades;
- `ready`: operação canônica identificada e elegível;
- `needs_preparation`: requisito não atendido com ação/remediação identificada;
- `unsupported`: não existe transição válida para a intenção;
- `external_handoff`: a ação depende do host/protocolo externo;
- `editing`: usuário preenche formulário canônico;
- `submitting`: comando enviado, ainda não confirmado;
- `refreshing`: comando confirmado, projeção do Kanban sendo reconciliada;
- `conflict`: versão/estado mudou; revisão humana do impacto requerida;
- `uncertain`: resposta incerta; **preservar intenção, body e idempotency key**;
- `success`: persistência confirmada e leitura atualizada.

Não gravar esses estados em entidades do domínio; são apenas apresentação.

### 7.2 Conteúdo do modal

**Cabeçalho:** título objetivo (`Start Objective`, `Make Task available`, `Claim Task`, `Complete Task`, `Review criteria`), identificação do item e origem → destino pretendido. Não afirmar que o destino foi atingido antes da confirmação.

**Corpo:** breve descrição da consequência, status da elegibilidade, requisitos faltantes, relação com critério/Blocker/dependência quando existir, ação de remediação diretamente acionável e campos **somente** da operação canônica necessária.

**Rodapé:** uma ação principal, Cancel/Back quando permitido, e feedback de espera/erro. Rodapé não pode cobrir o último campo; conteúdo central com scroll próprio.

**Casos específicos:**

1. **Start Objective**: mostrar critérios/requisitos atuais, Outcome ativo e impedimentos reportados; apresentar formulário de `StartObjective` somente quando elegível.
2. **Activate Task**: explicar que disponibilizar não significa atribuir nem adquirir execução; confirmar `ActivateWorkItem`.
3. **Claim/Acquire**: distinguir Task pronta, assignee declarado, capacidade efetiva de aquisição e autoridade real. Se protocolo assinado, handoff no host autorizado, sem coleta de credenciais no browser.
4. **Prove/Review**: exibir critérios atuais, evidências, artefatos e assessments aplicáveis. Se não existe submissão/review pendente, a ação *Review proof* é informativa/operacional e **não** move para Verification.
5. **Complete/Achieve**: explicar pendências e conclusão formal. **Task done não atinge Objective**; **Objective achieved não aceita Outcome**.
6. **Blocker/Issue**: Issue resolvida não remove Blocker automaticamente; liberar o impedimento demanda ação explícita.
7. **Revision mismatch**: oferecer revisão comparativa do estado atual sem reusar silenciosamente versão/acknowledgement obsoletos.
8. **Resposta incerta**: travar saída destrutiva e invocar recuperação canônica da intenção original; nunca fabricar uma segunda escrita.

### 7.3 Motor de preparação, sem duplicar domínio

Extrair de `kanban-operations.js` uma função de apresentação pura, por exemplo:

```text
prepareKanbanIntent({ view, targetLane, originLane })
  -> { kind: "command", capability, explanation }
   | { kind: "needs_preparation", reasons, remedies, candidateAction }
   | { kind: "external_handoff", capability, explanation }
   | { kind: "inspection", target: "criteria_or_delivery" }
   | { kind: "unsupported", explanation }
```

**Este é um contrato interno ilustrativo**, não uma API pública obrigatória. O objeto retornado **não** autoriza comandos nem cria body/fencing/assinatura por conta própria. Utilizar `workspaceActions(view)`, `WorkspaceCapability` (`action_id`, `availability`, `transport`, `reason_codes`, `remedies`, `observed_version`) e leitura fresca do servidor. O pipeline já existente prepara, valida, envia e recupera a operação.

Se as capabilities atuais não trouxerem uma explicação suficientemente específica, ampliar a **leitura** de modo aditivo, e não duplicar regras de negócio no frontend.

---

## 8. Inspector operacional: contrato de conteúdo e comportamento

### 8.1 Comportamento essencial

- Clique no card **não** oculta `.workspace-kanban-columns`.
- Inspector não é modal: o usuário pode manter contexto visual, voltar às lanes e abrir outro item de forma previsível.
- A seleção de card não desloca a câmera do Canvas, não muda o pai, não altera lifecycle e não faz write.
- Em desktop amplo, painel pode ser acoplado sem comprimir exageradamente as lanes; em HD, preferir sobreposição controlada. Em mobile, navegação para painel de leitura com retorno claro.
- Acesso a conteúdo completo via `View full details`, quando necessário; essa opção não deve ser a consequência automática do clique.
- Manter histórico de navegação para Criterion e registros relacionados; voltar preserva origem e scroll.

### 8.2 Organização do inspector

| Seção | Informações iniciais | Ações contextuais |
|---|---|---|
| **Overview** (sempre visível) | Nome, contexto pai, lifecycle/display state, objetivo da atividade, orientação curta e próxima ação real | `Start`, `Make available`, `Claim`, `Review`, `Complete` ou remediação, conforme capabilities. |
| **What to do** | Instruções, restrições, entregáveis e definição — sem inventar work list a partir de texto | Editar definição quando elegível. |
| **Responsibility & authority** | Owners/assignees declarados, executor/reviewer registrados, validade de lease/contrato, última atividade | Atribuir, renovar/reivindicar, handoff se autorizado. |
| **Criteria & proof** | Critérios atuais, status de assessment, vínculos de Evidence, material Artifact, gaps | Adicionar critério, evidência, artefato e avaliar pelo comando correto. |
| **Dependencies & impediments** | Dependências, Issues e Blockers com origem/efeito | Abrir referência, resolver Issue ou Blocker por ações próprias. |
| **Activity** | Timeline ordenada com autor, operação, horário, origem, resultado, cursor histórico | Ver registro e carregar mais com fronteira congelada. |
| **More actions / audit** | Alterações estruturais, operações raras, versões técnicas e detalhes de auditoria | Operações explicitamente autorizadas. |

**Regras de exibição:**

- Ao apresentar status de critério, distinguir `met`, `not met`, `inconclusive`, `waived`, `not assessed`, `outdated` e `not applicable` conforme o modelo real.
- Evidence ligada não significa critério atendido. Assessment de revisão anterior não prova estado vigente.
- Não assumir autor = executor, assignee = detentor de lease, contrato vigente = Agent online.
- Contexto carregado parcialmente deve indicar `not loaded`/`partial`, não `empty`/`0`.
- O próximo passo não deve vir apenas de ordenação numérica de capabilities. Escolher ação primária por **intenção operacional**, considerando status, requisitos, permissão e relevância; ações secundárias ficam agrupadas.
- Evitar listas técnicas extensas no primeiro viewport. O inspector permite expandir, pesquisar e abrir entidades relacionadas sem perder a Task.

### 8.3 Critérios, prova e verificação no mesmo contexto

Para um critério, abrir uma seção/página secundária do inspector com definição e revisão aplicável, Evidences relacionadas, Assessments e ação adequada. A pessoa deve conseguir:

1. Entender **o que precisa ser verdadeiro**.
2. Ver **que material existe** e quem o registrou.
3. Entender se o material foi **avaliado de forma válida**.
4. Registrar novo material ou assessment pela operação autorizada.
5. Retornar ao item e tentar a conclusão, com nova validação do backend.

Não fazer um checkbox livre mudar estado de critério. Uma dispensa autorizada permanece visualmente diferente de uma condição comprovada.

---

## 9. Toolbar, pesquisa, filtros, contadores e densidade

### 9.1 Toolbar

- Exibir um breadcrumb compacto indicando Outcome/Objective atual, tipo de Kanban e retorno ao Canvas.
- Uma pesquisa sobre **itens carregados** com rótulo explícito. Não prometer busca global se não foi implementada.
- Filtros recolhíveis de prioridade, participante declarado, objetivos obrigatórios/opcionais e cancelados conforme o tipo.
- `Refresh` visível, com feedback de leitura, sem recriar arbitrariamente a seleção.
- Ações `Add Objective`, `Add Task` ou `Add Subobjective` contextuais à hierarquia e modo atual; evitar rodapé de ações separado e persistente quando desnecessário.

### 9.2 Estados vazios e contadores

- Cada lane mostra contador de **cards carregados**; se cobertura parcial, rotular explicitamente, por exemplo `3 loaded` / `More items may exist`.
- Não apresentar zero como total absoluto quando a página ainda não carregou ou está incompleta.
- Lanes vazias têm título, mensagem curta e região de drop clara, sem caixas gigantes decorativas.
- Continuar usando paginação por cursor e `outcome_revision` coerente. Em mudança de revisão, pedir refresh; nunca mesclar páginas incompatíveis.
- Totais autoritativos por lane e filtros executados pelo servidor ficam **fora deste incremento**, salvo se uma dependência técnica for demonstrada. Não simular agregação global no cliente.

### 9.3 Densidade e responsividade

- Usar tamanho de fonte legível e espaçamento adaptativo, sem encolher texto em telas maiores.
- Cards com título destacado, linha de estado, até uma pendência e indicadores compactos de critério e responsabilidade.
- Deixar comprimento de conteúdo e título quebrar sem overflow nem truncar a ação principal.
- O inspector não deve tornar as colunas inutilizáveis em `1280×720` e `1366×768`.
- Desktop ultrawide utiliza área adicional para exibir mais lanes/cards, não para espaçar indefinidamente os elementos.
- Suportar zoom nativo 125%, 150%, 200% e 400%, com leitura vertical no viewport efetivo; nenhum controle obrigatório pode ficar inacessível.

---

## 10. Acessibilidade e recuperação de interação

### 10.1 Mouse, teclado e touch

- Drag com mouse/trackpad e affordance visível.
- Alternativa **sem drag**: menu contextual `Move to…` acessível por clique único, toque e teclado; apresenta os destinos aplicáveis com nome, requisito e resultado, sem select permanente.
- Navegação com Tab/Shift+Tab por controles, Enter/Space para operar e ESC para encerrar somente o nível atual quando permitido.
- O card deve ser inspecionável sem depender de hover nem movimentação precisa.
- Feedback de mudança de lane por texto `aria-live`/status; nomes claros para modal, inspector e botão de retorno.
- Indicadores de erro, conclusão e bloqueio não podem depender somente de cor.
- Quando o inspector fecha, foco retorna ao card correspondente; quando a operação conclui, foco retorna ao card na lane atualizada ou a substituto lógico se o item deixou o filtro atual.
- Evitar `button` com controles interativos aninhados. Separar a ação de abrir card, handle/menu e semântica acessível.
- Suportar preferência `prefers-reduced-motion` para animação de drag/drop, painel e modal.

### 10.2 Integridade de recuperação

- **Sem escrita no `dragstart`, `dragover`, `drop` ou abertura do inspector.** Escrita apenas após operação canônica confirmada.
- Bloqueios de saída já existentes continuam válidos enquanto houver submit pendente, draft ativo ou resultado incerto.
- Em conflito de versão, preservar texto do usuário e exigir revisão da versão nova antes de formar intenção diferente.
- Em resposta incerta, recuperar **mesmos bytes/body/key** em vez de refazer uma ação semanticamente diferente.
- Caso o comando tenha sido confirmado mas o refresh falhe, atualizar a projeção; não repetir a escrita.
- Leituras opcionais do mapa não podem bloquear preparação de forms legítimos.
- A invalidação por escopo/principal/credencial deve limpar UI, histórico contextual e respostas assíncronas antigas.

**Referências normativas:** WCAG 2.2, especialmente `2.5.7 Dragging Movements` (alternativa sem arrastar), `2.1.1 Keyboard`, `2.4.3 Focus Order`, `2.4.7 Focus Visible`, `2.5.8 Target Size (Minimum)` e padrões de diálogo do WAI-ARIA APG. A aprovação automática de Axe **não** substitui inspeção manual.

---

## 11. Plano de execução por fases e entregáveis

Implementar na **mesma linha evolutiva do PR #116**, por commits pequenos e observáveis, mantendo o PR em draft até aceite. As fases abaixo são dependências funcionais, não estimativas de calendário.

### Fase 0 — Congelar o contrato atual e registrar baseline **[P0]**

**Trabalho**

1. Confirmar commit HEAD, PR aberto, branch, cobertura existente e estado do CI.
2. Reproduzir as duas jornadas problemáticas: clique no card → página inteira; drag → fluxo com select/form/retorno.
3. Registrar capturas/DOM em HD, QHD, 4K e mobile; registrar viewport CSS, zoom e fonte dos dados, sem inventar métricas.
4. Catalogar comandos/permissions para cada tipo de card, separando legado e contrato assinado.
5. Auditar as funções `open`, `close`, `reset`, `navigate`, `inspect`, `move`, `act`, `render` e seus consumidores antes de migrar a superfície.
6. Executar as duas skills do repositório na análise inicial; registrar achados e critérios de aceite específicos.

**Entregáveis:** baseline versionado, matriz inicial das transições, testes existentes reproduzidos, inventário de contratos invariantes.

**Aceite:** cenário atual reproduzível com dados controlados; cada mudança futura consegue ser atribuída a um comportamento anterior.

### Fase 1 — Kanban como modo contextual do workspace **[P0]**

**Trabalho**

1. Desacoplar `createContextualKanban` de `dialog.showModal()`/`dialog.close()` como mecanismo da superfície principal.
2. Montar o Kanban como região dentro da área do Canvas, conservando o Canvas como visão padrão e acessível por retorno claro.
3. Preservar a pilha `frames`, origem, scroll por lane, filtros, buscas, revisão, seleção e `scopeKey`.
4. Migrar `onAction`, `operationHost`, `canLeaveOperation`, `finishOperation` e recuperação de foco **sem criar segundo formulário** nem alterar os comandos enviados.
5. Separar o container do Kanban do único `<dialog>` de operação, de modo que o formulário canônico possa ser efetivamente modal.
6. Validar a câmera antes/depois de abrir, operar e retornar ao Canvas, inclusive com read assíncrono atrasado.

**Entregáveis:** novo shell contextual Kanban, retorno ao Canvas, modal de operação reutilizado, testes atualizados de montagem/foco/câmera.

**Aceite:** abrir/fechar Kanban não altera dados nem câmera; operação aberta possui exatamente um modal nativo ativo; retorno a Canvas restaura seleção e foco.

### Fase 2 — Lanes tipadas, cards compactos e movimento **[P0]**

**Trabalho**

1. Implementar configuração separada de lanes para Objectives (3) e Tasks (5), com cancelados opcionais.
2. Resolver `backlog` e `todo` como lanes distintas sem criar novo lifecycle.
3. Garantir que Verification contenha somente itens com fato de revisão real reconhecido pelo backend.
4. Atualizar agrupamento, filtros, contadores, mensagens vazias e cards, mantendo sinalização de dados parciais.
5. Remover select `Move to…` por card; criar menu/controle acessível alternativo.
6. Tornar drag, possíveis destinos e feedback claros, com separação clique versus arraste.
7. Não exigir drag-and-drop para a única passagem possível de um trabalho.

**Entregáveis:** Kanban com lanes semânticas corretas, cards densos, alternativa sem drag, matriz de renderização e testes unitários.

**Aceite:** backlog→todo é realizável; Objective não apresenta fila de revisão inexistente; itens não mudam de lane antes de comando confirmado.

### Fase 3 — Motor de intenção e modal de transição **[P0]**

**Trabalho**

1. Refatorar `kanban-operations.js` para resolver intenções por entidade, lane atual, destino e capabilities existentes.
2. Incluir estados de preparação, bloqueio, handoff externo, submit, conflito, resultado incerto e refresh.
3. Reutilizar forms reais no modal contextual; não gerar handlers, payloads ou comandos paralelos.
4. Criar remediação inline: quando possível, iniciar o comando apropriado para adicionar critério, avaliação, prova, atribuição ou resolver impedimento e depois reavaliar a intenção original.
5. Tratar salto backlog→in_progress como etapas explícitas sequenciais, sem encadear comandos silenciosamente.
6. Tratar `Verification` como consulta/review, não como comando de mudança de lifecycle.
7. Executar validação final no backend, reconsultar a projeção após persistência e posicionar o card na lane real.
8. Manter handoff de contratos assinados e reaquisição/renew de leases legados estritamente separados.

**Entregáveis:** preparação da intenção, modal canônico, remediações, fluxos de sucesso e recuperação.

**Aceite:** nenhuma mudança de estado por soltura isolada; todos os comandos continuam seguindo validações/autorizações atuais; retentativa incerta não duplica efeitos.

### Fase 4 — Inspector compacto e operações de comprovação **[P1]**

**Trabalho**

1. Substituir ocultação total das colunas pelo inspector não modal.
2. Implementar `Overview`, `What to do`, `Responsibility & authority`, `Criteria & proof`, `Dependencies & impediments`, `Activity` e `More actions` por divulgação progressiva.
3. Mostrar apenas a próxima ação autorizada pertinente em destaque; outras ações agrupadas.
4. Permitir adicionar/inspecionar Criterion, Evidence, Artifact e Assessment em contexto, com owner e versão corretos.
5. Permitir inspecionar dependências, Issue e Blocker com operações distintas; preservar navegação de retorno.
6. Carregar registros e timeline sob demanda, rotulando cobertura e preservando o cursor histórico.
7. Abrir detalhe técnico completo somente por ação explícita, sem transformar todo clique em página longa.

**Entregáveis:** inspector responsivo e operacional, navegação por registros, lista de ações reduzida.

**Aceite:** as colunas continuam visíveis em desktop durante inspeção; a pessoa encontra objetivo, responsabilidade, pendências e comprovação sem sair do Kanban; não há prova fictícia.

### Fase 5 — Densidade, acessibilidade e robustez **[P1]**

**Trabalho**

1. Ajustar barras superiores, tipografia, espaçamento, largura das lanes, overflow, estados vazios e scroll.
2. Validar `1280×720`, `1366×768`, `1440×900`, `1920×1080`, `2560×1440`, `3840×2160`, `320×740`, `390×844` e viewport baixo resultante de zoom.
3. Testar zoom nativo 125%, 150%, 200% e 400%, teclado, menu alternativo, foco, leitor de tela manual e navegação por touch em dispositivo real quando disponível.
4. Testar nomes longos, 0/1/muitos cards, filtros ativos, tarefas com muitos critérios, conteúdo parcial e scroll profundo.
5. Testar conflitos, bloqueios, revogação, handoff, latência de read e recuperação incerta.
6. Atualizar CSS sem duplicações contraditórias no fim do stylesheet e eliminar seletores legados de modal não usado.
7. Capturar evidências antes/depois com fonte/runtime/viewport claros, atualizar docs e executar regressão.

**Entregáveis:** galeria de telas, relatório de acessibilidade (escopo declarado), matriz de UX, testes completos, documentação atualizada.

**Aceite:** nenhuma regressão funcional conhecida no conjunto executado; falhas de foco, overlay, scroll e gestos cobertas por testes determinísticos; pendências manuais registradas sem alegar conformidade integral.

---

## 12. Arquivos prováveis e responsabilidade da mudança

> A lista é um **mapa de intervenção**, não autorização para modificar indiscriminadamente todos os arquivos. Confirmar dependências na branch antes de editar.

| Arquivo ou área | Alteração esperada |
|---|---|
| `packages/wos-api/web/assets/workspace-kanban.js` | Desacoplar shell modal; manter frames, renderizar lanes/cards/inspector simultaneamente, coordenar seleção, drag, operação e foco. Principal arquivo do refactor. |
| `packages/wos-api/web/assets/workspace-kanban.css` | Layout de workspace, lanes, cards, drawer e responsividade; remover regras contraditórias do modal antigo. |
| `packages/wos-api/web/assets/kanban-operations.js` | Preparação pura de intenção por tipo, capabilities e destino; remediação, handoff e feedback, sem replicar domínio. |
| `packages/wos-api/web/assets/app.js` | Montagem do novo modo contextual, passagem do host de operação, navegação, guarda de saída e integração com o inspector/form canônico. |
| `packages/wos-api/web/assets/workspace-view.js` | Facetas do inspector, agrupamento de ações por intenção, seções de prova/autoridade e navegação contextual. |
| `packages/wos-api/web/assets/workspace-view.css` | Layout do inspector compacto e painéis auxiliares. |
| `packages/wos-api/web/assets/canvas.css` / `workspace-canvas.js` / `workspace-surfaces.css` | Somente integração de modo e câmera/seleção, sem modificar coordenadas espaciais ou comportamento de zoom desnecessariamente. |
| `packages/wos-api/web/assets/navigation.js` | Integrar retorno/navegação contextual, **se** o sistema atual de navegação exigir; evitar rota inventada. |
| `packages/wos-core/application/workspace_stage.go` | Preferir preservar contrato atual; alterar somente após análise de consumidores, ou adicionar projeção aditiva quando justificado. |
| `packages/wos-core/application/workspace_cards.go` / `workspace_read_types.go` | Alterar somente se leitura existente não sustentar os novos estados de apresentação/eligibilidade; garantir compatibilidade. |
| `packages/wos-core/application/workspace_capabilities.go` | Ampliar explicações/remediações de leitura apenas se necessário, respeitando guardas canônicas atuais. |
| `tests/web/unit/kanban-operations.test.mjs` | Matriz de intenções, entidade/destino, etapas inexistentes, bloqueios, handoff e remediações. |
| `tests/web/kanban-operations.spec.mjs` | Jornada real com drop, modal canônico, confirmação, persistência, critérios, conclusão e retorno. |
| `tests/web/contextual-kanban.spec.mjs` | Navegação, hierarquia, scroll, foco, viewport, cards e responsividade do novo shell. |
| `tests/web/contextual-kanban-guards.spec.mjs` | Segurança de escopo/revisão, credencial revogada, cancelamentos e leituras assíncronas. |
| `tests/web/accessibility.spec.mjs` / `operational-density.spec.mjs` | Alternativa a drag, modal/foco, targets, scroll e layout em viewport extremo. |
| `docs/ux/operational-kanban/` | Especificação, auditoria atualizada, exemplos operacionais, screenshots e resultados de aceitação. |
| `docs/adr/0039-contextual-kanbans-and-strict-completion.md` | Registrar a evolução da decisão de apresentação mantendo imutáveis as decisões de domínio/completion policy; preferir novo ADR curto se necessário. |

**Atenção ao legado de testes:** o conjunto atual assume `#workspace-kanban` como `<dialog>` e espera `dialog[open]` igual a 1 durante form hospedado internamente. Atualizar as asserções **porque a UX mudou**, não desabilitar cenários. Na experiência final, o Kanban não é modal; o formulário operacional é o modal, com isolamento de foco real.

---

## 13. Matriz mínima de cenários para aceitar a implementação

| Cenário | Ação | Resultado verificável |
|---|---|---|
| A01 | Abrir Kanban a partir de Outcome no Canvas | Tipo e pai corretos; Canvas preservado para retorno. |
| A02 | Abrir Kanban de Objective; alternar Tasks/Subobjectives | Somente filhos diretos; nenhum descendente misturado silenciosamente. |
| A03 | Clicar Task | Inspector abre sem esconder ou desmontar lanes; 0 comandos de escrita. |
| A04 | Trocar Task selecionada | Inspector atualiza; filtros e scroll preservados. |
| A05 | Arrastar Task `backlog` para `To do` | Prepara `ActivateWorkItem`; sem write até confirmação; ao persistir aparece em `To do`. |
| A06 | Arrastar `todo` para `In progress` com lease legado elegível | Abre comando de claim; servidor concede autoridade; UI reflete `in_progress`. |
| A07 | Arrastar `todo` para `In progress` com dependência impeditiva | Bloqueio + remediação; 0 comandos não autorizados, card não muda. |
| A08 | Arrastar `backlog` diretamente para `In progress` | Duas etapas explicadas; cada comando exige intenção válida e resultado persistido. |
| A09 | Arrastar Objective `planned` para `In progress` | Preparar `StartObjective` real; critério/Outcome/Blocker revalidados. |
| A10 | Arrastar Objective `planned` para `Achieved` quando elegível | `AchieveObjective` legítimo sem Start forçado. |
| A11 | Arrastar Objective sem requisitos para `Achieved` | Mostrar requisitos/criteria; não alterar lifecycle. |
| A12 | Arrastar Task `in_progress` para Verification sem review formal | Abrir inspeção de provas; **não** inventar estado `verification`. |
| A13 | Tarefa com review pendente real | Aparece em Verification; inspeção não aceita nem assina revisão. |
| A14 | Arrastar Task para Accepted com assessment pendente | Fluxo aponta critério concreto; conclusão bloqueada. |
| A15 | Registrar Artifact e Evidence; avaliar Criterion | São operações distintas; Evidence não aprova automaticamente Criterion. |
| A16 | Complete Task autorizado | Conclusão persiste e card muda; Objective não é automaticamente alcançado. |
| A17 | Task contratual assinada | Handoff correto, nenhuma assinatura/aquisição falsa em browser. |
| A18 | Lease expirado / usuário não detentor | Renovar/reivindicar somente conforme authority/fencing; sem auto-claim indevido. |
| A19 | Issue resolvida com Blocker ativo | Blocker permanece até liberação explícita. |
| A20 | Alteração concorrente durante modal | `conflict` preserva edição e exige revisão da intenção. |
| A21 | Comando persistido com resposta perdida | Recuperação da intenção original sem duplicar efeito. |
| A22 | Comando persistido com refresh falho | Refaz leitura, não reexecuta escrita. |
| A23 | Sair e retornar ao Canvas | Câmera, seleção e foco restaurados, inclusive sob read atrasado. |
| A24 | Alterar escopo/principal/revogar credencial | Kanban fecha/limpa; resposta antiga não ressuscita dados. |
| A25 | Navegação por teclado sem drag | Mesmo conjunto de intenções e requisitos, foco coerente e anúncios. |
| A26 | Toque em dispositivo móvel | Menu alternativo funcional; nenhum requisito exclusivo de hover/drag. |
| A27 | Zoom nativo 400% e viewport baixo | Modal, ação principal, saída e último campo acessíveis sem sobreposição. |
| A28 | Busca/paginação com cobertura parcial | Contagens corretamente rotuladas; revisão incompatível impede merge de páginas. |
| A29 | `strict_work` com filhos abertos/cancelados | Comportamento canônico preservado, sem aceitação por drag. |
| A30 | Múltiplos records/longos títulos/dados densos | Sem overflow horizontal do documento, cards e controles legíveis. |

**Verificação mínima:** combinar unit tests de intenção, Playwright com backend real e persistência observada, testes de Application/Domain, Axe como triagem, inspeção manual de teclado/zoom/touch/leitor de tela e capturas comparativas. Não substituir cenários reais por mocks visuais.

---

## 14. Critérios de conclusão do PR

O trabalho está pronto para revisão final somente quando **todos** os itens abaixo forem verdadeiros:

- [ ] O Canvas continua a visão espacial principal; o Kanban funciona como modo contextual, sem criar entidade persistida.
- [ ] Clicar no card abre inspector compacto e preserva lanes, busca, filtros, scroll e retorno.
- [ ] A Task distingue `backlog`, `todo`, `in_progress`, revisão real e `done` sem status fictícios.
- [ ] O Objective possui lanes alinhadas ao seu lifecycle, sem Verification artificial.
- [ ] Nenhum select permanente `Move to…` aparece sob os cards; existe alternativa equivalente sem drag.
- [ ] Drop prepara operação e nunca dispara mudança de lifecycle diretamente.
- [ ] O modal apresenta intenção, elegibilidade, requisitos, remediação e formulário canônico; não há empilhamento de diálogos nativos.
- [ ] Nenhum fluxo inventa autoridade, Agent online, assinatura, review, prova, conclusão ou aceite.
- [ ] Critérios, Evidence, Artifact, Assessment, Issue e Blocker mantêm semânticas distintas e ações operacionais descobríveis.
- [ ] Comandos de backend e seus guardas de autorização, revisão, fencing, idempotência e contratos são preservados.
- [ ] Conflitos e respostas incertas preservam dados/intenção e não produzem gravações duplicadas.
- [ ] As suites unitárias, browser, Go/application e CI relevantes passam após **atualização legítima** das expectativas visuais.
- [ ] HD/QHD/4K/mobile/zoom nativo/teclado/touch foram executados e documentados no alcance possível.
- [ ] Telas antes/depois e evidências de comportamento estão versionadas em `docs/ux/operational-kanban/`.
- [ ] Identidade visual e tokens existentes do WOS foram mantidos.
- [ ] O PR continua em draft até revisão explícita; sem merge, release, tag ou publicação automática.

---

## 15. Ordem operacional recomendada para o agente construtor

1. **Ler:** README, ADRs, as duas skills, os arquivos de UI e os testes existentes no HEAD atual; registrar qualquer divergência entre este plano e a branch.
2. **Formalizar:** matriz de lanes, intenções, capabilities e garantias de negócio, antes de mudar CSS.
3. **Refatorar shell:** retirar o Kanban do modal e montar como contexto de trabalho sem quebrar a navegação do Canvas.
4. **Implementar lanes/cards:** separar Objective de Task, Backlog de To do; substituir select por menu acessível.
5. **Implementar preparação de transição:** drop → leitura/capability → modal/requirements → comando canônico → refresh.
6. **Implementar inspector:** renderização simultânea com board, overview e facetas operacionais.
7. **Cobrir exceções:** permissões, leases, contratos assinados, conflito, revisão, idempotência, execução externa, blocos e escopo.
8. **Verificar visual e acessibilidade:** todos os viewports/zoom e estados representativos.
9. **Consolidar documentação:** auditoria baseada em evidência, screenshots, testes executados, limites declarados.
10. **Entregar para revisão:** resumo de arquivos alterados, fluxo antes/depois, matriz A01–A30 com resultados, logs de teste e pendências ainda abertas.

Não antecipar UI definitiva antes de validar os contratos de domínio; não abrir nova camada de backend só para acomodar um atalho visual; não marcar transição como concluída antes da resposta persistida do WOS.

---

## 16. Referências verificadas e documentos de apoio

**Código e documentação do WOS (branch examinada):**

- [PR #116](https://github.com/A1b3rt0M3rcad0/wos/pull/116)
- [`workspace-kanban.js`](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/packages/wos-api/web/assets/workspace-kanban.js)
- [`kanban-operations.js`](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/packages/wos-api/web/assets/kanban-operations.js)
- [`workspace-kanban.css`](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/packages/wos-api/web/assets/workspace-kanban.css)
- [`workspace_stage.go`](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/packages/wos-core/application/workspace_stage.go)
- [`workspace_capabilities.go`](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/packages/wos-core/application/workspace_capabilities.go)
- [`workspace_cards.go`](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/packages/wos-core/application/workspace_cards.go)
- [ADR-039](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/docs/adr/0039-contextual-kanbans-and-strict-completion.md)
- [Auditoria operacional](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/docs/ux/operational-kanban/audit.md)
- [Documentação de operação](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/docs/ux/operational-kanban/README.md)
- [Qualificação e limites de evidência](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/docs/ux/operational-kanban/validation.md)

**Padrões para validar durante implementação:**

- [WCAG 2.2 — Dragging Movements](https://www.w3.org/WAI/WCAG22/Understanding/dragging-movements.html)
- [WCAG 2.2 — Keyboard](https://www.w3.org/WAI/WCAG22/Understanding/keyboard.html)
- [WCAG 2.2 — Focus Order](https://www.w3.org/WAI/WCAG22/Understanding/focus-order.html)
- [WCAG 2.2 — Target Size (Minimum)](https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html)
- [WAI-ARIA APG — Dialog (Modal)](https://www.w3.org/WAI/ARIA/apg/patterns/dialog-modal/)

**Status da pesquisa:** as observações sobre o código e a arquitetura são fundamentadas na branch e nas capturas do usuário. Os novos padrões de UX são uma proposta de engenharia a ser qualificada por testes; não existem métricas de ganho de produtividade ou pesquisa independente com participantes atribuídas a este plano.
