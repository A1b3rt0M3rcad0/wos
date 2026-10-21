> **Execution authorization (2026-10-21):** The repository owner explicitly authorized implementing this plan in Draft PR #116 and permitted subagent use. This current instruction supersedes the historical request statement in §1.2 that only document delivery was requested. The architectural boundaries, staged workflow, and no-merge/no-release restrictions remain in force.

---

# WOS — Plano completo de implementação operacional

> **Documento de execução para desenvolvedores e agentes.**
> **Status:** planejamento; nenhuma implementação ou aprovação de testes é declarada por este documento.
> **Baseline:** `A1b3rt0M3rcad0/wos`, PR **#116**, branch **`fix/workspace-operational-density`**, SHA **`2aab252129f548ab54ad4232095e1bc83d76d372`**.
> **Referência temporal:** solicitação de 20/10/2026, fuso America/Sao_Paulo. O timestamp UTC do último update consultado do PR é `2026-10-21T01:20:08Z` — 20/10/2026 às 22:20:08 em São Paulo.
> **Escopo:** lifecycle de Objectives/Tasks, verificação e conclusão, credenciais e permissões, reservas, criação contextual, dependências, Issues, Kanban e paridade HTTP/MCP/CLI/SDK.
> **Execução proposta:** continuar na mesma branch e no mesmo PR; manter **Draft do início ao fim**, inclusive depois da qualificação. Sem merge, tag, release, publicação ou alteração de versão sem autorização posterior.
> **Organização:** 84 etapas executáveis, cada uma com objetivo, dependências, arquivos, implementação, testes, aceite e commit sugerido. Testes locais de cada alteração acompanham o respectivo commit; as etapas finais ampliam a qualificação, não a substituem.

## Sumário

1. [Contrato de execução e delimitação](#1-contrato-de-execucao-e-delimitacao)
2. [Baseline, evidências e reconciliação](#2-baseline-evidencias-e-reconciliacao)
3. [Decisões normativas de produto](#3-decisoes-normativas-de-produto)
4. [Máquinas de estado e regras operacionais](#4-maquinas-de-estado-e-regras-operacionais)
5. [Entrega, critérios e revisão](#5-entrega-criterios-e-revisao)
6. [Elegibilidade e conclusão automática](#6-elegibilidade-e-conclusao-automatica)
7. [Identidades, API Keys e autorização](#7-identidades-api-keys-e-autorizacao)
8. [Reserva e concorrência de execução](#8-reserva-e-concorrencia-de-execucao)
9. [Issues, Blockers e dependências](#9-issues-blockers-e-dependencias)
10. [Experiência de criação e Kanban](#10-experiencia-de-criacao-e-kanban)
11. [Arquitetura, arquivos e contratos](#11-arquitetura-arquivos-e-contratos)
12. [Persistência e migração](#12-persistencia-e-migracao)
13. [HTTP, MCP, SDK e CLI](#13-http-mcp-sdk-e-cli)
14. [Etapas de implementação](#14-etapas-de-implementacao)
15. [Matriz de testes e cenários](#15-matriz-de-testes-e-cenarios)
16. [Comandos de verificação](#16-comandos-de-verificacao)
17. [Integração com o plano existente de 44 etapas](#17-integracao-com-o-plano-existente-de-44-etapas)
18. [Entrega, rollout e aceite final](#18-entrega-rollout-e-aceite-final)
19. [Instrução de execução para agentes](#19-instrucao-de-execucao-para-agentes)
20. [Fontes e referências](#20-fontes-e-referencias)

<a id="1-contrato-de-execucao-e-delimitacao"></a>
## 1. Contrato de execução e delimitação

### 1.1 Resultado exigido

Ao terminar, o WOS deverá permitir que humanos e múltiplos agentes planejem, reservem, executem, entreguem, verifiquem e concluam trabalho sem depender de regras exclusivamente no navegador. O domínio continuará distinguindo **resultado**, **trabalho**, **autoridade temporária**, **evidência**, **avaliação**, **conclusão**, **problema** e **impedimento**.

O comportamento-alvo não é um editor genérico de status. Cada ação deve corresponder a um comando autorizado, versionado, idempotente e auditável, validado novamente contra o estado corrente do backend.

### 1.2 O que este documento não autoriza

Não executar este plano apenas por ele existir. O pedido desta conversa é entregar o `.md`. Não alterar o repositório, o PR, bancos de usuários, chaves ou permissões durante a produção do documento.

Na implementação futura, não introduzir runtime de agentes, LLM, scheduler de trabalho, workflow scriptável, microsserviço, Redis de coordenação, segundo banco operacional, novo frontend framework ou dependência da Woobe. A conclusão automática descrita aqui é uma **regra determinística de integridade**, não execução autônoma de trabalho.

Não adicionar sub-Objectives para resolver dependências. `ParentObjectiveID` já existe e precisa continuar consistente; a hierarquia existente será preservada, sem ampliar esse recurso como requisito desta iniciativa.

### 1.3 Fontes de verdade e mudanças explícitas

Ler `AGENTS.md`, `ROADMAP.md`, as seções pertinentes de `docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md`, ADRs 038–042 e o plano `docs/ux/operational-kanban/issue-and-dependency-plan.md` antes de implementar.

O `AGENTS.md` atual proíbe conclusão automática de Objective por conclusão de Task. Esta solicitação altera deliberadamente essa regra, **somente** para Objectives com política automática explicitamente autorizada. Atualizar ADR, contrato canônico e esse trecho de `AGENTS.md` juntos; não deixar a alteração escondida em um callback.

Continuam válidas: Outcome não é aceito automaticamente; Evidence não aprova Criterion; assignee não adquire lease; Issue não cria Blocker implicitamente; resolver Issue não libera Blocker implicitamente; propriedade não equivale a `depends_on`.

Este documento em português é um artefato de solicitação do proprietário. Código, identificadores, copy mantida na aplicação, ADRs novos e documentação de produto devem seguir o inglês já exigido pelo repositório. Exemplos de mensagens em inglês neste plano são propostas de copy, não mudanças de idioma desta conversa.

### 1.4 Preservação do produto

Preservar onboarding, autenticação, bootstrap, seleção de Workspace, identidade editorial, tipografia, paleta, robôs, post-its, câmera do Board, experiência de retorno, recuperação de requisição incerta e assinaturas existentes. O ajuste inevitável na criação inicial é tornar explícito o Objective proprietário antes de criar uma Task; não criar Objective oculto ou placeholder.

Para alterações de UX, ler as skills presentes no próprio repositório:

- `.agents/skills/laws-of-ux-auditor/SKILL.md`;
- `.agents/skills/product-design-and-ux/SKILL.md`.

Não atribuir ganhos quantitativos de usabilidade sem medição. Inspeção de código não equivale a teste de teclado, navegador, leitor de tela ou compreensão com usuários.

### 1.5 Disciplina de execução

Cada etapa tem um commit semântico. Quando uma etapa já estiver implementada de forma equivalente na HEAD real, verificar e registrar evidência; não duplicar módulos nem produzir commit vazio. Corrigir divergências pequenas no commit da etapa. Subdivisão só é aceitável quando necessária para manter alterações revisáveis, preservando rastreabilidade pelo ID da etapa.

Não executar `reset --hard`, force-push, rebase destrutivo, merge ou exclusão de arquivos sem necessidade autorizada. Não sobrescrever trabalho de outro agente. Antes de cada bloco, confirmar SHA, branch e estado do worktree. Se a HEAD avançar, reconciliar diferenças, sem descartar o que já foi entregue.

Todos os testes descritos são **requisitos futuros**, não resultados obtidos na preparação deste plano. A análise foi estática via GitHub; não houve execução local da aplicação nem da suíte do repositório.

<a id="2-baseline-evidencias-e-reconciliacao"></a>
## 2. Baseline, evidências e reconciliação

### 2.1 Achados que orientam a implementação

As referências abaixo apontam para a SHA fixa, não para uma branch móvel. O caminho completo funciona como indicação de implementação; os links verificáveis estão na seção 20.

| ID | Fato observado no baseline | Consequência para o plano |
|---|---|---|
| B01 | `domain/readiness.go:WorkItemReadiness` bloqueia Objective terminal, mas não `planned`. | Adicionar gate de Objective ativo e usar o mesmo predicado em todos os mecanismos de aquisição. |
| B02 | `application/service.go:ActivateWorkItem` ativa sem validar lifecycle do Objective. | Corrigir ativação; não limitar a correção ao drag-and-drop. |
| B03 | `CreateWorkItem` já exige `ObjectiveID`; criação aceita `backlog` ou `todo`, e `insertWorkItemCreation` só rejeita pai terminal. | Não reimplementar ownership; fechar bypass de criação diretamente disponível sob Objective planejado. |
| B04 | `domain/workspace_preconditions.go:Objective.CheckAchievement` aceita `planned` e `in_progress`. | No novo contrato operacional, exigir `in_progress` para achievement. |
| B05 | `completion_policy.go` oferece `compatible` e `strict_work`, sem fechamento automático. | Separar política de abrangência do trabalho da política de conclusão automática. |
| B06 | `WorkLease` tem Principal, Actor, claim e fencing, mas não ID da credencial. | Adicionar binding de credencial somente onde falta, preservando o protocolo assinado. |
| B07 | `CredentialPolicy` já restringe permissões, Outcomes, signing keys e quotas. | Evoluir o sistema existente, sem um segundo RBAC paralelo. |
| B08 | `commandPermission` contém fallback por substring e concentra WorkItem/Issue/Blocker em `work:write`. | Substituir inferência por catálogo explícito e permissões granulares. |
| B09 | `kanban-operations.js` deriva Verification de revisão real; arrastar para Verification abre inspeção, não entrega. | Distinguir inspeção de prova e submissão; introduzir a operação faltante onde o protocolo não a oferece. |
| B10 | `actions.js` e caminhos de `app.js` ainda oferecem criação de Task no contexto do Outcome. O footer do Kanban já restringe essa criação ao Objective. | Unificar todos os entrypoints, evitando reaplicar o erro do menu global. |
| B11 | `CreateIssueCommand` exige Objective, enquanto `human-forms.js` ainda não estabelece esse proprietário no formulário de Issue. | Adaptar formulário e transporte; tratar como gap de integração, não inexistência do domínio. |
| B12 | `dependencies.go` já possui preview e `ApplyDependencyChanges`; `dependency_impact.go` já percorre impactos limitados. | Reutilizar domínio e consultas; completar paridade e UX. |
| B13 | Migrações 0035 e 0036 já endurecem ownership de Task e Issue. | Não recriar essas migrações ou presumir que estão livres. |
| B14 | Plano de 44 etapas registra avanço até 26, enquanto trechos do ROADMAP ainda indicam 26 como próxima. | Confrontar código, testes, contratos e documentação antes de marcar conclusão. |
| B15 | `ReviewCase`/`ReviewContract` atuais fazem parte do protocolo assinado e exigem vínculos/digests próprios. | Não usar esse tipo como uma fila humana genérica removendo suas verificações. |
| B16 | `WorkSubmission` requer ContractID real e valida digest conforme protocolo. | Não inventar contrato para uma Task com lease legado apenas para reutilizar DTO. |
| B17 | `pipeline.go` já centraliza transação, autorização, idempotência, eventos e protocolos. | Inserir as novas invariantes nesse fluxo e em primitivas privadas, não em transportes. |
| B18 | CLI possui dispatch específico para workspace assinado, perfis e recuperação local. | Novos comandos não podem cair em fallback legado quando a operação exige assinatura. |

**Qualificação:** presença de código e teste não significa teste executado nesta análise. Documentação desatualizada não prevalece sobre o código observado para afirmar o estado real, mas uma regra nova precisa de ADR para prevalecer sobre comportamento antigo intencional.

### 2.2 Correções de precisão incorporadas ao planejamento

**Todas as Tasks, não apenas “Tasks obrigatórias” inventadas.** O modelo inspecionado não oferece um atributo genérico `Task.Required`. Para a conclusão automática, considerar todas as Tasks estruturalmente pertencentes ao escopo do Objective. `RequiredForOutcome` é atributo de Objective e não deve ser transferido por analogia para Task.

**Verification não existe igualmente em todos os protocolos.** O caminho assinado já dispõe de entrega/revisão; o lease legado permite conclusão direta. Prometer a mesma fila de verificação apenas rearranjando colunas seria incorreto. A seção 5 especifica a extensão mínima para os caminhos não assinados, sem enfraquecer o assinado.

**Definição congelada em Backlog deve ser preservada.** Rejeitar uma entrega não autoriza editar critérios arbitrariamente durante a execução. Correção de implementação e revisão de escopo são operações distintas.

**Não confundir política de revisão com conclusão automática.** Aceitar uma Task, avaliar Criterion de Objective e aplicar achievement automático são decisões separadas, ainda que algumas sejam executadas na mesma transação após autorização prévia.

### 2.3 Inventário de implementação reutilizável

Reutilizar `Outcome`, `Objective`, `WorkItem`, `SuccessCriterion`, `CriterionAssessment`, `Conclusion`, `WorkLease`, `WorkContract`, `WorkSubmission`, `ReviewCase`, `ReviewContract`, `Issue`, `Blocker`, `Relation`, `CredentialPolicy`, grants, read models, intents, outbox e coordenação por Outcome.

Não reescrever `workspace-kanban.js`, `human-forms.js`, `workspace-view.js`, `reference-picker.js` ou o pipeline inteiro. Extrair componentes localizados somente onde necessário para separar responsabilidades e manter código testável.

<a id="3-decisoes-normativas-de-produto"></a>
## 3. Decisões normativas de produto

### 3.1 Perfil operacional versionado

Introduzir uma configuração explícita de **perfil operacional** no Outcome:

- `compatibility_v1`: representa os dados históricos e comportamento que ainda não foram migrados;
- `operational_v2`: aplica as regras deste documento.

`operational_v2` é o default de **novos Outcomes criados após o rollout**, por qualquer transporte. A omissão do cliente não deve selecionar silenciosamente o perfil legado. Outcomes existentes recebem `compatibility_v1` durante o expand para preservar históricos; uma migração explícita com preview os leva ao novo perfil.

Essa compatibilidade é uma ponte de migração, não um bypass escolhido livremente pelo executor. Após a instalação-alvo migrar seus Outcomes abertos, impedir criação de novos Outcomes legados por clientes comuns. Historicamente concluídos podem continuar legados e somente consultáveis até uma reabertura controlada.

A opção de perfil não altera o protocolo de autoridade do Namespace (`legacy`, contracts, signed-v2). São eixos diferentes. O mesmo perfil operacional deve produzir o mesmo conjunto de obrigações de negócio, usando a autoridade apropriada a cada protocolo.

### 3.2 Matriz de decisões adotadas

| Tema | Decisão deste plano |
|---|---|
| Ownership | Task e Issue têm exatamente um Objective proprietário no mesmo Outcome. |
| Planejamento | Criar/editar definição de Task somente em Backlog; Objective Planned permite planejamento, não execução. |
| Ativação | Backlog → To Do exige Objective e ancestrais aplicáveis In Progress e Outcome ativo. |
| Aquisição | Exige novamente prontidão, permissões, autoridade exclusiva, scope e tempo do servidor. |
| Verificação | No perfil v2, todo aceite de Task requer uma entrega e uma decisão de revisão persistidas. |
| Independência | A revisão exige Principal diferente do conjunto de executores quando a política independente vigente assim determinar. Chaves diferentes do mesmo Principal não estabelecem independência. |
| Critérios | Task pode ser criada sem critérios, mas sua ativação v2 exige pelo menos um critério obrigatório ativo. |
| Conclusion automática | Default de novos Objectives v2: `automatic`; visível e configurável na criação por quem tem permissão para delegá-la. |
| Conclusão manual | `manual` permanece disponível, especialmente para migração ou decisões que exigem aprovação humana final. Usa os mesmos gates. |
| Empty Objective | Sem Tasks/filhos de trabalho aplicáveis, não ocorre conclusão automática; critérios próprios ainda podem permitir conclusão manual. |
| Cancelamento | Cancelled não conta como Accepted; uma exclusão por cancelamento exige decisão manual explícita, não autofechamento. |
| Outcome | Continua dependendo de aceitação explícita, inclusive quando todos os Objectives estiverem Achieved. |
| Issue | Pendência própria impede achievement do Objective; não cria Blocker e não impede toda Task indiscriminadamente. |
| Reserva | Controle exclusivo por credencial canônica, claim/execution e fencing; atribuição permanece independente. |
| UI | Arrastar e soltar é a única interação para mudar a etapa de um card; deve ser acessível por teclado e toque usando o mesmo fluxo. Não recolocar select, menu ou ação separada de mudança de status nos cards. |
| Autoridade assinada | Browser não adquire autoridade assinada nem assina resultados; oferece handoff explícito para host/CLI autorizado. |

### 3.3 Limites da automação

Não gerar critérios, Evidence ou Assessments para “fazer a porcentagem chegar a 100%”. Não tomar uma checkbox de criação como aprovação de um resultado futuro. A seleção de `automatic` delega uma regra fechada de conclusão, não julgamento semântico.

Quando uma organização exige aprovação humana explícita de Objective, configurar `manual`; quando exige apenas avaliações explícitas de seus critérios, `automatic` pode concluir após essas avaliações legítimas. Nenhum automatismo dispensará uma política de separação de funções.

<a id="4-maquinas-de-estado-e-regras-operacionais"></a>
## 4. Máquinas de estado e regras operacionais

### 4.1 Objective

Persistir os estados já existentes: `planned`, `in_progress`, `achieved`, `cancelled`. Não persistir `ready_to_achieve`, `blocked` ou uma fila fictícia de Verification para Objective; são projeções.

| Origem → destino | Comando/operação | Requisitos v2 |
|---|---|---|
| inexistente → planned | `CreateObjective` | Outcome existente aberto, parent válido opcional, autorização de criação. |
| planned → in_progress | `StartObjective` | Outcome ativo; ancestrais In Progress; critério obrigatório ativo; dependências hard satisfeitas; sem Blocker aplicável. |
| in_progress → achieved | manual ou conclusão determinística | Gates da seção 6; política e autoridade apropriadas; Conclusion real. |
| planned/in_progress → cancelled | `CancelObjective` | Justificativa; nenhuma execução/revisão ativa que fique abandonada; impacto explícito. |
| achieved/cancelled → planned | `ReopenObjective` | Permissão específica, motivo e revisão do impacto; preservar ConclusionHistory. |
| planned → achieved | nenhum no perfil v2 | Negar. Começar não deve ser executado implicitamente pelo achievement. |

Manter o retorno atual de `ReopenObjective` para `planned`, em vez de alterar silenciosamente para `in_progress`. Reabrir e iniciar são duas intenções distintas, mostradas claramente ao operador.

### 4.2 Task e apresentação

Persistir os estados de WorkItem existentes: `backlog`, `todo`, `in_progress`, `done`, `cancelled`. `Accepted` representa `done` acompanhado de conclusão válida; `Verification` representa entrega pendente de decisão real. Correção aguardando nova aquisição é `todo` com indicador `changes_requested`, não “em execução” sem executor.

| Origem → apresentação de destino | Operação | Requisitos/resultados |
|---|---|---|
| criação → Backlog | `CreateWorkItem` | Objective obrigatório, título, escopo validado. Sem lease automático. |
| Backlog → To Do | `ActivateWorkItem` | Perfil ativo, Objective/ancestrais In Progress, critérios e definição preparados. |
| criação → To Do | criação atômica explicitamente solicitada | Aplicar exatamente os gates da ativação sobre o estado final criado; caso inválido, rollback integral. |
| To Do → In Progress | claim/acquire conforme protocolo | Prontidão atual, ausência de lease/contract válido concorrente, credencial autorizada e fencing novo. |
| In Progress → Verification | entrega conforme protocolo | Titular válido; material/prova referenciada; snapshot de critérios; cessar autoridade de execução entregue. |
| Verification → Accepted | decisão de aprovação | Reviewer autorizado; independência quando exigida; assessments atuais; material e revisão exatos; sem impedimentos. |
| Verification → To Do | solicitar correções | Motivo e itens de correção; encerra a rodada; não concede lease ao executor automaticamente. |
| To Do → Backlog | `DeferWorkItem` | Nenhuma autoridade/revisão aberta; permissão de planejamento; preserva histórico. |
| In Progress → To Do | release/revoke conforme protocolo | Titular ou intervenção autorizada; sem alterar definição e sem apagar entregas anteriores. |
| terminal → To Do | `ReopenWorkItem` | Objective já reaberto **e iniciado**; motivo; Outcome aberto; inicia nova geração de trabalho. |
| não terminal → Cancelled | cancelamento explícito | Gates existentes de autoridade e intervenção; não tratar como aceitação. |

### 4.3 Predicados compartilhados

Criar nomes conceituais, marcados como novos, para consolidar lógica:

- `EvaluateObjectiveExecutionGate`: Outcome, lifecycle do Objective, ancestrais e bloqueios aplicáveis;
- `EvaluateTaskActivation`: prontidão de definição e gate de execução, sem exigir que a Task já esteja To Do;
- `EvaluateTaskAcquisition`: activation gate + dependências/schedule + autoridade;
- `EvaluateTaskAcceptance`: entrega, revisão, assessments, política e impedimentos;
- `EvaluateObjectiveCompletion`: relatório completo de obrigações, independente da autorização do solicitante;
- `AuthorizeObjectiveConclusion`: decisão de autoridade manual ou delegação automática.

Os nomes propostos não implicam que essas funções existam hoje. Implementá-los em arquivos locais do domínio/Application indicados na seção 11, reaproveitando os helpers atuais. Evitar a prática de mudar artificialmente o lifecycle de uma cópia para passar em um validador inadequado quando for possível separar predicados de estado e condições.

### 4.4 Abrangência dos gates

Aplicar à criação com `todo`, ativação, claim, reclaim, aquisição por ID, aquisição de próximo trabalho, renew/resume quando a autoridade de negócio deixou de valer, submissão, conclusão, conclusão administrativa e reabertura. Queries de ready-work, candidatos, capabilities e Kanban precisam mostrar os mesmos motivos.

Liberar/revogar autoridade, registrar incidente, consultar histórico ou acrescentar documentação permitida não deve ficar impossível por o parent estar bloqueado. **Gate de avançar trabalho não é gate de toda mutação.** Separar comandos de avanço, recuperação e observação.

### 4.5 Ancestrais e dependências

A exigência de ancestral em execução é regra de lifecycle, não uma relação `depends_on`. Não persistir relações ocultas por causa dela.

Essa regra pode criar esperas impossíveis combinada a dependências. Proibir Task com dependência hard de seu próprio Objective ou ancestral que precisa concluir o trabalho dessa Task; também proibir Objective dependente hard de descendente cujo início exige o próprio Objective. Validar essas incompatibilidades no grafo completo proposto, inclusive em batch e reparenting. Um DAG de `depends_on` sozinho não detecta todas as esperas causadas por hierarquia + conclusão.

### 4.6 Fechamento e reabertura

Não permitir criar ou mover Task para Objective terminal. Para reabrir Task sob Objective Achieved, primeiro reabrir o Outcome se necessário, reabrir Objective/ancestrais relevantes, iniciá-los e então reabrir a Task. Disponibilizar guia contextual que preserve a intenção, sem executar comandos ocultos.

Alterações em critérios, retração de prova ou reabertura de pré-requisitos não apagam conclusões históricas. Usar os mecanismos de contestação/histórico existentes para representar que uma conclusão pode exigir revisão; impedir que uma projeção apresente a prova antiga como atual. Mudanças que tornariam inválido um fechamento operacional atual devem exigir preview e reabertura explícita dos agregados afetados, ou ser negadas com motivos acionáveis.

Não cancelar Objective com contrato, lease ou revisão ativa e deixar o filho órfão de autoridade. Oferecer plano de drenagem/intervenção com as operações existentes; executar cada ato autorizado e auditado, sem prometer que revogar no WOS interrompe efeitos externos do agente.

<a id="5-entrega-criterios-e-revisao"></a>
## 5. Entrega, critérios e revisão

### 5.1 Critérios são definições; assessments são decisões

Cada Criterion deve manter ID, owner, revisão, obrigatoriedade e modo de verificação. Cada assessment precisa apontar a revisão exata e a entrega/ciclo aplicável. Alterar descrição ou revisão de Criterion não transfere o aceite anterior.

No perfil v2, ativar Task exige ao menos um Criterion obrigatório ativo. O formulário pode salvar Backlog incompleto. Não exigir Evidence na criação: Evidence é produzida ou registrada durante execução e revisão.

O comando de assessment deve continuar exigindo evidências conforme `VerificationMode`. Attestation só é suficiente quando o modo do critério permite e o ator possui permissão. `waived` não é sinônimo de `met`: o autofechamento exige `met`; dispensas seguem conclusão manual com autorização e justificativa específicas.

Não permitir que executor com permissão de entregar ganhe automaticamente permissão de avaliar critérios. Criação atômica de Task + critérios exige os direitos correspondentes, sem depender genericamente de `outcome:write` quando o owner real é Task.

### 5.2 Caminho assinado: reutilizar, não relaxar

Reutilizar `ReturnSignedWork`, `AcquireSignedReviewContract`, `ReturnSignedReview`, `ReviewCase`, `ReviewContract`, decisões e material assinados. Preservar digest, envelope, specification, política mínima congelada, credencial original, execução, fencing e independência.

Para Tasks v2 que exigem Verification, a aquisição assinada deve congelar piso `independent_review`, já suportado. A rota de retorno direto do protocolo assinado não satisfaz essa política e deve ser recusada para esse trabalho, mesmo que a credencial tenha capacidade de conclusão direta em outros contextos. Em contexto legado, não mudar retroativamente o piso de um contrato em voo: drenar ou migrar explicitamente.

O browser mostra requisitos, cria documentação permitida, abre material e prepara handoff. Não falsifica DSSE, não guarda chave privada e não usa sessão de browser para chamar autoridade assinada. Um humano pode revisar pelo CLI/host autorizado; a interface deve explicar a restrição do protocolo e o comando correspondente, não simplesmente esconder o botão sem motivo.

### 5.3 Caminhos não assinados: extensão necessária e delimitada

O baseline não oferece o mesmo ciclo de revisão para lease legado. Introduzir **registro de entrega e decisão não assinada**, não outra Task e não outro lease. Essa é uma extensão real do escopo de domínio, documentada em ADR.

Nomes propostos:

- `TaskDelivery`, em `packages/wos-core/domain/task_delivery.go` **[novo]**;
- `TaskDeliveryDecision`, no mesmo módulo ou `task_delivery_decision.go` **[novo]**;
- `TaskDeliveryRepository`, em `packages/wos-core/ports/task_delivery.go` **[novo]**;
- `SubmitTaskForReview`, `AcceptTaskDelivery`, `RequestTaskDeliveryChanges`, em `packages/wos-core/application/task_delivery.go` **[novo]**.

**TaskDelivery não substitui WorkSubmission.** Para `contracts_v1`, registra uma referência à `WorkSubmission` verdadeira criada pelo protocolo existente; os bytes do material não são duplicados. Para lease legado, guarda material de entrega próprio com referências a Artifacts/Evidence, sem inventar ContractID. Um discriminador armazenado `authority_kind=legacy_lease|work_contract_v1` determina qual vínculo é válido. A escolha é derivada no servidor, não aceita cegamente do payload.

Para `signed_v2`, não criar TaskDelivery duplicada. O read model comum de revisão lê a fonte assinada existente. Comandos não assinados rejeitam Tasks, contratos e casos cuja autoridade é assinada.

### 5.4 Contrato de TaskDelivery

Campos propostos:

| Campo | Regra |
|---|---|
| `id`, `scope`, `work_item_id` | Identificadores internos, mesmos Namespace/Outcome da Task. |
| `version` | Concorrência de decisão; string decimal no wire novo. |
| `authority_kind` | Derivado do estado da Task. |
| `claim_id` / `contract_id` | Exatamente o vínculo exigido pelo tipo; nunca ambos como autoridades simultâneas. |
| `fencing_token`, `execution_id?` | Snapshot da geração de execução encerrada por esta entrega. |
| `credential_id`, `principal_id`, `actor_ref` | Proveniência resolvida da autenticação e da autoridade. |
| `work_item_version`, `definition_digest` | Definição observada e congelada ao entregar. |
| `criteria_snapshot` | IDs e revisões dos critérios aplicáveis, sem copiar assessments falsamente. |
| `work_submission_id?` | Obrigatório para contracts_v1; proibido quando não existe WorkSubmission real. |
| `material?` | Somente para lease legado; summary e referências versionadas/imutáveis. |
| `status` | `pending`, `approved`, `changes_requested`, `withdrawn` ou `superseded`. |
| `supersedes_delivery_id?`, `round` | Correções explicitamente vinculadas, sem sobrescrever a rodada anterior. |
| `submitted_at`, `decided_at?`, `decision_id?` | Tempo do servidor e decisão persistida. |

Adicionar `PendingDeliveryID` ao WorkItem somente para o caminho não assinado. É mutuamente exclusivo com `PendingReviewCaseID` assinado. A regra atual “Task legada in_progress sempre possui lease” passa a permitir a variante **in_progress com entrega pendente e sem lease**; a integridade do vínculo é validada em Application e no armazenamento. Não liberar genericamente `in_progress` sem autoridade ou revisão identificável.

### 5.5 Entregar para verificação

`SubmitTaskForReview` recebe expected_version, autoridade atual exata, summary/material e idempotency key. Dentro de uma UoW: autenticar; autorizar entrega; coordenar Outcome; verificar lease/contrato vivo e credencial; validar gate de execução; registrar material imutável; congelar definição; criar TaskDelivery; encerrar a autoridade de execução; vincular PendingDeliveryID; incrementar versões; emitir eventos; retornar recibo.

No caminho contracts_v1, a operação compõe primitivas internas de criação de WorkSubmission e fechamento por entrega. Não chamar publicamente `SubmitWorkResult` e depois outro comando, deixando uma janela intermediária. Não modificar `WorkContract.Deliver` assinado para aceitar silenciosamente envelopes ausentes; criar primitiva autenticada separada com validação explícita do protocolo.

Falha em qualquer parte não deixa entrega pendente sem material, lease livre sem registro ou material aceito sem Task vinculada. Uma resposta perdida é recuperada com a mesma intenção, não com nova aquisição.

### 5.6 Aprovar ou solicitar correções

`AcceptTaskDelivery` exige versões da Task e entrega, delivery_id exato, assessments aplicáveis àquela rodada, permissões de revisão e aceite, política de independência, dependências e Blockers. Não exige o lease antigo do executor, pois esse lease foi encerrado na entrega. A autoridade vem da revisão autorizada, não da reutilização do lease expirado.

Gravar `TaskDeliveryDecision` imutável, atualizar a entrega, produzir Conclusion da Task, persistir `done` e limpar PendingDeliveryID na mesma transação. Dois revisores concorrentes sobre a mesma versão: somente uma decisão é aceita; repetição idempotente retorna a decisão original; decisão diferente falha.

`RequestTaskDeliveryChanges` exige motivo e findings acionáveis, encerra a rodada como `changes_requested` e retorna a Task para `todo`, sem lease. O executor deve adquirir novamente a Task. O novo material referencia a rodada anterior; assessments da entrega rejeitada não aprovam a nova entrega automaticamente.

Uma correção de implementação preserva a definição. Para alterar definição/critério, usar operação explícita de replanning: encerrar revisão pendente por retirada autorizada, resolver autoridade ativa, deferir Task elegível para Backlog, alterar definição, ativar e adquirir novamente. Não encadear isso sem confirmações.

### 5.7 Projeção comum e políticas

Criar `TaskVerificationView` **[novo read model]** com source_kind, delivery/case ID, rodada, material, status, executor, revisor, requisitos, motivo do bloqueio e ações possíveis. O cliente não precisa inferir o protocolo olhando campos nulos.

A política mínima pode exigir independência; quando não exige, um humano autorizado pode avaliar sua própria entrega em um **ato explícito separado**. Chaves e perfis de executor/revisor continuam separados por default administrativo. Em signed-v2, usar a revisão independente já existente, que pode ser mais restritiva que o mínimo do trabalho.

No perfil v2, `CompleteWorkItem`, `AdministrativeCompleteWorkItem`, `FinalizeWorkContract` e retorno assinado direto não podem contornar a existência da revisão exigida. Admin não significa dispensa automática de prova. Preservar comportamento histórico somente dentro da compatibilidade explicitamente delimitada.

<a id="6-elegibilidade-e-conclusao-automatica"></a>
## 6. Elegibilidade e conclusão automática

### 6.1 Configuração

Adicionar `ObjectiveCompletionMode` com `manual` e `automatic`; não reutilizar o campo Outcome.CompletionPolicy para outra semântica. `strict_work` continua a determinar obrigações de trabalho; `completion_mode` determina quem inicia a conclusão quando elas estiverem satisfeitas.

Novos Objectives v2 usam `automatic` como default **visível** quando o criador possui `objective:configure_completion`; sem essa permissão, o cliente deve selecionar `manual` explicitamente ou solicitar ao administrador a configuração, nunca receber uma delegação implícita. APIs respondem com erro explicativo a pedido de delegação não autorizado. A UI não muda a opção silenciosamente.

Outcomes legados migram com `manual`, a menos que uma operação de migração autorizada selecione automática após preview. Exigir `strict_work` para o modo automático. Alterar CompletionMode ou perfil usa expected_version, reason e permissão específica; congelar política mínima quando existir execução/revisão que não possa ser reinterpretada.

### 6.2 Relatório de elegibilidade

`ObjectiveCompletionReport` **[novo]** deve retornar, em uma leitura coerente:

- versão do Objective, OutcomeRevision e evaluated_at;
- política e ciclo atuais;
- quantidade total de Tasks aplicáveis e contagem por condição;
- Objectives descendentes ainda não concluídos;
- critérios exigidos, assessments aplicáveis e pendências;
- Issues próprias pendentes, incluindo cadeia de duplicatas;
- dependências hard não satisfeitas e Blockers aplicáveis;
- autoridades ou revisões ainda abertas;
- cancelamentos/dispensas que exigem revisão manual;
- `eligible`, `automatic_allowed`, `manual_allowed` e reason_codes separados.

Os totais que sustentam o gate são completos; samples de referências podem ser limitados. Se a leitura não consegue provar completude, não concluir. “20 de 20 cards carregados” não significa “todas as Tasks do Objective”.

### 6.3 Predicado do autofechamento

```text
can_auto_achieve(objective) =
    objective.lifecycle == in_progress
    AND outcome.lifecycle == active AND not archived
    AND objective.completion_mode == automatic
    AND delegated_completion_policy_is_current_and_valid
    AND applicable_work_set_is_nonempty
    AND every applicable Task is done with a valid acceptance conclusion
    AND every descendant Objective required by strict_work is achieved
    AND every required current Objective criterion has applicable assessment met
    AND no excluded cancelled work and no waiver requiring manual decision
    AND no pending Issue owned by this Objective
    AND every hard dependency is satisfied
    AND no applicable active Blocker
    AND no open execution authority or pending review in the applicable work
    AND all proof and closure policies are satisfied
```

O conjunto de trabalho inclui as Tasks do Objective e de sua subárvore existente, como já considera `strictCompletionWork`. Não atravessar relações `relates_to` nem `depends_on` para definir ownership. Objectives opcionais para o Outcome não se tornam obrigatórios apenas por existir essa política; dentro da subárvore, preservar a abrangência definida por strict_work.

Parent sem Task direta, mas com trabalho real em descendentes, não é vazio. Objective sem Tasks nem trabalho descendente não se fecha automaticamente. Para uma Task cancelada, o caminho automático permanece bloqueado; a conclusão manual pode usar o acknowledgement de exclusão existente, deixando explícito que não houve aceite daquele trabalho.

### 6.4 Conclusão não é uma avaliação inventada

A Conclusion automática referencia **assessments reais dos critérios do Objective**. O sistema não transforma “todas as Tasks aceitas” em assessment genérico. Se o critério do Objective exige validação de integração, alguém/ferramenta autorizada deve avaliar essa integração; até lá, o estado permanece In Progress.

Usar ActorKind `automation`, já existente, com provider `wos` e ID estável de `objective_completion`. Não introduzir `actor.kind=system`, inexistente no enum atual, apenas para um exemplo. Registrar Principal de serviço interno com autoridade limitada ao algoritmo, sem API Key exposta ao usuário.

A delegação automática deve registrar authorizing_principal_id, authorizing_credential_id, policy_revision, scope e data. O Principal interno não recebe permissões gerais de avaliar, criar provas, emitir chaves ou aceitar Outcomes. Não simular uma nova identidade independente para escapar de uma política que exige decisão humana final: nesse caso o Objective é manual.

### 6.5 Integração transacional

Adicionar um hook privado de derivação de conclusões à camada Application, somente para comandos capazes de alterar elegibilidade. A execução deve ocorrer após a mutação principal e antes da finalização de eventos/recibo, na mesma UoW e sob o mesmo Outcome guard.

Não chamar o handler público `AchieveObjective` recursivamente dentro de `CompleteWorkItem` ou de outra transação. Extrair a primitiva de achievement que recebe UoW, estado, decisão de autoridade já verificada e tempo; manual e automático a reutilizam.

Eventos principais e derivados compartilham CommandID/correlation e a revisão de Outcome definida para a operação composta. Avançar OutcomeRevision **uma única vez por comando composto**. Cada Objective modificado avança sua própria versão. Gerar IDs determinísticos por intenção ou alocados uma única vez dentro da operação idempotente, sem eventos duplicados no replay.

O resultado do comando deve expor, em metadado aditivo, as entidades automaticamente concluídas. Não alterar o conteúdo de um envelope assinado já emitido; vincular o efeito ao recibo e evento servidor, preservando a semântica existente dos fatos assinados.

### 6.6 Gatilhos de reavaliação

Reavaliar após aceite de Task, avaliação aplicável de Criterion do Objective, resolução de Issue própria, fechamento da canônica de duplicatas, liberação de Blocker, satisfação de dependência, mudança explícita de modo e comando de reconciliação. Entrega sem aceite não fecha Objective.

Dependências e duplicatas podem afetar Objectives fora da subárvore do evento, mas dentro do mesmo Outcome. Construir candidatos por consultas indexadas: proprietário/ancestrais + dependentes atingidos + proprietários de duplicatas afetadas + alvos de Blocker liberado. Deduplicar e avaliar em ordem determinística até ponto fixo, já que um Objective recém-achieved pode satisfazer dependência de outro.

Adotar inicialmente o orçamento de 64 conclusões derivadas por transação da E39, configurável por política de instalação e qualificado com grafos densos. Nunca concluir com leitura truncada. Se o limite de cascata for excedido, registrar uma pendência **idempotente de reconciliação de estado** no mecanismo durável existente e retornar o trabalho já aceito com status `completion_reconciliation_pending`. O processamento interno dessa pendência só reavalia a regra determinística; não seleciona Tasks nem executa agentes. Sem infraestrutura de processamento configurada, não habilitar modo automático para grafos acima do limite suportado e disponibilizar reconciliação explícita.

Preferir fechamento atômico para o caso normal. Qualquer fallback durável deve integrar o outbox existente, sem novo broker e sem pretender consistência imediata onde ela não foi implementada. A UI deve diferenciar “elegível, reconciliando” de “Achieved”.

### 6.7 Reabertura, ciclos e provas antigas

Adicionar um `completion_cycle` monotônico ao Objective ou mecanismo equivalente de aplicabilidade já demonstrado pelo repositório. Incrementar na reabertura e registrar o ciclo em assessments novos usados para achievement. Critérios mantêm suas revisões; o ciclo impede que uma conclusão reaberta volte automaticamente a Achieved com provas de um ciclo anterior.

Não copiar avaliações históricas para o novo ciclo. Reafirmar uma avaliação exige operação explícita autorizada com justificativa e prova atual, que cria outro registro. No perfil legado, preservar semântica histórica até migração.

Não executar autofechamento como efeito de `ReopenObjective`. `StartObjective` não deve produzir um fechamento silencioso usando avaliações históricas. O relatório orienta a revisão necessária do novo ciclo.

### 6.8 Concorrência e falhas

Teste dois últimos aceites concorrentes: uma única Conclusion e evento de achievement. Teste criação de Task concorrente com autofechamento: a ordem de lock decide; se a criação ocorrer primeiro, o Objective não fecha; se o fechamento ocorrer primeiro, a criação é rejeitada até reabertura.

Precondição não satisfeita é resultado normal de elegibilidade e não deve desfazer um aceite de Task válido. Erro real de persistência, corrupção ou falha de transação não pode ser engolido; rollback integral e recibo coerente. Replays não reavaliam nem refazem uma conclusão já incluída no resultado original.

<a id="7-identidades-api-keys-e-autorizacao"></a>
## 7. Identidades, API Keys e autorização

### 7.1 Modelo de identidade

Preservar Namespace, Principal, ActorRef, Credential, CredentialPolicy, SigningKey e perfil CLI como conceitos distintos. Role não é identidade, API Key não é ActorRef, assinatura não substitui permissão e metadata de tenant não autoriza acesso.

A autorização efetiva é a combinação restritiva de:

```text
grant atual do Principal
∩ permissões atuais da credencial e sessão
∩ escopo permitido
∩ capacidade do protocolo
∩ autoridade específica da execução/revisão
∩ precondições de domínio
```

A combinação representa condições cumulativas; ampliar uma delas não elimina as outras. A leitura de capabilities não concede autorização, e uma role é somente um template versionado de permissões concretas.

### 7.2 Catálogo explícito de requisitos

Substituir `commandPermission` baseado em substring por `CommandAuthorizationRequirements` **[novo]**, com mapeamento completo por comando. Comando desconhecido falha fechado; CI falha quando um comando público não possui regra.

Permissões propostas, usando `:` para novos direitos comuns e preservando os nomes `work.contract.*`/`work.review.*` existentes de protocolo:

| Grupo | Permissões propostas |
|---|---|
| Leitura | `state:read`, `credentials:read`, `audit:read` |
| Objectives | `objective:create`, `objective:update`, `objective:start`, `objective:conclude`, `objective:reopen`, `objective:cancel`, `objective:configure_completion` |
| Planejamento de Task | `task:create`, `task:update_definition`, `task:activate`, `task:defer`, `task:assign` |
| Execução legada | `task:claim`, `task:renew`, `task:release`, `task:deliver` |
| Revisão/aceite | `task:review`, `task:accept`, `task:request_changes`, `task:withdraw_delivery` |
| Controle de Task | `task:cancel`, `task:reopen`, `task:intervene` |
| Critérios/prova | `criterion:define`, `criterion:assess`, `criterion:waive`, `records:write` |
| Dependências | `dependency:write` |
| Issues | `issue:create`, `issue:update`, `issue:assign`, `issue:investigate`, `issue:resolve`, `issue:reopen` |
| Blockers | `blocker:create`, `blocker:release`, `blocker:cancel` |
| Segurança/configuração | `credentials:issue`, `credentials:update`, `credentials:revoke`, `credentials:rotate`, `permissions:manage`, `policy:manage` |

Não renomear todos os direitos publicados de uma vez. O catálogo deve documentar quais permissões antigas continuam válidas e qual transformação finita preserva sua capacidade histórica. Não usar wildcard em novas chaves de agentes.

Operações compostas exigem o conjunto de permissões correspondentes aos efeitos reais: criar Task + Criterion não é apenas `task:create`; Issue + Blockers exige ambas as capacidades; aceitar entrega e avaliar critérios exige direitos distintos; revogar autoridade de terceiro requer intervenção.

Um requisito pode depender do owner: definir Criterion em Objective verifica também o escopo daquele Objective; em Task, o escopo da Task. Não inferir permissões pelo nome inglês da operação nem pelo tipo de componente de frontend.

### 7.3 Compatibilidade dos grants

Existem validadores com limite 32 de permissões/operações. Ao ampliar o catálogo, atualizar esse limite de forma coordenada, por exemplo para **128 valores explícitos**, em domínio, contratos, storage, API, CLI e UI. Deduplicar e rejeitar nomes desconhecidos; ausência de permissão significa negar.

Criar uma tabela de expansão **versionada e congelada** dos direitos antigos. Ela só representa capacidades existentes no baseline; não autoriza automaticamente comandos adicionados no futuro. Uma credencial legada ampla não passa a poder emitir chaves ou configurar autofechamento sem concessão explícita.

Grants e políticas são interseccionados após canonicalização. Não expandir o grant e esquecer de restringir a CredentialPolicy. Não converter uma lista vazia em “todas”. Para credenciais novas, política explícita é obrigatória desde a emissão.

### 7.4 Roles administrativas

Implementar presets `reader`, `planner`, `executor`, `reviewer`, `issue_operator` e `administrator`. `custom` indica lista editada, não uma permissão. Ao emitir chave, materializar a lista concreta e registrar `role_template_id`/`role_template_version` como proveniência. Alterar o template no futuro não amplia chaves já emitidas.

Executor não recebe assessment, aceite, configuração de conclusão, gerenciamento de chaves ou alterações estruturais por padrão. Reviewer não recebe aquisição de execução por padrão. Planner pode criar e preparar trabalho, mas não aprovar provas automaticamente. Administrator pode administrar; não recebe autorização implícita para forjar revisão assinada ou ignorar critérios.

Mostrar preview de capacidades efetivas antes de salvar. Uma chave só pode exercer o que seu Principal também possui. Não prometer separação independente quando dois perfis apontam para o mesmo Principal.

### 7.5 Emissão atômica e metadados

Evoluir `SecurityService` e `SecurityAdministrationStore`, sem criar outro emissor de tokens. A emissão deve criar Credential + política restritiva + metadados + auditoria + recibo numa transação. Não emitir uma chave ampla e restringi-la em uma segunda requisição.

Metadados propostos: `name` (1–100 caracteres), `description` (até 1.000), `created_at`, `created_by`, `expires_at`, `revoked_at`, `last_used_at`, `role_template_id/version`, `policy_revision`, `replacement_of_id?`. Tamanhos são limites de implementação propostos, não fatos sobre o baseline.

O ID público da credencial pode servir de identificador truncado na lista. Não derivar prefixo exibido de hash secreto sem necessidade e não armazenar token recuperável. Mostrar o segredo uma única vez. Replay de emissão não revela novamente o token; resposta perdida exige revogar e substituir, preservando o comportamento seguro atual.

`last_used_at` é telemetria administrativa coalescida, não escrita em toda transação de domínio. A validação de revogação é síncrona, mesmo que o indicador de último uso seja atualizado posteriormente. Não registrar token, header Authorization, chave privada ou segredo em logs, eventos, URLs, histórico de shell ou screenshots.

### 7.6 Atualização, rotação e revogação

Permissões e escopo são alterados com expected_policy_revision e auditoria. A redução de privilégios tem efeito nas próximas operações/replays autorizados. A UI mostra o impacto em reservas e contratos sem prometer que o processo externo parou.

Rotação cria nova credencial restrita, com vínculo à antiga e política explícita de overlap. Por padrão, **não transfere lease**. Leases/contratos continuam com a credencial antiga até liberação, expiração ou transferência administrativa explícita. O protocolo assinado conserva seu procedimento próprio de rotação de SigningKey; não confundir com trocar token HTTP.

Revogação torna a credencial inválida para renovação, entrega e mutações imediatamente conforme a ordem transacional de revogação versus comando. Não liberar automaticamente uma reserva ainda válida para outro worker sem fencing/intervenção. Oferecer ato explícito de revogar autoridades e registrar a geração nova.

Não permitir remover a última capacidade administrativa recuperável do Namespace sem uma alternativa válida ou procedimento de host previamente demonstrado. Testar autorrevogação de sessão com confirmação e encerramento coerente da UI.

### 7.7 Escopos e autorização de leitura

Manter Namespace obrigatório e restrições existentes de Outcome. Para o primeiro incremento, não criar ACL por Task se a política atual de Outcome atende ao objetivo; a reserva fornece a exclusividade de execução por Task. Quando apresentar dependências, counters, preview e IDs relacionados, respeitar a mesma autorização de leitura e não revelar alvos fora do escopo.

Sessão de browser herda e restringe a credencial de origem. Resolver o ID canônico da chave para ownership não pode ampliar suas permissões nem pular revogação da sessão ou parent.

A separação entre `NewService` confiável embedded e `NewAuthorizedService` deve continuar explícita. A composição standalone usa a versão autorizada. Hosts embedded não recebem um bypass de lifecycle; para operações v2 vinculadas a credencial, precisam fornecer uma identidade confiável real ou escolher um protocolo suportado de host, jamais aceitar `credential_id` do payload como autenticação.

<a id="8-reserva-e-concorrencia-de-execucao"></a>
## 8. Reserva e concorrência de execução

### 8.1 Evoluir o que existe

A reserva legada possui TTL padrão de 300s, mínimo de 30s e máximo de 3.600s; WorkContract possui política própria com esses defaults. Preservar defaults no rollout inicial. Expor duração em segundos no protocolo e minutos legíveis na UI, usando o valor realmente retornado pelo servidor. Não hardcodar 15 minutos em um lugar e 5 em outro.

Adicionar à reserva legada `HolderCredentialID` e proveniência opcional da sessão que adquiriu. Preservar `PrincipalID`, `Actor`, `ClaimID`, `FencingToken`, `AcquiredAt` e `ExpiresAt`. Separar **duração máxima de cada lease** de **tempo contínuo máximo de renovação**: o baseline não estabelece limite total de sessão de trabalho; não introduzi-lo silenciosamente.

### 8.2 Identidade canônica da chave

Para API Key direta, o holder é a Credential autenticada. Para browser, resolver a credencial root usando o vínculo de sessão existente, validando parent, Principal, Namespace, Actor e revogação. `HolderCredentialID` é a chave root; `AcquiredSessionCredentialID` identifica a sessão para auditoria. Duas sessões da mesma chave root podem atuar na mesma reserva, se autorizadas; duas chaves diferentes do mesmo Principal não.

Esse comportamento é deliberado: a reserva é da chave usada, não do cookie transitório. Caso a instalação exija isolamento também por sessão, isso será uma política futura explícita, não uma inferência deste plano.

No protocolo assinado, preservar `SignedContractBinding.CredentialID`; browser continua proibido de exercer autoridade assinada. Não reescrever um binding assinado para usar root de sessão.

### 8.3 Aquisição, renovação, liberação e recuperação

| Operação | Regra |
|---|---|
| Acquire/claim | Nenhuma autoridade concorrente válida; Task pronta; reservar e mudar estado atomicamente. |
| Renew | Mesma credencial canônica, Principal, claim/execution e fence; ainda não expirado; política/grants continuam válidos. |
| Release | Mesmo holder, salvo intervenção; encerrar autoridade sem apagar o trabalho, preservando evento. |
| Reclaim | Só após expiração ou revogação explícita da autoridade; novo claim/execution e fence maior. |
| Administrative intervention | Permissão dedicada, motivo, expected versions, leitura do impacto; não bypass de critérios. |
| Transfer | Não implementar implicitamente. Quando oferecida, deve encerrar a autoridade anterior e emitir outra geração auditada. |

`expires_at == now` é expirado. A decisão usa relógio transacional autoritativo, depois dos locks relevantes, e nunca o timestamp do cliente. Obter lock após esperar vários segundos não pode aproveitar um `now` antigo que mantenha lease artificialmente vivo.

Passagem do tempo não muda lifecycle persistido. Expiração muda a projeção para recuperável/attention_needed. Preservar o fencing monotônico para impedir que uma execução antiga entregue após reacquisição.

### 8.4 Atribuição não é reserva

`assignee_refs` declara responsabilidade. Pode haver múltiplos participantes no modelo atual, mas somente uma autoridade de execução ativa. Não reduzir silenciosamente a cardinalidade existente de Task para um único assignee só porque Issue usa responsável único.

Trocar assignee não transfere lease. UI deve apresentar ambos: “Assigned to” e “Reserved by”, além de credencial identificável, validade e status recuperável. Não apresentar “online”/“executando agora” apenas porque há contrato válido.

### 8.5 Matriz de escrita durante a reserva

- Titular válido: ações da execução autorizadas pelo protocolo.
- Outro executor: leitura permitida; aquisição, renew, release do titular e entrega negados.
- Planner: pode preparar outros itens; definição da Task em execução continua congelada.
- Reviewer: pode revisar a entrega que passou à fila, conforme política; não assume o lease de execução.
- Issue operator: pode reportar problema e criar impedimento permitido; não ganha posse da Task.
- Administrador: pode realizar intervenção explícita; não editar silenciosamente material assinado ou aceitar critérios não satisfeitos.

### 8.6 Recibos e binding da credencial

O replay legado é hoje associado principalmente a Principal/comando/chave de idempotência. Adicionar verificação de credencial proprietária ao resultado das operações protegidas v2 **antes de divulgar o recibo**. Outra chave do mesmo Principal não pode recuperar autoridade de execução obtida pela primeira só porque conhece uma idempotency key.

Preservar identidade e fingerprint de intenções legadas publicadas. Fazer extensão versionada de receipt/binding, sem trocar chaves de deduplicação em silêncio. Replays históricos permanecem históricos; não refazer um claim para “corrigir” vínculo ausente.

### 8.7 Concorrência e mecanismos de storage

Usar Outcome guard, transação e versões existentes. Não adicionar mutex global de processo que falhe em múltiplas instâncias, nem Redis lock paralelo. Para PostgreSQL, preservar a ordem de locks de Namespace/segurança/protocolo antes de Outcome já estabelecida na composição. Para SQLite, preservar aquisição de writer e contrato do adapter. Memory permanece apoio de teste, não demonstra paridade multi-processo.

Serialização curta das mutações de um Outcome não impede execução externa simultânea de Tasks diferentes. Não manter a transação aberta durante o trabalho do agente, uma chamada HTTP/MCP externa ou consulta ao conteúdo de Artifact URI.

<a id="9-issues-blockers-e-dependencias"></a>
## 9. Issues, Blockers e dependências

### 9.1 Ownership e responsabilidade de Issue

Reutilizar ObjectiveID obrigatório, assignee opcional e comandos de atribuição já introduzidos na branch. Não inferir owner de `affected_refs`. Ao criar Issue a partir de Task, obter o Objective da própria Task; a partir de Objective, usar esse Objective; a partir do Outcome ou de material, exigir escolha explícita antes do envio.

A Issue pode afetar mais de uma entidade, inclusive de Objectives diferentes do mesmo Outcome. Isso não muda seu proprietário. Atribuir Issue é responsabilidade declarada e não adquire autoridade de execução sobre Tasks relacionadas.

`StartIssueInvestigation`/`InvestigateIssue` devem manter aquisição de responsabilidade concorrente por versão. Se a operação assume a Issue para o ator atual, não aceitar assignee forjado do cliente. Reatribuição de terceiro exige a permissão correspondente.

### 9.2 Fechamento de Issues e Objectives

Preservar `open`, `investigating`, `resolved`, `wont_fix`, `duplicate`. `open` e `investigating` impedem achievement do Objective proprietário. `wont_fix` exige motivo e não satisfaz Criterion. Uma duplicata só deixa de ser pendência quando sua cadeia canônica válida termina em `resolved` ou `wont_fix`.

Não fechar Objective por filtrar apenas a primeira página de Issues. Usar o relatório de completude já criado no backend. Preservar proteção contra reabertura de Issue que tornaria inconsistente um Objective já Achieved, incluindo proprietários de duplicatas que apontam à canônica reaberta.

Tasks podem ser aceitas enquanto existe Issue no mesmo Objective, quando a Issue não gera um impedimento efetivo para aquela Task e os demais gates estão satisfeitos. A Issue continuará bloqueando o achievement do Objective. Não aumentar esse gate para todas as Tasks sem base causal.

### 9.3 Blockers

Reutilizar alvo único e propagação `direct|subtree`. Subtree segue a hierarquia estrutural, não dependências. Um Blocker no pré-requisito pode fazer um dependente aguardar, mas isso é `hard_dependency_unsatisfied`, não uma nova instância de Blocker no dependente.

Ao resolver Issue com Blockers, apresentar cada alvo, efeito e estado; exigir seleção e confirmação explícitas dos Blockers a liberar. Reutilizar `ResolveIssueAndBlockers` e criação composta existente, incluindo o comando multi-Blocker já desenvolvido. Operação composta inválida falha inteira.

### 9.4 Dependências: semântica única

`source_ref` depende de `target_ref`. `hard` bloqueia os pontos definidos de prontidão e conclusão; `advisory` somente informa. Ambos respeitam mesmo Outcome, grafo válido e as restrições atuais de ciclo. Não tratar cancelamento do alvo como dependência satisfeita: `cancelled` não é `achieved` nem `done`.

“Depende de” edita edges cujo source é a entidade selecionada. “Dependem deste” mostra edges de outras entidades; alterar ou remover uma dessas edges exige autorização e editabilidade do **source real**, não apenas permissão sobre o target exibido.

Dependências saindo de Task só podem mudar em Backlog, conforme ADR-041. Não executar Defer automaticamente para habilitar edição. Objectives em execução não podem adquirir um hard dependency atualmente insatisfeito que contradiga a execução de seus descendentes; exigir preview e replanning/intervenção explícitos conforme os gates do agregado.

### 9.5 Edição em lote e impacto

Reutilizar `PreviewDependencyChanges` e `ApplyDependencyChanges`. Preview deve mostrar snapshot, entidades afetadas, edges antes/depois, bloqueios atuais, indisponibilidade, ciclos e origem da restrição. Apply exige a revisão observada, versões de sources e relations removidas, permissões de cada origem e validação do grafo final.

Não validar apenas a ordem sequencial dos add/remove quando o grafo final é válido; não aceitar um batch cujo resultado final produz ciclo ou espera impossível com hierarchy. Concorrência de duas inserções opostas deve ser resolvida pelo Outcome guard.

A consulta de impacto deve expor direção, profundidade, cobertura completa/parcial e motivo de truncamento. Contagens parciais não aparecem como totais completos. Na UI, preferir lista legível de pré-requisitos e dependentes; o Canvas é uma visualização adicional, não a única forma de editar.

<a id="10-experiencia-de-criacao-e-kanban"></a>
## 10. Experiência de criação e Kanban

### 10.1 Matriz de entrypoints

| Contexto | Ação principal de criação | O que não deve acontecer |
|---|---|---|
| Workspace sem Outcome | `Create Outcome` | Criar Task sem escopo/proprietário. |
| Card/inspector de Outcome | `Add Objective` | Abrir formulário de Task como se ela fosse filha direta do Outcome. |
| Kanban de Objectives de Outcome | `Add Objective` | Botão variar com a última seleção global de outro contexto. |
| Card/inspector de Objective | `Add Task`; `Report Issue` como ação de problema | Selecionar owner invisível ou trocar silenciosamente de Objective. |
| Kanban de Tasks | `Add Task` com Objective fixado | Criar sub-Objective por default ou omitir owner. |
| Contexto de Issues de Objective | `Report Issue` com owner fixado | Usar affected_refs como owner. |
| Lista global de Tasks | `Create Task` → escolher Objective | Submeter antes da escolha; preselecionar arbitrariamente o primeiro. |
| Task selecionada | `Report Issue`, material e operações permitidas | Criar “Task filha” sem suporte de domínio. |

Os endpoints continuam podendo ser endereçados sob `/outcomes/{id}/work-items` por isolamento. Isso não significa propriedade estrutural direta no Outcome. Não renomear rotas apenas por essa semântica; o payload e o domínio exigem Objective.

Centralizar roteamento de criação numa função pura de apresentação e nas capabilities do backend. Revisar `actions.js`, summary do Canvas, `canvasCreate`, `add-item`, estados vazios, menus, busca, atalhos, batch planner e onboarding, não apenas o footer do Kanban.

### 10.2 Formulário de Objective

Ordem proposta:

1. Contexto `Outcome / parent Objective` e texto curto sobre a condição a alcançar.
2. Título obrigatório, com foco inicial; descrição resumida opcional.
3. `Success criteria`: seção visível com “Add criterion”; pode estar vazia para Planned, mas mostrar requisito para iniciar.
4. `Completion`: automática/manual, valor explícito e explicação da delegação. Mostrar restrição de permissão quando necessário.
5. `Required for Outcome`: preservar default/editor atual, sem confundir com conclusão automática.
6. `Dependencies` colapsável, com seleção simples e hard/advisory legíveis.
7. `Advanced`: prioridade, responsáveis e reparenting permitido.

Botão principal: `Create Objective`; secundário opcional `Create and add another`, preservando o contexto e sem duplicar envios incertos. Não usar `Save changes` para uma criação se existe espaço para copy específica.

Não obrigar usuário a preencher IDs, expected_version, namespace, reason técnico ou configuração de assinatura para criar um Objective. IDs/versões são resolvidos pelo cliente confiável e validados no servidor.

### 10.3 Formulário de Task

Exibir proprietário no topo; a criação contextual mantém o campo fixo, com opção deliberada de mudar quando o fluxo permitir. Título obrigatório; resultado esperado/instruções resumidas; critérios visíveis; estado inicial Backlog explicado. Dados avançados incluem ExecutionSpec completo, dependências, prioridade, assignees e agendamento.

Separar “obrigatório para salvar Backlog” de “obrigatório para disponibilizar”. O objetivo é salvar uma ideia rapidamente sem criar uma Task acidentalmente executável. `Create in To Do` fica em avançado e só é habilitado quando o servidor comprova os gates; enviar a requisição diretamente também deve validá-los.

Não duplicar conteúdo automaticamente em description, instructions e deliverables. Mapear campos curtos de criação para uma única fonte canônica; mostrar o editor técnico expandido quando necessário. Preservar dados digitados ao expandir seções ou escolher dependência.

### 10.4 Criação com critérios e dependências atômicos

Critérios iniciais já são atômicos. Estender `CreateWorkItemCommand`/`CreateObjectiveCommand` ou a primitiva de plano existente com `initial_dependencies` quando a UI oferecer dependências no mesmo formulário. Source é a entidade criada; target e strength são escolhas explícitas.

Para batch Objective + Tasks, reutilizar `CreatePlanItems`, local keys e ordenação topológica. Se o formulário declara que criou tudo, Task, critérios e dependências devem estar na mesma transação. Não criar Task, falhar nas dependências e retornar sucesso genérico sem avisar o estado parcial.

Conservar limite existente de 50 itens e 100 critérios por batch, salvo alteração deliberada fundamentada em testes. Tratar os limites como contrato; não aumentar para compensar ausência de paginação ou UX.

### 10.5 Comportamento do Kanban

Manter modal contextual sobre Board montado e apenas um dialog modal aberto. A operação substitui o Kanban temporariamente e o retorno restaura contexto, filtros, item, scroll, foco e câmera.

Ao arrastar: carregar capability atual; mostrar operação e consequência; destacar requisitos faltantes; confirmar a intenção; enviar comando; só mover card após confirmação do servidor. Operação composta Backlog → execução continua com duas confirmações — ativar e adquirir — sem mudança visual antecipada.

Se o Objective estiver Planned, exibir banner `Start this Objective before making its Tasks available`, com ação `Start Objective` se autorizada e link para critérios/dependências impeditivas. Não oferecer um botão que starta implicitamente parent e Task no mesmo gesto.

Card de Task mostra título, estado, assignees resumidos, holder/expiração quando relevante, critérios atuais e motivo de bloqueio. Não lotar o card com todos os detalhes; o inspector mantém as operações operacionais perto do respectivo contexto.

### 10.6 Prova, entrega e aceite

Separar ações:

- `Review proof`: inspeciona critérios e material; não muda estado;
- `Submit for verification`: entrega real quando suportada pelo protocolo;
- `Approve delivery`: decisão real, apenas ao revisor autorizado;
- `Request changes`: rejeita a rodada com motivo e findings.

Para signed-host, apresentar `Continue in authorized host` com identificadores e comando apropriado sem token. Não dar a impressão de que o browser enviou uma entrega quando apenas abriu checklist.

Cada Criterion apresenta definição/revisão, estado da avaliação, evidências e ação permitida. Checkbox indica assessment atual `met`, não presença de Evidence. Abrir form para avaliação explícita; reprovação e waiver têm semântica própria. Não concluir automaticamente uma Task ao marcar a última checkbox: a decisão de aceite da entrega continua existindo.

### 10.7 Objective e progresso

Mostrar contagem autoritativa `Accepted 7 / 8 Tasks`, contagem separada de canceladas, critérios do Objective e pendências próprias. Se contagem é limitada ao carregado, usar esse rótulo e não calcular porcentagem global enganosa.

Modo automático: exibir `Automatically achieved after all requirements are met` e, quando pendente, o motivo concreto. Ao acontecer, timeline inclui `Objective achieved automatically`, política, critérios e comando causador. Modo manual: exibir `Ready to achieve` quando elegível e ação autorizada.

Não fechar Outcome automaticamente. Depois de todos os Objectives exigidos concluídos, oferecer a operação explícita de aceitação de Outcome e seus próprios requisitos.

### 10.8 Dependências na interface

No inspector, duas seções: `Depends on` e `Required by`. Cada linha mostra título, tipo, state, hard/advisory, satisfeita/pendente e efeito atual. Botões de adicionar/remover usam a direção correta. Ao operar em “Required by”, preview declara qual outra entidade será alterada.

No Canvas, modo conectar começa por entidade source selecionada, escolhe target, confirma frase `A depends on B`, strength e efeito. Não reaproveitar arraste de posição para criar dependência implicitamente. Oferecer a mesma operação por formulário.

### 10.9 Issues no Kanban

Reutilizar modal com contexto de Issues para Objective e listas autorizadas no Outcome. Lanes de apresentação: `Open`, `Investigating`, `Closed`; Closed mantém badge de `Resolved`, `Won't fix` ou `Duplicate`. Arrastar para Closed abre escolha da decisão, não um status setter.

Inspector mostra Objective proprietário, assignee, reportador, entidades afetadas, Blockers e Tasks corretivas. Resolver a Issue deixa a lista de Blockers explícita; liberar selecionados é ato confirmado. Ao tentar concluir Objective, linkar suas Issues pendentes sem misturar problemas de outros proprietários.

### 10.10 Acessibilidade, erros e retomada

Manter operação equivalente por clique/toque e teclado no inspector. A alternativa não precisa reintroduzir select de status nos cards. Teclado sozinho não é substituto suficiente para alternativa de ponteiro sem arraste; planejar ambos, conforme WCAG 2.5.7 [E01].

Usar foco previsível no primeiro campo e retorno ao opener, nomes acessíveis, alertas vivos sem excesso, estados não dependentes apenas de cor, seleção com texto e escala adequada. Testar 320/390 px, HD, QHD, 4K e zoom 125/150/200/400% sem assumir que screenshot prova acessibilidade integral.

403 não destrói o draft; explica que a permissão mudou. 409 de versão não reescreve expected_version automaticamente; permite comparar. Timeout mantém bytes e idempotency key. Write confirmado + falha no refresh faz retry só de leitura. Troca de Workspace/Outcome cancela respostas atrasadas e impede que dados de outro contexto reapareçam.

<a id="11-arquitetura-arquivos-e-contratos"></a>
## 11. Arquitetura, arquivos e contratos

### 11.1 Responsabilidades

```text
Browser / HTTP / MCP / Go SDK / CLI
                  |
                  v
Application: comandos, políticas, autorização, UoW e projeções
                  |
             Domain + Ports
                  |
     Memory / SQLite / PostgreSQL
                  |
   eventos duráveis + outbox existentes
```

Transporte transforma DTO; domínio valida invariantes puras; Application carrega estado/coordena agregados; storage implementa atomicidade e consultas. Nenhuma consulta externa de URL de Evidence/Artifact pertence à transação de conclusão. O WOS registra/verifica referências e decisões, não executa testes remotos.

### 11.2 Legenda do inventário

**[A]** arquivo-alvo de ajuste/reuso: existente no baseline ou introduzido por uma etapa anterior. A marca não afirma leitura integral de cada arquivo; confirmar existência, nome e símbolos na E01. **[N]** caminho proposto novo, não afirmação de existência. **[G]** arquivo gerado: editar a fonte/gerador e regenerar. **[V]** localizar/confirmar a variante concreta na execução, sem inventar que já foi inspecionada.

| Camada | Caminhos e responsabilidade |
|---|---|
| Contrato | [A] `AGENTS.md`, `ROADMAP.md`, `docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md`; [N] ADRs operacionais e `docs/implementation/operational-consolidation/`. |
| Lifecycle | [A] `packages/wos-core/domain/{objective,work_item,readiness,workspace_preconditions,work_item_operational_state}.go`; [N] `packages/wos-core/domain/operational_policy.go`. |
| Gates | [A] `packages/wos-core/application/{service,lifecycle_service,planning_creation,dependencies,work_contract_guards,workspace_preconditions}.go`; [N] `packages/wos-core/application/operational_gates.go`. |
| Completion | [A] `packages/wos-core/application/{completion_policy,workspace_conclusion,reviewer_policy}.go`; [N] `packages/wos-core/domain/objective_completion.go`; [N] `packages/wos-core/application/{objective_completion,objective_completion_query,objective_completion_effects}.go`. |
| Pipeline/eventos | [A] `packages/wos-core/application/pipeline.go`; [A] `packages/wos-core/domain/{event,audit,idempotency}.go`; [N] teste de composição e efeitos de completion. |
| Prova/revisão | [A] `packages/wos-core/domain/{criterion,documentary,work_submission,signed_review}.go`; [A] `packages/wos-core/application/{signed_return,signed_review_return,work_contract_service,workspace_proof}.go`; [N] `packages/wos-core/domain/{task_delivery,task_delivery_decision}.go`; [N] `packages/wos-core/application/{task_delivery,task_verification_query}.go`. |
| Autorização | [A] `packages/wos-core/{ports/authorization.go,application/authorization.go,application/security.go,application/security_admin.go,application/effective_permissions.go}`; [N] `packages/wos-core/application/command_authorization_catalog.go`. |
| Credenciais | [A] `packages/wos-core/domain/signing_identity.go`; [A] `packages/wos-core/ports/{security,security_admin,signing_identity}.go`; [N] `packages/wos-core/application/{credential_management,permission_templates}.go`. |
| Leases | [A] `packages/wos-core/domain/{work_item,work_item_lease,work_contract}.go`; [A] `packages/wos-core/application/{lease_service,lease_commands,signed_lease,signed_acquisition,work_contract_service}.go`; [N] `packages/wos-core/application/credential_authority.go`. |
| Issues/dependências | [A] `packages/wos-core/application/{issues_blockers,issues_blockers_compound,issue_assignment,issue_completion,issue_work_relation,dependencies,dependency_impact}.go`. |
| Ports novos | [N] `packages/wos-core/ports/{operational_policy,objective_completion,task_delivery}.go`. Adicionar somente capacidades necessárias; não exigir um mega-repository genérico. |
| Storage fonte | [A] `packages/wos-core/storage/sqlite/{repositories,security,security_admin,access,workspace_read,issues_blockers,migrations}.go`; [N] `packages/wos-core/storage/sqlite/{operational_policy,objective_completion,task_deliveries}.go`. |
| Storage memória | [A] diretório `packages/wos-core/storage/memory/`; [N] implementações correspondentes de novos ports e contracts de rollback. |
| Storage PG | [G] equivalentes sob `packages/wos-core/storage/postgres/`, incluindo migrações geradas; [A] `tools/postgresgen/generate.py` e templates específicos quando necessários. |
| Catálogo/API | [A] `packages/wos-api/commands/{catalog,contracts,workspace_schema,workspace_results}.go`; [G] `packages/wos-api/commands/catalog_generated.go`, `packages/wos-api/commands.openapi.json`; [A/V] `packages/wos-api/openapi.yaml`. |
| HTTP | [A] `packages/wos-api/http/{handler,commands,security,effective_permissions,work_items,objectives,issues_blockers,workspace}.go`; [N] `packages/wos-api/http/{task_deliveries,objective_completion,credential_management}.go` se a separação melhorar coesão. |
| MCP | [A] `packages/wos-api/mcp/{workspace,security,queries,resources}.go`; [G] `packages/wos-api/mcp/commands_generated.go`. |
| SDK | [A/V] `packages/wos-sdk-go/{client,workspace}.go`; [G] métodos do catálogo; [N] queries tipadas para completion, credenciais e verificação quando não forem geradas. |
| CLI | [A] `packages/wos-cli/{help,workspace_commands_v2,workspace_planning,workspace_inspect,work_commands_v2,lease_commands_v2,review_commands_v2}.go`; [N] `packages/wos-cli/{credential_commands_v2,issue_commands_v2,objective_commands_v2,dependency_commands_v2,task_delivery_commands_v2}.go`. |
| UI base | [A] `packages/wos-api/web/assets/{app,actions,human-forms,workspace-form-fields,workspace-actions,workspace-view,workspace-kanban,kanban-operations,workspace-entity-context,reference-picker,en-US}.js`. |
| UI novos | [N] `packages/wos-api/web/assets/{creation-routes,api-keys-view,api-key-form,task-delivery-form,objective-completion-view,dependency-editor}.js`; adaptar CSS existente, sem novo framework. |
| Testes | [A] `tests/web/`, suites Domain/Application/storage; [N] cenários identificados na seção 15 e novos arquivos de teste próximos das respectivas implementações. |

Arquivos novos só serão criados quando a responsabilidade não estiver adequadamente coberta por um módulo existente. Os nomes acima são alvos deliberados; agentes devem registrar renomeações necessárias, não criar dois módulos com a mesma responsabilidade.

### 11.3 Contratos de consulta propostos

`GetObjectiveCompletionReport`, `GetTaskVerificationView`, `ListCredentials`, `GetCredentialEffectivePermissions` e leitura completa de resumo do Kanban usam escopo autenticado, snapshot coerente e paginação/cobertura explícita.

Não retornar tokens, digests de autenticação ou private keys. Operadores comuns veem a identidade operacional mínima do holder; dados administrativos completos exigem permissão de credenciais. Nenhum booleano `can_*` do frontend concede autoridade.

Capabilities devem informar `action_id`, disponibilidade, transport (`browser_command|signed_host` já existente), reason_codes, entidades relacionadas, versão observada e remédios autorizados. Não extrair regras de negócio por regex de mensagens em inglês.

### 11.4 Taxonomia de erros proposta

Preservar códigos públicos existentes; adicionar reason_codes estruturados onde faltam. Exemplos novos:

| reason_code | Significado e remédio |
|---|---|
| `objective_not_started` | Iniciar Objective após cumprir os seus requisitos. |
| `ancestor_objective_not_started` | Inspecionar/iniciar ancestral indicado; nenhuma cascata implícita. |
| `task_objective_required` | Selecionar ou criar Objective no mesmo Outcome. |
| `task_definition_incomplete` | Completar definição/critério antes da ativação. |
| `task_delivery_required` | Entregar trabalho antes do aceite. |
| `task_review_pending` | Revisão ainda não decidida. |
| `task_review_protocol_mismatch` | Usar operação do protocolo real; sem fallback inseguro. |
| `assessment_not_current` | Revisão do critério, rodada ou ciclo diverge. |
| `lease_owned_by_other_credential` | Chave não é titular; não divulgar segredo/autoridade. |
| `lease_expired` / `stale_execution` | Recuperação explícita com nova geração. |
| `objective_completion_empty` | Sem trabalho aplicável para automação; revisão manual. |
| `objective_completion_requires_manual_ack` | Há cancelamento/waiver que não pode ser aceito automaticamente. |
| `objective_completion_delegation_missing` | Autoridade automática não configurada validamente. |
| `dependency_hierarchy_deadlock` | A relação proposta cria espera impossível com hierarchy. |
| `operational_migration_required` | Estado legado precisa de reconciliação explícita. |

Autenticação ausente/inválida: 401; permissão negada: 403; conflito de versão/autoridade: contrato 409 existente; precondição operacional: 412 onde o mapping atual permitir. Não mudar indiscriminadamente códigos históricos; testar a compatibilidade do mapeamento real em `http/handler.go`.

Exemplo **proposto** de detalhe de erro, adaptável ao envelope real:

```json
{
  "code": "precondition_failed",
  "message": "Start the Objective before making this Task available.",
  "details": {
    "reason_code": "objective_not_started",
    "subject_ref": {"kind": "work_item", "id": "<task-id>", "namespace_id": "<namespace-id>", "outcome_id": "<outcome-id>"},
    "related_refs": [{"kind": "objective", "id": "<objective-id>", "namespace_id": "<namespace-id>", "outcome_id": "<outcome-id>"}],
    "remedies": [{"action_id": "StartObjective"}]
  }
}
```

IDs entre `<...>` são placeholders explicativos, não IDs válidos para enviar. A presença de um remedy só orienta a próxima consulta autorizada; não autoriza start por quem não tem a permissão.

<a id="12-persistencia-e-migracao"></a>
## 12. Persistência e migração

### 12.1 Alocação de migrações

No baseline existem migrações até 0036. Reservar números seguintes somente depois de verificar a HEAD real. Nomes sugeridos, **provisórios**:

| Arquivo SQLite fonte [N] | Conteúdo |
|---|---|
| `packages/wos-core/storage/sqlite/migrations/0037_operational_profile.sql` | Perfil operacional do Outcome, modo/ciclo de conclusão do Objective e defaults de compatibilidade. |
| `packages/wos-core/storage/sqlite/migrations/0038_credential_management.sql` | Metadados, versão de permissões/template, auditoria e binding necessário aos novos recibos. |
| `packages/wos-core/storage/sqlite/migrations/0039_lease_credential_binding.sql` | Vínculo de lease legado com credencial e classificação de reservas sem atribuição demonstrável. |
| `packages/wos-core/storage/sqlite/migrations/0040_task_deliveries.sql` | Entregas/decisões não assinadas, vínculo pendente da Task e índices de fila. |
| `packages/wos-core/storage/sqlite/migrations/0041_objective_completion_authority.sql` | Delegação, proveniência de conclusão, ciclos e reconciliação durável especificada na E39. |

Se número já estiver ocupado, renumerar novas migrações e registrar o mapa. Não editar o conteúdo/checksum de migração já aplicada. PostgreSQL é gerado a partir de SQLite e transformações explícitas em `tools/postgresgen/generate.py`; atualizar esse gerador para SQL diferente por engine, em vez de editar o derivado manualmente.

### 12.2 Constraints e índices

Preservar FKs compostas por Namespace/Outcome/ID. Novo PendingDeliveryID deve referenciar entrega da própria Task, com validação que impossibilite conectar entrega de outro escopo. Garantir no máximo uma entrega não assinada aberta por Task e exclusão mútua com revisão assinada.

Índices sugeridos: Tasks por `(namespace_id,outcome_id,objective_id,lifecycle)`; Issues já indexadas por owner/lifecycle; entregas por scope/Task/status; credenciais por Namespace/Principal/status/expiry; policies por credencial/revisão; dependências por source e target com lifecycle/strength. Confirmar os existentes antes de criar duplicatas.

Não alterar a representação exata de uint64/fencing sem estratégia de compatibilidade. Novos wires usam string decimal; SQLite e PostgreSQL mantêm a precisão usada no repositório, incluindo limites e ordenação apropriados. Timestamps continuam com a precisão do adapter.

### 12.3 Auditoria pré-migração

Gerar relatório read-only determinístico contendo: Tasks/Issues órfãs, Tasks fora de Backlog sob Objective Planned, ancestrais incoerentes, contratos/revisões vivos, leases sem credencial, dependências impossíveis, Tasks sem critérios, políticas de chave ausentes e Objectives terminais com trabalho aberto.

Para cada caso, apresentar IDs, versões, escopo e correção sugerida sem aplicar automaticamente. Não selecionar “o único Objective” como owner de dado legado nem escolher “a única chave atual” como titular histórico de um lease; cardinalidade atual não comprova proveniência passada.

Reutilizar `tools/acceptance/issue_ownership_backfill.py` para ownership e expandir somente se a responsabilidade permanecer clara. Uma ferramenta nova de audit operacional pode compor esse relatório, sem duplicar a lógica de backfill nem fazer writes por default.

### 12.4 Reservas legadas não atribuíveis

Se evento/recibo original comprovar credential_id, a migração pode preencher o vínculo por mapeamento determinístico revisado. Caso contrário, não inventar. Drenar a reserva até expirar ou realizar intervenção explícita e reacquisição com fence novo. Resultado histórico continua indicando proveniência limitada.

Não permitir renovação indefinida de reserva sem binding sob operacional v2. A ativação do novo perfil deve ser bloqueada enquanto houver autoridade incompatível não reconciliada.

### 12.5 Defaults e transformação dos dados

Migrar schema de forma aditiva primeiro. Dados antigos recebem perfil compatível e conclusão manual. Novos endpoints não oferecem emissão de chave sem política; antes do cutover, revisar grants e aliases.

Migrar Outcomes abertos individualmente sob guard e expected_outcome_revision após preview. Não iniciar Objectives automaticamente só porque possuem Tasks já em execução. O operador escolhe iniciar com critérios válidos ou drenar/replanejar trabalho, em comandos auditados.

Backfill não reescreve Evidence, Assessment, signatures, ConclusionHistory, lifecycle de Task ou autoria para “validar” a nova regra. Migração de conteúdo e mudança de contrato são evidências diferentes.

### 12.6 Restore e versões de writer

Testar fresh install, upgrade real e restore limpo em ambos os bancos. O schema e o perfil devem impedir writer antigo incompatível de ignorar os novos gates; versões antigas não podem continuar escrevendo em paralelo após endurecimento.

Rollback de aplicação só é permitido para versão compatível com o schema e dados existentes. Não propor drop destrutivo das tabelas de entrega para voltar a um binário antigo. Se houver necessidade de voltar ao estado anterior completo, usar backup verificado em destino separado e reconciliar o período posterior; não perder histórico silenciosamente.

<a id="13-http-mcp-sdk-e-cli"></a>
## 13. HTTP, MCP, SDK e CLI

### 13.1 Surface de comandos

O HTTP já dispõe de `POST {prefix}/commands/{command}` e catálogo em `GET {prefix}/commands`. Usar esse mecanismo para comandos de domínio novos, com geração MCP/SDK. Para administração, evoluir `POST {prefix}/security/commands` e os caminhos existentes de credenciais sem criar uma segunda fonte de política.

Consultas adicionais podem usar:

```text
GET  {prefix}/namespaces/{namespace_id}/outcomes/{outcome_id}/objectives/{objective_id}/completion
GET  {prefix}/namespaces/{namespace_id}/outcomes/{outcome_id}/work-items/{work_item_id}/verification
GET  {prefix}/namespaces/{namespace_id}/outcomes/{outcome_id}/task-deliveries/{delivery_id}
GET  {prefix}/namespaces/{namespace_id}/credentials
GET  {prefix}/namespaces/{namespace_id}/credentials/{credential_id}/effective-permissions
```

São rotas **propostas**. `{prefix}` é configurável, como no handler existente. Usar `/task-deliveries`, não `/deliveries`, que já representa entregas da integração/outbox.

Todas as routes legadas de ação e o catálogo genérico invocam a mesma Application. Não corrigir apenas o catálogo e deixar um `PATCH`/action endpoint antigo contornar o gate. CSRF continua obrigatório para sessão browser; autenticação de chave não elimina validação de scope e payload.

### 13.2 Operações novas propostas

`ConfigureOutcomeOperationalProfile`, `SetObjectiveCompletionMode`, `ReconcileObjectiveCompletion`, `SubmitTaskForReview`, `AcceptTaskDelivery`, `RequestTaskDeliveryChanges`, `WithdrawTaskDelivery`, administração atômica de credenciais e consultas associadas. Nomes podem ser ajustados ao padrão real de geração, preservando a intenção e a rastreabilidade.

Não expor `AutoAchieveObjective` como comando público que qualquer agente pode invocar fingindo ser o sistema. A API pública pode solicitar reconciliação autorizada; o backend decide se a política delegada permite o efeito.

### 13.3 CLI: preservar o produto atual

O catálogo atual usa **`wos`** como executável principal, com compatibilidade `wosctl` na distribuição. Manter ambos conforme o mecanismo existente. Não criar outro package, instalador ou arquivo de configuração concorrente.

Reutilizar `planning apply`, `inspect`, `work checkout/next/renew/finish/recover`, `review checkout/finish/recover`, perfis e journal. Os novos comandos amigáveis devem chamar essas mesmas primitivas/SDK ou wrappers tipados, sem criar regras locais independentes.

CLI deve retornar JSON estável com operation, entidades/revisões, efeitos derivados, reason_codes e disposition. Não traduzir `delivered_for_review` para `accepted`; `finish` não significa sempre conclusão da Task. Não imprimir tokens em `--output json` de consultas/status.

### 13.4 Comandos propostos para administração e operação

Os exemplos abaixo são **contratos de UX a implementar**, não afirmação de que os subcomandos existem no baseline:

```bash
# Leitura e administração; usar arquivo protegido para o segredo emitido.
wos --profile admin auth keys list --output json
wos --profile admin auth keys create --file executor-key.json --secret-output /secure/executor.token
wos --profile admin auth keys permissions KEY_ID --output json
wos --profile admin auth keys revoke KEY_ID --expected-version VERSION --reason "Rotating worker identity"

# Planejamento e gates.
wos --profile planner --outcome OUTCOME_ID objective start OBJECTIVE_ID --expected-version VERSION
wos --profile planner --outcome OUTCOME_ID objective completion OBJECTIVE_ID --output json
wos --profile planner --outcome OUTCOME_ID task activate TASK_ID --expected-version VERSION

# Issues e dependências.
wos --profile operator --outcome OUTCOME_ID issue create --file issue.json
wos --profile operator --outcome OUTCOME_ID issue investigate ISSUE_ID --expected-version VERSION
wos --profile planner --outcome OUTCOME_ID dependency preview --file changes.json
wos --profile planner --outcome OUTCOME_ID dependency apply --file changes.json
```

Para work/review assinados, manter a sintaxe atual publicada. Uma Task nativa não assinada usa dispatch declarado de capabilities; signed não cai em fallback nativo quando um envelope ou key está ausente. Em erro de incompatibilidade de protocolo, orientar setup/host correto, sem tentar outra forma de mutação.

`--secret-output` deve criar arquivo exclusivo com acesso restrito, não sobrescrever existente por default e usar estratégia nativa equivalente no Windows. Segredos não entram em args, YAML versionado ou logs. Criar chave no servidor e falhar no armazenamento local requer receipt e orientação de revogação/substituição; não mentir que a emissão foi desfeita.

### 13.5 Paridade dos contratos

Gerar schema, MCP e SDK a cada alteração pública relevante, não apenas no final. Manter `tools/mcpgen`, `tools/openapigen`, `tools/signedgen/generate.py`, `tools/postgresgen/generate.py` e event catalog coerentes.

Para cada comando equivalente, testar HTTP direto, catálogo HTTP, MCP, SDK e CLI contra a mesma base: mesmo gate, mesma permissão, mesmo efeito persistido, mesma lógica de replay e ausência de vazamento. Erros de uso local do CLI podem ter envelope diferente, mas não alterar a decisão do servidor.

Não permitir que o SDK embutido faça escrita direta nos adapters. Consumidor embedded pode compor a autenticação, mas as invariantes do domínio e do perfil continuam executadas pela Application.

<a id="14-etapas-de-implementacao"></a>
## 14. Etapas de implementação

### Como usar as etapas

Executar na ordem indicada, respeitando dependências. `E01` etc. são IDs deste plano; não substituem os números do planejamento anterior. A seção 17 relaciona os dois documentos. As etapas marcadas como reaproveitamento continuam exigindo confirmação na HEAD real.

Os caminhos desta seção usam `[N]` para arquivo proposto novo, `[A]` para ajuste/reuso e `[G]` para derivado gerado. Um arquivo de teste novo com nome semelhante a um existente deve ser integrado à suíte existente quando isso evitar duplicação; registrar a escolha.

Cada commit inclui testes pertinentes, documentação de contrato alterado e geração necessária. Não adiar testes de segurança para E73. Recursos incompletos permanecem não anunciados nas capabilities de release; não habilitar `operational_v2` para produção enquanto seus invariantes, revisão e migração não estiverem completos.

A evidência por etapa fica em `docs/implementation/operational-consolidation/evidence/E##.md` e no manifest: SHA, arquivos, comandos, ambiente, resultados reais, skips, limitações e relação com o plano anterior. Não copiar resultado de outra SHA como teste da etapa atual.

## Fase A — Baseline e contrato operacional

<a id="e01"></a>
### E01 — Congelar baseline e reconciliar o plano existente

**Objetivo:** Estabelecer uma base verificável sem sobrescrever implementações já presentes.

**Depende de:** Nenhuma; iniciar por esta etapa.

**Arquivos:** [A] `AGENTS.md`; [A] `ROADMAP.md`; [A] `docs/ux/operational-kanban/issue-and-dependency-plan.md`; [N] `docs/implementation/operational-consolidation/baseline.md`; [N] `docs/implementation/operational-consolidation/manifest.json`; [N] `docs/implementation/operational-consolidation/acceptance.json`.

**Implementação:**

1. Confirmar PR #116 aberto/Draft, branch, SHA e worktree. Ler o contrato canônico, ADRs 038–042 e instruções locais.
2. Mapear as 44 etapas anteriores como implementada/verificada, implementada/não qualificada, parcial ou ausente, anexando arquivos e evidências por SHA.
3. Reconciliar a divergência documental sobre etapas 25–27; não concluir pelo texto do título ou pela existência da migração.
4. Executar baseline das suítes relevantes e guardar falhas anteriores separadamente; registrar indisponibilidade de PostgreSQL/browser sem classificá-la como sucesso.
5. Criar manifest com status planned para E01–E84, dependências, evidências e responsável por execução; iniciar acceptance.json com os cenários da seção 15 como não executados.

**Testes/verificação:** Inspeção de diffs, schema atual, catálogo de comandos, toolchain e resultados do baseline. Nenhuma mutation em dados reais.

**Aceite:** Outro agente identifica exatamente a SHA, o que já existe e o primeiro gap sem depender desta conversa.

**Commit sugerido:** `docs(wos): E01 record operational implementation baseline`.

<a id="e02"></a>
### E02 — Registrar ADRs e defaults do novo contrato

**Objetivo:** Tornar explícitas as mudanças de lifecycle, autorização, revisão e conclusão.

**Depende de:** E01

**Arquivos:** [A] `AGENTS.md`; [A] `docs/adr/README.md`; [A] `docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md`; [N] `docs/adr/0043-operational-objective-gates.md`; [N] `docs/adr/0044-credential-bound-authority.md`; [N] `docs/adr/0045-policy-driven-objective-completion.md`; [N] `docs/adr/0046-authenticated-task-verification.md`.

**Implementação:**

1. Confirmar números de ADR livres; os nomes 0043–0046 são propostos, não reservas preexistentes.
2. Documentar perfil operacional, modo automático/manual, ausência de novas Tasks/lanes persistidas, revisão não assinada delimitada e preservação dos fatos assinados.
3. Atualizar o invariant antigo que proíbe qualquer auto-achievement de Objective, mantendo a proibição de aceitar Outcome automaticamente.
4. Registrar que gate de ancestral é lifecycle, não dependência implícita, e definir prevenção de esperas impossíveis.
5. Definir defaults de criação, regras de migração e permissões de delegação; remover alternativas contraditórias do contrato executável.

**Testes/verificação:** Revisão cruzada dos ADRs com as seções 3–8 e o contrato assinado atual. Validar links e inexistência de instruções contraditórias.

**Aceite:** O algoritmo automático e o caminho humano de revisão têm autorização e semântica documentadas, não decisões pendentes delegadas ao implementador.

**Commit sugerido:** `docs(architecture): E02 define operational lifecycle and authority policies`.

<a id="e03"></a>
### E03 — Preparar fixtures operacionais e testes de caracterização

**Objetivo:** Reproduzir o estado atual e fornecer casos válidos para cada protocolo.

**Depende de:** E01, E02

**Arquivos:** [A] `packages/wos-core/application/test_objective_helpers_test.go`; [A] `packages/wos-core/domain/lifecycle_test.go`; [N] `packages/wos-core/application/operational_profile_fixture_test.go`; [N] `tests/web/fixtures/operational-policy.mjs`; [N] `tests/web/operational-policy-fixture.spec.mjs`.

**Implementação:**

1. Criar fixtures para Outcome ativo, Objective Planned/In Progress, Task em cada fase, duas credenciais do mesmo Principal e outro Principal revisor.
2. Reutilizar os bootstraps reais dos testes específicos de `legacy`, `contracts_v1` e `signed_v2`; o fixture compartilhado não deve rotular uma fase como se tivesse configurado protocolo, credencial assinada ou receipt.
3. Preparar dados comuns com dependências, Issues próprias, Blockers subtree e histórico de conclusão. Usar os cenários existentes de cada protocolo para receipts/entregas pendentes; não fabricar assinatura nem representar entrega ainda não suportada como dado real.
4. Registrar comportamentos antigos como caracterização do perfil compatível; os testes v2 futuros deverão negar os bypasses, não perpetuá-los.
5. Manter IDs determinísticos apenas em fixtures; não alterar geradores ou relógios reais para facilitar testes.

**Testes/verificação:** Fixtures compilam e produzem estados válidos segundo o baseline. Testes de caracterização ficam verdes e não marcam comportamento futuro como concluído.

**Aceite:** As demais etapas têm dados reutilizáveis para comparar protocolos e concorrência sem copiar bootstrap inconsistente.

**Commit sugerido:** `test(core): E03 add operational policy and identity fixtures`.

<a id="e04"></a>
### E04 — Adicionar tipos de política operacional ao domínio

**Objetivo:** Representar perfil, modo de conclusão e ciclo sem efeitos automáticos prematuros.

**Depende de:** E02, E03

**Arquivos:** [N] `packages/wos-core/domain/operational_policy.go`; [N] `packages/wos-core/domain/objective_completion.go`; [A] `packages/wos-core/domain/outcome.go`; [A] `packages/wos-core/domain/objective.go`; [A] `packages/wos-core/domain/serialization_test.go`; [N] `packages/wos-core/domain/operational_policy_test.go`.

**Implementação:**

1. Definir enums compatíveis e validações para compatibility_v1/operational_v2 e manual/automatic.
2. Adicionar campos aditivos aos aggregates com defaults de leitura legada e validação de combinações incompatíveis.
3. Representar completion_cycle e distinguir ciclo de vida de versão do agregado; preparar incremento de reopen.
4. Não permitir modo automatic com uma política de abrangência que ignore trabalho aplicável.
5. Garantir que bytes históricos assinados e fingerprints de comandos antigos não mudem por serialização de zeros/campos novos.

**Testes/verificação:** Roundtrip de dados antigos/novos, enum desconhecido, perfil omitido, combinação inválida, uint64 e snapshots históricos.

**Aceite:** Domínio lê estado antigo sem reinterpretar conclusões e valida configuração nova de forma determinística.

**Commit sugerido:** `feat(domain): E04 add versioned operational and completion policies`.

<a id="e05"></a>
### E05 — Persistir políticas e atualizar adapters

**Objetivo:** Garantir paridade de persistência e migração aditiva das políticas.

**Depende de:** E04

**Arquivos:** [N] `packages/wos-core/storage/sqlite/migrations/0037_operational_profile.sql`; [A] `packages/wos-core/storage/sqlite/repositories.go`; [A] `packages/wos-core/storage/sqlite/nested.go`; [A] `packages/wos-core/storage/sqlite/codec.go`; [A] `packages/wos-core/storage/memory/repositories.go`; [A] `tools/postgresgen/generate.py`; [G] `packages/wos-core/storage/postgres/migrations/0037_operational_profile.sql`; [N] `packages/wos-core/storage/sqlite/operational_profile_test.go`.

**Implementação:**

1. Alocar ordinal confirmado; defaults SQL de dados antigos devem ser compatibilidade/manual, não automatic.
2. Atualizar inserção/leitura/save/clone e qualquer projeção compacta necessária para não perder perfil ou ciclo.
3. Adicionar constraints de valores e versões; atualizar gerador PostgreSQL e manifest de migrations.
4. Confirmar que migrations 0035/0036 existentes não são alteradas ou reaplicadas.
5. Executar contratos de rollback, save expected_version e leitura pós-restart nos adapters.

**Testes/verificação:** Fresh install e upgrade com banco populado SQLite/PostgreSQL; roundtrip Memory; generator zero-diff.

**Aceite:** Perfil e modo sobrevivem a restart e são iguais nos três adapters sem modificar dados históricos de prova.

**Commit sugerido:** `feat(storage): E05 persist operational profile and completion mode`.

<a id="e06"></a>
### E06 — Criar preview e migração explícita de perfil

**Objetivo:** Migrar Outcomes existentes sem iniciar trabalho ou atribuir responsabilidades silenciosamente.

**Depende de:** E05

**Arquivos:** [N] `packages/wos-core/application/operational_policy.go`; [N] `packages/wos-core/application/operational_policy_query.go`; [N] `packages/wos-core/ports/operational_policy.go`; [N] `tools/acceptance/operational_profile_audit.py`; [A] `packages/wos-core/application/planning_creation.go`.

**Implementação:**

1. Implementar audit read-only de owners, lifecycles, critérios, contratos vivos e reservas sem binding.
2. Criar ConfigureOutcomeOperationalProfile com expected_outcome_revision, reason e escopo; manter recurso não anunciado até todos os gates v2 estarem prontos.
3. Negar migração com writers/autoridades incompatíveis; retornar referências e remédios concretos.
4. Registrar decisão de modo de conclusão para Objectives existentes, sem converter todos para automatic.
5. Definir novos Outcomes v2 por default no rollout final, com compatibilidade antiga apenas por procedimento restrito.

**Testes/verificação:** Dry-run não escreve; migração com conflito faz rollback; pendência não altera lifecycle; replay preserva evento/revisão.

**Aceite:** Há procedimento verificável para cada Outcome migrado, sem bypass permanente disponível ao executor.

**Commit sugerido:** `feat(core): E06 add explicit operational profile migration`.

<a id="e07"></a>
### E07 — Implementar predicados compartilhados de execução

**Objetivo:** Concentrar as regras de Objective/Task em funções usadas por queries e commands.

**Depende de:** E04, E05

**Arquivos:** [N] `packages/wos-core/application/operational_gates.go`; [A] `packages/wos-core/domain/readiness.go`; [A] `packages/wos-core/domain/workspace_preconditions.go`; [N] `packages/wos-core/application/operational_gates_test.go`.

**Implementação:**

1. Separar readiness de definição, gate de escopo e disponibilidade de autoridade.
2. Verificar Outcome ativo, Objective In Progress e ancestrais aplicáveis com detecção de corrupção/ciclo estrutural.
3. Produzir reason_codes estáveis e related_refs autorizáveis; não montar mensagens como único contrato.
4. Manter recuperação/observação separadas de avanço operacional.
5. Garantir tratamento intencional do perfil legado e proibir perfil desconhecido em novas mutações.

**Testes/verificação:** Tabela de estados cruzando parent/ancestor/Outcome, Blockers, agendamento, dependências e perfis; leitura não altera versões.

**Aceite:** Mesma entrada de domínio gera mesma decisão e motivos em todos os caminhos.

**Commit sugerido:** `feat(core): E07 centralize objective execution gates`.

<a id="e08"></a>
### E08 — Aplicar o gate a todas as aquisições e ready queries

**Objetivo:** Impedir aquisição por caminhos alternativos quando Objective não iniciou.

**Depende de:** E07

**Arquivos:** [A] `packages/wos-core/application/service.go`; [A] `packages/wos-core/application/lease_service.go`; [A] `packages/wos-core/application/work_contract_service.go`; [A] `packages/wos-core/application/work_contract_next.go`; [A] `packages/wos-core/application/signed_acquisition.go`; [A] `packages/wos-core/application/signed_next.go`; [A] `packages/wos-core/application/readiness_query.go`.

**Implementação:**

1. Aplicar o predicado a ClaimWorkItem, ReclaimWorkItem e aquisições por ID e next de todos os protocolos.
2. Atualizar candidatos e available-work para excluir itens inelegíveis sem ocultar reason em consultas detalhadas.
3. Revalidar dentro do guard após seleção: candidato retornado previamente não garante direito de aquisição.
4. Não mudar dados de autoridade ou lifecycle ao negar; conservar receipt/status adequados.
5. Atualizar snapshots compactos que alimentam signed-next para carregar perfil e estado de Objective necessários.

**Testes/verificação:** Task sob Planned nunca é adquirida por claim, next, reclaim ou signed acquisition; corrida Start versus Acquire segue ordem de commit.

**Aceite:** Não há transporte ou protocolo de aquisição que contorne o gate.

**Commit sugerido:** `fix(coordination): E08 enforce parent gates on every acquisition`.

<a id="e09"></a>
### E09 — Fechar bypass de ativação e criação em To Do

**Objetivo:** Aplicar a mesma regra ao liberar Task existente e criá-la já disponível.

**Depende de:** E07, E08

**Arquivos:** [A] `packages/wos-core/application/service.go`; [A] `packages/wos-core/application/planning_creation.go`; [A] `packages/wos-core/domain/work_item.go`; [A] `packages/wos-core/application/workspace_capabilities.go`; [N] `packages/wos-core/application/task_activation_gate_test.go`.

**Implementação:**

1. Validar Objective/ancestrais e Outcome em ActivateWorkItem sob Outcome guard.
2. Criar Task em todo somente após validar a definição final com initial criteria e referências.
3. Preservar criação em Backlog sob Objective Planned e seus critérios iniciais atômicos.
4. No batch, validar estado final e negar qualquer item diretamente executável incompatível.
5. Capability de ActivateWorkItem deve indicar motivo e remedy, sem usar apenas o lifecycle local da Task.

**Testes/verificação:** Create backlog permitido; create todo/activate negados sob Planned, Outcome terminal/arquivado e parent incompatível; nenhum write parcial.

**Aceite:** O backend garante o mesmo contrato independentemente da opção avançada do formulário.

**Commit sugerido:** `fix(core): E09 guard activation and initial todo creation`.

<a id="e10"></a>
### E10 — Validar hierarquia e dependências contra esperas impossíveis

**Objetivo:** Evitar que o novo gate de ancestral produza trabalho impossível de iniciar ou concluir.

**Depende de:** E07, E09

**Arquivos:** [A] `packages/wos-core/application/dependencies.go`; [A] `packages/wos-core/application/work_editing.go`; [A] `packages/wos-core/application/workspace_placement.go`; [A] `packages/wos-core/domain/relation.go`; [N] `packages/wos-core/application/hierarchy_dependency_deadlock_test.go`.

**Implementação:**

1. Preservar o veto existente à dependência hard da Task no próprio Objective.
2. Estender verificação a ancestrais e a Objective que depende de descendente impedido de começar pelo mesmo ancestral.
3. Aplicar em AddDependency, batch final, mudança de Objective da Task e reparent de ramo.
4. Retornar caminho explicativo e alternativas sem criar/remover dependências automaticamente.
5. Respeitar same-scope, Backlog-only da origem Task e limites de leitura; incompletude não autoriza alteração.

**Testes/verificação:** Grafo depends_on acíclico mas impossível com hierarchy é negado; advisory não cria gate; batch final e concorrência A→B/B→A.

**Aceite:** Nenhuma alteração estrutural introduz deadlock semântico detectável por essas regras.

**Commit sugerido:** `fix(planning): E10 reject hierarchy dependency deadlocks`.

<a id="e11"></a>
### E11 — Fortalecer cancelamento e reabertura hierárquicos

**Objetivo:** Preservar a consistência ao sair de estados terminais ou encerrar parents.

**Depende de:** E09, E10

**Arquivos:** [A] `packages/wos-core/application/lifecycle_service.go`; [A] `packages/wos-core/application/administrative_work.go`; [A] `packages/wos-core/domain/objective.go`; [A] `packages/wos-core/domain/work_item.go`; [A] `packages/wos-core/application/work_contract_guards.go`; [N] `packages/wos-core/application/operational_reopen_test.go`.

**Implementação:**

1. ReopenObjective retorna Planned e avança completion_cycle; não auto-start nem auto-achieve.
2. ReopenWorkItem v2 exige Objective/ancestrais já reabertos e In Progress, com Outcome aberto.
3. Negar cancelamento de Objective que abandone execução/revisão ativa; orientar liberação/intervenção explícita.
4. Preservar ConclusionHistory e separar correção da execução de alteração da definição.
5. Garantir que release/revoke continuam possíveis mesmo quando avanço está bloqueado.

**Testes/verificação:** Reopen de Task sob parent Achieved/Planned negado; histórico preservado; lease pode ser liberado após incidente; cancelamento concorrente.

**Aceite:** Recuperação é possível e nenhum estado terminal é reaberto ou propagado silenciosamente.

**Commit sugerido:** `fix(lifecycle): E11 guard hierarchical reopen and cancellation`.

<a id="e12"></a>
### E12 — Alinhar capabilities e erros estruturados

**Objetivo:** Explicar corretamente a disponibilidade sem criar um segundo motor de regras.

**Depende de:** E07, E08, E09, E11

**Arquivos:** [A] `packages/wos-core/application/workspace_capabilities.go`; [A] `packages/wos-core/application/workspace_preconditions.go`; [A] `packages/wos-core/application/work_item_operational_query.go`; [A] `packages/wos-core/domain/errors.go`; [A] `packages/wos-core/application/effective_permissions.go`; [N] `packages/wos-core/application/operational_capability_parity_test.go`.

**Implementação:**

1. Gerar preconditions/capabilities com os mesmos predicados dos writes.
2. Adicionar reasons de parent Planned, ancestral, definição incompleta e protocolo; limitar related refs à leitura autorizada.
3. Distinguir denied, blocked, unavailable e external handoff conforme o contrato atual de capabilities.
4. Não montar permissão pelo client state nem transformar capability em token de autorização.
5. Validar frescor de policy/revisão e invalidar consultas após revogação.

**Testes/verificação:** Cada capability available tem write equivalente permitido no mesmo snapshot; mudança posterior é revalidada e pode negar.

**Aceite:** Kanban, API e CLI recebem motivos reproduzíveis, sem interpretar mensagens textuais.

**Commit sugerido:** `feat(queries): E12 expose consistent operational preconditions`.

<a id="e13"></a>
### E13 — Qualificar verticalmente os gates antes de ampliar UX

**Objetivo:** Demonstrar que os gates não existem somente no Core ou no browser.

**Depende de:** E08, E09, E11, E12

**Arquivos:** [A] `packages/wos-api/http/handler_test.go`; [A] `packages/wos-api/http/effective_permissions_test.go`; [A] `packages/wos-api/mcp/workspace.go`; [A] `packages/wos-cli/workspace_planning_test.go`; [N] `tests/acceptance/operational_lifecycle_parity.py`.

**Implementação:**

1. Montar cenários de chamada pela rota histórica, catálogo HTTP, MCP e cliente usado pelo CLI.
2. Verificar criação orphan, ativação sob Planned, aquisição inválida e reabertura terminal.
3. Comparar estados, eventos e versões após cada negativa.
4. Adicionar testes de identidade dinâmica que muda entre query e write.
5. Atualizar schemas já afetados e contratos generated no mesmo commit quando necessário.

**Testes/verificação:** HTTP/MCP/SDK/CLI falham de forma equivalente e sem mutação; fixtures usam a SHA corrente.

**Aceite:** Integridade de lifecycle está protegida antes de qualquer melhoria cosmética.

**Commit sugerido:** `test(transports): E13 prove operational gate parity`.

## Fase B — Autorização e credenciais

<a id="e14"></a>
### E14 — Substituir autorização por nome por catálogo explícito

**Objetivo:** Eliminar o fallback que atribui direitos a comandos por substring.

**Depende de:** E02, E03

**Arquivos:** [A] `packages/wos-core/application/authorization.go`; [N] `packages/wos-core/application/command_authorization_catalog.go`; [A] `packages/wos-core/ports/authorization.go`; [A] `packages/wos-api/commands/catalog.go`; [N] `packages/wos-core/application/command_authorization_catalog_test.go`.

**Implementação:**

1. Inventariar todos os comandos públicos e requisitos atuais antes de alterar comportamento.
2. Definir registro explícito com permissões simples/compostas e resolução de recurso/owner.
3. Rejeitar comando não registrado; não retornar PermissionOutcomeWrite como fallback.
4. Expor requisitos à apresentação e geradores sem copiar tabelas no frontend.
5. Manter compatibilidade controlada para nomes publicados e composições confiáveis.

**Testes/verificação:** Teste de totalidade compara catálogo real ao registro de autorização; nomes desconhecidos negados; permissões atuais cobertas.

**Aceite:** Adicionar novo comando sem requisito de autorização quebra o teste, em vez de receber poder implícito.

**Commit sugerido:** `refactor(auth): E14 use explicit command authorization registry`.

<a id="e15"></a>
### E15 — Introduzir permissões granulares e limites coerentes

**Objetivo:** Separar planejamento, execução, revisão, Issues e administração.

**Depende de:** E14

**Arquivos:** [A] `packages/wos-core/ports/authorization.go`; [A] `packages/wos-core/domain/signing_identity.go`; [A] `packages/wos-core/application/security_admin.go`; [A] `packages/wos-core/application/effective_permissions.go`; [A] `packages/wos-api/web/assets/actions.js`; [N] `packages/wos-core/application/granular_permissions_test.go`.

**Implementação:**

1. Adicionar catálogo da seção 7 e Valid/AllPermissions coerentes.
2. Atualizar limite de listas de permissões/policies para 128 explicitamente em todos os validadores e schemas relevantes.
3. Atribuir direitos específicos a claim, delivery, review, accept, issue resolve, blocker release e gestão de chaves.
4. Garantir que namespace admin não substitui provas ou autoridade assinada.
5. Preparar labels de UI geradas/centralizadas e docs, sem wildcard de executor.

**Testes/verificação:** Matriz allow/deny por operação e papel; valores duplicados/desconhecidos, listas vazias e limite excedido.

**Aceite:** Uma chave de execução não consegue aprovar, alterar definição ou administrar apenas por ter acesso ao trabalho.

**Commit sugerido:** `feat(auth): E15 add granular operational permissions`.

<a id="e16"></a>
### E16 — Aplicar escopo e requisitos compostos na transação

**Objetivo:** Evitar bypass por operação composta, scope parcial ou revalidação incompleta.

**Depende de:** E14, E15

**Arquivos:** [A] `packages/wos-core/application/pipeline.go`; [A] `packages/wos-core/application/authorization.go`; [A] `packages/wos-core/application/planning_creation.go`; [A] `packages/wos-core/application/documentary_compound.go`; [A] `packages/wos-core/application/issues_blockers_compound.go`; [A] `packages/wos-core/storage/sqlite/access.go`.

**Implementação:**

1. Resolver recurso real e conjunto de permissões antes de writes, dentro da segurança transacional.
2. Aplicar a cada source de dependência, owner de Criterion e alvo de operação composta.
3. Validar restrição de Outcome em consultas, previews e recibos, não somente na mutação.
4. Revalidar grant, CredentialPolicy e revogação sob a ordem de locks existente.
5. Não permitir actor/credential enviados no body definirem identidade.

**Testes/verificação:** Criar Task+Criterion sem criterion:define nega integralmente; Issue+Blocker sem blocker:create nega; cross-Outcome e owner forjado.

**Aceite:** Autorização corresponde aos efeitos reais e nenhum composto reduz o conjunto de exigências.

**Commit sugerido:** `fix(auth): E16 enforce scoped composite authorization atomically`.

<a id="e17"></a>
### E17 — Migrar direitos legados sem conceder poderes futuros

**Objetivo:** Preservar capacidade histórica com transformação finita e auditável.

**Depende de:** E15, E16

**Arquivos:** [N] `packages/wos-core/application/permission_compatibility.go`; [A] `packages/wos-core/storage/sqlite/security.go`; [A] `packages/wos-core/storage/sqlite/signing_identity.go`; [A] `packages/wos-core/storage/memory/security.go`; [N] `packages/wos-core/storage/sqlite/permission_migration_test.go`; [N] `docs/implementation/operational-consolidation/permission-map.md`.

**Implementação:**

1. Construir mapa versionado dos direitos antigos para capacidades existentes na SHA baseline.
2. Canonicalizar grant e credential restriction antes da interseção; não elevar uma chave limitada.
3. Exigir concessão explícita de emissão de chaves e delegação automática, sem expansão por novidade.
4. Produzir dry-run por Principal/credencial e aplicar sob versão de Namespace/policy.
5. Preservar recibos históricos e documentar o cutover de permissões.

**Testes/verificação:** Comparação de conjunto efetivo antes/depois, revogação durante migração, replay, empty permissions, novo comando não herdado.

**Aceite:** O mapa não usa prefixo/wildcard e é revisável por administradores.

**Commit sugerido:** `feat(auth): E17 add versioned legacy permission migration`.

<a id="e18"></a>
### E18 — Implementar presets de papéis versionados

**Objetivo:** Simplificar emissão sem transformar role em uma segunda autorização.

**Depende de:** E17

**Arquivos:** [N] `packages/wos-core/application/permission_templates.go`; [N] `packages/wos-core/ports/permission_templates.go`; [N] `packages/wos-core/application/permission_templates_test.go`; [A] `packages/wos-api/web/assets/en-US.js`.

**Implementação:**

1. Definir reader, planner, executor, reviewer, issue_operator e administrator com listas explícitas.
2. Materializar os direitos ao aplicar template e registrar ID/versão apenas como proveniência.
3. Mostrar preview de interseção com grants do Principal e restrições do Namespace.
4. Marcar lista editada como custom; não alterar chaves existentes ao mudar preset.
5. Documentar que perfis diferentes do mesmo Principal não asseguram revisão independente.

**Testes/verificação:** Snapshots dos presets, executor sem approve/admin, reviewer sem execução, template atualizado não altera políticas emitidas.

**Aceite:** Roles reduzem configuração manual sem criar herança invisível de privilégios.

**Commit sugerido:** `feat(auth): E18 add versioned permission templates`.

<a id="e19"></a>
### E19 — Emitir credencial restrita e metadados atomicamente

**Objetivo:** Eliminar a janela entre criar token e aplicar suas restrições.

**Depende de:** E16, E18

**Arquivos:** [A] `packages/wos-core/application/security.go`; [A] `packages/wos-core/application/security_admin.go`; [N] `packages/wos-core/application/credential_management.go`; [A] `packages/wos-core/ports/security.go`; [A] `packages/wos-core/ports/security_admin.go`; [A] `packages/wos-core/storage/sqlite/security_admin.go`; [N] `packages/wos-core/storage/sqlite/migrations/0038_credential_management.sql`.

**Implementação:**

1. Criar Credential, policy explícita, metadados, evento administrativo e recibo em uma transação.
2. Validar Principal/grants existentes, delegação de Actor e escopo; secret criptograficamente aleatório permanece hash-only.
3. Implementar nome/descrição/expiração e vínculo ao template com limites definidos.
4. Não devolver novamente segredo em replay; definir recuperação de resultado incerto.
5. Atualizar Memory, fonte SQLite, gerador PostgreSQL e endpoints existentes para usar a primitiva única.

**Testes/verificação:** Falha entre insert e policy faz rollback; token recém-emitido nunca nasce amplo; retries não revelam segredo; conflito de Namespace.

**Aceite:** Toda chave nova é restrita desde o primeiro instante em que pode autenticar.

**Commit sugerido:** `feat(security): E19 issue constrained credentials atomically`.

<a id="e20"></a>
### E20 — Implementar gestão, rotação e observação de chaves

**Objetivo:** Oferecer administração completa sem expor tokens nem transferir autoridade acidentalmente.

**Depende de:** E19

**Arquivos:** [A] `packages/wos-core/application/credential_management.go`; [A] `packages/wos-core/ports/security_admin.go`; [A] `packages/wos-core/storage/sqlite/security_admin.go`; [N] `packages/wos-core/application/credential_management_query.go`; [N] `packages/wos-core/storage/sqlite/credential_management_test.go`.

**Implementação:**

1. Listar e consultar credenciais com paginação, filtros e campos seguros; incluir política efetiva.
2. Atualizar nome, descrição, expiração e policy sob expected revisions.
3. Rotacionar por nova credencial e referência à antiga, com overlap explícito e sem transferência de lease.
4. Implementar last_used_at coalescido sem serializar toda requisição de domínio em uma escrita de telemetria.
5. Manter segredo ausente de list/status/audit e proteger criação de arquivos de exportação pelos clientes.

**Testes/verificação:** Paginação, filtros, acessos negados, versões stale, token ausente no JSON/log e rotação sem mudança de holder.

**Aceite:** Administrador consegue entender escopo e uso de cada chave sem acesso ao segredo persistido.

**Commit sugerido:** `feat(security): E20 add credential lifecycle management`.

<a id="e21"></a>
### E21 — Endurecer revogação e trilha administrativa

**Objetivo:** Dar efeito transacional à revogação e impedir bloqueio administrativo involuntário.

**Depende de:** E19, E20

**Arquivos:** [A] `packages/wos-core/application/security_admin.go`; [A] `packages/wos-core/application/credential_management.go`; [A] `packages/wos-core/storage/sqlite/access.go`; [A] `packages/wos-core/storage/sqlite/security_admin.go`; [A] `packages/wos-core/application/pipeline.go`; [N] `packages/wos-core/storage/sqlite/credential_revocation_race_test.go`.

**Implementação:**

1. Revalidar token/root/session e grant antes de mutation ou disclosure de receipt.
2. Registrar revogação, alteração de permissões e rotation com autor, credencial, versão e motivo, sem segredos.
3. Definir ordem concorrente entre revogação e operação: o commit anterior pode valer; operações posteriores não.
4. Impedir remoção da última via administrativa recuperável sem procedimento explícito.
5. Expor preview de autoridades afetadas; revogar chave não apaga contratos/leases nem certifica interrupção do agente.

**Testes/verificação:** Revogar enquanto renew/submit/replay aguarda lock; sessões filhas invalidadas; última administração; auditoria atômica.

**Aceite:** Nenhuma operação após revogação confirmada aproveita autorização em cache para escrever.

**Commit sugerido:** `fix(security): E21 enforce revocation and administrative audit`.

## Fase C — Reserva vinculada à credencial

<a id="e22"></a>
### E22 — Vincular o lease legado à credencial no domínio

**Objetivo:** Diferenciar duas chaves do mesmo Principal sem confundir lease com assignee.

**Depende de:** E07, E21

**Arquivos:** [A] `packages/wos-core/domain/work_item.go`; [A] `packages/wos-core/domain/work_item_lease.go`; [A] `packages/wos-core/domain/state_validation.go`; [N] `packages/wos-core/domain/credential_lease_test.go`.

**Implementação:**

1. Adicionar HolderCredentialID e proveniência de sessão à reserva, com versão de binding para dados antigos.
2. Criar operações de domínio v2 que exigem Principal, credencial, claim e fence; preservar leitura dos leases históricos.
3. Não aceitar holder escolhido no payload público; métodos recebem identidade confiável da Application.
4. Manter increments monotônicos de fencing e semântica da expiração sem alterar lifecycle pelo relógio.
5. Preservar signatures públicas antigas quando necessário por meio de wrappers de compatibilidade restritos, nunca como bypass para Task v2.

**Testes/verificação:** Mesma chave permite, outra chave do mesmo Principal nega, outro Principal nega, fence antigo nega; TTL nas fronteiras e overflow.

**Aceite:** Um lease v2 não é utilizável por outra credencial, mesmo que ela conheça claim_id e fencing_token.

**Commit sugerido:** `feat(domain): E22 bind legacy leases to credential identity`.

<a id="e23"></a>
### E23 — Persistir binding e preparar migração de reservas

**Objetivo:** Manter a identidade da aquisição após restart e upgrade.

**Depende de:** E22

**Arquivos:** [N] `packages/wos-core/storage/sqlite/migrations/0039_lease_credential_binding.sql`; [A] `packages/wos-core/storage/sqlite/repositories.go`; [A] `packages/wos-core/storage/sqlite/codec.go`; [A] `packages/wos-core/storage/memory/repositories.go`; [A] `tools/postgresgen/generate.py`; [N] `packages/wos-core/storage/sqlite/lease_credential_migration_test.go`.

**Implementação:**

1. Mapear onde o lease é materializado em colunas/JSON e expandir sem perder os campos existentes.
2. Persistir binding de credencial e sessão, bem como estado legível de proveniência ausente.
3. Reutilizar audit read-only de E06 para classificar leases legados; preencher somente com prova explícita.
4. Impedir ativação v2 com reserva incompatível viva, oferecendo drenar/expirar/intervir.
5. Testar backup/restore sem alterar credenciais históricas revogadas ainda referenciadas.

**Testes/verificação:** Fresh/upgrade/restart em SQLite e PG; lease órfão de credencial não é atribuído arbitrariamente; JSON histórico preservado.

**Aceite:** A migração não concede a uma chave atual a posse de trabalho antigo sem evidência.

**Commit sugerido:** `feat(storage): E23 persist credential-bound lease provenance`.

<a id="e24"></a>
### E24 — Resolver credencial canônica e aplicar em comandos

**Objetivo:** Aplicar binding a claim, renew, release, reclaim e conclusão legada.

**Depende de:** E22, E23

**Arquivos:** [N] `packages/wos-core/application/credential_authority.go`; [A] `packages/wos-core/application/service.go`; [A] `packages/wos-core/application/lease_service.go`; [A] `packages/wos-core/application/lifecycle_service.go`; [A] `packages/wos-core/application/authorization.go`; [A] `packages/wos-core/application/workspace_capabilities.go`.

**Implementação:**

1. Resolver Credential de API ou root de sessão validando parent, Namespace, Principal, Actor, expiração e revogação.
2. Carregar tempo autoritativo após os locks relevantes e usar a mesma identidade em todos os comandos protegidos.
3. Não ampliar policy da sessão ao resolver root; a credencial é chave de ownership, não atalho de permissão.
4. Projetar own/other/expired sem expor token ou dados administrativos desnecessários.
5. Tratar ausência de identidade confiável em host embedded v2 como erro explícito.

**Testes/verificação:** Duas sessões da mesma root, duas roots do mesmo Principal, parent revogado, scope diferente, renew no limite da expiração.

**Aceite:** Todos os comandos de reserva e capabilities reconhecem o mesmo titular canônico.

**Commit sugerido:** `fix(coordination): E24 enforce canonical credential ownership`.

<a id="e25"></a>
### E25 — Proteger recibos, recuperação e intervenção

**Objetivo:** Impedir roubo de autoridade via replay e recuperação sem fencing.

**Depende de:** E24

**Arquivos:** [A] `packages/wos-core/application/pipeline.go`; [A] `packages/wos-core/domain/idempotency.go`; [A] `packages/wos-core/application/administrative_work.go`; [A] `packages/wos-core/storage/sqlite/repositories.go`; [N] `packages/wos-core/application/credential_receipt_replay_test.go`; [N] `packages/wos-core/storage/sqlite/lease_recovery_race_test.go`.

**Implementação:**

1. Associar operações protegidas v2 à credencial no receipt ou binding aditivo e validar antes de revelar o resultado.
2. Preservar fingerprints históricos e semântica de idempotency keys; não fazer replay virar aquisição nova.
3. Reclaim após expiração cria claim novo e fence maior; nenhuma chave pode renew lease expirado.
4. Intervenção administrativa exige permissão, motivo e versões; revogar e adquirir são atos rastreáveis.
5. Documentar que alterar assignee não altera titular e que trocar token não transfere lease.

**Testes/verificação:** Repetição de claim por outra chave mesmo Principal, resposta perdida, crash pós-commit, duas recuperações simultâneas e revogação durante renew.

**Aceite:** Recuperação permanece exata e nenhuma credencial alternativa recebe autoridade por conhecer o recibo.

**Commit sugerido:** `fix(coordination): E25 fence recovery and credential-bound replays`.

<a id="e26"></a>
### E26 — Qualificar binding dos contratos existentes

**Objetivo:** Garantir que a melhoria de leases não degrada os protocolos de contrato.

**Depende de:** E24, E25

**Arquivos:** [A] `packages/wos-core/domain/signed_review.go`; [A] `packages/wos-core/application/signed_security.go`; [A] `packages/wos-core/application/signed_lease.go`; [A] `packages/wos-core/application/signed_review_lease.go`; [A] `packages/wos-core/application/work_contract_service.go`; [A] `packages/wos-cli/pending_lease_v2.go`; [N] `packages/wos-core/application/authority_protocol_parity_test.go`.

**Implementação:**

1. Verificar e preservar SignedContractBinding.CredentialID, signer key, policy revision e execução.
2. Adicionar binding autenticado equivalente ao contracts_v1 onde o contrato atual não distingue credenciais.
3. Manter proibição de browser session em signed authority e recuperação da credencial original.
4. Revalidar gates de Objective em renew/resume de avanço, preservando release/revoke para recuperação.
5. Atualizar formatos locais do CLI por migração explícita e sem reescrever intenções pendentes.

**Testes/verificação:** Mesmo Principal com outra credencial em renew/return/replay; signed key rotacionada; policy mais restritiva; nenhuma conversão implícita de protocolo.

**Aceite:** Há uma regra comum de exclusividade, implementada no mecanismo apropriado a cada protocolo.

**Commit sugerido:** `test(authority): E26 qualify credential binding across protocols`.

## Fase D — Entrega e revisão efetivas

<a id="e27"></a>
### E27 — Modelar entrega e decisão não assinada

**Objetivo:** Representar Verification real para trabalho não assinado sem falsificar contrato.

**Depende de:** E02, E09, E26

**Arquivos:** [N] `packages/wos-core/domain/task_delivery.go`; [N] `packages/wos-core/domain/task_delivery_decision.go`; [A] `packages/wos-core/domain/work_item.go`; [A] `packages/wos-core/domain/work_submission.go`; [N] `packages/wos-core/domain/task_delivery_test.go`.

**Implementação:**

1. Definir TaskDelivery e decisão imutável com authority_kind, material ou WorkSubmissionRef, rodada, snapshot e proveniência.
2. Validar exclusive binding: legacy claim verdadeiro ou contrato v1 verdadeiro; signed rejeitado nesta variante.
3. Adicionar PendingDeliveryID com exclusão mútua em relação ao pending review assinado.
4. Permitir in_progress legado sem lease somente quando há entrega pendente válida; não remover o invariant genericamente.
5. Definir métodos de aceite/correção que não exigem reutilizar o lease encerrado do executor.

**Testes/verificação:** Material inconsistente, vínculo falso, digest/definição divergentes, pending delivery e signed review simultâneos, decisão duplicada e estados inválidos.

**Aceite:** A fila não pode existir sem entrega identificável e nenhuma signed validation é relaxada.

**Commit sugerido:** `feat(domain): E27 model authenticated task deliveries and decisions`.

<a id="e28"></a>
### E28 — Persistir entregas, decisões e fila não assinada

**Objetivo:** Garantir uma entrega aberta por Task e decisões duráveis.

**Depende de:** E27

**Arquivos:** [N] `packages/wos-core/ports/task_delivery.go`; [N] `packages/wos-core/storage/sqlite/task_deliveries.go`; [N] `packages/wos-core/storage/memory/task_deliveries.go`; [N] `packages/wos-core/storage/sqlite/migrations/0040_task_deliveries.sql`; [A] `tools/postgresgen/generate.py`; [N] `packages/wos-core/storage/sqlite/task_deliveries_test.go`.

**Implementação:**

1. Criar ports focados para insert/get/list/save/decision e implementar rollback em Memory e SQL.
2. Aplicar FKs compostas e unicidade condicional de pending; vincular Task, entrega, material e decisão.
3. Persistir material legado uma vez; contracts_v1 referencia WorkSubmission sem duplicação.
4. Adicionar índices de fila por owner/scope/status e leituras paginadas coerentes.
5. Registrar novo arquivo fonte no gerador PG e adaptar constraints específicas por engine.

**Testes/verificação:** Roundtrip, restart, duas entregas abertas, cross-scope FK, decisão append-only, transação interrompida, migrations fresh/upgrade.

**Aceite:** Banco recusa vínculos inválidos e uma entrega não desaparece após restart.

**Commit sugerido:** `feat(storage): E28 persist authenticated verification state`.

<a id="e29"></a>
### E29 — Implementar submissão atômica para Verification

**Objetivo:** Encerrar execução e criar entrega/revisão numa única operação.

**Depende de:** E24, E28

**Arquivos:** [N] `packages/wos-core/application/task_delivery.go`; [A] `packages/wos-core/application/work_contract_service.go`; [A] `packages/wos-core/domain/work_contract.go`; [A] `packages/wos-core/application/pipeline.go`; [N] `packages/wos-core/application/submit_task_for_review_test.go`.

**Implementação:**

1. Implementar SubmitTaskForReview com expected version, material, idempotency e autoridade fornecida pelo fluxo autenticado.
2. Validar lease vivo/holder, perfil, definição, critérios, referências de material e ausência de revisão pendente.
3. Para contracts_v1, compor WorkSubmission e fechamento por entrega com primitivas privadas; não reutilizar Deliver assinado sem envelope.
4. Encerrar autoridade, vincular pending delivery e emitir eventos de forma atômica.
5. Atualizar receipt/result para distinguir entregue e aceito, com versões exatas.

**Testes/verificação:** Expiração enquanto espera lock, outra chave, material faltante, falha após gravar material, timeout/replay e contrato assinado recusado.

**Aceite:** Uma entrega confirmada deixa a Task em Verification e nenhuma autoridade de execução antiga continua válida.

**Commit sugerido:** `feat(work): E29 submit authenticated work for review atomically`.

<a id="e30"></a>
### E30 — Implementar aprovação e solicitação de correções

**Objetivo:** Vincular aceite à entrega atual e a uma decisão autorizada.

**Depende de:** E29, E16

**Arquivos:** [A] `packages/wos-core/application/task_delivery.go`; [A] `packages/wos-core/application/reviewer_policy.go`; [A] `packages/wos-core/domain/task_delivery_decision.go`; [A] `packages/wos-core/domain/work_item.go`; [N] `packages/wos-core/application/task_delivery_decision_test.go`.

**Implementação:**

1. Implementar AcceptTaskDelivery e RequestTaskDeliveryChanges com versões da Task e delivery, motivo e assessment references.
2. Aplicar independência quando a política exige, considerando todos os executores da rodada e correções relevantes.
3. Aprovação produz decisão, Conclusion e done na mesma transação; limpar pending pointer.
4. Correções encerram rodada e retornam todo sem lease, preservando findings e material anterior.
5. Reutilizar avaliações de prova e bloqueios; não usar administrative complete como atalho de aprovação.

**Testes/verificação:** Dois revisores aprovando/rejeitando simultaneamente, self-review proibida, prova de outra entrega, critérios pendentes e dupla decisão.

**Aceite:** Task só chega a Accepted com decisão persistida e prova atual, sem depender do frontend.

**Commit sugerido:** `feat(review): E30 decide authenticated task deliveries`.

<a id="e31"></a>
### E31 — Preservar rodadas de correção e replanning explícito

**Objetivo:** Permitir correções sem reusar prova antiga nem editar definição em execução.

**Depende de:** E30, E11

**Arquivos:** [A] `packages/wos-core/application/task_delivery.go`; [A] `packages/wos-core/domain/task_delivery.go`; [A] `packages/wos-core/application/work_editing.go`; [A] `packages/wos-core/application/lifecycle_service.go`; [N] `packages/wos-core/application/task_delivery_correction_test.go`.

**Implementação:**

1. Associar nova entrega à anterior por supersedes e round monotônico; manter cadeia imutável.
2. Reaquisição após changes_requested exige Task todo, parent ativo e nova autoridade.
3. Adicionar retirada autorizada de entrega quando necessário para replanning, com motivo e histórico.
4. Definição só muda após retorno explícito a Backlog e ausência de revisão/lease ativo.
5. Marcar assessments da rodada anterior como não aplicáveis à nova entrega, sem apagá-los.

**Testes/verificação:** Correção sem reacquire, reference à rodada errada, revisão antiga aprovada após substituição, alteração de Criterion durante review.

**Aceite:** A história explica cada entrega e nenhuma correção obtém aceitação herdada.

**Commit sugerido:** `feat(review): E31 preserve correction rounds and replanning boundaries`.

<a id="e32"></a>
### E32 — Integrar política de Verification ao fluxo assinado

**Objetivo:** Usar a revisão assinada real para Tasks operacionais v2.

**Depende de:** E27, E30, E31

**Arquivos:** [A] `packages/wos-core/application/signed_acquisition.go`; [A] `packages/wos-core/application/signed_security.go`; [A] `packages/wos-core/application/signed_return.go`; [A] `packages/wos-core/application/signed_review_return.go`; [A] `packages/wos-core/domain/signed_review.go`; [A] `packages/wos-cli/work_commands_v2.go`.

**Implementação:**

1. Congelar independent_review na aquisição signed de Task v2 que requer Verification.
2. Negar return direto/conclusão direta nesse contexto, sem retirar capacidade de contextos legados elegíveis.
3. Adicionar o gate de Objective e completion profile nos caminhos de aprovação e correção assinados.
4. Não inserir TaskDelivery nativa para signed; expor read model a partir de ReviewCase verdadeiro.
5. Preservar bytes, digests, signed receipts, chaves e histórico publicado.

**Testes/verificação:** Fixtures DSSE antigas continuam verificáveis; direct return v2 negado; delivery→review→approved real; outro Principal versus outra chave.

**Aceite:** O mesmo fluxo de negócio é cumprido sem produzir assinatura fictícia ou canal de revisão alternativo.

**Commit sugerido:** `feat(signed): E32 enforce operational verification policy`.

<a id="e33"></a>
### E33 — Fechar todos os atalhos de aceite e de assessment

**Objetivo:** Garantir critérios, rodada e revisão em todos os caminhos terminais.

**Depende de:** E09, E30, E32

**Arquivos:** [A] `packages/wos-core/application/service.go`; [A] `packages/wos-core/application/administrative_work.go`; [A] `packages/wos-core/application/work_contract_service.go`; [A] `packages/wos-core/application/workspace_proof.go`; [A] `packages/wos-core/domain/criterion.go`; [N] `packages/wos-core/application/task_acceptance_bypass_test.go`.

**Implementação:**

1. Exigir pelo menos um critério obrigatório na ativação v2 e assessments atuais na aceitação.
2. Vincular assessment à entrega/critério/ciclo aplicável, estendendo proveniência quando faltante.
3. Bloquear CompleteWorkItem, FinalizeWorkContract e administrative complete como bypass da revisão exigida.
4. Manter waiver separado de met e restringir exceção manual conforme o contrato.
5. Testar edição/retração de material e contestação sem reescrever conclusões históricas.

**Testes/verificação:** Nenhuma combinação de endpoint legado, permissão admin ou critério sem prova contorna v2; comportamento legado restrito permanece documentado.

**Aceite:** Accepted significa uma decisão verificável do backend, não resultado de uma rota permissiva.

**Commit sugerido:** `fix(acceptance): E33 close direct completion and proof bypasses`.

<a id="e34"></a>
### E34 — Unificar projeção de verificação e estágio

**Objetivo:** Exibir Verification apenas quando houver uma entrega/caso real pendente.

**Depende de:** E30, E32, E33

**Arquivos:** [N] `packages/wos-core/application/task_verification_query.go`; [A] `packages/wos-core/application/workspace_stage.go`; [A] `packages/wos-core/application/workspace_cards.go`; [A] `packages/wos-core/application/workspace_capabilities.go`; [A] `packages/wos-core/application/work_item_operational_query.go`; [N] `packages/wos-core/application/task_verification_query_test.go`.

**Implementação:**

1. Implementar TaskVerificationView como projeção adaptada da fonte nativa ou assinada.
2. Priorizar estado terminal verdadeiro; não colocar done em Verification por pointer histórico.
3. Distinguir correction pending todo, autoridade ativa in_progress, review pendente e lease expirado.
4. Retornar ações e handoff pelo protocolo verdadeiro, motivos e versões coerentes.
5. Mostrar cobertura parcial em material/criteria sem usar amostra como prova completa.

**Testes/verificação:** Paridade de cards, detail, ready-work e CLI para cada estado/protocolo; terminal com campos históricos e revisão paginada.

**Aceite:** O estágio é derivado da fonte certa e a UI não inventa submissões.

**Commit sugerido:** `feat(queries): E34 unify task verification projections`.

## Fase E — Conclusão de Objectives

<a id="e35"></a>
### E35 — Construir relatório canônico de elegibilidade de Objective

**Objetivo:** Separar fatos completos de elegibilidade da autorização de quem solicita.

**Depende de:** E11, E33, E34

**Arquivos:** [N] `packages/wos-core/application/objective_completion.go`; [N] `packages/wos-core/application/objective_completion_query.go`; [A] `packages/wos-core/application/completion_policy.go`; [A] `packages/wos-core/application/issue_completion.go`; [A] `packages/wos-core/application/workspace_preconditions.go`; [N] `packages/wos-core/application/objective_completion_report_test.go`.

**Implementação:**

1. Reutilizar scans completos de strict_work, proof validation, hard dependencies e Issues próprias.
2. Contabilizar todas as Tasks da subárvore, sem criar atributo Task.required.
3. Representar vazio, cancelamento, waiver, autoridade/revisão aberta e Criteria do Objective ainda não avaliados.
4. Produzir totais completos e samples limitados, com motivo de incompletude explícito.
5. Usar esse relatório no manual achieve e capabilities sem auto-approve de critérios.

**Testes/verificação:** Mais itens que uma página, Optional Objective no Outcome, subtree, zero Tasks, duplicate chains, canceladas e critérios só das Tasks.

**Aceite:** Eligible verdadeiro significa que todas as obrigações aplicáveis foram verificadas na mesma leitura coerente.

**Commit sugerido:** `feat(completion): E35 add canonical objective eligibility report`.

<a id="e36"></a>
### E36 — Persistir delegação e proveniência da conclusão automática

**Objetivo:** Autorizar a regra automática sem emprestar a identidade do último executor.

**Depende de:** E19, E35

**Arquivos:** [A] `packages/wos-core/domain/objective_completion.go`; [N] `packages/wos-core/ports/objective_completion.go`; [N] `packages/wos-core/storage/sqlite/objective_completion.go`; [N] `packages/wos-core/storage/memory/objective_completion.go`; [N] `packages/wos-core/storage/sqlite/migrations/0041_objective_completion_authority.sql`; [A] `packages/wos-core/application/operational_policy.go`.

**Implementação:**

1. Registrar modo, delegante, credencial, versão de política, escopo e ciclo de conclusão.
2. Criar identidade interna de automação controlada pelo host, sem token público e sem poder de avaliar critérios.
3. Implementar SetObjectiveCompletionMode com permissão específica, motivo e expected version.
4. Impedir automatic quando houver aprovação humana final exigida ou política strict_work ausente.
5. Preparar armazenamento de registros de reconciliação determinística e auditoria de efeitos.

**Testes/verificação:** Executor não delega autoridade por concluir Task; modo automatic sem permissão nega; política revogada ou incompatível não fecha Objective.

**Aceite:** Toda conclusão automática tem uma delegação verificável distinta da autoria da prova.

**Commit sugerido:** `feat(completion): E36 persist constrained automatic completion authority`.

<a id="e37"></a>
### E37 — Adicionar autofechamento atômico ao pipeline

**Objetivo:** Concluir Objectives elegíveis como efeito de mutações legítimas, sem comando recursivo.

**Depende de:** E35, E36

**Arquivos:** [N] `packages/wos-core/application/objective_completion_effects.go`; [A] `packages/wos-core/application/pipeline.go`; [A] `packages/wos-core/application/service.go`; [A] `packages/wos-core/application/signed_review_return.go`; [A] `packages/wos-core/application/task_delivery.go`; [A] `packages/wos-core/domain/event.go`; [N] `packages/wos-core/application/objective_auto_achievement_test.go`.

**Implementação:**

1. Extrair primitiva privada comum de achievement; manual e automático a reutilizam.
2. Adicionar hook allowlisted após mutação e antes de evento/receipt, usando a mesma UoW e Outcome guard.
3. Buscar candidatos e iterar em ordem determinística, respeitando limites e sem usar contadores de UI.
4. Avançar OutcomeRevision uma vez no composto; incluir metadados dos agregados derivados e eventos com causation.
5. Precondição não satisfeita é normal; falha de persistência é rollback, nunca sucesso parcial silencioso.

**Testes/verificação:** Última Task, último Criterion do Objective, Issue resolvida, duas últimas Tasks concorrentes, resposta perdida e evento/outbox duplicado.

**Aceite:** Um Objective elegível fecha uma vez; o Outcome não fecha; nenhum Assessment é fabricado.

**Commit sugerido:** `feat(completion): E37 derive objective achievement atomically`.

<a id="e38"></a>
### E38 — Tratar reabertura, ciclos e invalidação de elegibilidade

**Objetivo:** Evitar autofechamento imediato com provas antigas após reabertura.

**Depende de:** E11, E35, E37

**Arquivos:** [A] `packages/wos-core/domain/objective.go`; [A] `packages/wos-core/domain/criterion.go`; [A] `packages/wos-core/application/service.go`; [A] `packages/wos-core/application/workspace_proof.go`; [A] `packages/wos-core/application/objective_completion.go`; [A] `packages/wos-core/domain/conclusion_contestation.go`; [N] `packages/wos-core/application/completion_cycle_test.go`.

**Implementação:**

1. Vincular aplicabilidade de assessments ao completion_cycle atual do Objective.
2. Reopen avança ciclo e preserva histórico; start não reafirma avaliações antigas.
3. Atualizar preview de mudanças em Criterion, prova, dependências e ownership que afetem conclusões.
4. Usar contestação/histórico existente para evidência contraditória, sem reescrever resultado passado.
5. Excluir comandos de recuperação que não mudam elegibilidade da execução indevida do hook.

**Testes/verificação:** Reopen→start não refecha sozinho; assessment novo explícito habilita; prova antiga, waiver e material retraído bloqueiam; histórico estável.

**Aceite:** A automação não entra em ciclo de reabrir/fechar e não trata conclusão histórica como prova atual.

**Commit sugerido:** `fix(completion): E38 bind achievement to current evaluation cycle`.

<a id="e39"></a>
### E39 — Implementar reconciliação durável e impacto indireto

**Objetivo:** Não perder reavaliação por dependências indiretas, duplicatas, restart ou limite de cascata.

**Depende de:** E37, E38

**Arquivos:** [A] `packages/wos-core/application/objective_completion_effects.go`; [A] `packages/wos-core/storage/sqlite/objective_completion.go`; [A] `packages/wos-core/ports/objective_completion.go`; [A] `packages/wos-core/application/dependencies.go`; [A] `packages/wos-core/application/issues_blockers.go`; [N] `packages/wos-core/application/objective_completion_reconcile.go`; [N] `packages/wos-core/application/objective_completion_reconcile_test.go`.

**Implementação:**

1. Construir conjunto de afetados por owners/ancestors, dependentes hard, Blockers liberados e duplicatas canônicas.
2. Adotar orçamento inicial explícito de 64 conclusões derivadas por transação; confirmar custo nos benchmarks, sem confundir com limite de Tasks lidas para o gate.
3. Se exceder, persistir pedido coalescido de reconciliação por Outcome/revisão, na mesma transação, com notificação no mecanismo de eventos/outbox existente.
4. Implementar consumidor interno de reconciliação determinística no processo de serviço e operação run-once para host embedded; não executar agentes nem introduzir broker.
5. Cada iteração recarrega fatos atuais e respeita política/credenciais administrativas; retry interno não altera a intenção original de um cliente.
6. Expor pending/last_processed_revision e comando de recomputação auditado para recuperação; antes de habilitar automatic, composição deve demonstrar consumidor ou capacidade síncrona completa.

**Testes/verificação:** Cascata >64, evento duplicado, restart, policy desabilitada enquanto pendente, canônica resolvida de outra Issue, criação concorrente e replay.

**Aceite:** Não há conclusão baseada em leitura truncada nem pendência perdida; UI distingue elegível/reconciliando de Achieved.

**Commit sugerido:** `feat(completion): E39 reconcile indirect and deferred completion effects`.

<a id="e40"></a>
### E40 — Expor progresso e closure report autoritativos

**Objetivo:** Fornecer contagens úteis sem confundir amostra com universo.

**Depende de:** E34, E35, E39

**Arquivos:** [A] `packages/wos-core/application/workspace_cards.go`; [A] `packages/wos-core/application/workspace_read_types.go`; [A] `packages/wos-core/application/workspace_frame.go`; [A] `packages/wos-core/storage/sqlite/workspace_read.go`; [A] `packages/wos-core/application/workspace_conclusion.go`; [N] `packages/wos-core/application/operational_progress_test.go`.

**Implementação:**

1. Adicionar progresso por estado, Tasks aceitas/canceladas, critérios pendentes e Issues próprias.
2. Preservar snapshots/revisões e usar consultas agregadas completas sob o read frame existente.
3. Separar manual_allowed, automatic_allowed, eligible e reconciliation_pending.
4. Retornar resumo legível de causa de bloqueio e referências para drill-down autorizado.
5. Não inventar porcentagem quando o denominador é desconhecido ou quando o conjunto é vazio.

**Testes/verificação:** Dados acima do limite da página, counters com filtros, parent/child, snapshot stale, cache após revogação e zero denominator.

**Aceite:** Cards e CLI podem explicar progresso real sem inferir conclusão por contagem carregada.

**Commit sugerido:** `feat(queries): E40 expose authoritative operational progress`.

## Fase F — Issues e dependências

<a id="e41"></a>
### E41 — Concluir paridade de ownership de Issues

**Objetivo:** Fechar o gap entre Objective obrigatório no domínio e formulários/DTOs antigos.

**Depende de:** E01, E16, E35

**Arquivos:** [A] `packages/wos-core/application/issues_blockers.go`; [A] `packages/wos-core/application/issues_blockers_compound.go`; [A] `packages/wos-api/http/issues_blockers.go`; [A] `packages/wos-api/http/issues_blockers_dto.go`; [A] `packages/wos-api/commands/contracts.go`; [A] `packages/wos-api/web/assets/human-forms.js`.

**Implementação:**

1. Confirmar implementação existente das etapas anteriores; reutilizar validators e migrações 0034/0036.
2. Propagar objective_id obrigatório em CreateIssue e comandos compostos por todos os contratos.
3. No formulário de Task/Objective, obter owner canônico; no Outcome, exigir seleção explícita.
4. Manter affected_refs independentes e múltiplos, sem owner inferido de proximidade ou título.
5. Atualizar erros de owner ausente e exemplos públicos.

**Testes/verificação:** Criar por Task, Objective, Outcome e material; owner de outro scope; antigo payload sem owner; backend e UI não discordam.

**Aceite:** Toda Issue nova tem dono válido e cada fluxo oferecido consegue fornecer esse dono.

**Commit sugerido:** `fix(issues): E41 complete objective ownership across entrypoints`.

<a id="e42"></a>
### E42 — Qualificar responsabilidade e investigação concorrente de Issues

**Objetivo:** Separar reportador, owner, assignee e autoridade de resolução.

**Depende de:** E15, E41

**Arquivos:** [A] `packages/wos-core/application/issue_assignment.go`; [A] `packages/wos-core/domain/issue.go`; [A] `packages/wos-core/application/issues_blockers.go`; [N] `packages/wos-core/application/issue_assignment_permission_test.go`; [A] `packages/wos-core/storage/sqlite/issues_blockers.go`.

**Implementação:**

1. Reutilizar AssignIssue, UnassignIssue e investigação com assignee existentes, corrigindo gaps de transporte quando encontrados.
2. Aplicar issue:assign, issue:investigate e issue:resolve separadamente.
3. Assumir investigação para identidade autenticada usa compare-and-swap da versão; reatribuir terceiro exige autorização.
4. Preservar eventos de assignment e motivo na liberação de responsabilidade de Issue em investigação.
5. Não criar lease de Task por assumir Issue e não dar autoridade sobre Tasks corretivas.

**Testes/verificação:** Dois operadores assumindo mesma versão, outro Principal resolvendo sem permissão, reportador ≠ assignee e assignment não altera Task.

**Aceite:** Há um assignee explícito por Issue, com concorrência e histórico consistentes.

**Commit sugerido:** `feat(issues): E42 enforce granular assignment and investigation`.

<a id="e43"></a>
### E43 — Consolidar o gate de Issues próprias e duplicatas

**Objetivo:** Reutilizar o gate implementado e incluí-lo no fechamento automático.

**Depende de:** E35, E41, E42

**Arquivos:** [A] `packages/wos-core/application/issue_completion.go`; [A] `packages/wos-core/application/issues_blockers.go`; [A] `packages/wos-core/application/workspace_preconditions.go`; [A] `packages/wos-core/application/objective_completion_effects.go`; [A] `packages/wos-core/application/issue_reopen_duplicate_policy_test.go`.

**Implementação:**

1. Verificar relatório completo de Issues próprias e cadeias duplicate, sem apenas contar Issues afetando o Objective.
2. Reutilizar proteção de reabertura quando canônica/duplicatas afetarem Objectives Achieved.
3. Notificar candidatos de completion após fechamento da canônica ou mudança de condição relevante.
4. Preservar distinção wont_fix ≠ critério satisfeito e Issue ≠ Blocker.
5. Retornar related refs com coverage clara, mesmo com milhares de duplicatas.

**Testes/verificação:** Duplicate→open bloqueia; duplicate→resolved libera esse gate; ciclo/cadeia inválida nega; owner distinto e reopen de canônica.

**Aceite:** Nem fechamento manual nem automático ignoram uma Issue pendente por paginação, duplicata ou ownership equivocado.

**Commit sugerido:** `fix(completion): E43 enforce owned issue closure in all paths`.

<a id="e44"></a>
### E44 — Concluir operações compostas de Issues e Blockers

**Objetivo:** Entregar criação/resolução com impactos múltiplos sem efeitos implícitos.

**Depende de:** E16, E41, E43

**Arquivos:** [A] `packages/wos-core/application/issues_blockers_compound.go`; [A] `packages/wos-core/application/workspace_blocker_preview.go`; [A] `packages/wos-api/http/issues_blockers.go`; [A] `packages/wos-api/mcp/workspace.go`; [N] `packages/wos-core/application/issue_blocker_permission_parity_test.go`.

**Implementação:**

1. Reutilizar ReportIssueWithBlockers existente, limite de 64 e validação de alvo/propagação.
2. Aplicar permissões de Issue e de cada Blocker em uma transação, com revisão observada do Outcome.
3. Na resolução composta, liberar apenas o conjunto explicitamente escolhido e validado.
4. Reavaliar completion de alvos afetados depois da liberação legítima; Issue resolvida sozinha não libera nada.
5. Atualizar DTOs, catálogo e exemplos para não manter somente a variante antiga de um Blocker.

**Testes/verificação:** Um alvo inválido desfaz todos, duplicatas de impacto, cross-scope, permissão parcial e resposta perdida.

**Aceite:** Cada impedimento criado/liberado corresponde a uma intenção explícita rastreável.

**Commit sugerido:** `feat(issues): E44 complete multi-blocker compound operations`.

<a id="e45"></a>
### E45 — Integrar relações corretivas Issue–Task

**Objetivo:** Exibir trabalho corretivo sem confundir vínculo documental com dependência.

**Depende de:** E41, E42, E44

**Arquivos:** [A] `packages/wos-core/application/issue_work_relation.go`; [A] `packages/wos-core/application/entity_workspace.go`; [A] `packages/wos-core/application/workspace_records.go`; [A] `packages/wos-api/web/assets/workspace-entity-context.js`; [N] `packages/wos-core/application/issue_work_relation_permission_test.go`.

**Implementação:**

1. Reutilizar relates_to e a operação tipada já criada, com versões de Issue/Task.
2. Mostrar relação em ambos os inspectors, mantendo ownership de cada entidade.
3. Não bloquear readiness apenas por existir vínculo corretivo; impedimento deve ser Blocker ou hard dependency explícito.
4. Criar Task corretiva somente sob Objective escolhido/válido; oferecer herdar o owner da Issue de modo visível.
5. Expor a relação ao CLI/SDK sem um novo campo paralelo de IDs desconectado da Relation canônica.

**Testes/verificação:** Vínculo/removal atômico, versões stale, múltiplas Tasks, scopes inválidos e readiness antes/depois idêntica.

**Aceite:** Operador acompanha a solução de uma Issue sem criar hierarquia ou causalidade falsas.

**Commit sugerido:** `feat(issues): E45 expose corrective task relationships`.

<a id="e46"></a>
### E46 — Concluir preview de dependências e validação do grafo

**Objetivo:** Usar a implementação existente como base única da edição de relações.

**Depende de:** E10, E16, E43

**Arquivos:** [A] `packages/wos-core/application/dependencies.go`; [A] `packages/wos-core/application/dependency_impact.go`; [A] `packages/wos-core/application/workspace_placement.go`; [A] `packages/wos-core/domain/relation.go`; [N] `packages/wos-core/application/dependency_final_graph_policy_test.go`.

**Implementação:**

1. Revisar preview/apply já presentes e preservar validação do grafo final.
2. Aplicar as restrições hierarchy/lifecycle adicionadas em E10 também às operações em lote.
3. Retornar source real, target, strength, efeito atual/potencial e motivo de bloqueio em preview.
4. Garantir que preview não persiste e não atua como autorização reutilizável.
5. Conservar hard/advisory e cancelamento não satisfatório sem regras inventadas no frontend.

**Testes/verificação:** Troca de edge com ciclo temporário mas grafo final válido; ciclo final inválido; Tasks frozen; reparent e hierarquia combinada.

**Aceite:** Todos os editores podem reutilizar uma única avaliação transacional do grafo.

**Commit sugerido:** `feat(dependencies): E46 complete final graph operational previews`.

<a id="e47"></a>
### E47 — Proteger edição bidirecional e batch de dependências

**Objetivo:** Garantir que editar uma dependência exige direitos sobre a origem verdadeira.

**Depende de:** E46

**Arquivos:** [A] `packages/wos-core/application/dependencies.go`; [A] `packages/wos-core/application/authorization.go`; [A] `packages/wos-core/application/work_contract_guards.go`; [N] `packages/wos-core/application/dependency_source_authorization_test.go`; [N] `packages/wos-core/storage/sqlite/dependency_batch_concurrency_test.go`.

**Implementação:**

1. Resolver requisitos por cada source em ExpectedSources, inclusive quando a UI abriu o target.
2. Versionar remoções e conferir expected_outcome_revision; rejeitar duplicatas no request.
3. Não permitir que permissão sobre target remova relação de uma Task cuja definição está congelada.
4. Emitir um evento por Relation alterada com uma única OutcomeRevision no batch.
5. Impedir race de opposite-edge insertion em processos distintos por guard existente.

**Testes/verificação:** Edits pela seção Required by, sources com permissões diferentes, conflito parcial rollback, dois batches concorrentes e replay.

**Aceite:** Direção da UI não muda quem é autorizado a editar a Relation.

**Commit sugerido:** `fix(dependencies): E47 enforce source authority for batch edits`.

<a id="e48"></a>
### E48 — Expor impacto e satisfação de dependências com cobertura

**Objetivo:** Fornecer o contexto necessário para UX de dependências e bloqueio indireto.

**Depende de:** E40, E46, E47

**Arquivos:** [A] `packages/wos-core/application/dependency_impact.go`; [A] `packages/wos-core/application/workspace_frame.go`; [A] `packages/wos-core/ports/authorization.go`; [A] `packages/wos-core/storage/sqlite/graph_read.go`; [A] `packages/wos-core/storage/sqlite/workspace_read.go`; [N] `packages/wos-core/application/dependency_impact_scope_test.go`.

**Implementação:**

1. Reutilizar upstream/downstream, paths e limites existentes de depth/nodes/edges.
2. Acrescentar flags de satisfação e indicação de qual gate é afetado, sem duplicar Blockers transitivos.
3. Garantir paginação e cursores revision-pinned; mudança invalida a continuidade anterior.
4. Separar contagem completa de samples e filtrar dados não autorizados antes de exposição.
5. Validar efeitos de Objective que alcançou Achieved automaticamente e acorda dependentes.

**Testes/verificação:** A→B→C, advisory, Blocker direto/subtree, partial coverage, stale cursor e Outcome não autorizado.

**Aceite:** Usuário compreende por que aguarda e quais outros Objectives serão afetados sem receber totais enganosos.

**Commit sugerido:** `feat(queries): E48 expose dependency satisfaction and impact`.

## Fase G — Paridade de interfaces

<a id="e49"></a>
### E49 — Consolidar catálogo público e schemas gerados

**Objetivo:** Qualificar a superfície pública final sem drift de DTO ou permissões.

**Depende de:** E12, E21, E26, E34, E39, E44, E47, E48

**Arquivos:** [A] `packages/wos-api/commands/catalog.go`; [A] `packages/wos-api/commands/contracts.go`; [A] `packages/wos-api/commands/workspace_schema.go`; [A] `packages/wos-api/commands/workspace_results.go`; [G] `packages/wos-api/commands/catalog_generated.go`; [G] `packages/wos-api/commands.openapi.json`; [A] `tools/openapigen/main.go`; [A] `tools/mcpgen/main.go`.

**Implementação:**

1. Inventariar comandos/queries adicionados nas etapas anteriores e verificar nomes, versões e descoberta.
2. Registrar requirements completos de permissão e hints de transporte, sem incluir segredo no catálogo.
3. Adicionar schemas de perfil, completion, TaskDelivery, credential management e reasons novos.
4. Usar strings decimais exatas nos contratos novos; preservar tipos históricos onde a compatibilidade exige.
5. Regenerar API/MCP/SDK e corrigir as fontes, não editar derivados manualmente.

**Testes/verificação:** Schema validation, required versus nullable, additionalProperties, limites, strings uint64 e catálogo sem comando descoberto sem autorização.

**Aceite:** Contratos gerados refletem exatamente o backend e descrevem o que foi implementado, não features futuras.

**Commit sugerido:** `feat(api): E49 consolidate operational command contracts`.

<a id="e50"></a>
### E50 — Completar rotas HTTP e administração compatível

**Objetivo:** Expor as novas operações sem criar bypass nas rotas antigas.

**Depende de:** E49

**Arquivos:** [A] `packages/wos-api/http/handler.go`; [A] `packages/wos-api/http/commands.go`; [A] `packages/wos-api/http/security.go`; [A] `packages/wos-api/http/work_items.go`; [A] `packages/wos-api/http/objectives.go`; [N] `packages/wos-api/http/task_deliveries.go`; [N] `packages/wos-api/http/objective_completion.go`; [N] `packages/wos-api/http/credential_management.go`.

**Implementação:**

1. Usar Application única tanto no catálogo quanto nas rotas REST legadas.
2. Adicionar consultas de completion/verification/task-deliveries e gestão de credenciais sob prefix configurável.
3. Preservar autenticação, CSRF, body limit, timeouts, idempotency e recibos.
4. Retornar erro equivalente para stale version/lease/permits, sem revelar recursos de outro scope.
5. Desambiguar task-deliveries de deliveries da integração; proteger no-store nas respostas sensíveis.

**Testes/verificação:** Rota histórica versus /commands, query scope, CSRF browser, headers ausentes, payload excedido e endpoints administrativos.

**Aceite:** Todas as rotas públicas aplicam o mesmo contrato de domínio e identidade.

**Commit sugerido:** `feat(http): E50 expose operational and credential workflows`.

<a id="e51"></a>
### E51 — Completar tools e resources MCP

**Objetivo:** Disponibilizar operação equivalente a agentes sem autoridade oculta de sessão MCP.

**Depende de:** E49, E50

**Arquivos:** [A] `packages/wos-api/mcp/workspace.go`; [A] `packages/wos-api/mcp/queries.go`; [A] `packages/wos-api/mcp/resources.go`; [A] `packages/wos-api/mcp/security.go`; [G] `packages/wos-api/mcp/commands_generated.go`; [N] `packages/wos-api/mcp/operational_parity_test.go`.

**Implementação:**

1. Regenerar comandos e adicionar reads de completion, verificação, credenciais e impacto.
2. Derivar identidade da conexão autenticada conforme adapter existente; não aceitar ActorRef como autoridade.
3. Manter escopo, limites, reasons e receipts da Application.
4. Não usar o estado da sessão MCP como lock/lease ou como armazenamento de negócio.
5. Recusar comandos nativos sobre Task assinada e indicar host apropriado.

**Testes/verificação:** Matriz equivalente HTTP/MCP, resource discovery autorizado, sessão expirada, key distinta mesmo Principal e replay.

**Aceite:** Agente via MCP não consegue fazer o que a API recusaria.

**Commit sugerido:** `feat(mcp): E51 expose operational workflow parity`.

<a id="e52"></a>
### E52 — Atualizar SDK Go e exemplos tipados

**Objetivo:** Oferecer contratos completos sem incentivar bypass por requests manuais.

**Depende de:** E49, E50, E51

**Arquivos:** [A] `packages/wos-sdk-go/client.go`; [A] `packages/wos-sdk-go/workspace.go`; [G] `packages/wos-sdk-go/commands_generated.go`; [N] `packages/wos-sdk-go/operational_queries.go`; [N] `packages/wos-sdk-go/operational_queries_test.go`; [A] `examples/remote-client/main.go`.

**Implementação:**

1. Gerar métodos de comandos e implementar queries adicionais com tipos seguros.
2. Preservar receipt/replay, expected versions, contexto cancelável e erros estruturados.
3. Não converter fence/revision uint64 em float64 nem auto-retry com versão nova.
4. Adicionar exemplo de criar Objective→Task, consultar gate, entregar e acompanhar revisão.
5. Documentar uso embedded e remoto com a mesma Application, distinguindo autenticação de domínio.

**Testes/verificação:** Roundtrip HTTP real, leitura de JSON de versões grandes, timeouts, erros e zero secret em logs.

**Aceite:** SDK executa as jornadas completas e não esconde conflicts ou policy requirements.

**Commit sugerido:** `feat(sdk): E52 add typed operational workflow support`.

<a id="e53"></a>
### E53 — Adicionar CLI administrativo de chaves e políticas

**Objetivo:** Permitir administração reproduzível sem expor segredos no terminal ou código.

**Depende de:** E20, E21, E52

**Arquivos:** [N] `packages/wos-cli/credential_commands_v2.go`; [A] `packages/wos-cli/workspace_commands_v2.go`; [A] `packages/wos-cli/help.go`; [A] `packages/wos-cli/secrets.go`; [A] `packages/wos-cli/profiles.go`; [N] `packages/wos-cli/credential_commands_v2_test.go`.

**Implementação:**

1. Implementar auth keys list/create/show/permissions/update/revoke/rotate com perfis existentes.
2. Ler intenção de arquivo/STDIN e escrever segredo uma vez em arquivo exclusivo protegido ou fluxo explicitamente confirmado.
3. Registrar versão/policy e receipt sem token; não gravar segredo em YAML versionado.
4. Adicionar help/catalog local sem acessar credenciais para --help.
5. Tratar emissão confirmada com falha local de salvar segredo como resultado recuperável por revogação/substituição, não como emissão inexistente.

**Testes/verificação:** Keyring/arquivo nativo, permissões Unix/Windows, stdout redigido, erro de disco, emissão repetida, perfil errado e privilege denial.

**Aceite:** Operador emite e revoga chaves granularmente sem vazar seus valores.

**Commit sugerido:** `feat(cli): E53 add safe credential administration commands`.

<a id="e54"></a>
### E54 — Adicionar CLI de Objective, Issue e dependências

**Objetivo:** Concluir a operação das entidades no CLI schema-2, sem depender do browser.

**Depende de:** E41, E42, E47, E48, E52

**Arquivos:** [N] `packages/wos-cli/objective_commands_v2.go`; [N] `packages/wos-cli/issue_commands_v2.go`; [N] `packages/wos-cli/dependency_commands_v2.go`; [A] `packages/wos-cli/workspace_planning.go`; [A] `packages/wos-cli/workspace_inspect.go`; [A] `packages/wos-cli/help.go`.

**Implementação:**

1. Adicionar wrappers tipados para start/closure report/completion mode e operações de Issue com owner/assignee.
2. Expor dependency preview/apply/impact e direção clara.
3. Reutilizar planning intent journal, perfil, scope e revisão observada.
4. Não auto-start Objective para fazer task activate funcionar; exibir remedy.
5. Manter JSON estável e códigos de saída consistentes com o CLI existente; documentar comandos novos como novos.

**Testes/verificação:** Jornada completa por CLI, stale scopes, batch source inválida, owner obrigatório, read-only não escreve e replay de planning.

**Aceite:** Agentes administram os fluxos novos usando contratos oficiais, não editando arquivos internos de estado.

**Commit sugerido:** `feat(cli): E54 add objective issue and dependency operations`.

<a id="e55"></a>
### E55 — Integrar entrega/revisão e recuperação no CLI

**Objetivo:** Garantir que work finish e review operem o protocolo correto e preservem a intenção.

**Depende de:** E26, E31, E32, E34, E52, E54

**Arquivos:** [N] `packages/wos-cli/task_delivery_commands_v2.go`; [A] `packages/wos-cli/work_commands_v2.go`; [A] `packages/wos-cli/review_commands_v2.go`; [A] `packages/wos-cli/pending_acquisition_v2.go`; [A] `packages/wos-cli/pending_lease_v2.go`; [A] `packages/wos-cli/return_send_v2.go`; [A] `packages/wos-cli/help.go`.

**Implementação:**

1. Roteamento declarado de nativo/contracts_v1/signed-v2 a partir de capabilities e autoridade existente.
2. Manter sintaxe publicada de work/review assinados e acrescentar suporte nativo sem fallback quando assinatura falta.
3. Exibir disposition: delivered_for_review, changes_requested ou accepted, e Objectives derivados separadamente.
4. Preservar a credencial/execução original em pending intents; key rotation não muda sua identidade.
5. Manter keepalive foreground existente e parada limpa; não iniciar processo oculto de agente ou renovar após revogação.

**Testes/verificação:** Crash antes/depois do commit, request incerta, outro perfil, rotação, fence stale, workspace recuperado e signed binding original.

**Aceite:** CLI nunca reporta Accepted por ter apenas entregado, nem troca autoridade para resolver erro automaticamente.

**Commit sugerido:** `feat(cli): E55 integrate protocol-aware delivery and review recovery`.

## Fase H — Criação contextual

<a id="e56"></a>
### E56 — Unificar todas as ações de criação contextual

**Objetivo:** Eliminar Add ambíguo e caminhos de Task diretamente sob Outcome.

**Depende de:** E12, E41, E50

**Arquivos:** [N] `packages/wos-api/web/assets/creation-routes.js`; [A] `packages/wos-api/web/assets/actions.js`; [A] `packages/wos-api/web/assets/app.js`; [A] `packages/wos-api/web/assets/workspace-kanban.js`; [A] `packages/wos-api/web/assets/workspace-action-menu.js`; [N] `tests/web/unit/creation-routes.test.mjs`.

**Implementação:**

1. Definir roteamento puro por contexto/coleção/seleção e combinar com capability do servidor.
2. Corrigir canvas summary, canvasCreate, add-item, menu, empty states, atalhos e lista global.
3. Outcome oferece Add Objective; Objective oferece Add Task; lista global exige escolher owner.
4. Remover fallback que usa a última seleção global para criar na entidade errada.
5. Não retirar endpoint escopado em Outcome nem inferir parent silenciosamente; ajustar onboarding só no vínculo obrigatório.

**Testes/verificação:** Matriz de todos os entrypoints com cliques e keyboard, troca de seleção, permissões carregando e estado vazio.

**Aceite:** Nenhuma ação contextual sugere propriedade inválida; Task criada sempre aponta ao Objective visível.

**Commit sugerido:** `fix(web): E56 centralize contextual entity creation`.

<a id="e57"></a>
### E57 — Reorganizar criação e preparação de Objective

**Objetivo:** Tornar claro o que é condição intermediária e o que falta para iniciá-la.

**Depende de:** E35, E36, E56

**Arquivos:** [A] `packages/wos-api/web/assets/human-forms.js`; [A] `packages/wos-api/web/assets/workspace-form-fields.js`; [A] `packages/wos-api/web/assets/en-US.js`; [N] `packages/wos-api/web/assets/objective-completion-view.js`; [N] `tests/web/objective-creation-flow.spec.mjs`.

**Implementação:**

1. Aplicar ordem de campos definida na seção 10, com critérios visíveis e advanced progressivo.
2. Exibir completion mode/default/delegação, required_for_outcome e parent sem confundir seus significados.
3. Permitir salvar Planned incompleto, mas mostrar requisitos reais de start.
4. Não expor manualmente IDs ou expected versions ao usuário comum.
5. Preservar dados, foco e contexto em expandir/colapsar e após validação negada.

**Testes/verificação:** Criação mínima, critérios múltiplos, automática sem permissão, manual autorizado, parent prefill e erro sem perda de draft.

**Aceite:** Usuário cria Objective sem formulário técnico e entende se poderá iniciá-lo.

**Commit sugerido:** `feat(web): E57 streamline objective creation and readiness`.

<a id="e58"></a>
### E58 — Reorganizar criação de Task com owner e critérios

**Objetivo:** Criar rapidamente Backlog sem esconder os requisitos de execução e aceite.

**Depende de:** E09, E33, E56

**Arquivos:** [A] `packages/wos-api/web/assets/human-forms.js`; [A] `packages/wos-api/web/assets/workspace-form-fields.js`; [A] `packages/wos-api/web/assets/reference-picker.js`; [A] `packages/wos-api/web/assets/en-US.js`; [N] `tests/web/task-creation-flow.spec.mjs`.

**Implementação:**

1. Exibir Objective proprietário fixado no contexto e escolha explícita no fluxo global.
2. Destacar título, resultado/instruções resumidas e critérios; advanced guarda detalhes extensos.
3. Preservar Backlog default e distinguir campos obrigatórios para salvar dos exigidos para ativar.
4. Create in To Do só fica disponível quando capability atual permite e backend também valida.
5. Evitar duplicar description/instructions/deliverables automaticamente; conservar a representação canônica.

**Testes/verificação:** Task mínima, ausência de owner, owner de outro scope, Planned bloqueando todo, critérios incompletos e campos opcionais preservados.

**Aceite:** O fluxo produz Task válida e nunca aparenta iniciar execução apenas por salvar.

**Commit sugerido:** `feat(web): E58 streamline task creation with explicit ownership`.

<a id="e59"></a>
### E59 — Criar definições com dependências atomicamente

**Objetivo:** Garantir que o formulário completo não deixe entidades parcialmente configuradas.

**Depende de:** E46, E47, E57, E58

**Arquivos:** [A] `packages/wos-core/application/planning_creation.go`; [A] `packages/wos-core/application/commands.go`; [A] `packages/wos-core/application/dependencies.go`; [A] `packages/wos-api/web/assets/human-forms.js`; [A] `packages/wos-api/web/assets/workspace-plan-editor.js`; [N] `packages/wos-core/application/initial_dependency_creation_test.go`.

**Implementação:**

1. Estender initial_dependencies nos comandos de criação ou compor CreatePlanItems com helpers internos.
2. Validar targets, strength, motivos, permissões e grafo com a nova entidade já identificada, dentro da UoW.
3. Suportar Objective+Tasks por local key sem órfão temporário; sem criar dependências a partir de ordenação visual.
4. Emitir eventos de entidades, critérios e Relations com uma OutcomeRevision coerente.
5. UI só confirma sucesso completo; falha preserva intenção e inputs para correção/replay.

**Testes/verificação:** Dependência inválida desfaz criação/criteria; batch local keys; permissão parcial; duas criações concorrentes e timeout.

**Aceite:** O que o formulário anuncia como criado corresponde ao conjunto confirmado atomicamente.

**Commit sugerido:** `feat(planning): E59 create complete definitions with initial dependencies`.

## Fase I — Administração de chaves

<a id="e60"></a>
### E60 — Criar listagem administrativa de API Keys

**Objetivo:** Dar visibilidade às credenciais sem expor segredos.

**Depende de:** E20, E21, E50

**Arquivos:** [N] `packages/wos-api/web/assets/api-keys-view.js`; [A] `packages/wos-api/web/assets/app.js`; [A] `packages/wos-api/web/assets/navigation.js`; [A] `packages/wos-api/web/assets/en-US.js`; [N] `tests/web/api-key-management.spec.mjs`.

**Implementação:**

1. Integrar a área ao acesso administrativo existente, preservando onboarding/navegação.
2. Exibir nome, Principal, perfil, scope, validade, estado e último uso com significado claro.
3. Implementar paginação/filtros, estados vazio/loading/denied/error e seleção de chave.
4. Consultar permissões efetivas e explicar quando grant do Principal limita o template.
5. Não renderizar token/digest privado nem inferir online a partir de last_used_at.

**Testes/verificação:** Reader sem credentials:read, administrador com many keys, key revogada, paginação e nenhuma string secreta no DOM.

**Aceite:** Administrador identifica cada integração e seu alcance sem recorrer à tela técnica.

**Commit sugerido:** `feat(web): E60 add API key inventory and effective access view`.

<a id="e61"></a>
### E61 — Criar formulário seguro de emissão e permissões

**Objetivo:** Tornar seleção de papel e escopo explícita e verificável antes da emissão.

**Depende de:** E18, E19, E60

**Arquivos:** [N] `packages/wos-api/web/assets/api-key-form.js`; [A] `packages/wos-api/web/assets/api-keys-view.js`; [A] `packages/wos-api/web/assets/reference-picker.js`; [A] `packages/wos-api/web/assets/en-US.js`; [N] `tests/web/api-key-issuance.spec.mjs`.

**Implementação:**

1. Coletar nome, Principal, validade, template e escopo; mostrar lista concreta de permissões.
2. Permitir customização agrupada sem caixas perigosas selecionadas por default.
3. Exibir diferença entre grant do Principal e restrição da chave, e impacto de independência.
4. Emitir por operação atômica e mostrar segredo uma única vez com confirmação de cópia/armazenamento.
5. Evitar autosave de token em localStorage/sessionStorage/draft; impedir duplicação em retries incertos.

**Testes/verificação:** Template custom, permissões incompatíveis, Principal inválido, segredo one-time, refresh após emissão e erro de rede.

**Aceite:** A chave nasce com o alcance que o operador revisou, sem janela de privilégio maior.

**Commit sugerido:** `feat(web): E61 add constrained API key creation workflow`.

<a id="e62"></a>
### E62 — Criar fluxos explícitos de rotação e revogação

**Objetivo:** Administrar chaves sem confundir rotação com transferência de trabalho.

**Depende de:** E20, E21, E61

**Arquivos:** [A] `packages/wos-api/web/assets/api-keys-view.js`; [A] `packages/wos-api/web/assets/api-key-form.js`; [A] `packages/wos-api/web/assets/api-client.js`; [A] `packages/wos-api/web/assets/intents.js`; [N] `tests/web/api-key-revocation.spec.mjs`.

**Implementação:**

1. Mostrar authorities afetadas, validade de overlap e o que continua reservado à chave antiga.
2. Exigir confirmação, motivo e versões; tratar autorrevogação e última via administrativa.
3. Ao reduzir permissões, atualizar UI e invalidar capabilities/in-flight reads pertinentes.
4. Não apagar keys históricas referenciadas por assinaturas, contratos ou auditoria.
5. Recuperar resposta incerta pelo recibo sem exibir segredo omitido como falha de emissão inexistente.

**Testes/verificação:** Rotação com lease ativo, revoke em sessão atual, permissão revogada enquanto form aberto, conflito de versão e refresh/read failure.

**Aceite:** Operador entende o efeito real e o backend impede uso posterior de acesso removido.

**Commit sugerido:** `feat(web): E62 add key rotation and revocation impact flows`.

<a id="e63"></a>
### E63 — Exibir responsabilidade e reserva como informações distintas

**Objetivo:** Tornar evidente quem responde pelo trabalho e quem detém sua autoridade atual.

**Depende de:** E24, E25, E26, E34

**Arquivos:** [A] `packages/wos-api/web/assets/workspace-view.js`; [A] `packages/wos-api/web/assets/workspace-card-summary.js`; [A] `packages/wos-api/web/assets/workspace-kanban.js`; [A] `packages/wos-api/web/assets/actions.js`; [A] `packages/wos-api/web/assets/en-US.js`; [N] `tests/web/credential-bound-lease.spec.mjs`.

**Implementação:**

1. Mostrar assignees separadamente de holder Principal/credencial e prazo de reserva.
2. Exibir countdown como projeção informativa baseada em server time; write sempre revalida no backend.
3. Ações renew/release/reclaim/intervene respeitam capability e protocolo.
4. Explicar outra chave do mesmo Principal como holder diferente; não inferir posse de assignee.
5. Mostrar atenção após expiração e recuperação explícita, sem mover card automaticamente para To Do.

**Testes/verificação:** Assignee diferente do holder, sessão/root, duas keys mesmo Principal, clock skew, lease expired e signed-host.

**Aceite:** A interface permite operar concorrência sem transmitir que atribuição equivale a lock ou presença online.

**Commit sugerido:** `feat(web): E63 distinguish task responsibility and active authority`.

## Fase J — Kanban e experiência operacional

<a id="e64"></a>
### E64 — Ajustar movimentações e bloqueios do Kanban

**Objetivo:** Transformar cada arraste numa operação válida, explicada e confirmada.

**Depende de:** E12, E34, E40, E56, E63

**Arquivos:** [A] `packages/wos-api/web/assets/kanban-operations.js`; [A] `packages/wos-api/web/assets/workspace-kanban.js`; [A] `packages/wos-api/web/assets/workspace-view.js`; [A] `packages/wos-api/web/assets/workspace-kanban.css`; [N] `tests/web/operational-kanban-transitions.spec.mjs`.

**Implementação:**

1. Mapear destinos para comandos reais, separando inspections, browser commands e signed-host handoffs.
2. Bloquear ativação/aquisição de Task quando parent/ancestral não iniciou e oferecer remedy contextual.
3. Manter card na origem enquanto prepara/confirma; atualizar somente após persistência reconhecida.
4. Preservar duas intenções em Backlog→execução e impedir auto-start de parent.
5. Exibir fases canceladas/expiradas sem contá-las como Accepted ou disponibilidade de execução.

**Testes/verificação:** DnD permitido/negado em cada protocolo, stale capability, Objective Planned, erro e retorno, data-state após sucesso real.

**Aceite:** Movimentação visual nunca é fonte da verdade nem substitui uma decisão do backend.

**Commit sugerido:** `feat(web): E64 align Kanban moves with operational commands`.

<a id="e65"></a>
### E65 — Integrar entrega e revisão no inspector

**Objetivo:** Distinguir inspecionar prova, entregar, avaliar e aceitar.

**Depende de:** E29, E30, E31, E32, E34, E64

**Arquivos:** [N] `packages/wos-api/web/assets/task-delivery-form.js`; [A] `packages/wos-api/web/assets/workspace-kanban.js`; [A] `packages/wos-api/web/assets/human-forms.js`; [A] `packages/wos-api/web/assets/workspace-view.js`; [A] `packages/wos-api/web/assets/workspace-evidence-form.js`; [N] `tests/web/task-verification-flow.spec.mjs`.

**Implementação:**

1. Adicionar ações com copy inequívoca para Review proof, Submit for verification, Approve delivery e Request changes.
2. Reutilizar formulários existentes de Evidence/Artifact/Assessment e a projeção comum de verificação.
3. Mostrar rodada e material exato; checkbox representa assessment atual, nunca apenas Evidence existente.
4. Para signed, apresentar handoff ao host com IDs seguros e sem assinatura no browser.
5. Preservar draft ao abrir documentação/remedy e voltar à entrega original; não empilhar dialogs.

**Testes/verificação:** Jornada nativa completa, rejeição/correção/nova rodada, assinatura externa, última checkbox não aceita Task, materiais stale e permissões.

**Aceite:** Task entra de fato em Verification e a ação de aceite é transparente e autorizada.

**Commit sugerido:** `feat(web): E65 integrate explicit delivery and verification flows`.

<a id="e66"></a>
### E66 — Exibir elegibilidade e conclusão de Objective

**Objetivo:** Mostrar progresso e fechamento automático sem sugerir aceite de Outcome.

**Depende de:** E35, E36, E37, E38, E39, E40, E64

**Arquivos:** [A] `packages/wos-api/web/assets/objective-completion-view.js`; [A] `packages/wos-api/web/assets/workspace-view.js`; [A] `packages/wos-api/web/assets/workspace-kanban.js`; [A] `packages/wos-api/web/assets/workspace-conclusion-form.js`; [A] `packages/wos-api/web/assets/en-US.js`; [N] `tests/web/objective-completion-flow.spec.mjs`.

**Implementação:**

1. Exibir contagem autoritativa, critérios próprios, pendências, modo e delegação.
2. Modo manual apresenta Ready to achieve e confirmação; automático apresenta condição e causa quando concluído.
3. Mostrar cancelamentos/waivers como motivos de revisão manual, não como sucesso.
4. Exibir reconciliation_pending quando houver processamento durável, sem mover para Achieved antes do commit.
5. No reopen, mostrar ciclo novo e avaliações a reafirmar; não oferecer re-aceite com prova antiga.

**Testes/verificação:** Último aceite, último Criterion, Issue pendente, zero Tasks, parent nested, auto→manual, reopened cycle e Outcome inalterado.

**Aceite:** Operador entende exatamente por que o Objective não fechou ou qual regra o concluiu.

**Commit sugerido:** `feat(web): E66 expose objective completion eligibility and audit`.

<a id="e67"></a>
### E67 — Criar editor legível de dependências em duas direções

**Objetivo:** Permitir criar e administrar pré-requisitos sem inversão de source/target.

**Depende de:** E46, E47, E48, E59

**Arquivos:** [N] `packages/wos-api/web/assets/dependency-editor.js`; [A] `packages/wos-api/web/assets/workspace-view.js`; [A] `packages/wos-api/web/assets/workspace-entity-context.js`; [A] `packages/wos-api/web/assets/human-forms.js`; [A] `packages/wos-api/web/assets/reference-picker.js`; [N] `tests/web/dependency-management.spec.mjs`.

**Implementação:**

1. Separar Depends on e Required by, com status e hard/advisory por linha.
2. Ao editar incoming, indicar o source real e suas restrições; não fingir edição da entidade selecionada.
3. Integrar preview/apply batch, referências de impacto e bloqueios existentes.
4. Filtrar escolhas inválidas por eligibility, mantendo a validação definitiva no servidor.
5. Permitir remover com motivo e comparar conflitos sem perder alterações preparadas.

**Testes/verificação:** A depende de B versus B depende de A, cycle/hierarchy deadlock, frozen source, incoming edit, batch conflito e partial coverage.

**Aceite:** A frase exibida, o preview e a Relation persistida descrevem o mesmo sentido.

**Commit sugerido:** `feat(web): E67 add bidirectional dependency management`.

<a id="e68"></a>
### E68 — Melhorar conexão e impacto no Canvas

**Objetivo:** Usar o Canvas como apoio visual, sem transformar posicionamento em vínculo estrutural.

**Depende de:** E48, E67

**Arquivos:** [A] `packages/wos-api/web/assets/workspace-canvas.js`; [A] `packages/wos-api/web/assets/canvas-data.js`; [A] `packages/wos-api/web/assets/app.js`; [A] `packages/wos-api/web/assets/workspace-entity-context.js`; [N] `tests/web/dependency-canvas-connection.spec.mjs`.

**Implementação:**

1. Adicionar modo de conexão explícito com source selecionado, target, strength e frase de confirmação.
2. Diferenciar hierarquia, dependência hard/advisory e Blocker por semântica/texto, não apenas cor.
3. Selecionar um nó permite ver impactos upstream/downstream e limites da visualização.
4. Cancelar conexão não muda posições, relations ou seleção original.
5. Manter operação equivalente pelo editor de lista e preservar performance com graph bounded.

**Testes/verificação:** Pan/zoom versus conectar, cancelar, touch/keyboard pelo editor, edge removida, grafo parcial e câmera após operação.

**Aceite:** O Canvas comunica relações reais sem criar vínculos por gestos ambíguos.

**Commit sugerido:** `feat(web): E68 add explicit dependency connections and impact views`.

<a id="e69"></a>
### E69 — Concluir read model e lanes de Issues

**Objetivo:** Integrar Issues ao Kanban contextual sem misturar ownership e afetação.

**Depende de:** E41, E42, E43, E44, E45, E64

**Arquivos:** [A] `packages/wos-core/application/workspace_cards.go`; [A] `packages/wos-core/application/workspace_read_types.go`; [A] `packages/wos-core/storage/sqlite/workspace_read.go`; [A] `packages/wos-api/web/assets/workspace-kanban.js`; [A] `packages/wos-api/web/assets/kanban-operations.js`; [N] `tests/web/issue-kanban.spec.mjs`.

**Implementação:**

1. Adicionar coleção de Issues por Objective com filtros, cursor, contagens e revisão coerentes.
2. Implementar lanes Open/Investigating/Closed preservando resultado terminal específico.
3. Contexto de Outcome pode agregar Issues autorizadas com owner explícito; não tratar affected_refs como filtro de ownership.
4. Criar/New Issue segue o owner do contexto; destino Closed abre decisão explícita.
5. Preservar uma modal, scroll/foco e retorno ao Board.

**Testes/verificação:** Issues de Objectives vizinhos, duplicate/wont_fix badges, cardinalidade de cards, paginação, pesquisa e criação contextual.

**Aceite:** Issue é operável no mesmo workspace com escopo e estado inequívocos.

**Commit sugerido:** `feat(web): E69 add objective-scoped Issue Kanban`.

<a id="e70"></a>
### E70 — Completar inspector e fluxos operacionais de Issue

**Objetivo:** Manter responsabilidade, investigação, resolução e impactos juntos.

**Depende de:** E42, E44, E45, E69

**Arquivos:** [A] `packages/wos-api/web/assets/workspace-view.js`; [A] `packages/wos-api/web/assets/human-forms.js`; [A] `packages/wos-api/web/assets/workspace-blocking-form.js`; [A] `packages/wos-api/web/assets/workspace-entity-context.js`; [N] `tests/web/issue-operational-flow.spec.mjs`.

**Implementação:**

1. Mostrar proprietário, assignee, reportador, entidades afetadas, relações corretivas e atividades.
2. Assumir/investigar/repassar/liberar seguem comandos e permissões granularizadas.
3. Fechar exige escolher resolved/wont_fix/duplicate com os respectivos campos e justificativas.
4. Criar Issue com múltiplos Blockers mostra cada impacto; resolver deixa seleção explícita de liberação.
5. Apresentar o efeito sobre achievement do Objective, sem alegar bloqueio automático de todas as Tasks.

**Testes/verificação:** Fluxo report→assign→investigate→resolve, duplicate pendente, wont_fix sem motivo, múltiplos Blockers e corrida de assignee.

**Aceite:** Operador fecha o problema sem liberar impedimentos ou aceitar resultados inadvertidamente.

**Commit sugerido:** `feat(web): E70 integrate Issue responsibility and resolution flows`.

<a id="e71"></a>
### E71 — Preservar navegação, drafts e recuperação incerta

**Objetivo:** Evitar que a integração dos fluxos reintroduza perda de dados ou comandos duplicados.

**Depende de:** E56, E57, E58, E61, E62, E65, E66, E67, E70

**Arquivos:** [A] `packages/wos-api/web/assets/app.js`; [A] `packages/wos-api/web/assets/intents.js`; [A] `packages/wos-api/web/assets/workspace-kanban.js`; [A] `packages/wos-api/web/assets/navigation.js`; [A] `packages/wos-api/web/assets/api-client.js`; [N] `tests/web/operational-intent-recovery.spec.mjs`.

**Implementação:**

1. Centralizar retorno ao contexto de origem, mantendo item, filtros, scroll, câmera e foco.
2. Invalidar in-flight reads na troca de scope ou revogação; resposta atrasada não reabre item de outra seleção.
3. Preservar draft em 403/409/412 e oferecer comparação sem substituir expected version automaticamente.
4. Timeout mantém mesmo body/key; write confirmado com refresh falho só repete read.
5. Bloquear saída apenas enquanto existe intenção realmente pendente e oferecer recuperação compreensível.

**Testes/verificação:** Network loss antes/depois do commit, fechar modal durante read, alterar Outcome, retomar draft e navegação Back/Escape.

**Aceite:** Nenhum fluxo usa nova intenção para recuperar um write já enviado nem perde texto por conflito.

**Commit sugerido:** `fix(web): E71 preserve operational navigation and exact intent recovery`.

<a id="e72"></a>
### E72 — Garantir operações por toque, teclado e acessibilidade

**Objetivo:** Remover dependência exclusiva de arraste sem recolocar select genérico de status.

**Depende de:** E64, E65, E66, E67, E68, E69, E70, E71

**Arquivos:** [A] `packages/wos-api/web/assets/workspace-kanban.js`; [A] `packages/wos-api/web/assets/workspace-view.js`; [A] `packages/wos-api/web/assets/workspace-kanban.css`; [A] `packages/wos-api/web/assets/workspace-board.css`; [A] `tests/web/accessibility.spec.mjs`; [N] `tests/web/operational-accessibility.spec.mjs`.

**Implementação:**

1. Manter operações contextuais acionáveis por clique/toque e teclado; DnD é apenas um atalho.
2. Verificar foco no dialog substituto e retorno ao opener, labels, regiões vivas e hierarquia.
3. Diferenciar estados por texto/ícone além de cor; testar nomes longos e escala.
4. Executar viewports e zoom previstos, sem ocultar botões fora do viewport nem forçar clicks nos testes.
5. Registrar Axe e testes manuais separadamente; não declarar conformidade global por uma suite escopada.

**Testes/verificação:** Mouse sem drag, touch por botões, teclado completo, 320/390 px e zoom 400%; screen reader manual registrado quando executado.

**Aceite:** Todas as operações essenciais são utilizáveis sem arraste e o foco permanece previsível.

**Commit sugerido:** `test(web): E72 qualify accessible operational interactions`.

## Fase K — Qualificação transversal

<a id="e73"></a>
### E73 — Executar matriz funcional e de autorização completa

**Objetivo:** Fechar lacunas de cobertura entre as funcionalidades já testadas localmente.

**Depende de:** E13, E21, E25, E33, E38, E43, E47, E49, E72

**Arquivos:** [N] `packages/wos-core/application/operational_matrix_test.go`; [N] `packages/wos-core/domain/operational_invariants_test.go`; [N] `tests/acceptance/operational_authorization_matrix.py`; [A] `docs/implementation/operational-consolidation/manifest.json`.

**Implementação:**

1. Executar todos os cenários da seção 15 com perfis/protocolos/roles definidos.
2. Comparar invariantes de read e write, public routes e trusted embedded.
3. Adicionar property tests para sequência de transições, ownership e ausência de aceitação sem prova.
4. Registrar denominador de cenários, pass/fail/skip e SHA exata.
5. Corrigir falhas encontradas sem enfraquecer asserts ou desabilitar casos.

**Testes/verificação:** Domain/Application completos e matriz positiva/negativa, sem inferir sucesso de testes não executados.

**Aceite:** Não há requisito funcional sem caso de aceite nem permissão crítica sem caso negativo.

**Commit sugerido:** `test(core): E73 qualify full operational authorization matrix`.

<a id="e74"></a>
### E74 — Testar concorrência em processos e bancos reais

**Objetivo:** Demonstrar exclusividade e atomicidade além do adapter Memory.

**Depende de:** E73

**Arquivos:** [N] `packages/wos-core/storage/sqlite/operational_concurrency_test.go`; [N] `packages/wos-core/storage/postgres/operational_concurrency_process_test.go`; [N] `tests/acceptance/operational_concurrency.py`; [A] `packages/wos-core/application/transaction_clock.go`.

**Implementação:**

1. Executar races de claims, renew/expiry/reclaim, revogação versus entrega e decisões de revisão.
2. Testar dois últimos aceites, criação concorrente com achievement e duas mudanças de dependência opostas.
3. Usar processos/serviços distintos compartilhando o mesmo PostgreSQL e equivalente suportado em SQLite.
4. Verificar clocks depois de espera por lock, deadlocks SQL, versões e eventos emitidos.
5. Registrar throughput/latência observados sem inventar SLA; retries de serialização preservam intenção e versões.

**Testes/verificação:** go test -race, testes multiprocesso, PostgreSQL real com DSN e SQLite real; Memory isoladamente não aprova a etapa.

**Aceite:** Somente uma autoridade/decisão vence por versão e os losers não deixam efeitos parciais.

**Commit sugerido:** `test(concurrency): E74 prove multi-agent operational isolation`.

<a id="e75"></a>
### E75 — Qualificar migrações, backup e restore

**Objetivo:** Verificar que o modelo novo funciona com dados e histórico reais legados.

**Depende de:** E23, E28, E36, E74

**Arquivos:** [A] `packages/wos-core/storage/sqlite/migrations.go`; [A] `tools/postgresgen/generate.py`; [N] `packages/wos-core/storage/sqlite/operational_upgrade_test.go`; [N] `packages/wos-core/storage/postgres/operational_upgrade_test.go`; [A] `tools/acceptance/issue_ownership_backfill.py`; [N] `tools/acceptance/operational_upgrade.py`.

**Implementação:**

1. Cobrir fresh DB, banco anterior à obrigatoriedade de owner, baseline 0036 e migrações novas.
2. Testar parada de contract migration por órfãos e posterior backfill explícito com recibo.
3. Restaurar leases, contratos, entregas, revisões, conclusions e policies em destino limpo.
4. Demonstrar recusa de writer antigo após cutover e preservação de eventos/assinaturas históricas.
5. Validar checkpoints de backup e recuperação sem drop destrutivo de dados novos.

**Testes/verificação:** SQLite e PG, órfãos, active legacy authority, secret ausente, signatures antigas, reboot e migration checksum.

**Aceite:** Upgrade e restore preservam significado e dados; não apenas uma instalação vazia passa.

**Commit sugerido:** `test(storage): E75 qualify operational upgrades and clean restore`.

<a id="e76"></a>
### E76 — Executar paridade e compatibilidade pública

**Objetivo:** Provar o mesmo comportamento por todas as superfícies suportadas.

**Depende de:** E49, E50, E51, E52, E53, E54, E55, E75

**Arquivos:** [N] `tests/acceptance/operational_transport_matrix.py`; [A] `packages/wos-api/http/handler_test.go`; [A] `packages/wos-api/mcp/server.go`; [A] `packages/wos-sdk-go/operational_queries_test.go`; [A] `packages/wos-cli/workspace_commands_v2.go`.

**Implementação:**

1. Executar as jornadas por REST histórico, command API, MCP, SDK remoto e CLI.
2. Comparar estado final, revisions, event semantics, auth decisions e idempotency.
3. Validar wire de versões grandes, schemas e campos required/nullable.
4. Manter envelopes assinados e registros históricos byte-compatible conforme fixtures.
5. Confirmar que o CLI/npm wrappers não divergem do binário principal.

**Testes/verificação:** Matriz completa com pelo menos um fluxo real de cada protocolo e transporte; erros estruturados e receipts.

**Aceite:** Nenhum cliente consegue executar operação que outro nega sob as mesmas condições de domínio.

**Commit sugerido:** `test(api): E76 qualify transport and signed compatibility`.

<a id="e77"></a>
### E77 — Executar jornadas E2E e regressão de onboarding

**Objetivo:** Confirmar o produto integrado, não somente componentes isolados.

**Depende de:** E72, E76

**Arquivos:** [N] `tests/web/operational-end-to-end.spec.mjs`; [N] `tests/web/onboarding-operational-regression.spec.mjs`; [A] `tests/web/package.json`; [A] `docs/implementation/operational-consolidation/manifest.json`.

**Implementação:**

1. Executar primeiro acesso, login, Namespace/Outcome, Objective, Task, critérios, start, claim, delivery e review.
2. Cobrir execução concorrente de dois agentes com UI humana de supervisão e políticas diferentes.
3. Executar Issue→Blocker→investigação→resolução e Objective auto/manual.
4. Comparar screenshots e DOM com baseline de onboarding, identidade e layout, sem trocar assets sem autorização.
5. Capturar origem/retorno de dialogs, versões persistidas intermediárias e falhas de rede reais simuladas de forma controlada.

**Testes/verificação:** Browser full suite com screenshots vinculados à SHA, testes de create/add entrypoints, mobile/HD e fluxo real de backend.

**Aceite:** Todas as jornadas-alvo funcionam sem regressão não autorizada no onboarding e sem cliques forçados para mascarar defeitos.

**Commit sugerido:** `test(web): E77 qualify complete operational journeys`.

<a id="e78"></a>
### E78 — Medir custo de consultas e cascatas

**Objetivo:** Evitar N+1 e regressões de latência sem otimizar por suposições.

**Depende de:** E40, E48, E74, E77

**Arquivos:** [A] `packages/wos-core/storage/sqlite/workspace_read.go`; [A] `packages/wos-core/storage/sqlite/graph_read.go`; [A] `packages/wos-core/storage/sqlite/objective_completion.go`; [N] `packages/wos-core/storage/sqlite/operational_benchmark_test.go`; [N] `tools/acceptance/operational_performance_report.py`.

**Implementação:**

1. Preparar datasets determinísticos com 10, 100, 1.000 e, dentro dos limites reais, 10.000 Tasks por Outcome.
2. Medir cards, completion report, ready-work, impact, key listing e autofechamento: latência, queries, rows e duração de guard.
3. Comparar baseline e nova SHA no mesmo ambiente; declarar limites não qualificados.
4. Ajustar índices, batch loading e orçamento de cascata quando os dados justificarem; não trocar consistência por cache incorreto.
5. Mostrar cobertura parcial somente em apresentação; gate de aceite continua completo.

**Testes/verificação:** Benchmarks SQLite/PG, planos de consulta no banco real e repetição suficiente para relatar distribuição; números são medidos, não inventados.

**Aceite:** Hot paths têm custo conhecido e nenhuma consulta de UI determina fechamento com scan truncado.

**Commit sugerido:** `perf(queries): E78 qualify operational reads and completion cascades`.

## Fase L — Entrega operacional e handoff

<a id="e79"></a>
### E79 — Consolidar eventos, observação e diagnóstico

**Objetivo:** Permitir diagnóstico sem confundir coordenação com execução externa.

**Depende de:** E39, E55, E76, E78

**Arquivos:** [A] `packages/wos-core/domain/event.go`; [A] `packages/wos-core/domain/audit.go`; [A] `packages/wos-core/application/pipeline.go`; [A] `packages/wos-core/application/objective_completion_reconcile.go`; [A] `tools/eventcatalog/check.py`; [N] `docs/operations/operational-diagnostics.md`.

**Implementação:**

1. Catalogar eventos de activation denial observável, lease binding, delivery/review e achievement automático.
2. Manter negativas em observação/auditoria adequada sem criar Domain Event de mutação que não aconteceu.
3. Incluir causation, command_id, scope e versões; nunca token, private key ou payload confidencial desnecessário.
4. Expor estados de reconciliação pendente e recuperação de operações sem fingir presença online do agente.
5. Documentar distinção entre task delivery e entrega de webhook/integration.

**Testes/verificação:** Catálogo zero-diff, evento de efeito composto, deduplicação, secrets scan em logs e erro com correlation ID.

**Aceite:** Operador identifica quem fez o quê, por qual chave e qual política sem expor credenciais.

**Commit sugerido:** `feat(observability): E79 catalog operational decisions and diagnostics`.

<a id="e80"></a>
### E80 — Criar cenário demonstrável e roteiro de aceite

**Objetivo:** Permitir verificação humana reproduzível do comportamento-alvo.

**Depende de:** E77, E79

**Arquivos:** [N] `tools/acceptance/operational_demo_fixture.py`; [N] `docs/implementation/operational-consolidation/walkthrough.md`; [N] `docs/implementation/operational-consolidation/acceptance.json`.

**Implementação:**

1. Criar fixture isolada com Outcome, dois Objectives dependentes, Tasks, uma Issue e perfis de planner/executor/reviewer.
2. Usar credenciais temporárias de teste e nunca versionar seus segredos.
3. Documentar cada gesto/comando, resultado persistido esperado e erro esperado em casos negativos.
4. Demonstrar Planned bloqueando execução, lease exclusivo, entrega/revisão e Objective automático sem fechar Outcome.
5. Incluir caminho manual com cancelamento explicitamente reconhecido e cenário de key rotation.

**Testes/verificação:** Roteiro executado em instalação limpa e repetível, com resultados registrados por SHA.

**Aceite:** A demonstração prova os gates; não é um mock visual ou conjunto de screenshots sem backend.

**Commit sugerido:** `docs(acceptance): E80 add reproducible operational walkthrough`.

<a id="e81"></a>
### E81 — Publicar runbook de rollout e migração coordenada

**Objetivo:** Tornar a adoção segura para instalações existentes sem publicação automática.

**Depende de:** E75, E76, E79, E80

**Arquivos:** [N] `docs/operations/operational-profile-upgrade.md`; [A] `docs/ux/operational-kanban/issue-ownership-backfill.md`; [A] `packages/wos-cli/README.md`; [A] `README.md`; [A] `ROADMAP.md`.

**Implementação:**

1. Descrever backup, auditoria, expand, backfill, drenagem, policy/grant migration, upgrade de clientes e contract cutover.
2. Definir como tratar Task ativa sob Planned e lease sem prova de credencial, sem correção silenciosa.
3. Explicar protocolo nativo versus assinado, nova exigência de revisão e modes de completion.
4. Documentar versão mínima de writer/CLI/SDK e procedimento de restauração compatível.
5. Manter release/tag/publicação como ato separado não autorizado por este plano.

**Testes/verificação:** Runbook validado em fixture legada e revisão de operadores; exemplos correspondem a comandos realmente implementados.

**Aceite:** Outro operador migra uma instalação e sabe quando parar por pendência concreta.

**Commit sugerido:** `docs(operations): E81 document coordinated operational rollout`.

<a id="e82"></a>
### E82 — Regenerar, executar CI completo e vincular à HEAD

**Objetivo:** Qualificar a integração final em uma SHA identificável.

**Depende de:** E73, E74, E75, E76, E77, E78, E79, E81

**Arquivos:** [A] `.github/workflows/ci.yml`; [A] `tools/postgresgen/generate.py`; [A] `tools/signedgen/generate.py`; [A] `tools/mcpgen/main.go`; [A] `tools/openapigen/main.go`; [A] `docs/implementation/operational-consolidation/manifest.json`.

**Implementação:**

1. Executar a sequência real de generators e checar diff zero depois de comitar os derivados corretos.
2. Rodar go test, vet, race, JS units, Playwright, storage real, signed fixtures e checks existentes de release/restore aplicáveis.
3. Atualizar fixture incompatível apenas para cumprir a regra nova legítima, não para pular validação.
4. Confirmar hosted CI da HEAD atual; não reutilizar checks verdes do baseline ou de commit anterior.
5. Só anunciar operational_v2 nas capabilities de produto quando todos os invariants e fluxos necessários estiverem completos.

**Testes/verificação:** CI completo na SHA final com testes PG não pulados; relatórios de falhas intermediárias preservados.

**Aceite:** Não há schema drift, funcionalidade parcialmente exposta ou evidência de outra SHA apresentada como atual.

**Commit sugerido:** `test(release): E82 qualify final operational integration head`.

<a id="e83"></a>
### E83 — Realizar aceite manual de compreensão e acessibilidade

**Objetivo:** Distinguir implementação funcional de experiência compreendida por pessoas.

**Depende de:** E80, E82

**Arquivos:** [N] `docs/implementation/operational-consolidation/manual-acceptance.md`; [A] `docs/implementation/operational-consolidation/acceptance.json`; [A] `tests/web/operational-accessibility.spec.mjs`.

**Implementação:**

1. Solicitar a um avaliador executar tarefas sem instruções de clique: criar Objective/Task, identificar bloqueio, distinguir assignee de holder, entregar/revisar.
2. Verificar se entende Depends on/Required by, Issue versus Blocker e o motivo de Objective não fechar.
3. Executar combinações manuais disponíveis de teclado/leitor de tela/dispositivo físico; registrar exatamente quais.
4. Corrigir problemas observados com evidência, preservando identidade e contratos.
5. Se avaliador ou hardware não estiver disponível, marcar esse aceite como pendente; não fabricar resultado de estudo.

**Testes/verificação:** Relatório com participantes/ambiente anonimizados, tarefas, resultados reais e limites. Automatizado permanece evidência separada.

**Aceite:** O status de aceite humano é honesto e os problemas encontrados têm resolução ou pendência explícita.

**Commit sugerido:** `docs(ux): E83 record independent operational acceptance`.

<a id="e84"></a>
### E84 — Encerrar o planejamento executado e entregar PR Draft

**Objetivo:** Deixar repositório, documentação e estado do PR reconciliados para decisão do proprietário.

**Depende de:** E82, E83

**Arquivos:** [A] `ROADMAP.md`; [A] `docs/ux/operational-kanban/issue-and-dependency-plan.md`; [A] `docs/implementation/operational-consolidation/manifest.json`; [A] `docs/implementation/operational-consolidation/acceptance.json`; [A] `CHANGELOG.md`.

**Implementação:**

1. Atualizar a matriz E01–E84 e o crosswalk com as 44 etapas anteriores usando commits/evidências reais.
2. Separar implementado, testado localmente, validado em CI e aceite humano pendente/concluído.
3. Preparar resumo de contratos alterados, migrações, compatibilidade, comandos e jornadas entregues.
4. Confirmar onboarding preservado, sem secrets versionados e sem alterações não relacionadas.
5. Confirmar PR #116 continua aberto/Draft; não marcar ready, não mergear, não criar tag ou publicar pacote.

**Testes/verificação:** Revisão final de diff, links, manifest, HEAD/checks e estado remoto do PR.

**Aceite:** O proprietário recebe uma entrega auditável e decide separadamente sobre aprovação/merge/release.

**Commit sugerido:** `docs(wos): E84 finalize operational implementation handoff`.

<a id="15-matriz-de-testes-e-cenarios"></a>
## 15. Matriz de testes e cenários

### 15.1 Regras de execução

Cada cenário abaixo precisa ter fixture, comando/query executado, resultado esperado, estado persistido e eventos/revisões observados. Para negativas, verificar também **ausência de efeitos parciais**. Não testar apenas status HTTP.

Executar cenários relevantes por Memory, SQLite e PostgreSQL; cenários de transporte pelo adapter real; concorrência por duas conexões/processos reais; UX por browser com backend. Um skip é não qualificado, não pass.

A policy de revisão, profile operacional e modo de autoridade são dimensões da matriz. Não afirmar equivalência com base em um único caminho assinado ou em um único humano administrador.

### LC — Lifecycle e hierarchy

| ID | Cenário | Resultado exigido |
|---|---|---|
| T-LC-01 | Criar Task em Backlog sob Objective Planned | Criação permitida com owner válido; nenhuma autoridade adquirida. |
| T-LC-02 | Ativar Task sob Objective Planned | Negar com objective_not_started; versão/eventos não mudam. |
| T-LC-03 | Criar Task diretamente em todo sob Planned | Negar atomicamente, inclusive initial criteria/dependencies. |
| T-LC-04 | Claim/acquire sob Planned | Negar em legacy, contracts_v1 e signed_v2. |
| T-LC-05 | Acquire-next quando apenas Tasks de Objectives Planned existem | Não adquirir; explicar ausência de trabalho pronto sem inventar Task. |
| T-LC-06 | Objective In Progress sob ancestral Planned | Negar início de trabalho descendente com referência ao ancestral. |
| T-LC-07 | AchieveObjective partindo de Planned em perfil v2 | Negar; não executar Start implicitamente. |
| T-LC-08 | Ativar quando Outcome é draft, terminal ou archived | Negar segundo gate; criar Backlog só quando planejamento é permitido. |
| T-LC-09 | Criar/reparent Task para Objective terminal | Negar e preservar owner anterior. |
| T-LC-10 | Cancelar Objective com lease/contract/review ativa | Negar ou exigir intervenção explícita; não abandonar autoridade. |
| T-LC-11 | Release de Task cujo Objective está bloqueado | Permitir recuperação ao titular válido; não exigir avanço habilitado. |
| T-LC-12 | Reabrir Task sob Objective Achieved/Planned | Negar até reabrir/iniciar parents aplicáveis. |
| T-LC-13 | ReopenObjective | Preservar ConclusionHistory, retornar Planned e avançar ciclo. |
| T-LC-14 | Chamada embedded ao domínio v2 | Invariantes se mantêm; identidade de credencial não pode vir de payload não confiável. |

### AU — Autorização e API Keys

| ID | Cenário | Resultado exigido |
|---|---|---|
| T-AU-01 | Novo comando sem registro de autorização | Teste de catálogo falha; execução nega por default. |
| T-AU-02 | Executor tenta editar definição ou aprovar entrega | Negar sem as permissões específicas. |
| T-AU-03 | Reviewer tenta adquirir execução | Negar salvo concessão explícita separada. |
| T-AU-04 | Criar Task+Criterion sem criterion:define | Negar toda a criação composta. |
| T-AU-05 | Criar Issue+Blocker sem blocker:create | Negar toda a operação composta. |
| T-AU-06 | Credencial restringe Outcome que o Principal pode ler | Restrição da credencial prevalece; sem vazamento em preview/counters. |
| T-AU-07 | Actor/Principal/Credential falsificados no corpo | Ignorar como autoridade ou rejeitar schema; usar identidade autenticada. |
| T-AU-08 | Grants amplos e CredentialPolicy vazia | Não converter vazio em acesso irrestrito. |
| T-AU-09 | Permissão desconhecida, duplicada ou acima do limite | Validar e rejeitar; catálogo/contratos têm os mesmos limites. |
| T-AU-10 | Template atualizado depois de emitir chave | Chave anterior conserva permissões materializadas. |
| T-AU-11 | Emissão falha antes de persistir policy | Rollback integral; não sobra token válido irrestrito. |
| T-AU-12 | Resposta de emissão perdida e replay | Recibo volta sem segredo; não reemitir secret nem nova chave ocultamente. |
| T-AU-13 | Revogação concorrente com mutation | Ordem transacional determina resultado; depois do revoke confirmado, negar. |
| T-AU-14 | Root credential revogada | Sessões filhas deixam de autenticar/autorizar. |
| T-AU-15 | Alteração da última via administrativa | Negar sem recovery válido ou confirmação do procedimento permitido. |
| T-AU-16 | List/status/audit de chaves | Nenhum token, digest secreto ou private key aparece. |
| T-AU-17 | Alias legado e comando novo sensível | Alias não concede capacidade não existente no mapa congelado. |

### LS — Reserva e autoridade

| ID | Cenário | Resultado exigido |
|---|---|---|
| T-LS-01 | Duas chaves tentam claim da mesma Task/version | Uma vence; outra recebe conflito, sem lease duplicado. |
| T-LS-02 | Duas chaves diferentes do mesmo Principal | A segunda não renew/release/deliver usando autoridade da primeira. |
| T-LS-03 | Duas sessões da mesma chave root | Podem operar o lease se ambas continuam válidas e autorizadas. |
| T-LS-04 | Outro Actor com o mesmo Principal em credencial incompatível | Binding/autoridade nega; não depende apenas de texto do assignee. |
| T-LS-05 | Renovar exatamente em expires_at | Negar: igualdade é expiração. |
| T-LS-06 | Reclaim após expirar | Novo claim/execution e fence estritamente maior. |
| T-LS-07 | Executor antigo entrega depois de reclaim | Negar stale_execution mesmo com material aparentemente válido. |
| T-LS-08 | Resposta perdida de claim e retry pela mesma credencial | Retornar a mesma aquisição, não duplicar nem estender prazo indevidamente. |
| T-LS-09 | Outra chave tenta replay da aquisição | Não divulgar authority receipt utilizável. |
| T-LS-10 | Rotação de chave com lease ativo | Nova chave não herda reserva; a antiga segue política de overlap. |
| T-LS-11 | Revogar chave com lease ativo | Nenhuma nova ação; reserva só liberada/recuperada por operação definida. |
| T-LS-12 | Mudar assignee durante execução | Não muda holder, claim ou fencing. |
| T-LS-13 | Clock skew do cliente e espera longa em lock | Servidor usa tempo transacional após espera, não timestamp do cliente. |
| T-LS-14 | Passagem do tempo com lease expirado | Projeção muda; lifecycle não muda sozinho. |
| T-LS-15 | Processo externo continua após revogação | WOS recusa writes antigos; UI não afirma ter interrompido o processo externo. |

### RV — Entrega, prova e revisão

| ID | Cenário | Resultado exigido |
|---|---|---|
| T-RV-01 | Abrir Review proof | Nenhuma entrega/estado é criado. |
| T-RV-02 | SubmitTaskForReview válido no lease legado | Registra entrega, encerra lease e vincula pending delivery atomicamente. |
| T-RV-03 | SubmitTaskForReview para contrato v1 | Usa WorkSubmission real e não duplica material. |
| T-RV-04 | Comando nativo aplicado a Task assinada | Negar mismatch; não fallback para revisão sem assinatura. |
| T-RV-05 | ReturnSignedWork com política v2 | Abre revisão assinada de fato; não conclui direto. |
| T-RV-06 | In Progress sem lease nem entrega/caso válido | Estado rejeitado; não usar booleano client-side para contornar. |
| T-RV-07 | Review independente com outra key do mesmo Principal | Negar; key/profile diferente não cria independência. |
| T-RV-08 | Dois revisores decidem simultaneamente | Uma decisão vence por versão; a outra não altera o histórico. |
| T-RV-09 | Evidence adicionada sem Assessment | Critério continua não aprovado. |
| T-RV-10 | Assessment de revisão antiga do Criterion | Não é aplicável à definição atual. |
| T-RV-11 | Assessment de entrega/rodada anterior | Não aceita a nova entrega automaticamente. |
| T-RV-12 | Waiver em critério obrigatório | Não aparece como met; exceção manual exige permissão e motivo. |
| T-RV-13 | Request changes | Rodada encerrada, findings persistidos e Task todo sem autoridade. |
| T-RV-14 | Correção tenta mudar definição enquanto review aberta | Negar; exigir retirada/replanning explícitos. |
| T-RV-15 | AdministrativeCompleteWorkItem em v2 sem review | Negar; administração não remove prova exigida. |
| T-RV-16 | Artifact URI aponta para recurso externo | Transação registra referência; não busca URL nem executa conteúdo remoto. |

### OC — Conclusão de Objective

| ID | Cenário | Resultado exigido |
|---|---|---|
| T-OC-01 | Todas as Tasks aceitas, Criterion próprio pendente | Objective permanece In Progress com motivo explícito. |
| T-OC-02 | Última Task aceita e todos os requisitos próprios satisfeitos | Modo automatic produz uma Conclusion e evento de achievement. |
| T-OC-03 | Modo manual elegível | Exibe Ready to achieve; não muda lifecycle até confirmação autorizada. |
| T-OC-04 | Objective sem trabalho direto nem descendente | Não auto-achieve por conjunto vazio. |
| T-OC-05 | Parent sem Task direta mas com trabalho em descendentes | Usa subárvore real e conclusão de filhos, sem considerar o parent vazio. |
| T-OC-06 | Uma Task cancelada e as demais aceitas | Não autofecha; exige decisão manual de exclusão quando permitida. |
| T-OC-07 | Issue própria open/investigating | Impede achievement manual/automático. |
| T-OC-08 | Issue afetando Task do Objective, mas pertencendo a outro | Não inventa gate de ownership; Blocker explícito continua aplicável. |
| T-OC-09 | Duplicate própria aponta para canônica open | Impede achievement até fechamento canônico válido. |
| T-OC-10 | Blocker aplicável permanece ativo após Issue resolvida | Objective continua bloqueado até liberação explícita. |
| T-OC-11 | Dependência hard não satisfeita | Não conclui; cancelled no alvo não conta como sucesso. |
| T-OC-12 | Último Criterion é avaliado depois de todas as Tasks | Reavalia e fecha quando a política permite. |
| T-OC-13 | Última pendência é dependência/Blocker/Issue | Evento correspondente provoca reavaliação dos alvos corretos. |
| T-OC-14 | Dois últimos aceites concorrentes | Uma Conclusion de Objective, sem eventos duplicados. |
| T-OC-15 | Criar Task concorre com autofechamento | Ou Task entra antes e impede conclusão, ou parent já fechado recusa criação. |
| T-OC-16 | Lista UI carrega somente parte das Tasks | Gate consulta universo completo; não fecha pela página carregada. |
| T-OC-17 | ReopenObjective e Start com assessments históricos | Não refecha; novo ciclo precisa de avaliação válida. |
| T-OC-18 | Tentativa de chamar auto-achieve como executor | Nenhum endpoint privilegiado permite assumir a identidade interna. |
| T-OC-19 | Cascata supera 64 conclusões | Persistir pendência e retomar deterministicamente sem declarar resultados não gravados. |
| T-OC-20 | Restart ou evento repetido durante reconciliação | Retomar sem perder candidato ou duplicar conclusão. |
| T-OC-21 | Todos os Objectives obrigatórios Achieved | Outcome permanece não aceito até sua decisão explícita. |
| T-OC-22 | Optional Objective para Outcome | Não se torna required_for_outcome por agregado ou contagem visual. |

### DP — Dependências e grafo

| ID | Cenário | Resultado exigido |
|---|---|---|
| T-DP-01 | A depende de B | Persistir source=A,target=B, com texto da UI equivalente. |
| T-DP-02 | Editar Required by a partir de B | Revalidar direitos/editabilidade do source A. |
| T-DP-03 | Task hard-depende do próprio Objective | Negar sem mutação. |
| T-DP-04 | Task hard-depende de ancestral que precisa daquela Task | Negar deadlock semântico mesmo sem ciclo explícito simples. |
| T-DP-05 | Objective hard-depende de descendente que não pode iniciar | Negar pela combinação hierarchy/lifecycle. |
| T-DP-06 | Advisory não satisfeito | Exibir contexto, não bloquear execução. |
| T-DP-07 | Batch remove/add só é válido no grafo final | Aplicar atomicamente quando resultado final é válido. |
| T-DP-08 | Batch com uma edge inválida ou source sem permissão | Rollback de todas as alterações. |
| T-DP-09 | Duas inserções opostas em processos distintos | No máximo uma efetivada; grafo permanece acíclico. |
| T-DP-10 | Alterar outgoing dependency de Task fora de Backlog | Negar sem Defer automático. |
| T-DP-11 | Impact query truncada por limite | Retornar partial coverage, não total de impacto fictício. |
| T-DP-12 | Cursor de impacto após revisão mudar | Invalidar/requerer refresh; não juntar snapshots incoerentes. |

### IS — Issues e Blockers

| ID | Cenário | Resultado exigido |
|---|---|---|
| T-IS-01 | Issue sem Objective no body | Negar pelo domínio/API; UI exige owner. |
| T-IS-02 | Issue criada a partir de Task | Owner corresponde ao Objective da Task, explicitamente mostrado. |
| T-IS-03 | Affected refs de múltiplos Objectives | Owner único permanece estável. |
| T-IS-04 | Duas investigações com mesma expected_version | Uma atribuição vence; a outra recebe conflito. |
| T-IS-05 | Issue assumida pelo operador | Nenhum lease de Task é criado. |
| T-IS-06 | Resolver Issue sem liberar Blocker | Blocker continua ativo. |
| T-IS-07 | Resolver e liberar subset de Blockers | Somente os explicitamente confirmados são liberados. |
| T-IS-08 | Report com múltiplos alvos, um inválido | Nenhuma Issue/Blocker parcial persiste. |
| T-IS-09 | wont_fix sem reason/resolution_summary | Negar. |
| T-IS-10 | Duplicate com ciclo ou alvo fora do Outcome | Negar sem encobrir pendência. |
| T-IS-11 | Reabrir canônica que sustenta conclusão de outro owner | Exigir reabertura/impacto explícito; não invalidar silenciosamente. |
| T-IS-12 | Vínculo relates_to Issue–Task | Não muda readiness sem Blocker/dependência real. |

### TR — Transportes e CLI

| ID | Cenário | Resultado exigido |
|---|---|---|
| T-TR-01 | Mesmo comando por REST histórico e catálogo | Mesmo serviço, validação, evento e resultado sem bypass. |
| T-TR-02 | HTTP/MCP/SDK/CLI sob mesma identidade/estado | Mesma decisão de autorização e domínio. |
| T-TR-03 | uint64 acima de 2^53 em novo wire | Preservar decimal exato; JS não arredonda versão/fencing. |
| T-TR-04 | Body com campos de identidade/estado não permitidos | Schema nega ou mapeia estritamente; não mass assignment. |
| T-TR-05 | Consulta GET de completion/verification | Não altera lifecycle nem dispara conclusão por leitura. |
| T-TR-06 | CLI work finish retorna delivered_for_review | Não reporta Task accepted como se fosse equivalente. |
| T-TR-07 | CLI signed sem signing key válida | Não tenta executar comando legado equivalente. |
| T-TR-08 | CLI troca profile após intenção pendente | Não reenvia a intenção com outra identidade/destino. |
| T-TR-09 | CLI create key e arquivo secreto já existe | Não sobrescreve por default; falha segura e recibo correto. |
| T-TR-10 | Secret-output falha após emissão confirmada | Relatar emissão realizada e necessidade de substituir; não supor rollback remoto. |
| T-TR-11 | Comando aplicado, refresh falhou | Cliente repete read, não novo write. |
| T-TR-12 | Revogação antes do replay de recibo | Autorização revalidada antes de divulgar resultado. |

### UX — Interface, navegação e acessibilidade

| ID | Cenário | Resultado exigido |
|---|---|---|
| T-UX-01 | Add no Outcome | Abre criação de Objective, nunca Task filha direta. |
| T-UX-02 | Add em Objective/Tasks Kanban | Abre Task com owner correto. |
| T-UX-03 | Criação na lista global | Exige escolher Objective; não escolhe primeiro item automaticamente. |
| T-UX-04 | Criação incompleta em Backlog | Salva quando campos mínimos existem e mostra o que falta para ativar. |
| T-UX-05 | Erro de validação/permissão/conflito | Preserva texto, seleção e intenção; mostra remédio aplicável. |
| T-UX-06 | DnD antes da confirmação | Card permanece na lane persistida. |
| T-UX-07 | Toque ou clique sem drag | Permite a mesma operação pelo inspector. |
| T-UX-08 | Teclado, Escape e browser Back | Foco e contexto retornam corretamente; uma modal por vez. |
| T-UX-09 | Troca de Outcome enquanto read antigo retorna | Não mostra dados/ações do contexto anterior. |
| T-UX-10 | Rotação/revogação com reserva ativa | Explica que posse não é transferida e não a altera implicitamente. |
| T-UX-11 | Última checkbox de critério | Não equivale automaticamente a aceitar a entrega da Task. |
| T-UX-12 | Concluído automaticamente | Timeline identifica política/causa e não afirma aceite do Outcome. |
| T-UX-13 | Somente parte dos cards carregada | Rótulo de cobertura parcial e contagem correta. |
| T-UX-14 | 320/390px e zoom 400% | Operações essenciais acessíveis, sem botões cortados ou sobreposição impeditiva. |
| T-UX-15 | Onboarding e identidade editorial | Sem alteração não autorizada em login, tokens, assets e fluxo inicial. |

### MG — Migração, restauração e geração

| ID | Cenário | Resultado exigido |
|---|---|---|
| T-MG-01 | Fresh install na última migration | Todos os schemas/constraints/policies novos presentes. |
| T-MG-02 | Upgrade com Task/Issue órfã | Parar contract migration; não adivinhar owner. |
| T-MG-03 | Lease antigo sem prova da credencial | Drenar/intervir; não atribuir a única chave atual. |
| T-MG-04 | Task ativa sob Planned no legado | Preview identifica e requer decisão explícita antes de migrar perfil. |
| T-MG-05 | Writer antigo depois do cutover | Recusar escrita incompatível. |
| T-MG-06 | Restore com revisão/entrega/lease pendente | Manter autoridade, versões e retomada coerentes. |
| T-MG-07 | Restore de fatos assinados históricos | Digests/envelopes continuam verificáveis sem transformação. |
| T-MG-08 | Gerador PostgreSQL depois de alteração SQL fonte | Derivados consistentes e commitados; zero-diff após regenerar. |
| T-MG-09 | PG tests sem DSN | Registrar não qualificado; não aprovar E74/E75/E82 por skip. |
| T-MG-10 | Manifest da qualificação | SHA, comandos e limitações reais; sem reutilizar verde de outra versão. |

### Rastreabilidade dos requisitos do proprietário

| Requisito | Implementação principal | Qualificação |
|---|---|---|
| Objective Planned não libera execução | E04–E13 | LC, TR, UX; gates em todos os protocolos. |
| Tasks passam por entrega/verificação antes do aceite | E27–E34, E55, E65 | RV, AU, TR, UX. |
| Objective conclui automaticamente quando realmente elegível | E35–E40, E66 | OC, LC, MG; concorrência em E74. |
| API Keys com permissões granulares | E14–E21, E53, E60–E62 | AU, LS, TR, UX. |
| Claim/lease vinculado à chave, com renew/release/expiry | E22–E26, E55, E63 | LS, AU, MG; processos reais. |
| Criação clara e progressiva | E56–E59 | UX, LC; onboarding em E77. |
| Dependências compreensíveis sem novas subentidades | E10, E46–E48, E67–E68 | DP, UX, LC. |
| Add respeita Outcome → Objective → Task | E41, E56–E59 | UX, LC, IS. |
| Issues integradas e ownership correto | E41–E45, E69–E70 | IS, OC, TR. |
| Backend é autoridade para UI/API/CLI/MCP/SDK | E07–E16, E49–E55, E73–E76 | Todas as matrizes pertinentes. |

<a id="16-comandos-de-verificacao"></a>
## 16. Comandos de verificação

### 16.1 Ambiente e uso seguro

Os comandos abaixo devem ser executados **na cópia de desenvolvimento autorizada**, com a toolchain fixada pelo repositório. Não foram executados durante a criação deste documento. Verificar `go.mod`, `.go-version`, arquivos npm e workflow atual antes de usar.

Não usar banco de produção como DSN de testes. A variável de teste observada no CI é **`WOS_TEST_POSTGRES_DSN`**, não `WOS_POSTGRES_DSN` da ferramenta de manutenção de ownership. Não confundir os dois contextos. O CI também usa `WOS_TEST_POSTGRES_CONTAINER` em testes que necessitam do container real.

```bash
# Confirmar contexto; não descartar alterações preexistentes.
git status --short
git branch --show-current
git rev-parse HEAD
cat .go-version
go version

# Formatação e suites de desenvolvimento.
gofmt -l packages/wos-core packages/wos-api packages/wos-cli packages/wos-sdk-go
go vet ./...
go test -count=1 ./packages/wos-core/domain ./packages/wos-core/application

# Banco real de teste, configurado fora do histórico versionado.
# export WOS_TEST_POSTGRES_DSN='<DSN de banco descartável dedicado>'
go test -p 1 -count=1 ./packages/wos-core/storage/sqlite ./packages/wos-core/storage/postgres

go test -p 1 -count=1 ./...
go test -p 1 -race -count=1 ./...
```

`gofmt -l` verifica; aplicar `gofmt -w` somente aos arquivos pertinentes da alteração. Não fazer uma reformatação massiva do repo para ocultar o diff operacional.

### 16.2 Geração canônica observada no CI

```bash
python3 tools/signedgen/generate.py
gofmt -w packages/wos-core/storage/memory/signed_work_contracts.go \
  packages/wos-core/storage/sqlite/signed_work_contracts.go
python3 tools/postgresgen/generate.py
go run ./tools/mcpgen
go run ./tools/openapigen
gofmt -w packages/wos-core/storage/postgres \
  packages/wos-api/mcp packages/wos-api/commands packages/wos-sdk-go
python3 tools/eventcatalog/check.py
```

Após revisar e comitar mudanças legítimas dos derivados, executar a geração novamente e exigir:

```bash
git diff --exit-code -- \
  packages/wos-core/storage/memory/signed_work_contracts.go \
  packages/wos-core/storage/sqlite/signed_work_contracts.go \
  packages/wos-core/storage/postgres \
  packages/wos-api/mcp packages/wos-api/commands \
  packages/wos-sdk-go packages/wos-api/commands.openapi.json
```

Se um arquivo novo precisa ser gerado, incluí-lo no gerador e no check. Não excluir caminho do zero-diff para esconder divergência.

### 16.3 Testes web

`tests/web/package.json` define `test:unit` e `test` no baseline. Usar o lockfile existente, sem atualização oportunista de dependências.

```bash
cd tests/web
npm ci
npm run test:unit
npx playwright test
# O script npm test também executa units + Playwright.
```

A preparação de browser/backend deve seguir a configuração Playwright atual do repo. Não presumir binário ou porta que não foram verificados. Executar com serviço e fixtures isolados, nunca contra a instância pessoal ativa do usuário.

### 16.4 Registro de resultados

Para cada comando registrar duração observada, código de saída, SHA, banco/protocolo e caminho de log. O tempo medido de um teste não é estimativa de entrega do projeto.

Conservar falhas intermediárias e o commit que as corrige. Não aumentar timeout, remover assert, forçar click, inserir retry automático de UI ou pular PostgreSQL apenas para obter verde. Se uma configuração estava realmente errada, registrar o diagnóstico e a correção legítima.

### 16.5 Checks adicionais do baseline a preservar

Preservar os checks existentes de signed codec, YAML restrito, backup/restore, full-disk cleanup, central-agent-workflow, schema generation, event catalog e arquivo central de planejamento. O workflow já registra jornadas nativas com review independente; eles devem continuar passando com as novas regras, ajustando fixtures para criar critérios e iniciar Objectives na ordem legítima.

Não avançar a versão de release nem publicar npm/imagens durante essa qualificação. A nova HEAD deve ter seus próprios checks hospedados; `statuses=[]` ou falta de um check não equivalem a sucesso.

<a id="17-integracao-com-o-plano-existente-de-44-etapas"></a>
## 17. Integração com o plano existente de 44 etapas

### 17.1 Regra de integração

Este plano **consolida e amplia** o anterior; não exige refazer 44 commits ou abrir outro PR. Na E01, reconciliar o que está realmente concluído. Um item antigo só pode ser marcado complete quando o respectivo contrato, código e aceite foram demonstrados na SHA pertinente.

O baseline já contém partes importantes de ownership, Issue gate, dependências em lote e migrations. Classificá-las como reaproveitamento/qualificação, não como funcionalidade inexistente. Os gaps de lifecycle do Objective, granularidade por chave e auto-completion são incrementos adicionais explícitos.

### 17.2 Crosswalk operacional

| Etapas do plano anterior | Tratamento neste plano |
|---|---|
| 01–04: baseline, decisões, inventário e auditoria | E01–E03, E06 e E75. Preservar evidências; atualizar inventário, não reexecutar writes antigos. |
| 05–17: ownership/assignment e adaptações iniciais | E09, E23, E41–E42 e E75. Confirmar funções existentes, completar integração e políticas novas. |
| 18: gate de Issues do Objective | E35 e E43; reutilizar no fechamento manual e automático. |
| 19–21: cobertura, fechamento independente e multi-Blockers | E40, E43–E44, E69–E70. |
| 22: dependency impact | E48 e E67–E68. Completar transporte e UX, não outro motor de grafo. |
| 23: dependency batch | E46–E47 e E59. Acrescentar nova autorização e deadlocks semânticos. |
| 24: associação corretiva Issue–Task | E45. Preservar relates_to canônico. |
| 25–26: ownership obrigatório em storage | E01 e E75. Migrações 0035/0036 já presentes: verificar, não recriar. |
| 27: API/OpenAPI | E49, mais geração incremental nos commits de feature. |
| 28: HTTP | E50 e E76. |
| 29: MCP | E51 e E76. |
| 30: SDK Go | E52 e E76. |
| 31: CLI | E53–E55 e E76. |
| 32: read model de Issues | E69. |
| 33: contexto Issues no modal | E69–E71. |
| 34: cards/filtros de Issues | E69–E70. |
| 35: inspector/assignment de Issue | E42 e E70. |
| 36: DnD/decisão de fechamento de Issue | E69–E72. |
| 37: criação de Issue com múltiplos impactos | E44 e E70. |
| 38: UX source/target de dependências | E67. |
| 39: modo conectar no Canvas | E68. |
| 40: editor de dependências em lote | E47 e E67. |
| 41: remover criação/placement inválidos no Outcome | E09, E56–E59. Confirmar o título/escopo exato do item antigo na E01. |
| 42: concorrência e storage | E74–E75. |
| 43: E2E/onboarding/acessibilidade | E72, E77 e E83. |
| 44: documentação/paridade/aceite | E76 e E79–E84. |

### 17.3 Trabalho que este plano acrescenta

Gates de Objective/ancestrais; prevenção de esperas impossíveis com hierarchy; permissões explícitas e granulares; emissão atômica de chave restrita; binding de lease e recibo à credencial; revisão nativa onde antes existia apenas conclusão direta; autofechamento com delegação/ciclo e reconciliação; políticas de migração; interface dedicada de API Keys.

Esses incrementos não devem ficar falsamente escondidos como “ajuste visual” da etapa antiga. Atualizar o roadmap, ADRs e notas de compatibilidade para mostrar a alteração de escopo real.

<a id="18-entrega-rollout-e-aceite-final"></a>
## 18. Entrega, rollout e aceite final

### 18.1 Gates de liberação interna

**Gate A — contrato definido:** ADRs, catálogo de permissões e políticas sem ambiguidades; baseline/crosswalk reconciliados.

**Gate B — integridade no Core:** ownership, lifecycle, autoridade, revisão e conclusão não têm bypass. Queries/capabilities explicam o mesmo contrato, sem executar mutações.

**Gate C — persistência e identidade:** migrações reais, revogação, restart/restore, precisão de versões e fencing, provenance e idempotência qualificados.

**Gate D — clientes integrados:** UI, HTTP, MCP, SDK e CLI completos; nenhuma camada usa estado local para aceitar trabalho.

**Gate E — produto utilizável:** onboarding preservado; entrypoints contextuais; revisão, dependências, Issues e chaves operáveis; alternativas ao drag existentes.

**Gate F — qualificação final:** CI da HEAD atual, logs e evidências por cenário; diferenças entre aceites automatizados e humanos declaradas. PR continua Draft.

### 18.2 Rollout técnico recomendado

1. Implantar schema aditivo e leituras compatíveis em ambiente de teste; novas políticas ainda não anunciadas como prontas.
2. Executar audit read-only da instalação e produzir mapping/reconciliation explícitos.
3. Atualizar/graduar grants e CredentialPolicies sem ampliação não autorizada; preparar perfis separados.
4. Drenar leases/contratos incompatíveis, migrar ownership quando necessário e qualificar restauração.
5. Atualizar serviço, CLI/SDK e schema coordenadamente; bloquear writers antigos incompatíveis.
6. Migrar cada Outcome aberto mediante preview e versão; configurar completion modes conscientemente.
7. Habilitar `operational_v2` para novas criações, após gates de implantação concluídos.
8. Acompanhar erros de precondição, conflitos, latência de guard e reconciliações pendentes no ambiente de validação.
9. Submeter o PR em Draft ao proprietário com evidências. Merge/publicação dependem de autorização separada.

### 18.3 Definition of Done

- Toda Task e Issue nova possui Objective válido; nenhum Add promete hierarquia inexistente.
- Objective Planned bloqueia a liberação/execução das Tasks conforme o perfil novo, em todos os transportes e protocolos.
- Task v2 só chega a Accepted com definição, entrega, assessments e revisão legítimos.
- O titular da execução é distinto de assignee; duas chaves diferentes do mesmo Principal não compartilham lease por acidente.
- API Keys nascem restritas, podem ser administradas e não têm segredo recuperável no storage.
- Autofechamento depende de política delegada, fatos completos e ciclo atual; não inventa prova e não aceita Outcome.
- Issues, Blockers e dependências mantêm significados independentes, integrados à prontidão e fechamento corretos.
- UI opera comandos reais, preserva drafts/navegação e tem alternativa ao arraste.
- HTTP/MCP/SDK/CLI expõem os mesmos efeitos/gates; signed-v2 não ganhou fallback sem assinatura.
- Migrações, concorrência, restart, restore e geradores foram verificados nos ambientes pertinentes.
- ROADMAP, ADRs, schemas e documentação descrevem o código real; resultados não executados estão explicitamente pendentes.
- Nenhum segredo, credencial real ou dado privado de operação foi versionado.
- PR permanece aberto/Draft; nenhuma release/tag/publicação foi executada.

Se a implementação automatizada estiver completa, mas faltarem testes manuais de dispositivo/compreensão, a entrega deve declarar **implementação qualificada automaticamente; aceite humano pendente**. Não marcar a evidência manual como passada nem descrever o conjunto como aceito integralmente.

### 18.4 Artefatos esperados da execução futura

Código e testes na branch do PR; ADRs e contrato atualizados; migrations e runbook; manifest E01–E84; mapeamento do plano de 44 etapas; matriz de testes com resultado; logs por SHA; screenshots/reprodução sem segredos; exemplos de CLI e SDK efetivamente executáveis; checklist de onboarding; resumo final de mudanças e pendências concretas.

Este `.md` é o planejamento de todos esses artefatos, não prova de que já foram produzidos no repositório.

<a id="19-instrucao-de-execucao-para-agentes"></a>
## 19. Instrução de execução para agentes

O bloco abaixo pode acompanhar este documento ao iniciar a implementação:

```text
Implemente o plano operacional do WOS contido neste documento.

1. Leia AGENTS.md, ROADMAP.md, o contrato canônico, ADRs e as skills locais
   de UX aplicáveis. Confirme a HEAD real da branch do PR #116.
2. Execute E01 antes de criar módulos. Reutilize o que já existe na branch;
   reconcilie o plano anterior de 44 etapas com o código e as evidências.
3. Continue na branch fix/workspace-operational-density e mantenha o PR Draft.
   Não faça merge, release, tag, publicação ou version bump.
4. Preserve onboarding, identidade editorial, contratos assinados, receipts,
   idempotência, precisão de versões e a separação entre WOS e runtime externo.
5. Siga E01–E84 e as dependências. Cada etapa deve produzir uma alteração
   semântica revisável, testes pertinentes, documentação/geração necessária
   e evidência vinculada à SHA. Não gere commits vazios para etapas já prontas.
6. Implemente regras na Domain/Application. UI, HTTP, MCP, SDK e CLI apenas
   consomem essas regras. Nenhum status setter, autorização por substring,
   owner inferido, assinatura fictícia ou fallback legado para signed-v2.
7. Não edite arquivos gerados como fonte de verdade. Atualize os geradores
   e a implementação fonte, regenere e verifique zero-diff.
8. Não invente resultado de teste, cobertura, benchmark ou estudo de UX.
   Skip e ambiente indisponível são evidência pendente, não sucesso.
9. Em conflito de contrato, escolha a regra normativa deste plano já aprovada
   pelo proprietário e registre o ajuste no ADR; não silencie a diferença.
   Em mudança de HEAD, reconcilie o trabalho existente antes de continuar.
10. Ao terminar cada etapa, atualize ROADMAP e manifest com o estado real.
    Ao final, entregue resumo, matriz de testes, migrações, comandos e limites
    qualificados. O PR deve continuar Draft.
```

### 19.1 Template de evidência por etapa

```markdown
# E## — Título

- Baseline/head antes:
- Commit da etapa:
- Perfil/protocolo/banco:
- Arquivos alterados:
- Comportamento implementado:
- Requisitos e cenários cobertos:
- Testes executados e códigos de saída:
- Resultados medidos:
- Falhas encontradas e correção:
- Testes não executados e motivo:
- Compatibilidade/migração:
- Reflexos no plano anterior:
- Próxima etapa elegível:
- Estado real: planned / in_progress / implemented / verified / blocked
```

`implemented` e `verified` não são sinônimos. A evidência pode registrar uma etapa implementada enquanto sua qualificação depende de ambiente específico, sem apresentar esse estado como conclusão integral.

### 19.2 Manifest proposto

```json
{
  "schema_version": 1,
  "repository": "A1b3rt0M3rcad0/wos",
  "pull_request": 116,
  "branch": "fix/workspace-operational-density",
  "baseline_sha": "2aab252129f548ab54ad4232095e1bc83d76d372",
  "execution_head_sha": null,
  "pr_must_remain_draft": true,
  "stages": [
    {
      "id": "E01",
      "state": "planned",
      "commit_sha": null,
      "evidence_path": "docs/implementation/operational-consolidation/evidence/E01.md",
      "tests_run": [],
      "tests_not_run": [],
      "previous_plan_steps": [1, 2, 3, 4]
    }
  ]
}
```

O exemplo contém somente E01 para ilustrar o contrato; a execução deve criar as 84 entradas. Campos nulos significam não executado, não resultado omitido por conveniência.

<a id="20-fontes-e-referencias"></a>
## 20. Fontes e referências

### 20.1 Fontes de implementação

As fontes de código abaixo pertencem à SHA fixa usada no baseline. A consulta inicial desta resposta confirmou o PR aberto/Draft e essa HEAD. O planejamento diferencia observação de código, decisão normativa nova e qualificação que ainda precisa ocorrer.

- **[R01]** [PR #116](https://github.com/A1b3rt0M3rcad0/wos/pull/116).

- **[R02]** [Contrato de desenvolvimento](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/AGENTS.md).

- **[R03]** [Estado de implementação registrado](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/ROADMAP.md).

- **[R04]** [Planejamento anterior de 44 etapas](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/docs/ux/operational-kanban/issue-and-dependency-plan.md).

- **[R05]** [Readiness de Task e Objective](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/domain/readiness.go).

- **[R06]** [Comandos de criação, ativação, aquisição e conclusão](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/application/service.go).

- **[R07]** [Criação atômica e validação de parent](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/application/planning_creation.go).

- **[R08]** [Autorização atual por comando](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/application/authorization.go).

- **[R09]** [Emissão e administração de credenciais](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/application/security_admin.go).

- **[R10]** [Políticas de credencial e aceitação](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/domain/signing_identity.go).

- **[R11]** [Task, WorkLease e lifecycle](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/domain/work_item.go).

- **[R12]** [Renovação e reclaim](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/domain/work_item_lease.go).

- **[R13]** [Binding e revisão assinada](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/domain/signed_review.go).

- **[R14]** [Material e WorkSubmission](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/domain/work_submission.go).

- **[R15]** [CompletionPolicy strict_work](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/application/completion_policy.go).

- **[R16]** [Precondições puras de Objective](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/domain/workspace_preconditions.go).

- **[R17]** [Pipeline transacional e idempotência](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/application/pipeline.go).

- **[R18]** [Dependências, preview e batch](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/application/dependencies.go).

- **[R19]** [Impacto de dependências](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/application/dependency_impact.go).

- **[R20]** [Issues, ownership e resolução](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/application/issues_blockers.go).

- **[R21]** [Formulários humanos](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-api/web/assets/human-forms.js).

- **[R22]** [Ações e contextos de UI](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-api/web/assets/actions.js).

- **[R23]** [Entry points, navegação e submissão de UI](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-api/web/assets/app.js).

- **[R24]** [Preparação de operações no Kanban](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-api/web/assets/kanban-operations.js).

- **[R25]** [Rotas HTTP existentes](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-api/http/handler.go).

- **[R26]** [Catálogo de comandos CLI](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-cli/help.go).

- **[R27]** [Dispatch do workspace CLI schema-2](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-cli/workspace_commands_v2.go).

- **[R28]** [Gerador PostgreSQL e sua fonte SQLite](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/tools/postgresgen/generate.py).

- **[R29]** [Workflow de qualificação](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/.github/workflows/ci.yml).

- **[R30]** [Scripts de testes web](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/tests/web/package.json).

- **[R31]** [Skill local de auditoria de UX](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/.agents/skills/laws-of-ux-auditor/SKILL.md).

- **[R32]** [ADR de ownership e Issues](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/docs/adr/0042-objective-owned-issues-and-mandatory-work-placement.md).

- **[R33]** [Migração existente de Task Objective obrigatório](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/storage/postgres/migrations/0035_required_work_item_objective.sql).

- **[R34]** [Migração existente de Issue Objective obrigatório](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/packages/wos-core/storage/postgres/migrations/0036_issue_objective_owner_required.sql).

- **[R35]** [Congelamento de planejamento e Kanban modal](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/docs/adr/0041-focused-kanban-modal-and-backlog-planning.md).

- **[R36]** [Política de conclusão e compatibilidade existente](https://github.com/A1b3rt0M3rcad0/wos/blob/2aab252129f548ab54ad4232095e1bc83d76d372/docs/adr/0039-contextual-kanbans-and-strict-completion.md).

### 20.2 Referências externas primárias consultadas

- **[E01]** [W3C — Understanding WCAG 2.2, Dragging Movements](https://www.w3.org/WAI/WCAG22/Understanding/dragging-movements.html). Fundamenta a necessidade de alternativa por ponteiro sem arraste; acessibilidade de teclado é uma obrigação distinta. Não estabelece que uma captura visual prove conformidade.
- **[E02]** [OWASP — Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html). Fundamenta menor privilégio, negação por default, validação em cada requisição e testes de autorização. O catálogo e os presets específicos deste plano são decisões de desenho para o WOS, não prescrições literais da OWASP.
- **[E03]** [PostgreSQL 18 — Explicit Locking](https://www.postgresql.org/docs/18/explicit-locking.html). Fundamenta coordenação por locks transacionais e ordem consistente de aquisição. Os locks concretos do WOS devem seguir os adapters/pipeline existentes e ser qualificados com concorrência real.

### 20.3 Limites desta entrega

Foi revisada a implementação real disponível pela conexão GitHub e mantida uma SHA explícita como baseline. Não foram executados testes do repositório, migrations, servidor, navegador ou benchmarks durante a elaboração deste arquivo. Os novos tipos, arquivos, rotas, permissões e comandos estão identificados como propostas de implementação; não são funcionalidades anunciadas como prontas.

**Conclusão do planejamento:** evoluir o WOS existente, completar seus gaps de integração e tornar o backend a autoridade uniforme para o trabalho compartilhado. A implementação termina com evidências verificáveis e PR Draft, não com uma alegação de conclusão baseada somente em código criado.
