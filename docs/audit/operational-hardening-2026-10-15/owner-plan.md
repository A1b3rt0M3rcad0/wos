# WOS — Plano de correção e endurecimento operacional (PR único)

**Status:** plano de implementação; nenhuma alteração de código realizada por este documento.  
**Repositório:** `A1b3rt0M3rcad0/wos`  
**Base auditada:** `master`, commit `b572269085494ddd4d6370716ff1cb360b46065f`, de 13/10/2026.  
**Release existente:** `v0.4.2`, publicada em 13/10/2026.  
**Escopo:** recuperação de contratos assinados, HTTP, entrega de integrações, consultas, concorrência, perfis de implantação, continuidade operacional e certificação.  
**Entrega obrigatória:** **um único Pull Request**, em uma única branch, com **múltiplos commits pequenos e sequenciais**. **Não fazer merge sem autorização explícita do proprietário.**

> **Regra de evidência:** a severidade abaixo descreve impacto potencial no cenário indicado. Não pressupõe incidente em produção nem corrupção de dados reproduzida. O plano diferencia defeito funcional, limite arquitetural deliberado e lacuna de operação/certificação. Antes de alterar comportamento, criar uma reprodução que falhe contra a base auditada, quando aplicável.

---

## 1. Objetivo e definição de conclusão

Entregar um WOS operacionalmente mais previsível, com recuperação segura de intenções assinadas, proteções HTTP adequadas, detecção de integrações degradadas, leituras de continuidade com custo controlado, concorrência ajustável por política e procedimentos de recuperação reproduzíveis; sem transformar o WOS em um scheduler, executor de agentes, gerenciador de infraestrutura externa ou produto diferente.

A implementação somente estará **concluída** quando:

1. Todos os achados P1 e P2 deste plano tiverem correção executada ou limitação explicitamente demonstrada e aceita pelo proprietário.
2. O comportamento novo estiver implementado e testado **em Memory, SQLite e PostgreSQL real** quando atravessar invariantes do Core/persistência; transportes HTTP/MCP/Go SDK e CLI nativo forem qualificados no escopo pertinente.
3. Provas de idempotência, CAS, fencing, segregação de papéis, scopes, revogação, histórico imutável e rollback continuarem válidas.
4. O comportamento de contratos assinados v2, contratos legados v1, schemas de workspace 1/2, idempotência persistida e assinaturas históricas continuar compatível, exceto alterações aditivas documentadas.
5. Os workflows obrigatórios passarem no SHA exato do PR, inclusive race, SQL real, browser, CLI nativo Linux/Windows, testes de falha, migração/restore, geração e distribuição.
6. As medições antes/depois e os limites de suporte forem arquivados em `docs/audit/` ou `docs/stable/`, com commit, ambiente, comandos executados e resultados brutos reproduzíveis.
7. Todo o código, testes, migrações, documentação, schemas gerados e eventuais ajustes de publicação estiverem **no mesmo PR**; **nenhum segundo PR de correção** será aberto.
8. O PR permanecer aberto/draft após a conclusão técnica, aguardando autorização expressa para merge.

### Decisões arquiteturais obrigatórias

- **WOS não executa agentes nem agenda trabalho.** Hosts executam agentes; WOS persiste estado, autoridade, evidências, planejamento e coordenação.
- **Não enfraquecer segurança para ganhar throughput.** CAS, `fencing_token`, validade temporal, identidade de credencial, assinatura Ed25519 e review independente são invariantes.
- **Sem reinterpretação de resultado ambíguo:** timeout/EOF/conexão perdida nunca significa “não houve commit”. O fluxo padrão continua buscando recibo/repetindo a **mesma** intenção.
- **Uma intenção aceita não pode ser descartada.** Uma intenção encerrada/cancelada no servidor nunca pode vir a ser aceita por uma chamada tardia.
- **Estado autoritativo no banco.** Cache, CLI, métricas, memória do Agent e /readyz não substituem o estado persistido.
- **Compatibilidade aditiva.** Não reescrever migrações antigas, envelopes assinados já emitidos ou recibos anteriores; migrations novas, forward-only.
- **PostgreSQL para instâncias compartilhadas de maior carga.** Não tentar introduzir múltiplos escritores SQLite: a serialização de escrita é característica do backend.
- **Backup, rotatividade de chaves, coleta de métricas e notificações pertencem à operação externa.** Fornecer comandos, contratos e diagnósticos; não criar um scheduler externo embutido no Core.

---

## 2. Inventário dos achados e impacto

| ID | Tipo | Prioridade | Situação observada | Quando compromete a operação |
|---|---|---|---|---|
| F01 | Lacuna funcional de recuperação | **P1** | `return` congelado preserva a intenção após rejeição remota; falta encerramento terminal comprovável | Agente autônomo precisa corrigir e repetir sem intervenção; intenção permanece pendente |
| F02 | Hardening HTTP | **P1** | `http.Server` explicita apenas `ReadHeaderTimeout`; proteção de conexão/admissão incompleta | Instância acessível sem proteção de borda, clientes lentos/concorrência excessiva |
| F03 | Observabilidade/entrega | **P1** | Worker opcional e `readyz` não certifica entrega; backlog/exhaustion exige monitoramento externo | Negócio depende dos webhooks para prosseguir trabalho |
| F04 | Custo de consulta | **P1/P2** | `GetContinuitySection` monta todo `OutcomeState` e todas as seções antes de paginar | Outcomes e históricos densos; leituras frequentes por muitos agentes |
| F05 | Limite operacional deliberado | **P2** | Teto fixo de 3 trabalhos e 1 revisão por Principal nas aquisições signed v2 | Muitos workers compartilham Principal e precisam de paralelismo autorizado |
| F06 | Limite de storage/deploy | **P2** | SQLite padrão do Compose utiliza uma conexão escritora; não equivale a perfil multiwriter | Uso compartilhado e altamente concorrente sem PostgreSQL |
| F07 | Dependência de operação | **P2** | Backup, restore, issuer, workspaces locais e outbox dependem de procedimentos coordenados | Falha de infraestrutura, mudança de instância, restauração parcial |
| F08 | Certificação incompleta | **P3** | Benchmark/cobertura limitados a envelopes documentados; 1.0 não certificado | Decisões de SLA ou promessa de escala sem medição válida |

**Importante:** F05 e F06 não são bugs. F03 tampouco significa que `/readyz` deveria retornar 503 quando um webhook de terceiro falha. Cada correção respeita essas distinções.

---

## 3. F01 — Recuperar intenções signed v2 definitivamente rejeitadas

### 3.1. Problema, localização e causa

**Locais principais:**

- `packages/wos-cli/return_sign_v2.go` — `prepareReturnV2`, congelamento do draft e proibição de segunda assinatura quando já existe `Local.Pending`.
- `packages/wos-cli/return_send_v2.go` — `recoverReturnV2` reenvia envelope inalterado após busca de recibo; não resolve rejeição terminal semanticamente corrigível.
- `packages/wos-core/application/signed_return.go` — `ReturnSignedWork`.
- `packages/wos-core/application/signed_review_return.go` — `ReturnSignedReview`.
- `packages/wos-core/application/pipeline.go` — idempotência transacional e `SignedOperationRepository`.
- `packages/wos-core/storage/{memory,sqlite,postgres}/signed_operations.go` e `ports/signed_operations.go` — resultados persistentes de operações assinadas.
- `packages/wos-core/domain/signed_operation.go` — representação de resultado aceito, atualmente **não** um estado terminal de rejeição.

Uma assinatura imutável evita mutações de conteúdo depois do envio — é uma garantia correta. O problema aparece quando o servidor rejeita definitivamente esse envelope sem aceitar trabalho e o cliente precisa criar outra intenção legítima. Simplesmente apagar o journal local não é seguro: um envio antigo ainda pode estar em trânsito. Uma busca que responde “not found” também **não prova** que nunca existirá commit.

### 3.2. Solução escolhida: resolução terminal autoritativa no servidor

**Não implementar `force --resign` nem `delete --pending` local.** Introduzir uma operação **nova e aditiva**, com nome a confirmar na catalogação pública, proposta `ResolveSignedReturnIntent`, mais um registro durável e imutável de resolução terminal.

Fluxo proposto:

1. O cliente preserva o envelope original, o `request_id`, a credencial original, o digest, o `Idempotency-Key`, a versão de autoridade e a identificação de Scope.
2. Diante de erro remoto, o CLI executa a recuperação existente. Se houver recibo aceito, fecha normalmente; se a resposta for incerta ou infraestrutura estiver indisponível, **não abandona**.
3. Quando o operador/host confirmar que deseja encerrar uma intenção sem aceitação, o CLI prepara **nova intenção de resolução**, assinada com chave atualmente autorizada, referenciando o envelope original, seu digest e o motivo explícito. Uma política de host pode automatizar essa solicitação **somente para erro classificado como definitivo**; o servidor continua sendo o árbitro final.
4. O servidor valida criptograficamente o envelope original, o emissor/destino, Namespace, Outcome, Principal, credencial originária, contrato, tipo de retorno, request ID e digest. Revalida o acesso atual; credencial expirada/revogada entra no fluxo administrativo de recuperação, **nunca** em bypass.
5. Sob a **mesma ordem de locks** usada na aceitação signed (`WorkProtocol`/Namespace e guard do Outcome; a ordem exata será confirmada e estabilizada em teste), consulta a aceitação durável existente e o estado atual do contrato. Se a operação já foi aceita, retorna o resultado/recibo original, não cria resolução negativa.
6. Se ainda não foi aceita, registra **atomica e duravelmente** uma decisão terminal `abandoned`/`closed_without_acceptance`, ligada ao **exato** envelope original, com razão, autor, timestamp transacional e dados necessários para impedir replay futuro.
7. Toda entrada de `ReturnSignedWork`/`ReturnSignedReview`, inclusive replay após TTL da cache, consulta a resolução terminal correspondente **na mesma fronteira transacional**; se cancelada, rejeita com código específico (`signed_intent_closed`) e nunca escreve acceptance, assessment, Task ou Outcome.
8. O servidor produz **recibo assinado de resolução** com Scope, identidade, comando original, digest, chave idempotente, decisão terminal e versão/prova necessárias. O recibo não afirma que alguma execução externa foi desfeita.
9. O CLI verifica criptograficamente o recibo com o issuer confiável/pinado, persiste e sincroniza em disco **antes** de alterar o estado local. Arquiva envelope/draft/consequência sem perda de bytes; libera o contrato local somente se comprovado seguro. Mantém histórico auditável da intenção descartada.
10. Uma nova intenção pode ser assinada **apenas se a autoridade do contrato ainda for válida**; caso contrário, deve ocorrer nova aquisição/correção pelas regras atuais. Nunca renovar CAS/fencing silenciosamente.

**Modelo proposto (novos tipos/arquivos, não existentes na base):**

- `domain.SignedReturnIntentResolution` e `ports.SignedReturnResolutionRepository`, com `GetByOriginalIntent`, `InsertTerminal` e consulta por idempotency key/digest; não misturar `abandoned` com `SignedOperationResult` que representa **sucesso confirmado**.
- Migrações **0027** (SQLite/PostgreSQL): `signed_return_intent_resolutions`, unicidade da identidade original `(namespace_id, principal_id, original_credential_id, original_command_name, original_idempotency_key)` e integridade referencial; índice de lookup. Conferir que 0027 é a próxima numeração antes da execução.
- Persistir recibo/decision assinado em armazenamento documental/assinatura existente ou tabela referenciada com invariantes de `issuer_history`; não colocar segredos nem chave privada na tabela.
- Comando/DTO de resolução e `SignedStateQuery` aditivo `return_resolution`, com leitura sempre autenticada e mesma credencial original (ou procedimento de recuperação administrative explicitamente autorizado).
- Estados locais aditivos `resolution_prepared`, `resolution_unknown`, `abandoned_confirmed`, `cleanup_pending`; journal separado para a nova intenção de resolução. Não reescrever o envelope antigo.

### 3.3. Contrato transacional e matriz de corrida obrigatória

| Interleaving | Resultado exigido |
|---|---|
| `ReturnSignedWork` comita antes de `Resolve...` | Resolução não declara abandono; devolve acceptance verificável |
| `Resolve...` comita antes de `ReturnSignedWork` | Retorno tardio rejeitado sem tocar Task, contrato, histórico ou revision |
| Duas resoluções simultâneas com **mesma** intenção de resolução | Um commit + replay do recibo idêntico |
| Duas resoluções concorrentes com mesmo original e motivos/chaves diferentes | Uma única decisão terminal; a outra recebe conflito/estado terminal sem criar segunda decisão |
| Resolução ocorre após perda de resposta HTTP | Reconsulta recupera exatamente o resultado persistido |
| Processo morre depois do commit e antes da limpeza do YAML | `recover` confirma recibo; não perde draft nem assina outra intenção |
| Resolução é solicitada por outro Principal/CID/Scope/issuer | 403; não revela se houve aceitação de outra identidade |
| Credencial original revogada, signer alterado, lease vencido | Não relaxar regra; exigir fluxo atual de recuperação/autoridade aplicável |
| Abandono confirmado, depois chega POST antigo pela HTTP/MCP/SDK | Rejeição determinística da intenção original; nenhuma alteração de estado |

**Pré-validação adicional:** aproximar as verificações técnicas do CLI das regras *estáveis* do servidor (`SignedFinding`, `criterion_assessments`, campos condicionais, evidência) para reduzir rejeições evitáveis. A pré-validação é uma melhoria de UX/DX, **não substitui** resolução transacional, porque políticas e estado podem mudar.

### 3.4. Testes de aceite de F01

- Teste que reproduz a intenção congelada após HTTP 422 na base inicial; prova que não é possível corrigi-la pelo mesmo fluxo sem intervenção.
- Testes de domínio/portas nas três implementações (Memory/SQLite/PostgreSQL real) cobrindo todos os interleavings acima.
- Testes HTTP/MCP/SDK de equivalência de erro, versão, permissões, idempotência e lookup do recibo.
- CLI nativo Linux/Windows: freeze → rejeição definitiva → resolve → recibo → edição autorizada → novo retorno; crash/ENOSPC em cada janela de publicação/local cleanup.
- Restore após resolução e repetição do POST antigo; antigo continua rejeitado.
- Fuzz de digest, payload DSSE e DTO estrito. Nunca interpretar erro 401/403/409, timeout ou 5xx como permissão implícita para abandono.
- Verificar regressão: a recuperação de sucesso já aceito, revogação, review independente, correção e histórico legado continuam idênticos.

**Aceite:** nenhuma execução aceita é descartada, nenhuma intenção oficialmente descartada pode ser aceita depois, e rejeições definitivas deixam de prender o workflow quando o host ainda possui autoridade para reexecutar.

---

## 4. F02 — Endurecer servidor HTTP, MCP e admissão de requisições

### 4.1. Problema e localização

- `packages/wos-api/internal/server/runtime.go`: `http.Server` atualmente usa `Addr`, `Handler` e `ReadHeaderTimeout: 5s`, sem os demais deadlines explícitos.
- `packages/wos-api/internal/server/config.go` e `env.go`: configuração restrita de HTTP; `HTTP.RequestTimeout` default 15s.
- `packages/wos-api/http/handler.go`: `context.WithTimeout` e `MaxBytesReader` para JSON.
- `packages/wos-api/internal/server/session.go`: corpo de login e regras de origem/cookie.
- `packages/wos-api/mcp/*`: caminho MCP stateless/Streamable HTTP que precisa de testes específicos de deadline.

Context deadline em handler não limita todas as fases de uma conexão (ex.: cliente enviando corpo lentamente). O problema é de endurecimento e dimensionamento, **não prova de exploração atualmente observada**.

### 4.2. Implementação

1. Adicionar campos configuráveis em `ServerConfig`/`HTTPConfig` (nomes finais consistentes com CLI/env): `read_header_timeout`, `read_timeout`, `write_timeout`, `idle_timeout`, `max_header_bytes`, `max_inflight_requests`.
2. Definir defaults explícitos **de referência**, sujeitos aos testes do MCP: `5s/30s/45s/60s`, `64 KiB`, `128` requisições em voo. Não tratá-los como SLA nem pressupor que servem para todos os clientes.
3. Validar desigualdades entre `request_timeout`, `write_timeout`, shutdown e tempo de resposta; rejeitar valores negativos, overflow e opções que invalidem o protocolo MCP. Permitir overrides de implantação documentados.
4. Aplicar proteção de conexões em `http.Server` e middleware de **admissão global não bloqueante**, com política explícita de saturação (`503 Service Unavailable` + `Retry-After`) sem iniciar mutação. Rate limiting por identidade/cliente externo, se necessário, pertence preferencialmente ao ingress; não confiar cegamente em `X-Forwarded-For`.
5. Definir limite de corpo e comportamento de cancelamento uniformes para REST, MCP, sessão, rotas auxiliares; confirmar que limites não truncam retorno legítimo dentro do contrato de 256 KiB.
6. Diferenciar deadline de requisição de deadline de escrita do socket. Para eventual stream/SSE suportado, usar estratégia dedicada, testada, em vez de aplicar `WriteTimeout` genérico e interromper resposta viva.
7. Garantir resposta clara e consistente em sobrecarga, conexão abortada e timeout, sem repetir automaticamente writes/idempotency keys.
8. Adicionar métricas agregadas de rejeição por saturação, requests em voo, timeouts e cancelamentos, sem labels de token/IP arbitrário.
9. Documentar defesa de borda obrigatória para deployment público: TLS, rate limiting, limites de conexões, headers, timeouts e dimensionamento; separar proteção local da proteção de ingress.

### 4.3. Testes de aceite

- Slowloris/headers lentos; corpo lentamente enviado; `Content-Length` excedente; header excessivo; conexão idle; requisição cancelada pelo cliente.
- Saturação com N+1 clientes; liberar slot após panic/cancelamento/deadline; `/livez` e `/readyz` continuam acessíveis em sobrecarga conforme reserva operacional definida.
- REST autenticado e não autenticado, MCP real (JSON e eventual streaming), browser session/cookie/origem.
- Mutação cuja resposta expira mas já comitou: recuperar o mesmo recibo sem duplicar efeito.
- Shutdown com requisições em voo não regressa; nenhuma goroutine/conexão permanece presa após encerramento forçado.

**Aceite:** limites efetivos nas camadas pertinentes, sem quebra das operações normais, com teste de tráfego hostil e compatibilidade HTTP/MCP.

---

## 5. F03 — Saúde real da entrega de integrações e supervisão do worker

### 5.1. Problema e localização

- `packages/wos-api/integration/worker.go`: `Run` processa até 25 ticks por ciclo, sequencialmente, e usa `http.Client.Timeout=10s`.
- `packages/wos-core/storage/{sqlite,postgres}/integration.go`: estados `pending/leased/succeeded/exhausted`, `ClaimDelivery` e `FinishDelivery` com fencing e seis tentativas.
- `packages/wos-api/internal/server/runtime.go`: worker opcional, execução em goroutine, `/readyz` verifica storage/issuer, não o progresso da entrega.
- `packages/wos-api/observability/observer.go`: métricas cumulativas, sem visão autoritativa do backlog/idade do item mais antigo.

`/readyz = 200` significa que **a API pode servir requisições**, não que receptores de webhooks estejam funcionando. Tornar a API unready porque um terceiro caiu pode agravar o incidente.

### 5.2. Implementação

1. Criar port de **diagnóstico read-only** da fila, com agregação autoritativa e escopada no banco (sem carregar payloads), por status e idade: `pending`, `leased`, `exhausted`, `oldest_due_age`, `oldest_lease_age`, `next_due`, tentativas totais e destino configurado **sem URL/secret em resposta**.
2. Adicionar visão operacional autenticada, proposta `GET /api/v1/operations/delivery-health` ou endpoint compatível com o catálogo existente; exigir permissão administrativa de Namespace e aplicar filtros de Scope sem vazamento entre tenants.
3. Acrescentar contadores estáveis, `worker_started`, `last_tick`, `last_successful_delivery`, `last_claim`, `worker_running` e razão da última falha classificada. Não usar estado em memória para afirmar que fila está vazia.
4. Supervisão explícita: monitorar término inesperado do worker e reportar `degraded`/alerta. Distinguir cancelamento intencional de falha; não engolir erro fatal. Manter shutdown ordenado e bounded.
5. Definir estados operacionais `healthy`, `degraded`, `disabled`, `unknown` em **observabilidade separada**. `/livez` e `/readyz` não devem falhar só porque o receptor externo está indisponível. A ausência de worker quando a instância depende de entrega deve aparecer em diagnóstico/alerta.
6. Configurar política de backlog/idade e alertas no sistema de monitoramento externo. Referência inicial: evento due sem entrega por 5 minutos ou qualquer `exhausted`; personalizar em produção.
7. Melhorar throughput de forma mínima e segura: opção `delivery_worker_concurrency` com default **1** e teto configurável (referência inicial: 4 para PostgreSQL), cada worker com `ClaimDelivery` e fencing existentes. Não prometer ganho antes dos testes; SQLite deve manter default 1 e avisar sobre paralelismo.
8. Medir concorrência, starvation e recuperação entre réplicas. Preservar `at-least-once`, dedupe por `integration_event_id` no consumidor e tentativas estáveis. Nunca converter webhook em execução de agente.
9. Validar/otimizar índices de `delivery_due` e consultas agregadas com `EXPLAIN`, somente com evidência de scan/custo.

### 5.3. Testes de aceite

- Worker desligado com backlog: API ready, diagnóstico `disabled/degraded` conforme modo; nenhuma alegação enganosa de entrega.
- Worker ativo com destino indisponível: pending/leased/exhausted corretamente reportados, alertas sem dados sensíveis.
- Worker morre/é cancelado e reinicia: leases vencem e são retomadas por outro worker; `fencing_token` impede ACK obsoleto.
- Dois servidores PostgreSQL / múltiplos workers: não há duas confirmações autoritativas da mesma tentativa; duplicatas externas continuam possíveis e documentadas.
- Backpressure e 100–1.000 mensagens sintéticas em ambiente isolado, com medição de backlog, throughput e latência.
- Sem regressão na assinatura HMAC, validação SSRF/DNS/IP, redirects, retries, contagem de tentativas e `redelivery` explícito.

**Aceite:** falha de integração não é silenciosa; operadores conseguem diferenciá-la de falha da API; não há perda da disciplina de entrega nem dependência de um serviço terceiro para declarar o Core ready.

---

## 6. F04 — Eliminar hidratação integral desnecessária em continuity/graph

### 6.1. Problema, causa e localização

- `packages/wos-core/application/continuity.go`:
  - `GetContinuity` chama `GetOutcomeState` antes de limitar a resposta.
  - `GetContinuitySection` também chama `GetOutcomeState`, projeta todas as seções e só depois pagina a escolhida.
  - `continuitySections` serializa e ordena todos os objetos; `sectionPage` localiza cursor por digest do item.
- `packages/wos-core/application/query.go`: `GetOutcomeState` carrega WorkItems, Objectives, relações, Issues, Blockers, documentos etc.
- `packages/wos-core/application/focal_graph.go`: `GetOutcomeGraph` inicia por `GetOutcomeState`; `GetWorkContext` ainda consulta listas amplas de relações/links/decisões.
- `packages/wos-core/storage/{memory,sqlite,postgres}/` e `ports/`: faltam operações especializadas para várias projeções de baixo custo.

**Dados verificados:** em fixture de 1.000 WorkItems ricos, PostgreSQL teve rich continuity `p95 ≈120,33 ms`, `≈12,73 MB` alocados/chamada, com 3 leitores produzindo `≈24,36 snapshots/s` em uma execução local sem profiling. Resultados profiled têm números distintos e são mantidos separadamente; não misturar fontes. Não existe capacidade certificada de 10.000 itens.

### 6.2. Solução

1. Criar `ports.ContinuityReadRepository` (nome proposto) com leituras transacionais: metadados/revisão, páginas por seção, contagens agregadas e projeções mínimas. Implementar nos três adapters.
2. Construir caminho otimizado para seções diretas (`work_items`, `objectives`, `issues`, `blockers`, `artifacts`, `evidence`, `decisions`, `relations`, `external_references`) com keyset e `LIMIT N+1` **no banco**, sem desserialização de todas as entidades.
3. Para seções derivadas (`ready_work`, `blocked_work`, `in_progress_work`, `attention_needed_work`, dependências, proof/progress, blockers): definir consultas especializadas/joins e cálculo focal do grafo com todas as dependências **necessárias**, sem assumir que status materializado no momento da escrita permanece válido após expiração de lease.
4. Preservar autorização no momento da leitura e consistência de `OutcomeRevision`/`EvaluatedAt` dentro de uma única transação. Paginar sob a mesma semântica de snapshot; alterações de revisão devem provocar `snapshot_changed`.
5. Preservar os campos públicos (`schema_version`, `sections`, `counts`, `omitted`, `section_cursors`, `progress`, `evaluated_at`, `truncated`) e a classificação operacional atual; não perder itens nem modificar significado de `ready`, `blocked`, `scheduled`.
6. Compatibilidade de cursor: **não invalidar silenciosamente cursores v0.4.2**. Se novo keyset exigir `cursor_version=2`, aceitar legados por período de migração com fallback verificável; não confundir ordenação SQL por ID com ordenação antiga por JSON serializado. Provar equivalência ou documentar evolução aditiva clara.
7. Empregar índices compostos apropriados `(namespace_id,outcome_id,sort_key,id)` e índices de relações/links por origem/destino. Adicionar migração forward-only SQLite/PostgreSQL somente onde necessário; não alterar migrations antigas.
8. Adaptar `GetContinuity` completo para páginas limitadas por seção + agregações SQL. O `MaxSnapshotBytes` deve limitar **a resposta** sem demandar reconstrução crescente de todos os dados. Determinar bytes/cortes com comportamento determinístico.
9. Otimizar `GetOutcomeGraph` com travessia de adjacências por lotes, respeitando profundidade e limites existentes. Para `GetWorkContext`, substituir varreduras de histórico não relacionado por consultas indexadas escopadas ao WorkItem/Objective/refs; preservar exatamente a prova pertinente.
10. Manter `GetOutcomeState` para casos que realmente exigem visão integral, mas não permitir que os endpoints paginados o utilizem como atalho sem custo controlado e registrado.
11. Explicitar estratégia de versões/cursors sem misturar `SQLite`, `PostgreSQL` e `Memory` dentro de `application`; só o adapter conhece SQL e índice.

### 6.3. Critérios de aceitação mensuráveis

- Fixture **100/1.000/10.000** itens, incluindo critérios e avaliações reais, em SQLite e PostgreSQL real; Memory nos testes de contrato/paridade.
- Respostas e cursores iguais/semanticamente equivalentes aos da base auditada em fixture fixa; páginas consecutivas sem repetições, omissões, mistura de versões ou vazamento entre Namespaces.
- `ready/blocked` e lease calculados corretamente no instante de avaliação, inclusive expiração **sem** escrita de Task.
- Para paginação simples em 10.000 itens, heap/alloc de aplicação cresce em função do tamanho da página e das relações **efetivamente necessárias**, e não proporcional ao total de WorkItems. Demonstrar com perfis/`benchmem`; não aceitar apenas resposta menor.
- Meta de engenharia para as **seções simples**: redução de pelo menos **30% no p95 e 50% nos bytes alocados** versus baseline medido na mesma máquina e dataset de 10.000; se não alcançada, registrar causa e exigir aceite explícito. Para seções derivadas, definir alvo próprio depois do baseline, sem mascarar varreduras inevitáveis.
- Sem piora acima de **10% no p95** nas consultas focais de 1.000 itens, no mesmo ambiente estatístico; falhas por flutuação devem gerar reexecução controlada, não edição arbitrária da meta.
- Retenção dos limites 256 KiB, páginas de até 100 itens, erros precisos e testes de restore/cursor antigo.

**Aceite:** paginar não implica construir todo o Outcome; relatórios de custo demonstram ganho na escala alvo, sem alteração indevida de semântica.

---

## 7. F05 — Tornar cotas por Principal configuráveis sem ampliar autoridade

### 7.1. Problema e localização

- `packages/wos-core/application/signed_acquisition.go`: `counts.PrincipalWork >= 3`.
- `packages/wos-core/application/signed_review_acquisition.go`: `counts.PrincipalReview >= 1`.
- `packages/wos-core/domain/signing_identity.go`: `CredentialPolicy` e `WorkAcceptancePolicy` já contêm limites por credencial e por Namespace; não há campo explícito para o teto por Principal.
- `packages/wos-core/application/signing_identity.go`: administração CAS de políticas.
- `storage/{memory,sqlite,postgres}/signing_identity.go`: políticas persistidas como `state_json`.

O teto atual é uma proteção intencional. O ajuste pretendido é **parametrização administrativa**, não remoção de limite. Múltiplos agentes com Principal compartilhado continuam a representar a mesma autoridade para controle de quota e segregação de review.

### 7.2. Solução

1. Adicionar à política **de Namespace** campos opcionais como `max_active_work_per_principal` e `max_active_review_per_principal` (nomes finais a validar no catálogo). **Defaults para registros antigos: 3/1.**
2. Limitar a alteração dos novos campos à `WorkAcceptancePolicy` com `OutcomeID=nil` e `WorkItemID=nil`; regras por WorkItem não podem aumentar um teto global de Principal.
3. Aplicar os limites na mesma transação de `AcquireSignedWorkContract`/`AcquireSignedReviewContract`, incluindo acquires `--next` e reaquisições. Cotas de credencial e Namespace continuam independentes e efetivas.
4. Validar intervalo explícito e máximo seguro (proposta 1..100), versão CAS, Principal/Namespace correto e ações administrativas autorizadas; nunca permitir `0 = ilimitado`.
5. Expor cota efetiva e motivo de rejeição em consultas administrativas/`doctor`, sem listar contratos de outro Principal sem autorização.
6. Testar duas credenciais do **mesmo** Principal, múltiplos Outcomes, Principal distintos, alteração de política com contratos ativos, revogação e 50 aquisições concorrentes.
7. Rejeição não deve consumir lease, quota, idempotency result de sucesso nem incrementar revisão de Outcome.

**Aceite:** defaults antigos reproduzem a base, aumento configurado funciona apenas com autorização administrativa, e nenhum ceiling de credencial/Namespace/fencing/review é contornado.

---

## 8. F06 — Perfis operacionais e uso correto de SQLite/PostgreSQL

### 8.1. Problema e localização

- `compose.yaml` e `.env.example`: perfil amigável local em SQLite; `WOS_MCP_ENABLED=true` no Compose.
- `packages/wos-core/storage/sqlite/adapter.go`: `SetMaxOpenConns(1)` deliberado; não há escrita paralela.
- `packages/wos-core/storage/postgres/adapter.go`: `Serializable`, pool default 16, lock timeout 5s e statement timeout 30s.
- `packages/wos-api/internal/server/config.go`, `docs/operations.md` e `docs/stable/operations-reference.md`.

O problema não é que SQLite esteja incorreto, e sim tratá-lo como se tivesse capacidade de multiwriter. Não proibir SQLite para workloads pequenos e single-instance só para “parecer production”.

### 8.2. Solução

1. Preservar `compose.yaml` e `.env.example` como quick-start local com SQLite, sem alterar seu significado.
2. Criar um **perfil/referência de deployment compartilhado** com PostgreSQL, TLS no ingress, credenciais específicas de instância, volumes persistentes, health checks, worker opcional, limites de recursos e gestão externa de segredos.
3. Adicionar a `wos config validate` (ou a um subcomando `doctor`, se o contrato público exigir) opção aditiva `--deployment=shared`/`--production`: diagnóstico read-only de storage, TLS de borda declarado, issuer persistente, autenticação, webhook worker, backup/configuração de endpoints e limites de concorrência. **Advertências, não falso bloqueio** de perfis legítimos.
4. Não adicionar novo adapter/ORM, cluster de SQLite nem fila externa. Identificar com medição a faixa em que PostgreSQL é recomendado.
5. Documentar cotas, pressupostos para múltiplas réplicas, clocks/transaction-time, migração e supervisão de banco com capacidade e pool.
6. Em CI, assegurar que imagens/Go toolchain usados no Docker e release não divergem acidentalmente; corrigir drift se confirmado, usando a fonte/versionamento do pipeline.

### 8.3. Aceite

- Compose local antigo sobe inalterado; dados persistem após restart.
- Perfil compartilhado sobe com PostgreSQL real e modo `api_token`; issuer/credencial e permissões validados; acesso externo sem TLS não é recomendado como seguro.
- Teste de contenção: 1/4 Outcomes, 2/8/50 consumidores, colisão de versionamento e retries exatos; registrar p95, conflitos, throughput, pool wait, CPU e memória com ambiente.
- Testar que uma instância SQLite nunca afirma disponibilidade de múltiplos escritores e que advisories do `doctor` são específicos e acionáveis.

**Aceite:** configurações promovem o backend correto por workload, sem misturar “funciona localmente” com certificação de serviço compartilhado.

---

## 9. F07 — Continuidade operacional, backup, restore e issuer

### 9.1. Problema e localização

- `docs/stable/operations-reference.md` já recomenda cópia consistente por hora, 24 horários/7 diários/4 semanais, restore mensal e separação de chave privada.
- `docs/operations.md` documenta `wos db backup`/`restore` (SQLite), `pg_dump`/`pg_restore` (PostgreSQL), migrações forward-only e reenvio possível de outbox.
- `packages/wos-api/internal/server/issuer_recovery.go`, `packages/wos-api/internal/server/runtime.go`, `packages/wos-cli/recovery.go` e `packages/wos-cli/workspace_v2.go`.

Testes de restore existem, mas não é correto tratar o procedimento documentado como tarefa automaticamente executada na instalação. **RPO/RTO só são reais se medidos no ambiente.**

### 9.2. Solução pragmática

1. Entregar `tools/operations/` com scripts **opt-in** de backup/restore/checklist e exemplos cron/systemd/Kubernetes CronJob para execução externa. Não introduzir scheduler nem serviço remoto no Core.
2. Capturar manifest/checksum e metadados do backup (`source_commit`, `schema_version`, `server_id`, watermark de estado e hora); armazenar em local protegido. Nunca incluir seed privada ou tokens em artefato público.
3. Criar comando/script de **verificação read-only** de backup/restore contra **base nova**, validando schema, namespaces, grants, revogações, issuer/trust, leases/fences, signed results, novas resoluções de F01, outbox e versões.
4. Suportar restauração com writers interrompidos, sem sobrescrever base em operação. Reconciliação após backup não apaga efeitos externos de webhooks/Tools; declarar necessidade de dedupe.
5. Incluir guia de recuperação de issuer e histórico de chaves públicas, com checagem de pin aprovado e impedimento de geração silenciosa de outra identidade.
6. Incluir guia para proteção e recuperação do workspace CLI separado do backup do banco (drafts, journal, pending intentions, receipts) e verificação no restart.
7. Emitir métricas/procedimentos para idade do último backup, último restore validado, oldest outbox pending, quota do banco e taxa de crescimento de histórico; coleta/alarmes externos.
8. Testar restore de v0.4.2 para schema aditivo desta entrega, além de rejeição de binário antigo contra schema novo. Não modificar SQL antigo para “consertar” restore.

### 9.3. Aceite

- Backup consistente e restore **em novo destino** de SQLite/PostgreSQL reais com checksum verificado.
- Recovery de contrato de execução, review, correção e resolução terminal pendentes, com **a mesma autoridade/versões/fencing** após restore.
- Testes de token revogado e issuer/pinned history; seed privada ausente gera falha explícita, não identidade substituta silenciosa.
- Reprodução do caso “webhook entregue externamente antes do snapshot e pendente no snapshot”; deduplicação demonstrada no consumidor de teste.
- Medir RPO/RTO do fixture e registrar como medição daquele ambiente, não como SLA global.

---

## 10. F08 — Qualificar escala, compatibilidade e versão

### 10.1. O que está comprovado e o que falta

- `v0.4.2` está publicada, com artefatos nativos e CI no SHA da base.
- O produto **não** é `1.0.0` certificado, conforme critérios do README/ROADMAP.
- Há matrizes reais de race/SQL, restarts, signed return/review/correction, fuzz, distribuição, browser, capacity 100/1.000 e relatórios com limitações.
- Há validação humana de compreensão/UX e zoom/leitor de tela ainda pendente; isso não equivale a bug operacional do Core.
- 10.000 itens, milhares de writers assinados, percentis de produção, comportamento de múltiplas instâncias e operação real com SLA continuam não demonstrados em toda extensão.

### 10.2. Trabalho requerido dentro do PR

1. Reexecutar testes integrados por SHA e arquivar relatórios com parâmetros, ambiente e falhas.
2. Expandir testes de leitura 100/1.000/10.000 e signed-workload 1/4 Outcomes, 2/8/50 concorrentes; acrescentar backlog de integração, rejeições de transporte e resolução terminal.
3. Introduzir SLOs de **teste** delimitados e medir p50/p95/p99, alocações, CPU, DB pool wait, conflitos CAS, latência da fila, retries e taxas de recuperação; diferenciar indicadores de laboratório e SLA externo.
4. Registrar compatibilidade v0.4.2 → novo source: migração, schemas, antigos recibos/DSSE, CLI/workspace v1/v2 e leitoras antigas contra DB atualizado (falha segura).
5. Consolidar relatórios operacionais e limites suportados em `docs/stable/`/`docs/operations.md`; não marcar 1.0 como publicada nem adicionar número de usuários/sessões sem evidência.
6. Revisar documentação histórica que ainda diga que 0.4.2 não foi publicada, distinguindo data do registro histórico e estado atual.

---

## 11. Contratos e compatibilidade — regras transversais

| Superfície | Regra do PR |
|---|---|
| API HTTP `/api/v1` | Manter rotas/DTOs atuais; somente endpoints/campos opcionais aditivos. Não mudar sem versionamento os significados de erro/replay. |
| MCP | Novas tools geradas do mesmo catálogo; autorização/paridade iguais ao HTTP. Não expor debug/secrets ao Agent. |
| Go SDK | Adicionar métodos tipados para resolução e diagnóstico; não modificar retorno legado para afirmar aceitação sem recibo. |
| CLI `wos` / alias `wosctl` | Subcomandos aditivos, estados/journal compatíveis. Sem exclusão local de intenção incerta. |
| Signed v2 | Manter DSSE, Ed25519, binding original, `request_id`, CID, namespace, issuer, idempotency e fencing. Resolução usa envelope novo, nunca mutação do antigo. |
| v1 e schema 1 | Sem mudança de lifecycle/contrato; testes regressivos obrigatórios. |
| PostgreSQL/SQLite | Migração **0027+** (se sequência ainda disponível), checksums novos, sem editar 0001–0026, restore em DB novo. |
| Memory | Semântica idêntica para testes e embedded hosts; não assumir durabilidade entre processos. |
| Observabilidade | Sem payloads, tokens, URL/segredo de webhook, prompts ou descrições documentais em logs/labels. |
| Configuração | Defaults existentes preservados quando possível; novos parâmetros com valores seguros, validação e documentação. |
| Publicação | Não publicar uma nova release ou declarar estável sem gates específicos do mesmo source e fluxo já aprovado. |

### Matriz de ameaças/invariantes de não regressão

- Duas credenciais mesmo Principal não contornam quota, aceite da intenção original, separação de review ou revogação.
- Um Principal não observa recibo/tombstone/histórico de outro Namespace.
- Um envelope incorreto mas criptograficamente bem assinado não altera autoridade nem vira critério concluído.
- Limites de conexão e respostas de overload não criam repetição implícita de mutação.
- Redelivery de webhook é **at-least-once**; o receptor deduplica por identidade estável.
- Cursor de continuidade não mistura `OutcomeRevision`, `Namespace`, `evaluated_at` ou critério de filtro incompatível.
- Admin e host podem reconfigurar operação; WOS não passa a decidir as próximas Tasks nem iniciar Agents.

---

## 12. Execução — uma branch, um PR, vários commits

**Branch sugerida:** `fix/wos-operational-hardening-v042`  
**Título sugerido do PR:** `fix(wos): harden signed recovery, delivery health, HTTP and continuity at scale`  
**Estado inicial/final do PR:** `Draft`, sem auto-merge e sem merge manual até autorização explícita.  
**Base:** último `master` autorizado; registrar SHA inicial e revalidar qualquer alteração de base antes de continuar.

**Regra:** cada item abaixo é **um commit no mesmo PR**, em ordem. Nenhuma feature branch adicional, PR auxiliar de testes, PR de documentação ou PR separado de migração. Se integração exigir outro PR por razão técnica, interromper esse caminho e renegociar o escopo — não fragmentar silenciosamente.

### Commit 01 — `test(baseline): freeze reproducible operational failures and invariants`

- Criar reproduções de F01 (review/work congelado e sem caminho terminal); F02 (conexões lentas/saturação); F03 (worker degradado com API ready); F04 (reconstrução integral apesar de paginação) e F05 (cota 3/1).
- Preservar golden responses/cursors v0.4.2 e fixtures SQL. Capturar baseline de 100/1.000/10.000 quando viável no CI isolado, separando medições locais de CI.
- Definir falha esperada, sem modificar mecanismos para passar artificialmente. Testes novos não podem derrubar todos os gates antes da implementação sem estratégia de red/green controlada.
- **Gate:** há evidência reproduzível e nenhum erro atual é classificado com severidade acima da observada.

### Commit 02 — `feat(core): define signed return terminal-resolution contract`

- Criar domínio/port/DTO de resolução, estados, validação estrita e recibo assinado aditivo.
- Especificar identidade imutável de intenção original e invariantes de autoridade; registrar ADR curto.
- Criar implementações Memory e scaffolding SQL + migrations novas para SQLite/PostgreSQL.
- **Gate:** contrato imutável, constraints/uniqueness e migração/rollback transacional testados.

### Commit 03 — `fix(core): serialize signed acceptance against terminal resolution`

- Integrar arbitragem nos comandos `ReturnSignedWork`, `ReturnSignedReview`, no pipeline de signed operations e no storage.
- Adicionar `ResolveSignedReturnIntent`, replay e consulta read-only; garantir ordem de locks e a impossibilidade de aceitação após decisão terminal.
- **Gate:** todas as corridas aceito↔cancelado e múltiplas resoluções passam nos três adapters; invariantes de Outcome/Task, cas, authority, receipts imutáveis.

### Commit 04 — `feat(api): expose signed resolution through HTTP MCP and Go SDK`

- Atualizar catálogo gerado de commands, OpenAPI, MCP, SDK, DTOs estritos, observações sem segredos e auth.
- Testar equivalência de transporte, `Idempotency-Key`, erro terminal tipado e revalidação de credencial.
- **Gate:** um envelope idêntico produz decisão/replay equivalente via HTTP e MCP; SDK não reenvia mutações implicitamente.

### Commit 05 — `feat(cli): reconcile rejected frozen returns with signed terminal receipts`

- Adicionar comandos read-only de diagnóstico e comando de resolução para work/review; estados locais/journal aditivos.
- Preservar draft, envelope, CID, recibo e prova no local seguro; só permitir novo draft após resolução persistida e autoridade válida.
- Alinhar pré-validações estáveis do CLI e skill guides sem ensinar o Agent a ignorar erros de autoridade.
- **Gate:** flows Linux/Windows, original receipt, resolução, reexecução, segurança de arquivo, alterações concorrentes.

### Commit 06 — `test(recovery): prove restart timeout corruption and ENOSPC journeys`

- Reproduzir perda de resposta após commit, crash de CLI em cada fase, corrupção controlada, lease expirada, mismatch de issuer e `ENOSPC` real.
- Executar restore de banco com cancelamento terminal e replay tardio; corrigir qualquer lacuna encontrada sem novo PR.
- **Gate:** reconciliadores nunca apagam alteração não confirmada nem criam dupla execução.

### Commit 07 — `fix(server): bound HTTP connections requests and admission`

- Introduzir config/env de deadlines, limites de headers e in-flight, middleware de overload e métricas.
- Ajustar MCP/streaming com testes de transporte real; atualizar `config validate` e testes de shutdown.
- **Gate:** saturação e slow-client não retêm conexões indefinidamente; chamadas normais e MCP permanecem compatíveis.

### Commit 08 — `feat(integration): expose durable backlog and supervised worker health`

- Agregações read-only por Namespace no armazenamento, diagnóstico admin e observabilidade de heartbeat/tick/ACK.
- Supervisão do worker, estados disabled/degraded/healthy/unknown e erro de configuração.
- **Gate:** backlog/exhaustion detectável sem derrubar `/readyz` por falha externa; auth/sem segredos.

### Commit 09 — `perf(integration): add bounded delivery concurrency with fencing`

- Adicionar concorrência opt-in/backpressure, default 1, perfil PostgreSQL, teste multiworker/duas instâncias e fairness suficiente.
- Validar `ClaimDelivery`/`FinishDelivery` e índices, medir throughput e retomada após processo morto.
- **Gate:** idempotência da fila, at-least-once, HMAC e fencing intactos.

### Commit 10 — `refactor(core): introduce bounded continuity read ports`

- Introduzir API de projeções e contagens no Core/ports, mantendo API pública unchanged.
- Definir contrato de cursor legado/novo e de tempo de avaliação/revisão.
- **Gate:** adapter Memory reproduz todas as seções e paridade com goldens v0.4.2.

### Commit 11 — `perf(storage): index scoped continuity and focal graph queries`

- Implementar SQLite/PostgreSQL com páginas no banco, agregações e leitura por referência/dependência.
- Adicionar migrações/indexes novos e verificar planos SQL, concorrência e autorização.
- **Gate:** resultados equivalentes, scans evitados, sem N+1 e sem novas leituras fora do escopo.

### Commit 12 — `perf(application): stop full Outcome hydration on paged reads`

- Migrar `GetContinuitySection`, `GetContinuity`, `GetOutcomeGraph` e trechos de `GetWorkContext` para portas otimizadas; preservar versão, ordem/cursor e provas.
- Testes de diferença de snapshots e expiração de lease sem alteração de revisão.
- **Gate:** respostas correctas, 10.000 itens no envelope medido e metas de alocação/latência cumpridas ou bloqueio explícito.

### Commit 13 — `feat(signing): configure per-Principal execution and review quotas`

- Acrescentar campos opcionais de Namespace em `WorkAcceptancePolicy`, fallback 3/1 e validação/admin CAS.
- Aplicar limites nas aquisições assinadas sem permitir bypass entre credenciais/Outcomes.
- **Gate:** testes de corrida e default histórico passam Memory/SQLite/PostgreSQL; política não amplia autoridade existente.

### Commit 14 — `feat(ops): qualify shared PostgreSQL deployment and config diagnostics`

- Criar referência de deployment compartilhado e diagnósticos opt-in de produção; preservar Compose local.
- Medir contenção de SQLite vs PostgreSQL; verificar ingress/TLS/segredos/issuer/pool e toolchain do Docker.
- **Gate:** Compose local e perfil PostgreSQL qualificam sem depender de secrets exemplares em produção.

### Commit 15 — `feat(ops): provide verifiable backup restore and issuer runbooks`

- Scripts opt-in, checksums/manifests, validação de backup restaurado, documentação de issuer/key history/CLI workspaces e retenção.
- Testar restore em arquivos e bancos novos, recovery de outbox e credenciais revogadas.
- **Gate:** restore com prova de estado, sem sobrescrever instância e sem introduzir scheduler no Core.

### Commit 16 — `test(acceptance): enforce full cross-adapter operational matrix`

- CI de contratos, race, fuzz, carga, browser, Linux/Windows nativos, schema/compatibilidade, distribuição e documentação de skips.
- Capturar relatórios de performance e regressão por SHA; integrar testes de falhas reais.
- **Gate:** sem skip injustificado em teste obrigatório, sem falha de segurança/estado e com relatórios completos.

### Commit 17 — `docs(release): reconcile v0.4.2 baseline and operational acceptance`

- Atualizar README, ROADMAP, docs de operação/ADR, matriz de suporte/limitações, OpenAPI/SDK docs, changelog/changes e golden contract fingerprints.
- Preparar somente metadados de versão compatíveis com **este mesmo PR**, se a política de publicação exigir. Não abrir um PR mecânico adicional para concluir a **correção**; ajustar preparação ou adiar publicação até autorização, mantendo fonte revisada.
- **Gate:** docs refletem código e versão efetivamente publicada; nenhuma declaração infundada de 1.0/SLA.

### Commit 18 — `chore(verify): bind final acceptance to exact PR source`

- Reexecutar toda matriz após o último ajuste; fixar SHA, `go version`, PostgreSQL, Node/npm, artefatos, hashes e evidências.
- Registrar checklist assinável de revisão do proprietário, testes que passaram, skips e quaisquer divergências aceitas.
- **Gate:** o PR fica **Draft, aberto, não mergeado**, com todos os commits e gates verdes/justificadamente bloqueados. **Somente autorização explícita permite merge.**

> **Commits de correção adicionais são permitidos dentro desse mesmo PR**, entre os commits planejados, desde que cada um preserve causa clara, teste de regressão e contexto. Nunca fazer squash prematuro que esconda a rastreabilidade; não abrir segundo PR.

---

## 13. Matriz integrada de verificação obrigatória

| Área | Memory | SQLite real | PostgreSQL real | HTTP | MCP | Go SDK | CLI / plataforma |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| Return/resolution aceito versus cancelado | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | Linux + Windows |
| Idempotência, revogação, CAS e fencing | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | Linux + Windows |
| Review independente/correction | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | Linux + Windows |
| State/continuity/cursor legado e novo | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | Cliente onde aplicável |
| Lease expirado sem mutação | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | Cliente onde aplicável |
| Quota por Principal e múltiplas credenciais | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | CLI assinado |
| Backlog, lease de delivery e recuperação | Opcional | ✓ | ✓ | ✓ | — | — | 2 réplicas em PostgreSQL |
| HTTP slow-client/saturação/timeout | — | ✓ | ✓ | ✓ | ✓ | ✓ | Browser |
| Backup/restore/issuer e schema futuro | — | ✓ | ✓ | ✓ | ✓ | ✓ | Linux + Windows, conforme suporte |
| Performance 100/1.000/10.000 | Contrato | ✓ | ✓ | Via smoke | Via smoke | Via smoke | Medições de cliente |
| Browser, release, arquivo corrompido/ENOSPC | — | ✓ | ✓ | ✓ | ✓ | ✓ | Linux + Windows |

**Comandos mínimos (adaptar aos workflows existentes, não tratá-los como evidência por si):**

```bash
go mod tidy
go vet ./...
go test ./...
go test -race ./...
# Testes reais de PostgreSQL: WOS_TEST_POSTGRES_DSN em banco descartável.
# Executar geradores oficiais e exigir git diff limpo para código gerado.
# Reexecutar browser, package/release, native CLI/keyring e aceitação de distribuição.
# Reexecutar ferramentas de benchmark/continuity com as fixtures declaradas.
# Executar os cenários de falha e restore novos, com source SHA anexado.
```

Não executar teste destrutivo contra banco de produção. Evidência proveniente de uma versão anterior não certifica o SHA final.

---

## 14. Plano de implantação e rollback

### Antes do merge

1. Gerar backup consistente do estado vigente em ambiente de teste equivalente, incluindo banco, trust history, configuração/issuer protegido e workspaces locais aplicáveis.
2. Verificar migração v0.4.2 → schema novo em clone, mantendo versão anterior íntegra para comparação.
3. Validar comportamento de CLI antigo e serviço antigo contra schema novo: **falha segura**, sem writes silenciosos e sem “rollback de dados” por downgrade.
4. Executar canário de modo `contracts_v1`/schema-1, signed-v2 existente sem resolução nova, e signed-v2 com resolução nova.
5. Verificar nova configuração HTTP/MCP/worker sem sobrecarga; preparar observabilidade e alarmes antes do tráfego.

### Ordem de rollout recomendada

1. Aplicar backup/issuer check e interromper writers incompatíveis.
2. Implantar servidor novo, com migrations checksummed e configuração conservadora (delivery concurrency = 1, cotas padrão 3/1, fluxos anteriores válidos).
3. Rodar readiness e diagnóstico operacional autenticado; testar replay, webhook e consulta paginada.
4. Atualizar SDK/CLI de forma controlada; nova resolução é opt-in e deve negociar/confirmar suporte do servidor.
5. Ativar limites/quotas de maior paralelismo e mais workers **apenas após medição**; acompanhar conflitos, fila, snapshots e erros de retry.
6. Registrar source SHA, resultados de smoke e regra objetiva de abortar rollout.

### Estratégia de rollback

- Migrações são forward-only: **não** executar SQL downgrade destrutivo, não apontar binário antigo para schema novo incompatível.
- Se necessário, isolar a instalação e restaurar backup **em banco novo**, com binário/configuração correspondente; reconciliar eventos e fatos ocorridos após o snapshot antes de reabrir ingress.
- Decisões de abandono assinadas e receipts precisam permanecer verificáveis na restauração; nunca ressuscitar aceitação de intenção terminal só por troca de cache/workspace.
- Quotas novas e concorrência do worker devem poder voltar aos defaults **sem** schema downgrade; limites HTTP têm rollback por configuração, preservada compatibilidade de cliente.
- Toda consequência externa (webhook já entregue, ferramenta já executada por host) exige reconciliação própria — WOS não fornece exatamente-once externo.

---

## 15. Checklist final de revisão e proibições

### Bloqueadores de merge

- [ ] Apenas **um PR** contém todos os commits, migrations, scripts, testes, schemas e documentação.
- [ ] Os achados P1 possuem reprodução, correção e prova de não regressão; limitações remanescentes estão explicitadas.
- [ ] Intenção signed aceita não pode ser abandonada; intenção abandonada não pode voltar a ser aceita; corridas cobertas nas três stores.
- [ ] HTTP/MCP suportam deadlines e saturação controlados, sem replay implícito de mutação.
- [ ] Saúde da fila é observável e autorizada; `/readyz` não confunde indisponibilidade de terceiro com incapacidade da API.
- [ ] Continuity paginado não hidrata integralmente o Outcome sem necessidade; cursor legado, revisões e tempo preservados.
- [ ] Quotas por Principal respeitam defaults, CAS e ceilings superiores; nenhuma elevação implícita de privilégio.
- [ ] Perfil PostgreSQL, backup/restore, issuer e CLI workspaces foram exercitados com fixtures reais.
- [ ] `go test ./...`, `go test -race ./...`, `go vet ./...`, SQL real, browser, fuzz, release e native Linux/Windows passam no SHA final.
- [ ] Relatórios de escala apresentam números reais, método, fonte SHA, hardware e limitações — sem promessas de SLA.
- [ ] Os documentos não confundem v0.4.2 publicada com 1.0 ainda não certificado.
- [ ] Nenhum novo PR/merge automatizado foi usado para dividir a solução.
- [ ] **Merge proibido até autorização explícita do proprietário.**

### Fora de escopo, deliberadamente

- Implementar orquestração/execução autônoma de Agents dentro do WOS.
- Transformar WOS em scheduler de jobs, motor de workflow ou gerenciador genérico de runtime.
- Criar integração proprietária de banco de terceiros, SaaS de monitoramento ou sistema de secret management dentro do Core.
- Reescrever contratos signed v2, remover suporte v1, reescrever a história ou publicar versão `1.0.0` sem certificação.
- Migrar todo sistema para event sourcing, Redis, Kafka ou outro banco só para mascarar limites de consulta.
- Declarar isolamento de credenciais quando executor e reviewer compartilham o mesmo usuário/OS/filesystem; isso exige isolamento real do host.

---

## 16. Referências verificadas da base

- [Commit base `b572269`](https://github.com/A1b3rt0M3rcad0/wos/commit/b572269085494ddd4d6370716ff1cb360b46065f) e [release v0.4.2](https://github.com/A1b3rt0M3rcad0/wos/releases/tag/v0.4.2).
- [Core: `pipeline.go`](https://github.com/A1b3rt0M3rcad0/wos/blob/b572269085494ddd4d6370716ff1cb360b46065f/packages/wos-core/application/pipeline.go), [signed work return](https://github.com/A1b3rt0M3rcad0/wos/blob/b572269085494ddd4d6370716ff1cb360b46065f/packages/wos-core/application/signed_return.go), [signed review return](https://github.com/A1b3rt0M3rcad0/wos/blob/b572269085494ddd4d6370716ff1cb360b46065f/packages/wos-core/application/signed_review_return.go).
- [CLI: `return_sign_v2.go`](https://github.com/A1b3rt0M3rcad0/wos/blob/b572269085494ddd4d6370716ff1cb360b46065f/packages/wos-cli/return_sign_v2.go), [CLI: `return_send_v2.go`](https://github.com/A1b3rt0M3rcad0/wos/blob/b572269085494ddd4d6370716ff1cb360b46065f/packages/wos-cli/return_send_v2.go).
- [Continuity](https://github.com/A1b3rt0M3rcad0/wos/blob/b572269085494ddd4d6370716ff1cb360b46065f/packages/wos-core/application/continuity.go), [OutcomeState](https://github.com/A1b3rt0M3rcad0/wos/blob/b572269085494ddd4d6370716ff1cb360b46065f/packages/wos-core/application/query.go), [Graph/Focal](https://github.com/A1b3rt0M3rcad0/wos/blob/b572269085494ddd4d6370716ff1cb360b46065f/packages/wos-core/application/focal_graph.go).
- [HTTP runtime](https://github.com/A1b3rt0M3rcad0/wos/blob/b572269085494ddd4d6370716ff1cb360b46065f/packages/wos-api/internal/server/runtime.go), [Integration worker](https://github.com/A1b3rt0M3rcad0/wos/blob/b572269085494ddd4d6370716ff1cb360b46065f/packages/wos-api/integration/worker.go), [Postgres outbox](https://github.com/A1b3rt0M3rcad0/wos/blob/b572269085494ddd4d6370716ff1cb360b46065f/packages/wos-core/storage/postgres/integration.go).
- [Signed quotas de execução](https://github.com/A1b3rt0M3rcad0/wos/blob/b572269085494ddd4d6370716ff1cb360b46065f/packages/wos-core/application/signed_acquisition.go), [review](https://github.com/A1b3rt0M3rcad0/wos/blob/b572269085494ddd4d6370716ff1cb360b46065f/packages/wos-core/application/signed_review_acquisition.go), [policy](https://github.com/A1b3rt0M3rcad0/wos/blob/b572269085494ddd4d6370716ff1cb360b46065f/packages/wos-core/domain/signing_identity.go).
- [Operação e capacity](https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/stable/operations-reference.md), [roadmap](https://github.com/A1b3rt0M3rcad0/wos/blob/master/ROADMAP.md), [compatibilidade](https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/stable/compatibility.md), [piloto Codex](https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/audit/codex-managed-pilot-2026-10-12/report.json).

**Observação de execução:** nomes de novos arquivos/tipos/DTOs/endpoints são propostas de design, não afirmações de que já existem. O agente implementador deve escolher as nomenclaturas finais coerentes com os catálogos e geradores existentes, sem alterar as garantias e critérios de aceite deste documento.
