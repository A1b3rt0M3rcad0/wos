# WOS — Plano de evolução para contratos assinados, profiles independentes e workspace mínimo

**Produto:** WOS — Woobe Outcome Server  
**Natureza:** planejamento de implementação; não é uma declaração de funcionalidades já entregues.  
**Data da análise:** 8 de outubro de 2026  
**Repositório:** `A1b3rt0M3rcad0/wos`  
**Branch de referência:** `master`  
**Commit analisado:** `b75aa3e0ca6d649cafadf8450d9b5d606367dda4`  
**Árvore Git:** `e51f8d10d4be81030a3220b8383121f2d45084c5`  
**Versão declarada no snapshot:** `0.2.0` ([S19])  
**Protocolo proposto neste documento:** `signed_contracts_v2`  
**Formato local proposto:** `schema_version: 2`  
**Binários preservados:** `wos` para servidor/administração; `wosctl` para o cliente API-only.

> **Objetivo:** permitir que agentes independentes trabalhem no mesmo projeto, cada um com seu profile, sua credencial de autenticação/autorização e sua chave de assinatura. Cada participante mantém localmente apenas `profile.yaml` e os contratos pendentes em `contract/`. O servidor emite contratos assinados, recebe entregas ou revisões assinadas, aplica a política de aceitação, preserva o histórico e confirma quando o CLI pode excluir cada contrato local.

## Sumário

1. [Decisão executiva e limites](#s01)
2. [Baseline da master e análise de diferenças](#s02)
3. [Requisitos consolidados e invariantes](#s03)
4. [Modelo de domínio e estados](#s04)
5. [Workspace mínimo, profiles e ambientes](#s05)
6. [Credenciais, permissões e política de revisão](#s06)
7. [Protocolo de assinaturas e confiança](#s07)
8. [Formato dos documentos locais](#s08)
9. [Persistência local mínima, recuperação e exclusão](#s09)
10. [Aquisição múltipla, leases e concorrência](#s10)
11. [Fluxos completos pelo CLI](#s11)
12. [Casos de uso, API, MCP e SDK](#s12)
13. [Persistência no servidor e migrações SQL](#s13)
14. [Eventos e triggers](#s14)
15. [Interface humana, skills e integração com agentes](#s15)
16. [Migração do protocolo e do workspace existentes](#s16)
17. [Erros e procedimentos de recuperação](#s17)
18. [Mapa de alterações por pacote](#s18)
19. [Ondas de implementação e sequência de PRs](#s19)
20. [Matriz de testes e rastreabilidade](#s20)
21. [Medição de contexto, custo e desempenho](#s21)
22. [Critérios finais de aceite e publicação](#s22)
23. [Instrução de execução para agentes de desenvolvimento](#s23)
24. [Fontes e limites da análise](#s24)

<a id="s01"></a>
## 1. Decisão executiva e limites

### 1.1. Evoluir a implementação existente, não reconstruí-la

A `master` examinada já possui o núcleo de contratos, o cliente `wosctl`, submissões, checkpoints, recuperação idempotente, SQL, HTTP/MCP, SDK Go, interface humana e distribuição preparada. O trabalho deste plano é complementar e, em pontos específicos, substituir decisões de experiência local e de aceitação. Não reiniciar C01–C12 nem criar um segundo servidor de contratos. [S01][S02][S03]

A evolução contém quatro mudanças estruturantes:

1. **Identidade operacional por profile:** seleção independente por processo/chamada, credenciais separadas e referências seguras às chaves.
2. **Documentos assinados:** emissão pelo servidor, entrega pelo executor e decisão pelo revisor, todos vinculados a identidades e versões exatas.
3. **Passagem de responsabilidade para revisão:** o executor pode encerrar sua obrigação e desaparecer; a revisão não depende da manutenção de seu lease.
4. **Um arquivo local por contrato pendente:** progresso, evidências, decisão e metadados transitórios convivem no próprio documento; o arquivo é removido somente após confirmação durável e inequívoca do servidor.

### 1.2. Decisões adotadas

| Tema | Decisão deste plano |
|---|---|
| Fonte de verdade | Estado confirmado no WOS Server; rascunhos ainda não enviados existem somente no cliente. |
| Estrutura por agente | `.wos/profiles/<nome>/profile.yaml` e `.wos/profiles/<nome>/contract/`. |
| Configuração compartilhada | `.wos/project.yaml`, sem segredos e sem profile ativo global mutável. |
| Contratos locais | Um YAML por contrato, seja de execução ou de revisão. |
| Histórico local | Não criar diretórios permanentes de checkpoints, submissions, reviews, receipts, outbox ou cache. |
| Recuperação | Embutir intenção pendente no contrato; antes de existir contrato, usar seção técnica limitada do `profile.yaml`. |
| Assinatura | Ed25519; protocolo DSSE; payloads WOS serializados de forma canônica com JCS. |
| Geração de chave privada | Local no CLI por padrão; importação segura de chave preexistente opcional. |
| Autorização | Aplicada no servidor; permissões não são declaradas pelo YAML como fonte de autoridade. |
| Conclusão direta | Permitida somente pela interseção de credencial, Principal, escopo e política efetiva. |
| Revisão obrigatória | Entrega encerra contrato de execução como `delivered`; abre caso de revisão e bloqueia nova execução ordinária. |
| Correções | Novo contrato de execução, vinculado à decisão e à submissão anterior; nunca reativar contrato terminal. |
| Exclusão local | Automática após recibo confirmado que autorize a remoção da versão exata do arquivo. |
| Execução de agentes | Responsabilidade externa: host, Codex, Claude, Woobe ou outro consumidor. |
| Triggers | Persistem sinais após fatos confirmados; não iniciam agentes por conta própria. |
| Compatibilidade | Cutover explícito por Namespace; leitura histórica preservada; sem downgrade silencioso. |

### 1.3. O que não entra nesta evolução

Não implementar um scheduler de agentes, seleção automática de modelos, uma IDE, um motor de scripts, integração proprietária com cada runtime, execução de testes pelo servidor, armazenamento arbitrário de arquivos de projetos ou um sistema de merge Git. Também não transformar este plano em uma entrega de billing, alteração de licença ou infraestrutura comercial de Cloud.

O WOS continuará independente da Woobe. Um backend de produto continua proprietário do seu domínio de negócio. O estado pedagógico da Lipo, por exemplo, não passa a pertencer ao WOS.

As demonstrações multiagentes não dependerão de nomes comerciais específicos de modelos. A disponibilidade de um modelo e a capacidade de o host iniciar subagentes serão verificadas no ambiente do piloto. Configurações de Codex sugeridas em conversas anteriores não constituem requisitos técnicos validados deste plano.

<a id="s02"></a>
## 2. Baseline da master e análise de diferenças

### 2.1. Evidência examinada

O snapshot foi fixado pelo SHA da `master`; os links de código ao final apontam para esse commit. Foram examinados documentos operacionais, ADRs, roadmap e trechos centrais de implementação. Esta análise **não executou novamente a suíte de testes**, não verificou cada linha do repositório e não equivale a uma auditoria criptográfica. [S00–S17]

O relatório atual de contratos registra verificação local de **509 casos/subcasos**, quatro jornadas Chromium e testes de distribuição sobre o source `2ebe93fa18cbc2ec3d9570fd2725cec5ff9d3ddf`. Esse source de teste não deve ser substituído pelo SHA da `master` na narrativa. O mesmo relatório diferencia testes locais, Windows nativo pendente, publicação externa e integração efetiva com runtimes. São evidências registradas pelo projeto, não execuções realizadas nesta análise. [S03]

### 2.2. Inventário diferencial

| Área | Existe no snapshot | Complemento necessário |
|---|---|---|
| Outcome, Objective, Roadmap | Domínio persistente, planos versionados, critérios e dependências. | Preservar; melhorar somente projeções/contexto necessários ao novo ciclo. |
| WorkContract | Spec imutável, holder, execution ID, fencing, versões de contrato/lease, expiração. | Emissão assinada, política de aceitação vinculada e terminal de entrega para revisão. |
| Integridade | `SemanticDigest`, SHA-256 sobre JSON canônico. | Autenticidade criptográfica; digest não é assinatura. |
| Submissão | `WorkSubmission` imutável, material e evidências vinculadas. | Envelope assinado persistido e retorno final atômico. |
| Finalização | `FinalizeWorkContract` exige autoridade vigente do executor. | Revisor concluir a Task por sua própria autoridade, sem credencial/lease do executor. |
| Revisão | Assessments vinculados a submission; política independente por deployment/Outcome. | Caso e contrato de revisão; política por credencial/tarefa sem enfraquecer piso do deployment. |
| Autorização | Grants por Namespace/Principal; credenciais opacas revogáveis. | Capacidades restritivas por credencial, chave de assinatura registrada e escopos de operação. |
| CLI | `packages/wos-cli`, binário `wosctl`, cliente API-only. | Profiles reais, aquisição múltipla, assinatura, retorno e limpeza. |
| Configuração local | `WOSWorkspace` schema 1; campo `Profile`; root fixo `.wos/work`. | Schema 2 e diretórios independentes; o campo atual não comprova isolamento multi-profile. |
| Arquivos locais | Documentos separados de contrato/checkpoint/result; journals/receipts. | Consolidar em arquivo único por contrato e metadados limitados no profile. |
| Recuperação | Intenção gravada antes da chamada, receipt e replay, destination binding. | Preservar as garantias em armazenamento embutido; não remover idempotência por simplificação visual. |
| Concorrência local | Escritas atômicas, `os.Root`, locks cooperativos. | Locks por profile/operação/contrato sem bloquear outros agentes ou usar inode substituído como lock. |
| API/MCP/SDK | Casos de uso compartilhados e catálogo gerado. | Novos comandos tipados, envelope comum, verificações também em Application/embedded. |
| Eventos | Catálogo inclui `work_item.completed` e eventos de contratos. | Reaproveitar conclusão; acrescentar entrega, revisão e correção. |
| Storage | Memory, SQLite e PostgreSQL; migrações até `0019` no snapshot. | Novas tabelas/índices e transições compatíveis; nunca reescrever migrações aplicadas. |
| Publicação | Versão `0.2.0` preparada; manifests, testes e artefatos. | Aceite do novo source, Windows nativo e publicação autorizada; não inferir release publicada. |

Fontes do inventário: [S01][S02][S03][S04][S05][S06][S07][S08][S09][S10][S11][S12][S13][S14][S18].

### 2.3. Quatro incompatibilidades que precisam ser resolvidas explicitamente

**A. Apagar o arquivo após `SubmitWorkResult` não fecha a autoridade atual.** Hoje submissão e conclusão são operações distintas. O novo comportamento exige um comando de retorno que encerre a responsabilidade do executor e crie a revisão na mesma transação. Apenas acrescentar `os.Remove` ao CLI produziria uma experiência incorreta. [S07][S08][S15]

**B. O revisor atual não substitui o holder na finalização.** `contractForWrite` autoriza por Principal, execution ID e fencing do contrato. A nova revisão não deve usar a chave do executor nem um bypass administrativo para chegar a `DONE`. [S07][S08]

**C. A autorização atual não distingue duas chaves do mesmo Principal pela política de revisão pretendida.** É necessário introduzir limites por credencial e revalidá-los transacionalmente, sem trocar a identidade estável pelo nome da pasta ou pela chave pública. [S09][S10]

**D. A retirada das pastas de journal não permite retirar o journal lógico.** Antes da aquisição não existe ID de contrato; depois do envio pode existir commit remoto sem resposta. O profile e o contrato precisam carregar informação suficiente para reconciliar esses dois casos. [S11][S12]

### 2.4. ADRs a complementar

Manter os ADRs 018, 019 e 020 como registros históricos e criar novos ADRs que os complementem ou os substituam **somente no protocolo v2**. Os números definitivos devem ser reservados após revalidar a `master` no início da implementação.

Os temas obrigatórios são: profiles/workspace mínimo; assinatura/interoperabilidade; credenciais e independência de revisão; entrega final para revisão; recuperação e exclusão local; cutover v2. Não alterar os registros antigos para fazer parecer que descreviam a arquitetura nova desde o início. [S15][S16][S17]

<a id="s03"></a>
## 3. Requisitos consolidados e invariantes

### 3.1. Requisitos funcionais

| ID | Requisito |
|---|---|
| R01 | Criar profile pelo CLI usando credencial existente, com validação de identidade e escopo. |
| R02 | Gerar/importar chave de assinatura separada da credencial de autenticação. |
| R03 | Manter múltiplos profiles independentes no mesmo projeto. |
| R04 | Selecionar profile por chamada ou ambiente do processo, sem estado global compartilhado mutável. |
| R05 | Adquirir uma Task específica ou até N Tasks elegíveis, com contrato separado para cada uma. |
| R06 | Receber especificação e autoridade assinadas pelo servidor. |
| R07 | Registrar no próprio contrato progresso, artefatos, evidências e resposta a critérios. |
| R08 | Assinar e retornar o contrato pelo CLI, sem manipulação manual de criptografia pelo LLM. |
| R09 | Finalizar diretamente quando a credencial e a política autorizarem e os critérios forem satisfeitos. |
| R10 | Encaminhar a revisão obrigatória sem permitir que o executor contorne a política. |
| R11 | Entregar contrato de revisão a outro Principal, com credencial e assinatura próprias. |
| R12 | Solicitar correções e permitir novo contrato de execução com contexto focal das mudanças. |
| R13 | Aprovar a versão exata da entrega e finalizar a Task sem depender do executor. |
| R14 | Excluir automaticamente o contrato local após retorno aceito e responsabilidade encerrada. |
| R15 | Preservar arquivo e intenção em falha, resposta incerta ou rejeição. |
| R16 | Manter histórico, assinaturas, evidências, decisões e recibos no servidor. |
| R17 | Emitir eventos e triggers depois de conclusão confirmada, com deduplicação por consumidores. |
| R18 | Suportar os mesmos casos de uso por HTTP, MCP, SDK e cliente local. |
| R19 | Não permitir bypass por endpoints legados, interface humana ou chamadas embedded. |
| R20 | Oferecer contexto mínimo suficiente, expansão explícita e medição de custo. |
| R21 | Migrar os clientes/Namespaces existentes sem inventar assinaturas históricas. |
| R22 | Retomar após perda de sessão/cliente e preservar autoridade por fencing. |
| R23 | Emitir erros tipados com procedimento de recuperação sem expor segredos. |
| R24 | Provar o fluxo com processos independentes e storage real, inclusive no Windows anunciado. |

### 3.2. Invariantes não negociáveis

- Uma Task não possui dois contratos de execução efetivamente válidos ao mesmo tempo.
- Um contrato terminal nunca volta a ficar ativo. Retomar um contrato ainda ativo e adquirir um novo após expiração são operações diferentes.
- Um arquivo local não é uma capability bearer: copiá-lo não transfere autoridade.
- Principal, credencial de autenticação, signing key, profile, execution ID e ActorRef são identificadores distintos.
- Assinatura válida não dispensa autorização vigente, escopo, fencing, versões ou critérios.
- A assinatura vincula uma chave ao conteúdo; não atesta, sozinha, veracidade de teste, identidade do modelo ou qualidade do software.
- Uma entrega só usa avaliações da mesma versão de material e dos mesmos critérios.
- Revisão obrigatória não é removida por alteração de `profile.yaml`, escolha de outro nome local ou rotação de chave.
- Nenhuma operação gera `DONE` apenas porque o YAML contém `status: done`.
- Exclusão local só acontece para dados exatamente cobertos por confirmação durável.
- Apagar o YAML nunca apaga código, worktree, arquivos de evidência externos ou histórico do servidor.
- Uma falha de materialização local não provoca liberação compensatória silenciosa do contrato remoto.
- Tasks `DONE` não concluem automaticamente Objectives ou Outcomes.
- Os limites de privacidade/autorização não dependem do conteúdo de um prompt.

<a id="s04"></a>
## 4. Modelo de domínio e estados

### 4.1. Conceitos preservados e acrescentados

| Conceito | Responsabilidade | Estratégia |
|---|---|---|
| WorkItem | Trabalho operacional e lifecycle existente. | Evoluir sem substituir por uma nova Task paralela. |
| WorkContract | Compromisso de execução, spec e autoridade temporária. | Acrescentar protocolo/assinaturas/política e terminal `delivered`. |
| WorkSubmission | Material imutável apresentado pelo executor. | Reutilizar, preservando digest e autoria; vincular assinatura. |
| CriterionAssessment | Avaliação explícita de critério/revisão. | Reutilizar; vincular à entrega e à decisão de revisão ou autoavaliação autorizada. |
| ReviewCase | Processo de aceitação de uma submissão específica. | Novo agregado pequeno, pertencente ao mesmo escopo da Task. |
| ReviewContract | Autoridade temporária para decidir um ReviewCase. | Novo contrato especializado, sem fingir ser execução da Task. |
| ReviewDecision | Decisão assinada e imutável, com findings e assessments. | Nova entidade ligada ao ReviewCase/ReviewContract. |
| SigningKey | Chave pública registrada e seu ciclo de validade. | Novo registro; chave privada não pertence à persistência de domínio. |
| CredentialPolicy | Limites e política de aceitação associados a uma credencial. | Extensão da autorização existente. |
| SignedArtifact / AcceptanceReceipt | Envelope aceito e comprovante da operação. | Registros imutáveis, escopados e recuperáveis. |

Não criar um agregado `Agent` obrigatório apenas para reproduzir Principal. O WOS continua apto a representar humanos, agentes e aplicações pelo modelo de identidade já existente. Nome de modelo, runtime e sessão são metadados de correlação, nunca fonte de autorização.

### 4.2. Lifecycle da WorkItem

Preservar os valores existentes:

```text
backlog → todo → in_progress → done
                       └────→ cancelled
```

Exibir `needs_review`, `changes_requested`, `review_in_progress`, `ready_for_correction` e `authority_expired` como projeções derivadas de contratos, ReviewCase, dependências e bloqueios. Não misturar a apresentação do processo de revisão com os estados centrais da Task.

### 4.3. Lifecycle do WorkContract v2

```text
active ── retorno direto aceito ────────→ completed
   │
   ├──── retorno para revisão aceito ─→ delivered
   ├──── prazo efetivo atingido ──────→ expired
   └──── revogação administrativa ────→ revoked
```

`delivered` é um novo terminal: **a obrigação de execução foi cumprida mediante entrega recebida para revisão; a Task ainda não foi aceita como concluída**. É uma modalidade de encerramento por retorno aceito, não uma liberação voluntária sem resultado.

Não reutilizar `completed` com significado diferente, pois isso tornaria ambíguo o histórico atual. Não renomear contratos antigos ou emitir retroativamente o novo evento. O ADR novo deve explicar a extensão à regra anterior de finalização vinculada a `DONE`. [S05][S15]

### 4.4. Passagem de responsabilidade atômica

O retorno para revisão deve, em **uma transação**, registrar o material/evidências, persistir a submissão assinada, fechar o WorkContract como `delivered`, encerrar sua autoridade, abrir ReviewCase e registrar o bloqueio operacional de nova execução enquanto esse caso aguarda decisão.

Essa transição é o que autoriza a exclusão do contrato local. Não basta persistir uma submissão e deixar o contrato de execução ativo.

O ponteiro existente `CurrentContractID` pode continuar referenciando o último contrato para compatibilidade de leitura, mas **não pode ser usado como prova isolada de autoridade**. O estado efetivo do contrato e os marcadores de revisão determinam autorização/readiness. A projeção `CurrentLease` deve ser esvaziada ao entregar ou concluir.

Acrescentar à WorkItem referências suficientes para navegação e guards: `pending_review_case_id`, `latest_review_case_id` e, quando necessário, vínculo de correção pendente. Não duplicar o corpo de review, as evidências ou toda a máquina de estados na WorkItem.

### 4.5. ReviewCase e ReviewContract

ReviewCase vincula `work_item_id`, `source_contract_id`, `submission_id`, `submission_digest`, `spec_digest`, revisão da política, critérios e número da rodada. Seu alvo não pode ser trocado depois de aberto.

```text
pending → in_review → approved
             │
             ├─────→ changes_requested
             └─────→ pending            # reviewer expirou/revogou autoridade

pending/in_review → cancelled ou superseded  # comando explícito autorizado
```

ReviewContract possui holder, execution ID, fencing próprio, versões, emissão assinada e lease. Pode terminar por decisão aceita, expiração ou revogação. A expiração de revisão reabre a disponibilidade do **ReviewCase**, não a implementação da Task.

Na primeira entrega, haverá **um revisor decisor por rodada**. Quórum com vários revisores simultâneos não é requisito deste plano. O esquema pode guardar política revisionada para uma extensão posterior, sem implementar votação agora.

### 4.6. Aprovação e correção

**Aprovação:** o servidor verifica autoridade do reviewer, assinatura, submissão alvo, independência, assessments e elegibilidade atual. Persiste decisão, avaliações, conclusão da Task, fechamento da revisão, eventos e recibo atomicamente. O WorkContract original continua terminal `delivered`; não é reativado nem reescrito como uma nova execução.

**Correção:** a decisão guarda findings endereçáveis por ID e requisito. Fecha o contrato de revisão, limpa a pendência de revisão e permite uma nova aquisição de execução quando os demais guards permitirem. O novo WorkContract referencia a entrega anterior e os findings que devem ser respondidos. Seu fencing é maior que o de execuções antigas.

**Decisão inconclusiva:** o reviewer pode registrar decisão assinada `inconclusive`, explicar o que falta e encerrar o seu contrato de revisão. O caso fica pendente; não aprova critérios nem disponibiliza uma correção sem solicitação explícita. Para evitar ciclos cegos, a projeção indica o motivo e eventual Blocker autorizado.

**Aprovação impedida por mudança concorrente:** o comando atômico falha sem registrar uma aprovação parcial que o usuário interprete como `DONE`. O contrato local fica preservado. Após consulta, o reviewer pode corrigir sua decisão ou aguardar o bloqueio ser resolvido. Não criar um finalizador autônomo implícito.

### 4.7. Proteção da especificação durante revisão

Fechar o lease do executor não libera mudanças silenciosas na especificação. Enquanto houver ReviewCase aberto, alterações materiais de critérios, dependências ou escopo devem falhar ou passar por comando administrativo explícito que cancele/superseda a revisão e abra uma nova rodada.

Revisões de Roadmap que não modificam a obrigação contratada continuam possíveis pelas regras atuais. Findings de correção que ampliem o escopo além da especificação exigem replanejamento autorizado; o reviewer não ganha permissão de arquitetura ou planejamento por estar revisando.

<a id="s05"></a>
## 5. Workspace mínimo, profiles e ambientes

### 5.1. Estrutura canônica

```text
meu-projeto/
├── .wos/
│   ├── project.yaml
│   └── profiles/
│       ├── planner/
│       │   ├── profile.yaml
│       │   └── contract/
│       ├── executor_a/
│       │   ├── profile.yaml
│       │   └── contract/
│       │       ├── <contract-id-1>.yaml
│       │       └── <contract-id-2>.yaml
│       └── reviewer_a/
│           ├── profile.yaml
│           └── contract/
│               └── <review-contract-id>.yaml
├── AGENTS.md
└── src/
```

Quando o profile não tiver trabalho pendente, `contract/` fica vazio. Não manter cópias finalizadas, arquivos `.done`, subdiretório de histórico nem recibos permanentes no projeto.

Segredos protegidos no sistema operacional são armazenamento de credenciais, não um segundo histórico de trabalho. Arquivos temporários necessários a uma escrita atômica ou lock são artefatos técnicos transitórios; não formam uma nova estrutura operacional a ser administrada pelo agente.

### 5.2. `project.yaml`

Modelo proposto; IDs nos exemplos de uso devem ser substituídos por IDs reais do servidor.

```yaml
schema_version: 2
kind: WOSProject
name: meu-projeto
connection:
  server_url: https://wos.example.test
  expected_server_id: "<server-id>"
scope:
  namespace_id: "<namespace-uuid>"
  default_outcome_id: "<outcome-uuid>"
workspace:
  profiles_directory: profiles
```

O nome do projeto é apresentação. Um repositório pode executar trabalho de vários Outcomes; `default_outcome_id` é um default de endereçamento, não um novo boundary de autorização. Contratos sempre carregam escopo exato.

A raiz do workspace deve ser descoberta por caminho canônico ou `--workspace`. Perfis não podem trocar o servidor silenciosamente usando uma URL recebida no contrato. Vínculos de confiança aprovados na criação do profile devem ser comparados em toda operação.

### 5.3. `profile.yaml`

```yaml
schema_version: 2
kind: WOSProfile
name: executor_a
binding:
  server_id: "<server-id>"
  server_origin: https://wos.example.test
  namespace_id: "<namespace-uuid>"
  principal_id: executor-a
  credential_id: "<credential-uuid>"
authentication:
  credential_ref: "keyring:wos/executor-a/api"
signing:
  key_id: "<registered-key-id>"
  private_key_ref: "keyring:wos/executor-a/signing"
  public_key_fingerprint: "sha256:<fingerprint>"
lease:
  requested_ttl_seconds: 300
output:
  default_format: json
_local:
  schema_version: 1
  pending_operations: []
```

`binding` é uma restrição local de uso, não uma afirmação que o servidor deve confiar. O servidor identifica o Principal a partir da autenticação, consulta a chave pública registrada e rejeita divergências.

A seção `_local` é administrada pelo CLI e normalmente não aparece na saída destinada ao agente. Contém apenas operações ainda não reconciliadas que não cabem em contrato existente: aquisição, enrollment e operações de planejamento/administração iniciadas por esse profile. Após confirmação/materialização, o registro é removido. Não guarda histórico.

### 5.4. Seleção e concorrência de profiles

Ordem de seleção:

```text
--profile explícito → WOS_PROFILE do processo → profile único inequívoco
```

Com vários profiles e nenhum seletor, uma mutação deve falhar com `profile_required`. Não escolher o primeiro diretório, não compartilhar “profile ativo” por arquivo e não herdar inadvertidamente o último usado por outro chat.

`WOS_WORKSPACE` pode endereçar a raiz do projeto. O CLI resolve referências a segredos apenas para o profile selecionado; não carrega todos os profiles ou um `.env` contendo as chaves de todos os agentes.

Em múltiplos chats que compartilham o mesmo ambiente de ferramentas, a seleção segura é usar `--profile` em cada chamada. Uma variável de ambiente só isola processos realmente distintos; dois chats do mesmo host não necessariamente têm ambientes separados.

### 5.5. Credenciais e filesystem

Suportar `env:` para compatibilidade com a configuração atual e `keyring:` como backend preferido do cliente interativo. Para containers/CI, permitir referência explícita a secret montado fora do repositório, sujeito a permissões/ACL e verificação de caminho. Não implementar arquivo de senha em claro como fallback silencioso do keyring.

Não receber API token por flag cujo valor fique exposto em histórico/listagem de processos. Oferecer prompt sem eco e `--token-stdin`. Importar signing key por entrada dedicada e protegida; nunca colar a chave no contrato nem retorná-la no JSON de diagnóstico.

Criar nomes de profile sob allowlist, sem separadores, `..`, caracteres reservados do Windows ou colisões por caixa. Reutilizar as proteções de confinamento existentes, ampliando os testes. `profile remove` deve recusar exclusão enquanto houver contratos/intenções pendentes, salvo reconciliação explícita; remoção local não revoga ou conclui trabalho remoto. [S11]

### 5.6. Git e isolamento de execução

Adicionar `.wos/profiles/` ao ignore por padrão. `project.yaml` pode ser versionado somente se seu conteúdo for apropriado para o repositório; o CLI não deve sobrescrever `.gitignore` inteiro. Nunca versionar credenciais, drafts de entregas, assinaturas pendentes ou `_local`.

Para agentes que editam código em paralelo, recomendar worktrees/branches independentes fora de `contract/`. O CLI não apaga worktrees quando limpa um contrato. A revisão aponta para commit ou artefato imutável, não para o HEAD mutável do diretório de outro agente.

Profiles são separação operacional, não isolamento de segurança entre processos da mesma conta. A integração que exigir isolamento forte deverá provisionar sandbox, usuário, container ou serviço de assinatura com autorização independente.

<a id="s06"></a>
## 6. Credenciais, permissões e política de revisão

### 6.1. Modelo de autorização efetiva

A política não deve ser uma string `role: reviewer` confiada ao YAML. Aplicar no servidor:

```text
operações permitidas =
    grant vigente do Principal no Namespace
  ∩ limites da credencial autenticada
  ∩ escopos autorizados para Outcome/Task/operação
  ∩ autoridade vigente do contrato
  ∩ regras atuais de domínio
```

A exigência de revisão é uma restrição adicional. A política efetiva usa a condição mais restritiva entre deployment, Namespace, Outcome/Task, credencial de aquisição e eventual endurecimento posterior. A política fixada na aquisição é um piso: trocar por uma credencial mais permissiva antes de enviar não rebaixa um contrato que nasceu com revisão obrigatória.

Um administrador pode alterar uma política, mas a aplicação a compromissos em andamento exige comando explícito e histórico. Endurecimento de política passa a impedir conclusão direta; relaxamento não remove silenciosamente a obrigação já contratada.

### 6.2. Perfis de permissão sugeridos

Os nomes abaixo são presets de provisionamento, não tipos fixos de agentes:

| Preset | Capacidades | Limites |
|---|---|---|
| `planner` | Ler contexto; criar/editar planejamento, critérios e dependências quando permitido. | Não recebe execução, aprovação ou administração automaticamente. |
| `executor_direct` | Adquirir, renovar, sincronizar, retornar execução e solicitar conclusão direta. | Não ignora evidências/criteria gates; não aprova trabalho de terceiros por padrão. |
| `executor_reviewed` | Adquirir, renovar, sincronizar e retornar para revisão. | Não pode finalizar diretamente mesmo que declare `done`. |
| `reviewer` | Adquirir revisão; avaliar evidências; aprovar ou pedir correções. | Independência verificada pelo servidor; não administra contratos de terceiros. |
| `operator` | Administrar políticas, revogar e reconciliar quando expressamente autorizado. | Intervenção administrativa não é assinatura do executor/revisor nem prova de cumprimento. |

Introduzir permissões granulares, por exemplo:

```text
work.contract.acquire                 # existente, manter nome
work.contract.return                  # nova
work.contract.complete_direct         # nova
work.contract.revoke                  # existente, manter nome
work.review.acquire                   # nova
work.review.decide                    # nova
identity.signing_key.enroll            # nova e limitada por enrollment
identity.signing_key.rotate            # nova
identity.signing_key.revoke             # nova, conforme alvo e autoridade
```

Manter permissões atuais para leitura, planejamento e registros. Não renomear em massa o catálogo misto de `:` e `.` durante esta evolução. Não mapear automaticamente `work:write` ou `conclusion:write` para conclusão direta v2. A migração exige concessão explícita. [S09]

### 6.3. Política associada à credencial

Acrescentar uma política restritiva endereçada por `credential_id`, separada do segredo/digest do token:

```yaml
credential_policy:
  revision: 1
  permitted_operations:
    - state:read
    - work.contract.acquire
    - work.contract.return
  acceptance_floor: independent_review
  allowed_outcome_ids: []      # sem filtro adicional quando explicitamente autorizado
  allowed_signing_key_ids:
    - "<key-id>"
  max_active_work_contracts: 3
```

A ausência de política em credencial legada **não** deve significar poderes v2 completos. Definir estado de compatibilidade explícito. Uma credencial restritiva também não amplia um grant do Principal.

Autenticação deve disponibilizar internamente `credential_id` e revisão da política, além do Principal, Namespace, ActorRef e digest já existentes. Esses campos vêm do servidor. Revogação do token, do grant ou da chave de assinatura deve ser revalidada no snapshot transacional antes do commit.

### 6.4. Independência de revisão

Para o fluxo individual por tarefa, um reviewer não pode ter participado da execução da mesma Task/linha de correção que está aceitando. Consultar os fatos persistidos, não somente o último holder. Trocar credencial, chave, profile, ActorRef ou execution ID não muda o Principal.

Preservar o deployment que já exige independência de **todo o Outcome**: essa restrição é mais forte e continua prevalecendo. Não convertê-la automaticamente para independência apenas por Task. [S06]

Dois Principals controlados pelo mesmo humano são identidades diferentes, mas não comprovam independência organizacional. O modo inicial garante separação entre Principals; instalações que precisem de separação entre pessoas/equipes devem usar vínculo administrativo adicional, como `review_separation_group`, e validar esse grupo. Não anunciar resistência a identidades fictícias criadas pelo próprio administrador.

### 6.5. Cadastro inicial e rotação de chaves

O fluxo de criação do profile utiliza credencial já emitida ou provisionada por administrador. O profile não concede a si mesmo permissões.

O primeiro registro de chave requer um **enrollment autorizado**, vinculado a Namespace, Principal e finalidade, com nonce de uso único e prazo curto. O CLI prova posse da chave privada assinando o desafio. O challenge só contém material público e pode ser persistido transitoriamente no profile.

Não permitir que qualquer possuidor de uma API Key substitua livremente a chave pública de assinatura. Isso reduziria os dois controles a um único segredo: o atacante usaria o token para cadastrar sua própria chave. Primeiro enrollment depende de autorização administrativa; rotação depende de chave anterior válida ou recuperação administrativa explícita.

Depois do registro, o servidor armazena somente a chave pública, fingerprint, vínculos, algoritmo, datas e estado. Em rotação, manter chaves públicas históricas para verificação das entregas antigas. `retired` pode significar “não usar para novas operações”; `revoked` identifica revogação administrativa. O estado histórico de validação no momento da aceitação é preservado, sem fingir que toda assinatura antiga continua autorizando operações atuais.

<a id="s07"></a>
## 7. Protocolo de assinaturas e confiança

### 7.1. Escolha técnica

Adotar **Ed25519** por meio de implementação estabelecida, preferencialmente `crypto/ed25519` da biblioteca padrão Go, com testes de vetores e validação de tamanho antes de chamar APIs que possam falhar com entrada inválida. A RFC 8032 e a documentação Go são referências técnicas; não implementar primitivas criptográficas próprias. [N01][N02]

Adotar **DSSE 1.0.2** como protocolo de assinatura: autentica o tipo de payload e os bytes da mensagem. A especificação separa protocolo de formato de armazenamento, o que permite uma apresentação YAML legível sem alterar a assinatura. O `keyid` do envelope é apenas uma indicação de busca e não uma autorização. [N03][N04]

Preservar **JCS/RFC 8785** para gerar bytes canônicos dos payloads WOS. O WOS já usa JSON canônico para digest; a nova assinatura é uma garantia distinta. Não normalizar Unicode silenciosamente, não perder precisão numérica e não mudar a regra histórica de digests v1. [S05][N05]

A combinação é uma decisão de protocolo WOS: DSSE não exige JCS. Aqui JCS serve à equivalência entre a apresentação YAML e o payload tipado, enquanto DSSE define o que é assinado. A versão de protocolo, os tipos de payload e os vetores completos devem ser publicados antes da implementação dos consumidores.

### 7.2. Material assinado

| Documento lógico | Assinante | Vinculações mínimas |
|---|---|---|
| Especificação emitida | WOS Server | Servidor, escopo, Task, contrato, spec, digest, protocolo. |
| Grant de autoridade | WOS Server | Holder, chave/política permitida, contrato, execution ID, fencing, lease/revisões, spec digest. |
| Retorno de execução | Executor | Identidade autenticável, key ID assinado, autoridade, material/evidências, critérios, intenção e versões. |
| Decisão de revisão | Reviewer | ReviewCase/contrato, submission ID/digest, assessments, decisão e findings. |
| Recibo de aceitação | WOS Server | Identidade, request/idempotency/digest, efeitos confirmados e autorização de limpeza. |

Separar assinatura da especificação e assinatura do grant resolve a renovação: renovar o lease não altera a spec nem exige reconstruir seu histórico. O CLI substitui somente o grant e metadados de estado depois da resposta confirmada. Uma tomada de execução muda execution ID/fencing e exige novo grant.

Uma nova assinatura de servidor sobre o mesmo snapshot após rotação não deve modificar o digest da especificação. Os registros antigos permanecem verificáveis pela chave pública histórica.

### 7.3. Tipos de payload

Definir tipos distintos e versionados; nomes abaixo são identificadores de protocolo, não dependência de um domínio web:

```text
application/vnd.wos.contract-spec.v2+json
application/vnd.wos.contract-authority.v2+json
application/vnd.wos.work-return.v2+json
application/vnd.wos.review-return.v2+json
application/vnd.wos.acceptance-receipt.v2+json
application/vnd.wos.key-enrollment.v2+json
```

Não aceitar um payload assinado como `work-return` em um comando de revisão. O algoritmo permitido é definido pelo protocolo e pelo registro de chave; não pelo cliente. Não aceitar `none`, chaves públicas vindas no próprio contrato como fonte de confiança, URLs de descoberta de chave fornecidas pelo documento ou algoritmos alternativos sem versão/capability explícita.

### 7.4. Envelope de transporte e representação local

No transporte, usar o envelope DSSE convencional:

```json
{
  "payloadType": "application/vnd.wos.work-return.v2+json",
  "payload": "<base64 dos bytes canônicos>",
  "signatures": [
    {"keyid": "<chave-registrada>", "sig": "<base64 da assinatura>"}
  ]
}
```

No primeiro protocolo WOS, exigir uma assinatura de identidade por mensagem. A rotação mantém um conjunto de chaves confiáveis, sem introduzir quórum de assinaturas.

No YAML, permitir `payload` legível como mapping e `proof` separado. O codec tipado transforma essa representação nos mesmos bytes e envelope. Essa apresentação local **não deve ser anunciada como o JSON envelope DSSE literal**; é uma representação WOS equivalente, com vetores de ida e volta.

O servidor verifica os bytes recebidos e entrega à Application o mesmo objeto derivado desses bytes. Não verificar um envelope e executar um `command` paralelo, não assinado, enviado ao lado dele. Campos extras do envelope DSSE não alteram autoridade; o payload WOS tem schema estrito e rejeita campos desconhecidos/duplicados.

### 7.5. Conteúdo de um retorno

O payload de retorno inclui, no mínimo:

```text
protocol_version, operation_kind, request_id, idempotency_key
server_id, namespace_id, outcome_id
principal_id, signer_key_id
contract_id, work_item_id, execution_id, fencing_token
spec_digest, authority_digest, expected_contract_version
expected_lease_version, expected_work_item_version, policy_revision
completion_intent: auto | direct | review
result/evidence/artifact/criterion-response material
supersedes_submission_id quando aplicável
```

Para review, trocar o material pelo alvo exato e pela decisão: `review_case_id`, `review_contract_id`, `submission_id`, `submission_digest`, versões, findings e assessments.

Fencing permanece string decimal exata. Versões novas que possam ultrapassar a faixa interoperável de números JSON devem ter representação exata definida no DTO v2; não alterar DTOs v1. URLs e referências são dados, não instruções para o servidor executar ou buscar conteúdo.

### 7.6. Verificação na operação nova

Executar validação em camadas:

1. Aplicar limites de bytes e parser; autenticar transporte e endereçamento.
2. Encontrar chave registrada em conjunto autorizado; verificar assinatura e tipo de mensagem.
3. Derivar comando exclusivamente do payload validado; checar correspondência com rota/escopo autenticado.
4. Abrir a pipeline transacional existente; verificar idempotência e adquirir os guards necessários.
5. Revalidar credencial, grant, key status, política e escopo usando estado coerente.
6. Revalidar lease pelo relógio autoritativo, execution ID, fencing, versões e digest da spec.
7. Validar referências, critérios e semântica da transição.
8. Persistir material, efeitos, envelopes, eventos, outbox e recibo no mesmo commit.

Verificar assinatura antes da transação evita segurar locks durante parte do processamento, mas o estado da chave/permissão é revalidado dentro dela. Nenhum booleano `signature_verified: true` fornecido pelo consumidor deve substituir essa verificação. Application/embedded precisam de mecanismo verificável, não somente middleware HTTP.

### 7.7. Confiança no servidor

O profile associa credenciais a `server_id`, origem aprovada e trust set do emissor. O primeiro contato usa TLS e onboarding explícito; operações não interativas podem fornecer fingerprint/pin de fonte confiável. Um `project.yaml` vindo de um repositório desconhecido não é autorização para enviar o token a outro destino.

Não seguir redirects de mutação que levem credenciais ou envelopes a outro host. Renovação, importação ou recovery devem recusar destino diferente até reconfiguração explícita. A identidade do servidor precisa ser persistente entre reinícios/réplicas, não um UUID gerado a cada boot.

Chaves privadas de emissão pertencem à operação do servidor. Todas as réplicas autorizadas verificam o mesmo trust set; a emissão usa um signer configurado. Para a primeira entrega, utilizar chave local/secret montado já disponível em memória durante a transação. Não fazer chamadas a KMS remoto dentro da transação de domínio. Um signer remoto posterior exige desenho próprio de emissão recuperável.

### 7.8. Replay, revogação e tempo

Distinguir nova operação de consulta a operação confirmada. Um retorno novo assinado com chave revogada ou autoridade expirada falha. Um recibo histórico confirmado pode ser consultado com uma **credencial atualmente autorizada**, sem exigir que a chave que assinou no passado ainda aceite operações novas.

O replay de intenção já confirmada não executa a transição novamente nem renova lease. Preserva o resultado e a validação registrada no primeiro commit; antes de divulgá-lo, revalida acesso atual ao recibo. Se a credencial original perdeu acesso, recuperação administrativa ou outra credencial permitida não pode ser simulada como o agente antigo.

O relógio do agente não prolonga autoridade. Data de assinatura pode ser metadado; a validade operacional depende do grant e do tempo autoritativo do servidor. Renovar ou tomar execução depois de preparar um retorno exige reconciliar a intenção anterior antes de assinar outra intenção com nova autoridade.

### 7.9. Definição exata de digests e vinculação da intenção

Fixar estas regras no schema/vetores, evitando circularidade:

```text
spec_digest      = SHA-256(JCS(spec normalizada conforme sua versão))
authority_digest = SHA-256(JCS(payload do grant de autoridade))
request_digest   = SHA-256(bytes canônicos do payload de retorno)
```

O digest da spec não inclui o próprio campo `spec_digest`, sua proof, grant ou `_local`. O cabeçalho do documento é coberto pela assinatura de emissão, mesmo quando não integra o digest da spec. A assinatura utiliza a mensagem definida pelo protocolo DSSE; o digest de consulta não substitui a mensagem assinada nem seleciona uma variante pré-hashed por acidente.

Para idempotência v2, calcular a identidade/fingerprint a partir do comando e payload verificado, preservando o contrato exato de escopo e Principal. Não usar a representação base64 ou ordem das propriedades do envelope como intenção de negócio. Mudanças de `signer_key_id`, CAS, autoridade, material ou finalidade produzem intenção diferente e não podem ser reapresentadas com a chave idempotente original.

Todos os payloads assinados identificam o signer e o `signer_key_id` dentro do conteúdo autenticado. Comparar esses valores à chave pública efetivamente verificada e à identidade/finalidade esperada; o `keyid` externo serve somente para localizar candidatos autorizados. A serialização de arrays, campos ausentes, strings temporais e números é definida pelo DTO versionado, não por heurística do parser. Datas de operação/nonce são congeladas na preparação, nunca regeneradas a cada retry.

<a id="s08"></a>
## 8. Formato dos documentos locais

### 8.1. Um arquivo, responsabilidades separadas

Um único YAML pode conter documentos lógicos diferentes. Isso não significa que o executor edita o que o servidor assinou.

```text
ContractFile
├── schema_version / kind
├── issued
│   ├── specification: payload + proof      # servidor; imutável
│   └── authority: payload + proof          # servidor; substituído só em refresh autorizado
├── execution OU review                    # rascunho editável pelo agente
└── _local                                 # CLI; intenção congelada, binding e recibo transitório
```

Não manter simultaneamente `execution` e `review` para o mesmo contrato. O discriminador do contrato determina o schema. Evidências da execução anterior que o revisor recebe são parte do material de entrada assinado ou referências verificáveis do caso; a decisão nova fica em `review`.

### 8.2. Exemplo de contrato de execução

Exemplo ilustrativo de estrutura v2, abreviado; placeholders não são payloads válidos para envio. O schema gerado a partir de DTOs será a referência executável.

```yaml
schema_version: 2
kind: WOSContractFile
issued:
  specification:
    payload:
      protocol_version: 2
      contract_kind: execution
      server_id: "<server-id>"
      namespace_id: "<namespace-uuid>"
      outcome_id: "<outcome-uuid>"
      work_item_id: "<work-item-uuid>"
      contract_id: "<contract-uuid>"
      spec_digest: "sha256:<digest>"
      spec:
        outcome_intent: Disponibilizar autenticação da aplicação
        title: Rejeitar tokens expirados
        instructions:
          - Implementar a validação no módulo de autenticação existente.
          - Não modificar o contrato público do endpoint.
        constraints:
          - Preservar a resposta documentada de acesso não autorizado.
        criteria:
          - id: "<criterion-uuid>"
            revision: 1
            description: Token expirado é rejeitado por teste de integração.
            verification_mode: evidence_review
        context_refs: []
        dependencies: []
    proof:
      payload_type: application/vnd.wos.contract-spec.v2+json
      key_id: "<server-signing-key>"
      signature: "<base64>"
  authority:
    payload:
      protocol_version: 2
      contract_id: "<contract-uuid>"
      spec_digest: "sha256:<digest>"
      principal_id: executor-a
      execution_id: "<execution-uuid>"
      fencing_token: "7"
      contract_version: 1
      lease_version: 1
      expires_at: "2026-10-08T20:00:00Z"
      acceptance_mode: independent_review
      acceptance_policy_revision: 3
    proof:
      payload_type: application/vnd.wos.contract-authority.v2+json
      key_id: "<server-signing-key>"
      signature: "<base64>"
execution:
  progress:
    summary: Implementação concluída; aguardando envio.
    completed:
      - Validação de expiração implementada.
    pending: []
    next_action: Retornar a entrega ao WOS.
  result:
    summary: Tokens expirados passam a retornar a resposta documentada.
    repository:
      base_commit: "<sha-base>"
      produced_commit: "<sha-exato>"
      dirty: false
    artifacts:
      - local_key: commit-implementation
        type: git_commit
        reference: "<repositorio-e-sha>"
        source_version: "<sha-exato>"
    evidence:
      - local_key: integration-test
        type: test
        description: Execução do teste de token expirado.
        artifact_local_key: commit-implementation
        source_ref: "<referencia-duravel-do-relatorio>"
        source_version: "<sha-exato>"
        checksum: "sha256:<checksum-do-relatorio>"
        observation:
          command: go test ./internal/auth/... -run TestExpiredToken
          exit_code: 0
    criterion_responses:
      - criterion_id: "<criterion-uuid>"
        criterion_revision: 1
        claimed_result: met
        evidence_local_keys:
          - integration-test
        rationale: A evidência corresponde ao commit submetido.
    addressed_findings: []
  completion_intent: auto
_local:
  draft_revision: 4
  prepared_return: null
  acceptance_receipt: null
```

`claimed_result` é a declaração do executor. No modo de revisão, não se torna automaticamente um assessment aprovado. No modo direto, a política pode autorizar autoavaliação explícita, desde que compatível com o `verification_mode` e demais critérios; o servidor registra quem a realizou.

### 8.3. Exemplo da parte editável de revisão

```yaml
review:
  decision: changes_requested
  summary: Falta testar a expiração exatamente no limite do tempo.
  criterion_assessments:
    - criterion_id: "<criterion-uuid>"
      criterion_revision: 1
      result: not_met
      evidence_refs:
        - "<evidence-uuid>"
      rationale: O teste cobre apenas um token expirado há vários segundos.
  findings:
    - local_key: expiry-boundary
      criterion_id: "<criterion-uuid>"
      description: O instante exato de expiração não foi validado.
      required_change: Adicionar teste determinístico com relógio controlado.
      acceptance_hint: A comparação rejeita now maior ou igual a exp.
```

Na aprovação, `decision: approved` exige assessments explícitos dos critérios necessários e evidências da submissão alvo. A simples string `approved` não comprova atendimento.

### 8.4. Evidências e materiais

Reutilizar `local_key`, os registros documentais e as verificações de escopo existentes. O retorno atômico v2 resolve as chaves locais para IDs de servidor na mesma transação e vincula o material aceito ao envelope assinado original. Os IDs novos gerados pelo servidor não são reescritos como se tivessem sido assinados pelo executor. O recibo liga o digest do retorno às entidades registradas.

Quando já houve `sync`, o contrato pode usar IDs de registros existentes junto de source version/checksum. Conteúdo alterado depois de sincronizado recebe novo registro e nova chave local; nunca atualizar evidência histórica em lugar.

Não embutir logs, binários ou datasets inteiros. Guardar observação/resumo limitado e referência durável com versão/checksum. Uma referência exclusivamente local, que o reviewer não possa recuperar, deve ser indicada como indisponível; não excluí-la junto com o contrato. O servidor não busca URLs externas dentro da transação.


O reviewer também pode produzir evidências complementares, por exemplo a execução independente de um teste. O retorno de revisão inclui registros documentais limitados e assinados, vinculados ao mesmo `submission_id`/digest e ao commit efetivamente revisado. Essas evidências pertencem à decisão de review: não são inseridas retroativamente no payload assinado do executor. Assessments podem referenciar material original ou material complementar admitido pela política, sempre com proveniência e alvo exatos. Estender a validação atual para distinguir essas duas origens, sem permitir evidências de outra submissão/rodada.

### 8.5. Limites e schemas

Preservar o perfil YAML restrito existente para os campos semânticos. Definir um schema 2 explícito para os campos técnicos adicionais. [S04][S16]

| Limite | Proposta inicial |
|---|---|
| Spec congelada | Preservar até 128 KiB; preferir referências. |
| Requisição remota total | Preservar teto de 256 KiB, incluindo envelope. |
| Payload assinado de retorno | Teto inicial de 180 KiB antes de base64, sujeito também ao teto total. |
| Documento local v2 completo | Até 1 MiB, para comportar apresentação, provas e uma intenção técnica congelada. |
| Profundidade do payload semântico | 16 níveis. |
| Nós semânticos YAML | 10.000, preservando proteção atual. |
| Escalares semânticos | Preservar 64 KiB; blob técnico codificado tem limite próprio derivado do envelope. |
| Registros documentais por operação | Preservar 100 no total; múltiplos syncs explícitos para volume maior. |
| Intenções de aquisição simultâneas por profile | Limite configurável e pequeno; default proposto de 10, sem histórico acumulado. |

Esses valores são limites de contrato propostos, não capacidade medida. O teto local maior não autoriza transmitir mais dados ao servidor nem colocar megabytes no contexto do modelo. `work show --for-agent` omite `_local` e as provas técnicas por padrão, sem omitir instruções/limitações relevantes.

Gerar schemas por DTOs; fixtures de ida e volta devem garantir que um contrato lido no Windows e Linux produz os mesmos bytes assinados quando o conteúdo semântico é o mesmo. Mudança de conteúdo de strings, inclusive Unicode distinto, não é “mera formatação” e deve invalidar a assinatura correspondente.

<a id="s09"></a>
## 9. Persistência local mínima, recuperação e exclusão

### 9.1. Estado local efêmero não significa ausência de gravação durável

A experiência do usuário contém apenas profile e contratos. Internamente, o CLI precisa persistir a intenção **antes** do envio. Caso contrário, uma falha após commit remoto pode duplicar a aquisição ou apagar uma entrega ainda não recebida.

Substituir o destino de armazenamento do mecanismo atual de `journal.go`, não suas garantias. O novo componente pode se chamar `PendingOperationStore`, com implementações embutidas no profile e no contrato, sem `.wos/outbox` ou `.wos/receipts`. [S12]

### 9.2. Antes de existir contrato

Para aquisição:

1. Obter lock técnico do profile para a preparação.
2. Gerar `request_id` e idempotency key; construir payload exato com destino/escopo/identidade.
3. Persistir em `profile.yaml` uma entrada pendente, com estado `prepared`.
4. Enviar a mesma intenção; uma resposta perdida mantém `sent_unknown`.
5. Receber e verificar documento emitido/recibo.
6. Persistir a resposta necessária na mesma entrada, estado `accepted_unmaterialized`.
7. Materializar o contrato por escrita atômica no diretório correto.
8. Só então remover a entrada pendente do profile.

Se o processo cair entre 6 e 7, recovery recria o contrato. Se cair entre 7 e 8, recovery compara identidade/digest do arquivo existente e não o sobrescreve com versão vazia. Nunca obter outra Task para compensar a perda do arquivo recém-adquirido.

Para planejamento e operações administrativas sem contrato, reutilizar o mesmo padrão limitado de intenção no profile. Segredos de provisionamento nunca são gravados nessa entrada: persistir referências ao secret store e identificadores públicos. Emissão de token com resposta única exige protocolo próprio de entrega/revogação, não reexposição do token por receipt.

### 9.3. Durante execução, assinatura e envio

A seção `_local.prepared_return` contém a intenção imutável, envelope assinado, bindings, digest do draft, versões esperadas e idempotency key. A chave privada não faz parte dessa seção.

```text
draft
  → prepared_signed
  → sent_unknown
  → accepted_confirmed
  → cleanup_pending
  → arquivo removido
```

Rejeição determinística mantém a intenção e o diagnóstico até reconciliação. Não transformar erro de transporte em rejeição sem confirmação. Uma intenção aceita historicamente não deve ser alterada para “corrigir” um erro posterior de filesystem.

Enquanto o retorno está incerto, o CLI deve recusar a criação de outro retorno com novo conteúdo/chave de idempotência. O agente pode produzir novas alterações em uma execução separada autorizada, mas não sobrescrever o único rascunho ainda não reconciliado.

`sync` e `renew` também precisam de intenção recuperável; não exigem assinatura individual de cada leitura/heartbeat. Para evitar corrida entre edição e atualização técnica, operações de CLI no mesmo contrato usam lock e compare-and-swap local. Nenhum lock local substitui as verificações remotas.


Ao preparar um retorno final, pausar mutações concorrentes de `renew`/`sync` para aquele contrato no cliente. O keepalive deve ignorar temporariamente contratos com retorno preparado ou resultado incerto e indicar essa condição ao host; não trocar o grant por baixo da assinatura congelada. `finish` reconcilia renovações pendentes, prepara e envia sob exclusão local por contrato. Uma renovação feita por outro consumidor ainda pode gerar conflito remoto, que exige reconciliação explícita. Não tratar o lock de um contrato como lock de todos os profiles.

### 9.4. Recibo e autorização de limpeza

Resposta proposta do retorno, com corpo assinado pelo servidor:

```json
{
  "protocol_version": 2,
  "request_id": "<request-id>",
  "idempotency_key": "<same-key>",
  "request_digest": "sha256:<signed-payload-digest>",
  "principal_id": "executor-a",
  "contract_id": "<contract-id>",
  "submission_id": "<submission-id>",
  "outcome_revision": "<exact-revision>",
  "accepted": true,
  "disposition": "accepted_for_review",
  "contract_status": "delivered",
  "work_item_lifecycle": "in_progress",
  "review_case_id": "<review-case-id>",
  "local_obligation_closed": true,
  "accepted_at": "<server-time>"
}
```

Para conclusão direta, `disposition: completed`, WorkItem `done`, WorkContract `completed`. Para decisão de revisão aceita, o recibo identifica o ReviewContract e a decisão. Todos carregam o vínculo exato ao retorno.

O cliente remove o arquivo somente se:

- a resposta/receipt é autêntica, escopada e corresponde à intenção pendente;
- a obrigação local daquele contrato foi encerrada, não apenas um checkpoint recebido;
- o conteúdo editável atual continua igual ao material coberto pelo retorno aceito;
- não há outra intenção local não reconciliada;
- o arquivo alvo é o arquivo regular esperado, dentro do profile correto.

A remoção é por arquivo, nunca `rm -rf contract/`. Se falhar, reportar `remote_committed: true, cleanup_pending: true`, preservar o recibo no mesmo arquivo e permitir nova tentativa. Isso não é uma falha do trabalho remoto.

### 9.5. Recuperação de resposta perdida

Recovery deve tentar o replay exato ou consulta de receipt autorizada. A ausência de receipt por expiração de retenção **não prova ausência de commit**. Complementar com consulta por `request_id`/digest vinculada ao registro durável de aceitação do contrato.

Se ainda assim não for possível provar a recepção, não excluir. Retornar `acceptance_unknown` e orientar reconciliação. A retenção mínima do registro de aceitação deve acompanhar a existência do contrato histórico; não depender exclusivamente do cache de idempotência.

### 9.6. Locks, atomicidade e falhas locais

Reutilizar confinamento, arquivos regulares, escrita em temporário no mesmo diretório, flush/rename e testes de plataforma. [S11]

O lock deve usar uma identidade estável derivada de caminho canônico, profile e contrato. Não bloquear o inode do próprio YAML e depois substituí-lo por rename, pois isso pode deixar dois processos acreditando que têm exclusividade sobre versões diferentes.

A implementação terá um port de mutex interprocesso: named mutex no Windows e mecanismo equivalente testado no Unix. Quando o Unix exigir um arquivo técnico de lock, ele reside no diretório de runtime do sistema, com permissões restritas, sem payload nem journal. Não o apagar enquanto a exclusividade depender de seu inode. Isso não cria histórico ou diretório operacional adicional em `.wos/`. Arquivos temporários de escrita no projeto são transitórios e tratados no recovery.

Não deixar lock do profile retido durante toda a execução da Task. Usá-lo somente para alterações do profile ou protocolo de materialização; usar lock do contrato para operações naquele documento. Planejar ordem de locks e testar deadlock: profile antes de contrato quando ambos forem necessários; processamento de rede de longa duração não mantém lock de todos os contratos.

Uma proteção baseada em locks cooperativos não impede um editor externo de alterar arquivos. Por isso, verificar novamente digest/identidade antes de sobrescrever ou excluir e abortar ao detectar edição concorrente. Proteger contra symlink/reparse, case collisions, hardlink indevido e troca de arquivo alvo conforme recursos testados de cada plataforma.

### 9.7. O que não pode ser prometido

Sem sincronização prévia, a destruição completa do disco perde o progresso ainda apenas local. Não há como o servidor reconstruir bytes que nunca recebeu. O CLI oferece `sync` explícito e o host pode chamá-lo periodicamente; o servidor preserva apenas o que confirmou.

Da mesma forma, exclusão do YAML é remoção do arquivo operacional, não apagamento forense de dados em SSD, backup ou logs do sistema. O requisito do produto é não acumular contratos finalizados no workspace.

<a id="s10"></a>
## 10. Aquisição múltipla, leases e concorrência

### 10.1. Semântica de quantidade

Estender o checkout atual com `--count N`. `count` significa número máximo de contratos adquiridos, não tamanho da página de candidatos. Preservar a distinção atual entre busca completa, página parcial e ausência de Task elegível. [S04][S13]

A primeira implementação usa **N aquisições independentes**, não uma transação distribuída de lote. Cada item tem idempotency key e resultado próprios. O CLI persiste o progresso do lote no profile e apresenta sucesso parcial explicitamente. Se ocorrer erro no item 3 de 5, os dois contratos anteriores continuam válidos e materializados.

Não fazer rollback por revogação dos contratos já adquiridos. Não repetir o lote inteiro com novas chaves depois de timeout. Uma busca vazia repetida com a mesma chave continua sendo a mesma busca; nova busca deliberada usa nova intenção.

### 10.2. Elegibilidade

Na aquisição, validar Task/Outcome ativos, dependências, blockers, prazo, ausência de execução válida e ausência de ReviewCase que impeça execução. Para correções, permitir nova execução da mesma Task `in_progress` quando a projeção indicar `ready_for_correction`.

A ordenação determinística já existente por prioridade, criação e ID deve ser preservada, salvo ADR específico. Não acrescentar um modelo de IA para escolher a próxima Task.

### 10.3. Quotas e fairness

Aplicar teto de contratos simultâneos por Principal e, opcionalmente, por credencial e Namespace. O teto de credencial nunca pode ultrapassar o teto do Principal, e criar profiles extras não deve contornar esse limite.

Default de produto sugerido para piloto: até 3 contratos de execução ativos por executor e 1 revisão ativa por reviewer, configurável pelo operador. Esses valores são políticas iniciais propostas, não limitação arquitetural nem plano comercial definitivo.

O host deve adquirir quantidade compatível com sua capacidade de execução/renovação. Reservar dezenas de Tasks sem processá-las não é benefício de paralelismo.

### 10.4. Renovação e retomada

Preservar os defaults existentes de lease enquanto não houver mudança administrativa explícita: 300 segundos por padrão, mínimo 30 e máximo de política inicialmente 3.600 segundos. Uma duração máxima total de posse pode ser acrescentada como política separada, nunca confundida com TTL de cada renovação. [S05]

`keepalive` permanece foreground e supervisionado pelo host. Não iniciar daemon oculto no `profile create` ou `checkout`. Para vários contratos, permitir selecionar `--all-active` e retornar resultado individual, respeitando CAS e perda de autoridade de cada um.

Retomada por mesma identidade durante lease ativo pode preservar a execução ou usar takeover explícito. Takeover muda execution ID/fencing, não estende TTL implicitamente. Depois de expiração, nova aquisição; o arquivo antigo é reconciliado e seu progresso pode ser importado como dado, nunca como autoridade antiga.

Qualquer retorno preparado antes de takeover/renovação que altere campos vinculados precisa ser reconciliado ou abandonado de forma explícita antes de nova assinatura. Não modificar a intenção congelada sob a mesma chave de idempotência.

### 10.5. Expiração não equivale a interrupção do processo

Um processo local pode continuar escrevendo código após perder o lease. O WOS recusará operações de estado obsoletas, mas não controla o filesystem externo. O host deve observar falhas de renovação e parar ou isolar a execução. Não afirmar que fencing do WOS impede side effects em Git, APIs ou bancos externos que não verifiquem esse fencing.

<a id="s11"></a>
## 11. Fluxos completos pelo CLI

> Os comandos de profile, signing e revisão apresentados abaixo são a **interface-alvo proposta**. O snapshot tem `wosctl`, mas não se deve executar estes exemplos como se todas as extensões já estivessem disponíveis. Valores `<...>` são substituídos pelos IDs reais e dados de provisionamento.

### 11.1. Preparar projeto e identidade

O administrador provisiona no WOS os Principals, grants, credenciais com política e enrollments. Não colocar a credencial administrativa nos profiles dos agentes de execução.

```bash
# Inicialização do workspace v2, preservando o binário de cliente existente.
wosctl init --server https://wos.example.test \
  --namespace <namespace-id> --outcome <outcome-id> \
  --workspace-schema 2

# Entrada protegida da API Key; chave privada gerada no secret store.
wosctl profile create executor_a \
  --token-stdin --generate-signing-key \
  --enrollment <enrollment-id>

wosctl --profile executor_a auth status --output json
wosctl --profile executor_a profile inspect --output json
```

A criação verifica a identidade antes de fixá-la no profile, obtém challenge de enrollment, prova posse, registra chave pública e valida a configuração final. Uma criação interrompida deixa estado explícito `enrollment_pending`, sem gerar outra chave indefinidamente ou perder a referência à chave já criada.

O comando `profile inspect` exibe Principal, servidor, Namespace, key ID, fingerprint e permissões efetivas resumidas; nunca tokens/chave privada. O preset de permissões vem do servidor, não de flags que o agente possa usar para elevar seus próprios privilégios.

### 11.2. Fluxo A — credencial com conclusão direta permitida

```bash
# Adquirir uma Task conhecida.
wosctl --profile executor_a work checkout <work-item-id> --version <version>

# Ou adquirir até três Tasks elegíveis.
wosctl --profile executor_a work checkout --next --count 3

# Ler o trabalho sem incluir envelope técnico no contexto.
wosctl --profile executor_a work show <contract-id> --for-agent

# Executar ferramentas do projeto fora do WOS, preencher execution no YAML.
# Sincronização intermediária é opcional, mas recomendada antes de longas interrupções.
wosctl --profile executor_a work validate <contract-id>
wosctl --profile executor_a work sync <contract-id>

# Caminho principal: prepara, assina, envia e limpa após confirmação.
wosctl --profile executor_a work finish <contract-id> --completion direct
```

O retorno direto v2 deve registrar evidências/artefatos novos, submissão, autoavaliações permitidas, conclusão, contrato `completed`, WorkItem `done`, eventos e receipt em **uma transação de domínio**. Não reutilizar a sequência atual do `finish` como se vários commits fossem atomicidade do retorno final. [S04][S08]

Os critérios podem demandar informação que o executor não possui. Nesse caso o comando falha com explicação, mantém o arquivo e não finge que “permissão completa” satisfaz uma avaliação externa.

Resultado humano esperado:

```text
Retorno aceito. Task concluída. Contrato encerrado.
Arquivo local removido.
```

Resultado JSON inclui IDs/digests e distingue `remote_committed` de `local_cleanup`. Os outros contratos do profile continuam intactos.

### 11.3. Assinar e enviar em passos separados

```bash
wosctl --profile executor_a work sign <contract-id> --completion auto
wosctl --profile executor_a work send <contract-id>
```

`sign` grava a intenção congelada no próprio contrato. `send` envia exatamente essa intenção. Se o agente editar o material depois de assinar, o CLI recusa o envio como versão divergente. Se a intenção já foi enviada e sua confirmação é desconhecida, não permite re-assinar sem recovery.

`work finish` é o atalho recomendado para a mesma sequência. Em modo v2, assinatura de retorno é obrigatória; `--sign` pode ser um alias de clareza, não um opt-in que permita retorno unsigned.

### 11.4. Fluxo B — credencial com revisão obrigatória

```bash
wosctl profile create executor_b \
  --token-stdin --generate-signing-key --enrollment <enrollment-id>

wosctl --profile executor_b work checkout --next --count 2
wosctl --profile executor_b work show <contract-id> --for-agent

# Executa, documenta evidências e retorna.
wosctl --profile executor_b work finish <contract-id> --completion auto
```

A política efetiva resolve `auto` para revisão. Uma solicitação explícita `direct` sem permissão retorna erro de autorização; o CLI não apresenta o resultado como conclusão direta.

O servidor aceita o retorno, fecha o WorkContract como `delivered`, abre ReviewCase, mantém WorkItem `in_progress` com apresentação `needs_review` e devolve receipt de responsabilidade encerrada. O CLI exclui o YAML do executor. O executor pode encerrar sua sessão imediatamente.

### 11.5. Fluxo C — reviewer solicita correções

```bash
wosctl profile create reviewer_a \
  --token-stdin --generate-signing-key --enrollment <enrollment-id>

wosctl --profile reviewer_a review checkout --next --count 1
wosctl --profile reviewer_a review show <review-contract-id> --for-agent

# Inspeciona artefatos exatos, executa verificações fora do servidor,
# preenche findings e assessments na seção review do mesmo YAML.
wosctl --profile reviewer_a review validate <review-contract-id>
wosctl --profile reviewer_a review finish <review-contract-id>
```

Com `decision: changes_requested`, o servidor registra decisão assinada, fecha ReviewContract, torna a Task elegível para correção quando não bloqueada e emite os fatos correspondentes. O CLI exclui o contrato de revisão confirmado.

O servidor não envia a API Key do executor ao reviewer. Também não entrega ao reviewer permissão administrativa para contornar o lifecycle.

### 11.6. Fluxo D — corrigir e reapresentar

```bash
wosctl --profile executor_b work checkout <work-item-id> --version <current-version>
wosctl --profile executor_b work show <new-contract-id> --for-agent

# Corrige os findings e registra evidências novas, com novos commits/checksums.
wosctl --profile executor_b work finish <new-contract-id> --completion auto
```

O novo contrato contém referências à entrega anterior, findings pendentes e nova autoridade. A especificação não pode mudar silenciosamente para incorporar exigências fora do escopo. `addressed_findings` vincula cada resposta ao finding original e suas evidências.

Outro executor autorizado pode assumir a correção conforme política de atribuição; não reutiliza a identidade nem a assinatura do primeiro. Nova entrega abre nova rodada de revisão e não herda automaticamente a aprovação de nenhum material anterior.

### 11.7. Fluxo E — aprovar e concluir sem executor ativo

O reviewer adquire a revisão da última submissão, marca `approved` com assessments válidos e retorna. Na transação de aceitação o WOS conclui a WorkItem e o contrato de revisão, sem renovar ou reativar o contrato de execução já entregue.

```bash
wosctl --profile reviewer_a review checkout --next
wosctl --profile reviewer_a review finish <review-contract-id>
```

Critério de aceite decisivo: o processo do executor foi encerrado e o YAML dele já não existe, mas a aprovação e o `DONE` continuam funcionando exclusivamente com o profile do reviewer.

### 11.8. Recuperação e manutenção

```bash
wosctl --profile executor_b work recover --all-pending
wosctl --profile executor_b work refresh <contract-id>
wosctl --profile executor_b work renew <contract-id>
wosctl --profile executor_b work keepalive --foreground --all-active

# Diagnose não revela segredos nem altera autoridade.
wosctl --profile executor_b doctor
```

`recover` reconcilia intenções existentes. Não é alias de `checkout --next`. `refresh` preserva o draft local e sinaliza conflitos. `resume --takeover` mantém a semântica explícita de fencing já existente, aplicando novo grant assinado. Uma tomada de execução nunca fica escondida em `show` ou `validate`.

### 11.9. Múltiplos chats

Três chats/processos podem usar:

```text
Chat A: wosctl --profile planner ...
Chat B: wosctl --profile executor_b ...
Chat C: wosctl --profile reviewer_a ...
```

O mesmo repositório e Outcome são compartilhados; credenciais, contratos e intenções pendentes não são. O host fornece isolamento do código e dos segredos quando necessário. A criação dos chats/subagentes e a escolha do modelo continuam fora do servidor WOS.

<a id="s12"></a>
## 12. Casos de uso, API, MCP e SDK

### 12.1. Evoluir o catálogo único

Preservar HTTP/MCP/SDK sobre os mesmos Application services e catálogo tipado. A rota HTTP `/api/v1` pode permanecer para os comandos aditivos; versão de rota, protocolo de trabalho e schema YAML são conceitos diferentes. Não promover toda a API para v2 apenas porque o contrato de trabalho evoluiu.

| Caso de uso | Comando/consulta-alvo | Estratégia |
|---|---|---|
| Identidade/capabilities | `get_identity`, capabilities de assinatura/política | Estender leitura autenticada. |
| Enrollment | criar challenge, registrar prova de posse | SecurityService; DTOs e autorização próprios. |
| Rotação/revogação | comandos de signing key | Preservar histórico e limites por identidade. |
| Aquisição de trabalho | `acquire_work_contract`, `acquire_next_work_contract` | Evoluir por protocolo do Namespace; emissão assinada. |
| Renovação/retomada | comandos existentes de work contract | Retornar grant assinado atualizado. |
| Retorno final de trabalho | `return_work_contract` | Novo comando atômico com envelope assinado. |
| Aquisição de review | `acquire_review_contract`, `acquire_next_review_contract` | Novo, com independência e alvo imutável. |
| Retorno final de review | `return_review_contract` | Novo comando assinado; approve/changes/inconclusive. |
| Consulta de aceitação | receipt e `contract_acceptance` por request/digest | Nova projeção durável, restrita ao acesso atual. |
| Contexto de correção/revisão | consultas focais com expansão | Evoluir projeções existentes. |
| Administração | revogar contrato específico, cancelar caso, alterar política | Explícito, auditado, sem impersonation. |

O nome definitivo de consultas auxiliares deve acompanhar o gerador atual. Os dois comandos de retorno final constituem a interface operacional principal desta evolução.

### 12.2. Pipeline de retorno de execução

```text
envelope autenticado e verificado
  → UnitOfWork / idempotência / guard
  → autorização atual e policy floor
  → autoridade + versões + spec
  → registrar documentos referenciados por local_key
  → formar submissão imutável e ligar assinatura
  → [direct] assessments permitidos + conclusão + WorkContract completed + Task done
    [review] WorkContract delivered + ReviewCase pending
  → Domain Events + Integration Events + Triggers/outbox
  → receipt assinado + resultado idempotente
  → commit
```

Extrair helpers transacionais das operações existentes. Não chamar métodos públicos que abrem suas próprias transações de dentro de outra transação; não duplicar a máquina de estados em HTTP/CLI. Evidências, submissão e resultado final precisam permanecer atomicamente vinculados.

### 12.3. Pipeline de review

A pipeline de review autoriza pelo ReviewContract, não pelo WorkContract do executor. Obtém a submissão do caso, valida digest, versão dos critérios, evidências e independência atual. Para `approved`, valida blockers/dependências/estado dos pais novamente antes do commit.

Substituir o pressuposto atual de `bindSubmissionAssessment` que exige uma submissão do contrato de execução corrente por uma resolução do alvo de aceitação compatível com o protocolo. Manter o comportamento v1 para históricos/Namespaces não migrados. [S08]

### 12.4. Fechar caminhos de bypass

Inventariar e testar: comandos legados de claim/complete/release; `FinalizeWorkContract`; escrita direta de lifecycle; assessments genéricos; conclusão administrativa; comandos HTTP gerados; MCP; UI; chamadas ao Core embedded.

Em `signed_contracts_v2`, os caminhos não assinados de entrega/decisão devem falhar com `signed_protocol_required`. Não basta desabilitar um botão na UI. Autoavaliação direta deve passar pela política assinada e não ser simulada com `assessment:write` seguido de uma conclusão não assinada.

Administração emergencial permanece possível apenas como intervenção explícita autorizada, com razão e evento próprios, sem fabricar assinatura ou “aprovação normal”. As exigências atuais de revogar autoridade antes de cancelar/alterar material devem ser preservadas e estendidas à revisão.

### 12.5. Paridade para consumidores remotos

HTTP e MCP recebem o mesmo envelope e aplicam o mesmo resultado. O SDK Go fornece tipos de payload, codec e helpers de assinatura/verificação, mas não recebe uma chave privada do servidor nem decide política no cliente.

Consumidores MCP que não possam assinar pelo modelo podem usar seu harness/SDK para realizar a assinatura fora do contexto do LLM. A chave não deve ser passada como argumento de tool visível ao modelo. Em Namespace v2, não criar um atalho unsigned só para acomodar um cliente incapaz; ele usa adapter de assinatura ou permanece no protocolo explicitamente compatível.

### 12.6. Read models focais

Oferecer consultas separadas para: material de execução, caso de revisão, histórico de contratos, histórico de decisões e recibos. A leitura inicial retorna o necessário para a obrigação atual e informa `omitted`/cursores para expansão.

Não carregar todo o Outcome para verificar uma entrega se consultas/indexes focais forem suficientes. Manter coerência de snapshot e `outcome_revision`. Prova de independência deve usar consulta indexada por identidade/Task ou Outcome, não download de timeline inteira para o CLI.

<a id="s13"></a>
## 13. Persistência no servidor e migrações SQL

### 13.1. Extensão aditiva

Usar as implementações Memory, SQLite e PostgreSQL existentes. Não acrescentar Redis, outro banco, event sourcing completo ou um serviço independente de assinatura como requisito.

O snapshot contém migrações até `0019`. Reservar os próximos números em P00 conforme a branch efetiva; exemplos de nomes abaixo são propostas, não instrução para reutilizar número já ocupado.

| Grupo de dados | Conteúdo | Restrições essenciais |
|---|---|---|
| `signing_keys` | Public key, Principal/servidor, Namespace, finalidade, algoritmo, fingerprint, validade e estado. | IDs estáveis; fingerprint único conforme escopo; sem chave privada. |
| `signing_key_enrollments` | Challenge, nonce, Principal, finalidade, validade, consumo. | Uso único e vínculo exato ao enrollment autorizado. |
| `credential_policies` | Capabilities restritivas, revisão mínima, escopos, key IDs permitidos, revisão. | FK para credencial; não amplia grant. |
| `work_acceptance_policies` | Políticas por escopo e suas revisões. | Ordem de precedência definida; alterações auditadas. |
| `contract_signed_issuances` | Snapshot/spec e grants emitidos, envelopes/digests, key IDs. | Append-only; identidade/versão exatas. |
| `signed_returns` | Bytes verificados, proof, autoria, operação, request ID/digest e hora de aceitação. | Append-only; unicidade por escopo/Principal/intenção. |
| `review_cases` | Alvo de revisão, rodada, status, policy snapshot e versões. | Submissão e WorkItem no mesmo escopo; alvo imutável. |
| `review_contracts` | Autoridade do reviewer, lease/fencing, status e versões. | No máximo um contrato efetivamente ativo por caso. |
| `review_decisions` | Decisão, assessments, findings e assinatura vinculada. | Imutável; alvo e autoria persistidos. |
| `contract_acceptances` | Efeitos do retorno, receipt e vínculo de limpeza. | Recuperável mesmo após expiração do cache de idempotência. |
| WorkItem/WorkContract | Marcadores de protocolo, revisão/correção e terminal `delivered`. | Não reescrever material v1 nem números de fencing. |

Preferir tabelas/projeções pequenas a colunas JSON gigantes para toda a autorização. Payloads imutáveis podem ser armazenados como bytes canônicos/JSON com metadados indexados. O material assinado deve preservar os bytes exatos e o vínculo a entidades geradas no commit.

### 13.2. Índices e guards

Índices mínimos: chave por ID/fingerprint/Principal; policy por credencial/escopo; autoridade ativa por Task/caso; ReviewCase pendente por Outcome/prioridade/criação; histórico de execução por Principal e Task/Outcome; aceitação por contrato/request/digest; submissão por ID/WorkItem; decisões por caso/rodada.

A expiração efetiva continua dependendo do relógio autoritativo, não de um job ter alterado status. Índices de unicidade parcial não substituem a transação que expira a autoridade antiga antes de adquirir outra. Preservar serialização do Outcome e controle otimista atuais; medir antes de mudar granularidade.

Fencing de execução e review são monotônicos nos respectivos alvos. Não converter `uint64` para signed int ou número JavaScript. Preservar as representações exatas já adotadas nas implementações SQL.

### 13.3. Dados históricos

Não acrescentar assinatura a uma submissão antiga como se fosse do executor original. Dados v1 ficam marcados como `legacy_unsigned`, com digest e autoria autenticada originais. Uma atestação administrativa posterior pode ser outro registro, mas não substitui a origem histórica.

Preservar fingerprints/receipts antigos e serialização de comandos v1. Fields novos não devem aparecer como `null` em um comando antigo e alterar sua identidade idempotente. Tratar encode v1 e v2 como contratos explícitos, com golden tests.

### 13.4. Backup e restore

Backup/restore deve preservar grants, policies, chaves públicas históricas, envelopes, ReviewCases, contratos, submissões, aceitação e índices. Material secreto de assinatura do servidor é operado por mecanismo separado e precisa de procedimento próprio de restauração.

Após restore sem a chave privada antiga, o servidor pode verificar histórico e configurar nova chave de emissão, mas não fingir que possui o signer antigo. `server_id` e trust set precisam ser reconciliados explicitamente. Testar retomada com contrato ativo, entregue para revisão e correção pendente.

<a id="s14"></a>
## 14. Eventos e triggers

### 14.1. Reaproveitar a infraestrutura existente

O catálogo atual já contém `work_item.completed`, `work_contract.acquired`, `work_contract.result_submitted`, `work_contract.completed`, `work_contract.renewed`, `work_contract.expired` e `work_contract.revoked`, com entrega `at_least_once`. Não recriar um segundo mecanismo de triggers nem trocar o significado desses fatos sem versão. [S14]

### 14.2. Fatos novos propostos

| Fato | Significado |
|---|---|
| `work_contract.delivered` | Retorno de execução aceito para revisão; autoridade do executor encerrada. |
| `work_review.opened` | ReviewCase criado para uma submissão exata. |
| `work_review.acquired` | Reviewer adquiriu autoridade. |
| `work_review.renewed` | Lease da revisão renovado. |
| `work_review.expired` | Autoridade efetivamente expirada registrada por mutação apropriada. |
| `work_review.revoked` | Revogação administrativa da autoridade de revisão. |
| `work_review.changes_requested` | Decisão aceita que exige correção. |
| `work_review.inconclusive` | Decisão aceita sem comprovação suficiente; caso continua pendente. |
| `work_review.approved` | Revisão aprovada sobre material exato. |
| `work_review.cancelled` / `superseded` | Encerramento explícito do caso sem aprovação normal. |

Eventos de chaves/políticas pertencem ao catálogo de auditoria de segurança adequado; não os inserir artificialmente em um Outcome sem relação. Preservar a separação administrativa existente.

### 14.3. Sequências observáveis

Retorno direto:

```text
registros documentais → result_submitted → assessments/conclusion
→ work_contract.completed → work_item.completed → receipt
```

Retorno para revisão:

```text
registros documentais → result_submitted → work_contract.delivered
→ work_review.opened → receipt
```

Aprovação de revisão:

```text
review decision/assessments → work_review.approved
→ work_item.conclusion_recorded → work_item.completed → receipt
```

A ordem final deve ser fixada no catálogo/testes conforme a pipeline existente. Os eventos de uma mutação compartilham a revisão transacional definida pelo Core e possuem índices estáveis. Replay não cria novos fatos.

### 14.4. Semântica de integração

Triggers consomem fatos públicos e persistem sinais no mesmo compromisso transacional que a alteração de estado. Entrega a consumidores continua podendo duplicar; cada consumidor deduplica pelo ID estável. Uma chamada de webhook fora da transação não participa da decisão de aceitar a Task.

Não chamar um sinal genericamente de `work.available` sem reavaliar elegibilidade. A conclusão de uma Task pode não desbloquear nenhuma outra. O consumidor recebe a notificação, consulta a disponibilidade atual e então decide se adquire trabalho.

A assinatura de contratos e a autenticação/assinatura de delivery de integração são funções diferentes. Não compartilhar automaticamente a chave de assinatura do agente com um webhook.

<a id="s15"></a>
## 15. Interface humana, skills e integração com agentes

### 15.1. Interface humana existente

Evoluir o workspace atual, sem substituir a UI por um produto novo. Mostrar:

- Principal/credencial/key ID da execução e status da assinatura;
- autoridade vigente versus assinatura histórica válida;
- modo de aceitação efetivo e motivo de revisão obrigatória;
- submissão exata, evidências, critérios e decisões da rodada;
- histórico de entregas/correções e motivo de revogação;
- diferenças entre Task `done`, execução `delivered` e review pendente.

Nunca exibir “validado” como sinônimo de qualidade comprovada apenas porque a assinatura foi verificada. Usar rótulos distintos: assinatura verificada, autorização aceita, evidências registradas, critérios atendidos e Task concluída.

### 15.2. Mutações humanas em escopo assinado

A UI não pode continuar usando os endpoints unsigned de assessment/finalize para contornar o v2. Na primeira entrega, administração, consultas e provisionamento continuam disponíveis; aprovação assinada deve usar um cliente capaz de assinar, inicialmente o `wosctl`.

Quando a UI ainda não oferecer assinatura sob identidade do reviewer, seus botões de aprovação devem ficar explicitamente indisponíveis naquele protocolo e indicar o fluxo suportado. Não assinar no servidor fingindo ser o humano. Assinatura no navegador ou signer delegado é uma extensão separada, com identidade de custódia explícita e testes próprios, não uma dependência oculta do aceite CLI.

### 15.3. Atualização das cinco skills existentes

| Skill | Alteração |
|---|---|
| `wos-coordination` | Ensinar contrato v2, policy floor, conclusão direta versus review e fonte de verdade. |
| `wos-continuity` | Descobrir profile, recuperar pendências e buscar contexto focal antes de nova aquisição. |
| `wos-delegation` | Delegar com profile e identidade distintos; não compartilhar chaves nem assumir isolamento de subagente. |
| `wos-review` | Checkout de revisão, verificação de submission/digest, findings e retorno assinado. |
| `wos-workspace` | Estrutura mínima, edição permitida, `finish`, exclusão confirmada e proibição de histórico paralelo. |

A instrução operacional central para um agente deve ser curta:

```text
Use somente o profile designado. Recupere pendências antes de adquirir trabalho.
Leia o contrato atual e as referências necessárias. Não edite issued nem _local.
Realize o trabalho, registre evidências vinculadas a versões exatas e retorne pelo CLI.
Não altere status no servidor por atalhos. Em erro, siga recovery sem trocar a intenção.
Não exclua manualmente o contrato. O CLI faz a limpeza após aceitação confirmada.
```

As skills não geram chaves, iniciam modelos ou criam isolamento por mera instalação. Exemplos de hosts devem informar o que foi realmente executado e o que é apenas documentação.

### 15.4. Piloto com Woobe e outros hosts

O piloto Woobe usa os mesmos contratos HTTP/MCP/SDK com identidade de executor/reviewer. O runtime guarda os segredos fora do contexto do agente e realiza a assinatura por adapter. O WOS continua opcional e substituível na arquitetura da Woobe.

Para Codex/Claude ou outro host local, provar três processos independentes antes de afirmar suporte a subagentes nativos com modelos diferentes. Registrar versão do host, forma real de seleção de modelo e identidade usada nas chamadas. O nome “Astra”, “Sol Light” ou equivalente não substitui evidência de qual modelo executou o trabalho.

<a id="s16"></a>
## 16. Migração do protocolo e do workspace existentes

### 16.1. Versões independentes

Separar versão do produto, da API, do protocolo de trabalho, do payload de assinatura e do arquivo local. `0.2.0` é a versão declarada do snapshot; `schema_version: 2` e `signed_contracts_v2` são propostas. A release comercial resultante será escolhida no processo de release existente, não inferida deste documento.

### 16.2. Cutover por Namespace

Preservar `legacy → draining → contracts_v1`. Acrescentar uma evolução explícita:

```text
contracts_v1 → draining_to_signed_v2 → signed_contracts_v2
```

Durante o novo draining, bloquear novas aquisições v1 e definir janela limitada para conclusão/revogação das autoridades restantes. Não ativar o v2 enquanto houver contratos v1 válidos que dependam do fluxo antigo de finalização. Revisões informais pendentes de v1 devem ser resolvidas ou migradas por operação administrativa explícita com histórico, sem fabricar assinaturas do agente.

Antes da ativação:

1. Confirmar schema atualizado e todas as réplicas com suporte ao novo protocolo.
2. Retirar writers antigos, inclusive acessos diretos ao banco, conforme procedimento existente.
3. Provisionar signer do servidor, trust set e políticas/keys das identidades iniciais.
4. Verificar ausência de autoridade incompatível e prontidão de clientes.
5. Ativar com CAS sobre versão do protocolo/política e recibo auditável.
6. Após ativação, endpoints legados mutantes falham explicitamente para aquele escopo.

Não aplicar downgrade automático em incidente. Depois de existirem dados v2, rollback de binário exige versão que compreenda v2 ou manutenção/restauração coordenada. Não permitir que servidor antigo ignore silenciosamente `delivered` ou ReviewCase aberto.

### 16.3. Migração local de schema 1 para schema 2

Criar `wosctl workspace migrate --to 2 --profile <nome> --dry-run`. O dry-run inventaria configuração, contratos, drafts, journals, receipts e conflitos sem alterar nada.

A conversão deve seguir esta ordem:

1. Validar o workspace antigo e o destino; obter lock técnico exclusivo de migração.
2. Reconciliar intenções `prepared`/`sent_unknown`, inclusive aquisições sem arquivo materializado.
3. Consultar o estado remoto de cada contrato e confirmar quais obrigações permanecem.
4. Unificar contrato/checkpoint/result em um documento v2 compatível, mantendo drafts e origem.
5. Materializar `profile.yaml` com binding explícito; exigir enrollment para operações assinadas novas.
6. Marcar contratos v1 ainda presentes como `legacy_unsigned`; não re-assinar retroativamente a spec com identidade falsa.
7. Comparar inventário antes/depois e provar ausência de perda de drafts/intenção.
8. Solicitar confirmação de descarte das cópias antigas após conversão verificada. Backup transitório pode ser exportado para destino explicitamente escolhido, não uma nova pasta permanente de histórico em `.wos/`.

A migração pode ser pausada/reexecutada sem duplicar aquisições. Se o servidor ainda é v1, o cliente mantém compatibilidade de leitura/recuperação, mas não apresenta o contrato como assinado. Fluxo assinado completo exige Namespace v2.

### 16.4. Compatibilidade de comandos

Os comandos v1 existentes continuam documentados por versão. No v2, `work finish` passa a usar o retorno atômico assinado. `work submit`/`finalize` antigos não podem ser atalhos unsigned: o cliente deve recusar seu uso incompatível e indicar `work finish` ou o comando específico v2, preservando replay histórico autorizado.

Manter códigos de saída atuais quando o significado for o mesmo. Introduzir códigos novos somente quando necessários e publicar a matriz; clientes JSON devem usar campos tipados, não interpretar frases. `review pending` v1 e `accepted_for_review` v2 são resultados diferentes: no v2 a entrega foi aceita e a obrigação local encerrou, portanto é sucesso remoto, não erro de execução.

<a id="s17"></a>
## 17. Erros e procedimentos de recuperação

| Código proposto | Condição | Comportamento e recuperação |
|---|---|---|
| `profile_required` | Seletor ausente com vários profiles. | Não iniciar mutação; exigir profile explícito. |
| `profile_binding_mismatch` | Destino/identidade difere do profile. | Não enviar segredo; reconfiguração explícita. |
| `credential_inactive` | API Key expirada/revogada. | Preservar contratos; autenticar por credencial autorizada antes de recuperar. |
| `signing_key_unregistered` | Chave não cadastrada para a identidade. | Orientar enrollment autorizado; não cadastrar automaticamente. |
| `signing_key_inactive` | Chave não aceita operações novas. | Rotacionar/recuperar explicitamente; consultar eventual receipt antes de novo envio. |
| `invalid_signature` | Verificação criptográfica falhou. | Preservar YAML; verificar profile, chave e bytes; não informar segredo esperado. |
| `signature_payload_mismatch` | Draft mudou depois da assinatura. | Bloquear envio; reconciliar intenção pendente antes de preparar outra. |
| `issuer_untrusted` | Emissor fora do trust set. | Não aceitar contrato como válido; atualizar confiança por canal autorizado. |
| `signed_protocol_required` | Operação unsigned em escopo v2. | Usar cliente/adapter compatível; não cair para endpoint antigo. |
| `review_required` | Conclusão direta não autorizada. | Usar retorno `auto`/review com nova intenção deliberada após rejeição confirmada. |
| `reviewer_not_independent` | Principal/grupo participou da execução proibida. | Designar outro reviewer autorizado. |
| `stale_review_target` | Submissão/caso foi substituído. | Não aprovar versão antiga; adquirir revisão atual após reconciliação. |
| `lease_expired` / `stale_execution` | Autoridade perdida. | Não re-assinar para forçar aceitação; nova aquisição/takeover quando permitido. |
| `spec_digest_mismatch` | Spec local alterada ou alvo incompatível. | Recuperar spec do servidor, preservando draft como dado não autorizado. |
| `version_conflict` | CAS não corresponde. | Ler/reconciliar; não aumentar versão automaticamente mantendo intenção antiga. |
| `evidence_unavailable` | Critério requer evidência não acessível/registrada. | Produzir ou registrar evidência adequada; não marcar met por ausência de erro. |
| `criteria_not_satisfied` | Critérios de conclusão não atendidos. | Corrigir material ou obter avaliação permitida; manter arquivo. |
| `review_case_open` | Nova execução ordinária enquanto review pendente. | Consultar/revisar/cancelar caso por operação própria. |
| `active_contract_limit` | Teto de aquisição atingido. | Processar contratos existentes; não criar profiles para contornar quota. |
| `idempotency_conflict` | Mesma chave com intenção diferente. | Reconciliar operação original; nova intenção somente por decisão explícita. |
| `transport_uncertain` | Pode ter ocorrido commit remoto. | Replay/receipt da mesma intenção; manter arquivo. |
| `acceptance_unknown` | Não foi possível comprovar aceitação. | Não excluir; consulta canônica ou reconciliação administrativa. |
| `local_materialization_failed` | Aquisição aceita, gravação local falhou. | `remote_committed: true`; recuperar do pending no profile; não adquirir outra Task. |
| `local_cleanup_failed` | Retorno aceito, exclusão falhou. | `cleanup_pending: true`; repetir só a limpeza verificada. |
| `local_draft_changed` | Arquivo editado durante envio/limpeza. | Preservar ambas as informações no documento quando possível; não apagar edição não confirmada. |
| `unsupported_schema` | Cliente/protocolo incompatível. | Atualizar cliente ou migrar explicitamente. |

Respostas incluem `request_id`, operação, entidade, categoria, possibilidade de retry da **mesma intenção**, estado de commit conhecido e ação de recuperação. Não responder genericamente “assinatura inválida” para erro de permissão, lease, chave revogada ou schema.

Não dizer que toda assinatura inválida é automaticamente recuperável. Um documento corrompido sem cópia confiável pode exigir nova obtenção da especificação; uma identidade revogada pode não ter autorização para continuar.

<a id="s18"></a>
## 18. Mapa de alterações por pacote

Os arquivos existentes abaixo foram identificados no snapshot ou no inventário de árvore. Novos caminhos são sugestões a confirmar em P00; não representam arquivos já implementados.

| Pacote/área | Arquivos existentes relevantes | Alterações |
|---|---|---|
| Domain | `domain/work_contract.go`, `work_submission.go`, `work_item.go`, `work_item_operational_state.go`, `work_protocol.go`, `criterion.go` | `delivered`, binding de aceitação, ReviewCase/ReviewContract/Decision, projeções e guards. |
| Application | `application/work_contract_results.go`, `work_contract_service.go`, `work_contract_next.go`, `work_contract_queries.go`, `work_contract_guards.go` | Retornos atômicos, emissão assinada, resolução de material, readiness/correção e consultas focais. |
| Revisão/política | `application/reviewer_policy.go`, `authorization.go`, `security.go`, `security_admin.go`, `pipeline.go` | Policy floor por credencial, enrollment, revalidação transacional e revisão sem executor. |
| Ports | `ports/authorization.go`, `security.go`, `security_admin.go`, `work_contracts.go`, `work_protocol.go`, `transaction.go` | Port de signing/verification, registry de chaves, policies, reviews e aceitação. |
| Codec público novo | Proposta `packages/wos-core/signing/` | Tipos puros/codec DSSE-JCS e integração Ed25519 reutilizável, sem banco/segredo/HTTP. |
| Storage | `storage/{memory,sqlite,postgres}/work_contracts.go`, `security*.go`, `access.go`, `work_protocol.go`, migrações | Persistência dos registros novos; paridade, índices, CAS, histórico e restore. |
| Comandos públicos | `packages/wos-api/commands/catalog.go`, `contracts.go`, `encode.go`, `receipt.go`, gerados | DTOs v2 e descrição de capabilities; manter serialização v1. |
| HTTP | `http/work_contracts.go`, `security.go`, `criteria.go`, `handler.go` | Novas rotas/comandos, verificação coerente e rejeição de bypass. |
| MCP | `mcp/queries.go`, `resources.go`, `security.go`, gerados | Envelope equivalente, queries focais e mesmas autorizações. |
| Server | `internal/server/`, configuração/runtime | Signer/trust set/server ID, status de readiness e configuração de policies. |
| SDK Go | `packages/wos-sdk-go/work_contracts.go`, `client.go`, `raw.go`, gerados | Tipos de retorno/review, helpers de assinatura e consulta de receipts. |
| CLI config | `packages/wos-cli/documents.go`, `client.go` | Config v2, seleção de profile, secret refs, enrollment e binding. |
| CLI filesystem | `workspace.go`, `journal.go`, `recovery.go`, `syncdir_*` | Store embutido de pending ops, locks estáveis, migração e exclusão segura. |
| CLI trabalho | `work.go`, `work_operations.go`, `planning.go` | Checkout múltiplo, sign/send/finish, review, sync/renew e JSON estável. |
| CLI YAML | `yaml.go`, `schemas/`, `tools/wosclischemas/main.go` | Schema 2 único por contrato; codecs/limites e vetores. |
| UI | `packages/wos-api/web/assets/app.js`, `presentation.js`, `index.html` | Estado de assinatura/review, provisionamento e remoção dos atalhos ilegais. |
| Skills | `packages/wos-skill/skills/*/SKILL.md` | Fluxo mínimo, profiles, assinatura, revisão e recovery. |
| Eventos | `docs/integration-events-v1.json`, mapeamento Application, `tools/eventcatalog/check.py` | Novos fatos aditivos, correspondência e deduplicação. |
| Geração | `tools/openapigen`, `tools/mcpgen`, `tools/postgresgen` | Regenerar a partir da fonte tipada, sem patches manuais nos gerados. |
| Distribuição | `tools/distribution/`, `tools/releases/`, `.github/workflows/`, manifests | Novos schemas/dependências/notices, plataforma nativa e source exato. |
| Documentação | `AGENTS.md`, `ROADMAP.md`, README, ADRs, operações, contratos e arquitetura canônica | Registrar explicitamente mudanças de semântica e estado de entrega. |

### 18.1. Dependências permitidas

O cliente continua API-only: não importa adaptadores de banco nem acessa SQL para concluir contratos. Compartilhar um codec criptográfico puro não altera essa regra. Domain não conhece keyring, filesystem do agente, runtime de IA, servidor HTTP ou segredos.

Application controla a transação e usa ports. O signer do servidor é composto no runtime; o signer local é composto no CLI/harness. Não duplicar criptografia em implementações independentes sem vetores comuns.

### 18.2. Refactors limitados ao necessário

Não aproveitar esta evolução para renomear todos os pacotes, trocar framework HTTP, substituir storage ou alterar o sistema de release. Extrair helpers de sincronização/avaliação é necessário para atomicidade; reescrever o Core inteiro não é.

<a id="s19"></a>
## 19. Ondas de implementação e sequência de PRs

### 19.1. Ordem e dependências

```text
P00 — baseline, ADRs e fixtures
  ├─ P01 — protocolo de assinatura e codec
  └─ P02 — identidades, enrollment e políticas
       └─ P03 — autoridade assinada e domínio de entrega/revisão
            ├─ P04 — retorno direto/entrega para revisão
            └─ P05 — revisão, correções e aprovação
                 └─ P06 — HTTP/MCP/SDK completos

P03 exige P01 e P02; as setas acima não dispensam o codec de assinatura.

P01/P02 + DTOs estáveis → P07 — profiles e workspace v2
P06/P07              → P08 — aquisição múltipla e recuperação
P04/P05/P08          → P09 — sign/send/finish e limpeza
P06/P09              → P10 — cutover e migração
P06/P09              → P11 — UI, skills e documentação operacional
P10/P11              → P12 — aceite integrado, plataformas e medições
P12                  → P13 — release coordenada
```

Toda onda inclui seus próprios testes. P12 consolida o sistema, não adia a validação das etapas anteriores. As dependências permitem trabalho paralelo em branches de escopo restrito, mas integração sempre parte da `master` atual e de contratos de interface acordados.

### P00 — Fixar baseline, decisões e contrato de implementação

**Objetivo:** tornar a evolução retomável por qualquer agente sem depender desta conversa.

**Entregas:** adicionar este plano ao repositório quando a implementação for autorizada; registrar seção nova no ROADMAP sem alterar C01–C12 para “não feito”; criar ADRs novos; inventariar bypass e tabelas; reservar migrações; definir schema e política de compatibilidade; criar fixtures exemplares de direto, review, correção e resposta incerta.

**Ações:** revalidar SHA da master e diferença desde `b75aa3e`; ler `AGENTS.md` e seções canônicas afetadas; congelar os nomes de campos/estados/eventos; separar requisitos obrigatórios de extensões adiadas; documentar o significado de `delivered` e da exclusão local.

**Aceite:** nenhuma decisão operacional central fica implícita; fixtures representam a estrutura mínima; todo requisito R01–R24 tem fase e teste; nenhum comando proposto é anunciado como disponível.

**PR sugerido:** `docs: specify signed contracts v2 and minimal profile workspace`.

### P01 — Codec e provas criptográficas

**Objetivo:** produzir e verificar mensagens compatíveis antes de persistir fluxos complexos.

**Entregas:** tipos de payload; codec JCS/DSSE; Ed25519 por implementação estabelecida; limites e vetores; verificação de finalidade e key ID assinado; formato local equivalente; golden tests para Windows/Linux e consumidores independentes.

**Ações:** preservar `SemanticDigest` v1; definir digest v2 e exact bytes; testar duplicidade/Unicode/números/algoritmo/tamanho; rejeitar material não canônico quando exigido pelo protocolo; provar que o payload executado é o verificado; expor helpers puros, sem secret store ou SQL.

**Aceite:** vetores positivos/negativos passam; alteração de spec/material/purpose é detectada; mera mudança permitida de apresentação YAML não invalida o conteúdo canônico; nenhuma chave privada é serializada.

**PR sugerido:** `feat(signing): add versioned DSSE Ed25519 contract codec`.

### P02 — Identidade, chaves e políticas por credencial

**Objetivo:** autenticar e autorizar cada agente de forma independente.

**Entregas:** registry de chaves, enrollment com prova de posse, rotação/revogação, credential policy e policy floor; Memory/SQLite/PostgreSQL; comandos administrativos e leitura de identidade/capabilities.

**Ações:** incluir credential ID no contexto autenticado; limitar cadastro inicial; preservar grants existentes; impedir novo key registration somente com token roubado; suportar mesma identidade com várias credenciais restritivas; revalidar revogação no snapshot transacional.

**Aceite:** profile não amplia permissões; chave de outro Principal é rejeitada; rotação não libera autorrevisão; políticas legadas não viram full-access v2 automaticamente; backup/restore preserva chaves públicas e policies.

**PR sugerido:** `feat(auth): add signing enrollment and credential acceptance policies`.

### P03 — Autoridade assinada e estados do contrato

**Objetivo:** acrescentar o domínio mínimo de aceitação v2 sem quebrar v1.

**Entregas:** `delivered`, ReviewCase, ReviewContract, Decision e referências de correção; emissão de spec/grant assinados; renovação/takeover com grants novos; guards de Task/review e persistência inicial nos três adapters.

**Ações:** separar assinatura de spec e grant; preservar monotonicidade de fencing; rever `CurrentLease`/`CurrentContractID`; impedir execução ordinária com caso aberto; definir política snapshot e restrições de mudança de especificação durante review.

**Aceite:** aquisição exclusiva continua provada; contratos terminais não reativam; encerramento para review impede nova implementação concorrente; cliente não precisa ainda estar pronto para provar domínio/SQL.

**PR sugerido:** `feat(core): add signed authority and review handoff states`.

### P04 — Retorno final atômico do executor

**Objetivo:** uma entrega assinada gerar exatamente um resultado durável.

**Entregas:** `return_work_contract`; reuso de sync documental em transação; material/submission/signature binding; conclusão direta autorizada ou entrega para revisão; receipt assinado recuperável.

**Ações:** extrair helpers de `work_contract_results.go`; não encadear transações públicas; resolver local keys; assessments explícitos; aplicar policy floor; persistir documentos e decisão de aceitação junto; testar falha após cada etapa da transação.

**Aceite:** direto resulta em Task `done`; reviewed resulta em contrato `delivered` e caso pendente; assinatura inválida não altera domínio; timeout pós-commit não duplica retorno; nenhum critério é considerado atendido por mera presença de evidência.

**PR sugerido:** `feat(core): accept signed work returns atomically`.

### P05 — Revisão independente e correções

**Objetivo:** revisão funcionar sem executor ativo ou contrato local antigo.

**Entregas:** checkout/renew/revoke de ReviewContract; `return_review_contract`; decisão assinada; assessments vinculados; correction acquisition; revalidação dos guards na aprovação.

**Ações:** substituir dependência de holder do executor no caminho novo; rever `bindSubmissionAssessment`; manter piso de independência por Outcome onde configurado; preservar rodadas históricas; tratar inconclusivo, revogação, expiração e alteração administrativa do caso.

**Aceite:** executor encerra sessão após entrega; reviewer pede correção; novo executor entrega; reviewer aprova e chega a `DONE`; identidade que participou da execução é rejeitada quando a política veda; review antigo não aprova material novo.

**PR sugerido:** `feat(review): add signed review contracts and correction rounds`.

### P06 — API, MCP, SDK e consultas focais

**Objetivo:** disponibilizar o protocolo por todos os transports sem divergência.

**Entregas:** catálogo, schemas, OpenAPI, recursos MCP, SDK; receipts pequenos/consultáveis; erros tipados; queries de contrato/caso/correção; teste de bypass de endpoints v1.

**Ações:** gerar artefatos; validar DTOs de autoridade decimal; assinar/validar no caminho público apropriado; adicionar fixture cross-transport; garantir scope binding e nenhuma confiança no `principal_id` declarado; fechar bypass embedded.

**Aceite:** adquirir HTTP, retornar MCP, revisar via SDK e consultar no HTTP produz estado único; fallback unsigned v2 é rejeitado; schema/codegen sem drift; limites/documentos omitidos são explícitos.

**PR sugerido:** `feat(api): expose signed contract and review protocol across transports`.

### P07 — Profiles e formato local mínimo

**Objetivo:** materializar a experiência local definida pelo proprietário.

**Entregas:** `project.yaml` schema 2, `profile.yaml`, diretório `contract/`; seleção por flag/env; backend de segredos; create/inspect/list/remove; enrollment; codec de documento único.

**Ações:** implementar binding de destino; resolver apenas um profile; proteger nomes/caminhos/Windows; manter credenciais fora do YAML; permitir env ref legado; separar dados públicos de campos `_local`; JSON/diagnóstico sem segredo.

**Aceite:** dois profiles na mesma pasta possuem credenciais e contratos independentes; ausência de seletor é erro com múltiplos profiles; nenhum diretório permanente de drafts/journals é criado; criação interrompida recupera enrollment.

**PR sugerido:** `feat(cli): add independent profiles and minimal workspace schema v2`.

### P08 — Aquisição múltipla e journal embutido

**Objetivo:** simplificar os arquivos sem perder recuperação/autoridade.

**Entregas:** `--count`, partial results, pending operations no profile, pending operation no contrato, locks estáveis, recovery e materialização idempotente; renew/keepalive de vários contratos.

**Ações:** refatorar `journal.go` por port de armazenamento; preservar frozen payload e destino; testar crash entre cada gravação/requisição; não reter locks de profile durante execução; quota por Principal; contagem separada de scan limit.

**Aceite:** lote parcialmente aceito é recuperado sem novos contratos; queda após commit e antes de arquivo local materializa o original; keepalive concorrente não perde drafts; processo antigo não toma lock/autoridade nova silenciosamente.

**PR sugerido:** `feat(cli): add multi-checkout and embedded pending-operation recovery`.

### P09 — Assinar, enviar e excluir após confirmação

**Objetivo:** completar o loop do desenvolvedor em um contrato local efêmero.

**Entregas:** `work sign/send/finish`, equivalentes de review, receipt binding, exclusão atômica verificada, recovery de cleanup, diagnósticos.

**Ações:** congelar intenção antes de envio; comparar draft após resposta; distinguir commit remoto e falha local; remover somente o contrato correspondente; preservar dados em erro; encaminhar modo `auto` pela política efetiva; desabilitar semânticas unsigned legadas em v2.

**Aceite:** direto e reviewed excluem arquivo quando a obrigação encerra; review e correção também; assinatura inválida/timeouts mantêm; edição não confirmada nunca é apagada; nenhuma pasta submissions/reviews/receipts passa a existir.

**PR sugerido:** `feat(cli): finish signed contracts with receipt-bound local cleanup`.

### P10 — Migração e ativação do protocolo

**Objetivo:** evoluir instalações reais sem substituir história ou permitir writers incompatíveis.

**Entregas:** `draining_to_signed_v2`, comandos de pré-checagem/ativação, migração do workspace v1, dry-run e recovery da migração; testes com dados v0.2.0 reais.

**Ações:** drenar autoridade antiga; manter leitura/replay histórico; preservar encoder v1; não fabricar assinaturas; bloquear binários incompatíveis; converter local drafts/journals antes de descartar qualquer arquivo.

**Aceite:** upgrade/restore com dados legados mantém identidades/digests/recibos; volta de binário antigo é recusada; migração interrompida recomeça sem perda; v2 não ativa sem pré-condições satisfeitas.

**PR sugerido:** `feat(migration): add explicit signed-protocol and workspace cutover`.

### P11 — UI, skills e guias operacionais

**Objetivo:** fazer agentes e operadores entenderem o protocolo real.

**Entregas:** projeções de review/assinaturas/políticas, onboarding de chaves, rotas UI seguras, cinco skills atualizadas, exemplos CLI/HTTP/MCP, guias de rotação/restore/erro e documentação de limites.

**Ações:** não apresentar assinatura como avaliação de qualidade; eliminar instruções de editar issued ou apagar manualmente YAML; demonstrar profile explícito por chamada; documentar limitações de sandbox e suporte de host; evitar receitas com modelos/configurações não verificadas.

**Aceite:** uma sessão nova segue a skill e conclui o trabalho sem histórico de chat; UI não contorna review; docs e schemas concordam; instalação de skill não é contada como integração executada.

**PR sugerido:** `feat(ux): expose signed work and update agent operating guides`.

### P12 — Aceite integrado, falhas, plataformas e medições

**Objetivo:** provar o produto-alvo sobre o source final.

**Entregas:** matriz completa de testes, jornadas multiprocesso, Linux/Windows nativos, SQL real/restore, fuzz, carga, prova de cleanup, piloto runtime e benchmark de contexto/custo com controles.

**Ações:** executar cenários T01–T96; registrar source, binário, ambiente e skipped; reproduzir falhas de rede/disco; validar paridade HTTP/MCP/SDK/UI; medir contenção por Outcome e revisão; registrar o que não foi possível executar.

**Aceite:** gates essenciais passam no mesmo source; nenhuma porcentagem de completude é derivada só da quantidade de arquivos/commits; integrações não executadas permanecem explicitamente pendentes.

**PR sugerido:** `test: verify signed multi-agent workflows and recovery across platforms`.

### P13 — Release coordenada

**Objetivo:** distribuir a evolução sem confundir build, teste e publicação.

**Entregas:** manifests/version notes, notices/dependências, schemas empacotados, binários/skills/SDK coerentes, recibos nativos e instruções de upgrade.

**Ações:** usar pipeline existente; exigir source/hashes exatos; atualizar claims de suporte apenas após prova; separar publicação npm/GHCR/GitHub e bloqueios de credenciais/conta; não publicar automaticamente apenas por concluir o plano.

**Aceite:** pacote instalado executa init/profile/checkout/finish/review/recover; release contém versão de protocolo compatível; publicação real só é declarada após confirmação dos registries.

**PR sugerido:** `build: coordinate signed WOS client server and skills release`.

### 19.2. Regra de atualização do ROADMAP

Cada PR atualiza a seção nova com: requisitos atendidos, testes executados, source testado, pendências e interfaces alteradas. Uma fase pode estar “implementada, gate de plataforma pendente”; isso não equivale a “aceita para aquela plataforma”. Não apagar estados históricos de C01–C12.

Nenhuma estimativa de dias ou número fixo de commits faz parte do aceite. Os PRs sugeridos são fronteiras de integração; podem ser divididos para revisão sem mudar a ordem operacional principal.

<a id="s20"></a>
## 20. Matriz de testes e rastreabilidade

**Esta seção define 96 cenários de aceitação propostos. Não afirma que esses 96 cenários já foram implementados ou executados.** Cada caso pode exigir testes de Domain, Application, storage, transport e cliente. Reutilizar testes atuais quando equivalentes e acrescentar cobertura para a mudança de semântica.

### 20.1. Identidade e autorização — T01 a T08

| ID | Cenário | Asserção principal |
|---|---|---|
| T01 | Criar profile com token válido e enrollment autorizado. | Principal/Namespace reais são fixados; chave privada não aparece em arquivo/log. |
| T02 | Criar com token inválido, revogado ou expirado. | Nenhum profile operacional completo nem chave registrada indevidamente. |
| T03 | Dois profiles/credenciais do mesmo Principal. | Compartilham limites/identidade de revisão; não viram dois atores independentes. |
| T04 | Chave de assinatura pertencente a outro Principal. | Retorno rejeitado, mesmo com assinatura matematicamente válida. |
| T05 | Alterar permissões/role no YAML. | Nenhuma elevação; servidor ignora campos não autorizados/rejeita schema. |
| T06 | Credencial reviewed tenta `direct`. | Rejeição ou `auto` previamente solicitado para revisão; nunca `DONE` direto. |
| T07 | Deployment exige independência do Outcome. | Policy de Task menos restritiva não enfraquece o deployment. |
| T08 | Revogar credencial/grant entre autenticação e commit. | Revalidação transacional impede nova mutação. |

### 20.2. Enrollment, rotação e emissão — T09 a T16

| ID | Cenário | Asserção principal |
|---|---|---|
| T09 | Reutilizar challenge de enrollment. | Segunda tentativa não registra outra chave; replay permitido só da intenção idêntica confirmada. |
| T10 | Challenge expirado, de outro Principal ou outra finalidade. | Registro rejeitado. |
| T11 | API Key sozinha tenta substituir signing key. | Sem enrollment autorizado/prova anterior, não substitui. |
| T12 | Rotação autorizada com chave anterior ou recovery administrativo. | Nova chave válida; histórico antigo verificável e atribuído à chave original. |
| T13 | Chave retirada/revogada assina operação nova. | Rejeição; consulta histórica distingue validade passada de autoridade atual. |
| T14 | Rotação da chave do servidor. | Specs antigas verificáveis; grants novos assinados pelo novo emissor confiável. |
| T15 | Servidor sobe sem signer configurado no protocolo v2. | Readiness/novas emissões falham explicitamente; não gera chave efêmera silenciosa. |
| T16 | Réplicas com trust set/server ID incompatíveis. | Configuração não passa o gate; não alterna identidades sem aviso. |

### 20.3. Assinatura e representação — T17 a T24

| ID | Cenário | Asserção principal |
|---|---|---|
| T17 | Vetores Ed25519/DSSE e codec independente. | Bytes/verificação compatíveis; comprimentos inválidos retornam erro, não panic. |
| T18 | Alterar instrução, critério ou dependência da spec assinada. | Verificação/digest rejeita a mudança. |
| T19 | Alterar material, checksum ou decisão após assinatura. | Não executa a intenção modificada. |
| T20 | Reusar assinatura de execução em payload de revisão/receipt. | Purpose mismatch rejeitado. |
| T21 | Duplicar chaves YAML/JSON, alias, merge ou campo desconhecido. | Parser/schema rejeita sem interpretação ambígua. |
| T22 | Diferenças de espaços/ordem permitida versus Unicode distinto. | Apresentação equivalente preserva assinatura; conteúdo semanticamente distinto não. |
| T23 | Fencing acima de 2^53 e próximo de uint64 máximo. | Ida e volta exata; overflow rejeitado. |
| T24 | Envelope e comando paralelo não assinado com valores diferentes. | Só payload verificado chega à Application; campo paralelo não influencia. |

### 20.4. Autoridade e exclusividade — T25 a T32

| ID | Cenário | Asserção principal |
|---|---|---|
| T25 | Dois Principals adquirem a mesma Task simultaneamente. | Um vencedor; o outro recebe conflito/ausência elegível, sem duas autoridades. |
| T26 | Expiração no instante exato do retorno. | Relógio do servidor decide; assinatura anterior ao prazo não autoriza commit tardio. |
| T27 | Takeover por mesmo Principal. | Nova execution/fencing; antiga não renova, sincroniza ou finaliza. |
| T28 | Renovação repetida com mesma idempotency key. | Não estende tempo duas vezes; resultado original. |
| T29 | Contrato entregue/completed tenta reativação. | Rejeição; somente nova aquisição conforme estado da Task. |
| T30 | Executor tenta readquirir Task com ReviewCase aberto. | Bloqueio por revisão pendente, mesmo sem lease de execução ativo. |
| T31 | Reviewer expira durante análise. | Outro reviewer pode adquirir o caso; decisão antiga não finaliza. |
| T32 | Instâncias com clock skew e PostgreSQL real. | Tempo autoritativo mantém exclusividade e prazo coerentes. |

### 20.5. Retorno e conclusão direta — T33 a T40

| ID | Cenário | Asserção principal |
|---|---|---|
| T33 | Retorno direto autorizado com critérios atendidos. | Docs/submission/assessments/Task/contrato/events/receipt no mesmo commit. |
| T34 | Falha injetada em cada estágio do retorno. | Rollback integral; nada parcialmente aceito. |
| T35 | Critério obrigatório sem avaliação válida. | Não conclui, apesar de assinatura/credencial válidas. |
| T36 | Evidência retirada ou versão de artefato diferente. | Rejeita material inconsistente e preserva histórico. |
| T37 | Registro com local_key duplicada, referência desconhecida ou escopo diferente. | Não gera entidades cross-scope nem mapeamento ambíguo. |
| T38 | Policy endurecida após aquisição. | Retorno respeita regra atual mais restritiva; não faz downgrade silencioso. |
| T39 | Policy relaxada/credencial trocada após aquisição reviewed. | Contrato mantém piso original até intervenção explícita registrada. |
| T40 | Retorno para review aceito. | WorkContract `delivered`, caso pendente e obrigação local encerrada atomicamente. |

### 20.6. Revisão e correção — T41 a T48

| ID | Cenário | Asserção principal |
|---|---|---|
| T41 | Reviewer conclui após executor/processo/arquivo desaparecer. | Aprovação usa exclusivamente autoridade do reviewer. |
| T42 | Executor troca profile/key e tenta revisar. | Histórico do Principal impede bypass. |
| T43 | Aprovação de submissão antiga após correção nova. | Não aceita; digest/round alvo permanece exato. |
| T44 | Changes requested com findings endereçáveis. | Caso e decisão preservados; correção recebe novo contrato/fencing. |
| T45 | Novo executor autorizado assume correção. | Autoria independente e ligação à submissão/findings anteriores. |
| T46 | Aprovação com blocker/dependência incompatível inserida concorrentemente. | Nenhum `DONE` parcial; contrato de revisão preservado para reconciliação. |
| T47 | Mudar material da spec durante review sem comando administrativo. | Operação bloqueada; replanejamento explícito supersede quando autorizado. |
| T48 | Review inconclusiva, cancelada ou revogada. | Sem aprovação falsa; disponibilidade do caso/Task segue a política definida. |

### 20.7. Workspace, profiles e edição — T49 a T56

| ID | Cenário | Asserção principal |
|---|---|---|
| T49 | Dois chats operam profiles diferentes no mesmo projeto. | Sem troca de credenciais, contrato ou pending operations. |
| T50 | Múltiplos profiles sem flag/env. | Mutação falha; não escolhe o último/primeiro profile. |
| T51 | Mudar servidor/Namespace/profile binding com intenção pendente. | Não redireciona token ou retorno a outro destino. |
| T52 | Path traversal, symlink/reparse, arquivo especial ou nome reservado. | Confina leitura/gravação/exclusão aos alvos válidos. |
| T53 | Colisão por caixa e `.yaml`/`.yml` do mesmo contrato. | Rejeição explícita de ambiguidade em plataformas relevantes. |
| T54 | Editar draft enquanto CLI renova/atualiza metadados. | Compare-and-swap preserva a edição ou reporta conflito. |
| T55 | Arquivo maior que limite ou envelope base64 fora do teto. | Erro determinístico antes de consumo descontrolado; sem truncar. |
| T56 | Agente consulta `show --for-agent`/diagnósticos. | Sem secret/proof técnica desnecessária; campos omitidos declarados. |

### 20.8. Falhas, receipts e exclusão — T57 a T64

| ID | Cenário | Asserção principal |
|---|---|---|
| T57 | Crash após aquisição no servidor e antes do YAML local. | Recovery usa pending do profile e materializa o contrato original. |
| T58 | Crash após materialização e antes de limpar pending do profile. | Não sobrescreve draft nem adquire outra Task. |
| T59 | Resposta de retorno perdida após commit. | Replay/receipt confirma o original; não duplica evidência/conclusão/trigger. |
| T60 | Disco cheio/permissão negada ao guardar receipt ou remover arquivo. | Commit remoto comunicado; cleanup recuperável; nada apagado por aproximação. |
| T61 | Rascunho alterado entre envio e resposta. | Não exclui conteúdo não coberto pelo receipt. |
| T62 | Receipt errado/de outro contrato/servidor/Principal/digest. | Não autoriza exclusão. |
| T63 | Cache de idempotência expirado, contrato com aceitação histórica. | Recupera pelo registro durável; ausência de cache não vira “falhou”. |
| T64 | Crash após delete ou temporário/lock abandonado. | Reconciliação segura; sem duplicação e sem roubar lock de processo vivo. |

### 20.9. Lotes e concorrência multiprocesso — T65 a T72

| ID | Cenário | Asserção principal |
|---|---|---|
| T65 | `--count 3`, apenas duas Tasks elegíveis. | Retorna duas aquisições e explica incompletude/ausência sem fabricar terceira. |
| T66 | Falha no segundo item de lote de cinco. | Contratos anteriores preservados; recovery não reinicia lote com novas intenções. |
| T67 | Busca paginada incompleta sem Task na primeira página. | Não declara ausência global; preserva cursor/search_complete. |
| T68 | Repetir busca vazia idempotente após surgir Task. | Replay vazio original; nova busca exige nova intenção. |
| T69 | Quota por Principal e profiles/credenciais adicionais. | Não contorna limite de contratos ativos. |
| T70 | Dois processos manipulam o mesmo contrato com atomic rename. | Lock estável e CAS evitam duas conclusões/limpezas concorrentes. |
| T71 | Keepalive de lote com um contrato expirado/revogado. | Resultado por contrato; outros não são perdidos/cancelados. |
| T72 | Ordem de locks com aquisição, sync, recovery e cleanup simultâneos. | Sem deadlock/lock global que serialize toda a execução do projeto. |

### 20.10. Transportes e ausência de bypass — T73 a T80

| ID | Cenário | Asserção principal |
|---|---|---|
| T73 | Aquisição HTTP, retorno MCP, revisão SDK. | Mesmo domínio e resultado; sem estado oculto de sessão MCP. |
| T74 | Endpoints antigos complete/finalize/assessment em Namespace v2. | Rejeitam caminho incompatível; nenhum bypass por payload equivalente. |
| T75 | Chamada embedded direta e comandos administrativos. | Políticas/assinaturas aplicadas; intervenção administrativa não vira review normal. |
| T76 | UI tenta mutação unsigned onde assinatura é exigida. | Não conclui; UX mostra fluxo realmente suportado. |
| T77 | Redirect de mutação para outro host. | Não envia token/assinatura a destino novo. |
| T78 | Requisição com ActorRef ou Principal declarados para outro agente. | Autoria/autoridade derivadas da identidade autenticada e delegação permitida. |
| T79 | Resposta grande/receipt omitido parcialmente. | Resultado compacto mantém prova de commit e caminho explícito de consulta. |
| T80 | Regenerar OpenAPI/MCP/SQL/schemas/SDK. | Sem drift; golden v1 e v2 preservam tipos e fingerprints. |

### 20.11. Migração e histórico — T81 a T88

| ID | Cenário | Asserção principal |
|---|---|---|
| T81 | Upgrade de banco com dataset v0.2.0. | WorkItems, contratos, critérios, signatures ausentes e receipts históricos corretos. |
| T82 | Cutover com contrato v1 ainda válido. | Ativação recusada até drenagem adequada. |
| T83 | Cliente/writer antigo após ativação v2. | Mutação incompatível rejeitada; não ignora ReviewCase/status novo. |
| T84 | Workspace v1 com drafts e sent_unknown. | Migração preserva conteúdo e reconcilia intenção antes de descartar originais. |
| T85 | Migração interrompida em cada etapa. | Reexecução segura, sem nova aquisição ou assinatura histórica fabricada. |
| T86 | Restore SQLite e PostgreSQL com execução/review/correção pendentes. | Continuidade e fencing corretos nos três cenários. |
| T87 | Histórico após rotação/revogação e perda do signer privado antigo. | Verificação com chaves públicas preservada; nova emissão exige signer configurado. |
| T88 | Replays antigos após encoder/DTO v2 introduzido. | Mesma intenção v1 mantém fingerprint e resultado originais. |

### 20.12. Produto integrado e distribuição — T89 a T96

| ID | Cenário | Asserção principal |
|---|---|---|
| T89 | Três processos: planner, executor e reviewer, com correção. | Jornada completa sem compartilhar histórico de chat/credenciais. |
| T90 | Evento `work_item.completed` e webhook duplicado. | Só após commit de `DONE`; consumidor deduplica por identidade estável. |
| T91 | Queda de servidor/worker antes e depois de outbox commit. | Sinal não desaparece nem exige execução exatamente uma vez de ponta a ponta. |
| T92 | Linux e Windows nativos: profiles, keys, filesystem e cleanup. | Binário real testado; cross-compilação isolada não conta. |
| T93 | Pacote instalado offline com schemas/notices/skills. | Fluxo autenticado de profile a finish/recover funciona fora da árvore de source. |
| T94 | Carga em um e vários Outcomes, históricos densos e reviews. | Medir latência/conflitos/throughput sem enfraquecer guards para “passar”. |
| T95 | Benchmark de contexto e custo com comparadores controlados. | Custo por entrega aceita inclui retries/review; resultado sem promessa fabricada. |
| T96 | Nova sessão usa só profile, contrato e referências do servidor. | Consegue continuar; ao finalizar, apenas profile permanece, com `contract/` vazio. |

### 20.13. Rastreabilidade dos requisitos

| Requisitos | Ondas principais | Testes de referência |
|---|---|---|
| R01–R02 | P01, P02, P07 | T01–T16, T92 |
| R03–R04 | P07, P08 | T49–T56, T69–T72 |
| R05 | P03, P08 | T25, T30, T65–T69 |
| R06 | P01, P03, P06 | T14–T24, T73 |
| R07 | P04, P07, P09 | T33–T37, T54–T56 |
| R08 | P01, P06, P09 | T17–T24, T59–T64 |
| R09 | P02, P04 | T05–T08, T33–T39 |
| R10 | P03, P04 | T06–T07, T30, T38–T40 |
| R11 | P02, P05, P09 | T31, T41–T48 |
| R12 | P05, P09 | T43–T47, T89 |
| R13 | P05, P06 | T35, T41–T48 |
| R14–R15 | P08, P09 | T57–T64, T70 |
| R16 | P02–P05, P10 | T12–T14, T63, T81–T88 |
| R17 | P04–P06, P12 | T59, T90–T91 |
| R18–R19 | P06, P10, P11 | T73–T80, T83 |
| R20 | P06, P11, P12 | T56, T79, T94–T96 |
| R21 | P10 | T81–T88 |
| R22 | P03, P08, P12 | T26–T32, T57–T64, T86, T96 |
| R23 | P06, P09, P11 | T02, T04, T13, T19, T57–T64, T79 |
| R24 | P12, P13 | T89–T96 |

### 20.14. Níveis de teste

Domain testa estados/invariantes sem rede. Application testa autorização, assinatura/port, transações e falhas com relógio/IDs controlados. Storage replica os contratos em Memory, SQLite e PostgreSQL reais. Transportes usam clientes independentes. CLI roda processos distintos e injeta falhas de disco/rede. UI/skills exercitam o comportamento anunciado, não apenas renderização/instalação.

Fuzz deve cobrir parsers/envelopes, conversão YAML/JCS, path/binding, IDs/versões/fencing, mapeamentos de evidência e cursores. Testes de race são necessários, mas não substituem concorrência entre processos ou conexões SQL distintas.

<a id="s21"></a>
## 21. Medição de contexto, custo e desempenho

### 21.1. Separar benefício do WOS de benefício do modelo

Comparar três estratégias sobre o mesmo conjunto de trabalho e mesmos critérios de aceite:

| Comparador | Execução |
|---|---|
| A | Um agente/harness com coordenação convencional do projeto. |
| B | Planejador e executores de capacidades/custos diferentes, sem WOS, usando coordenação externa equivalente na medida possível. |
| C | Mesmos modelos/harnesses de B, usando WOS para contratos, continuidade, revisão e contexto focal. |

A versus C mede a solução inteira. B versus C ajuda a separar o ganho de coordenação do ganho de simplesmente escolher modelos mais baratos. Não atribuir ao WOS toda diferença de preço entre modelos.

O piloto pode começar com 30 Tasks distribuídas em alguns Outcomes e evoluir para amostras maiores. Esse tamanho é uma proposta de experimento, não garantia de significância estatística. Relatar variação por tipo de tarefa, falhas e repetições em vez de uma porcentagem universal.

### 21.2. Métricas mínimas

Medir tokens de entrada/saída/cache quando o provedor os expuser, custo efetivamente faturado ou estimado com tarifa documentada, tempo até aceitação, número de reexecuções, revisões, chamadas de ferramenta, intervenções humanas e retrabalho de integração.

Para o WOS, medir payload inicial de contrato, expansões de contexto, bytes retornados, latência de acquire/sync/return/review, conflitos/retries, custo de verificação/assinatura, queries por operação e crescimento do histórico.

```text
custo_por_task_aceita =
  (planejamento + execução + revisão + correções + coordenação mensurável)
  / quantidade de Tasks aceitas

custo_por_outcome_aceito =
  custo total da jornada, inclusive Tasks malsucedidas e integração
  / quantidade de Outcomes efetivamente aceitos
```

Se o host usa assinatura fixa e não expõe custo/token confiável, registrar utilização observável e tempo; não converter arbitrariamente uma mensalidade em tarifa por milhão de tokens. Medições reportadas pelo próprio agente são autodeclaradas até confrontadas com telemetria/faturas/CI.

### 21.3. Critério de qualidade equivalente

Usar os mesmos testes de integração, requisitos, evidências e avaliação final. Uma Task `DONE` internamente não basta para provar qualidade equivalente se cada estratégia usa critérios diferentes. Incluir integração do código final para detectar entregas isoladas que passam em seus testes, mas não funcionam juntas.

Apresentar custo total e taxa de aceitação, não somente economia de contexto. Reportar tanto sucesso de retomada quanto quantidade de material que precisou ser recuperado do servidor.

### 21.4. Desempenho da infraestrutura

Reutilizar a metodologia de carga registrada pelo projeto, acrescentando mistura de execução/review, múltiplos Outcomes e históricos densos. O relatório atual aponta contenção relevante no mesmo Outcome e não é SLA de produção. [S03][S13]

Definir budgets de regressão depois de medir a baseline no mesmo ambiente. Medir p50/p95/p99, throughput, conflitos, CPU e memória. Não impor um número absoluto inventado como “10 ms por contrato” sem benchmark.

Otimizar consultas focais/índices antes de remover guards transacionais. O custo criptográfico deve ser medido junto de storage e rede; uma assinatura rápida não torna uma transação SQL complexa automaticamente barata.

<a id="s22"></a>
## 22. Critérios finais de aceite e publicação

### 22.1. Demonstração obrigatória do fluxo direto

Em Namespace v2, criar profile via CLI, validar credencial e enrollment, adquirir mais de um contrato, executar uma Task, registrar evidências, assinar e retornar. O servidor valida, conclui a Task, preserva histórico/receipt e emite o evento. O CLI remove somente o contrato concluído; os outros permanecem.

Repetir com retorno aceito cuja resposta foi perdida. O recovery deve provar a mesma aceitação e fazer a limpeza, sem nova submissão ou eventos duplicados.

### 22.2. Demonstração obrigatória com revisão

Criar executor reviewed e reviewer independentes. O executor adquire, executa, retorna e seu YAML é removido após handoff confirmado. Encerrar o processo do executor. O reviewer obtém seu próprio contrato, solicita correção e retorna; seu YAML também desaparece.

Uma nova sessão autorizada adquire a correção, responde aos findings e retorna. Outro contrato de review avalia a submissão exata e aprova. A Task passa para `DONE` sem reutilizar chave ou lease do executor. Todas as rodadas permanecem no servidor.

### 22.3. Lista de aceite operacional

- [ ] Estrutura local estabilizada em `project.yaml`, `profiles/<nome>/profile.yaml` e `contract/`.
- [ ] Não há histórico/journal permanente em diretórios adicionais do projeto.
- [ ] Credenciais e signing keys distintas, cadastradas e revogáveis sem impersonation.
- [ ] Assinaturas de emissão, retorno, review e receipt interoperáveis e testadas.
- [ ] Autorização transacional e policy floor impedem conclusão indevida.
- [ ] Retorno para review encerra a obrigação do executor sem concluir a Task.
- [ ] Reviewer pode aprovar/corrigir sem executor ativo.
- [ ] Resultado/evidências/assessments permanecem vinculados a versões exatas.
- [ ] Cleanup depende de receipt e não apaga edição não confirmada.
- [ ] Batch, expiry, takeover, recovery e limites por Principal funcionam.
- [ ] HTTP/MCP/SDK/embedded respeitam o mesmo protocolo; UI não abre bypass.
- [ ] Triggers de conclusão usam fatos duráveis e consumidores idempotentes.
- [ ] Migração preserva contratos/drafts/receipts legados sem fabricar assinaturas.
- [ ] Linux e Windows anunciados foram executados nativamente no source final.
- [ ] Skills e documentação correspondem ao CLI distribuído.
- [ ] Claims de economia, desempenho e suporte são acompanhados de evidência real.

### 22.4. Aceite de código não é publicação

Manter separados: implementação, validação local, CI hospedado, plataforma nativa, piloto com runtime real, pacote instalado e publicação externa. O relatório existente já diferencia esses estágios; a nova evolução deve manter essa disciplina. [S03][S17]

O source do relatório final precisa ser o mesmo dos binários e manifests. Uma execução de testes em parent/head anterior não certifica automaticamente o commit final. Mudança apenas documental pode ser explicada como tal, mas deve ser identificada.

### 22.5. Definição de conclusão do projeto-alvo

O objetivo deste plano estará atingido quando um novo desenvolvedor conseguir instalar o cliente, criar seus profiles, executar os dois fluxos e recuperar falhas seguindo a documentação, sem arquivos operacionais extras e sem depender de conhecimento desta conversa.

O resultado central não é a quantidade de entidades novas. É a seguinte propriedade:

> **Cada agente recebe uma obrigação autenticada, trabalha com a própria identidade, retorna uma entrega verificável e pode desaparecer após a confirmação. O servidor mantém a continuidade, a revisão e a verdade sobre o trabalho.**

<a id="s23"></a>
## 23. Instrução de execução para agentes de desenvolvimento

O texto abaixo pode iniciar a implementação, depois que o proprietário autorizar alterações no repositório:

```text
Implemente o plano WOS de contratos assinados, profiles independentes e workspace mínimo.

Antes de alterar qualquer arquivo:
1. Leia AGENTS.md, ROADMAP.md, os ADRs afetados e este plano.
2. Atualize a visão da master e compare-a com o baseline
   b75aa3e0ca6d649cafadf8450d9b5d606367dda4.
3. Identifique o que já foi entregue por outros PRs; não reconstrua C01–C12.
4. Crie sua branch a partir da master atual e escolha a próxima onda desbloqueada.

Preserve estas decisões:
- WOS é servidor de estado/coordenação, não runtime nem scheduler de agentes.
- wosctl é cliente API-only; wos permanece servidor/administração.
- Cada profile tem profile.yaml e contract/; não criar árvores permanentes de
  checkpoints, submissions, reviews, receipts, outbox, cache ou histórico.
- Recuperação lógica continua existindo, embutida no profile/contrato.
- Assinatura não substitui autorização, critérios, lease, fencing ou revisão.
- Entrega aceita para review encerra a obrigação do executor; review independe dele.
- Contratos terminais não reativam; correções usam nova autoridade.
- Exclusão local só após aceitação durável do material exato.
- Nenhum bypass unsigned é permitido em Namespace migrado para o protocolo v2.

Implemente testes junto do comportamento. Preserve Memory/SQLite/PostgreSQL,
HTTP/MCP/SDK, idempotência e o histórico v1. Use os geradores para artefatos gerados.
Não altere migrações já aplicadas ou fingerprints históricos.

Ao encerrar cada PR:
- Atualize ROADMAP, ADRs, schemas e documentação afetados.
- Liste requisitos Rxx atendidos e testes Txx efetivamente executados.
- Informe source, plataforma, comandos, resultados e pendências.
- Não anuncie integração Codex/Claude/Woobe por mera instalação de skill.
- Não anuncie Windows por cross-compilação nem publicação por build local.
- Não marque uma onda concluída por número de commits ou por código existir.

Mantenha o escopo da onda. Não acrescente billing, scheduler, modelo de IA,
integração proprietária ou domínio de produto à implementação deste protocolo.
```

<a id="s24"></a>
## 24. Fontes e limites da análise

### 24.1. Fontes do repositório, fixadas no snapshot

As referências abaixo apontam para arquivos no SHA analisado, não para uma `master` que possa mudar depois. Descrições de comportamentos existentes se apoiam nesses arquivos; as demais decisões são propostas deste plano.

| Ref. | Fonte | Uso nesta análise |
|---|---|---|
| [S00] | Commit `b75aa3e` | Identificação exata da master examinada. |
| [S01] | `README.md` | Boundary do produto, distribuição e referência ao ciclo C01–C12. |
| [S02] | `ROADMAP.md` | Estado declarado das ondas, sem confundir tarefas históricas com lacunas atuais. |
| [S03] | `docs/verification-work-contracts-2026-10-08.md` | Evidência de testes registrada e gates externos ainda separados. |
| [S04] | `packages/wos-cli/README.md` | Comandos, workspace, journal, limites YAML, exit codes e fluxo atual de finish. |
| [S05] | `packages/wos-core/domain/work_contract.go` | Estados, lease, spec, digest, fencing e versões. |
| [S06] | `packages/wos-core/application/reviewer_policy.go` | Independência atual por deployment e histórico de Outcome. |
| [S07] | `packages/wos-core/application/work_contract_results.go` — autoridade/sync | Verificação do holder, sync documental e atomização atual. |
| [S08] | Mesmo arquivo — submit/finalize/assessment binding | Dependência do executor e vínculo de avaliação à submissão. |
| [S09] | `packages/wos-core/ports/authorization.go` | Permissões atuais e ports de autorização transacional. |
| [S10] | `packages/wos-core/application/security.go` | Credencial opaca, Principal, grants, sessões e revogação. |
| [S11] | `packages/wos-cli/workspace.go` | `os.Root`, atomic write, limites, locks e verificações de caminho. |
| [S12] | `packages/wos-cli/journal.go` | Frozen intent, destination binding, outbox/receipt e recovery existentes. |
| [S13] | `docs/work-contract-operations.md` | Operação, draining, review/lease, migração e consultas. |
| [S14] | `docs/integration-events-v1.json` | Eventos públicos efetivos e semântica de entrega. |
| [S15] | `docs/adr/0018-work-contract-authority.md` | Autoridade durável, terminais atuais, finalização e bypass inventory. |
| [S16] | `docs/adr/0019-wosctl-yaml-workspace.md` | Cliente API-only e decisões locais que serão evoluídas. |
| [S17] | `AGENTS.md` | Hierarquia de decisões, boundaries, testes e atualização obrigatória do roadmap. |
| [S18] | `packages/wos-cli/documents.go` | Config/formatos locais reais; root `.wos/work` e schema 1. |
| [S19] | `VERSION` | Versão declarada `0.2.0`, sem inferir publicação externa. |

### 24.2. Referências técnicas primárias

| Ref. | Documento | Uso |
|---|---|---|
| [N01] | RFC 8032 — EdDSA | Referência técnica de Ed25519 e vetores. |
| [N02] | Go `crypto/ed25519` | Implementação padrão e requisitos da API. |
| [N03] | DSSE Protocol, versão 1.0.2 | Protocolo de assinatura com tipo de payload e bytes exatos. |
| [N04] | DSSE Envelope, versão 1.0.2 | Envelope convencional e distinção de outras representações. |
| [N05] | RFC 8785 — JCS | Serialização canônica, sem confundir com autenticação. |

A especificação DSSE e o uso de JCS são escolhas explícitas deste plano, não tecnologias já comprovadas como presentes integralmente no WOS. JCS já aparece no digest do Core; assinatura de identidade é o complemento proposto. As referências técnicas são documentos primários, não certificação da futura implementação.

### 24.3. O que foi e não foi verificado

Foi verificado o snapshot da branch por consulta ao GitHub e lidos arquivos centrais de código/documentação. Não foram realizadas alterações no repositório, execução da suíte, publicação, provisionamento de credenciais ou instalação de profiles durante a elaboração deste arquivo.

A confiança é alta quanto às diferenças identificadas nos caminhos examinados: workspace schema 1, journal separado, digest sem assinatura nesses modelos e finalização dependente de autoridade do executor. A completude de toda a superfície de bypass e dos impactos de schema exige o inventário de P00 e os testes de P06/P10/P12.

Se a master avançar depois do SHA registrado, o agente de implementação deve reconciliar o delta antes de começar, preservando as decisões operacionais do proprietário e aproveitando implementações equivalentes já entregues.

---

**Síntese final:** manter o núcleo já construído e completar um protocolo de delegação, entrega e revisão assinadas, com autoridade própria por participante e estado local mínimo. O CLI deve esconder a complexidade de segurança/recuperação, não removê-la. O servidor deve preservar a responsabilidade e a continuidade do trabalho sem assumir a execução dos agentes.

[S00]: https://github.com/A1b3rt0M3rcad0/wos/commit/b75aa3e0ca6d649cafadf8450d9b5d606367dda4
[S01]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/README.md
[S02]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/ROADMAP.md
[S03]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/docs/verification-work-contracts-2026-10-08.md
[S04]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/packages/wos-cli/README.md
[S05]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/packages/wos-core/domain/work_contract.go
[S06]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/packages/wos-core/application/reviewer_policy.go
[S07]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/packages/wos-core/application/work_contract_results.go
[S08]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/packages/wos-core/application/work_contract_results.go#L285
[S09]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/packages/wos-core/ports/authorization.go
[S10]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/packages/wos-core/application/security.go
[S11]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/packages/wos-cli/workspace.go
[S12]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/packages/wos-cli/journal.go
[S13]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/docs/work-contract-operations.md
[S14]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/docs/integration-events-v1.json
[S15]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/docs/adr/0018-work-contract-authority.md
[S16]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/docs/adr/0019-wosctl-yaml-workspace.md
[S17]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/AGENTS.md
[S18]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/packages/wos-cli/documents.go
[S19]: https://github.com/A1b3rt0M3rcad0/wos/blob/b75aa3e0ca6d649cafadf8450d9b5d606367dda4/VERSION
[N01]: https://www.rfc-editor.org/info/rfc8032/
[N02]: https://pkg.go.dev/crypto/ed25519
[N03]: https://github.com/secure-systems-lab/dsse/blob/master/protocol.md
[N04]: https://github.com/secure-systems-lab/dsse/blob/master/envelope.md
[N05]: https://www.rfc-editor.org/rfc/rfc8785.html
