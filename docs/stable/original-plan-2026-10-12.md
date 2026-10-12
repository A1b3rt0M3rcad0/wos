# WOS — Estado atual e plano de conclusão da versão 1.0.0

**Atualizado em:** 12 de outubro de 2026  
**Repositório:** https://github.com/A1b3rt0M3rcad0/wos  
**Referência analisada:** `master`, commit `40d1703bdc8af718b15b7f7bab3d66b1def29806`  
**Versão atual do código:** `0.4.0`  
**Versão-alvo:** `1.0.0`  
**Escopo deste plano:** somente o WOS, incluindo Core, servidor, API, MCP, interface humana, `wosctl`, instalador nativo de skills e distribuição.

> **Regra de status:** `CONCLUÍDO` significa implementação presente e evidência de execução no escopo indicado. `PARCIAL` significa funcionalidade implementada, mas com algum critério de aceite ainda não demonstrado. `PENDENTE` significa entrega ou evidência ainda necessária. Uma versão preparada não equivale a uma versão publicada.

## 1. Objetivo da 1.0.0

Lançar o **WOS como sistema autônomo e confiável de coordenação persistente de trabalho**, independente do runtime de agentes, utilizável por humanos e aplicações via interface web, HTTP, MCP e CLI. O WOS deve permitir planejar, reservar trabalho, entregar material, revisar, corrigir, retomar depois de falhas e preservar histórico e autoridade de forma verificável.

### Matriz de suporte proposta para a 1.0.0

| Componente | Escopo inicial de suporte | Fora do escopo obrigatório |
| --- | --- | --- |
| Servidor `wos` | Linux amd64, via binário/container | Servidor Windows/macOS/ARM sem homologação nativa |
| Armazenamento | SQLite para desenvolvimento/local; PostgreSQL para instalação compartilhada | Novos bancos ou cache distribuído como requisito |
| Interfaces | HTTP `/api/v1`, MCP Streamable HTTP e stdio local, interface web | Conector proprietário para cada agente/produto |
| Cliente `wosctl` | Linux amd64 e Windows amd64 | macOS/ARM sem testes nativos |
| Contratos | `signed_contracts_v2` como recomendação para novos ambientes; compatibilidade legada explícita | Migração silenciosa ou assinatura retroativa do legado |
| Instalação de skills | `wos skill` nativo Go; Codex, Claude, Hermes, OpenClaw e destino genérico | Certificação de execução em toda versão de todo agente |
| Integração | SDK Go, HTTP, MCP, eventos e webhooks | WOS executar, hospedar ou agendar agentes |

**Princípio:** não introduzir um novo motor ou alterar a ontologia do WOS apenas para chegar à versão 1.0. A prioridade é consolidar contratos públicos e comprovar estabilidade operacional.

## 2. O que já foi feito

### 2.1. Core, domínio e coordenação — CONCLUÍDO

- [x] Estrutura Go em Domain, Application, Ports e adapters, com fronteiras de dependência verificadas.
- [x] `Namespace`, `Outcome`, `Objective`, `WorkItem`, critérios de sucesso e avaliações explícitas.
- [x] Roadmaps com Draft, publicação de revisões imutáveis, ativação separada e histórico.
- [x] Dependências operacionais, detecção de ciclos, prioridades, readiness, Issues e Blockers distintos.
- [x] Evidence, Artifacts, Decisions, relações, eventos, histórico de avaliações e conclusões contestáveis.
- [x] Escritas transacionais, `expected_version`, idempotência, guards por Outcome e tokens de fencing.
- [x] Consultas de continuidade, contexto focal, timeline, paginação e omissões explícitas.
- [x] Triggers e outbox de integração sem executar agentes autonomamente.

### 2.2. Contratos, assinatura e revisão — CONCLUÍDO no escopo implementado

- [x] Contratos de trabalho e aquisição explícita com lease, renovação, expiração, retomada e revogação autorizada.
- [x] Protocolo `signed_contracts_v2` com envelopes e recibos assinados Ed25519.
- [x] Identidades de servidor/issuer persistentes; enrollment e políticas de credenciais.
- [x] Perfis independentes de executor/revisor, integração com armazenamento protegido de segredos no host e binding de credenciais.
- [x] Devolução de trabalho assinada, aceite vinculado ao material, revisão independente e ciclos de correção.
- [x] Recuperação de intenções e recibos originais sem renovar silenciosamente autoridade ou alterar payload/CAS.
- [x] Limpeza local do contrato somente depois do recibo de encerramento verificado, preservando material alterado ou incerto.
- [x] Ativação de protocolo por Namespace com drenagem e verificação explícita de writers antigos.

**Importante:** implementação presente não significa que todos os 96 cenários do plano de aceitação estejam formalmente auditados; isso aparece em pendências.

### 2.3. Persistência e operações — CONCLUÍDO como fundação

- [x] Adaptadores Memory, SQLite e PostgreSQL, com testes compartilhados de contrato.
- [x] Migrações SQL numeradas, checksums e controle de compatibilidade de schema.
- [x] Backup/restauração SQLite e jornadas de restore PostgreSQL com verificação de continuidade.
- [x] Testes de concorrência com conexões SQL reais, race detector, falhas transacionais e redelivery.
- [x] Health checks `/livez` e `/readyz`; encerramento gracioso e execução de contêiner não root.
- [x] Logs estruturados e endpoint JSON de métricas operacionais.
- [x] Documentação de autenticação, recovery de issuer, atualização, restore e integração.

### 2.4. Superfícies públicas e DX — CONCLUÍDO no escopo implementado

- [x] API HTTP e MCP usando serviços de Application comuns; SDK Go e catálogo tipado de comandos.
- [x] MCP Streamable HTTP e stdio local; autenticação/autorizações no servidor.
- [x] Interface web em inglês com seis áreas de navegação, formulários específicos, seleção de referências pesquisável, editor visual de Roadmap e tratamento de conflito.
- [x] `wosctl` com projetos/profiles, checkout, show, sign, send, finish, recover e operações de revisão.
- [x] Skills `wos-coordination`, `wos-delegation`, `wos-continuity`, `wos-review` e `wos-workspace`.
- [x] **Instalador nativo Go integrado ao comando `wos skill`** (PR #87 integrado): instalação, atualização, verificação, listagem, remoção, `--dry-run`, projeto, global e diretório genérico.
- [x] Instalação de skills não requer Node, npm, API, token ou servidor em execução, quando se dispõe do binário nativo `wos`.

**Comandos já implementados:**

```bash
wos skill install --agent codex
wos skill install --agent claude
wos skill install --agent hermes
wos skill install --agent openclaw
wos skill install --agent all
wos skill install --agent generic --dir /caminho/skills
wos skill doctor --agent codex
wos skill update --agent codex
wos skill uninstall --agent codex
```

`--agent all` cobre os quatro destinos conhecidos; qualquer outro agente compatível com Agent Skills exige configurar o caminho com `--agent generic --dir`. A skill não instala MCP nem autoriza o agente automaticamente.

### 2.5. Empacotamento e validação — CONCLUÍDO em código/testes, publicação separada

- [x] Binários Go e pacote npm do servidor preparados, além de artefatos nativos do cliente.
- [x] Novo pacote npm independente para o CLI, com executáveis Go `wos` e `wosctl` para Linux/Windows amd64.
- [x] Wrapper npm para seleção de plataforma/integridade; execução funcional permanece em Go. **Node é necessário para instalar pelo npm, não para o instalador Go de skills.**
- [x] Nome npm do CLI corrigido para `@woobeai/wos-cli` depois do erro `E404 Scope not found` no escopo anterior.
- [x] Verificação de SHA-256, instalação de tarball offline, integridade de binários e releases vinculadas ao commit.
- [x] Testes nativos Linux/Windows do CLI e keyring; testes de navegador e de pacotes no commit de publicação.
- [x] PR #90 integrado à `master` e versão `0.4.0` preparada.
- [x] Suite backend completa da execução de publicação atual aprovada, incluindo Go, race, migração, restore e carga.

**Estado de publicação do CLI:** o workflow do commit `40d1703b` terminou com **falha apenas no job de publicação**, depois que os cinco grupos de validação passaram. O publicador executou `npm publish`, mas a leitura subsequente de `dist.integrity` não confirmou imediatamente o valor esperado (`Published npm integrity could not be verified`). **Não está comprovado se o npm aceitou o pacote e ainda não propagou os metadados, ou se a publicação não se tornou disponível.** Exigir consulta ao registro e comparação da integridade antes de repetir o upload.

## 3. O que falta — prioridade P0 (bloqueia a conclusão formal da release)

### P0-01 — Encerrar e comprovar a publicação npm do CLI

**Situação:** a tentativa inicial falhou pelo escopo `@a1b3rt0m3rcad0`; o nome foi corrigido para `@woobeai/wos-cli`. No commit `40d1703b`, todas as verificações de Go, race, PostgreSQL, navegador, pacotes e plataformas nativas passaram. O job de publicação foi executado, mas terminou em **falha de verificação pós-publicação**: `Published npm integrity could not be verified`. O comando `npm publish` não gerou erro próprio no log, portanto **é possível** que o registro esteja apenas em propagação; isso precisa ser comprovado. Não criar nova versão nem sobrescrever a mesma versão sem verificar o registro.

**Ações:**

1. Consultar diretamente o registro npm e o endpoint de metadados de `@woobeai/wos-cli@0.4.0`; confirmar `dist.integrity` e SHA do artefato gerado pelo commit `40d1703b`.
2. **Se a versão já existir com a integridade correta:** registrar o sucesso e corrigir o pós-publish do workflow para aguardar a propagação; não tentar publicar novamente.
3. **Se a versão ainda estiver ausente após consulta confiável:** executar um retry controlado do pacote idêntico, com o mesmo source/artifact, verificando identidade e autorização do publisher.
4. **Se existir com integridade diferente:** interromper imediatamente; nunca sobrescrever uma versão npm imutável. Investigar proveniência e preparar uma nova versão quando necessário.
5. Ajustar o publisher para fazer polling limitado após `npm publish`, diferenciando `404` temporário, erro de autenticação, rede e divergência de integridade; registrar diagnóstico claro, sem expor token.
6. Instalar a versão **publicada**, e não apenas o tarball local, em Windows e Linux limpos. Executar `wos version`, `wosctl version --output json`, `wos skill install`, `doctor`, `update` e `uninstall`; validar `--agent all` e `generic`.
7. Registrar recibo de publicação e comandos de instalação no README, sem declarar Windows como plataforma certificada do *servidor*.

**Aceite:** versão pública resolvível pelo npm, integridade idêntica ao artefato aprovado, instalação real e execução das operações nativas em Linux/Windows. Falhas de propagação não podem ser confundidas com pacote inexistente ou permitir republicação divergente.

### P0-02 — Formalizar o contrato da API/CLI/MCP para SemVer 1.0

**Ações:**

1. Congelar contratos públicos de HTTP, MCP, SDK Go, `wosctl`, formatos YAML, respostas de erro, payloads e recibos de assinatura.
2. Definir matriz de compatibilidade entre servidor, cliente, skills, versão do protocolo assinado, workspace schema e migrations.
3. Corrigir metadados de versão desatualizados: `packages/wos-api/mcp/server.go` ainda anuncia `0.1.0-dev`; revisar também `packages/wos-api/openapi.yaml`, sem confundir versão de API `/v1` com versão SemVer do produto.
4. Explicitar que `signed_contracts_v2` é opt-in e que versões antigas preservam histórico, sem autorizar writes legados após cutover.
5. Adicionar testes de não regressão e de compatibilidade com as versões anteriores efetivamente suportadas.

**Aceite:** metadados coerentes e contrato estável documentado; testes detectam incompatibilidade antes de publicar.

### P0-03 — Fechar auditoria dos cenários de contratos assinados

**Situação documentada:** 96 cenários T01–T96; 87 classificados como `mapped_requires_assertion_audit`, 6 como `implemented_requires_execution_and_assertion_audit` e 3 como `partial`. Isso **não significa 96 testes com falha**, nem que todas essas funcionalidades estejam ausentes.

**Ações:**

1. Vincular cada cenário crítico a assertions observáveis e ao commit efetivamente testado, com recibos e evidências verificáveis.
2. Reexecutar suites Memory/SQLite/PostgreSQL, cliente HTTP/MCP real e CLI nativo onde o cenário exigir.
3. Completar a jornada T89 com planner/executor/reviewer e correção; registrar claramente limites de isolamento do ambiente.
4. Finalizar ou delimitar formalmente T94 (carga e histórico denso), usando a matriz de capacidade adotada para a 1.0.
5. Não tornar alegações quantitativas sobre economia de tokens/custo do T95 condição de publicação; se não houver medição, registrar como **não demonstrado**, sem usar números inventados.
6. Diferenciar cenários aprovados, parcialmente aprovados, não aplicáveis e deliberadamente excluídos do contrato 1.0.

**Aceite:** nenhuma asserção crítica de autoridade, assinatura, fencing, revisão, resultado atômico ou recuperação fica sem cobertura vinculada à release candidata.

### P0-04 — Decidir licença e política pública antes da versão estável

**Constatação:** o repositório e os artefatos atuais declaram **Apache-2.0**. Essa é uma licença open source, não uma licença restritiva Source-Available.

**Ações:** confirmar a estratégia jurídica e comercial pretendida; manter Apache-2.0 conscientemente **ou** adotar, antes da 1.0, termos source-available com revisão de copyright, arquivos de licença, dependências e pacotes já distribuídos. Não apresentar uma política proprietária enquanto os artefatos publicados continuam sob outra licença.

**Aceite:** LICENSE, README, binários, imagens, npm e documentação comercial descrevem a mesma política efetiva.

### P0-05 — Fechar publicação coordenada e configuração do GitHub

**Situação:** `master` foi observada sem proteção (`protected: false`). Apenas um rascunho GitHub `v0.1.0`, sem assets, foi encontrado; a versão `0.4.0` no repositório não demonstra GitHub Release final. O CLI tem fluxo de publicação separado; serviço e skills não devem ser anunciados como publicados sem prova.

**Ações:**

1. Definir checks obrigatórios da `master`: backend/PostgreSQL/race, navegador, pacotes e CLI nativo Linux/Windows.
2. Ativar proteção da branch e impedir merge que ignore o gate.
3. Resolver a propriedade/escopo npm dos pacotes **de servidor** e **skills**, separados de `@woobeai/wos-cli`, antes de tentar sua publicação.
4. Configurar publicação versionada com SHA original, assets verificados, checksums, histórico de changelog e proteção de tags.
5. Eliminar inconsistências entre releases antigas em Draft, releases preparadas e releases oficiais, sem substituir artefatos imutáveis.
6. Publicar a 1.0 somente a partir do commit integrado cujas matrizes de testes estejam verdes.

**Aceite:** um único commit de release, quatro gates obrigatórios aprovados, artefatos recuperáveis e documentação que aponta para versões efetivamente acessíveis. Publicação manual/autorizada para releases futuras, salvo decisão explícita diferente.

## 4. O que falta — prioridade P1 (necessário para declarar operação estável em produção)

### P1-01 — Estabelecer limites de escala e desempenho

**Evidência atual:** em fixture documentada, PostgreSQL apresentou média de cerca de **92,2 ms** para snapshot com avaliações e **34,35 snapshots/s** com quatro leitores no mesmo Outcome. Uma carga com 50 consumidores no mesmo Outcome registrou **1.034 conflitos/retries**. Esses dados são de teste, não um SLA de produção.

**Ações:**

1. Medir pelo menos dois perfis: muitos contratos no **mesmo Outcome** e contratos distribuídos em **múltiplos Outcomes**.
2. Testar diferentes densidades de histórico (por exemplo: 100, 1.000 e 10.000 WorkItems, quando viável) e revisões/avaliações reais.
3. Medir p50/p95/p99, throughput de aquisição/devolução/revisão, conflitos transacionais, espera de guards/pool, consultas SQL, CPU e memória.
4. Investigar `GetContinuity` → `GetOutcomeState`: limitar bytes devolvidos não torna o custo de reconstrução completa do estado independente do tamanho do Outcome.
5. Otimizar projeções focais e consultas paginadas onde houver leitura excessiva, **sem** enfraquecer CAS, guards ou fencing.
6. Definir/documentar envelopes máximos e metas de desempenho sustentados pelos resultados, sem anunciar capacidade não medida.

**Aceite:** não há regressão de consistência nem falha silenciosa de autoridade nos limites declarados; as metas escolhidas são demonstradas no ambiente de referência.

### P1-02 — Política de crescimento de dados, backup e restore

**Ações:**

1. Documentar retenção por classe de dado: histórico, recibos, eventos, auditoria, outbox, checkpoints e idempotência.
2. Preservar trilhas e assinaturas necessárias à prova de trabalho; não aplicar expurgo genérico em registros imutáveis.
3. Definir automação/rotina de backups SQLite/PostgreSQL, periodicidade, retenção e ensaios de restauração.
4. Exercitar restore em instalação nova, verificando credenciais, issuer, leases, contratos assinados, submissions, histórico, outbox e replays.
5. Documentar procedimento seguro de migração, rollback por restauração e retirada de writers antigos.
6. Definir RPO/RTO do deployment suportado, sem assumir valores inexistentes no repositório.

**Aceite:** uma instância com contratos pendentes pode ser restaurada e retomada, preservando as obrigações e provas; a operação tem procedimento reproduzível.

### P1-03 — Operação, observabilidade e diagnóstico

**Ações:**

1. Registrar métricas utilizáveis de conflitos, latência, erros, backlog de entrega, retries, contratos expirados e disponibilidade.
2. Definir alertas e dashboards de referência, sem obrigar adoção de OpenTelemetry/Prometheus no Core.
3. Revisar os limites atuais de logs e métricas JSON (contadores em memória reiniciam com o processo; faltam percentis de produção), indicando quando coletor externo é necessário.
4. Garantir métricas suficientemente detalhadas para distinguir saturação de Outcome guard, pool SQL, autenticação, falha de cliente e armazenamento.
5. Validar operação em Docker com ingress TLS, segredos externos, persistência montada, health checks e recursos limitados.

**Aceite:** falhas operacionais e contenção são diagnosticáveis sem depender de inspeção manual de conteúdo sensível.

### P1-04 — Homologação final de segurança e isolamento

**Ações:** revisar permissões por Namespace/Principal, credenciais revogadas, callbacks MCP/HTTP, replay cross-scope, bound profiles, issuer rotation/recovery e testes de autorização negativa. Tornar explícita a proibição de credenciais de desenvolvimento em deployments compartilhados e a política de TLS no ingress. Auditar configurações de produção com tokens, URLs de webhook e leitura de métricas.

**Aceite:** nenhum fluxo validado permite obter autoridade por payload, ActorRef ou ExternalContext sem autenticação e autorização correspondentes.

### P1-05 — Validação humana e acessibilidade da interface

**Situação:** 31 jornadas de navegador e 14 testes web unitários são evidência automatizada. O ROADMAP registra **não realizados**: baseline com usuários representativos, teste manual com screen reader e zoom nativo 200%/400%.

**Ações:** executar jornada humana não assistida de criação e ativação de Outcome, planejamento, bloqueio, avaliação de prova, handoff e retomada após conflito. Validar leitores de tela e zoom nativo. Corrigir obstáculos reproduzidos, sem novo redesenho visual abrangente.

**Aceite:** jornadas críticas completadas sem manipular JSON/UUID desnecessariamente; problemas graves de acessibilidade observados são corrigidos. Não declarar certificação WCAG ou percentual de usabilidade sem evidência.

### P1-06 — Instalação real como consumidor independente

**Ações:** usar a versão distribuída, fora do monorepo, em uma máquina Linux e uma Windows; conectar a serviço em staging; provisionar perfil autorizado; adquirir contrato, executar, registrar progresso, entregar, revisar, corrigir e recuperar resposta incerta. Confirmar compatibilidade de `wosctl` após atualização do servidor e após troca de sessão do agente.

**Aceite:** cliente e servidor funcionam de ponta a ponta utilizando somente interfaces públicas, sem dependência de paths, secrets ou helpers internos do checkout original.

## 5. O que NÃO deve bloquear a 1.0.0

- Suporte nativo de servidor a macOS, Windows ou ARM sem demanda comprovada.
- Execução, orquestração, scheduling, inferência ou delegação automática de agentes **dentro** do WOS.
- Integrações proprietárias específicas com cada runtime, CRM ou produto externo.
- Novo banco, Redis, fila distribuída ou infraestrutura adicional sem gargalo demonstrado.
- Economia de tokens/custo ou benchmark comparativo A/B/C com significância estatística — desejável, mas não fundamento obrigatório do contrato WOS.
- Qualquer promessa de que instalar uma skill implica integração funcional certificada para toda versão de todo agente.
- Plataforma Cloud gerenciada, billing, assinatura e multi-região, que constituem um projeto de distribuição/operação separado.

## 6. Ordem recomendada de execução

| Etapa | Entregas | Gate para avançar |
| --- | --- | --- |
| **A — Fechar 0.4.0** | P0-01: publicação npm CLI, instalação real e recibos | Pacote público, hash consistente e smoke Linux/Windows |
| **B — Congelar contrato 1.0** | P0-02, P0-04: versões/protocolos/licença | Contrato público e licença aprovados |
| **C — Verificação crítica** | P0-03; P1-04; P1-06 | Jornadas assinadas, permissão, replay e recovery reproduzíveis |
| **D — Estabilização operacional** | P1-01, P1-02, P1-03, P1-05 | Envelope de escala, restauração, diagnóstico e UX validados |
| **E — Release final** | P0-05, documentação, build e publicação 1.0.0 | Gates verdes no SHA definitivo; tag/assets/registry conferidos |

Cada etapa deve gerar evidências vinculadas ao commit real. **Não alterar a versão para 1.0.0 somente porque o checklist foi escrito.**

## 7. Checklist final de aprovação da 1.0.0

- [ ] Escopo suportado e contratos públicos congelados.
- [ ] Nome e distribuição do CLI no npm comprovados e testados em instalação limpa.
- [ ] Versões CLI/MCP/OpenAPI/artefatos/documentação sem drift.
- [ ] Decisão de licença consistente com `LICENSE` e pacotes distribuídos.
- [ ] `master` protegida por checks obrigatórios.
- [ ] Matriz de cenários críticos assinados auditada e aprovada, com limites explicitados.
- [ ] Concorrência e carga medidas nos envelopes de produção declarados.
- [ ] Backup, restore, upgrade e recuperação de issuer/contratos comprovados.
- [ ] Segurança, autenticação, isolamento e autorizações negativas homologados.
- [ ] Telemetria e procedimento de diagnóstico operacionais documentados.
- [ ] Jornadas humanas/acessibilidade verificadas no escopo assumido.
- [ ] Consumidor independente executa o ciclo completo sem checkout do WOS.
- [ ] Release 1.0.0 gerada a partir de um SHA imutável com verificações de Linux, Windows CLI, PostgreSQL, race, navegador e distribuição aprovadas.
- [ ] Assets/checksums/notas publicados e instalação de versão oficial efetivamente confirmada.

## 8. Evidências e referências verificáveis

- Código e estado: https://github.com/A1b3rt0M3rcad0/wos/tree/master
- Implementação e histórico: https://github.com/A1b3rt0M3rcad0/wos/blob/master/ROADMAP.md
- PR do instalador Go: https://github.com/A1b3rt0M3rcad0/wos/pull/87
- PR da distribuição npm Linux/Windows: https://github.com/A1b3rt0M3rcad0/wos/pull/89
- PR de ajuste de escopo: https://github.com/A1b3rt0M3rcad0/wos/pull/90
- Publicação CLI no commit final: https://github.com/A1b3rt0M3rcad0/wos/actions/runs/38291302569
- Matriz de cenários assinados: https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/signed-acceptance-matrix.json
- Evidência de Codex/correção: https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/audit/signed-v030-verification-2026-10-09.md
- Performance e contenção: https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/benchmarks.md
- UX final: https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/ux/final-acceptance.md
- Operação: https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/operations.md

**Conclusão:** a maior parte das funcionalidades centrais já está implementada. A versão 1.0.0 depende principalmente de **contratos estáveis, evidência completa de autoridade/recuperação, limites operacionais declarados, configuração segura e publicação reproduzível** — não de criar um segundo sistema de coordenação.
