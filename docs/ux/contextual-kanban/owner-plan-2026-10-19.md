# WOS — Plano de implementação: Board preservado, Kanbans contextuais e inspector operacional

> **Status:** proposta detalhada para análise e aprovação; **não implementada neste documento**.  
> **Data de elaboração:** 19/10/2026.  
> **Produto:** WOS — Work Outcome System.  
> **Repositório:** [`A1b3rt0M3rcad0/wos`](https://github.com/A1b3rt0M3rcad0/wos).  
> **Base estudada:** [PR #116](https://github.com/A1b3rt0M3rcad0/wos/pull/116), branch `fix/workspace-operational-density`, commit `184be7c32193a9e1369338128302088a0e37144d` (draft na consulta de 19/10/2026).  
> **Âmbito:** experiência humana no navegador, contratos de leitura/ações, projeções e **somente** as evoluções de domínio necessárias.  
> **Diretriz visual:** manter a linguagem editorial e os assets, robôs, post-its, tipografia, cores e o motor do Board existentes.  
> **Destinatário:** responsável pelo produto, designer de interação e agente de implementação.

---

## 0. Decisão fundamental — LEIA ANTES DE IMPLEMENTAR

**Não substituir o Board por um Kanban. Não converter o Canvas em um painel comum.**

O **Outcome Board já é o mapa principal do trabalho**: representa o Outcome, os Objectives, as Tasks e suas conexões. Ele deve continuar existindo com a mesma identidade visual, motor de layout, pan/zoom, controles e navegação. A alteração proposta é **a interação com cada post-it**:

1. Um clique normal em um post-it abre uma **barra lateral compacta** — um **inspector operacional contextual**.
2. Quando o post-it representa um **Outcome**, a ação principal da lateral é **Abrir Kanban de objetivos**.
3. Quando representa um **Objective**, a ação principal é **Abrir Kanban de tarefas**.
4. O Kanban abre como **um modal grande dentro da experiência do Board**, sobre o Canvas. Não navega para outra página; não cria outro objeto de domínio `Kanban`.
5. A partir do modal, o usuário administra trabalho, abre detalhes e avança ao quadro do próximo nível **dentro do mesmo modal**, com breadcrumbs; **nunca empilha modais Kanban**.
6. Quando representa uma **Task**, a lateral prioriza as ações reais de execução, comprovação, entrega e revisão, **sem Kanban próprio**.
7. **Issue** e **Blocker** abrem inspeções específicas de resolução e impacto; não são automaticamente convertidos em Tasks nem criam novas hierarquias.
8. Informações documentais (Criteria, Evidence, Artifacts, Assessments, Conclusions, Contracts, Decisions, histórico) **não são cards no mapa de trabalho principal**. Permanecem nas seções da lateral e na camada documental já existente.
9. A lateral possui **somente dois grupos expansíveis principais**: **Detalhes e gerenciamento** e **Histórico de atividade**. Dentro do primeiro há subtópicos compactos; não adicionar cinco novos painéis globais permanentes.
10. A UI **não contorna** regras de autoridade, contratos, assinaturas, revisões, dependências, bloqueios ou aceitação do WOS. Ela apresenta esses mecanismos como ações contextualizadas.

**Resultado esperado:** um usuário pode planejar, executar, comprovar, verificar, corrigir e concluir sem conhecer previamente o vocabulário interno de todas as entidades. Um agente e uma integração continuam usando os contratos reais do WOS.

### 0.1. Frase-guia do produto

> O Board mostra **como tudo se conecta**. O Kanban mostra **em que etapa está cada trabalho**. A lateral mostra **o que é, o que falta e o que posso fazer agora**.

### 0.2. Não fazer

- Não trocar o mapa por um Kanban como visualização padrão.
- Não reproduzir a lateral atual, longa e indistinta, dentro de um modal.
- Não criar `Kanban`, `Column`, `Card` ou `BoardItem` como novas entidades persistidas só para a apresentação.
- Não inferir dependência a partir de posição, setas decorativas ou ordem visual.
- Não fazer drag & drop alterar lifecycle sem passar pelo comando, protocolo e autorização corretos.
- Não tratar `Evidence` como aprovação, `Artifact` como evidência automaticamente validada, `Issue` como `Blocker`, `Task done` como `Objective achieved` ou `Objective achieved` como `Outcome achieved`.
- Não aceitar automaticamente todos os critérios porque um usuário marcou um checklist.
- Não inventar status de revisão formal quando não existe registro correspondente no backend.
- Não trocar uma operação de revisão assinada por um POST administrativo do navegador.
- Não fazer upload real de arquivos sem existir serviço de armazenamento, autenticação, limites e retenção explicitamente implementados.
- Não alterar silenciosamente o comportamento de clientes HTTP/MCP/CLI/SDK existentes.

---

## 1. Objetivo e situação atual

### 1.1. Problema de experiência

No uso atual, ao selecionar um post-it o usuário encontra uma quantidade elevada de dados técnicos: definições, pessoas, critérios, evidências, relações, autoridade, histórico, opções e ações. No exemplo do Board analisado, a lateral chega a mostrar uma sequência extensa de eventos enquanto o trabalho mais relevante compete por atenção com várias operações administrativas.

O fluxo exige conhecimento desnecessário sobre `Assessment`, `EvidenceLink`, `Conclusion`, `WorkItem`, `Blocker`, submissões e versões. Apesar de esses conceitos serem relevantes para o mecanismo do WOS, **não precisam ser a primeira interface cognitiva**.

O fluxo pretendido é:

- **Descobrir** a situação do item ao clicar nele.
- **Compreender** a próxima obrigação sem buscar em várias telas.
- **Agir** com uma única ação principal coerente com permissões e estado.
- **Aprofundar** quando necessário, expandindo Detalhes ou Histórico.
- **Gerenciar** os itens filhos por um Kanban contextual, sem perder o mapa.
- **Validar** entrega e condições de sucesso em uma jornada guiada.

### 1.2. O que já existe e deve ser reaproveitado

| Componente/capacidade identificada | Situação na base estudada | Uso nesta evolução |
|---|---|---|
| Outcome Board com mapa, conexões, pan/zoom e post-its | Implementado | **Preservar**; única alteração é interação/painel/modal. |
| `Outcome`, `Objective`, `WorkItem` (Task) | Implementados | Continuam sendo os principais cards operacionais. |
| Hierarquia de subobjetivos, Tasks diretas e dependências hard/advisory | Implementada | Usar como origem dos Kanbans e dos indicadores de requisito. |
| `Issue`, `Blocker`, propagação direta/subtree | Implementados | Painéis especializados, não confundir problema e impedimento. |
| `SuccessCriterion`, revisões e `CriterionAssessment` | Implementados | Checklist visual de critérios com avaliações reais. |
| `Evidence`, `EvidenceLink`, `Artifact` | Implementados | Área contextual de materiais, relação com entregas e critérios. |
| `Conclusion`, submissões e revisão independente | Implementados em seus protocolos aplicáveis | Preservar decisão explícita de aceitação. |
| `WorkAttention` e estado operacional derivado | Implementados | Produzir “o que falta fazer”, indicadores e ação seguinte. |
| `EntityNote`/WorkActivity, timeline paginada | Implementados | Histórico compacto, sob demanda, com autores e proveniência. |
| `workspace-view`, `workspace-cards`, `workspace-timeline` | Implementados no PR #116 | Reusar, estender projeções quando necessário. |
| Inspector reutilizado entre Board e vistas especializadas | Evoluído no PR #116 | Tornar compacto e orientado a ações sem duplicar regras. |
| Criação rápida/batch e edição contextual | Evoluídas no PR #116 | Reusar criação mínima e contexto do pai. |
| Task Board separado em Work | Existente | Manter como vista especializada; **não** confundir com o novo modal contextual. |
| Camada documental `records` e mapa de trabalho `work` | Existentes | Trabalho como default; documentos exploráveis apenas quando solicitado. |

Fontes: [Arquitetura](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/docs/architecture.md), [guia do PR](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/docs/ux/operational-workspace/README.md), [contratos de Workspace](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/docs/operational-workspace-api.md), [WorkActivity](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/docs/work-activity.md).

### 1.3. Distinção entre implementado e proposto

- **Já implementado**: domínio, endpoints de leitura contextual, camada documental, ações de critérios/material, mapa, inspector e Kanban especializado de Tasks.
- **Novo na proposta**: dois painéis expansíveis padronizados, ações primárias por tipo/situação, Kanbans contextuais modais para Outcome e Objective, navegação hierárquica no modal, checklist guiado de verificação, entrada unificada de materiais, painel de pendências/correções, política opcional de conclusão estrita e suas projeções de elegibilidade.
- **Possível evolução adicional, não requisito da primeira entrega**: solicitação formal durável de verificação de Outcome/Objective, com fila própria de revisores. Até existir, não chamar um simples rascunho de “revisão solicitada”.

---

## 2. Modelo mental e regras invariantes

### 2.1. Cinco conceitos humanos fundamentais

| Conceito | Pergunta que responde | Relacionamento operacional |
|---|---|---|
| **Outcome** | Qual resultado final queremos entregar? | Possui critérios finais e Objectives; pode possuir Tasks diretas. |
| **Objective** | Qual parte verificável do resultado precisamos alcançar? | Pertence ao Outcome, pode ter Objective pai, Tasks e critérios próprios. |
| **Task** | Qual trabalho concreto deve ser executado? | Pertence a um Objective ou diretamente ao Outcome; possui execução e possível revisão. |
| **Issue** | Qual problema foi encontrado e precisa de solução? | Afeta entidades; pode originar Blockers e trabalho corretivo. |
| **Blocker** | O que está impedido e por quê? | Tem alvo explícito e propagação, não se confunde com Issue. |

### 2.2. Conceitos de apoio — visíveis, mas não cards principais

- **Criterion**: condição que precisa ser avaliada para aceitar o trabalho ou resultado.
- **Assessment**: avaliação autorizada de uma versão específica de Criterion (`met`, `not_met`, `inconclusive`, `waived`), com razão e fundamento.
- **Evidence**: observação/fonte usada na verificação, favorável, contrária ou contextual conforme o **EvidenceLink**.
- **Artifact**: material produzido/referenciado: código, commit, PR, documento, relatório, arquivo externo etc.
- **Submission/Delivery**: versão de entrega submetida pelo protocolo de execução.
- **Conclusion**: decisão explícita de conclusão, vinculada às obrigações satisfeitas.
- **Relation/Dependency**: relacionamento persistido. Só `depends_on` hard cria um pré-requisito de readiness.
- **EntityNote/WorkActivity**: documentação imutável de atualização/execução, conforme identidade e protocolo.
- **Roadmap**: plano versionado opcional, sem implicar automaticamente dependências operacionais.
- **Contract/Lease/Principal**: autoridade de execução; nunca inferir de assignee ou card arrastado.

### 2.3. Invariantes obrigatórias

1. Uma nota não executa, aceita ou conclui trabalho.
2. Registrar ou vincular Evidence não executa Assessment.
3. Um Assessment não altera automaticamente o lifecycle do item pai.
4. Critério `met` exige avaliação legítima na versão aplicável; `waived` significa exceção autorizada, **não sucesso verificado**.
5. Avaliação de submissão anterior/digest anterior não aceita submissão posterior.
6. Decisões devem preservar autor, escopo, versões e idempotência.
7. Direitos são decididos pelo backend e identidade atual, não por botões apresentados.
8. Owner/assignee representa responsabilidade declarada; executor autorizado, contrato ativo e revisor são conceitos diferentes.
9. `Issue` pode estar aberta sem bloquear; `Blocker` pode continuar ativo após a resolução da Issue até liberação explícita.
10. A interface pode usar rótulos simples, mas deve manter os resultados canônicos do domínio.

---

## 3. Arquitetura da experiência

### 3.1. Navegação de alto nível

```text
Outcome Board (mapa original) — CONTINUA COMO ESTÁ
  ├── Clique Outcome
  │     └── Inspector compacto → [Abrir Kanban de objetivos]
  │                                  └── Modal: quadro de Objectives
  │                                        ├── Selecionar Objective → detalhes no próprio modal
  │                                        └── Abrir tarefas → mesmo modal, contexto Objective
  ├── Clique Objective
  │     └── Inspector compacto → [Abrir Kanban de tarefas]
  │                                  └── Modal: quadro de Tasks daquele Objective
  ├── Clique Task
  │     └── Inspector de execução/entrega → ação seguinte
  ├── Clique Issue/Blocker (camada documental ou contexto)
  │     └── Inspector de problema/impedimento
  └── Selecionar ligação
        └── Contexto da relação, direção e ação permitida
```

As vistas especialistas já existentes **não são removidas**. A nova jornada é a navegação mais direta para o Board, não uma substituição de Plan, Work, Issues, Evidence ou Activity.

### 3.2. Modos da lateral

**Modo compacto inicial:** cabeçalho + estado + resumo + próxima ação + até três ações relevantes + dois grupos recolhidos. **Modo de aprofundamento:** os grupos se expandem e carregam os dados adequados, com rolagem interna. **Modo de formulário:** edição/registro pontual dentro de superfície existente; preserva rascunho; navegação de retorno clara.

No Board, o inspector mantém largura contida e não interfere nos controles do Canvas. Nos viewports compactos, usa overlay sobre o palco do mapa (não sobre o cabeçalho) e corpo rolável; em telas largas pode ser docked conforme a lógica existente.

### 3.3. Dois grupos expansíveis universais

**Grupo 1 — Detalhes e gerenciamento:** subtópicos:
- **Critérios de aceitação**: listar, criar, revisar quando permitido, consultar avaliação e ir para Verificar critérios.
- **Materiais**: **Artefatos** e **Evidências**, duas listas distintas em um mesmo agrupamento; adicionar, consultar, vincular, retirar quando permitido.
- **Dependências e relações**: pré-requisitos, dependentes, direção, natureza hard/advisory e acesso ao alvo. Aparece apenas onde aplicável.
- **Problemas e impedimentos**: Issues, Blockers e pendências de correção associados.
- **Participantes e informações avançadas**: responsabilidade declarada, autoridade real, contexto técnico, conclusão e outros campos; inicialmente recolhidos, sem misturar credenciais/identidade.

**Grupo 2 — Histórico de atividade:** timeline de eventos relevantes, notas, marcos de execução, entregas, revisões e alterações, com autor, origem e tempo; registro de atualização quando permitido; paginação “Carregar anteriores”.

Não abrir ambos integralmente por padrão. Os contadores resumidos podem vir no mesmo snapshot; a lista completa só é carregada sob demanda. Estado `unknown/not_loaded/partial/unavailable` deve aparecer honestamente, nunca como zero.

---

## 4. Padrão detalhado do inspector lateral

### 4.1. Anatomia exata

```text
┌──────────────────────────────────────┐
│ OBJECTIVE                        [×] │
│ Implementar API de pagamentos        │
│ [Em execução]                        │
│                                      │
│ O QUE É                              │
│ Entregar os endpoints ...            │
│                                      │
│ PROGRESSO                            │
│ 4/6 Tasks aceitas; 2/3 critérios     │
│ Aguardando uma verificação           │
│                                      │
│ PRÓXIMA AÇÃO RELEVANTE               │
│ [Abrir Kanban de tarefas]            │
│ [Adicionar Task] [Verificar objetivo]│
│                                      │
│ ▸ Detalhes e gerenciamento           │
│     Critérios, materiais, relações,  │
│     problemas e participantes         │
│                                      │
│ ▸ Histórico de atividade             │
│                                      │
│ Mais ações  ·  Editar                │
└──────────────────────────────────────┘
```

**Hierarquia de prioridade visual:** (1) identidade e estado; (2) frase de significado; (3) progresso e próximo impedimento; (4) CTA primário; (5) ações rápidas; (6) detalhes; (7) histórico; (8) operações avançadas. O histórico não pode ocupar toda a lateral no estado inicial.

### 4.2. Regras globais de ação

- Sempre **uma ação primária**. Pode ser “Abrir Kanban”, “Continuar trabalho”, “Enviar entrega”, “Revisar” ou “Resolver impedimento”, de acordo com o tipo e o estado.
- Até **duas ou três ações secundárias** visíveis; demais ações em “Mais ações”, agrupadas como Planejar, Verificar, Relacionar, Problemas, Registros, Administração.
- Botão sem permissão não deve induzir o usuário a pensar que poderá operar. Omissão ou indisponibilidade explicada por capability/razão tipada; se houver remediação possível, ação contextual correspondente.
- A ação “Verificar” não pode ser sinônimo de “Concluir”: verificação gera Assessment; conclusão é ato separado.
- Não mostrar “Em execução” somente porque existe assignee. Quando relevante, exibir “Responsável declarado”, “Executor autorizado”, “Revisor”, “Último executor”.
- Em estados terminais, priorizar “Ver resultado e comprovações” e “Histórico”; “Reabrir” fica em operação legítima, quando permitida.
- Mensagens de validação devem orientar a correção: “Falta definir uma condição de sucesso”, “Dependência X ainda não aceita”, “É necessário revisar a entrega atual”. Nunca exibir `precondition_failed` bruto ao usuário como explicação principal.

### 4.3. Regras de conteúdo

- Cabeçalho sempre expõe tipo, nome, estado real ou derivado devidamente rotulado e acesso ao parent/breadcrumb.
- Descrição mostra uma síntese (com “ver descrição completa”). Nunca truncar sem expandir.
- Progresso sempre declara **o denominador** e o critério: `4/6 Tasks aceitas`, não um percentual arbitrário da organização.
- Exibir contadores apenas se coverage for completo para a respectiva métrica. Se incompleto: “4 de pelo menos 6 carregadas” / “mais registros disponíveis”.
- Pendências: mostrar **a primeira obrigação acionável** e “ver todas”, a partir das projeções do servidor.
- Cada subtópico deve conter ações locais, sem obrigar a voltar ao topo.
- As seções conservam expandido/recolhido para a entidade selecionada durante a sessão, mas uma nova seleção começa limpa, salvo preferência explícita.

---

## 5. Post-its — conteúdo e ações específicos

### 5.1. Outcome — resultado final

**Resumo inicial:** título; desired state; lifecycle (draft/active/achieved/failed/abandoned); contagem total e aceita de Objectives obrigatórios; Tasks diretas; critérios finais atendidos/pendentes; blockers relevantes; pendência dominante. Informar se há objetivos opcionais não contabilizados como obrigação.

**CTA principal:** `Abrir Kanban de objetivos` (disponível mesmo em Outcome draft, com conteúdo planejado; não pressupõe execução ativa).

**Ações secundárias:** `Adicionar objetivo`, `Adicionar Task direta` ou `Verificar entrega` conforme elegibilidade. Na preparação de draft, priorizar `Definir condição de sucesso`/`Ativar Outcome` se faltante; o Kanban continua encontrável e não escondido.

**Detalhes e gerenciamento:**
- Critérios finais do Outcome e respectivos Assessments.
- Evidências e artefatos do resultado final, agrupados sem mistura semântica.
- Objectives obrigatórios (inclusive fora da parte carregada do mapa) e opcionais; acesso a cada um.
- Tasks diretas e seu progresso; relação com policy de conclusão.
- Issues/Blockers que afetam o Outcome; origem e propagação.
- Responsáveis declarados, metadados, roadmap associado (opcional).
- Prévia do impacto da conclusão e dos itens abertos.

**Histórico:** preparação, ativação, alterações, critérios, assessments, mudanças e conclusão/reabertura, com histórico de descendentes quando explicitamente solicitado e origem corretamente identificada.

**Estados especiais:** `achieved` → CTA `Ver entrega aceita`; `failed/abandoned` → `Ver decisão`; `draft` → permitir planejamento, impedir concluir; `archived` → somente as ações autorizadas e leitura aplicável. Não expor dependência hard de Outcome, que não é o mesmo relacionamento de WorkItem/Objective.

### 5.2. Objective — parte verificável do resultado

**Resumo:** título, objetivo pai (se houver), lifecycle, se é obrigatório para o Outcome, progresso das Tasks diretas/descendentes conforme escopo declarado, estado dos subobjetivos, critérios próprios, dependências hard ainda insatisfeitas, impedimentos e quem responde pelo objetivo.

**CTA principal:** `Abrir Kanban de tarefas` — inclusive para Objective `planned`.

**Ações secundárias:** `Adicionar Task`, `Adicionar subobjetivo`, `Iniciar objetivo` ou `Verificar objetivo` conforme disponibilidade. Não ocupar o painel com todas simultaneamente.

**Detalhes e gerenciamento:**
- Critérios próprios e avaliação.
- Evidências/artifacts produzidos no escopo do objetivo.
- Dependências explícitas (`A depende de B`); distinguir dependência de hierarquia.
- Subobjetivos e Tasks; links para abertura contextual.
- Issues e Blockers relacionados, diretos ou propagados.
- Responsáveis, informações editáveis e prévia do impacto de conclusão.

**Histórico:** alterações de planejamento, início, associação/movimentação de Tasks, atividades pertinentes, avaliações, aceitação e reabertura. Uma Task filha não vira um “evento realizado pelo Objective”; identificar sujeito real.

**Estados especiais:** `planned` pode receber plano e ser aberto; `in_progress` pode ser executado; `achieved` aceita e mostra prova; `cancelled` é terminal, não equivale a aceito. Objetivos podem hoje ser concluídos sem passar por `in_progress` se os gates canônicos permitirem; não criar etapa Start obrigatória apenas pela UI.

### 5.3. Task (`WorkItem`) — execução concreta

**Resumo:** título, descrição/instrução de execução, parent, lifecycle, display state (`ready`, `blocked`, `waiting_dependencies`, `awaiting_review` etc., apenas se aplicável e com fonte correta), current executor quando real, assignee quando apenas declarado, critérios, artefatos, evidências, última atividade, próxima obrigação e estado do contrato ou lease.

**CTA principal dinâmica:**

| Situação observada | Próxima ação apresentada |
|---|---|
| `backlog` | `Preparar trabalho` ou `Disponibilizar`, se autorizado. |
| `todo` e pronta | `Assumir/Iniciar trabalho` apenas pelo canal de autoridade permitido. |
| `todo` com dependência | `Ver pré-requisito` e `Resolver pendência`, não iniciar indevidamente. |
| `in_progress` com autoridade válida da identidade | `Registrar avanço` ou `Enviar entrega`, conforme obrigações. |
| `in_progress` com autoridade de outro participante | `Ver andamento`; não permitir ações de executor. |
| Assinada, `pending_review` verdadeiro | `Ver revisão` ou `Revisar entrega` para revisor autorizado; exibir pendência ao executor. |
| Correções solicitadas | `Ver correções` e fluxo legítimo de aquisição/correção. |
| Lease/contrato vencido ou incerto | `Reconciliar/Retomar` conforme protocolo; não fingir execução ativa. |
| `done` | `Ver resultado aceito` / `Ver comprovações`. |
| `cancelled` | `Ver cancelamento`; reabrir apenas quando permitido. |

**Ações secundárias:** `Adicionar material` (Artifact ou Evidence), `Verificar critérios` (somente avaliação autorizada; caso contrário, `Preparar comprovações`), `Reportar problema`, `Editar instruções` se ainda permitido.

**Detalhes:**
- O que fazer e como validar.
- Critérios com estado por revisão e cobertura.
- Artifacts produzidos e Evidence vinculadas com stance e origem.
- Dependências, impedimentos, autoria, contrato/lease e participação.
- Resumo de entrega atual, revisão pendente, findings e próximos passos.
- Contexto técnico avançado (spec, versões/digests) apenas em detalhes técnicos.

**Histórico:** progresso, checkpoints, evidências registradas, submissions, reviews, findings, conclusão, retorno e alterações.

**Não há `Abrir Kanban` em Task**: ela é folha operacional. O botão de retorno leva ao Kanban do Objective ou à área de Tasks diretas do Outcome.

### 5.4. Issue — problema

**Resumo:** descrição do defeito, severidade, lifecycle, entidades afetadas, pessoa que reportou, questões em investigação, decisões de resolução e eventual trabalho corretivo relacionado.

**CTA:** `Investigar`, `Resolver problema` ou `Ver resolução`, conforme estado e permissão.

**Ações:** `Criar/vincular Task corretiva` no escopo legítimo; `Impedir trabalho` via criação de Blocker explícito com prévia de impacto; `Registrar atualização`; `Marcar duplicada`/`Não corrigir` quando legítimo.

**Detalhes:** Evidências vinculáveis à Issue, Issues relacionadas, Blockers cuja causa é a Issue, afetados e implicações, correções e material. Issue **não possui CriterionSet próprio no domínio atual**. Não renderizar um falso editor de critérios da Issue. Se forem desejados critérios de solução, colocá-los na Task/Objective corretivo, ou aprovar projeto distinto de mudança do domínio.

**Histórico:** reporte, investigação, vínculos, notas, resoluções, reaberturas. **Resolver Issue não libera todos os Blockers automaticamente**; fluxo deve oferecer seleção explícita de quais impactos liberar.

### 5.5. Blocker — impedimento efetivo

**Resumo:** descrição, causa (Issue, WorkItem, Objective, Decision ou origem externa conforme suporte), alvo impedido, propagação e estado; impacto direto ou na subárvore com contagens confiáveis.

**CTA:** `Ver impacto` quando ativo, `Liberar impedimento` quando usuário tem autoridade e causa efetivamente removida, ou `Ver resolução` quando terminal.

**Detalhes:** causa, alvo exato, propagation (`direct`/`subtree` de acordo com tipo), dependentes afetados, Issue relacionada, justificativa de liberação, referência da resolução.

**Histórico:** criação, atualização de descrição, resolução/cancelamento. Registro de evidência sobre a causa/entidade afetada, não criando link inválido diretamente para o Blocker.

**Regra:** `Blocker` impedindo um Outcome tende a propagar para sua árvore. **Não** criar esse Blocker automaticamente quando a entrega de um Outcome é reprovada, pois isso pode bloquear exatamente as tarefas de correção.

### 5.6. Outros registros da camada documental

| Registro | Comportamento ao selecionar quando a camada documental estiver habilitada |
|---|---|
| Criterion | Exibir owner, condição, revisão, modo de verificação, Evidence vinculada, Assessment aplicável e ações autorizadas. |
| Evidence | Exibir origem, descrição, stance por link, proprietário, contexto e retraction; nunca “aprovada” sem Assessment. |
| Artifact | Exibir tipo, fonte/URI, versão/digest, relação com entrega/Task e Evidence derivada, quando houver. |
| Assessment | Exibir resultado, autor/revisor, justificativa, revisão e submissão a que pertence; consulta histórica. |
| Decision | Exibir decisão, fundamento, status e entidades associadas. |
| Roadmap/plan reference | Abrir plano/revisão em sua interface apropriada, sem transformar em dependência. |
| Relation/EvidenceLink | Explicar endpoints e direção, suporte/contradição/contexto e ações válidas. |
| Contract/ReviewCase/Submission | Consultar pelo contexto da Task; dados sensíveis e comandos de assinatura só nos canais autorizados. |

**Importante:** o mapa de trabalho atual já oculta registros documentais por padrão, mas possui camada explícita de registros. Preservar essa capacidade; não apagar o domínio nem remover deep links existentes.

---

## 6. Kanbans contextuais: funcionamento exato

### 6.1. Regras comuns aos dois Kanbans

- O Kanban é **uma projeção de leitura**, não uma entidade persistida; uma abertura não produz Domain Event.
- Abre em **modal sobre o Canvas**, com backdrop discreto e foco isolado. Board, pan e post-its atrás do modal ficam inertes enquanto modal aberto.
- Ocupa a maior região útil sem extrapolar a viewport; mantém cabeçalho com título e breadcrumb, resumo e controle Fechar.
- Reutiliza o modelo de Workspace, identidade e permissões; não reutilizar indevidamente a lógica do Task Board global como fonte de dados para Objetivos.
- Quatro etapas visuais: **A fazer**, **Em execução**, **Verificação** e **Aceitos**. Os rótulos de detalhe esclarecem se algo está “pronto para verificar” ou “aguardando revisor”.
- **Bloqueado, dependências pendentes, agendado, correções solicitadas, contrato expirado** são indicadores/alertas sobre cards, não necessariamente colunas extras.
- Cancelados/abandonados não entram em “Aceitos”; filtro/área separada **Cancelados/fora de escopo**, conservando histórico e contagem.
- Cada card exibe título, estado, progresso pertinente, critério/pendência dominante, responsável/autoridade com rótulo verdadeiro, bloqueios e menu contextual.
- A rolagem horizontal do Kanban é permitida em viewport estreito; não comprimir quatro colunas até o texto ficar ilegível.
- Listagem paginada com estado `partial`; cards carregados não devem fingir totalidade.
- Mudar ordenação manual pode ser uma preferência local de apresentação; **não** cria dependências nem muda ownership. Qualquer movimentação de fase exige comando real e permissão.
- Modal e mapa devem compartilhar uma revisão coerente de dados. Após mutation, atualizar as projeções necessárias preservando contexto de navegação e scroll; no conflito, solicitar refresh adequado.

### 6.2. Kanban do Outcome: Objectives

**Entrada:** clique no post-it Outcome → lateral → `Abrir Kanban de objetivos`.

**Conteúdo:** Objectives do Outcome organizados pelo lifecycle/estágio operacional; exibir subobjetivos com indentação/caminho ou agrupamento navegável, não achatar a hierarquia sem identificá-la. Cards podem ter badge `Obrigatório para Outcome` ou `Opcional para Outcome` (a flag atual é global). Dependências explícitas entre Objectives aparecem como indicador e link.

**Ações:** `Adicionar objetivo`, `Adicionar Task direta`, `Verificar Outcome`/`Ver pendências`, filtros (obrigatórios, bloqueados, com revisão pendente, responsáveis), busca, alternância para `Mapa de dependências` que **fecha/suspende modal mantendo seleção** ou navega a foco correspondente no Canvas.

**Tasks diretas do Outcome:** não são Objectives. Exibir aba/segmento `Tarefas diretas` dentro do mesmo modal, com as mesmas quatro colunas de Tasks e contexto correto; não inventar um Objective “Sem objetivo” persistido. Na política estrita, suas obrigações devem ser contabilizadas conforme regra explícita (ver § 11).

**Ao clicar Objective no modal:** abrir uma área de inspeção contextual dentro do modal (painel inline ou drawer contido), sem novo modal de nível superior; CTA `Abrir tarefas` muda conteúdo para o Kanban daquele Objective, com breadcrumb `Outcome > Objective`.

**Critério para Verificação do Objective:** não criar revisão formal imaginária. Pode estar “Pronto para verificar” quando o backend reportar que suas obrigações operacionais foram satisfeitas mas falta avaliação/conclusão; isso é **etapa de apresentação derivada**, não nova lifecycle nem afirmação de que um revisor foi designado.

### 6.3. Kanban do Objective: Tasks

**Entrada:** clique Objective no Board → lateral → `Abrir Kanban de tarefas`; ou Outcome Kanban → Objective → `Abrir tarefas`.

**Conteúdo:** Tasks **diretamente vinculadas àquele Objective**. Subobjetivos continuam acessíveis em seção/aba `Subobjetivos`, com abertura do seu próprio contexto no mesmo modal. Resumo superior: `x/y Tasks aceitas`, critérios do próprio Objective e pré-requisitos de aceitação. Uma Task pode estar `todo` e `ready`, ou `todo` e `blocked`; mostrar ambos sem tratar como dois lifecycles distintos.

**Ações:** `Adicionar Task` com parent preselecionado, `Adicionar subobjetivo`, filtrar por estado/responsável/prioridade, abrir Task, mostrar dependência relacionada, alternar para verificar Objective quando elegível.

**Ao clicar Task no modal:** painel de detalhe e ações **dentro do modal** ou transição controlada ao inspector preservando o modal e o foco de retorno; nunca stack de dialogs. A UI deve permitir editar/registrar materiais sem perder scroll da coluna.

### 6.4. Mapeamento dos estados às quatro colunas

| Tipo | Estado canônico / condição | Coluna | Sinalização obrigatória |
|---|---|---|---|
| Objective | `planned` sem condições de conclusão | A fazer | Dependências hard, preparação, owner. |
| Objective | `in_progress` | Em execução | Tasks aceitas e pendências. |
| Objective | Planejado/em andamento, pronto à avaliação mas sem conclusão | Verificação | `Pronto para verificar` — **derivado**, não ReviewCase formal. |
| Objective | `achieved` | Aceitos | Evidência de conclusão, data e critérios aplicáveis. |
| Objective | `cancelled` | Cancelados | Nunca chamar aceito. |
| Task | `backlog`/`todo` | A fazer | `ready`/`waiting_dependencies`/`blocked`/`scheduled`. |
| Task | `in_progress` com execução ou correção legítima | Em execução | Contrato/lease, autor e status real de progresso. |
| Task | `pending_review`/ReviewCase efetiva no protocolo assinado | Verificação | `Aguardando revisão independente`; link para ReviewCase. |
| Task | Entrega submetida em protocolo sem ReviewCase formal | Verificação **somente** se existe estado de submissão/etapa verificável; caso contrário, manter fase canônica e CTA de conclusão | Não fabricar fila. |
| Task | `done` com conclusão real | Aceitos | Resultado e provas. |
| Task | `cancelled` | Cancelados | Não conta como aceito. |
| Task | Correções solicitadas | Em execução | Badge `Correção solicitada`, apontamentos visíveis. |

A coluna exibida não pode ser a única informação operacional; usar badges para status críticos e mensagem explicativa no inspector.

### 6.5. Transição por drag & drop e alternativa por teclado

**Fase inicial:** drag & drop pode ser **apenas reorganização visual**. Se implementarmos drag entre colunas como transição de estado, deve chamar uma ação de domínio explícita com confirmação onde necessária e rollback visual em caso de rejeição. Não permitir uma “mudança otimista” parecer concluída antes do servidor confirmar.

- `A fazer → Em execução` (Task): requer ativação/readiness e aquisição de lease ou contrato, possivelmente handoff ao host assinado. Se não permitido, mostrar por que e o próximo passo; **não forçar** estado.
- `Em execução → Verificação`: abre `Enviar entrega`, monta submissão e segue protocolo real; não altera lifecycle indiscriminadamente.
- `Verificação → Aceitos`: somente conclusão aceita pelo backend, com avaliação e identidade elegíveis.
- `Aceitos → Em execução`: exige operação de reabertura autorizada e justificativa; nunca um drag simples.
- Objective `planned → in_progress`: aciona `Start` quando aplicável, respeitando exigência de critério ativo; não obrigar a iniciar para simplesmente abrir quadro.
- Objective `→ achieved`: avaliação e conclusão explícitas; não move só porque suas Tasks terminaram.

**Acessibilidade:** menu `Mover para...`/`Ações de estado` com as mesmas capabilities. Keyboard-only completo sem depender do drag.

### 6.6. Navegação e retorno

- Um único modal Kanban montado por vez.
- Breadcrumb troca contexto sem empilhar modal: `Outcome > Objective > Task` (Task é inspeção, não terceiro Kanban).
- `Voltar` retorna ao nível anterior e restaura filtros, coluna, scroll e card selecionado.
- `Fechar` retorna ao Canvas exatamente com Outcome, zoom, pan, colapsos e post-it selecionado anteriores.
- `Escape` fecha primeiro menu/form/inspeção interna, depois modal; proteção contra perda de rascunho.
- Deep link opcional para reabrir o contexto se o roteamento já suportar identidade segura; não expor escopos privados em URL desprotegida.
- Mudança de Outcome/Workspace, revogação ou logout limpa os dados privados de modal/inspector e não permite resposta tardia restaurá-los.

---
## 7. Criação manual: menos campos e mais contexto

### 7.1. Princípio

**Entrada mínima e detalhamento progressivo.** Registrar algo no plano não é iniciar trabalho, nem autorizar execução. Um campo só é obrigatório na criação se o domínio realmente o exigir naquele momento. Outros requisitos aparecem **quando o próximo comando realmente precisar deles**, com reparo guiado e rascunho preservado.

### 7.2. Criar Outcome

1. Usuário escolhe `Novo Outcome`.
2. Informar **nome** e **resultado desejado** em linguagem humana. O domínio atual exige `title` e `desired_state`: manter esses dados, mas sem expor nomes técnicos.
3. Criar como `draft`. Não produzir Criterion fictício a partir de texto.
4. Abrir o Board preservado com foco no novo Outcome.
5. Mostrar `Adicionar objetivo`, `Adicionar tarefa direta` e `Definir condição de sucesso`.
6. Ao ativar, se faltar critério ativo requerido, apresentar: **“Como saberemos que este resultado foi alcançado?”**, editor de Criterion pré-preenchido apenas como sugestão não confirmada; após confirmação, criar Criterion real e permitir revisar a ativação.
7. A criação de Objectives e Tasks pode anteceder a ativação; nada adquire execução automaticamente.

### 7.3. Criar Objective

1. Abrir pelo menu do Outcome, lateral ou Kanban.
2. Solicitar inicialmente **título**; parent já vem do contexto e é exibido antes de salvar.
3. Campo opcional simples: **“Obrigatório para entregar este Outcome?”** (valor inicial **sim**, como já acontece no formulário humano do PR #116). Ressaltar que esse indicador hoje é **global para o Outcome**, não “obrigatório só para o pai”.
4. Criar em `planned`, aparecer imediatamente na coluna A fazer do quadro de Objectives.
5. `Adicionar critério` e `Adicionar Task` ficam disponíveis dentro do inspector/kanban.
6. Ao tentar iniciar Objective sem Criterion exigido, exibir formulário curto para definir condição de sucesso; **não** exigir iniciar apenas para abrir ou planejar seu Kanban.

### 7.4. Criar Task

1. A partir de Objective, o parent fica pré-selecionado e visível. A partir do Outcome, ela será Task direta.
2. Solicitar inicialmente **título**. Descrição, instruções, contexto, entregáveis, critérios, responsável e data ficam opcionais/expansíveis quando permitidos.
3. Criar em `backlog`; sem claim, contrato ou agente executando.
4. Exibir no Kanban imediatamente e adicionar CTA `Preparar/Disponibilizar` conforme estado.
5. Na preparação/entrega, verificar se há critérios necessários pelo protocolo. O modelo legado permite concluir determinadas Tasks sem Criteria requeridos, exigindo resumo de resultado; **não transformar toda Task trivial em um ritual artificial de avaliação obrigatória**.
6. Se houver dependência hard, refletir readiness real. A posição visual da Task não define sua ordem de execução.

### 7.5. Criação em lote

Reusar `create_plan_items` no PR #116 (até 50 itens e 100 critérios iniciais conforme limites documentados), mantendo pai correto, prévia de árvore e confirmação atômica. O lote jamais deve criar uma Task sob Objective incorreto por perda do contexto do modal. Falha em qualquer entrada implica rollback integral.

### 7.6. Microcopy de criação e preparação

| Situação | Texto proposto | Ação |
|---|---|---|
| Outcome sem condição de sucesso | “Defina como saberemos que o resultado foi entregue.” | `Definir condição de sucesso` |
| Objective sem critério exigido para iniciar | “Antes de iniciar, descreva o resultado que comprova este objetivo.” | `Adicionar critério` |
| Task em backlog | “Esta tarefa foi planejada, mas ainda não está disponível para execução.” | `Disponibilizar` |
| Task bloqueada | “A tarefa depende de ‘X’, ainda não aceito.” | `Abrir pré-requisito` |
| Necessidade de autorização de execução | “Esta operação requer uma sessão de execução autorizada.” | `Continuar no host autorizado` |
| Não há pendências conhecidas, mas dados parciais | “A leitura está incompleta; carregue o restante antes de concluir.” | `Carregar restante` |

---

## 8. Critérios de aceitação: checklist que registra avaliações reais

### 8.1. Experiência desejada

Dentro de **Detalhes e gerenciamento → Critérios de aceitação**:

```text
Critérios de aceitação                         [ + Adicionar ]
2 atendidos · 1 pendente

✓ Login devolve token válido                 [Ver]
✓ Token inválido é rejeitado                 [Ver]
○ Token expirado é rejeitado                 [Verificar]

[Verificar critérios]     [Histórico de avaliações]
```

A lista curta mostra o estado aplicável **à revisão atual do Criterion e à entrega atual**, não a última avaliação histórica indiscriminadamente. O usuário pode ver o critério e as evidências associadas sem navegar até outra coleção.

### 8.2. Adicionar Criterion

1. Abrir `+ Adicionar` sob Outcome, Objective ou Task.
2. Título/pergunta operacional: **“O que deve ser verdadeiro para aceitar este resultado?”**
3. Descrição adicional opcional.
4. Campo `Obrigatório para aceitar` com semântica real (não inventar default em API antiga).
5. Modo de verificação: padrão sugerido `evidence_review` no fluxo humano, com `attestation` e `external_evaluation` disponíveis em modo avançado.
6. Prévia do owner e do efeito sobre a conclusão (se exigido, passa a integrar as obrigações).
7. Confirmar criação pelo comando existente; conservar editor aberto em erro corrigível.
8. Revisões de Criterion só são autorizadas quando o estado permitir. Se alterado, a UI marca que avaliações anteriores podem não se aplicar à nova definição.

### 8.3. Modal “Verificar critérios”

O modal apresenta o **sujeito, a versão observada e o conjunto de critérios aplicáveis**. Cada item traz a condição completa, materiais já vinculados, modo de verificação, resultado atual e botão de avaliação.

```text
Verificar entrega — Implementar autenticação JWT
Critérios: 2 de 3 atendidos

1. Login retorna token válido
   ● Atendido    ○ Não atendido    ○ Inconclusivo
   Evidências: teste de autenticação [inspecionar]
   Justificativa: validado pela execução de teste ...

2. Token inválido é rejeitado
   ● Atendido    ○ Não atendido    ○ Inconclusivo
   Evidências: relatório CI #234 [inspecionar]

3. Token expirado é rejeitado
   ○ Atendido    ○ Não atendido    ○ Inconclusivo
   [Vincular evidência] [Registrar observação]
   Justificativa: ______________________________

[Registrar avaliações autorizadas] [Voltar sem salvar]
```

**Regras do formulário:**

- Seleções `Atendido`, `Não atendido`, `Inconclusivo`; `Dispensado/waived` é ação excepcional **separada**, com permissão e justificativa obrigatórias, nunca camuflada como atendido.
- A UI pode manter rascunhos locais de seleção, mas cada persistência deve executar `record_criterion_assessment` verdadeiro, ou operação de revisão assinada nos protocolos que exigem isso.
- O editor coleta rationale, Evidence IDs e identificação do Criterion/revision corrente; campos exigidos seguem VerificationMode.
- `evidence_review`: exigir materiais elegíveis e efetiva avaliação; `attestation`: declaração explícita e identidade válida; `external_evaluation`: fonte/resultado externo conforme contrato. Nunca assumir que um link passou por teste automático do WOS.
- Evidência adversa (`contradicts`) precisa continuar acessível, mesmo quando existem várias provas favoráveis. Nunca usar “3 evidências” como score de aprovação.
- Um mesmo Evidence pode ser usado em diferentes critérios, com relações e stances independentes; evitar duplicar registros.
- Não atualizar um Assessment antigo para fazê-lo parecer atual após alteração da definição, source digest ou submissão.
- Em UI para executor **sem permissão de avaliar**, substituir controles por `Preparar comprovações` / `Solicitar revisão` (se houver fluxo implementado). O executor pode observar a lista, não autoavaliar.
- Em signed v2, o navegador apresenta instruções/contexto e handoff para Profile/host autorizados. Não conter chave privada ou emitir assinatura pelo próprio browser.
- A aplicação revalida critérios e versões na submissão. Se um registro foi alterado em paralelo, pedir nova revisão explícita do conjunto afetado.

### 8.4. Estados de Critério mostrados ao usuário

| Projeção humana | Condição real |
|---|---|
| Não verificado | Sem Assessment aplicável atual. |
| Atendido | Assessment atual `met`, aplicável ao Criterion revision e à entrega pertinente. |
| Não atendido | Assessment atual `not_met`, com justificativa. |
| Inconclusivo | Assessment atual `inconclusive`, revisão adicional exigida. |
| Dispensado | `waived` autorizado; exibir distinto de sucesso verificado. |
| Prova indisponível | Evidence referenciada retirada/inaplicável ou link não autorizado; remediação. |
| Definição alterada | Revisão de Criterion mudou; reassessment quando necessário. |
| Dados incompletos | Paginação/coverage não permite concluir todos os critérios. |

### 8.5. Ação “Verificar entrega” vs “Registrar avaliações” vs “Aceitar”

- **Verificar entrega:** abrir pacote de avaliação (critérios, artefatos, evidências, histórico de submissão).
- **Registrar avaliações:** gravar os resultados dos critérios com a autoridade necessária.
- **Aceitar/concluir:** comando de Conclusion separado, permitido só depois de satisfazer os gates pertinentes.

Podem ser apresentados em **uma única jornada visual com etapas**, mas não colapsados em uma mutation sem validação. Um botão de conclusão não deve simplesmente “marcar todos como atendidos”.

---

## 9. Materiais: Artefatos e Evidências sem confusão

### 9.1. Modelo mental

- **Artefato (Artifact):** “o que produzi ou entreguei”. Ex.: commit, pull request, binário, documento, relatório de execução, URL de deploy.
- **Evidência (Evidence):** “o que observamos para avaliar um critério”. Ex.: teste de expiração de token, log de CI, inspeção de resultado, medição, parecer.
- **EvidenceLink:** “como essa evidência se relaciona com este critério ou entidade”; `supports`, `contradicts` ou `context`, sempre com rationale próprio.
- Um **Artefato pode originar uma Evidência** quando alguém o referencia explicitamente como fonte de comprovação. Não transformar automaticamente qualquer Artefato em teste aprovado.

### 9.2. Onde aparece

**Detalhes e gerenciamento → Materiais**:

```text
Materiais                                         [+ Adicionar material]

Artefatos (2)
  ├── PR #204 — Endpoint JWT                   [Abrir] [Detalhes]
  └── Commit 92b3...                           [Abrir] [Detalhes]

Evidências (3)
  ├── Teste de login válido                   [Critério 1]
  ├── Rejeição de token inválido              [Critério 2]
  └── Inspeção de logs                        [Contexto]
```

Contadores e listas separados. Permitir filtrar por critério e estado. Mostrar itens retirados (withdrawn/retracted) como históricos sem contá-los como comprovação atual.

### 9.3. Fluxo “Adicionar material”

1. Abrir um único mini-wizard e escolher **“O que foi produzido” (Artefato)** ou **“Comprovação” (Evidência)**.
2. Para Artifact: informar nome/descrição, tipo e referência externa suportada (URI, provider e identificador/versão/checksum quando cabíveis); contexto e relação `produces` ou `relates_to` coerente. Salvar com `register_artifact_in_context` quando aplicável.
3. Para Evidence: escolher origem (`external`, `artifact`, `observation`, `existing`), descrever observação, tipo real, referência ou fonte existente; escolher Criterion alvo (pré-selecionado se a operação começou nele), stance e rationale.
4. Revisar destino e relação antes de salvar. Registrar Evidence e EvidenceLinks **atomicamente** com `register_evidence_with_links` quando o contrato permitir.
5. Após sucesso, atualizar lista, contador e critérios vinculados no mesmo contexto; **não** marcar Criterion automaticamente como `met`.
6. Caso uma fonte não tenha comprovação confiável (link externo apontado pelo usuário), exibir como referência fornecida pelo participante, não como resultado de inspeção de conteúdo realizada pelo servidor.

### 9.4. Fontes admitidas e transparência

| Fonte | Interface | Fato demonstrado |
|---|---|---|
| URL/ID externo | “Vincular fonte externa” | Fonte declarada; não prova que WOS abriu ou validou conteúdo. |
| Artifact já registrado | “Usar artefato existente” | Origem rastreável, ainda sujeita à avaliação. |
| Observação autenticada | “Registrar observação” | Prova de que uma identidade registrou determinada observação; não de que o teste externo aconteceu. |
| Evidence existente | “Reutilizar evidência” | Relacionamento adicional, sem cópia duplicada. |
| Envio de arquivo real | **Não disponível nesta fase** | Exigiria feature própria de upload, storage, quotas e segurança. |

### 9.5. Pacote de entrega

Ao clicar `Enviar entrega`, montar uma revisão com:

1. Identidade, WorkItem e resultado esperado.
2. Resumo do trabalho realizado; campos obrigatórios conforme protocolo.
3. Artifacts produzidos (incluindo suas versões/digests quando aplicáveis).
4. Evidence referenciadas, por critério, com links e stance.
5. Critérios atuais e suas pendências, **sem substituição de Assessment por contagem**.
6. Submissão/contrato/lease e canal de assinatura, quando exigidos.
7. Prévia do efeito esperado, quem precisa revisar e quais obrigações continuam abertas.

`Adicionar material` registra fatos; `Enviar entrega` cria submissão pelo protocolo de trabalho legítimo. São ações diferentes. Se o protocolo for assinado, a submission e digest devem permanecer vinculados à autoridade e à versão correta.

---

## 10. Histórico de atividade: auditável, mas discreto

### 10.1. Aparência

**Segundo e último grande grupo expansível**. Inicialmente recolhido, com contador/última ação disponível:

```text
▸ Histórico de atividade                     12 registros
  Último: “Entrega submetida” · há 18 min
```

Ao expandir, exibir **as 3 a 5 atualizações recentes mais pertinentes**, não dezenas de cartões integralmente expandidos. Cada evento: tipo, autor/identidade de fato, alvo, quando **registrado**, ocorrência declarada quando existente, resumo e link ao registro original.

### 10.2. Fontes

- Eventos de domínio registrados (criação, ativação, alteração, relação, critério, conclusão).
- EntityNotes documentais autenticadas (atualização, descoberta, observação, correção por append).
- WorkActivity assinada pelo executor habilitado no contrato, incluindo progresso, problema, correção, validação e resultado.
- Checkpoints, submissões e decisões de revisão disponíveis no protocolo aplicável.
- Histórico de descendentes, **quando explicitamente habilitado**, preservando sujeito real de cada ação.

Não mostrar WorkActivity duas vezes como nota e evento correspondente. `occurred_at` não substitui o horário de registro e não deve permitir ordenar falsamente a auditoria. Anotações corrigidas mantêm original e correção vinculada.

### 10.3. Ações

- `Registrar atualização`: formulário breve “O que aconteceu?” e tipo documental adequado; `append_entity_note` com autor autenticado.
- Quando a identidade é executor assinado autorizado, `Registrar avanço de execução` usa o protocolo de WorkActivity; não substituir por nota genérica para burlar autoridade.
- `Ver histórico completo` abre timeline paginada/filtro apropriado; carregar anteriores pelo cursor de frontier. Um refresh de eventos novos inicia nova fronteira e não mistura páginas antigas.
- Pesquisar/filtrar eventos, critérios, entregas, participantes quando houver necessidade real, sem tornar todos os filtros permanentes na lateral.

### 10.4. Estado e cobertura

- `Completo`, `Parcial`, `Não carregado`, `Indisponível`: mensagens distintas.
- Cursor e versões observadas são preservados; alterações estruturais que mudam a pertinência de descendentes exigem nova leitura consistente.
- Histórico é apêndice cronológico imutável, não fonte que modifica lifecycle por si só.

---

## 11. Fluxos de execução, revisão, conclusão e política estrita

### 11.1. Fluxo da Task — ponta a ponta

```text
Task criada (backlog)
   ↓ Preparar/Disponibilizar (todo)
   ↓ Adquirir autoridade (lease/contrato; no protocolo aplicável)
Em execução
   ├── Registrar avanço (atividade/nota correta)
   ├── Registrar Artifact produzido
   ├── Registrar Evidence e vinculá-la ao Criterion
   └── Consultar “O que falta?” (WorkAttention real)
   ↓ Enviar entrega (Submission pelo protocolo aplicável)
Verificação / revisão autorizada
   ├── Critério atendido → Assessment aplicável
   ├── Critério não atendido/inconclusivo → apontamento + correção
   └── Política de revisor independente → outro Principal elegível
   ↓ Aceitar explicitamente (Conclusion)
Task done
   ↓ Dependentes são reavaliados quanto à readiness
```

**Reprovação:** Task **não** muda para `done`; registrar finding/review result aplicável, mostrar “Correção solicitada”, retomar por aquisição legítima e produzir nova entrega com digest/revisão próprios. Evidência histórica não vira aprovação da nova submissão.

**Sem revisão independente obrigatória:** o fluxo ainda valida critérios e conclusão pelo mecanismo autorizado do Workspace. Não significa finalizar pelo simples drag ou pela presença de artifacts.

**Ações do executor versus revisor:**

| Fase | Executor | Revisor elegível |
|---|---|---|
| Preparação | Ver instructions, registrar materiais quando permitido | Consultar condições, se autorizado. |
| Execução | Atualizar progresso e produzir entrega com contrato válido | Inspecionar estado conforme leitura autorizada. |
| Revisão pendente | Consultar status e findings, não aprovar sua própria submissão sob política independente | Avaliar Evidence, registrar resultados, aceitar/rejeitar conforme protocolo. |
| Correção | Adquirir autorização de correção e retomar | Inspecionar nova submissão posteriormente. |
| Aceita | Ver conclusão e histórico | Ver o resultado e a base de aceitação. |

### 11.2. Fluxo do Objective

1. Criar Objective e planejar Tasks/subobjetivos; Kanban funciona ainda em `planned`.
2. Iniciar Objective **apenas quando apropriado e com os requisitos próprios**. Um Objective elegível também pode ser concluído diretamente de planned no modelo existente.
3. Acompanhar Tasks e dependências no Kanban contextual.
4. Para solicitar conclusão, montar **Prévia de aceitação do Objective**: Criteria próprios, avaliações, Evidence, Tasks e subobjetivos requeridos pela política aplicável, Blockers e dependências.
5. Se houver obrigação incompleta, mostrar por nome e botão `Abrir item`; não bloquear com mensagem abstrata.
6. Verificar critérios atuais via avaliação autorizada; concluir explicitamente com `achieve_objective` quando backend confirmar todos os gates.
7. Em caso de reprovação, manter Objective operacionalmente corrigível (`planned`/`in_progress` conforme canônico), registrar apontamento e criar Task corretiva ou Issue **apenas se necessária**.
8. Após `achieved`, reavaliar dependências dos demais Objectives/Tasks; não executar automaticamente nenhum agente.

### 11.3. Fluxo do Outcome

1. Manter o mapa do Outcome, abrir Kanban de Objectives para acompanhamento.
2. Quando o usuário escolher `Verificar entrega do Outcome`, abrir visão consolidada com condição final, Criteria, Assessments atuais, Objectives globalmente `required_for_outcome`, Tasks diretas e pendências da política aplicável.
3. Indicar todo objetivo obrigatório fora do conjunto atualmente carregado na visualização, a partir da projeção autoritativa (não depender de cards presentes no Canvas).
4. Se todos os gates canônicos e de política estiverem atendidos, habilitar `Concluir Outcome` com prévia/razão/versão observada; registrar Conclusion explícita.
5. Se reprovar, manter o Outcome ativo (não marcar `failed` por simples devolução), registrar justificativa e eventual trabalho corretivo; só concluir depois de reavaliar.
6. Após `achieved`, respeitar o fechamento de escopo operacional do Outcome. Novos trabalhos exigem fluxo legítimo de reabertura, não um clique para adicionar post-it.

### 11.4. Divergência atual: conclusão com descendentes abertos

**Fato verificado:** na implementação estudada, `workspace_conclusion.go` mostra quantidade de descendentes abertos como **impacto**, não como uma condição automática extra de aceitação. Um pai elegível pode ser concluído com descendentes abertos sob as regras atuais. Isso difere da experiência desejada pelo proprietário, que espera conclusão do conjunto obrigatório.

**Mudança proposta:** incluir no **Outcome** uma política de conclusão versionada e explícita:

- `compatible` — preserva semântica atual dos clientes e records já existentes.
- `strict_work` — para novos Outcomes criados pelo fluxo humano, exigir resolução das obrigações de trabalho descendentes apropriadas, além dos Criteria próprios, permissões, Blockers, dependências e regras de revisão.

**Compatibilidade:** API/CLI/MCP/SDK antigos que omitirem o novo campo continuam com o valor/semântica `compatible`; registros existentes **não** mudam implicitamente. No formulário de criação humano, `strict_work` pode ser o default **de experiência**, mostrado claramente e confirmado.

### 11.5. Definição de `strict_work` (proposta implementável)

**Para Objective:**

1. Todas as Tasks **operacionais** ligadas diretamente a esse Objective precisam estar aceitas (`done`) ou **explicitamente canceladas com justificativa registrada**; cancelamento não será contado como Task aceita. Tasks em backlog/todo/in_progress bloqueiam conclusão.
2. Todos os subobjetivos diretamente associados ao Objective precisam estar `achieved` ou explicitamente cancelados com justificativa; percorrer descendentes de modo coerente sem ciclos.
3. Critérios obrigatórios próprios precisam estar satisfeitos na revisão correta, por avaliações aplicáveis.
4. Dependências hard e Blockers efetivos permanecem gates independentes.
5. **Cancelamentos não são aprovações:** se houver qualquer cancelamento, a prévia de conclusão deve listá-lo nominalmente e exigir confirmação explícita de que a exclusão é aceitável para o resultado, com registro da decisão na Conclusion (ou em registro auditável associado). Não permitir “concluir tudo” silenciosamente após cancelar todas as Tasks.
6. Verificar autorização e revisões novamente na transação de conclusão.

**Para Outcome:**

1. Todos os Objectives marcados `required_for_outcome` precisam estar `achieved`, inclusive subobjetivos fora do mapa carregado — preserva regra atual.
2. Tasks diretamente vinculadas ao Outcome passam a contar no gate `strict_work` (aceitas ou justificadamente canceladas, com confirmação explícita quando canceladas). Não criar um Objective fictício.
3. Critérios finais obrigatórios precisam estar satisfeitos; blockers ativos, política de revisão e versões correntes são validados.
4. Não tratar Objectives opcionais como obrigatórios só por sua posição ou por estarem mostrados no Kanban do Outcome.
5. Ao concluir, mostrar escopo que será fechado e consequências sobre trabalho restante, inclusive exceções de cancelamento, com revisão explícita e `observed_outcome_revision` quando disponível.

**Nota de precisão:** a flag já existente `RequiredForOutcome` é uma obrigação global, mesmo para um subobjetivo. Para a política `strict_work` há também uma obrigação **estrutural local** de concluir o trabalho diretamente organizado sob o Objective. Essas duas dimensões devem ser nomeadas distintamente na UI e validadas no servidor. Não renomear a flag antiga fingindo que só vale para o pai.

**Extensão mínima de backend necessária:** persistir/ler a política e incluir o gate descendente no pipeline transacional de conclusão, com tipos de erro e lista de afetados; registrar explicitamente cancelamentos reconhecidos pelo autor/revisor na conclusão ou referência auditável apropriada. Não depender de contagem client-side nem de estado de página. A forma exata do registro de exceção deve ser escolhida de modo a preservar o schema/contrato de Conclusion existente ou evoluí-lo aditivamente.

### 11.6. Solicitar verificação de Objective/Outcome

**Primeira entrega:** `Verificar entrega` abre e conduz avaliações e conclusão pelos comandos autorizados existentes. A opção `Solicitar verificação` **só pode aparecer** se existir um mecanismo real de encaminhamento a revisor elegível. Não gerar uma promessa de fila com uma nota genérica.

**Incremento posterior opcional:** implementar `VerificationRequest` aditivo **somente para Objective/Outcome**, com `target_ref`, `target_version`, `outcome_revision`, revisões de Criteria, autor da solicitação, status (`requested`, `in_review`, `accepted`, `changes_requested`, `superseded`), parecer e referencias de Assessments/Conclusion. Deve respeitar a política de identidade e caducar quando definição/escopo mudar. O fluxo de Task signed continua usando o `ReviewCase` já existente — **não duplicar** esse mecanismo.

### 11.7. Tabela dos gates

| Operação | Condições mínimas e resultado |
|---|---|
| Ativar Outcome | Draft não arquivado, ao menos um Criterion requerido ativo e demais gates atuais; lifecycle `active`. |
| Iniciar Objective | Precondições de Objective, incluindo Criterion requerido conforme regra atual; não pressupõe iniciar Tasks. |
| Tornar Task disponível | Task backlog elegível, transição ao estado canônico apropriado. |
| Adquirir Task | Readiness, ausência de Blockers impeditivos, dependências hard satisfeitas, identidade/contrato/lease válidos. |
| Enviar entrega | Protocolo correspondente, autoridade atual, resumo e materiais exigidos. |
| Avaliar Criterion | Permissão, modo de verificação, definição atual, prova/revisor elegível e, quando aplicável, submissão/digest atuais. |
| Completar Task | Conclusão/resultado válido, critérios obrigatórios satisfeitos ou condição canônica alternativa, assinatura/review conforme protocolo. |
| Concluir Objective | Gates atuais + `strict_work` quando configurado; Conclusion explícita. |
| Concluir Outcome | Gates atuais + `strict_work` quando configurado; Conclusion explícita. |
| Reabrir terminal | Comando de reabertura autorizado, motivo e revisão correta; preservar histórico. |

---

## 12. Issues, Blockers, correções e pendências de entrega

### 12.1. Três situações que **não** devem ser confundidas

1. **Falha no critério durante a revisão:** Assessment `not_met` ou `inconclusive` e apontamento; a entrega ainda não foi aceita.
2. **Issue:** problema formalmente registrado para investigação/solução e rastreabilidade, com entidades afetadas.
3. **Blocker:** impedimento real que suspende progressão operacional de um alvo, com propagação definida.

Nem toda falha de critério exige Issue. Nem toda Issue exige Blocker. **Uma entrega rejeitada não deve criar automaticamente um Blocker `subtree` do Outcome.**

### 12.2. O que mostrar no inspector após reprovação

```text
Pendências de entrega  (2)
  • Idempotência falhou — critério não atendido
    [Ver evidência] [Corrigir na Task]
  • Webhook sem retentativa — problema registrado
    Issue #18  ·  Task corretiva #26
    [Abrir correção]

[Registrar atualização] [Criar Task corretiva] [Reportar Issue]
```

Essa é uma **projeção contextual**, não uma entidade `Block` obrigatória. Chamar a seção `Pendências de entrega`, `Correções` ou `Impedimentos` conforme o tipo; o termo `Block` só é justificável futuramente se houver vida própria, responsável, política e histórico independentes.

### 12.3. Fluxo de correção

1. Revisor aponta critério `not_met` e associa evidências/finding.
2. Executor consulta apontamentos pela Task, retoma pela autoridade correta, registra correções e nova submissão.
3. Se a mudança exceder a Task original, criar Task corretiva dentro de Objective existente ou criar Objective corretivo no Outcome.
4. Se o problema merecer rastreabilidade formal, criar Issue associada à entidade e material; se impedir progressão operacional, criar Blocker explícito depois de **prévia de impacto**.
5. Corrigir, registrar evidências atuais, avaliar novamente, aceitar Task/Objective/Outcome quando satisfeitos.
6. Resolver Issue; liberar apenas Blockers realmente removidos e explicitamente selecionados, com operação autorizada.

### 12.4. Correção ao nível de Outcome

Se o Outcome está ativo e falha no critério final, criar Objective corretivo requerido para a entrega, com suas Tasks e condições próprias. O Outcome continua ativo e verificável. Não concluir como `failed` automaticamente e não criar Blocker global que impeça as próprias correções.

### 12.5. Propagação de Blockers existente

- Outcome: propagação `subtree`.
- Objective: `direct` ou `subtree`.
- Task: `direct`.
- Evidence não se liga diretamente ao Blocker como alvo permitido de EvidenceLink no contrato atual; registrar prova no alvo, causa ou Issue elegível.

Antes de aplicar/revogar impedimento, mostrar nomes e contagem das entidades afetadas, cobertura da amostra, autorização e prévia de impacto. Manter coerência de versões entre prévia e confirmação.

---
## 13. Arquitetura técnica e contratos de integração

### 13.1. Estratégia de implementação

**Começar reutilizando capacidades já existentes, não iniciando outro subsistema paralelo.** O WOS mantém:

- Domain como fonte da verdade de lifecycle, dependency, Criterion, Evidence, Issue, Blocker, contrato e conclusão.
- Application como ponto de coordenação de autorização, idempotência, transaction/Outcome guard, versionamento, eventos e outbox.
- SQLite/PostgreSQL com paridade funcional, além de adapter Memory quando aplicável.
- HTTP, MCP, SDK Go e CLI como interfaces do **mesmo** estado, com regras e contratos coerentes.
- Navegador como camada de apresentação e orchestration de ações humanas legítimas; **não** como segundo motor de regras.

### 13.2. Reutilização imediata das leituras

| Leitura atual | Uso na nova UI |
|---|---|
| `POST O/workspace-view` | Inspector selecionado: header, lifecycle, placement, critérios, pessoas, entrega, atenção, capacidades, cobertura, relações, ação correta. |
| `POST O/workspace-cards` com `parent_ref` ou referências | Buscar filhos diretos/compact summaries no Kanban e enriquecer post-its carregados; limite vigente 1–100, default 50. |
| `GET O/workspace-timeline` | Histórico paginado por sujeito/descendentes com frontier estável. |
| `GET O/entity-notes/{id}` | Corpo completo de observação registrada. |
| Leitura de graph/continuity e searches existentes | Dependências, contexto, seleção, ligação entre Board e inspector. |
| `WorkAttention`/capabilities | Pendências reais, correções e próximo passo; não duplicar lógica no JS. |
| `workspace_rules` em capabilities | Tipos de EvidenceLink e propagação de Blocker realmente permitidos. |

**O** representa `/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}`. Leitura `POST` não efetua mutation e mantém proteção de autenticação e same-origin/CSRF conforme contrato atual.

### 13.3. Lacuna das leituras atuais para Kanban robusto

`WorkspaceCardSummary` traz `lifecycle`, `display_state`, critérios e `attention_reason`, mas nem sempre traz todos os dados necessários para **estágio de revisão assinado, agregados de progresso e contagens completas por coluna**. Além disso, uma lista de filhos paginada e filtrada localmente não pode ser tomada como a verdade sobre todos os filhos.

**Abordagem em duas etapas:**

- **V1 de interface:** compor Kanban a partir de `workspace-cards`, `workspace-view` e contexto por demanda; classificar somente quando os dados canônicos disponíveis sustentam o estágio; mostrar `desconhecido/parcial` caso contrário. Não fazer N+1 em centenas de cards.
- **V2 da API (necessária para Kanban com escala/contagens autoritativas):** adicionar **projeção de leitura**, não agregado de domínio, por exemplo `POST O/workspace-kanban`. Nome final sujeito ao catálogo de comandos vigente. O endpoint agrega em **um frame coerente** colunas, pré-requisitos, revisão, contadores e paginação.

Proposta de contrato **novo** (exemplo ilustrativo; não existe na base examinada):

```json
{
  "parent_ref": {
    "namespace_id": "NAMESPACE_ID",
    "outcome_id": "OUTCOME_ID",
    "kind": "objective",
    "id": "OBJECTIVE_ID"
  },
  "content": "tasks",
  "filters": { "search": "", "blocked_only": false, "assignee": null },
  "limit": 50,
  "cursor": null
}
```

Resposta sem converter versions por floating point:

```json
{
  "outcome_revision": "128",
  "evaluated_at": "2026-10-19T20:00:00Z",
  "parent": { "kind": "objective", "id": "OBJECTIVE_ID", "title": "Implementar API" },
  "summary": {
    "accepted_tasks": 4,
    "total_tasks": 6,
    "blocked_tasks": 1,
    "criteria_required": 3,
    "criteria_met": 2,
    "coverage": "complete"
  },
  "items": [
    {
      "ref": { "namespace_id": "NAMESPACE_ID", "outcome_id": "OUTCOME_ID", "kind": "work_item", "id": "TASK_ID" },
      "version": "5",
      "title": "Implementar endpoint",
      "lifecycle": "in_progress",
      "stage": "verification",
      "stage_reason": "pending_independent_review",
      "attention": ["awaiting_review"],
      "authority_state": "none",
      "criteria": { "required": 2, "met": 1, "pending": 1 }
    }
  ],
  "coverage": { "status": "partial", "next_cursor": "..." }
}
```

**Cuidados:** esse exemplo demonstra formato desejado, não implica que strings fictícias `NAMESPACE_ID` são IDs válidos para executar o endpoint. O servidor real deve usar IDs canônicos, validação e autenticação existentes. `summary` precisa ser separado da paginação dos cards; não deduzir `total_tasks` a partir de `items.length`. Se não houver contagem global completa, cobertura deve declará-la parcial/indisponível, sem fabricar zero. `stage` é projeção, não lifecycle persistido.

### 13.4. Backend: classificação autoritativa de estágio e pendência

Criar função de projeção única, testada em Domain/Application conforme responsabilidade real, por exemplo `ProjectKanbanStage`, **sem novo state machine do WOS**:

- Entrada: entidade canônica, contexto do Outcome, dependências, blocking, contrato/lease, review case ou entrega, criteria atuais, policy e instante de avaliação.
- Saída: `column`, `display_status`, `reason_codes`, `action_capabilities`, `coverage`.
- Respeitar precedência: Task realmente `done` vai para Aceitos; Task assinada com ReviewCase pendente vai para Verificação; correção solicitada vai para trabalho/atenção; Objective `achieved` vai para Aceitos; cancelled nunca vira concluído.
- Um Objective “pronto para verificação” é **eligible candidate**, não ReviewCase real. A interface usará etiqueta clara.
- Estados de dependência e Blocker são sobrepostos como badges, não transformados em conclusão.
- Não consultar material externo nem usar IA para decidir se um critério foi atendido.

### 13.5. Ações existentes que compõem os fluxos

| Intenção humana | Comando/serviço existente a consultar antes de propor novo |
|---|---|
| Criar Outcome/Objective/Task | `create_outcome`, `create_objective`, `create_work_item` e `create_plan_items`. |
| Definir/ajustar requisito | `add_criterion`, `revise_criterion`, `retire_criterion`. |
| Registrar avaliação | `record_criterion_assessment`, ou protocolo de review assinado aplicável. |
| Registrar artefato | `register_artifact_in_context`. |
| Registrar evidência com vínculo | `register_evidence_with_links`. |
| Atualização documental | `append_entity_note`. |
| Atividade sob contrato assinado | `append_work_activity` e controles assinados existentes. |
| Associar Task a Objective | `set_work_item_objective` com preview guardado. |
| Associar Objective ao pai | `set_objective_parent` quando elegível. |
| Dependências | `add_dependency` / `remove_dependency`. |
| Reportar impedimento | `create_issue`, `report_issue_with_blocker` ou `create_blocker`. |
| Resolver Issue/impactos | `resolve_issue` / `resolve_issue_and_blockers` / `resolve_blocker`. |
| Concluir Objective/Outcome | `achieve_objective` / `achieve_outcome` com revisão das pré-condições. |
| Assumir/entregar/revisar Task | Comandos do protocolo de execução efetivamente habilitado, inclusive Profiles assinados quando necessários. |

**Nenhuma nova action de frontend** pode fazer duas escritas incompatíveis em sequência quando for necessária atomicidade (por exemplo, Evidence + EvidenceLink). Reusar o comando composto existente. Qualquer novo comando composto passa pelo mesmo Application pipeline transacional; não chamar handlers públicos recursivamente.

### 13.6. Evolução persistida da política de conclusão

Evoluir Outcome de modo aditivo com `completion_policy` versionada (`compatible`, `strict_work`), ou uma configuração tipada equivalente, definida no agregado e persistida pelos adapters. Ela é selecionável na criação ou edição permitida; alteração após trabalho aceito deve passar por impacto/autoridade explícitos.

Adicionar ao serviço de conclusão:

1. Consultar policy do Outcome sob lock de coordenação.
2. Calcular o conjunto de Tasks diretas/subobjetivos aplicáveis com consultas não truncadas ou checagens paginadas/coerentes no servidor; **não** usar Board/caches client-side.
3. Verificar lifecycles, cancelamentos, revisões e identidade do autor que reconhece exceções.
4. Agregar blockers, hard dependencies, Critérios e requisitos de avaliação atuais já existentes.
5. Produzir `reason_code` estável (`required_child_task_open`, `required_child_objective_open`, `cancelled_work_requires_ack`, etc.), referências de filhos afetados e remediações; nomes exibidos obtidos por leitura autorizada.
6. Na conclusão, vincular o resultado à revisão/versão e à relação de obrigações relevantes sem reescrever conclusões históricas.
7. Gerar eventos/outbox pelo pipeline canônico.

**Migrations:** mudanças forward-only para SQLite e PostgreSQL, com checagem de versão, checksum e teste de upgrade a partir dos schemas aceitos. `compatible` deve ser o fallback explícito para records antigos, sem backfill agressivo que altere entregas passadas. Garantir comportamento equivalente em Memory/SQLite/PostgreSQL.

### 13.7. Versionamento, authorization e idempotência

- Enviar `expected_version` e `expected_outcome_revision`/`observed_outcome_revision` onde o contrato requer; usar strings decimais exatas para projeções novas. Não converter através de `Number` JavaScript.
- Cada mutation tem **uma intenção exata** e `Idempotency-Key` válido. Em resultado incerto, **reconciliar a intenção original**, não reenviar dados alterados com outra chave sem verificar.
- Capabilities são previsões de disponibilidade, não concessões de autoridade. O serviço reavalia tudo ao gravar.
- Proteger fluxo de executor/revisor independente, assinatura, grupo de separação de deveres, origem de credenciais e fencing.
- Limpar drafts e dados privados quando muda identidade, Namespace, Outcome ou permissão; respostas antigas não devem restaurar state.
- Preview de reparent, blocking ou conclusão pode caducar. Mostrar mudança, manter rascunho local e solicitar confirmação sobre revisão atual.
- Quando a UI conhece somente cobertura parcial, proibir “tudo concluído” como conclusão derivada do cliente; servidor deve resolver integralidade.

### 13.8. Integração com CLI, MCP e SDK

- O **novo Kanban é principalmente UX do navegador**. CLI e agentes não passam a ter que ler cards ou um YAML novo.
- Quando nova leitura `workspace-kanban` ou nova policy forem APIs públicas, regenerar **OpenAPI, catálogo de comandos, MCP, SDK Go e CLI de inspeção** de forma compatível conforme padrões do projeto, sem caminhos divergentes de autorização.
- Novas mutations de política usam o mesmo domínio/Application e transportes conforme política de exposição; não dar uma exceção ao browser que CLI/MCP não tenham direito de acessar no seu escopo legítimo.
- O WOS continua **não executando/schedulando Agents**, não sendo um runtime. A elegibilidade após conclusão pode ser observada pelos sistemas externos; nenhuma ação de UI implica autoexecução de um agente.

---

## 14. Design visual, responsividade e comportamento da interface

### 14.1. Identidade existente — preservação obrigatória

| Token visual | Valor utilizado nesta direção |
|---|---|
| Canvas | `#EFEDE8` |
| Workspace | `#F5F3EF` |
| Surface | `#FFFDF9` |
| Ink | `#292620` |
| Copper | `#A7441D` |
| Selected | `#F1DFD0` |
| Grid | `#C2BCB2` |
| Brand | `#FF5500` |

Usar a hierarquia e os tokens já vigentes no CSS da branch como fonte operacional, sem duplicar cores codificadas arbitrariamente. Preservar traços editoriais, robôs, post-its, ilustrações, grid, texturas, `Instrument Serif`/`DM Sans`, componentes e interações do Board. A intenção é **melhorar hierarquia e densidade**, não refazer a marca.

### 14.2. Padrão de inspector

- Cabeçalho compacto, fixo no painel; nome/estado quebram linhas sem deslocar controles globais.
- CTA principal destacado, largura total; ações secundárias em linha apenas quando houver espaço.
- Dois disclosures visualmente discretos, com ícones, contadores e foco visível.
- Área de conteúdo com rolagem independente; footer de ações acessível sem cobrir formulário.
- Materiais, critérios e atividade usam linhas compactas; não renderizar todos os detalhes completos no primeiro nível.
- Referências a pessoas são rótulos explícitos (`Responsável`, `Executor autorizado`, `Revisor`), não ícones ambíguos de presença.
- Exibir avisos de coverage, estados incertos e ações bloqueadas com linguagem humana.

### 14.3. Geometria do modal Kanban

**Especificação de layout proposta, calibrável após screenshots e testes:**

- Modal central dentro do viewport útil da aplicação (não como navegação permanente), largura preferencial `min(94vw, 1680px)` com largura mínima **não forçada** em viewports menores; altura `min(88dvh, available-stage-height)` e rolagens internas intencionais. Usar `dvh`/medidas reais do palco e fallback compatível onde necessário.
- Em desktop, topo com breadcrumb, título, progress summary, fechar; abaixo toolbar curta (busca, filtros, criar); colunas ocupam o restante da altura.
- Largura de coluna responsiva (por exemplo `minmax(240px, 1fr)` em telas amplas, mínimo preservado em HD), com rolagem horizontal **dentro do Kanban** quando quatro colunas não couberem confortavelmente.
- Modal em mobile: surface quase full-screen, breadcrumb resumido, colunas em swipe horizontal ou seletor de etapa acessível, um fluxo de detalhe por vez.
- Painel de detalhe interno no modal não pode estreitar quatro colunas a ponto de ilegibilidade. Em HD, usar overlay dentro do modal ou substituir coluna temporariamente; QHD pode usar inspeção docked.
- Não criar duas scrollbars verticais competindo em todo o documento: modal tem barra de conteúdo; colunas podem ter scroll próprio com posição preservada; ações e navegação continuam visíveis.
- Quando o usuário fecha o modal, o viewport original do mapa (zoom, pan, seleção, colapsos) deve ser idêntico.

### 14.4. Viewports a validar

- Desktop compacto: **1280×720**, **1366×768**, além de **1024×768** onde o layout suportar.
- Full HD: **1920×1080**.
- QHD/alta densidade: **2560×1440**, incluindo escala/nativo zoom relevantes.
- 4K: **3840×2160** para densidade e distribuição, não alongar cards até perder leitura.
- Mobile: **320×740**, **390×844**, **430×932** e teclado virtual real quando possível.
- Zoom real do navegador **125%, 150%, 200%**; não confundir apenas redução do viewport com teste de native browser zoom.

A interface precisa conservar a mesma disposição conceitual e densidade apropriada em telas grandes. O modal não deve virar um “oceano vazio” em QHD nem esmagar post-its em HD.

### 14.5. Acessibilidade e teclado

- Modal semanticamente de diálogo: título acessível, foco inicial apropriado, **focus trap**, restauração de foco no botão que abriu, `Escape` com hierarquia de fechamento; mapa atrás inerte enquanto aberto.
- Disclosures com `aria-expanded`, aria-label e estado explícito; botões de adicionar material e critério com rótulos inequívocos.
- Cards navegáveis por teclado, Enter/Space para inspecionar; alternativas à movimentação por drag.
- Status não representado apenas por cor; badge/texto/iconografia redundantes.
- Alvos interativos pelo menos **44×44 CSS px** em mobile/controles pertinentes, coerentes com baseline.
- Formularios com labels, mensagem de erro associada ao campo, justificativas de avaliações e alertas anunciáveis.
- Preservar digitação e rascunhos, inclusive durante refresh e erros de rede.
- Não duplicar ID de elementos ao montar inspector no Board/modal. Reusar host do inspector ou renderizar views semanticamente equivalentes com gerenciamento correto de foco e cleanup.

### 14.6. Linguagem e microcopy

**Preferir:** `Condição de sucesso`, `Evidência`, `Material produzido`, `Enviar entrega`, `Verificar critérios`, `Correção solicitada`, `Quem está trabalhando`, `O que falta`, `Abrir Kanban`, `Ver dependência`, `Verificar resultado`.

**Deixar para níveis técnicos:** `Assessment`, `ExpectedVersion`, `ReviewCaseID`, `FencingToken`, `CriterionRevision`, `SubmissionDigest`, `precondition_failed`, `EntityRef`, `WorkItem`.

Evitar mensagem genérica “Não foi possível”. Especificar condição real, ação possível e entidade afetada. Ex.: “Não é possível aceitar este objetivo: a Task ‘Testar reembolso parcial’ ainda está em execução. [Abrir Task]”.

---

## 15. Tratamento de erros, concorrência e estados de carregamento

### 15.1. Estados de interface

Cada área carregável possui quatro estados úteis: **carregando**, **carregado completo**, **parcial / mais registros** e **indisponível/erro**. Lista vazia só é exibida como “nenhum registro” quando o backend confirma cobertura completa. Seleção em modal não carrega a árvore documental inteira sem necessidade.

### 15.2. Conflitos diferentes exigem soluções diferentes

| Evento | Comportamento correto |
|---|---|
| Precondição de domínio não atendida | Mostrar motivo tipado e ação de reparo, preservando rascunho. |
| Mudança concorrente (`version_conflict`) | Mostrar estado atual vs rascunho, exigir decisão explícita; não rebasear silenciosamente. |
| Impact preview stale | Atualizar prévia, limpar confirmação anterior, conservar justificativa e pedir nova aprovação. |
| Timeout / resposta incerta | Consultar receipt/recovery da intenção original; não emitir operação diferente com a mesma chave. |
| Falha de leitura após escrita confirmada | Atualizar leitura; não repetir mutation. |
| Revogação de permissão | Interromper edição, limpar dados protegidos, exibir feedback e fechar ações incompatíveis. |
| Contrato/lease expirado | Mostrar autoridade expirada e caminho legítimo de retomar/reconciliar. |
| Evidence retirada | Marcar prova como inaplicável, não aceitar critério com histórico antigo sem reassessment exigido. |
| Mudança de parent durante interação | Invalidar páginas antigas, atualizar breadcrumbs e posição; não enviar comando ao parent incorreto. |
| Desconexão/latência | Mostrar estado de envio/reconciliação e impedir clique duplo; permitir leitura do último snapshot com aviso apropriado. |

### 15.3. Requisitos de desempenho

**Alvos de projeto, não benchmarks medidos nesta proposta:**

- Inspector com informação básica primeiro; detalhes pesados só sob demanda e paginados.
- Um Kanban com dezenas/centenas de Tasks usa leitura em lote e/ou projeção dedicada, jamais N requisições por card.
- Colunas com scroll virtualizado quando o volume exigir; preservar acessibilidade e busca do backend.
- Interações locais como abrir disclosure, selecionar filtro e retornar breadcrumb devem ser imediatas, sem repintar o Canvas inteiro.
- Testar p95 de abertura inspector e Kanban em ambientes de referência antes de definir SLOs numéricos. A PR atual contém testes de performance de leituras, mas suas medições de serviço/serialização não equivalem a latência HTTP de navegador real.
- Evitar refetch desnecessário de histórico completo a cada seleção; coerência e autorização têm prioridade sobre cache agressivo.

---

## 16. Arquivos e módulos candidatos a alteração

> **Atenção:** os caminhos a seguir foram identificados na branch base examinada. São **alvos de inspeção/edição**, não afirmações de que novos componentes ou endpoints já existem. O agente implementador deve verificar o HEAD efetivo antes de editar.

### 16.1. Frontend embutido

| Arquivo existente | Responsabilidade nesta evolução |
|---|---|
| `packages/wos-api/web/assets/workspace-canvas.js` | Preservar mapa, seleção e câmera; acrescentar ponto de abertura do Kanban a partir da seleção quando aplicável. |
| `packages/wos-api/web/assets/workspace-card-summary.js` | Resumo operacional breve dos post-its; contadores e indicador de próximo passo sem misturar coverage. |
| `packages/wos-api/web/assets/canvas-data.js` | Integrar leitura coerente e preservação de contexto; manter camadas work/records. |
| `packages/wos-api/web/assets/workspace-inspector.js` | Painel compacto, alternância inline/modal, preservação de nós, drafts, foco e scroll. |
| `packages/wos-api/web/assets/workspace-view.js` | Redesenhar o conteúdo: resumo e dois grupos expansíveis; distinguir variantes por tipo. |
| `packages/wos-api/web/assets/workspace-action-menu.js` | Ranking contextual de CTA e categorias de ações secundárias, baseado em capabilities. |
| `packages/wos-api/web/assets/workspace-actions.js` | Conectar ações permitidas sem derivar autorização no browser. |
| `packages/wos-api/web/assets/workspace-evidence-form.js` | Simplificar wizard contextual de Evidence, escolha de Criterion e relação. |
| `packages/wos-api/web/assets/workspace-conclusion-form.js` | Prévia dos gates/descendentes da nova policy; exceções explícitas, versões e remediação. |
| `packages/wos-api/web/assets/app.js` | Orquestração de seleção, rotas internas, estado do modal e refresh após mutations. |
| `packages/wos-api/web/assets/workspace-board.css` | Card e layout do Kanban; reutilizar tokens. |
| `packages/wos-api/web/assets/workspace-workflows.css` | Formulários guiados, critérios, evidências e entregas. |
| `packages/wos-api/web/assets/workspace-surfaces.css`, `workspace-view.css`, `workspace-dialogs.css` | Inspector, modal, overlay, foco, rolamento e responsividade. |
| `packages/wos-api/web/assets/en-US.js` | Novos textos/erros traduzíveis coerentes com idioma base vigente; não inserir hardcode misturado. |

**Novos módulos candidatos** (nomes propostos; dividir apenas se realmente reduzir acoplamento):

- `workspace-kanban.js`: composição do modal, colunas, navegação contextual, filtros, paginação e seleção.
- `workspace-kanban.css`: dimensão/scroll/responsividade específicos, usando tokens existentes.
- `workspace-acceptance-flow.js`: wizard compartilhado de critérios, Evidence e decisão; nunca calcula eligibility final localmente.
- `workspace-materials.js`: apresentação/entrada simples de Artifact e Evidence sem copiar validação do backend.
- `workspace-inspector-summary.js`: apresentação por tipo/estado, caso o arquivo atual se torne excessivamente acoplado.

Não duplicar o renderer do mapa nem a implementação inteira do inspector. Preferir compor os componentes existentes.

### 16.2. Domain/Application

| Arquivo/camada candidata | Modificação esperada |
|---|---|
| `packages/wos-core/domain/outcome.go` | Policy de conclusão aditiva, validações e serialização compatível quando aprovadas. |
| `packages/wos-core/domain/objective.go` | Sem alterar lifecycle base; reutilizar avaliações/conclusão e regras de parent. |
| `packages/wos-core/domain/work_item.go` | Sem novo lifecycle de Kanban; manter lease/contract e conclusão canônica. |
| `packages/wos-core/domain/work_item_operational_state.go` | Reusar display state e readiness; auxiliar classificador do Kanban se fizer sentido neste boundary. |
| `packages/wos-core/domain/criterion.go` | Manter revisionamento, `RequiredSatisfied`, `waived` e avaliação aplicável. |
| `packages/wos-core/application/workspace_read_types.go` | Ampliar projeções necessárias, cobertura e candidate `KanbanView` com versões exatas. |
| `packages/wos-core/application/workspace_cards.go` | Base de filhos diretos e summaries; evitar N+1 e contagens parciais fraudulentas. |
| `packages/wos-core/application/workspace_capabilities.go` | Expor elegibilidade e reason codes para CTAs sem escolher autoridade arbitrariamente. |
| `packages/wos-core/application/workspace_preconditions.go` | Gates atômicos de conclusão estrita aplicados só sob policy. |
| `packages/wos-core/application/workspace_conclusion.go` | Exibir impacto, itens abertos e cancelamentos reconhecidos, conforme policy. |
| `packages/wos-core/application/work_attention.go` | Pendências de Task, correção e eventual próximo passo; preservar projeção read-only. |
| `packages/wos-core/application/workspace_timeline.go` | Reusar fronteira/histórico; projetar apresentação resumida sem nova fonte de eventos. |
| `packages/wos-core/application/workspace_proof.go` | Preservar validação de Evidence ligada ao Assessment. |
| `packages/wos-core/application/workspace_blocker_preview.go` | Prévia de propagação e lista/contagem de afetados. |
| `packages/wos-core/application/workspace_placement.go` | Mudança de parent explícita, sem impacto do drag local de apresentação. |

### 16.3. Adapters e contratos públicos

- Persistir policy nova em SQLite/PostgreSQL (e Memory), com migrations numeradas forward-only, checksums e testes de rollback/restauração documentada.
- Expor novas projeções no HTTP e MCP **somente quando de fato criadas**, com OpenAPI e schema gerados em `packages/wos-api/openapi.yaml` e `packages/wos-api/commands.openapi.json` conforme geradores existentes.
- Verificar o `packages/wos-sdk-go` e `packages/wos-cli` para paridade de `completion_policy` e leitura Kanban se a superfície pública for ampliada.
- Não introduzir Cloud, scheduler, engine de Agent, armazenamento de arquivos, serviço de billing ou dependência da Woobe.

---

## 17. Plano de execução por etapas e dependências

### Fase 0 — Congelar baseline e caracterizar contratos

**Objetivo:** garantir que a mudança não destrua o que já funciona.

1. Confirmar HEAD e estado do PR #116; não assumir que o SHA desta proposta ainda é o último.
2. Registrar screenshot/snapshot atual do Board, inspector em cada tipo, Task Board, mobile e QHD; incluir renderização com registros parciais.
3. Caracterizar commands/capabilities reais, revisões de Criteria, signed protocols, Errors, Assessment, Problem/Blocker, Task `done` e conclusão de Objective/Outcome.
4. Construir fixtures reais: Outcome com Objective obrigatório/opcional; Tasks diretas; hierarquia de subobjetivos; Task signed; revisor diferente; Issue sem Blocker; blocker com propagação; Evidence contraditória; cancelamentos; cursor parcial; concorrência.
5. Definir métricas de compreensão iniciais e revisar termos de UI.

**Aceite:** baseline reproduzível; testes existentes intactos; lista fechada de operações reais e de gaps.

### Fase 1 — Inspector compactado (sem reescrever domínio)

1. Criar layout global 6 blocos (identificação, resumo, próxima ação, rápidas, Detalhes, Atividade).
2. Implementar variações Outcome/Objective/Task/Issue/Blocker.
3. Padronizar `Detalhes e gerenciamento` e `Histórico de atividade`, ambos recolhidos por padrão.
4. Introduzir linha “O que falta” derivada de server facts; seção de materiais com Artifact/Evidence separados.
5. Reutilizar controles existentes de prévias, referência, leitura de critérios e action menu; oferecer texto explicativo de ações bloqueadas.
6. Garantir tamanho/scroll do inspector no Board HD e mobile, com toolbar e mapa intactos.

**Aceite:** usuário encontra CTA correto sem rolar; histórico não ocupa tela inicial; todas as capacidades antigas continuam alcançáveis.

### Fase 2 — Kanban modal contextual (Outcome e Objective)

1. Criar modal e state machine **visual**: `closed`, `outcome_objectives`, `outcome_direct_tasks`, `objective_tasks`, `objective_subobjectives`, `item_inspection`.
2. Implementar navegação por breadcrumb, retorno e restauração da câmera do Canvas.
3. Renderizar colunas e estado canônico/derivado; bloquear falsa interpretação de review/autoridade.
4. Implementar paginação, busca, filtros, badges de dependência/Blocker, acessibilidade e mobile.
5. Criar/abrir Task ou Objective sempre no contexto correto, conservando drafts e seleção.
6. Integrar com fontes existentes antes de construir nova API de leitura. Caso coverage/performance exijam, evoluir para `workspace-kanban` (projeção agregada).
7. Implementar ações de transição pelo menu contextual; drag entre colunas somente após fluxos autorizados inteiramente testados.

**Aceite:** Board original invariável; modal abre/fecha sem perder câmera; quadros têm parent correto, indicadores honestos e ações legítimas.

### Fase 3 — Critérios e materiais em fluxo guiado

1. Checklist de critérios com `met`/`not_met`/`inconclusive`, waiver avançado e justificativa; estado derivado da avaliação aplicável.
2. Adicionar/editar Criterion pelos comandos reais e com guardas.
3. Wizard “Adicionar material” dividido em Artifact/Evidence e associação do Criterion, evidência existente e stance.
4. Integrar submissão/revisão de Task por protocolo, inclusive signed-host handoff e correções solicitadas.
5. Exibir pacote de revisão coerente com submission/digest, Criteria e fontes; nunca avaliar automaticamente Evidence.
6. Histórico recente compacto e “ver todos” ligado à timeline verdadeira.

**Aceite:** executor consegue anexar materiais e entregar; revisor autorizado avalia critérios; conclusão depende de avaliação legítima; há testes de falsa aprovação/revision stale.

### Fase 4 — Conclusão estrita opt-in e pendências corretivas

1. Projetar e persistir `completion_policy` em Outcome, com `compatible` para registros/API antigos e `strict_work` para novos Outcomes criados pela UI (após aprovação do produto).
2. Implementar gates transacionais para Tasks diretas, Tasks sob Objective, subobjetivos, critérios, blockers, dependências, conclusão e cancelamentos explícitos.
3. Atualizar preview e indicadores da UI, distinguindo “Task aceita”, “Task cancelada” e “Critério dispensado”.
4. Inserir mensagens específicas e links diretos a itens pendentes.
5. Agrupar correções e Issues por entidade afetada, sem nova entidade `Block` na primeira entrega.
6. Garantir que Outcome reprovado continue ativo e possa receber trabalho corretivo.

**Aceite:** conclusão prematura é recusada pelo backend sob policy estrita; antigos Outcomes continuam com semântica compatível; exceptions/cancelamentos são visíveis e auditáveis.

### Fase 5 — Endurecimento, documentação e qualificação

1. Testes integrados de versões, idempotência, signed protocols, Identity, scope, review, Evidence, Blocker, conflitos e restore.
2. Screenshots por viewport/estado e verificação visual sem clipping.
3. Testes de acessibilidade e teclado; cobertura de zoom real e touch.
4. Estudo de compreensão com participantes independentes e resultados registrados, separado dos testes automatizados.
5. Regenerar OpenAPI, catálogo e SDK/CLI/MCP quando existir contrato novo.
6. Atualizar guias de UX, ADR (se política de domínio nova), changelog e exemplos; manter README principal focado, sem inflar com detalhes internos.
7. Qualificar PR e migrations; **não** fundir/publicar automaticamente.

**Aceite:** invariantes técnicas comprovadas, estudo de compreensão executado ou pendência explicitamente reconhecida, protocolo de release respeitado.

### 17.1. Grafo de dependência das fases

```text
F0 — Baseline
    ├── F1 — Inspector compacto ───────┐
    └── F2 — Kanban modal ─────────────┤
                                       ├── F3 — Critérios / Material / Revisão
                                       │                │
                                       └────────────────┴── F4 — Política estrita
                                                            │
                                                            └── F5 — Qualificação
```

F1 e F2 podem progredir em paralelo após as interfaces compartilhadas de seleção e projeções serem estabilizadas. F4 necessita da UI de verificação e dos cenários de hierarquia completos para que a policy seja compreensível e testável.

---

## 18. Matriz extensa de testes de aceitação

> Convenção: **Dado → Quando → Então**. Repetir cenários relevantes em Memory/SQLite/PostgreSQL; executar API real e Browser no mesmo ciclo. “Passou no Playwright” não significa que teste humano foi executado.

| ID | Cenário | Critério verificável |
|---|---|---|
| UX-001 | Selecionar Outcome no mapa | Inspector compacta resumo e apresenta `Abrir Kanban de objetivos` sem deslocar a câmera. |
| UX-002 | Selecionar Objective no mapa | Mostra `Abrir Kanban de tarefas`, parent verdadeiro, critérios e progresso. |
| UX-003 | Selecionar Task | Mostra próxima ação correta ao protocolo/estado, sem botão de Kanban. |
| UX-004 | Selecionar Issue/Blocker | Renderiza fatos próprios sem falso executor nem checklist inválido. |
| UX-005 | Abrir/fechar Kanban | Modal não navega para página distinta; restitui pan, zoom, seleção e foco. |
| UX-006 | Entrar em Objective dentro do modal | Troca contexto do único modal, breadcrumb e dados de parent corretos. |
| UX-007 | Task direta do Outcome | Visível em segmento específico, nunca “Objective fictício”. |
| UX-008 | Subobjetivo aninhado | Preserva caminho e relação obrigatória global sem achatar a hierarquia semanticamente. |
| UX-009 | Filtros e paginação | Não contam apenas cards carregados como total; coverage correta. |
| UX-010 | Desktop 1280×720 / 1366×768 | Cabeçalho, botões, Kanban e inspector alcançáveis sem overflow oculto. |
| UX-011 | QHD / 4K | Densidade visual mantida, colunas e post-its legíveis, sem excesso de espaço vazio. |
| UX-012 | Mobile e zoom nativo | Navegação, rolagem, formulários e controles operáveis com teclado/touch. |
| UX-013 | Keyboard-only | Abrir/modal/navegar/avaliar/fechar/foco devolvido; alternativa ao drag. |
| UX-014 | Conteúdo parcial | Mostra “parcial/desconhecido”, nunca “0 pendentes” inventado. |
| CREATE-001 | Criar Outcome simples | Recolhe nome+estado desejado, mantém draft, sem Criterion inventado. |
| CREATE-002 | Criar Objective | Parent correto, flag global requerida visível e lifecycle planned. |
| CREATE-003 | Criar Task dentro de Objective | Backlog, parent correto, sem contrato/claim. |
| CREATE-004 | Criar batch misto | Previsão correta e atomicidade total na gravação/rejeição. |
| CRIT-001 | Critério ativo ausente na ativação | Explica condição, permite criar Criterion e repetir intenção após revisão. |
| CRIT-002 | `met` autorizado | Assessment persistido com versão/rationale/evidência aplicável. |
| CRIT-003 | `not_met`/`inconclusive` | Não conclui item, mostra apontamento/ação corretiva. |
| CRIT-004 | `waived` autorizado | Exibido distinto de atendido, com justificativa e permission checks. |
| CRIT-005 | Revisor não elegível | Browser não permite/salva aprovação; signed path não é contornado. |
| CRIT-006 | Criterion revisado após Assessment | Avaliação antiga não vale automaticamente para revisão nova. |
| CRIT-007 | Nova Submission com digest diferente | Assessment de submissão antiga não aprova nova entrega. |
| CRIT-008 | Evidence retirada | Prova retracted gera reavaliação e impede conclusão quando aplicável. |
| MAT-001 | Registrar Artifact | Relation correta ao contexto; não cria Evidence/Approval implícitos. |
| MAT-002 | Registrar Evidence por Criterion | Source, stance, rationale e vínculo criados atomicamente. |
| MAT-003 | Reusar Evidence existente | Não duplica Evidence; vínculo independente mantém sua stance. |
| MAT-004 | Evidence contraditória | Visível ao revisor e não escondida pela contagem de provas favoráveis. |
| MAT-005 | Registro externo sem fetch | UI informa origem fornecida, não afirma inspeção pelo WOS. |
| TASK-001 | Task todo pronta | Aquisição exige identidade e protocolo correto; card só reflete sucesso confirmado. |
| TASK-002 | Task dependente de outra | Ação de início não ignora hard dependency. |
| TASK-003 | Task assinada entregue | ReviewCase verdadeiro leva a Verificação e vincula submission atual. |
| TASK-004 | Revisor independente | Executor não se autoaprova sob política separada de identidade. |
| TASK-005 | Correção solicitada | Histórico/finding preservados e nova execução exige autoridade própria. |
| TASK-006 | Task done | Coluna Aceitos com Conclusion; sem execução automática de dependente. |
| OBJ-001 | Objective planned abre Kanban | Navegação não altera lifecycle nem requer Start artificial. |
| OBJ-002 | Objective com Tasks done, Criterion pendente | Não aceita automaticamente; mostra `Verificar critérios`. |
| OUT-001 | Outcome com Objective requerido não achieved | Conclusão recusada, mesmo se esse Objective está fora dos cards carregados. |
| OUT-002 | Outcome compatível com descendentes abertos | Semântica existente preservada, observada a legislação de gates atuais. |
| OUT-003 | `strict_work` com Task aberta | Conclusão recusada no backend com referência à Task. |
| OUT-004 | `strict_work` com Tasks diretas abertas | Outcome não é aceito omitindo trabalho diretamente vinculado. |
| OUT-005 | `strict_work` com cancelamentos | Exibe e exige reconhecimento explícito; cancelado não conta aceito. |
| OUT-006 | Verificação final reprovada | Outcome continua ativo/corrigível, sem Blocker subtree automático. |
| ISSUE-001 | Issue sem Blocker | Problema registrado, trabalho não é falsamente bloqueado. |
| ISSUE-002 | Issue com Blocker | Prévia e gravação atômicas; impacto correto por escopo. |
| ISSUE-003 | Resolver Issue | Não libera automaticamente Blockers não selecionados. |
| ISSUE-004 | Criar Task corretiva | Vincular ao Objective legítimo com critérios próprios quando necessários. |
| HIST-001 | Abrir Histórico | Mostrar timeline paginada com autor, tempo, tipo e sujeito correto. |
| HIST-002 | Corrigir nota | Cria nova nota de correção imutável, mantém original. |
| HIST-003 | Ler descendentes | O evento não troca o autor/sujeito real pelo pai. |
| ERR-001 | Version conflict | Preserva rascunho e exige revisão manual do estado atual. |
| ERR-002 | Outcome revision mudou após prévia | Prévia antiga é rejeitada; acknowledgement perde validade. |
| ERR-003 | Resposta incerta | Recovery com intenção original, sem duplicidade. |
| ERR-004 | Sessão revogada em modal | Fecha ações privadas; resposta tardia não restaura informações. |
| ERR-005 | Falha após mutation confirmada | Atualiza read sem repetir write. |
| API-001 | Nova projeção Kanban | Retorna revisões exatas, escopo autorizado, coverage honesta e paginação estável. |
| API-002 | Paridade de adapters | Migrations, política e gates idênticos em SQLite/PostgreSQL. |
| API-003 | Paridade de transportes | HTTP/MCP/SDK/CLI preservam regras de autorização, versão e idempotência. |
| A11Y-001 | Leitor de tela | Headings, status, modal, disclosures e mensagens associados corretamente. |
| PERF-001 | Kanban grande | Sem N+1; medição reproduzível com fixture e ambiente declarados. |

### 18.1. Conjunto mínimo de fixtures

- **F01:** Outcome draft vazio, nenhum Criterion; usuário adiciona objetivos e Tasks.
- **F02:** Outcome ativo com Objective obrigatório, Objective opcional e Tasks diretas.
- **F03:** Hierarquia com três níveis de Objectives, Tasks e dependências hard/advisory.
- **F04:** Task em `backlog`, `todo-ready`, `todo-blocked`, `in_progress`, `done`, `cancelled`.
- **F05:** Signed v2 com executor A, revisor B, submissão vigente, review pendente e correção.
- **F06:** Critérios nos modos attestation/evidence_review/external_evaluation; versões anteriores, waiver, retracted Evidence.
- **F07:** Issue isolada, Issue causadora de Blocker, Blocker externo, direct/subtree, liberação parcial.
- **F08:** Mais que 100 filhos/documentos e páginas parciais; busca/ordenação/contagem real.
- **F09:** Alteração concorrente de Criterion, parent e Conclusion preview com intenção aberta.
- **F10:** Outcome com policy `compatible` e outro `strict_work`, incluindo cancelamento e trabalho direto.
- **F11:** Permissões distintas para reader, planner, executor, reviewer e administrador.
- **F12:** Mobile e zoom com inspector/form longos e nomes de trabalho extensos.

---

## 19. Métricas de experiência e estudo humano

### 19.1. Métricas sugeridas — metas, **não medidas**

| Jornada observável | Meta inicial proposta |
|---|---|
| Criar Outcome + Objective + Task | Sem leitura de documentação nem formulários técnicos obrigatórios fora do fluxo esperado. |
| Encontrar como administrar Tasks de um Objective | Até 2 ações após selecionar seu post-it (`Abrir Kanban`). |
| Identificar o próximo trabalho disponível | Até 30 segundos em fixture de dificuldade média, com indicador correto. |
| Compreender por que Task não pode iniciar | Identificar dependência/Blocker e abrir o item causador sem ajuda. |
| Registrar Artifact e Evidence no Criterion correto | Operação contextual sem abrir três áreas independentes. |
| Distinguir comprovante e aceitação | Usuário explicar que anexar prova não aprova o trabalho. |
| Identificar quem executa | Usuário distinguir responsável declarado de executor autorizado. |
| Detectar rejeição e correção | Localizar apontamento e próxima ação em até 30 segundos. |
| Retomar Board após modal | Pan/zoom/foco idênticos. |

### 19.2. Protocolo de teste

Reusar/estender `docs/ux/operational-workspace/human-study.md`, cuja base declara estudo independente ainda não executado. Recrutar pelo menos **5 participantes independentes**, fixar coorte antes de examinar resultados, usar instalação de teste com dados descartáveis e registrar tempo, assistência, erro, êxito e abandono. Meta de aceitação inicial: **≥4/5** completam cada jornada crítica sem assistência; publicar taxas brutas e falhas. Não substituir teste de pessoas por screenshot ou jornada automatizada. Se faltarem participantes, registrar pendência real e **não** alegar que usabilidade foi validada.

### 19.3. Métricas instrumentáveis sem dados pessoais extras

- Tempo/clicks até abrir Kanban a partir do post-it.
- Tempo até encontrar o critério pendente.
- Taxa de entrada e conclusão do fluxo `Adicionar material`.
- Erros de precondição mostrados e remediação efetuada.
- Falhas de confirmação por preview stale e recuperação correta.
- Relação entre visualizações de Canvas, Kanban e inspector, sem capturar material sensível das Tasks.

Essas métricas não substituem análise qualitativa de compreensão; instrumentar apenas o necessário, respeitando autorização/privacidade.

---

## 20. Critérios finais de aprovação do trabalho

A implementação só deverá ser declarada funcionalmente completa quando todos os critérios abaixo forem efetivamente verificados:

### 20.1. UX/Board

- [ ] O Board original continua sendo a visualização espacial principal, com mesmos post-its, identidade, motor, pan/zoom e ligações.
- [ ] Um clique em Outcome/Objective/Task/Issue/Blocker abre um inspector compacto pertinente ao tipo.
- [ ] O inspector tem **exatamente dois grupos expansíveis principais**: `Detalhes e gerenciamento` e `Histórico de atividade`.
- [ ] Outcome e Objective mostram **Abrir Kanban** como operação saliente, sem esconder operações críticas de preparação.
- [ ] Modal Kanban abre sobre Board, não abre segunda página nem empilha Kanbans.
- [ ] Navegação Outcome → Objective → Task preserva breadcrumb, filtros, seleção, foco e scroll.
- [ ] Ao fechar, o Canvas retorna ao mesmo pan/zoom/seleção anteriores.
- [ ] Task possui um fluxo claro de execução/entrega, não um Kanban próprio.
- [ ] Critérios, Evidence, Artifacts e outras entidades documentais não poluem a camada principal do mapa.
- [ ] A interface é legível, acessível e utilizável em HD, Full HD, QHD, 4K e mobile.

### 20.2. Domínio e operações

- [ ] Criar plano não cria lease/contract nem exige preencher propriedades não necessárias naquele momento.
- [ ] Checklist registra Assessments legítimos com revisão, Evidence e identity conforme modo/política.
- [ ] Artifact e Evidence aparecem separados e podem ser relacionados corretamente, sem equivalência automática.
- [ ] A revisão de Task respeita a submissão atual, revisão independente e correções assinadas quando exigidas.
- [ ] Objective/Outcome só concluem com Criteria próprios satisfeitos e gates do domínio.
- [ ] `strict_work` bloqueia conclusão prematura; `compatible` preserva semântica antiga.
- [ ] Cancelamento é identificado e justificado, não contado como Task aceita.
- [ ] Issue e Blocker são entidades diferentes; rejeição de entrega não gera propagação indiscriminada.
- [ ] Histórico é paginado, auditável, com autor/sujeito e correção append-only.
- [ ] Mudanças de posição visual não implicam comandos de relação/ownership.
- [ ] Auth, versões, idempotência, recovery, migrations e adapters preservam integridade.

### 20.3. Qualificação

- [ ] Regressões de browser/Go/API e compatibilidade executadas no HEAD final, sem simplesmente copiar números de PR anterior.
- [ ] Capturas e resultados vinculados ao mesmo source revision e assets servidos.
- [ ] Cenarios de assinatura, permissão, conflito, parcials, review e cancelamento cobertos.
- [ ] Estudo humano realizado e relatado **ou** pendência de aceitação explicitamente registrada.
- [ ] Nenhuma publicação, merge, mudança Cloud ou refatoração de produto não aprovada foi feita por este plano.

---

## 21. Exemplos de jornadas completas

### Cenário A — Primeiro uso: Outcome até uma Task

1. Usuário cria Outcome `Publicar integração de pagamentos`, informando nome e desired state.
2. Abre Board e vê post-it central.
3. Clica Outcome e escolhe `Abrir Kanban de objetivos`.
4. No modal, cria Objective `Implementar API` (obrigatório para Outcome).
5. Seleciona Objective e abre seu Kanban no **mesmo modal**.
6. Adiciona Task `Criar endpoint de pagamento`, que nasce em backlog.
7. Fecha modal: mapa mantém zoom/câmera e agora contém os mesmos nós/conexões com estado atualizado.
8. Quando pronto, define critérios onde necessários, ativa Outcome e prepara a Task.

**Não acontece:** criação de Kanban persistido, claim automático, Criterion fictício ou atualização de lifecycle pelo simples ato de abrir o modal.

### Cenário B — Task com material, critérios e revisão independente

1. Agente A assume Task por Profile/contrato real.
2. Registra progresso, Artifact `PR #203` e Evidence `CI run 4401` ligada ao Criterion `Testes passam`.
3. Abre a lateral: mostra progresso, dois materiais distintos, Criterion ainda **não avaliado**.
4. Agente A submete entrega assinada; ReviewCase fica pendente.
5. O Kanban mostra Task em `Verificação` com indicação `Aguardando revisor`, não `done`.
6. Revisor B autenticado inspeciona materiais/critério e registra Assessment adequado ao protocolo.
7. Havendo aceite e gates satisfeitos, conclusão muda Task para `done`.
8. Tasks dependentes são reavaliadas; nenhuma execução automática é fabricada.

### Cenário C — Critério reprovado sem criar problema artificial

1. O revisor encontra falha na expiração do JWT.
2. Marca Criterion `Não atendido`, fornece justificativa/finding e relaciona Evidence adversa.
3. Task aparece com badge `Correção solicitada`, sem conclusão.
4. O executor volta pela autoridade correta, corrige, gera novo Artifact/submissão.
5. Avaliações referentes à submissão anterior não aceitam automaticamente a nova.
6. Nova avaliação é registrada; somente então Task pode ser aceita.

**Não acontece:** criação obrigatória de Issue, Blocker e Objective para um ajuste pontual.

### Cenário D — Outcome com problema mais amplo

1. Todas as Tasks planejadas estão aceitas, mas Criterion do Outcome `Retentativas não duplicam cobranças` falha.
2. O Outcome permanece `active`; inspector mostra `Critério final não atendido`.
3. Usuário cria Objective corretivo `Revisar idempotência`, marcado obrigatório para Outcome, com Tasks específicas.
4. Um Issue documenta a causa quando justificado. Blocker só é criado se houver impedimento operacional de fato e alvo claro.
5. Correção gera Artifact/Evidence novos, Objective é avaliado e aceito, Outcome é reavaliado e concluído.

### Cenário E — Política estrita e cancelamentos

1. Objective tem quatro Tasks: duas aceitas, uma aberta e uma cancelada com motivo.
2. `Verificar objetivo` mostra explicitamente: `2 aceitas`, `1 aberta que impede conclusão`, `1 cancelada que exige reconhecimento de escopo`.
3. O usuário abre a Task pendente e a conclui legitimamente.
4. O autor elegível analisa o cancelamento na prévia e registra reconhecimento explícito, sem contabilizá-lo como trabalho aceito.
5. Criteria próprios são avaliados e o Objective é concluído.

**Não acontece:** fechamento silencioso com Task aberta, nem conversão de cancelado em `done`.

### Cenário F — Concorrência de humano e agentes

1. Humano mantém inspector e modal do Kanban abertos; dois agentes trabalham em Tasks diferentes.
2. Um agente atualiza Criterion/Task e incrementa Outcome revision.
3. Humano tenta confirmar uma conclusão a partir de preview antigo.
4. Backend rejeita preview stale; interface mantém motivo digitado, atualiza dados e pede confirmar novo impacto.
5. Se a requisição enviada tiver resultado incerto, o browser usa receipt/recovery com intenção original.

**Não acontece:** mistura de páginas/versões, gravação cega por drag, duplicação de Assessment ou assignee usado como executor autorizado.

---

## 22. Guia para o agente implementador

**Ordem de trabalho obrigatória:**

1. Ler este plano e a documentação de arquitetura/contratos da branch efetiva; confirmar HEAD e comportamento existente.
2. Criar testes de caracterização/fixtures **antes** de alterar os lifecycles e de reorganizar os componentes.
3. Preservar 100% dos requisitos de autoridade e a identidade visual existente. Não tentar “simplificar” removendo validação no servidor.
4. Implementar fluxo funcional **verticamente**: primeiro um Outcome simples abre/fecha Kanban e retorna ao mapa; depois um Objective com Tasks; depois inspector/critério/material; por fim policy estrita.
5. Para cada etapa, executar testes unitários, de integração, regressão browser, responsive e aderência a SQL real.
6. Se uma capability existente não permitir uma UX proposta, **identificar a lacuna**, criar operação de Application coerente **apenas quando justificada**, ou manter a opção inacessível até existir workflow válido. Nunca inventar sucesso.
7. Evitar refatorações fora do escopo, alterações na marca, substituição do motor de mapa e duplicação do domínio.
8. Atualizar documentação e registrar decisões de modelagem (especialmente policy de conclusão) com referência precisa ao código/ADR aplicável.
9. Manter resultado como PR revisável, com evidências de tests/capturas e lista de pendências. Não efetuar merge/release sem aprovação.

### 22.1. Fontes canônicas para consulta durante a execução

- [PR #116 — workspace operacional](https://github.com/A1b3rt0M3rcad0/wos/pull/116).
- [Guia operacional do PR](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/docs/ux/operational-workspace/README.md).
- [Contratos HTTP/MCP/SDK/CLI](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/docs/operational-workspace-api.md).
- [Relacionamentos do Board](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/docs/workspace-relationships.md).
- [Arquitetura](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/docs/architecture.md).
- [Estudo de compreensão ainda pendente na base](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/docs/ux/operational-workspace/human-study.md).
- [Registro de atividades e obrigações](https://github.com/A1b3rt0M3rcad0/wos/blob/fix/workspace-operational-density/docs/work-activity.md).

### 22.2. Escopo de aprovação por bloco

| Bloco | Decisão proposta | Razão |
|---|---|---|
| Board existente preservado | **Aprovar** | Mapa estrutural já resolve visualização de hierarquia e dependências. |
| Inspector simplificado com duas seções | **Aprovar** | Reduz informação inicial sem apagar operações do domínio. |
| Kanban em modal para Outcome e Objective | **Aprovar** | Dá operação contextual de trabalho sem trocar de tela. |
| Task sem Kanban, com fluxo execução/entrega | **Aprovar** | Evita aninhamento inútil e prioriza trabalho concreto. |
| Checklist verdadeiro de Critérios | **Aprovar** | Mostra aceitação de forma legível e preserva Assessment. |
| Artifacts e Evidence separados em Materiais | **Aprovar** | Resolve ambiguidade entre produzido e comprovado. |
| Issue e Blocker distintos | **Aprovar** | Preserva semântica e evita bloqueios artificiais. |
| `strict_work` como policy nova e default humano | **Aprovação explícita de produto/domínio** | Altera condições de conclusão para novos Outcomes; compatibilidade precisa ser garantida. |
| Criar novo `Block` agregado | **Não incluir no primeiro escopo** | Agrupamento contextual atende à necessidade sem multiplicar ontologia. |
| Criar fila formal para revisão de pais | **Incremento posterior opcional** | Evita fingir status ou duplicar ReviewCase existente. |
| Upload real de arquivos | **Fora deste escopo** | Exige infraestrutura própria e não é apenas UI. |
| Revisão geral de estética/Cloud/runtime | **Fora deste escopo** | Preservar foco e arquitetura. |

---

## 23. Síntese final do contrato de experiência

```text
BOARD ATUAL PRESERVADO
  ↓ selecionar post-it
INSPECTOR OPERACIONAL COMPACTO
  ├── Quem é / qual estado
  ├── O que falta / quem responde
  ├── Uma ação primária contextual
  ├── Duas ou três ações rápidas
  ├── ▸ Detalhes e gerenciamento
  │      ├── Critérios → Verificar → Assessments reais
  │      ├── Materiais → Artifacts + Evidence + relações
  │      ├── Dependências / Bloqueios / Issues
  │      └── Participantes / informações avançadas
  └── ▸ Histórico de atividade → timeline real paginada

OUTCOME ─── [ABRIR KANBAN DE OBJETIVOS] ───┐
                                           │ modal contextual ÚNICO
OBJECTIVE ── [ABRIR KANBAN DE TASKS] ──────┘
                  ↓
TASK ── Preparar → Executar → Comprovar → Entregar → Verificar → Aceitar
                                 ↑                   │
                                 └───── Corrigir ────┘

OBJECTIVE ── Critérios próprios + trabalho obrigatório → Conclusão explícita
OUTCOME ──── Critérios finais + objetivos obrigatórios → Conclusão explícita
```

**Resultado de produto:** o usuário humano opera o WOS como um Board familiar, entra em Kanbans contextuais sem sair do Canvas e toma decisões simples e fundamentadas. O sistema continua gerenciando contratos, versões, evidências, critérios, dependências e aceitação, mas só mostra essa complexidade quando ela é relevante para uma decisão ou ação específica.

**Estado do plano:** documento concluído para revisão/aprovação; implementar em branch/PR controlado após alinhamento sobre a policy de conclusão estrita. Nenhuma alteração de código foi efetuada por este documento.
