# WOS — Plano de implementação de contratos de execução, CLI e workspace YAML

**Versão do plano:** 1.0  
**Data:** 8 de outubro de 2026  
**Repositório:** `A1b3rt0M3rcad0/wos`  
**Baseline inspecionada:** `master`, commit `bcd1714ad7cded666890d0b5c9f711772c4f8d7a`  
**Natureza:** especificação e planejamento de implementação; não representa funcionalidade entregue.  
**Escopo desta entrega:** somente este documento. Não foram modificados código, branches, PRs ou configurações do repositório.

> **Decisão central:** uma WorkItem pode ter, no máximo, um contrato de execução válido. A reserva criada pelo contrato termina exclusivamente por **expiração sem renovação**, **revogação administrativa** ou **finalização aceita pelo WOS**. O WOS é a fonte canônica; o YAML é uma representação local, não uma concessão autônoma de autoridade.

## Índice

1. [Resultado esperado e decisões vinculantes](#s01)
2. [Baseline real e diferenças a implementar](#s02)
3. [Arquitetura e fronteiras de responsabilidade](#s03)
4. [Modelo de domínio proposto](#s04)
5. [Máquinas de estado e invariantes](#s05)
6. [Lease, expiração, renovação e fencing](#s06)
7. [Autorização, titularidade e escopo do bloqueio](#s07)
8. [Versões, snapshots e mudanças de planejamento](#s08)
9. [Aquisição e retomada transacionais](#s09)
10. [Progresso, checkpoints e sincronização](#s10)
11. [Entrega, revisão e finalização](#s11)
12. [Persistência, índices e eventos](#s12)
13. [Contratos HTTP, MCP e SDK](#s13)
14. [Modelo de arquivos YAML](#s14)
15. [Parsing, validação e integridade](#s15)
16. [CLI: comandos e experiência operacional](#s16)
17. [Workspace local, journal e recuperação](#s17)
18. [Renovação no cliente e execução desconectada](#s18)
19. [Fluxos completos de uso](#s19)
20. [Planejamento, Roadmaps e desenvolvimento local](#s20)
21. [Interface humana e skills](#s21)
22. [Compatibilidade e migração](#s22)
23. [Distribuição e plataformas](#s23)
24. [Mapa de alterações no repositório](#s24)
25. [Plano de execução por ondas e PRs](#s25)
26. [Matriz de testes](#s26)
27. [Desempenho e observabilidade](#s27)
28. [Critérios de aceite e rastreabilidade](#s28)
29. [Decisões consolidadas e extensões posteriores](#s29)
30. [Instruções para o agente implementador](#s30)
31. [Fontes e limites da verificação](#s31)

---

<a id="s01"></a>
## 1. Resultado esperado e decisões vinculantes

### 1.1 O produto que será entregue

Evoluir o WOS de coordenação por WorkItems e leases para um modelo em que cada aquisição produz um **WorkContract persistente, auditável, versionado e materializável localmente**. Esse contrato reúne a autorização temporária de execução, o snapshot da especificação do trabalho e as referências necessárias para executar e comprovar a entrega.

O modelo deve atender a dois cenários independentes:

| Cenário | Consumidor | Interface principal | Onde o trabalho é executado |
|---|---|---|---|
| Sistemas agentic | Woobe e outros runtimes | MCP, HTTP ou SDK | No runtime e nas ferramentas do consumidor |
| Desenvolvimento local | Codex, Claude Code, humanos e outros agentes de programação | CLI e arquivos YAML | No workspace e nas ferramentas do agente |

Esses cenários **não precisam compartilhar servidor, Namespace, Outcome, planejamento ou executor**. Interoperabilidade é uma capacidade do protocolo, não uma obrigação de uso combinado.

### 1.2 Decisões já fixadas pelo usuário

- O WOS permanece independente da Woobe.
- O WOS armazena, valida e coordena o trabalho; não executa a tarefa e não inicia agentes.
- O contrato adquirido reserva a execução de uma WorkItem exclusivamente para seu titular.
- As únicas saídas da reserva são expiração, revogação administrativa e finalização validada.
- A expiração pode ser evitada por renovação autorizada antes do vencimento.
- Atualizar progresso, sincronizar arquivos ou registrar um checkpoint não finaliza o contrato.
- Um contrato vencido ou revogado não pode concluir a tarefa posteriormente.
- O YAML é opcional: consumidores programáticos não precisam gerar arquivos.
- A mesma regra é aplicada no Core, independentemente da interface utilizada.

### 1.3 Escolhas técnicas deste plano

As escolhas abaixo são propostas de implementação, não decisões preexistentes do repositório: entidade `WorkContract`; checkpoint estruturado; submissão imutável de resultado; pacote `wos-cli`; executável cliente `wosctl`; schemas locais com `schema_version`; separação entre versão do contrato e versão do lease; comandos compostos de aquisição e sincronização.

Nomes de arquivos e divisão fina de PRs podem ser refinados. Alterar as três formas de encerramento, permitir dois contratos válidos, tornar o YAML fonte de verdade ou fazer o WOS executar agentes exigiria nova decisão arquitetural.

### 1.4 Experiência-alvo

```text
Planejar no WOS
    ↓
Descobrir ou receber uma WorkItem
    ↓
Adquirir contrato no servidor
    ↓
Receber JSON por MCP/HTTP ou materializar YAML pela CLI
    ↓
Executar fora do WOS, renovar e registrar checkpoints
    ↓
Submeter resultado e provas
    ↓
Avaliar critérios, quando necessário
    ↓
Finalizar explicitamente no WOS
```

A operação de checkout deve devolver contexto suficiente para começar a tarefa sem reproduzir toda a conversa de planejamento ou carregar todo o Outcome.

---

<a id="s02"></a>
## 2. Baseline real e diferenças a implementar

A branch consultada aponta para o commit informado no cabeçalho. O commit corresponde à integração da preparação de versão 0.1.0; isso não constitui, por si só, comprovação de publicação em registries ou aceite operacional desta nova proposta. [R01]

### 2.1 Capacidades encontradas

| Área | Comportamento observado na baseline | Tratamento no novo modelo |
|---|---|---|
| WorkItem | Agregado com lifecycle, critérios, conclusão e `CurrentLease` | Preservar identidade e lifecycle; tornar o contrato a autoridade da execução |
| Lease | `claim_id`, Principal, Actor, fencing, aquisição e expiração | Reutilizar a semântica, incorporada ao WorkContract |
| TTL | Default de 300 s; mínimo de 30 s; máximo de 3.600 s | Preservar esses defaults iniciais e tornar a política explícita/configurável |
| Claim | Só aceita WorkItem `todo`; incrementa fencing e entra em `in_progress` | Emitir contrato e snapshot na mesma transação |
| Renew | Exige titular, claim e fencing; não renova lease vencido | Preservar; separar CAS do lease do conteúdo de progresso |
| Reclaim | Substitui lease vencido e mantém `in_progress` | Fechar o contrato anterior como expirado e emitir outro |
| Release | O titular pode liberar a tarefa, retornando-a a `todo` | Remover essa possibilidade no protocolo novo |
| Complete | Exige lease válido e critérios/resultados | Finalizar contrato e WorkItem atomicamente, com submissão identificável |
| Administração | Cancelamento e conclusão administrativos específicos | Introduzir revogação explícita; impedir bypass do contrato pelo caminho antigo |
| Concorrência | Outcome guard, versões e idempotência | Preservar, estendendo a novos agregados e comandos |
| Tempo | PostgreSQL usa autoridade do banco nos serviços autorizados | Preservar em todos os novos caminhos |
| HTTP/MCP | Comandos convergem para Application; catálogo tipado | Ampliar o catálogo, sem regras de domínio nos adapters |
| SDK Go | Cliente HTTP fino, sem retries automáticos de mutações | Ampliar DTOs, consultas e métodos, mantendo a política de retry explícita |
| CLI existente | `wos` administra servidor, configuração, MCP stdio e banco | Não confundir com uma CLI cliente de checkout/sync, que ainda precisa ser criada |
| Skills | Orientam claim/renew/evidence/complete e ainda mencionam release | Atualizar para contratos e para as três saídas permitidas |

Fontes: `work_item.go`, `work_item_lease.go`, `lease_service.go`, `administrative_work.go`, contratos públicos, SDK, entrypoint e skill de coordenação. [R02–R09, R12–R13]

### 2.2 Diferenças que não podem ficar implícitas

**A nova política não é apenas um formato de exportação.** Ela altera comportamento de domínio: o titular deixa de poder liberar voluntariamente a reserva. Implementar somente YAML e CLI, deixando `Release` acessível por HTTP/MCP/Core, não entregaria o modelo acordado.

**A conclusão administrativa atual precisa ser revisada.** Hoje há um caminho que substitui verificações de titularidade/expiração, embora mantenha obrigações de prova. No protocolo novo, não se deve usar esse comando para aceitar um contrato vencido ou revogado. O administrador revoga e, quando necessário, um executor autorizado adquire um novo contrato para concluir. [R05]

**O SDK não é atualmente desacoplado de todos os tipos internos.** O cliente inspecionado importa tipos de `wos-api/commands`, Application e Domain. Seu comportamento continua sendo de cliente HTTP. Não descrever a introdução da CLI como se um pacote de DTOs totalmente independente já existisse. [R08]

**A expiração não é uma transição automática de lifecycle da WorkItem.** A implementação de reclaim já preserva `in_progress`; isso deve continuar. A tarefa pode estar em andamento, mas sem contrato válido, e então ser recuperável mediante aquisição explícita. [R03]

### 2.3 Forma de documentar a evolução

Adicionar um novo conjunto de ondas ao `ROADMAP.md`, sem reescrever o aceite histórico das Waves 01–18. Criar ADRs próprios para as mudanças de semântica. A hierarquia declarada no `AGENTS.md` coloca instruções atuais do proprietário acima de decisões anteriores, mas exige registrar as divergências e manter o roadmap fiel à execução. [R10–R11]

---

<a id="s03"></a>
## 3. Arquitetura e fronteiras de responsabilidade

```text
                       WOS independente
┌───────────────────────────────────────────────────────────┐
│ HTTP / MCP / cliente humano                               │
│                         ↓                                 │
│ Application: adquirir, renovar, sincronizar, finalizar     │
│                         ↓                                 │
│ Domain: Outcome, WorkItem, WorkContract, critérios e prova │
│                         ↓                                 │
│ Ports → Memory / SQLite / PostgreSQL                      │
│                         ↓                                 │
│ Estado atual + histórico + eventos + outbox                │
└───────────────────────┬───────────────────────────────────┘
                        │ API pública
             ┌──────────┴────────────┐
             │                       │
     Woobe / outro runtime       wosctl / wos-cli
       MCP / HTTP / SDK             │
             │                 .wos/ + YAML
     Ferramentas do Agent            │
                            Codex / Claude / humano
```

### 3.1 O que pertence a cada componente

**Core:** autorização de domínio, invariantes, aquisição, TTL, fencing, checkpoints, submissões, conclusão, versões e atomicidade.

**HTTP/MCP:** autenticação, envelopes, limites, JSON Schema, tradução de erros e exposição dos mesmos casos de uso.

**SDK:** transporte, deadlines, erros tipados, precondições e helpers opt-in de lease. Não decide qual tarefa executar nem altera automaticamente a intenção de uma mutação.

**CLI:** configuração de conexão, serialização YAML, materialização de workspace, diff de resultados, journal de requisições, renovação explicitamente iniciada e mensagens adequadas a agentes.

**Runtime/agente:** raciocínio, escolha do trabalho dentro do escopo permitido, ferramentas, execução de testes, criação de código, supervisão de subprocessos e permissões do ambiente.

**WOS UI:** inspeção, revisão, intervenção administrativa autorizada e acompanhamento da mesma fonte canônica.

### 3.2 Exclusões

Não criar scheduler de agentes, motor de shell, sistema de containers, executor de testes, sincronizador Git automático, serviço de armazenamento arbitrário de arquivos, workflow genérico ou dependência do runtime Woobe.

Não transformar o Roadmap do WOS em domínio pedagógico ou estado de um produto consumidor. O WOS organiza o trabalho de seus participantes; cada produto mantém seu próprio domínio de negócio.

Não importar regras do lock de deployment da Woobe. O contrato do WOS trata uma WorkItem, usa Principal autenticado e não equivale a um lock global de projeto, release ou credencial de CI/CD.

### 3.3 Decisão sobre o executável

Criar **`packages/wos-cli` como componente cliente e `wosctl` como executável**. Preservar `wos` como entrypoint de servidor/administração existente. Isso evita dois binários diferentes com o mesmo nome e permite instalar apenas o cliente em Windows ou macOS.

Os exemplos anteriores `wos work checkout` eram ilustrativos. Neste plano, a sintaxe executável proposta é `wosctl work checkout`. Uma unificação futura do front-end não pode alterar os contratos de domínio nem introduzir escrita direta no banco pelo cliente.

---

<a id="s04"></a>
## 4. Modelo de domínio proposto

### 4.1 WorkItem continua sendo o trabalho

A WorkItem preserva seu ID entre planejamento, aquisições, expirações e replanejamentos. Não criar uma nova WorkItem para cada tentativa e não utilizar o contrato como substituto de Objective ou Roadmap.

Mantêm-se os lifecycles `backlog`, `todo`, `in_progress`, `done` e `cancelled`. Disponibilidade e recuperação são projeções, não novos lifecycles livres.

Para alimentar os campos novos do snapshot, acrescentar à WorkItem um value object opcional `ExecutionSpec`, com `instructions`, `constraints`, `deliverables`, `scope_hints` e `context_refs`. Título, descrição, critérios e dependências continuam vindo de suas fontes canônicas existentes, sem cópia gravável paralela. Os comandos de criação/edição passam a aceitar esse value object com validação e versão; alterações materiais durante contrato válido seguem a proteção da seção 8.

WorkItems antigas sem `ExecutionSpec` usam campos vazios e sua descrição existente. Não inferir automaticamente instruções, permissões ou critérios a partir de prosa durante a migração. O planejador pode completar explicitamente a especificação antes da aquisição.

### 4.2 WorkContract torna uma aquisição uma entidade durável

| Campo conceitual | Responsabilidade |
|---|---|
| `id` | Identidade única desta aquisição; emitida pelo servidor |
| `namespace_id`, `outcome_id`, `work_item_id` | Escopo canônico |
| `holder_principal_id` | Titular autenticado; nunca escolhido pelo YAML |
| `actor_ref` | Proveniência declarada/autorizada, distinta da credencial |
| `status` | `active`, `expired`, `revoked` ou `completed` |
| `version` | CAS do conteúdo mutável/estado contratual |
| `lease_version` | CAS específico de renovação e anexação da execução |
| `execution_id` | Identidade da anexação operacional atual |
| `fencing_token` | Geração monotônica da autoridade sobre a WorkItem |
| `acquired_at`, `expires_at`, `last_renewed_at` | Tempos controlados pelo servidor |
| `lease_policy` | Política efetiva de TTL e sua revisão |
| `work_item_version_at_acquire` | Versão observada na aquisição |
| `outcome_revision_at_acquire` | Referência histórica; não CAS global do executor |
| `spec_snapshot`, `spec_digest` | Especificação congelada da tarefa e sua integridade |
| `plan_references` | Roadmap/revisão/nós pertinentes na aquisição |
| `latest_checkpoint_id` | Referência de conveniência ao checkpoint atual |
| `latest_submission_id` | Referência à última submissão; não prova de aceite |
| `closed_at`, `close_reason`, `closed_by` | Auditoria da única transição terminal |
| `legacy_source` | Metadados explícitos quando houver importação de estado antigo |

O contrato é uma **entidade durável com especificação imutável e lease mutável**. Não é necessário que todos os campos sejam uma única tabela física, mas sua autoridade deve ter apenas uma origem.

### 4.3 Uma única autoridade, não dois locks concorrentes

O `WorkLease` existente deve ser refatorado para se tornar o componente de lease do contrato. Durante compatibilidade, a API pode projetar `current_lease` a partir do contrato; não manter dois estados graváveis que possam divergir.

O high-water mark de fencing continua associado à WorkItem e nunca diminui. A tabela de contratos mantém o histórico das aquisições. A exclusividade não depende da presença de um arquivo local nem de uma flag `locked` enviada pelo agente.

### 4.4 WorkCheckpoint

Checkpoint imutável e append-only, vinculado a contrato e WorkItem. Campos mínimos:

- ID, sequência, autor autenticado e instante de aceite;
- resumo do que foi feito e do que falta;
- próxima ação sugerida pelo executor;
- obstáculos e referências a Issues/Blockers;
- referências a artefatos, evidências e à revisão de código/documento trabalhada;
- IDs de progresso/submissão relacionados, quando houver;
- marca explícita de informações desconhecidas ou não verificadas.

Um novo checkpoint não reescreve o anterior. `latest_checkpoint_id` é uma projeção para retomada. Não exigir upload de logs completos ou histórico de chat.

### 4.5 WorkSubmission

Entrega imutável, distinta de progresso. Deve registrar o resultado material proposto, referências exatas dos artefatos, evidências, digest do conteúdo submetido, base contratual e proveniência.

Uma correção produz nova submissão, com `supersedes_submission_id`, sem apagar a anterior. A WorkItem só fica concluída quando uma submissão identificada passa pela finalização.

### 4.6 Assessment e Conclusion

Reutilizar SuccessCriterion, CriterionAssessment e Conclusion existentes. Acrescentar vínculo explícito da avaliação de uma entrega à `submission_id` e à base material avaliada quando o dono do critério for uma WorkItem sob contrato.

Não basta que o critério mantenha o mesmo título/revisão: uma avaliação de um artefato antigo não deve aprovar silenciosamente uma nova entrega materialmente diferente. A conclusão registra a submissão aceita e as avaliações/revisões utilizadas.

### 4.7 Sem novas entidades de runtime

Não criar Agent, Session, Run, Model, Tool ou Conversation internos ao WOS. Identificadores externos dessas entidades, quando úteis, são correlação/proveniência; não concedem autoridade nem substituem Principal e contrato.

---

<a id="s05"></a>
## 5. Máquinas de estado e invariantes

### 5.1 Contrato

```text
             adquirir
                │
                ▼
              active
             /   |   \
            /    |    \
       expired revoked completed
```

Não existem transições de volta para `active`. Renovação e checkpoint são operações dentro de `active`. Uma nova aquisição produz outro ID de contrato.

| Operação | Precondição principal | Estado após aceite |
|---|---|---|
| Acquire | Trabalho elegível e nenhum contrato válido | Novo contrato `active` |
| Renew | Titular/execução/fencing válidos e prazo não vencido | Mesmo contrato `active`, novo vencimento |
| Sync/checkpoint | Contrato ainda válido e CAS compatível | Mesmo contrato `active` |
| Submit | Contrato válido e entrega estruturalmente válida | Mesmo contrato `active`, nova submissão |
| Finalize | Contrato válido, submissão e critérios aceitos | Contrato `completed`; WorkItem `done` |
| Expire | Tempo autoritativo maior ou igual ao vencimento | Contrato `expired`; progresso preservado |
| Revoke | Permissão administrativa, contrato ainda ativo e motivo | Contrato `revoked`; progresso preservado |

### 5.2 Estado da WorkItem após encerramentos

**Expiração:** mantém `in_progress`; a projeção informa `recoverable=true` se as demais condições permitirem. Isso é diferente de afirmar que o trabalho nunca começou.

**Revogação:** mantém `in_progress` por padrão, sem titular válido. Uma alteração administrativa adicional pode cancelar ou replanejar a tarefa, com seu próprio evento e autorização.

**Finalização:** passa para `done`. O encerramento não a devolve à fila.

**Reabertura de trabalho concluído:** mantém a Conclusion anterior em histórico, reabre a WorkItem pelo comando existente e exige novo contrato para uma nova execução. Não reativa o contrato concluído.

### 5.3 Invariantes normativas

| ID | Regra |
|---|---|
| INV-01 | No máximo um contrato efetivamente válido por WorkItem |
| INV-02 | Validade depende de estado, prazo e autorização, não do YAML |
| INV-03 | Somente expiração, revogação e conclusão encerram a reserva |
| INV-04 | O titular não possui `release` no protocolo novo |
| INV-05 | Contrato terminal nunca volta a ativo |
| INV-06 | Nova aquisição recebe novo ID e fencing estritamente superior |
| INV-07 | Fencing antigo não modifica progresso contratual nem conclui trabalho |
| INV-08 | Renovação não cria contrato, não altera especificação e não revive vencidos |
| INV-09 | Checkpoint/sync/submissão não implicam conclusão |
| INV-10 | WorkItem, contrato, conclusão e recibo de finalização são confirmados atomicamente |
| INV-11 | Evidência não é avaliação; avaliação não equivale automaticamente à conclusão |
| INV-12 | WorkItem concluída não conclui automaticamente Objective/Outcome |
| INV-13 | Scope, Principal e permissões são verificados em todos os transports |
| INV-14 | Exclusividade é de execução da tarefa, não de leitura nem de todo o Outcome |
| INV-15 | Resultado local não substitui campos controlados pelo servidor |
| INV-16 | Repetição idempotente recupera a mesma intenção, não cria nova autoridade |
| INV-17 | Expiração não depende de sweeper para invalidar autoridade |
| INV-18 | A aquisição é válida antes da materialização local; falha de disco não a desfaz implicitamente |
| INV-19 | Os critérios e a base material aceitos ficam identificáveis historicamente |
| INV-20 | A CLI não executa comandos arbitrários contidos no YAML |

---

<a id="s06"></a>
## 6. Lease, expiração, renovação e fencing

### 6.1 Política inicial proposta

Preservar os valores existentes como defaults de transição, não como garantia universal de duração adequada:

| Parâmetro | Default inicial | Regra |
|---|---:|---|
| `default_ttl_seconds` | 300 | Usado quando o cliente não solicita outro prazo |
| `min_ttl_seconds` | 30 | Menor prazo aceito |
| `max_ttl_seconds` | 3600 | Maior prazo por renovação, configurável pelo operador |
| Cadência cliente | Aproximadamente TTL / 3 | Renovar antes do limite; usar deadline conservador |
| Duração total máxima | Sem cap adicional por default | Pode ser configurada; nunca confundida com TTL por renovação |

Uma tarefa de quatro horas não requer lease inicial de quatro horas: pode manter um lease de cinco minutos com renovações autorizadas. Um perfil para trabalho local pode admitir TTL maior, mas continua obedecendo ao vencimento real do servidor.

A política é configurada no deployment/Namespace por autoridade administrativa, não por valores ilimitados enviados no YAML. Registrar revisão da política. Mudanças normais de política passam a valer nas próximas renovações/aquisições; encurtar retroativamente um contrato exige intervenção administrativa explícita.

### 6.2 Tempo autoritativo

Definir `valid = status == active AND acquired_at <= now < expires_at`.

No instante `now == expires_at`, o contrato está vencido. O relógio do arquivo, do terminal, do modelo e do computador do agente não decide validade.

Nos serviços autorizados PostgreSQL, obter o tempo do banco após adquirir os guards relevantes, como já estabelece o ADR-016. SQLite e Memory usam o Clock configurado. Ambientes embedded distribuídos devem fornecer autoridade de tempo compartilhada. [R12]

Não realizar validações remotas de artefatos, chamadas de rede ou execução de testes dentro da transação. A decisão de validade deve ocorrer depois de adquirir os locks e junto à mutação protegida, com transação curta.

### 6.3 Expiração efetiva versus registro histórico

A consulta pode projetar `effective_status=expired` mesmo que a linha ainda esteja gravada como `active`, caso o prazo tenha passado. Essa diferença é explícita; não autoriza execução.

A materialização do encerramento ocorre por:

1. próxima aquisição, que fecha o contrato vencido e cria o novo na mesma transação; ou
2. reconciliador interno limitado, que registra expirações pendentes.

Esse reconciliador é manutenção determinística de estado, não scheduler de tarefas. `effective_at` registra o vencimento e `recorded_at` o momento em que o fato foi persistido. Emitir o evento de expiração apenas uma vez.

GETs não devem escrever silenciosamente. A ausência do reconciliador não pode permitir uma renovação ou conclusão tardia.

### 6.4 Renovação

`renew_work_contract` verifica identidade, contrato, execução, fencing, `expected_lease_version` e validade temporal. Avança apenas a versão do lease e o registro correspondente de mudança, sem editar o resultado local ou o snapshot da tarefa.

A mesma chave idempotente repetida devolve a renovação original; não adiciona sucessivos períodos de TTL. Cada nova intenção de renovar usa uma chave nova, persistida antes do envio.

Após resposta incerta, o consumidor precisa reconciliar recibo/estado. Não considerar o lease renovado apenas porque uma requisição foi enviada.

### 6.5 Fencing

Manter um contador monotônico por WorkItem. Verificar overflow em **todos** os caminhos que o incrementam, incluindo a aquisição inicial, e não apenas o reclaim.

Representar tokens nos novos DTOs JSON/YAML como strings decimais canônicas, evitando arredondamento em consumidores JavaScript. Converter para inteiro sem sinal com validação explícita no Core. Na persistência, usar representação exata compatível com todo o domínio admitido; não reduzir silenciosamente um `uint64` legado a `BIGINT` com sinal.

O fencing protege mutações aceitas pelo WOS. Para commits, deploys, pagamentos, arquivos compartilhados ou outros efeitos externos, o consumidor precisa de suas próprias garantias. O servidor não consegue desfazer um efeito realizado fora dele por um processo desconectado.

### 6.6 Retomada de sessão do mesmo titular

Separar **ler/materializar um contrato existente** de **assumir sua execução em outro processo**.

Uma anexação explícita `resume_work_contract`, autorizada ao mesmo Principal e com CAS, pode gerar novo `execution_id` e incrementar o fencing, invalidando o executor anterior sem encerrar o contrato. Não estende o prazo implicitamente. CLI deve exigir `--takeover` quando essa invalidação for solicitada.

Isso não cria reserva permanente separada: continua existindo um único contrato com um lease. Depois do vencimento, `resume` falha e é necessária nova aquisição.

Duas cópias deliberadamente compartilhadas da mesma credencial e da mesma autorização ainda representam a mesma identidade. O WOS não pode descobrir qual processo físico recebeu legitimamente uma cópia; usar identidades separadas, anexação explícita e lock local cooperativo.

---

<a id="s07"></a>
## 7. Autorização, titularidade e escopo do bloqueio

### 7.1 Permissões propostas

Os nomes abaixo são novas permissões conceituais; devem ser integrados ao modelo real de grants, sem criar um segundo sistema de autorização.

| Ação | Permissão e condição adicional |
|---|---|
| Ler tarefa/contrato/checkpoints | Leitura no Namespace e no escopo aplicável |
| Adquirir trabalho | `work.contract.acquire`; escopo elegível e autorizado |
| Renovar/sincronizar/submeter | Permissão de execução e titularidade efetiva |
| Retomar com takeover | Titular atual; CAS explícito e autorização de execução |
| Finalizar | Titularidade efetiva, autorização de conclusão e provas válidas |
| Avaliar critérios | Permissão de avaliação e política de independência |
| Revogar contrato | `work.contract.revoke`; motivo obrigatório |
| Alterar política de lease | Administração do Namespace/deployment |
| Cancelar/reabrir WorkItem | Permissões de lifecycle, sem contornar contratos ativos |

A credencial autentica um Principal. `actor_ref`, `client_id`, nome do agente, sessão, hostname ou dados em YAML não substituem essa identidade. Duas chaves do mesmo Principal não são automaticamente dois titulares distintos.

No modo local confiável com identidade compartilhada, nomes diferentes de agentes não oferecem isolamento de titularidade. Para testar e operar executores independentes, provisionar Principals/credenciais distintos no perfil autenticado apropriado.

### 7.2 O que o lock protege

O lock protege a **execução e as mutações que representam execução daquela WorkItem**: avanço contratual, checkpoints, submissões e finalização.

Outros participantes autorizados continuam podendo ler, revisar, registrar observações, administrar e trabalhar em outras WorkItems. Não bloquear todo o Outcome por toda a duração do contrato. O Outcome guard é um mecanismo transacional curto, não o lease de longa duração da tarefa.

O contrato também não é lock de diretório, branch Git, arquivo ou ambiente de deploy. Dois WorkItems podem tocar o mesmo arquivo; coordenação dessa sobreposição pertence ao planejamento e ao ambiente de execução.

### 7.3 Impedir bypass por comandos existentes

Auditar os casos de uso de atualização da WorkItem, critérios, lifecycle, relações, cancelamento, conclusão administrativa e alterações de Outcome/Objective. A validação deve ocorrer no Application/Domain sob a mesma transação, não apenas no handler de `acquire_work_contract`.

No protocolo novo:

- `release_work_item` não libera contrato ativo, nem quando chamado diretamente por outro transport;
- completar a WorkItem diretamente exige o fluxo contratual ou retorna erro de protocolo;
- cancelar uma tarefa com contrato ativo exige revogação administrativa explícita; uma operação composta pode revogar e cancelar atomicamente, registrando ambos os fatos;
- alterar materialmente especificação/critério/dependência de uma tarefa contratada exige revogação prévia;
- concluir, abandonar ou arquivar um Outcome com contratos ativos afetados deve ser recusado até a resolução explícita desses contratos; não encerrá-los em cascata silenciosamente;
- cancelar Objective com tarefas contratadas segue a mesma proteção.

A revogação de uma credencial ou grant retira acesso, mas não cria um quarto encerramento contratual: a reserva remanescente termina por expiração ou revogação administrativa do contrato.

### 7.4 Impedimentos durante o trabalho

Issues, Blockers, novas evidências e mudanças no estado de dependências podem aparecer durante a execução. Não congelar esses fatos para preservar artificialmente a elegibilidade.

A projeção passa a indicar `execution_allowed=false` quando um impedimento aplicável exige parada. O contrato pode continuar `active` até uma das três saídas. Consultas e renovação informam os motivos; o executor interrompe efeitos externos incompatíveis. A política pode permitir renovação durante espera por revisão/insumo, sem tratar ausência de progresso como conclusão.

### 7.5 ExternalContext e Woobe

ExternalContext é endereçamento/correlação, não autorização. A Woobe ou outro host pode injetar Namespace e contexto autorizado sem expor identificadores internos ao modelo. Um Agent não pode ampliar seu escopo preenchendo outros valores no payload.

No workspace local, IDs podem constar no YAML quando fizerem parte do contexto autorizado do desenvolvedor. Ainda assim, a credencial e as permissões no servidor governam todas as operações. [R06]

---

<a id="s08"></a>
## 8. Versões, snapshots e mudanças de planejamento

### 8.1 Versões distintas

| Marcador | O que protege | O que não representa |
|---|---|---|
| `schema_version` | Formato do documento/DTO | Versão do servidor ou da tarefa |
| `work_item.version` | Intenção de mutar o agregado WorkItem | Geração do lease |
| `contract.version` | Checkpoint/submissão/estado contratual | Número da revisão do Roadmap |
| `lease_version` | Renovação e anexação operacional | Progresso entregue |
| `fencing_token` | Autoridade operacional vigente sobre a WorkItem | Chave de autenticação |
| `outcome_revision` | Ordem transacional do Outcome | Lock global exigido em todo sync |
| `roadmap.revision_number` | Versão publicada do planejamento | Estado atual de execução |
| `spec_digest` | Integridade da especificação congelada | Prova de titularidade |
| `submission_digest` | Identidade do resultado material submetido | Avaliação positiva |

Separar CAS do lease evita que um heartbeat torne obsoleto o resultado que o agente está editando. Isso não elimina o Outcome guard nem o histórico de mutações.

### 8.2 Conteúdo do snapshot contratual

Congelar o enunciado da tarefa, escopo, entregáveis, restrições de trabalho, critérios e suas revisões, dependências declaradas e referências de planejamento necessárias.

Não incorporar o Outcome inteiro por padrão. Evidências anteriores, decisões e documentos extensos entram como referências verificáveis, com metadados suficientes para decidir se precisam ser abertos.

Distinguir claramente:

- **Especificação contratada:** o que foi solicitado nesta aquisição.
- **Estado vivo:** lease atual, situação de dependências, bloqueios e avaliações.
- **Referências históricas:** revisão do Roadmap e dados observados na aquisição.

### 8.3 Replanejamento

Uma nova revisão de Roadmap não cancela por si só tarefas ou contratos. A ordem visual do plano não cria dependências operacionais.

Se a mudança apenas reorganiza o plano, o contrato pode continuar. Se altera uma obrigação material da tarefa, revogar o contrato, modificar explicitamente a WorkItem e emitir nova aquisição. Não permitir que `sync` atualize silenciosamente o contrato para o plano mais recente.

Uma mudança no estado de uma dependência — por exemplo, reabertura de trabalho antes concluído — não altera o snapshot histórico, mas pode impedir execução/finalização até nova avaliação.

### 8.4 Edição de critérios durante execução

Proibir alteração material dos critérios da WorkItem enquanto há contrato válido. Avaliações desses critérios continuam permitidas, respeitando independência e revisão. Para mudar o que deve ser entregue, utilizar revogação e nova especificação.

Critérios de Objective/Outcome não são automaticamente incorporados como obrigações locais de cada WorkItem. Uma alteração nesses níveis não pode ser tratada como reescrita invisível da especificação contratada. A certificação final desses agregados permanece separada.

### 8.5 Contexto omitido

Preservar `truncated`, `omitted`, contagens e cursores explícitos. O contrato deve conter a especificação mínima executável ou indicar `requires_context_expansion=true`; não chamar uma tarefa incompleta de pronta para execução apenas porque existe um claim.

A especificação imutável do contrato pode ser paginada por cursores vinculados ao seu digest. O contexto vivo utiliza revisão/instante da consulta. Não invalidar a paginação de um snapshot imutável apenas por um heartbeat posterior.

---

<a id="s09"></a>
## 9. Aquisição e retomada transacionais

### 9.1 AcquireWorkContract

**Entrada:** Scope, WorkItemID, versão esperada da WorkItem, TTL solicitado, metadados permitidos do consumidor e chave idempotente. Principal e Actor efetivos vêm da autenticação.

**Sequência obrigatória:**

1. Validar autenticação, autorização atual, schema e limites.
2. Entrar na infraestrutura existente de UnitOfWork/idempotência.
3. Adquirir Outcome guard e os registros necessários em ordem fixa.
4. Obter tempo autoritativo depois dos locks.
5. Carregar WorkItem, contexto operacional e contrato relevante.
6. Verificar CAS e elegibilidade: Outcome ativo/não arquivado, Objective permissivo, dependências, Blockers e restrições temporais.
7. Se há contrato válido, rejeitar a aquisição. Não roubar contrato do mesmo Principal implicitamente.
8. Se o contrato anterior expirou, materializar o encerramento, preservando histórico.
9. Incrementar fencing com proteção de overflow.
10. Criar WorkContract e snapshot mínimo coerente na mesma transação.
11. Alterar WorkItem de `todo` para `in_progress`, ou preservar `in_progress` na recuperação.
12. Persistir alterações, evento(s), outbox e recibo; avançar `outcome_revision` conforme a unidade de comando.
13. Confirmar a transação e devolver contrato/contexto, ou recibo de commit se o resultado ultrapassar o limite.

Não chamar publicamente `claim`, fazer uma leitura posterior sem vínculo de revisão e então afirmar que essa leitura foi o snapshot adquirido. Snapshot e autorização precisam ter base transacional identificável.

### 9.2 AcquireNextWorkContract

Adicionar uma operação composta para selecionar e adquirir trabalho elegível em **um Outcome explícito**. Não selecionar agentes e não percorrer todos os tenants.

Filtros iniciais: conjunto autorizado de WorkItem IDs, Objective e prioridades. Ordenação determinística: prioridade, criação e ID, preservando uma política claramente documentada. Ordenação por capacidades de executor fica fora do gate inicial.

Validar elegibilidade novamente na transação de aquisição. Usar busca limitada e índices; não hidratar todo o histórico para escolher um item. Retornar um único contrato ou `acquired=false` com razões de ausência de trabalho.

A resposta `acquired=false` também pertence à intenção idempotente daquela consulta mutante. Para tentar novamente após mudança de estado, criar uma nova intenção/chave; repetir a chave antiga não deve adquirir arbitrariamente outra tarefa.

### 9.3 Recuperação após vencimento ou revogação

O novo executor lê o contexto da WorkItem e o último checkpoint aceito, valida os artefatos disponíveis e adquire um novo contrato. A recuperação revalida todas as condições relevantes, não apenas a ausência de lease válido.

O resultado deve indicar `recovery=true`, contrato anterior e checkpoint sugerido. O consumidor não precisa confiar na narrativa do executor anterior: verifica commit, testes e referências materiais.

### 9.4 Concorrência entre aquisição, renovação e finalização

| Corrida | Resultado exigido |
|---|---|
| Dois acquires da mesma WorkItem | Um contrato válido; o outro recebe conflito ou nenhum trabalho |
| Renew antes do vencimento × acquire de recuperação | Renovação aceita impede reclaim; ordem definida pelo guard e tempo autoritativo |
| Renew depois do vencimento × acquire | Renovação falha; nova aquisição pode vencer |
| Finalize × revoke | Só uma transição terminal aceita |
| Finalize antes de expirar × expire | Conclusão confirmada não é posteriormente convertida em expiração |
| Revoke do contrato A × aquisição B | Revogação não pode atingir B por usar apenas WorkItemID |
| Takeover × sync do processo anterior | A geração antiga não é aceita depois da anexação nova |

Toda intervenção administrativa deve identificar **o contrato específico**, e não apenas a tarefa, para não revogar uma aquisição posterior por engano.

### 9.5 Falha na materialização local

A transação do WOS e a escrita no filesystem não são atômicas entre si. Portanto:

- antes de adquirir, a CLI verifica diretório, permissões, espaço e lock local;
- persiste a intenção/idempotency key antes do envio;
- se o servidor confirma e o disco falha, registra/exibe `acquired_not_materialized` quando possível;
- a recuperação consulta o mesmo recibo e materializa o contrato já adquirido;
- não cria outro contrato e não chama release como compensação;
- se o workspace foi perdido, a descoberta autorizada dos contratos do titular permite reencontrar a aquisição.

---

<a id="s10"></a>
## 10. Progresso, checkpoints e sincronização

### 10.1 SyncWorkContract é um caso de uso composto e limitado

Criar um comando específico que possa, em uma única transação, registrar um checkpoint e um conjunto limitado de Artifact/Evidence/EvidenceLink relacionados à execução.

Ele não é um executor genérico de qualquer comando do WOS. Usar DTOs tipados, operações permitidas explicitamente e os mesmos métodos de domínio existentes. Reutilizar helpers transacionais internos; não chamar vários métodos públicos que abrem transações independentes e anunciar atomicidade inexistente.

**Entrada mínima:** Scope, contract_id, execution_id, fencing, expected_contract_version, spec_digest, checkpoint/progresso, itens documentais e chave idempotente.

**Resultado:** IDs canônicos criados/reutilizados, nova contract_version, checkpoint_id, outcome_revision, command_id e recibo de confirmação.

### 10.2 Identidade local de itens documentais

Usar `local_key` dentro de uma intenção de sincronização para que um checkpoint possa referenciar artefatos/evidências enviados no mesmo payload. O recibo retorna o mapa `local_key → canonical_id`.

Uma chave local não é um ID global. A mesma chave com conteúdo diferente exige nova revisão/intenção; não sobrescreve Evidence imutável. Repetir exatamente o comando recupera os mesmos IDs.

### 10.3 O que pode ser editado localmente

Progresso, notas de execução, próxima ação, referências de artefatos, evidências produzidas e resultado proposto. O usuário pode atualizar esses campos enquanto a CLI não está sincronizando.

O cliente lê um snapshot consistente dos bytes para montar o payload. Mudanças feitas depois dessa leitura pertencem ao próximo sync. O recibo fica associado ao digest dos bytes/intenção efetivamente enviados.

### 10.4 O que não pode ser atualizado pelo result.yaml

Titular, Scope, expiração, fencing, versão esperada arbitrária, definição da tarefa, critérios, estado canônico `done`, grants e revisão do plano. O servidor rejeita campos desconhecidos ou fora da operação; não os ignora silenciosamente.

Uma entrada local `observed_result: passed` registra a alegação/observação produzida pelo executor. Não cria automaticamente um CriterionAssessment `met`.

### 10.5 Sincronização parcial e limites

Manter o teto atual de 256 KiB por comando remoto. Um lote maior deve ser dividido em unidades confirmáveis, com recibos próprios e progresso explícito. Não prometer atomicidade do conjunto de vários lotes.

A finalização referencia apenas IDs cuja persistência foi confirmada. Se o resultado da mutação vier omitido por tamanho, tratar como commit realizado e expandir por consultas autorizadas, como no contrato atual. [R06]

### 10.6 Checkpoint depois da perda de autoridade

Depois de expirar/revogar, `sync_work_contract` do contrato antigo falha. A CLI preserva o trabalho local como `unsent` e apresenta instrução de reconciliar/adquirir nova autoridade.

Observações históricas eventualmente úteis podem ser registradas por um caminho documental autorizado, com proveniência de contrato encerrado, sem alterar o progresso aceito desse contrato e sem liberar dependências. Não transformar esse caminho em uma conclusão tardia disfarçada.

---

<a id="s11"></a>
## 11. Entrega, revisão e finalização

### 11.1 Três operações distintas

| Operação | Significado | Encerra o contrato? |
|---|---|---|
| Sync/checkpoint | Salvar andamento e referências | Não |
| SubmitWorkResult | Congelar uma proposta de entrega verificável | Não |
| FinalizeWorkContract | Aceitar uma submissão e concluir a WorkItem | Sim, se todas as condições passarem |

Essa separação permite revisão independente sem inventar o lifecycle `done_pending_review` para a WorkItem. A projeção da entrega pode indicar `awaiting_review` ou `ready_to_finalize`, mas o contrato continua ativo e sujeito a vencimento.

### 11.2 SubmitWorkResult

Validar autoridade, CAS, base contratual, documentos e referências confirmadas. Persistir uma submissão imutável. Retornar os critérios ainda não avaliados, avaliações incompatíveis e próximos passos determinísticos.

Quando forem necessárias correções, uma nova submissão substitui a anterior apenas como referência corrente. Não reutilizar avaliações da versão anterior sem um vínculo de prova compatível e explícito.

### 11.3 Avaliação

O revisor consulta a submissão, verifica os artefatos no commit/versão exatos e registra avaliações no modo definido por cada critério.

A política de revisor independente deve continuar sendo aplicada com base na identidade autenticada. Não criar uma nova opção de CLI que contorne essa política nem aceitar autoavaliação pelo fato de o usuário editar o YAML.

O WOS valida a estrutura e a consistência das referências/avaliações. A constatação de que testes realmente passaram vem do avaliador ou evidência externa apropriada; não de executar testes no servidor WOS.

### 11.4 FinalizeWorkContract

Em uma única transação:

1. Verificar autorização atual e recuperar idempotência confirmada quando aplicável.
2. Adquirir guards e ler tempo autoritativo.
3. Validar contrato ativo, titular, execução, fencing e versões esperadas.
4. Confirmar Scope, spec_digest e a submissão exata a finalizar.
5. Validar lifecycle do Outcome/Objective/WorkItem e impedimentos/dependências atuais.
6. Verificar critérios obrigatórios na revisão correta, prova utilizável e política de avaliação.
7. Construir Conclusion com submission_id, assessments, obrigações e autoria.
8. Alterar WorkItem para `done` e contrato para `completed`.
9. Persistir eventos/outbox/recibo e confirmar.

Critérios ausentes ou negativos não liberam o lock. O resultado permanece submetido, mas o contrato só termina quando houver aceite, vencimento ou revogação.

### 11.5 Atalho ergonômico `finish`

`wosctl work finish` pode executar uma sequência explícita: sincronizar os itens pendentes, submeter a entrega e tentar finalizar. Cada mutação tem sua chave e recibo. O comando imprime o ponto alcançado e não afirma conclusão se ficou esperando revisão.

Uma entrega sem critérios obrigatórios ainda exige resumo e as precondições aplicáveis. Não inventar um caminho automático de aprovação de critérios para fazer `finish` sempre retornar sucesso.

### 11.6 Revisão que demora mais que o lease

O titular pode continuar renovando enquanto aguarda a revisão, desde que o consumidor tenha escolhido manter a reserva. Se o contrato expirar, a submissão não desaparece, mas ninguém pode finalizar usando a autoridade antiga.

Um novo titular adquire contrato e pode criar uma submissão que referencia o resultado anterior como origem. A aceitação exige base de especificação e avaliações compatíveis, explicitamente verificadas. A posse não é transferida pelo simples fato de existir uma entrega pendente.

### 11.7 Resposta perdida após conclusão

Se a conclusão foi confirmada e a resposta se perdeu, repetir a mesma chave/intenção retorna o recibo original, mesmo que já tenha passado o prazo que o lease teria. Isso é recuperação de uma operação passada, não uma nova finalização fora do prazo.

A autorização atual para ler o recibo continua sendo verificada. Um usuário cujo acesso foi revogado não recebe resultados históricos protegidos apenas por conhecer uma idempotency key.

---

<a id="s12"></a>
## 12. Persistência, índices e eventos

### 12.1 Novas estruturas propostas

| Estrutura | Conteúdo |
|---|---|
| `work_contracts` | Identidade, titularidade, lifecycle, versões, lease e fechamento |
| `work_contract_specs` | Snapshot imutável, digest e referências de aquisição |
| `work_contract_checkpoints` | Checkpoints append-only e sequência |
| `work_contract_submissions` | Entregas imutáveis, digests e supersessão |
| Vínculo de assessments | Submission/base material avaliada quando aplicável |
| Metadados de protocolo do Namespace | Estado de migração e protocolo ativo |

Artefatos e evidências continuam nas estruturas existentes. Não criar uma cópia paralela de todo o modelo documental para cada contrato.

Para fencing, a proposta física é `NUMERIC(20,0)` com domínio validado em PostgreSQL e string decimal canônica em SQLite, com validação exata no adapter/Domain. Não ordenar tokens lexicalmente para decidir autoridade. O contador da WorkItem e o valor no contrato devem usar a mesma faixa sem sinal e rejeitar incremento acima do máximo suportado.

`work_contract_specs` pode ser incorporada à tabela principal se o desenho físico justificar; o contrato de imutabilidade é obrigatório. Não guardar o YAML bruto como fonte canônica de regras: persistir dados tipados/JSON normalizado e gerar a representação na borda.

### 12.2 Unicidade e acesso

Criar unicidade para um registro aberto por `(namespace_id, outcome_id, work_item_id)` e índices para histórico por tarefa, contratos do titular e contratos por vencimento.

O índice de unicidade não pode usar o relógio como predicado dinâmico. Uma linha `active` vencida é fechada transacionalmente antes de inserir a nova. A validade temporal continua sendo verificada no Domain/Application.

Adicionar chaves estrangeiras e validações de escopo para impedir vínculo de checkpoint, submissão ou evidência de outro Namespace/Outcome.

### 12.3 Versões e gravações

Renovação grava o componente de lease e avança `lease_version`; checkpoint/submissão avançam `contract.version`; alterações de lifecycle avançam as versões dos agregados afetados.

Todos os caminhos continuam integrados ao Outcome guard e ao mecanismo de Domain Events/outbox existente. Uma renovação não deve regravar todo o snapshot, histórico de checkpoints ou critérios da WorkItem.

### 12.4 Eventos propostos

- `work_contract.acquired`
- `work_contract.renewed`
- `work_contract.execution_resumed`
- `work_contract.checkpoint_recorded`
- `work_contract.result_submitted`
- `work_contract.expired`
- `work_contract.revoked`
- `work_contract.completed`

Mapear esses fatos ao catálogo público versionado de Integration Events. Não expor nomes de structs, paths locais, credenciais ou logs privados no envelope público.

Um único comando pode registrar múltiplos fatos ordenados dentro da mesma `outcome_revision`, como expirar A e adquirir B, ou concluir WorkItem e contrato. Documentar essa unidade explicitamente.

### 12.5 Revogação e expiração

Revogação registra Principal administrativo, motivo, contrato exato, geração invalidada e instante. Tentativas de revogar um contrato já terminal não devem alterar sua causa de encerramento.

O administrador pode retirar autoridade de um contrato em escopo existente mesmo quando o trabalho deixou de ser elegível, inclusive em estados legados/restaurados inconsistentes. Não exigir que a tarefa volte a estar pronta para permitir uma revogação. Se o prazo já venceu, preservar expiração como causa efetiva, em vez de reclassificar o passado como revogação.

O reconciliador de expiração usa lotes limitados, ordem estável de locks e identidades idempotentes para evitar eventos duplicados em múltiplas réplicas. Expiração retroativa após conclusão é proibida.

### 12.6 Adapters e backup

Implementar a mesma semântica em Memory, SQLite e PostgreSQL. Atualizar UnitOfWork, repositories, migrations, geradores existentes, fixtures, backup e restore.

O restore deve preservar IDs, specs, high-water mark de fencing, conclusões, recibos e causas de encerramento. Um lease restaurado com prazo já vencido permanece sem autoridade. Não somar o tempo de indisponibilidade ao prazo, a menos que exista uma nova operação autorizada — que não pode renovar um contrato já vencido.

---

<a id="s13"></a>
## 13. Contratos HTTP, MCP e SDK

### 13.1 Mutações novas

Preservar a convenção atual de catálogo de comandos: HTTP em `POST /api/v1/commands/{command_name}` e MCP com `wos_{command_name}`. Os nomes seguintes são propostos, não existentes na baseline. [R06]

| Command name | Função |
|---|---|
| `acquire_work_contract` | Adquirir uma tarefa identificada |
| `acquire_next_work_contract` | Selecionar/adquirir uma elegível no escopo explícito |
| `renew_work_contract` | Renovar antes do vencimento |
| `resume_work_contract` | Anexar nova execução do mesmo titular, com fencing novo |
| `sync_work_contract` | Registrar progresso/checkpoint/documentos tipados |
| `submit_work_result` | Persistir entrega imutável |
| `finalize_work_contract` | Aceitar entrega e concluir tarefa/contrato |
| `revoke_work_contract` | Revogar administrativamente um contrato específico |

Não expor `release_work_contract`. Um eventual pedido de devolução é solicitação/Issue, não encerramento automático.

### 13.2 Consultas novas ou ampliadas

| Consulta | Conteúdo mínimo |
|---|---|
| Capabilities | Protocolos, schemas, limites e features suportadas |
| GetWorkContract | Estado atual, validade efetiva, versões e autorização operacional |
| GetWorkContractSpec | Snapshot imutável e digest |
| ListWorkContracts | Por WorkItem/titular/status, com paginação |
| GetContractCheckpoint / ListCheckpoints | Continuidade focal |
| GetWorkSubmission / ListSubmissions | Entregas verificáveis e estado de revisão derivado |
| GetCommandReceipt | Recuperação de intenção incerta, com autorização atual |
| GetWorkContext ampliado | Contrato, checkpoint, submissão e motivos operacionais |
| ListAvailableWork | Trabalho inicial e recuperável, sem perder distinção de lifecycle |

Adicionar rotas REST sob `namespaces/{namespace_id}/outcomes/{outcome_id}/work-contracts`. O namespace de API permanece `/api/v1` enquanto essa for a versão real do transport; não confundir com `schema_version: 1` do arquivo.

### 13.3 Envelope de comando ilustrativo

```json
{
  "command": {
    "scope": {
      "namespace_id": "0199d000-0000-7000-8000-000000000001",
      "outcome_id": "0199d000-0000-7000-8000-000000000010"
    },
    "work_item_id": "0199d000-0000-7000-8000-000000000024",
    "expected_work_item_version": 7,
    "ttl_seconds": 300,
    "consumer": {
      "kind": "local_cli",
      "label": "codex-workspace"
    }
  }
}
```

No HTTP, `Idempotency-Key` fica no header. No MCP, usar o envelope atual com `idempotency_key` e `command`. IDs e valores do exemplo são fictícios; nomes finais de DTO devem ser gerados e testados a partir do contrato tipado.

### 13.4 Erros acionáveis

| Código proposto | Significado | Ação do cliente |
|---|---|---|
| `work_already_claimed` | Há contrato válido | Não executar; escolher outra tarefa ou encerrar |
| `work_not_eligible` | Dependência, blocker, lifecycle ou prazo impede aquisição | Ler razões e replanejar |
| `contract_expired` | Prazo acabou | Parar mutações; reconciliar e adquirir novamente quando permitido |
| `contract_revoked` | Autoridade removida administrativamente | Parar; não fazer takeover automático |
| `stale_execution` | Execution ID/fencing antigo | Parar processo antigo e ler estado |
| `contract_version_conflict` | Progresso/estado mudou | Comparar base, local e servidor |
| `lease_version_conflict` | Renovação/anexação concorrente | Ler estado; não alterar a intenção cegamente |
| `contract_spec_mismatch` | Arquivo/base não corresponde à especificação emitida | Recarregar spec, preservando resultado local |
| `submission_not_accepted` | Critérios/prova ainda não permitem finalizar | Exibir obrigações exatas |
| `idempotency_conflict` | Mesma chave com intenção diferente | Não reenviar alterando a chave sem reconciliação |
| `contract_protocol_required` | Comando legado incompatível com Namespace migrado | Atualizar o cliente |
| `unsupported_schema_version` | Formato desconhecido | Interromper antes de qualquer mutação |
| `payload_too_large` | Limite ultrapassado | Dividir itens documentais explicitamente |

Preservar os erros existentes quando tiverem a mesma semântica. Não criar códigos redundantes apenas para renomear `version_conflict`. Os códigos finais devem ter mapeamento consistente HTTP/MCP/SDK/CLI e fixtures geradas.

### 13.5 SDK e clientes

Adicionar métodos tipados para todos os comandos/queries. Manter os envelopes públicos em snake_case e TTL em segundos. Não introduzir retries implícitos de mutações no cliente básico.

Um helper opt-in de renovação pode existir, mas deve receber contexto cancelável, política explícita, callback de perda de autoridade e armazenamento de intenção adequado ao consumidor. SDK e CLI não podem trocar a versão esperada automaticamente para fazer qualquer mutação passar.

Gerar catálogo HTTP/MCP/OpenAPI/SDK a partir da mesma inscrição tipada. Schemas de arquivos locais devem compartilhar os tipos de documento, mas não expor campos de autenticação nem detalhes do servidor.

---

<a id="s14"></a>
## 14. Modelo de arquivos YAML

### 14.1 Organização do workspace

```text
.wos/
├── config.yaml                         # Configuração compartilhável, sem segredos
├── .gitignore                          # Ignora estado operacional local
└── work/
    └── <work_item_id>/
        └── <contract_id>/
            ├── contract.yaml           # Snapshot emitido pelo servidor; somente leitura lógica
            ├── checkpoint.yaml         # Próximo checkpoint proposto localmente
            ├── result.yaml             # Entrega proposta pelo executor
            ├── state.json              # Últimos valores observados; gerenciado pela CLI
            ├── workspace.lock          # Exclusão cooperativa do cliente local
            ├── outbox/                 # Intenções pendentes, com payload exato e chave
            └── receipts/               # Recibos de comandos confirmados
```

Usar `.yaml` por padrão e aceitar `.yml` como extensão equivalente. Isso não muda o schema nem a semântica. Não manter dois arquivos concorrentes `result.yaml` e `result.yml` para a mesma finalidade: recusar ambiguidade ou exigir seleção explícita.

O checkout gera `contract.yaml`, `checkpoint.yaml`, `result.yaml` e estado inicial. Ele não escreve na configuração do Codex/Claude nem instala MCP automaticamente.

### 14.2 Configuração local

**Schema proposto; os endereços e IDs abaixo são exemplos.** A identidade do servidor e do Namespace deve ser confirmada ao inicializar o workspace. A referência de credencial aponta para o ambiente, não inclui o segredo.

```yaml
schema_version: 1
kind: WOSWorkspace
profile: development
connection:
  server_url: "https://wos.example.test"
  credential_ref: "env:WOS_DEV_TOKEN"
scope:
  namespace_id: "0199d000-0000-7000-8000-000000000001"
  default_outcome_id: "0199d000-0000-7000-8000-000000000010"
workspace:
  root: ".wos/work"
  format: yaml
lease:
  requested_ttl_seconds: 300
output:
  default_format: text
```

O arquivo não é suficiente para autorizar acesso. Mudança de `server_url` ou Namespace não pode reaproveitar silenciosamente credenciais/receipts de outro destino. O cliente valida o binding antes de enviar qualquer intenção pendente.

### 14.3 contract.yaml — snapshot da aquisição

O documento mostra a autorização **na emissão**, não um relógio vivo. Renovações e takeover são refletidos no estado do servidor e em `state.json`. A CLI não utiliza cegamente `initial_fencing_token` depois de uma retomada.

```yaml
schema_version: 1
kind: WorkContract
metadata:
  contract_id: "0199d000-0000-7000-8000-000000000100"
  namespace_id: "0199d000-0000-7000-8000-000000000001"
  outcome_id: "0199d000-0000-7000-8000-000000000010"
  work_item_id: "0199d000-0000-7000-8000-000000000024"
  issued_at: "2026-10-08T17:00:00Z"
  spec_digest: "sha256:29168be4ad687d46f8ddfde44bea9e16e3f891fba9aff873efdc7836cb7aa088"
  snapshot_complete: true
grant_at_issue:
  holder_principal_id: "developer-alberto"
  execution_id: "0199d000-0000-7000-8000-000000000101"
  initial_fencing_token: "41"
  expires_at: "2026-10-08T17:05:00Z"
  ttl_seconds: 300
base:
  work_item_version: 8
  outcome_revision: 132
  plan_references:
    - roadmap_id: "0199d000-0000-7000-8000-000000000050"
      revision_number: 3
      node_key: implement-authentication
spec:
  title: "Implementar autenticação JWT"
  description: >-
    Implementar emissão e validação de tokens conforme a arquitetura
    aprovada, preservando os contratos existentes da API.
  objective:
    id: "0199d000-0000-7000-8000-000000000020"
    statement: "Disponibilizar autenticação verificável"
  instructions:
    - "Implementar emissão e validação dos tokens."
    - "Criar testes para token expirado e assinatura inválida."
  constraints:
    - "Não alterar o contrato público de cadastro."
    - "Não inserir segredos ou chaves privadas no repositório."
  scope_hints:
    permitted_paths:
      - "internal/auth/**"
      - "tests/auth/**"
  dependencies:
    - work_item_id: "0199d000-0000-7000-8000-000000000023"
      required_condition: done
  criteria:
    - id: "0199d000-0000-7000-8000-000000000201"
      revision: 1
      title: "Tokens expirados são rejeitados"
      required: true
      verification_mode: evidence_review
    - id: "0199d000-0000-7000-8000-000000000202"
      revision: 1
      title: "Testes de autenticação aprovados"
      required: true
      verification_mode: evidence_review
  deliverables:
    - "Commit ou PR identificável e acessível ao revisor."
    - "Evidência de testes vinculada à mesma revisão de código."
  context_refs:
    - label: "Decisão sobre autenticação"
      entity_kind: decision
      entity_id: "0199d000-0000-7000-8000-000000000301"
```

`scope_hints` orienta o executor, mas não implementa sandbox. A aplicação efetiva de permissões de arquivos e ferramentas pertence ao host. O digest deste exemplo cobre o objeto `spec` segundo a regra da seção 15.

### 14.4 checkpoint.yaml — proposta de continuidade

```yaml
schema_version: 1
kind: WorkCheckpointDraft
contract_id: "0199d000-0000-7000-8000-000000000100"
spec_digest: "sha256:29168be4ad687d46f8ddfde44bea9e16e3f891fba9aff873efdc7836cb7aa088"
summary: "Emissão de tokens implementada; validação ainda incompleta."
completed:
  - "Implementação inicial da emissão de tokens."
pending:
  - "Rejeição de tokens expirados."
  - "Testes de integração."
next_action: "Implementar verificação de expiração e executar os testes."
workspace_state:
  repository_ref: "https://git.example.test/team/backend"
  base_commit: "1111111111111111111111111111111111111111"
  working_commit: "2222222222222222222222222222222222222222"
  dirty: true
artifact_refs: []
evidence_refs: []
issue_refs: []
notes: "Alterações ainda não commitadas não estão transferidas ao WOS."
```

`dirty: true` é relevante: outro executor não recebe os arquivos alterados apenas por ler esse checkpoint. Para handoff remoto, o resultado parcial precisa estar em um commit, patch ou artefato durável acessível. O WOS armazena a referência, não faz upload implícito do workspace.

### 14.5 result.yaml — entrega proposta

Os campos documentais abaixo pertencem ao novo schema local; o mapper da CLI deve convertê-los para comandos/DTOs tipados do servidor. `observed_result` não é um assessment.

```yaml
schema_version: 1
kind: WorkResult
contract_id: "0199d000-0000-7000-8000-000000000100"
spec_digest: "sha256:29168be4ad687d46f8ddfde44bea9e16e3f891fba9aff873efdc7836cb7aa088"
summary: "Emissão e validação JWT implementadas, com testes de autenticação."
artifacts:
  - local_key: auth-implementation
    title: "Implementação de autenticação"
    artifact_type: git_commit
    uri: "https://git.example.test/team/backend/commit/3333333333333333333333333333333333333333"
    source_version: "3333333333333333333333333333333333333333"
evidence:
  - local_key: auth-tests
    title: "Resultado dos testes de autenticação"
    evidence_type: test_result
    uri: "https://ci.example.test/runs/456"
    source_version: "3333333333333333333333333333333333333333"
    observed_result: passed
    relates_to_artifact: auth-implementation
criterion_evidence:
  - criterion_id: "0199d000-0000-7000-8000-000000000201"
    criterion_revision: 1
    evidence_keys:
      - auth-tests
  - criterion_id: "0199d000-0000-7000-8000-000000000202"
    criterion_revision: 1
    evidence_keys:
      - auth-tests
unresolved_issue_refs: []
```

Não incluir `status: done`, `approved: true`, `expires_at` editável ou `admin_override` nesse documento. A intenção de finalizar é um comando explícito, não um campo com autoridade no arquivo.

### 14.6 state.json — estado observado pela CLI

```json
{
  "workspace_schema_version": 1,
  "server_url": "https://wos.example.test",
  "contract_id": "0199d000-0000-7000-8000-000000000100",
  "observed_contract_version": 3,
  "observed_lease_version": 5,
  "execution_id": "0199d000-0000-7000-8000-000000000101",
  "fencing_token": "41",
  "effective_status": "active",
  "expires_at": "2026-10-08T17:10:00Z",
  "evaluated_at": "2026-10-08T17:05:00Z",
  "latest_checkpoint_id": "0199d000-0000-7000-8000-000000000401",
  "last_confirmed_command_id": "0199d000-0000-7000-8000-000000000501"
}
```

Mesmo esse arquivo é apenas cache. Depois de reiniciar o consumidor, a CLI consulta o estado vivo antes de autorizar continuação. Nunca considerar o arquivo uma prova de lease vigente.

### 14.7 Política de Git

Versionar opcionalmente `.wos/config.yaml` sem credenciais e instruções de integração aprovadas. Ignorar `work/`, state, outbox e receipts por padrão.

Contratos podem conter informações sensíveis de trabalho. Exportá-los para Git deve ser uma ação explícita, sem tokens e com revisão do conteúdo. Git não substitui o histórico canônico do WOS.

---

<a id="s15"></a>
## 15. Parsing, validação e integridade

### 15.1 Perfil YAML definido pelo produto

Adotar um subconjunto explícito de YAML 1.2, tendo a revisão 1.2.2 como referência de sintaxe. O produto impõe limites adicionais para documentos contratuais; isso não significa suportar todos os recursos da linguagem. [E01]

Requisitos do parser da CLI:

- UTF-8, um documento por arquivo e raiz do tipo mapping;
- chaves string, duplicatas rejeitadas e campos desconhecidos rejeitados;
- rejeitar tags customizadas, aliases/anchors e merge keys neste perfil;
- não executar interpolação de shell, includes, templating ou carregamento arbitrário;
- booleanos/números com regras determinísticas; IDs, tokens e timestamps tratados como strings conforme schema;
- limitar profundidade, contagem de nós, tamanho de string e bytes antes de alocar estruturas ilimitadas;
- mensagens com path do campo e linha/coluna quando disponíveis;
- aceitar `.yaml` e `.yml` com a mesma semântica.

Proposta de limites iniciais: 256 KiB por documento operacional, profundidade máxima 16 e limite agregado de itens documentais por comando coerente com o teto remoto. Os valores são parâmetros de produto a validar com fixtures, não medições do sistema atual.

Selecionar e fixar a versão da biblioteca Go de YAML no primeiro PR de parser, após verificar esses requisitos. A baseline de `go.mod` não inclui uma biblioteca YAML direta para essa função. Não inserir parsing YAML no Domain. [R16]

### 15.2 Digest

Definir `spec_digest = "sha256:" + hex(SHA256(JCS(spec)))`, usando canonicalização JSON conforme RFC 8785. Aplicar a mesma regra a objetos explicitamente definidos de submissão. Não calcular sobre os bytes YAML, pois comentários e formatação não são o conteúdo semântico contratado. [E02]

Fixar o conjunto de campos coberto e validar com vetores interoperáveis. Usar números dentro do domínio suportado; valores de grande precisão, como fencing, são strings nos novos documentos. O serializer JSON padrão sem verificação de canonicalização não é, por si só, comprovação de conformidade com JCS.

Digest detecta divergência, mas não autentica o titular. Mesmo que o agente recalcule um hash depois de editar o arquivo, o servidor compara com o contrato canônico já emitido.

Para `submission_digest`, canonicalizar um objeto tipado composto por `contract_id`, `work_item_id`, `spec_digest`, `summary`, referências canônicas de artifacts e suas versões/checksums, Evidence IDs e vínculos critério/revisão/evidência. Excluir ID gerado da própria submissão, timestamp de persistência e campos de transporte. Resolver `local_key` antes desse cálculo. Definir defaults no schema: coleções vazias são `[]`, e ausência não é convertida arbitrariamente em `null`. Preservar a ordem das listas; a canonicalização de objetos não transforma listas em conjuntos.

### 15.3 Proteção do workspace

Resolver paths relativos dentro da raiz configurada, recusar escape por `..` e tratar symlinks/reparse points de forma explícita. Não escrever fora do workspace por um path vindo do servidor ou de um arquivo não confiável.

Aplicar escrita atômica por arquivo usando temporário no mesmo filesystem, flush apropriado e rename/substituição compatível com a plataforma. Tratar falha entre os arquivos como materialização incompleta recuperável, não como transação distribuída perfeita.

Arquivo local editado não pode redirecionar o envio de uma credencial para outro servidor sem nova confirmação do binding de confiança. URLs de artifacts são referências: WOS/CLI não devem buscar automaticamente qualquer endereço ou executar conteúdo encontrado nele.

---

<a id="s16"></a>
## 16. CLI: comandos e experiência operacional

Todos os comandos desta seção são propostos para `wosctl`. Não confundir com funcionalidades já disponíveis no binário `wos` da baseline.

### 16.1 Catálogo mínimo

| Comando | Semântica |
|---|---|
| `wosctl init` | Criar workspace e confirmar servidor/Namespace |
| `wosctl auth status` | Mostrar identidade/escopo resolvidos, sem expor token |
| `wosctl capabilities` | Inspecionar versão de protocolo, schemas e limites |
| `wosctl outcome list/get/create` | Consultar/criar Outcomes pelos contratos existentes |
| `wosctl objective list/get/create` | Gerenciar planejamento básico com autorização |
| `wosctl work list/get/create` | Consultar/criar WorkItems |
| `wosctl work checkout <id>` | Adquirir contrato e materializar o workspace |
| `wosctl work checkout --next --outcome <id>` | Selecionar/adquirir uma elegível |
| `wosctl work status <id>` | Mostrar contrato efetivo, prazo, pendências e omissões |
| `wosctl work refresh <id>` | Atualizar estado/contexto sem sobrescrever resultado local |
| `wosctl work resume <id>` | Reencontrar/materializar aquisição existente; verificar autoridade |
| `wosctl work resume <id> --takeover` | Anexar nova execução do mesmo titular, com fencing novo |
| `wosctl work renew <id>` | Solicitar uma renovação |
| `wosctl work keepalive <id> --foreground` | Loop de renovação explicitamente iniciado |
| `wosctl work diff <id>` | Mostrar o que será sincronizado e divergências de base |
| `wosctl work validate <id>` | Validar estrutura e binding local; não declarar aceite remoto |
| `wosctl work checkpoint <id>` | Registrar checkpoint com recibo |
| `wosctl work sync <id>` | Persistir itens permitidos, sem concluir |
| `wosctl work submit <id>` | Congelar entrega e obter submission_id |
| `wosctl work finalize <id> --submission <id>` | Tentar a conclusão validada |
| `wosctl work finish <id>` | Sequência ergonômica sync → submit → finalize |
| `wosctl work recover <id>` | Reconciliar journal/recibos, sem adquirir outra tarefa silenciosamente |
| `wosctl contract list/get/history` | Inspecionar aquisições e histórico |
| `wosctl contract revoke <id> --reason <texto>` | Revogação administrativa autorizada |
| `wosctl review ...` | Consultar submissões e registrar assessments autorizados |
| `wosctl roadmap ...` | Adaptar operações existentes de draft/publicação/ativação |

Não incluir `unlock` ou `release` acessível ao titular. O comando administrativo recebe contract_id, não apenas WorkItemID.

### 16.2 Comportamento agent-friendly

Todos os comandos devem suportar `--output json`; stdout contém somente resultado estruturado nesse modo, e diagnósticos vão para stderr. Não misturar banners, progress bars ou cores nos dados.

Mutações não interativas exigem parâmetros completos; não abrir prompts que deixem um agente preso. Operações destrutivas/administrativas necessitam intenção explícita e motivo, além da autorização do servidor. Um `--yes` não concede permissão.

O JSON de saída deve conter `operation`, `outcome_id`, `work_item_id`, `contract_id`, `committed`, `requires_action`, IDs/versões relevantes e `receipt_path`, quando houver.

### 16.3 Exit codes propostos

| Código | Categoria |
|---:|---|
| 0 | Operação solicitada concluída, sem obrigação pendente daquele comando |
| 2 | Uso/schema/configuração inválidos |
| 3 | Autenticação/autorização |
| 4 | Conflito de versão/base |
| 5 | Autoridade vencida, revogada ou fencing obsoleto |
| 6 | Transporte indisponível ou resultado de commit ainda incerto |
| 7 | Resultado submetido, mas finalização depende de revisão/critério |
| 8 | Nenhum trabalho elegível para `checkout --next` |
| 9 | Materialização/journal local incompletos |

Os códigos são estáveis por versão do cliente. SDK/MCP preservam erro de domínio correspondente; a CLI não reduz tudo a “erro inesperado”.

### 16.4 Preflight e dry-run

`diff` e `validate` não alteram o servidor. A validação local confirma formato, binding e consistência documental; não garante lease vigente ou prontidão atual sem consulta.

`checkout --dry-run` apenas mostra candidatos e condições observadas. Não apresenta isso como reserva. Para adquirir, é necessário executar a operação mutante real.

---

<a id="s17"></a>
## 17. Workspace local, journal e recuperação

### 17.1 Journal write-ahead do cliente

Antes de qualquer mutação, persistir uma entrada com destino vinculado, Scope, comando, chave idempotente, payload canônico exato, digest, versões esperadas e timestamp local informativo.

Estados locais sugeridos: `prepared`, `sent_unknown`, `confirmed` e `rejected`. Esses são estados de transporte do cliente, não lifecycles da tarefa.

Após resposta confirmada, salvar o recibo antes de marcar a intenção como finalizada. Não excluir a entrada pendente antes de ter prova do commit/erro terminal.

### 17.2 Recuperação determinística

| Situação | Tratamento |
|---|---|
| Timeout antes ou depois do commit | Repetir a mesma intenção/chave ou consultar recibo |
| Arquivo result.yaml mudou após envio | Reconciliar primeiro o payload original; tratar a edição como nova intenção |
| Recibo foi retornado, mas state.json não foi salvo | Restaurar o estado a partir do recibo e leitura atual |
| Retenção do recibo expirou | Conferir histórico/entidades; não presumir falha nem duplicar a operação |
| Outra sessão sincronizou progresso | Comparar base/local/remoto; não aplicar last-write-wins |
| Contrato já terminou | Preservar arquivo local e marcar autoridade encerrada |
| Configuração aponta para outro servidor | Recusar replay até restaurar o binding correto |

### 17.3 Serialização de comandos locais

Manter lock por workspace/contrato, não global para todas as tarefas. Renewal e sync podem usar versões distintas, mas atualizações de `state.json` e journal devem ser coordenadas.

Lockfile cooperativo serve apenas à CLI local. PID sozinho não prova posse após reboot ou em outra máquina; associar identificador de processo/inicialização e tratar lock abandonado sem alterar o estado remoto.

O takeover remoto é uma operação diferente de remover um lockfile abandonado. Nenhuma limpeza local revoga contrato no servidor.

### 17.4 Reconciliar sem apagar trabalho

`refresh` nunca sobrescreve `result.yaml` ou `checkpoint.yaml` modificados. Especificação imutável pode ser restaurada a partir do servidor, guardando uma cópia do arquivo divergente para inspeção.

Em conflito, apresentar três bases: snapshot local original, intenção local e estado atual. Não editar automaticamente versão ou titular para fazer o comando passar.

### 17.5 Segredos e logs

Não gravar bearer token em config versionável, contrato, resultado, command journal, stdout ou recibo. Usar referência a variável de ambiente ou integração explícita com armazenamento de credenciais do sistema operacional.

Logs locais e do servidor devem ter redaction. O payload necessário ao retry pode conter informações de trabalho; aplicar permissões locais adequadas e política de retenção, sem confundi-lo com conteúdo público de Git.

---

<a id="s18"></a>
## 18. Renovação no cliente e execução desconectada

### 18.1 Quem renova

No runtime remoto, o host ou um helper explicitamente configurado acompanha o lease. No desenvolvimento local, `wosctl work keepalive --foreground` ou integração equivalente do host renova enquanto estiver em execução.

Não depender de o modelo lembrar de renovar. Também não afirmar que um simples checkout instala um serviço permanente ou detecta automaticamente se o Codex/Claude ainda está trabalhando.

O loop recebe cancelamento, duração máxima opcional e feedback de perda de autoridade. Ao terminar, apenas para de renovar; a reserva termina quando expirar, salvo finalização ou revogação anterior.

### 18.2 Supervisão explícita

O consumidor deve ligar a vida do renovador à sessão autorizada. Um renovador abandonado não pode manter indefinidamente uma tarefa reservada sem decisão do host.

A CLI não inicia o agente. O usuário/host administra ambos os processos e, quando disponível, sinaliza liveness ao renovador. Ausência de mudança em arquivos não comprova inatividade: o agente pode estar lendo, testando ou aguardando revisão.

### 18.3 Desconexão

Um contrato não é uma licença offline ilimitada. O prazo continua avançando no servidor.

Durante desconexão, o consumidor pode conservar rascunhos locais, mas não pode assegurar exclusividade além do último prazo conhecido. Ao atingir o limite conservador, interrompe efeitos compartilhados e não afirma continuar como executor autorizado.

Depois de reconectar: resolver intenções incertas, consultar contrato atual, verificar titularidade/fencing e então renovar ou adquirir novamente conforme o estado. Não replayar automaticamente toda a outbox com novas chaves.

### 18.4 Falha e retomada

Fechar o terminal ou perder a sessão não finaliza a tarefa. Se o lease ainda estiver válido, o mesmo titular pode retomar com anexação explícita. Se venceu, qualquer consumidor autorizado que satisfaça a política pode adquirir uma nova tentativa.

O arquivo antigo continua útil como referência histórica, mas não como autoridade. O cliente deve mostrar claramente “contrato expirado/revogado” em vez de somente “workspace carregado”.

---

<a id="s19"></a>
## 19. Fluxos completos de uso

### 19.1 Planejador e executor local em sessões diferentes

1. O planejador cria Outcome, Objectives, WorkItems, critérios e dependências reais usando WOS.
2. Publica/ativa o Roadmap pertinente e deixa as tarefas adequadas elegíveis.
3. A sessão de planejamento termina sem adquirir contratos para tarefas que não executará.
4. O executor recebe o Outcome autorizado ou o descobre por uma consulta limitada.
5. Executa checkout de uma tarefa ou `--next`.
6. A CLI adquire o contrato e materializa contexto, spec e arquivos de resultado.
7. O agente lê apenas o necessário, executa com suas ferramentas e mantém o lease pelo host/CLI.
8. Registra checkpoints e provas com referências duráveis.
9. Submete a entrega; o revisor avalia os critérios exigidos.
10. O titular finaliza e recebe o recibo. Tarefas dependentes tornam-se elegíveis somente quando as condições reais forem satisfeitas.

O planejador não deve adquirir todos os contratos do Roadmap “para reservar o plano”. Contrato é aquisição de execução, não aprovação de planejamento.

### 19.2 Sequência ilustrativa no terminal

Os IDs devem vir da criação/consulta real. As variáveis abaixo representam valores já descobertos, não identificadores a inventar.

```bash
# Cliente configurado e autenticado por referência de credencial.
wosctl init --profile development
wosctl auth status --output json
wosctl capabilities --output json

# Consultar e adquirir uma tarefa específica.
wosctl work list --outcome "$OUTCOME_ID" --available --limit 5 --output json
wosctl work checkout "$WORK_ITEM_ID" --ttl 300 --output json
wosctl work status "$WORK_ITEM_ID" --output json

# O host inicia/supervisiona esta renovação separadamente da execução do agente.
wosctl work keepalive "$WORK_ITEM_ID" --foreground

# Durante o trabalho, em outra invocação do cliente:
wosctl work validate "$WORK_ITEM_ID"
wosctl work diff "$WORK_ITEM_ID"
wosctl work checkpoint "$WORK_ITEM_ID" --output json
wosctl work sync "$WORK_ITEM_ID" --output json

# Entrega e finalização continuam distinguíveis.
wosctl work submit "$WORK_ITEM_ID" --output json
wosctl work finalize "$WORK_ITEM_ID" --submission "$SUBMISSION_ID" --output json
```

Quando houver várias pastas históricas da mesma WorkItem, os comandos exigem seleção inequívoca do contrato ou usam o contrato atual validado no servidor. Nunca escolher silenciosamente a pasta de modificação mais recente.

### 19.3 Woobe como consumidor independente

O runtime da Woobe recebe uma tarefa de seu próprio caso de uso, injeta o contexto autorizado e expõe as operações MCP/HTTP correspondentes. Adquire um contrato, mantém seu estado operacional no host, executa as ferramentas e publica checkpoints/resultados.

Nenhum `.wos/` é necessário nesse cenário. A especificação é o mesmo objeto semântico, entregue por JSON. Sessões/runs do runtime continuam fora do domínio do WOS.

A configuração de TTL/renovação e a interrupção por perda de autoridade pertencem ao host. O Agent não deve ser responsável por manter IDs internos sensíveis na conversa se o host puder vinculá-los com segurança às chamadas.

### 19.4 Expiração e troca de executor

```text
Executor A → acquire → contrato C1, fencing 41
Executor A → checkpoint P1 aceito
C1 vence sem renovação
Executor B → acquire recovery
WOS fecha C1 como expired e emite C2, fencing 42
Executor B lê P1 e verifica o artefato real
Executor A envia resultado com C1/41 → rejeitado
Executor B entrega com C2/42 → revisão → finalização
```

A tarefa permanece a mesma. O histórico distingue o trabalho tentado por A do resultado finalmente aceito sob B.

### 19.5 Revogação administrativa

Administrador consulta C1, informa motivo e sua versão esperada. O WOS invalida a autoridade de C1 e registra o fato. Outro executor só assume mediante nova aquisição.

A UI/CLI do antigo titular deve mostrar revogação assim que consultar/renovar/sincronizar. O servidor não garante interromper imediatamente um processo externo que não está se comunicando com ele.

### 19.6 Falha do processo após commit remoto

Cliente persiste intenção, servidor confirma checkpoint ou conclusão e a conexão cai antes da resposta. Na retomada, a CLI reenvia a intenção original ou consulta o recibo. Não cria Evidence duplicada, não conclui duas vezes e não adquire outro contrato.

### 19.7 Trabalho bloqueado durante execução

Um insumo indispensável está ausente. O executor registra Issue e Blocker explícitos, salva checkpoint e interrompe os efeitos dependentes desse insumo. Pode manter o lease enquanto aguarda, conforme política do host, ou parar a renovação e permitir expiração. Não existe `status: blocked` livre nem `release` alternativo.

---

<a id="s20"></a>
## 20. Planejamento, Roadmaps e desenvolvimento local

### 20.1 Planejamento continua no modelo já existente

Outcome define o resultado; Objective define condição verificável; WorkItem define trabalho; Roadmap organiza referências. O novo contrato não duplica essas entidades.

O mínimo para o planejador entregar trabalho executável inclui: enunciado suficiente, entregáveis, critérios de sucesso, dependências reais, contexto acessível e estado operacional correto. Um documento de planejamento anexado como Artifact, sozinho, não cria WorkItems nem relações de dependência.

### 20.2 CLI de planejamento

Adaptar os comandos existentes para criação/consulta/edição autorizada de Outcomes, Objectives, WorkItems, critérios e Roadmaps. Permitir descrições longas por arquivo e operações determinísticas por IDs.

Manter operações de draft, publicação e ativação separadas. Não fazer `work sync` interpretar alterações do arquivo de resultado como mudanças de roadmap.

### 20.3 O que não será um reconciliador GitOps

A primeira versão não deve ler uma pasta e apagar/cancelar automaticamente tudo o que desapareceu dela. Também não deve transformar cada edição local de YAML em publicação automática do plano.

Um importador declarativo de Outcomes/Roadmaps completos exigiria resolução de identidades, limites de lote, controle de remoções, precondições e semântica própria. Não é necessário para entregar o checkout/sync contratual e fica como extensão posterior, sem contaminar o protocolo inicial.

### 20.4 Paralelismo

Agentes podem executar WorkItems independentes simultaneamente. O planejador declara `depends_on` somente quando existe dependência real. Agrupamento hierárquico ou ordem no Roadmap não substitui essa relação.

Quando diferentes tarefas alteram o mesmo arquivo ou ambiente, o planejador/host precisa de política de integração. O contrato não fornece worktrees, merge, lock de filesystem ou isolamento de sandbox.

---

<a id="s21"></a>
## 21. Interface humana e skills

### 21.1 Interface do WOS

Atualizar a interface existente em `packages/wos-api/web`, sem criar um segundo painel independente. [R15]

Na lista/quadro e no detalhe da WorkItem, mostrar:

- contrato atual e estado efetivo;
- titular e Actor/proveniência, sem credenciais;
- prazo avaliado pelo servidor e instante da consulta;
- última renovação, checkpoint e submissão;
- condição de recuperação quando expirado/revogado;
- requisitos de prova pendentes e conclusão vinculada;
- histórico de contratos e motivos de encerramento.

A contagem regressiva é apresentação. Antes de uma mutação, o servidor revalida o prazo; o JavaScript não concede autoridade.

### 21.2 Ações humanas

Remover a ação de liberação voluntária no protocolo novo. Mostrar revogação apenas a administradores autorizados, com contrato exato e motivo obrigatório.

Separar “registrar avaliação”, “finalizar contrato” e “revogar contrato”. O revisor não precisa possuir o lease para avaliar, mas não deve concluir usando a autorização de outro titular.

Permitir visualizar diferenças entre submissões, artifacts e checkpoints sem expor payloads técnicos enormes. Exibir falhas de CAS e exigir atualização do estado antes de nova intenção.

### 21.3 Skills existentes

Atualizar `wos-coordination`, `wos-continuity`, `wos-delegation` e `wos-review` e suas referências, preservando a descoberta progressiva.

A skill atual ainda orienta claim/renew/complete e menciona stop/release; essa orientação precisa acompanhar a mudança de política. [R13]

Instruções novas essenciais:

- consultar capabilities antes de assumir nomes de comandos;
- receber Scope autorizado; nunca inventar IDs existentes;
- não executar trabalho sem aquisição confirmada;
- não alterar a especificação em contract.yaml;
- guardar checkpoints materiais, não apenas texto de chat;
- parar efeitos incompatíveis quando perder autoridade;
- não usar release; pedir intervenção administrativa ou deixar expirar;
- não afirmar conclusão sem recibo aceito;
- não presumir que instalar uma skill conecta MCP ou concede permissões.

### 21.4 Nova orientação para o workspace local

Adicionar uma skill/referência específica, como `wos-workspace`, para o fluxo `wosctl` + YAML. Ela deve conhecer `status`, `diff`, `recover`, `checkpoint`, `submit` e `finalize`, com exemplos de conflitos e expiração.

Não exigir que Codex/Claude rodem dentro da Woobe. Não modificar automaticamente seus arquivos de configuração pessoais nem prometer compatibilidade com todos os clientes apenas porque os arquivos de skill foram instalados.

---

<a id="s22"></a>
## 22. Compatibilidade e migração

### 22.1 Decisão de migração

Usar migração explícita e controlada por Namespace/deployment. Durante a transição, o servidor pode suportar o protocolo antigo para escopos ainda não migrados e o novo para os migrados, mas **uma mesma WorkItem não pode ter duas autoridades de escrita concorrentes**.

Esse modo antigo é transitório, não uma opção permanente para contornar as três saídas. O aceite do novo modelo exige que seus Namespaces-alvo tenham concluído o cutover.

### 22.2 Caminho recomendado: drenagem

1. Publicar versão de servidor capaz de ler o estado antigo e preparar o novo schema.
2. Verificar backup/restore e inventariar leases, versões, critérios e recibos.
3. Atualizar todos os processos servidores do deployment; não deixar binários antigos gravando durante o cutover.
4. Colocar o Namespace em drenagem: recusar novas aquisições antigas e impedir prolongamento indefinido dos leases existentes.
5. Permitir que trabalho em curso conclua validamente ou que o prazo vigente expire. Intervenções administrativas existentes devem ser explícitas e auditadas.
6. Confirmar ausência de leases efetivamente válidos antes de ativar o protocolo novo.
7. Migrar referências e estado remanescente, preservando WorkItems `in_progress` recuperáveis.
8. Ativar o novo protocolo e liberar aquisições de WorkContract.
9. Atualizar consumidores, UI, SDK e skills; testar o bloqueio dos caminhos antigos.

Drenagem é preferível a inventar um snapshot histórico da aquisição de um lease antigo. A transição tem início/fim e estado observável; não pode ocorrer silenciosamente quando um agente faz checkout.

### 22.3 Histórico legado sem falsificação

Um lease antigo pode ter claim ID, titular, fencing e datas disponíveis, mas não o snapshot exato da tarefa no instante de aquisição. Nesse caso, registrar a origem e a incompletude; não chamar um snapshot feito na migração de snapshot original.

Contratos antigos já concluídos podem não ter dados suficientes para reconstrução completa. Preservar eventos/conclusões existentes e declarar a cobertura histórica disponível. Não fabricar contratos retrospectivos para preencher a UI.

### 22.4 Matriz de compatibilidade

| Operação antiga | Antes do cutover | Depois do cutover |
|---|---|---|
| Consultas de WorkItem | Mantidas | Mantidas, com projeções compatíveis quando possível |
| `claim_work_item` / `reclaim_work_item` | Comportamento legado durante a transição | Recusar novas mutações ou exigir ponte explicitamente certificada ao protocolo novo |
| `renew_work_item_lease` | Somente leases legados admissíveis na drenagem | Cliente novo usa contrato; não manter lease paralelo |
| `release_work_item` | Histórico/semântica legado até o marco definido | Recusar liberação voluntária |
| `complete_work_item` | Conclusão de lease legado ainda válido | Exigir finalização contratual |
| Conclusão administrativa antiga | Apenas política legada no escopo não migrado | Não pode concluir usando contrato vencido/revogado |
| Recibos históricos | Preservados | Legíveis com autorização; não reinterpretados como novas concessões |

Preferir erro explícito de protocolo a uma ponte incompleta que prometa compatibilidade e permita bypass. Caso se implemente uma ponte, ela precisa passar a mesma suíte de exclusividade e finalização.

### 22.5 Migrations

Usar o próximo número real de migration na branch de implementação, não assumir que o número observado neste plano continuará livre. Arquivos sugeridos: `NNNN_work_contracts.sql` e migrations subsequentes para checkpoints/submissões/índices.

Realizar mudanças aditivas e backfill verificável antes de ativar novos writers. Depois do cutover, `CurrentLease` legado deixa de ser fonte gravável para esse escopo.

Preservar fingerprints e recibos de comandos confirmados. Mudar serialização de DTO antigo ou converter números para strings exige versionamento explícito; não alterar o significado de uma chave idempotente já utilizada.

### 22.6 Retorno operacional

Antes de ativar o novo protocolo, é possível abandonar a ativação mantendo os dados aditivos, conforme teste de compatibilidade.

Depois que houver contratos novos confirmados, não voltar para um binário antigo que ignora sua autoridade. Priorizar correção adiante. Restauração de backup é procedimento explícito e precisa considerar todas as gravações posteriores; não descrevê-la como rollback sem perda por definição.

Marcador de versão de writer no banco ajuda apenas se o binário o respeitar. O controle de deployment deve impedir a execução de servidores antigos; não presumir que código legado reconhecerá automaticamente um campo que não existia.

---

<a id="s23"></a>
## 23. Distribuição e plataformas

### 23.1 Pacotes

O monorepositório possui `wos-api`, `wos-core`, `wos-sdk-go`, `wos-npm` e `wos-skill`. Adicionar `wos-cli` nessa organização, sem criar outro repositório ou microserviço como pré-requisito. [R14]

O cliente `wosctl` usa a API pública; não abre SQLite nem importa adapters de storage. O servidor local pode continuar separado, inclusive em ambiente Linux/contêiner, enquanto o cliente roda na máquina de desenvolvimento.

### 23.2 Plataformas de aceite

**Primeiro gate cliente:** Linux amd64 e Windows amd64, com execução real dos testes de filesystem, paths, saída JSON, interrupção e renovação. Windows é importante para o uso local proposto; não basta cross-compilar.

**Extensão subsequente:** macOS arm64/amd64 e outras combinações, após testes reais ou runners adequados. O suporte do cliente não amplia automaticamente a matriz de suporte do servidor/SQLite.

A versão de Go deve respeitar `go.mod` e as ferramentas fixadas do repositório no momento da implementação. Não atualizar toolchain apenas para introduzir a CLI. [R16]

### 23.3 Release

Integrar o cliente à política coordenada de versões e artefatos do repositório. Produzir binário, checksum, manifesto e instruções de conexão. Um pacote npm específico de cliente pode ser adicionado somente com identidade de publicação confirmada; nome e disponibilidade de registry não são assumidos neste plano.

Não iniciar servidor, instalar daemon de heartbeat, baixar scripts ou editar configurações de agente durante instalação. Instalar arquivos, iniciar processo, autenticar e adquirir trabalho são ações distintas.

### 23.4 Versionamento público

Versionar separadamente: produto, API/transport, schema YAML, protocolo de execução e schema de eventos. `schema_version: 1` não obriga o pacote a ser 1.0.0 nem significa que a API é nova.

A mudança de release voluntário e dos comandos de execução precisa de nota de incompatibilidade. Mesmo em uma série 0.x, não ocultar a alteração de comportamento.

---

<a id="s24"></a>
## 24. Mapa de alterações no repositório

Caminhos marcados como **novo** são propostas. Os demais foram encontrados na baseline ou são diretórios explicitamente presentes nela; os nomes finos de arquivos de geração devem ser confirmados antes da implementação.

| Área | Caminhos principais | Alteração |
|---|---|---|
| Domain | `packages/wos-core/domain/work_item.go` | Refatorar autoridade de execução, release, conclusão e invariantes |
| Domain/lease | `packages/wos-core/domain/work_item_lease.go` | Reutilizar validação temporal/fencing no componente contratual |
| Domain/contratos | `packages/wos-core/domain/work_contract.go` **novo** | Entidade, transições, validade e versões |
| Domain/continuidade | `work_checkpoint.go`, `work_submission.go` **novos no Domain** | Registros imutáveis e invariantes |
| Application | `packages/wos-core/application/service.go` | Impedir bypass por criação/edição/critérios/conclusão |
| Application/lease | `packages/wos-core/application/lease_service.go` | Novo caminho e compatibilidade controlada |
| Application/admin | `packages/wos-core/application/administrative_work.go` | Revogação e adaptação de cancelamento/conclusão privilegiados |
| Application/contratos | `work_contract_commands.go`, `work_contract_service.go`, `work_contract_queries.go` **novos** | Casos de uso e consultas |
| Ports | `packages/wos-core/ports/` | Repositories/UnitOfWork/tempo para novas estruturas |
| Storage | `packages/wos-core/storage/{memory,sqlite,postgres}/` | Persistência, índices, migração, contrato comum e restore |
| Catálogo | `packages/wos-api/commands/` | Inscrição tipada e compatibilidade |
| HTTP | `packages/wos-api/http/` | Rotas, envelopes, erros, capabilities e limites |
| MCP | `packages/wos-api/mcp/` | Tools/queries do mesmo Application |
| API schema | `packages/wos-api/commands.openapi.json`, `openapi.yaml` | Regenerar; não editar divergências à mão |
| SDK | `packages/wos-sdk-go/client.go` e métodos gerados | Queries/commands, erros e renovação opt-in |
| Cliente | `packages/wos-cli/` **novo** | Parser, mapper, workspace, journal e CLI |
| Binário cliente | `packages/wos-cli/cmd/wosctl/` **novo** | Entry point sem bootstrap de servidor |
| UI | `packages/wos-api/web/` | Contrato, prazo, recuperação, submissões e revogação |
| Skills | `packages/wos-skill/skills/` | Atualização das quatro skills e guia local |
| Geradores | `tools/mcpgen`, `tools/openapigen`, `tools/postgresgen/` | Estender as fontes/fixtures de geração existentes |
| Aceite | `tests/acceptance/`, `tests/boundary/`, `tests/web/` | Jornadas novas e paridade |
| Cliente/testes | `packages/wos-cli/**/_test.go` e fixtures **novos** | Parsing, IO, journal, crash e plataformas |
| Documentação | `AGENTS.md`, `ROADMAP.md`, `README.md`, `docs/contracts.md`, `docs/agents.md`, ADRs | Registrar nova política e instruções corretas |

Estrutura interna sugerida da CLI:

```text
packages/wos-cli/
├── cmd/wosctl/
├── cli/                 # Parsing de argumentos e saída
├── config/              # Perfil, binding e referência de credencial
├── documents/           # Schemas/validação YAML e conversão para DTOs
├── workspace/           # Paths, escrita, locks e estado local
├── journal/             # Intenções e recibos
├── lease/               # Renovador explícito e cancelável
└── testdata/            # Fixtures válidas/inválidas e jornadas
```

A CLI não deve duplicar a lógica de prontidão, de critérios ou de transição do Domain. Validação local antecipa erros; a decisão final permanece no WOS.

---

<a id="s25"></a>
## 25. Plano de execução por ondas e PRs

Executar por dependência. Os números abaixo são identificadores do plano, não números de PR do GitHub. Criar branches a partir da `master` real e integrar recortes verificáveis; não afirmar progresso por quantidade de commits.

### Onda C01 — ADRs, inventário e contratos normativos

**Dependências:** nenhuma; leitura obrigatória da baseline atualizada.

**Entregas:** ADR de WorkContract e três encerramentos; ADR de YAML/CLI; mapa de comandos legados; esquema de versões/tempo; política de migração; roadmap com estas ondas. Inventariar os caminhos de bypass e os schemas/geradores reais.

**Aceite:** os documentos não deixam ambíguo quem pode adquirir, renovar, concluir e revogar. Schemas preliminares e fixtures concordam com as decisões. Nenhuma funcionalidade nova é marcada como entregue.

**PR sugerido:** `docs/contracts-execution-model`.

### Onda C02 — Domain e testes puros

**Dependências:** C01.

**Entregas:** WorkContract, lease incorporado, WorkCheckpoint, WorkSubmission, invariantes, fencing exato/overflow, definição de validade efetiva e regras de alteração da especificação. Ajustar WorkItem sem introduzir nova máquina de runtime.

**Aceite:** testes de transição cobrem as três saídas, limite exato de prazo, proibição de release e reativação de terminal. Tarefa e contrato permanecem identidades distintas.

**PR sugerido:** `feat/work-contract-domain`.

### Onda C03 — Ports, Memory e fluxo transacional mínimo

**Dependências:** C02.

**Entregas:** repositories/UnitOfWork; acquire/renew/revoke; aquisição de recuperação; snapshots transacionais; takeover explícito; idempotência e eventos em Memory. Separar contrato/lease versions.

**Aceite:** dois consumidores concorrentes nunca mantêm contratos válidos simultâneos; idempotência retorna a aquisição original; perda de autoridade bloqueia mutações. Nenhuma transação pública aninhada é usada como substituto de atomicidade.

**PR sugerido:** `feat/work-contract-application`.

### Onda C04 — SQLite, PostgreSQL e schema aditivo

**Dependências:** C03.

**Entregas:** migrations aditivas, índices, contrato de storage comum, autoridade temporal, materialização de expiração, backup/restore e inventário de legado. Implementar fencing sem perda de precisão.

**Aceite:** mesma suíte nos três adapters; corrida em conexões/processos distintos; restore preserva história e contratos vencidos não revivem. Teste ignorado por ausência de PostgreSQL não conta como paridade aprovada.

**PR sugerido:** `feat/work-contract-storage`.

### Onda C05 — HTTP, MCP, SDK e consultas focais

**Dependências:** C03 e C04.

**Entregas:** catálogo tipado, rotas/queries, capabilities, limites e códigos de erro; SDK; acquire-next limitado e contexto suficiente; history/checkpoints paginados. Regenerar OpenAPI/MCP/SQL/SDK conforme os geradores reais.

**Aceite:** um cliente HTTP adquire e o mesmo Principal retoma/consulta via MCP, sem divergência semântica. Catálogos e schemas gerados passam verificação de drift. O receipt omitido por tamanho é tratado como sucesso confirmado.

**PR sugerido:** `feat/work-contract-transports`.

### Onda C06 — Checkpoints, sincronização e entrega verificável

**Dependências:** C04 e C05.

**Entregas:** sync composto limitado; mapeamento de local keys; submissões imutáveis; vínculo de avaliações à entrega; finalize atômico; regras de impedimentos e alterações de especificação; proteção dos comandos existentes.

**Aceite:** um contrato percorre aquisição → checkpoint → submissão → revisão → conclusão em HTTP e MCP. Um resultado parcial, uma avaliação obsoleta ou uma revisão pendente não encerram a reserva.

**PR sugerido:** `feat/work-contract-results`.

### Onda C07 — CLI mínima e documentos YAML

**Dependências:** C05 e C06.

**Entregas:** pacote/binary `wosctl`; init/auth/capabilities; checkout/status/refresh; schemas e parser; config sem segredo; materialização e binding de destino. Selecionar/pinar a biblioteca YAML por testes, não apenas por popularidade.

**Aceite:** no terminal, adquirir tarefa real em servidor de teste e produzir os documentos esperados; rejeitar YAML malformado, duplicado ou incompatível antes de mutação. O cliente não acessa o banco nem inicia agentes.

**PR sugerido:** `feat/wosctl-workspace`.

### Onda C08 — Journal, renovação e recuperação local

**Dependências:** C07.

**Entregas:** outbox local, recibos, locks cooperativos, escrita atômica, diff/validate/sync/checkpoint/submit/finalize/finish; renovador foreground; retomada/takeover explícitos; recuperação após resposta perdida e falha de disco.

**Aceite:** processo morto em pontos controlados não duplica comandos nem perde a identidade da aquisição. Falha de materialização mantém a reserva recuperável. Dois processos locais não corrompem journal/state. Os cenários rodam em Linux e Windows.

**PRs sugeridos:** `feat/wosctl-sync-recovery` e `feat/wosctl-lease-lifecycle`, divididos por dependência real.

### Onda C09 — Interface humana, skills e planejamento cliente

**Dependências:** C06 e C08.

**Entregas:** painel de contratos/submissões, revisão e revogação; remoção de release; atualização de skills e exemplos; wrappers cliente de planejamento sobre os comandos existentes.

**Aceite:** humano consegue revisar e revogar com autorização; agente local consegue iniciar uma sessão nova e recuperar o contexto sem conversa anterior. A instalação de skill é testada separadamente da execução real do cliente-alvo.

**PR sugerido:** `feat/work-contract-ux-skills`.

### Onda C10 — Migração e bloqueio de bypass legado

**Dependências:** C04–C09.

**Entregas:** drenagem, backfill honesto, capability/protocolo por Namespace, bloqueios de comandos antigos, notas de migração e proteção operacional contra servidores antigos. Preservar recibos/fingerprints.

**Aceite:** um dataset de 0.1 é migrado sem mudar identidades/critério/conclusões. Nenhum caminho antigo libera voluntariamente, conclui com fencing obsoleto ou cria lease paralelo no Namespace migrado. O ensaio de migração inclui falha/restart em etapas intermediárias.

**PR sugerido:** `feat/work-contract-migration`.

### Onda C11 — Jornadas independentes e concorrência ampliada

**Dependências:** C09 e C10.

**Entregas:** piloto local com CLI/YAML; piloto programático HTTP/MCP; teste real de consumidor Woobe quando o ambiente estiver disponível; takeover, expiração, revogação, revisão e restore; fixtures de carga e relatórios.

**Aceite:** os dois casos de uso funcionam separadamente. A aceitação por cliente HTTP/MCP independente não é rotulada como certificação da Woobe sem executar esse consumidor. Evidências apontam para o commit final testado.

**PR sugerido:** `test/work-contract-acceptance`.

### Onda C12 — Distribuição, documentação final e release

**Dependências:** C11.

**Entregas:** artefatos cliente, checksums/manifests, matriz de plataformas, notas de incompatibilidade, guias de operação, atualização de docs e roadmap com status comprovado.

**Aceite:** build e testes finais no commit integrado; pacotes instalados em ambientes-alvo; server e cliente identificam suas versões; publicação só é marcada concluída com comprovação de artefatos publicados.

**PR sugerido:** `release/work-contracts-client`.

### Ordem resumida

```text
C01 → C02 → C03 → C04 → C05 → C06 → C07 → C08
                                             ↓
                                            C09 → C10 → C11 → C12
```

Escrita de testes e documentação acompanha cada onda, não fica concentrada nas últimas. Não adiar a proteção de domínio para depois da distribuição da CLI.

---

<a id="s26"></a>
## 26. Matriz de testes

Executar os casos de domínio/application em Memory e a suíte contratual de storage em SQLite/PostgreSQL. Casos de transporte devem verificar HTTP e MCP; casos de workspace devem executar no filesystem real das plataformas declaradas.

### 26.1 Estado, tempo e exclusividade

| ID | Caso | Resultado obrigatório |
|---|---|---|
| T01 | Acquire de WorkItem elegível | Um contrato ativo e snapshot coerente |
| T02 | Dois acquires simultâneos | No máximo um vencedor |
| T03 | Acquire pelo mesmo titular com nova intenção | Não cria segundo contrato ativo |
| T04 | Replay da aquisição confirmada | Mesmo ID, fencing e recibo |
| T05 | Renew antes do vencimento | Mantém contrato e estende prazo validamente |
| T06 | Renew exatamente no vencimento | Rejeitado |
| T07 | Renew depois do vencimento | Rejeitado, sem ressurreição |
| T08 | Horários de réplicas divergentes | Autoridade do banco decide |
| T09 | Espera longa por lock | Validade verificada depois da espera |
| T10 | Sweeper desligado | Lease vencido continua sem autoridade |
| T11 | Sweeper em duas réplicas | Um fato terminal de expiração |
| T12 | Expiração sem nova aquisição | WorkItem não vira done nem perde progresso |
| T13 | Reacquire após expiração | Novo ID e fencing maior |
| T14 | Reacquire após revogação | Novo ID e pré-condições revalidadas |
| T15 | Reacquire com dependência pendente | Rejeitado |
| T16 | Overflow de fencing em todos os caminhos | Erro sem wraparound/commit parcial |
| T17 | Fencing maior que inteiro exato de JavaScript | Round-trip sem perda |
| T18 | Takeover do mesmo titular | Nova execução; geração anterior rejeitada |
| T19 | Takeover por outro titular | Rejeitado |
| T20 | Copiar contrato sem autenticação | Não concede autorização |

### 26.2 Encerramento, prova e invariantes cruzadas

| ID | Caso | Resultado obrigatório |
|---|---|---|
| T21 | Release pelo titular | Não encerra contrato no protocolo novo |
| T22 | Revogação administrativa com motivo | Fecha contrato exato e preserva histórico |
| T23 | Revogação sem permissão/motivo | Rejeitada |
| T24 | Revogar contrato já expirado/concluído | Não reescreve causa terminal |
| T25 | Revoke × finalize concorrentes | Uma única causa de fechamento |
| T26 | Sync/checkpoint válido | Mantém reserva |
| T27 | Submit válido | Mantém reserva até finalização |
| T28 | YAML afirma conclusão | Nenhuma conclusão sem comando validado |
| T29 | Evidence anexada sem assessment | Não aprova critério automaticamente |
| T30 | Assessment da revisão errada | Não justifica finalização |
| T31 | Assessment de submissão materialmente diferente | Não é reutilizado silenciosamente |
| T32 | Revisor independente exigido | Executor não aprova a própria entrega |
| T33 | Evidência retraída | Não sustenta nova conclusão |
| T34 | Finalize com contrato válido e prova correta | WorkItem/contrato/Conclusion/receipt atômicos |
| T35 | Falha entre escritas da finalização | Rollback completo |
| T36 | Finalize com lease expirado ou fencing antigo | Rejeitado |
| T37 | Completion administrativa legada | Não contorna o contrato no escopo migrado |
| T38 | Mudança de spec/critério/dependência durante contrato válido | Exige fluxo explícito de revogação |
| T39 | Novo Blocker durante execução | Motivo vivo aparece; conclusão é impedida |
| T40 | Nova revisão de Roadmap só muda ordem | Não reescreve contrato ou dependências |
| T41 | Arquivar/concluir Outcome com contratos ativos afetados | Exige resolução explícita |
| T42 | Reabrir WorkItem concluída | Histórico preservado; novo contrato requerido |
| T43 | Finalizar WorkItem | Não conclui Objective/Outcome automaticamente |
| T44 | Revogação administrativa em estado legado inconsistente | Não depende de elegibilidade de execução para retirar autoridade |

### 26.3 Transporte, idempotência e namespace

| ID | Caso | Resultado obrigatório |
|---|---|---|
| T45 | HTTP e MCP com mesmo caso de uso | Mesmo efeito, versões e invariantes |
| T46 | Timeout depois de commit | Replay recupera sucesso original |
| T47 | Chave idempotente com payload alterado | Conflito explícito |
| T48 | Replay de renew | Não estende novamente |
| T49 | Replay de finalize após antigo prazo de lease | Recupera recibo da conclusão passada |
| T50 | Credencial revogada tenta ler replay | Acesso negado |
| T51 | IDs de outro Namespace em referências | Rejeição sem vazamento |
| T52 | ExternalContext manipulado | Não amplia autorização |
| T53 | Body/resultado acima do limite | Rejeição de entrada ou commit receipt de saída conforme contrato |
| T54 | Contexto truncado | Omissões e expansão explícitas |
| T55 | Paginação da spec após renew | Snapshot imutável continua identificável |
| T56 | Conflito de contract.version versus lease_version | Não mistura progresso com heartbeat |
| T57 | Cliente legado em Namespace migrado | Não cria lock paralelo nem libera tarefa |
| T58 | Acquire-next sem candidato no recorte limitado | Não confunde busca incompleta com ausência total |

### 26.4 YAML, filesystem e operação local

| ID | Caso | Resultado obrigatório |
|---|---|---|
| T59 | YAML válido e equivalente `.yml` | Mesmo objeto semântico |
| T60 | Chave duplicada/campo desconhecido | Erro com localização; nenhuma mutação |
| T61 | Alias/tag/merge/include não permitido | Rejeitado |
| T62 | Documento profundo/grande/múltiplo | Limite aplicado antes de processamento indevido |
| T63 | Spec editada e hash recalculado pelo cliente | Servidor rejeita divergência canônica |
| T64 | Campo de titular/TTL/done no result.yaml | Rejeitado como campo não permitido |
| T65 | Path traversal/symlink/reparse point | Sem escrita fora da raiz autorizada |
| T66 | Config troca servidor antes do replay | Credencial/intenção não é enviada a destino diferente |
| T67 | Falha de disco depois de acquire | Recupera a aquisição; não libera nem duplica |
| T68 | Crash em cada etapa do journal | Recibos e payloads reconciliáveis |
| T69 | Edição local durante sync | Recibo corresponde aos bytes efetivamente enviados |
| T70 | Refresh com result.yaml modificado | Resultado local preservado |
| T71 | Dois processos CLI no mesmo workspace | Sem corrupção de journal/state |
| T72 | Keepalive cancelado | Para de renovar; não conclui/libera |
| T73 | Desconexão até vencer | Cliente não se apresenta como autorizado |
| T74 | Token em erro/config/journal/log | Não é persistido/exibido indevidamente |
| T75 | Windows: paths, rename, locks, sinais | Comportamento validado, não apenas compilação |
| T76 | stdout JSON | Parseável sem logs/banners misturados |

### 26.5 Migração, restauração e jornadas

| ID | Caso | Resultado obrigatório |
|---|---|---|
| T77 | Dataset legado com lease vigente | Drenagem não inventa contrato histórico |
| T78 | Dataset com lease expirado | WorkItem recuperável, sem revival |
| T79 | Histórico incompleto | Incompletude explícita, sem fatos fabricados |
| T80 | Restart no meio do cutover | Recomeço determinístico, um protocolo ativo por escopo |
| T81 | Restore de contrato active já vencido no tempo real | Sem autoridade após restaurar |
| T82 | Restore de contrato completed | Resultado/recibo/provas preservados |
| T83 | Planejador A e executor B sem histórico de chat | B obtém contexto suficiente no WOS |
| T84 | Consumidor remoto sem arquivos YAML | Mesmo modelo operacional |
| T85 | Consumidor local sem Woobe | Jornada completa pela CLI |
| T86 | Agentes distintos por Principal | Titularidade não depende do nome do agente |
| T87 | Publicação de artefatos cliente | Versões/checksums correspondem ao commit testado |
| T88 | Bypass pelo Core/Application/transport antigo | As três saídas continuam sendo as únicas |

Os nomes de testes concretos podem variar; os IDs funcionam como rastreabilidade no roadmap/relatório. Casos relevantes devem incluir tabelas de combinações, não apenas um exemplo feliz.

---

<a id="s27"></a>
## 27. Desempenho e observabilidade

### 27.1 Objetivos verificáveis de eficiência

Reduzir a jornada inicial de várias chamadas para uma aquisição composta, sem perder contexto nem atomicidade. Separar renovação de lease de gravação do documento inteiro. Manter leituras focais e histórico paginado.

Não prometer economia de tokens ou latência em milissegundos sem medição. Medir por jornada: chamadas, bytes, tokens do contexto entregue quando o host permitir, SQL queries, tempo de transação, conflitos, retries e qualidade da aceitação.

### 27.2 Benchmarks propostos

Fixtures com 100, 1.000 e 10.000 WorkItems; variação de critérios/dependências e de 0, 10 e 100 checkpoints por tarefa. Exercitar 2, 10 e 50 consumidores concorrentes, dentro da infraestrutura de teste disponível, separando SQLite e PostgreSQL.

Medir acquire específico, acquire-next, renew, sync pequeno, consulta focal, finalização e recuperação após restart. Registrar p50/p95/p99, throughput, leituras/escritas SQL, bytes e hardware/configuração usados.

Metas estruturais iniciais:

- renew não carrega todo o histórico nem regrava spec;
- contexto de uma tarefa não cresce com todo o histórico do Outcome;
- contratos, checkpoints e submissões possuem paginação;
- resultado remoto respeita 256 KiB e declara omissões;
- acquire-next usa recortes/indexação e informa quando a busca é incompleta;
- nenhuma chamada de rede externa ocorre sob o Outcome guard.

Se um recorte de candidatos não contém trabalho elegível, retornar `search_complete=false` e cursor/razão quando existir mais espaço de busca. Somente uma busca concluída pode afirmar “nenhum trabalho elegível”.

### 27.3 Renovação e capacidade

O volume aproximado de renovações é `contratos_ativos / intervalo_de_renovação`. Por exemplo, 500 contratos com renovação a cada 100 segundos produzem aproximadamente 5 intenções de renovação por segundo, antes de retries. É uma estimativa aritmética de carga, não benchmark do WOS.

Como mutações continuam usando o guard do Outcome, distribuir muitas execuções num único Outcome pode concentrar contenção. Medir antes de tentar remover o guard; não enfraquecer invariantes para melhorar um benchmark isolado.

### 27.4 Sinais operacionais

Instrumentar contagens de aquisição/conflito, expiração, revogação, finalização, stale fencing, falha de renovação, resultado aguardando revisão, replay idempotente e recuperação de workspace.

Medir duração/bytes dos comandos e lag entre `expires_at` e registro do evento de expiração. Esse lag é de materialização, não uma extensão de autoridade.

Não usar contract_id, user_id ou todos os IDs de Outcome como labels irrestritas de métricas. IDs detalhados entram em logs/eventos consultáveis com autorização. Nenhuma instrumentação deve expor credenciais.

---

<a id="s28"></a>
## 28. Critérios de aceite e rastreabilidade

### 28.1 Aceite funcional do modelo

- [ ] Aquisição confirmada gera um único contrato válido com snapshot identificável.
- [ ] Expiração, revogação e finalização são as únicas saídas da reserva.
- [ ] Release voluntário e bypass por comandos antigos estão bloqueados no protocolo novo.
- [ ] Renovação não revive contrato e não conflita artificialmente com toda edição de progresso.
- [ ] Novo contrato ou takeover invalida gerações antigas conforme a semântica definida.
- [ ] Checkpoints, submissões e conclusões permanecem auditáveis.
- [ ] O resultado final está vinculado aos critérios e à base material avaliados.
- [ ] Finalização atualiza contrato/WorkItem/prova/recibo atomicamente.
- [ ] A recuperação revalida elegibilidade e não apaga trabalho anterior.
- [ ] Namespaces e autorização atual são respeitados também no replay.

### 28.2 Aceite do cliente e das interfaces

- [ ] Woobe/outro runtime pode usar HTTP/MCP/SDK sem arquivos locais.
- [ ] Um consumidor local pode usar CLI/YAML sem depender de Woobe.
- [ ] O checkout materializa arquivos com schema explícito e sem segredos.
- [ ] A CLI preserva intenção/recibos e recupera respostas incertas.
- [ ] Renovação é explicitamente supervisionada; instalação não inicia processos ocultos.
- [ ] YAML editado não altera titular, prazo, fencing, critério ou lifecycle canônico.
- [ ] Linux e Windows passam testes reais do cliente antes de serem anunciados como suportados.
- [ ] UI e skills refletem a política nova e distinguem submissão de conclusão.

### 28.3 Aceite operacional de entrega

- [ ] Memory, SQLite e PostgreSQL passam seus testes aplicáveis.
- [ ] Migração e restore foram executados sobre dados, não apenas compilados.
- [ ] Schemas/catálogos/SDK gerados estão sincronizados.
- [ ] Relatórios identificam commit, ambientes, testes executados e skips.
- [ ] O roadmap aponta implementações e evidências reais, sem recontar histórico como entrega nova.
- [ ] Publicação é comprovada separadamente do build local.

### 28.4 Rastreabilidade entre requisito e implementação

| Requisito | Ondas principais | Evidência central |
|---|---|---|
| Um contrato válido por tarefa | C02–C05 | T01–T04, T13–T20 |
| Três encerramentos, sem release | C02, C06, C10 | T21–T25, T37, T88 |
| Expiração confiável | C03–C04 | T05–T12, T81 |
| Snapshot contratual e replanejamento | C02, C05–C06 | T38–T40, T54–T55 |
| Progresso e handoff | C06, C08 | T26–T27, T69–T70, T83 |
| Entrega verificável | C06, C09 | T28–T36, T42–T43 |
| Paridade de transport | C05–C06 | T45–T58, T84 |
| YAML/CLI local | C07–C08 | T59–T76, T85 |
| Recuperação sem duplicação | C03, C08 | T04, T46–T49, T67–T68 |
| Migração compatível e honesta | C10 | T57, T77–T82, T88 |
| Consumidores independentes | C09, C11 | T83–T86 |
| Distribuição comprovada | C12 | T75, T87 |

**Definição de concluído:** todos os requisitos obrigatórios do modelo e os gates declarados para a release estão comprovados no commit integrado. Funcionalidades de uma plataforma/consumidor ainda não exercitado permanecem declaradas como não certificadas.

---

<a id="s29"></a>
## 29. Decisões consolidadas e extensões posteriores

### 29.1 Escolhas recomendadas para esta implementação

| Questão | Escolha |
|---|---|
| Arquivo ou servidor como fonte de verdade? | Servidor; arquivo é snapshot/rascunho |
| Contrato obrigatório para executar WorkItem no modelo novo? | Sim, em todas as interfaces |
| YAML obrigatório para todos os clientes? | Não |
| Reserva permanente mais lease separado? | Não; um contrato com lease renovável |
| Release voluntário? | Não |
| Renovação automática pelo modelo? | Não; host/cliente explícito e cancelável |
| Expiração depende de manutenção assíncrona? | Não; o tempo invalida imediatamente; manutenção registra o fato |
| Resultado autoaprovado por YAML? | Não |
| Dois arquivos para spec e resultado? | Sim; especificação emitida separada de proposta editável |
| Sincronização last-write-wins? | Não; versões, digest, comparação e recibos |
| Lock de todo o Outcome? | Não; somente guard transacional curto e contrato por WorkItem |
| Cliente compartilha binário de servidor? | Não inicialmente; `wosctl` é cliente, `wos` permanece servidor |
| Novo serviço ou repositório? | Não como pré-requisito |
| SDK básico com retry automático? | Não; helpers opt-in e política explícita |
| Importação GitOps de todo o plano? | Fora do gate inicial |

### 29.2 Extensões posteriores, sem alterar a base

Filtragem por capacidades do executor; importação declarativa de planejamento completo; cache incremental de contexto com deltas; integração de credential store adicional; interfaces de acompanhamento por notificações; novos alvos de distribuição.

Nenhuma dessas extensões pode introduzir um scheduler autônomo no WOS, transferir posse por edição local, reativar contrato terminal ou permitir outro caminho de encerramento sem decisão explícita.

### 29.3 Parâmetros a fechar durante implementação

Versão exata da biblioteca YAML, números de migrations, limites finais de lote, forma de empacotamento npm e nome/número da release são escolhas operacionais a confirmar contra o estado real naquele momento.

Esses parâmetros não deixam a arquitetura em aberto: a semântica de aquisição, exclusividade, validade, prova e encerramento já está definida neste plano. Mudanças pequenas de nomenclatura devem atualizar schemas, exemplos, skills e testes conjuntamente.

---

<a id="s30"></a>
## 30. Instruções para o agente implementador

Este trecho pode ser usado como contrato de execução da implementação no repositório:

> Implemente a evolução de contratos de execução do WOS seguindo este plano e as decisões atuais do proprietário. Antes de alterar arquivos, leia AGENTS.md, ROADMAP.md, os ADRs e o código pertinente na master real. Compare a baseline com o commit referenciado neste documento e registre diferenças relevantes. Não assuma que componentes propostos já existem.
>
> Preserve o WOS como serviço independente de estado e coordenação. Não crie runtime, scheduler de agentes, executor de ferramentas ou dependência da Woobe. O cliente local usa a API pública e não acessa storage diretamente.
>
> Faça a reserva de execução terminar somente por expiração, revogação administrativa ou finalização aceita. Remova a liberação voluntária no protocolo novo e proteja os caminhos HTTP, MCP, SDK, UI, comandos legados e Core/Application contra bypass.
>
> Execute as ondas em ordem de dependência, entregando recortes verificáveis. Cada PR deve conter testes correspondentes, atualização do roadmap e documentação das mudanças observáveis. Não marque uma onda como concluída só porque arquivos foram criados ou o projeto compilou.
>
> Preserve versões, idempotência, authority de tempo, fencing, snapshots, provas e histórico. Não recalcule versões automaticamente para fazer uma intenção obsoleta passar. Diferencie resultado submetido, avaliação e conclusão.
>
> Mantenha o branch de implementação derivado da master atual, revise as alterações antes de integrar e registre o commit testado. Não trate testes ignorados ou CI não executado como aprovação. Não publique pacotes, altere ambientes remotos ou execute migração destrutiva sem a autorização aplicável.
>
> Ao terminar cada recorte, informe o que foi efetivamente implementado, os comandos/testes executados, a evidência, as pendências e a próxima dependência. Atualize ROADMAP.md sem apagar o histórico anterior.

### Primeiro recorte recomendado

Começar por C01–C03: ADRs, invariantes de contrato e fluxo transacional em Memory. Isso comprova a mudança de domínio antes de investir na ergonomia do arquivo.

O primeiro vertical slice com usuário deve vir depois: servidor com storage/transport novos + CLI que adquire, materializa, registra um checkpoint, submete e finaliza. Só então ampliar distribuição e automação de recuperação.

---

<a id="s31"></a>
## 31. Fontes e limites da verificação

### 31.1 O que foi verificado

Consulta da branch `master` pelo conector GitHub e inspeção de arquivos de domínio, Application, contratos, entrypoint, SDK, skill e documentação no SHA fixado. A seleção foi orientada a leases, concorrência, prova, interfaces e distribuição do novo modelo.

Este trabalho **não executou a suíte de testes do WOS, não fez checkout executável do projeto, não validou um deployment Woobe e não implementou os componentes propostos**. Os testes descritos são requisitos de aceite futuros. Não há alegação de que esta proposta já funcione no repositório.

A referência à estrutura/semântica existente é de alta confiança nos arquivos consultados. Números de migrations, detalhes de geração, dependências e estado da branch devem ser reconferidos na implementação. O restante do documento é proposta arquitetural explicitamente identificada.

### 31.2 Fontes do repositório

Os links abaixo são fixados por commit quando aplicável. As referências `[Rxx]` no texto apontam para esta relação.

**[R01] Baseline da master e commit consultado.**  
`https://github.com/A1b3rt0M3rcad0/wos/commit/bcd1714ad7cded666890d0b5c9f711772c4f8d7a`

**[R02] WorkItem, lease, release e conclusão.**  
`https://github.com/A1b3rt0M3rcad0/wos/blob/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/packages/wos-core/domain/work_item.go`

**[R03] Renovação, reclaim, limite temporal e fencing.**  
`https://github.com/A1b3rt0M3rcad0/wos/blob/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/packages/wos-core/domain/work_item_lease.go`

**[R04] Application de renovação e reclaim.**  
`https://github.com/A1b3rt0M3rcad0/wos/blob/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/packages/wos-core/application/lease_service.go`

**[R05] Cancelamento e conclusão administrativos.**  
`https://github.com/A1b3rt0M3rcad0/wos/blob/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/packages/wos-core/application/administrative_work.go`

**[R06] Contratos HTTP/MCP, SDK, limites, continuidade e prova.**  
`https://github.com/A1b3rt0M3rcad0/wos/blob/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/docs/contracts.md`

**[R07] Entry point operacional do binário wos existente.**  
`https://github.com/A1b3rt0M3rcad0/wos/blob/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/packages/wos-api/cmd/wos/main.go`

**[R08] Cliente SDK Go e seus imports/comportamento de retry.**  
`https://github.com/A1b3rt0M3rcad0/wos/blob/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/packages/wos-sdk-go/client.go`

**[R09] Application Service, critérios e conclusão de WorkItem.**  
`https://github.com/A1b3rt0M3rcad0/wos/blob/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/packages/wos-core/application/service.go`

**[R10] Contrato de desenvolvimento e fronteiras do repositório.**  
`https://github.com/A1b3rt0M3rcad0/wos/blob/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/AGENTS.md`

**[R11] Roadmap e estado declarado de implementação.**  
`https://github.com/A1b3rt0M3rcad0/wos/blob/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/ROADMAP.md`

**[R12] ADR-016: autoridade temporal de leases.**  
`https://github.com/A1b3rt0M3rcad0/wos/blob/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/docs/adr/0016-postgres-lease-time-authority.md`

**[R13] Skill de coordenação existente.**  
`https://github.com/A1b3rt0M3rcad0/wos/blob/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/packages/wos-skill/skills/wos-coordination/SKILL.md`

**[R14] Pacotes presentes na baseline.**  
`https://github.com/A1b3rt0M3rcad0/wos/tree/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/packages`

**[R15] Adapter web existente.**  
`https://github.com/A1b3rt0M3rcad0/wos/tree/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/packages/wos-api/web`

**[R16] Go module e dependências declaradas.**  
`https://github.com/A1b3rt0M3rcad0/wos/blob/bcd1714ad7cded666890d0b5c9f711772c4f8d7a/go.mod`

### 31.3 Referências de formato

**[E01] YAML Language Development Team — YAML 1.2, revisão 1.2.2.** Referência de sintaxe; o perfil contratual restrito deste plano é uma escolha do WOS.  
`https://yaml.org/spec/1.2.2/`

**[E02] RFC 8785 — JSON Canonicalization Scheme (JCS).** Referência para a canonicalização dos objetos usados nos digests.  
`https://www.rfc-editor.org/rfc/rfc8785`

---

**Síntese final:** o WOS mantém a tarefa e a autoridade de execução; o contrato registra uma aquisição temporária e exclusiva; MCP/HTTP/SDK e CLI/YAML expressam o mesmo protocolo; progresso e prova são persistidos sem encerrar a reserva; expiração, revogação administrativa ou finalização aceita são as únicas formas de encerramento.
