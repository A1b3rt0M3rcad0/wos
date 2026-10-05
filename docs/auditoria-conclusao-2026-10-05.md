# WOS — Auditoria de situação e plano de conclusão

**Data de corte:** 5 de outubro de 2026.  
**Objetivo:** entregar um sistema de Outcomes independente, operável por humanos, agentes e aplicações, com continuidade durável, comprovação e coordenação compartilhada.  
**Repositório:** `A1b3rt0M3rcad0/wos`.

## 1. Escopo e conclusão

Esta é uma auditoria dirigida de arquitetura, implementação, testes e entregabilidade. Foram consultados o repositório remoto, trechos de código das camadas de domínio/aplicação/HTTP/storage, o plano de arquitetura, o roadmap, o PR ativo de planejamento e os resultados efetivos do GitHub Actions. Não houve alteração no repositório nem execução local de testes nesta análise. A existência de testes e a execução bem-sucedida da suíte remota não equivalem a uma certificação exaustiva do produto.

**Conclusão:** o WOS já tem uma implementação substancial do núcleo de estado e coordenação, com persistência SQLite e interface HTTP. Não está apenas no estágio de design. Entretanto, ainda não entrega integralmente a experiência de um sistema compartilhado por humanos e agentes: Roadmaps precisam de fechamento e integração; continuidade precisa de consultas completas e limitadas; faltam MCP operacional, identidade/autorização multiusuário, PostgreSQL, entrega de sinais de integração e a experiência humana de produto.

O trabalho restante não é reconstruir o domínio. É completar suas interfaces, tornar a continuidade utilizável, finalizar a operação compartilhada e provar os cenários completos.

### 1.1 Base integrada

- Branch: `master`.
- Commit: `2f9967689fa954b7c371781bc7a00c1ae760825d`.
- Commit de integração: `feat: Wave 10 verifiable conclusions (#12)`.
- Data do commit: 2 de outubro de 2026, 18:03:49, horário de Brasília.
- CI #206, execução `37064544568`: sucesso.

### 1.2 Trabalho em andamento, separado da entrega integrada

- PR #13: `feat: Wave 11 Roadmaps and planning history`.
- Branch: `feat/wave-11-roadmap-planning-history`.
- Commit observado: `0bd7c538dfdc4eb4a0732f60c3ec57bd341da345`.
- PR aberto e não integrado; 25 commits e 30 arquivos alterados no momento da leitura.
- CI #228, execução `37340922999`, job `111867478636`: sucesso, concluído em 5 de outubro de 2026 às 13:29:53, horário de Brasília.

A descrição inicial do PR já não representa todo o código da branch. Publicação, slots ativos, histórico, SQLite, HTTP e testes adicionais foram implementados depois da abertura.

### 1.3 Leitura correta do CI

A verificação remota passou por higiene de módulos, formatação, `go vet`, testes unitários/de contrato, detector de corridas, build, versão, validação de configuração e smoke HTTP. O smoke de processo consulta `/readyz` e `/livez`; ele não demonstra sozinho uma jornada completa humano → agente → humano, nem interfaces ainda ausentes, como MCP.

## 2. O que já existe e deve ser preservado

| Capacidade | Situação observada | Consequência |
|---|---|---|
| Core público em Go | Implementado | Pode ser composto sem runtime de agentes. |
| Separação `wos-core` / `wos-api` | Implementada | Não há necessidade de redistribuir o projeto em vários serviços para concluir o produto. |
| Outcome, Objective e WorkItem | Implementados | Lifecycle e conclusão são conceitos distintos. |
| Critérios, avaliações e Conclusions | Implementados na base integrada | A revisão avaliada e as evidências utilizadas podem ser preservadas historicamente. |
| Transações, versões, revisão do Outcome e idempotência | Implementados | Devem ser reutilizados pelas novas interfaces. |
| SQLite e adapter de memória | Implementados | PostgreSQL não está implementado no snapshot analisado. |
| Dependências e readiness | Implementados | Disponibilidade é derivada; não deve virar um status livremente gravável pelo frontend. |
| Issues e Blockers | Implementados separadamente | Resolver um problema não libera automaticamente todos os impedimentos. |
| Claims, leases e fencing | Implementados | Incluem renovação, liberação, retomada e proteção contra executor antigo. |
| Evidence, Artifact, EvidenceLink e Decision | Implementados | Evidência não equivale a avaliação; Artifact é referência documental, não armazenamento obrigatório de arquivos. |
| HTTP | Implementado para grande parte das operações | Ainda utiliza identidade local fixa e não tem a superfície completa de descoberta/continuidade. |
| Snapshot de Outcome | Implementado parcialmente | Já agrega entidades, bloqueios e contestações; ainda não é o contrato completo de continuidade. |
| Roadmaps | Implementação avançada no PR #13 | Não devem ser contados como entrega de `master` antes da integração. |
| Health checks e shutdown gracioso | Implementados | Precisam ser ampliados para os novos modos, não recriados. |
| MCP | Ainda não implementado na árvore inspecionada | Uma API HTTP utilizável por automação não equivale a um servidor MCP entregue. |
| Autenticação remota e grants | Pendentes | O servidor atual exige autenticação local. |
| PostgreSQL | Pendente | O runtime atual rejeita modos diferentes de SQLite. |
| Triggers e entrega durável | Pendentes como capacidade completa | Domain Events persistidos não são uma implementação de webhook/outbox completa. |
| Cliente visual humano | Não encontrado no repositório auditado | Deve virar uma entrega explícita para o objetivo amplo de produto. |

## 3. Definição de pronto

O WOS estará pronto para o objetivo de produto quando um humano puder criar um resultado verificável, organizar o plano, delegar trabalho e acompanhar sua execução; quando agentes distintos puderem consultar e atualizar esse mesmo estado por uma interface padronizada; e quando um terceiro participante conseguir continuar o processo após perda da sessão ou reinício do servidor, sem reconstruir o contexto a partir de chat.

A conclusão deve ser apoiada em avaliações explícitas e histórico preservado. Uma tarefa feita não pode concluir automaticamente um Objective; um Objective alcançado não pode, sozinho, certificar o Outcome inteiro.

Há duas entregas que não devem ser confundidas:

**Servidor 0.1:** Core, API, MCP, persistência, autorização, contratos, operação e documentação independentes.

**Produto humano + agente:** o servidor acima e um caminho humano realmente utilizável, seja um cliente oficial do WOS, seja uma integração de referência que cubra todos os fluxos. Esta auditoria recomenda um cliente web oficial mínimo. Não se presume que ele já faça parte das Waves existentes.

## 4. Backlog detalhado por frente

Os itens abaixo são trabalho recomendado. “Reconciliar” significa conferir evidência e documentação; não significa reimplementar uma funcionalidade já existente. As prioridades são de dependência e aceite, não estimativas de prazo ou percentual.

### E00 — Reconciliar o estado real e o contrato da entrega

**Situação:** README, cabeçalho do roadmap, seções de ondas e descrição do PR divergem entre si. Há testes implementados para itens ainda desmarcados. Existe também o PR #9 antigo de Wave 09, enquanto a entrega correspondente já foi integrada por outro PR.

- [ ] E00.01 Registrar um snapshot de auditoria com branch, commit, PR e CI de referência.
- [ ] E00.02 Atualizar o README para distinguir funcionalidades executáveis de funcionalidades projetadas.
- [ ] E00.03 Corrigir a onda corrente e o estado dos milestones no roadmap.
- [ ] E00.04 Revisar todos os checkboxes pendentes das Waves 10 e 11 contra código e testes concretos.
- [ ] E00.05 Para cada item considerado pronto, registrar arquivo/teste e execução que sustentam o aceite.
- [ ] E00.06 Atualizar a descrição do PR #13 para refletir a branch atual.
- [ ] E00.07 Comparar o PR #9 com a entrega integrada e encerrar/superseder o fluxo antigo somente depois de preservar eventual trabalho exclusivo.
- [ ] E00.08 Definir explicitamente quais interfaces humanas compõem a primeira entrega de produto.
- [ ] E00.09 Separar obrigações da versão 0.1 de extensões futuras, mantendo todos os requisitos originalmente prometidos que não forem formalmente renegociados.
- [ ] E00.10 Revisar a licença própria do WOS e acrescentar o arquivo correspondente antes da publicação aberta.

**Aceite:** nenhuma funcionalidade aparece simultaneamente como planejada, concluída e ainda não integrada sem explicação; a versão a entregar possui critérios verificáveis.

### E01 — Fechar Roadmaps e planejamento histórico

**Situação:** o PR #13 já contém criação, drafts versionados, publicação, snapshots, slots ativos, histórico de ativação, arquivamento, memória, SQLite e HTTP. `ReplaceRoadmapDraft` já aceita razão/metadados; existem testes de concorrência e testes SQLite de snapshots que sobrevivem a alterações posteriores e restart.

- [ ] E01.01 Revisar integralmente as alterações do PR sobre a base integrada, sem usar a descrição inicial como inventário atual.
- [ ] E01.02 Confirmar a separação entre versão do agregado, versão do draft, número da revisão e revisão transacional do Outcome.
- [ ] E01.03 Conferir os testes de referências cross-Outcome, cross-Namespace e fora da subárvore de Objective permitida.
- [ ] E01.04 Cobrir versões obsoletas tanto do Roadmap quanto do draft e respostas de erro esperadas.
- [ ] E01.05 Confirmar rollback integral de publicação mais alteração explícita de dependências quando qualquer parte falhar.
- [ ] E01.06 Validar ciclos de agrupamento, ordenação e dependências operacionais, sem confundir esses três grafos.
- [ ] E01.07 Conferir a imutabilidade de rótulos, referências, critérios e hash publicados após alterações no estado vivo e reinício.
- [ ] E01.08 Garantir que remover nó não exclua/cancele a entidade operacional e que cancelar trabalho não reescreva revisão publicada.
- [ ] E01.09 Validar unicidade de slot ativo e histórico também sob concorrência real de conexões SQLite.
- [ ] E01.10 Decidir se ativar um plano exige precondição sobre o slot ativo anterior, além da versão do Roadmap candidato.
- [ ] E01.11 Se o produto exigir proteção contra substituição por visão obsoleta, implementar CAS/versão do slot ou referência ativa esperada.
- [ ] E01.12 Conferir o contrato de arquivar/desarquivar Outcome e Roadmap, preservando consultas e histórico de acordo com a política declarada.
- [ ] E01.13 Validar migrações 0008/0009 a partir de banco existente, não apenas banco vazio.
- [ ] E01.14 Sincronizar HTTP, OpenAPI, documentação e testes antes de integrar o PR.

**Ponto de contrato:** o teste atual de ativação admite duas ativações concorrentes de Roadmaps diferentes, serializa ambas e preserva um único plano ativo ao final. Isso cumpre unicidade. Não prova que o segundo ator observou e consentiu com a substituição do plano ativado pelo primeiro. A decisão de adicionar uma precondição de slot deve ser explícita; não é correto declarar automaticamente que o comportamento atual é um bug.

**Aceite:** planos podem ser preparados, publicados, substituídos e consultados historicamente sem apagar trabalho e sem produzir duas fontes de verdade operacionais.

### E02 — Completar descoberta, continuidade e consultas

**Situação:** `GetOutcomeState` já reúne o estado central e contestações, mas carrega coleções inteiras. A estrutura atual não possui o envelope completo de limites, omissões e cursores; não inclui a projeção de Roadmaps. O registro de rotas inspecionado não oferece listagem da coleção de Outcomes, busca, grafo ou timeline completos.

- [ ] E02.01 Adicionar listagem e descoberta de Outcomes autorizados dentro do Namespace.
- [ ] E02.02 Adicionar filtros por lifecycle, arquivo, autoria/responsabilidade, prioridade, texto e contexto externo autorizado.
- [ ] E02.03 Definir paginação e ordenação determinísticas para Outcomes e coleções internas.
- [ ] E02.04 Construir um contrato de snapshot compacto, com versão de schema, `outcome_revision`, `evaluated_at`, limites e omissões.
- [ ] E02.05 Oferecer detalhe incremental por seção, em vez de exigir todo o grafo a cada consulta.
- [ ] E02.06 Definir a coerência entre múltiplas páginas: snapshot fixado, revisão conhecida ou invalidação explícita quando a base muda.
- [ ] E02.07 Incluir plano ativo, revisão, projeção do estado vivo e acesso às revisões históricas sem confundi-los.
- [ ] E02.08 Implementar queries de grafo com tipos de relações, direção, profundidade, limites e indicação de truncamento.
- [ ] E02.09 Implementar timeline paginada e filtrável, com principal, ator, comando, entidade e revisão.
- [ ] E02.10 Distinguir eventos de domínio internos da representação pública estável de histórico.
- [ ] E02.11 Expor decisões vigentes e suas predecessoras, não somente uma lista indistinta de documentos.
- [ ] E02.12 Expor bloqueios diretos/herdados com origem, motivo e o que precisa ser resolvido.
- [ ] E02.13 Expor trabalho pronto, em curso, programado, indisponível por lease e necessitando atenção.
- [ ] E02.14 Criar um pacote de contexto focal de WorkItem com objetivo, critérios, dependências, decisões e evidências pertinentes.
- [ ] E02.15 Permitir ordenar candidatos prontos por regras explícitas, sem fazer o WOS decidir autonomamente a execução.
- [ ] E02.16 Separar métricas de esforço/trabalho concluído de critérios comprovados e resultados alcançados.
- [ ] E02.17 Mostrar contestações e obrigações ainda não satisfeitas no resumo do Outcome.
- [ ] E02.18 Definir as consultas disponíveis para Outcomes arquivados e a retomada após desarquivamento.
- [ ] E02.19 Impedir acesso indevido por filtros, busca, cursores, referências externas e mensagens de erro.
- [ ] E02.20 Implementar índices e leituras agregadas para evitar scans repetidos e consultas por entidade onde não forem necessários.
- [ ] E02.21 Medir tamanho das respostas, quantidade de queries e espera de transação com datasets representativos.
- [ ] E02.22 Criar teste de continuação por cliente novo usando apenas descoberta, snapshot e cursores.

**Aceite:** um consumidor sem histórico de conversa descobre o Outcome certo, entende o plano e suas pendências, obtém o contexto necessário e continua o trabalho sem interpretar uma resposta truncada como estado completo.

### E03 — Identidade, autorização e contexto externo

**Situação:** o servidor exige modo local; o handler produz Principal/ActorRef a partir do resolver fixo. O `ActorRef` local é do tipo humano. A autorização privilegiada de algumas operações existe, mas não equivale a um sistema completo de usuários, credenciais e grants por Namespace.

- [ ] E03.01 Introduzir resolução de credencial por requisição, independente do DTO de domínio.
- [ ] E03.02 Preservar a distinção entre Principal autenticado e ActorRef que representa a autoria.
- [ ] E03.03 Suportar humanos, contas de serviço e agentes com identidade rastreável.
- [ ] E03.04 Definir autenticação de navegador por sessão ou integração com provedor de identidade, sem impor que WOS implemente um IdP próprio.
- [ ] E03.05 Implementar credenciais de integração com emissão, expiração, rotação e revogação.
- [ ] E03.06 Implementar gestão/listagem de Namespaces e grants, com criação e administração autorizadas.
- [ ] E03.07 Autorizar todas as leituras e mutações, incluindo listas, buscas, históricos, snapshots e recursos MCP.
- [ ] E03.08 Revalidar acesso antes de devolver resposta armazenada de replay idempotente.
- [ ] E03.09 Implementar autoria delegada somente quando a credencial estiver autorizada a representar aquele ator.
- [ ] E03.10 Definir permissões de planejamento, execução, avaliação, waiver, conclusão, arquivamento e administração.
- [ ] E03.11 Permitir política de revisor independente quando um deployment exigir separação entre execução e aprovação.
- [ ] E03.12 Definir o escopo de contexto externo confiável e a política de sua indexação e exposição.
- [ ] E03.13 Não aceitar `tenant_id`, `user_id` ou Namespace enviados pelo modelo como concessão de acesso.
- [ ] E03.14 Separar correlação de execução/sessão de autorização e propriedade do Outcome.
- [ ] E03.15 Disponibilizar referências externas para localizar Outcomes sem adivinhar IDs internos.
- [ ] E03.16 Validar revogação e troca de permissões durante uso por clientes long-lived.
- [ ] E03.17 Aplicar proteção de navegador, política de origem e tratamento de credenciais conforme o mecanismo de autenticação escolhido.
- [ ] E03.18 Testar isolamento entre dois Namespaces com dados semelhantes e atores delegados distintos.

**Aceite:** humanos e agentes podem atuar no mesmo Outcome com autoria correta e permissões próprias; nenhum metadado enviado pelo consumidor contorna a autorização.

### E04 — MCP operacional e adequado a agentes

**Situação:** a árvore de API inspecionada não contém implementação MCP. A Wave 13 continua prevista. A implementação deve aproveitar o SDK oficial e as regras já existentes, não replicar o domínio em handlers.

- [ ] E04.01 Fixar SDK e perfis de protocolo suportados, incluindo a matriz de compatibilidade com os clientes-alvo.
- [ ] E04.02 Implementar transporte stdio para uso local e Streamable HTTP para uso remoto conforme o perfil escolhido.
- [ ] E04.03 Implementar descoberta/capacidades, negociação ou lifecycle correspondente e encerramento correto de cada perfil.
- [ ] E04.04 Registrar ferramentas de descoberta de Outcomes, snapshot, contexto focal, grafo, timeline e candidatos de trabalho.
- [ ] E04.05 Expor criação, edição e transições de Outcome, Objective e WorkItem.
- [ ] E04.06 Expor drafts, publicação, revisões e ativação de Roadmaps.
- [ ] E04.07 Expor claim, renovação, liberação, reclaim e conclusão preservando o fencing token.
- [ ] E04.08 Expor Issues, Blockers, Decisions, Artifacts e Evidence.
- [ ] E04.09 Expor critérios, avaliações e conclusões com autoria e permissões corretas.
- [ ] E04.10 Definir schemas explícitos, erros estruturados, campos obrigatórios e ações de recuperação de conflitos.
- [ ] E04.11 Preservar idempotency key, versões esperadas, identificação de claim e correlação entre tentativas.
- [ ] E04.12 Limitar payload e detalhamento das ferramentas de leitura; permitir expansão por cursores ou referências.
- [ ] E04.13 Não apresentar candidatos ordenados como autorização automática para executar trabalho.
- [ ] E04.14 Tratar desconexão/timeout sem assumir que uma mutação não foi persistida.
- [ ] E04.15 Garantir que estado de protocolo não seja o local exclusivo de estado operacional.
- [ ] E04.16 Aplicar autenticação e escopo confiáveis também ao modo remoto MCP.
- [ ] E04.17 Executar a mesma suíte de comandos por HTTP e MCP, comparando resultado e invariantes.
- [ ] E04.18 Demonstrar um cliente MCP real independente da Woobe.
- [ ] E04.19 Demonstrar uma integração Woobe → WOS por MCP sem tornar a Woobe requisito do servidor.

**Aceite:** um cliente MCP real descobre, interpreta, assume, atualiza e retoma trabalho por chamadas ao mesmo Application Service utilizado pela API.

### E05 — Experiência humana sobre o mesmo sistema

**Situação:** não foi localizado cliente visual no repositório. Um teste de domínio com ator humano comprova neutralidade do Core, mas não entrega a experiência de um usuário final.

**Decisão proposta:** manter o Core independente e acrescentar um cliente web mínimo, ou outra interface humana oficial de referência. A implementação visual não deve conter uma segunda máquina de estados ou regras paralelas de conclusão.

- [ ] E05.01 Entregar autenticação, escolha de Namespace e navegação entre Outcomes autorizados.
- [ ] E05.02 Criar Outcome com estado desejado, contexto e critérios claros, sem exigir que o usuário conheça IDs internos.
- [ ] E05.03 Exibir portfólio de Outcomes com filtros e estados vazios úteis.
- [ ] E05.04 Disponibilizar visão do Outcome, progresso comprovado, plano ativo, participantes e pendências.
- [ ] E05.05 Permitir decomposição e manutenção de Objectives e WorkItems com relações explícitas.
- [ ] E05.06 Oferecer lista/quadro de trabalho usando estados derivados fornecidos pelo servidor.
- [ ] E05.07 Traduzir ações visuais em comandos válidos; não gravar `ready` ou `blocked` como lifecycle por arrastar um cartão.
- [ ] E05.08 Permitir assumir, liberar e retomar trabalho com informação de lease e de outro participante.
- [ ] E05.09 Oferecer edição de draft, visualização de alterações, publicação e ativação de Roadmap.
- [ ] E05.10 Exibir revisões históricas separadas do estado vivo das referências.
- [ ] E05.11 Exibir Issues e Blockers com alvo, causa, herança e confirmação de liberação.
- [ ] E05.12 Permitir registrar referências de Artifact e Evidence, preservando o contrato reference-only do MVP.
- [ ] E05.13 Entregar avaliação explícita de critérios com resultado, evidência e justificativa.
- [ ] E05.14 Criar uma visão de pendências de revisão/decisão sem inventar, necessariamente, um novo agregado Inbox.
- [ ] E05.15 Exibir motivos de impedimento de conclusão e contestações posteriores.
- [ ] E05.16 Mostrar histórico de autoria, decisões e replanejamentos em linguagem compreensível.
- [ ] E05.17 Tratar conflito de versão com recarga/comparação, sem sobrescrita silenciosa.
- [ ] E05.18 Tratar operações em andamento, timeout de resultado incerto, erros, indisponibilidade e expiração de acesso.
- [ ] E05.19 Entregar fluxos de arquivar, consultar histórico e retomar Outcome.
- [ ] E05.20 Testar teclado, acessibilidade, responsividade e compreensão dos estados centrais.
- [ ] E05.21 Demonstrar interação alternada entre a interface humana e um agente MCP no mesmo Outcome.

**Aceite:** uma pessoa sem escrever HTTP manualmente consegue operar a jornada principal e compreender exatamente o que os agentes fizeram ou ainda precisam fazer.

### E06 — Fechar semântica de comprovação e colaboração

**Situação:** avaliação imutável, revisões de critérios, Conclusions e contestações por avaliação posterior/evidência retraída já têm implementação e testes. O fechamento deve concentrar-se na matriz completa de invariantes entre entidades e políticas de uso.

- [ ] E06.01 Confirmar a equivalência dos modos attestation, evidence review e external evaluation em todos os transports e bancos.
- [ ] E06.02 Preservar identificação/revisão do avaliador externo e procedência da Evidence quando aplicável.
- [ ] E06.03 Testar que Evidence anexada não conclui critério, WorkItem, Objective ou Outcome implicitamente.
- [ ] E06.04 Testar que uma revisão nova de critério não reaproveita avaliação antiga como comprovação atual.
- [ ] E06.05 Aplicar autorização e razão obrigatória a waiver e caminhos administrativos.
- [ ] E06.06 Decidir e testar o que acontece quando um Objective obrigatório é reaberto depois de um Outcome alcançado.
- [ ] E06.07 Decidir e testar alterações posteriores nas obrigações estruturais que sustentaram uma Conclusion.
- [ ] E06.08 Ampliar a projeção de contestação se o contrato exigir refletir perda de validade dessas obrigações, além de Evidence e assessments.
- [ ] E06.09 Preservar a Conclusion histórica e nunca reabrir lifecycle silenciosamente por uma nova observação.
- [ ] E06.10 Definir como o usuário diferencia “resultado declarado alcançado” de “conclusão atualmente contestada”.
- [ ] E06.11 Validar retirada/retração documental depois de conclusões sem apagamento de histórico.
- [ ] E06.12 Provar que dois atores que compartilham credencial só são distinguíveis até o nível de delegação realmente autorizado, sem prometer identidade que não foi autenticada.
- [ ] E06.13 Definir a experiência de lease para humano de longa duração e agente de curta duração sem duplicar o contrato de execução.
- [ ] E06.14 Testar que executor com lease vencido e fencing antigo não pode finalizar após reclaim por outro participante.

**Observação de auditoria:** a função de projeção de contestações inspecionada considera avaliações contraditórias e Evidence retraída, mas não percorre as obrigações estruturais da Conclusion. Isso identifica um cenário a especificar e testar; não demonstra, isoladamente, que toda a lógica de conclusão esteja errada.

**Aceite:** o WOS distingue trabalho realizado, prova apresentada, avaliação autorizada e conclusão do resultado, inclusive diante de fatos posteriores.

### E07 — PostgreSQL e paridade de persistência

**Situação:** há adapters de memória e SQLite. Não foi encontrada implementação PostgreSQL, e `OpenRuntime` rejeita outro driver.

- [ ] E07.01 Implementar migrations PostgreSQL para entidades operacionais, documentais, planejamento, histórico, grants e integração.
- [ ] E07.02 Implementar repositories e UnitOfWork com o mesmo contrato funcional.
- [ ] E07.03 Implementar coordenação por Outcome e regras explícitas de isolamento transacional.
- [ ] E07.04 Garantir que snapshots e `outcome_revision` correspondam a uma leitura coerente.
- [ ] E07.05 Implementar QueryStore/read models e índices para filtros, busca, grafo e timeline.
- [ ] E07.06 Executar a suíte compartilhada de contratos em SQLite e PostgreSQL reais.
- [ ] E07.07 Testar corrida de dependências opostas, ciclos indiretos e alterações concorrentes de hierarquia.
- [ ] E07.08 Testar claims, renovações, reclaim e fencing com conexões e processos distintos.
- [ ] E07.09 Testar unicidade de slots, supersessão de Decision e conclusão/avaliação concorrentes.
- [ ] E07.10 Testar idempotência sob concorrência, erro de commit, rollback e reconexão.
- [ ] E07.11 Configurar pool, deadlines, lock timeout e tratamento controlado de deadlocks/erros transitórios.
- [ ] E07.12 Integrar seleção do driver ao runtime e à configuração, sem alterar semântica de domínio.
- [ ] E07.13 Testar upgrade de schema e restauração em versões suportadas dos dois bancos.
- [ ] E07.14 Publicar limites e resultados de benchmark separados para SQLite e PostgreSQL.

**Aceite:** trocar o banco não altera regras de domínio, histórico, escopo, resultado de comandos ou garantias de coordenação prometidas.

### E08 — Domain Events, Triggers e entrega durável

**Situação:** o pipeline persiste mutações, Domain Events e resultados idempotentes transacionalmente. Isso é uma base útil, não o fluxo completo de Integration Events, regras e entregas externas.

- [ ] E08.01 Definir um catálogo público e versionado de eventos de integração.
- [ ] E08.02 Separar esse catálogo de detalhes internos do schema e nomes de structs/comandos em Go.
- [ ] E08.03 Produzir Integration Events/outbox na mesma transação do fato que os originou.
- [ ] E08.04 Implementar Triggers declarativos com filtros limitados e determinísticos.
- [ ] E08.05 Persistir TriggerFiring com identidade e deduplicação estáveis.
- [ ] E08.06 Implementar destinos e estado de entrega, tentativas, sucesso, falha e esgotamento.
- [ ] E08.07 Implementar workers com leases para evitar posse concorrente indevida da mesma entrega.
- [ ] E08.08 Aplicar retry com backoff, limites, redelivery e observação de pendências.
- [ ] E08.09 Assinar webhooks e gerir credenciais/segredos de destino.
- [ ] E08.10 Aplicar política de endereços de destino, redirecionamento, tamanho e timeout.
- [ ] E08.11 Provar persistência de pendências depois de crash/restart.
- [ ] E08.12 Expor consulta operacional de entregas e redelivery autorizado.
- [ ] E08.13 Documentar entrega pelo menos uma vez e idempotência necessária no consumidor.
- [ ] E08.14 Testar entrega duplicada, destino indisponível, assinatura inválida e retomada de worker.
- [ ] E08.15 Garantir que nenhum Trigger execute LLM, WorkItem ou Tool por conta própria.

**Aceite:** o consumidor recebe um sinal durável e identificável; ele decide se inicia qualquer execução. O WOS preserva a fronteira entre notificar e executar.

### E09 — Contratos de integração e clientes

- [ ] E09.01 Fechar OpenAPI, DTOs e erros para todas as funcionalidades incluídas na versão.
- [ ] E09.02 Definir estabilidade/versionamento de comandos, eventos, ETags e resultados de idempotência.
- [ ] E09.03 Criar cliente Go fino, sem regras de domínio duplicadas.
- [ ] E09.04 Expor auth, timeout, paginação, erros tipados, versões e claim tokens no cliente.
- [ ] E09.05 Preservar a mesma idempotency key nos retries da mesma intenção.
- [ ] E09.06 Oferecer reconciliação de resultado incerto sem criar uma segunda operação.
- [ ] E09.07 Entregar cliente TypeScript/contrato gerado se necessário ao frontend; não bloquear o MVP por SDKs em todas as linguagens.
- [ ] E09.08 Validar paridade entre OpenAPI e comportamento HTTP, não apenas sintaxe YAML.
- [ ] E09.09 Manter exemplos executáveis de embedded Go, standalone HTTP, MCP independente e Woobe.
- [ ] E09.10 Demonstrar produto consumindo WOS diretamente e agentes consumindo por MCP.
- [ ] E09.11 Documentar que IDs de Run/Session servem para correlação, não para possuir o estado do Outcome.
- [ ] E09.12 Documentar integração de referência com produto real sem mover regras específicas desse produto para o Core.

**Aceite:** um terceiro integra e continua um Outcome a partir de documentação e exemplos publicados, sem depender de decisões implícitas das conversas de desenvolvimento.

### E10 — Distribuição, operação e recuperação

**Situação:** binário, configuração, SQLite, health checks e shutdown já existem. Falta completar e provar a distribuição e os modos adicionados.

- [ ] E10.01 Entregar builds reproduzíveis e matriz de plataformas oficialmente suportadas.
- [ ] E10.02 Acrescentar Dockerfile e Compose de referência quando esse for o modo de distribuição declarado.
- [ ] E10.03 Completar comandos de servidor, configuração e migrations; a CLI administrativa não precisa virar outro produto de gerenciamento completo para esta versão.
- [ ] E10.04 Ampliar readiness para schema compatível, storage escolhido e componentes necessários do runtime.
- [ ] E10.05 Testar desligamento gracioso com requisições e entregas em andamento.
- [ ] E10.06 Implementar e documentar backup consistente dos bancos e os requisitos do ambiente.
- [ ] E10.07 Restaurar em instalação limpa e verificar referências, histórico, grants, idempotência e entregas pendentes.
- [ ] E10.08 Definir retomada de leases e timestamps depois da restauração.
- [ ] E10.09 Documentar upgrade, compatibilidade, recuperação e rollback operacional suportado.
- [ ] E10.10 Acrescentar logs estruturados de comando/query, ator/principal, correlação, revisão, duração e resultado.
- [ ] E10.11 Acrescentar métricas de consultas, conflito, idempotência, transação/guard, leases e entregas.
- [ ] E10.12 Instrumentar transport, application, storage e integração sem importar observabilidade no Domain.
- [ ] E10.13 Evitar conteúdo documental e credenciais integrais nos logs por padrão.
- [ ] E10.14 Documentar limites de tamanho, retenção e manutenção compatíveis com a continuidade prometida.
- [ ] E10.15 Entregar instalação inicial e exemplo independente da Woobe testados em ambiente limpo.

**Aceite:** alguém consegue instalar, operar, atualizar, reiniciar e restaurar o WOS sem perder o estado necessário à continuação.

### E11 — Validação final e release

- [ ] E11.01 Manter os testes existentes e ampliar CI para as capacidades novas.
- [ ] E11.02 Executar matriz funcional HTTP/MCP sobre SQLite/PostgreSQL.
- [ ] E11.03 Executar testes de interface humana para criação, execução, revisão, replanejamento e retomada.
- [ ] E11.04 Acrescentar smoke de processo que percorra uma jornada de domínio, além de health checks.
- [ ] E11.05 Testar resposta perdida após commit, timeout antes de commit e retries equivalentes.
- [ ] E11.06 Injetar falhas entre mutação, eventos, outbox, idempotência e commit para provar atomicidade.
- [ ] E11.07 Testar concorrência com banco real, não apenas goroutines usando adapter em memória.
- [ ] E11.08 Aplicar fuzz/property tests a grafo, referências, cursores e serialização de contratos.
- [ ] E11.09 Executar migrations partindo de snapshots de versões anteriores.
- [ ] E11.10 Provar restauração e continuação por consumidor novo.
- [ ] E11.11 Medir latência, tamanho de resposta, queries e throughput em ambiente documentado.
- [ ] E11.12 Publicar benchmarks como medições, distinguindo-os das metas propostas no design.
- [ ] E11.13 Revisar licença, changelog, contratos, documentação e artefatos de distribuição.
- [ ] E11.14 Executar a demonstração integrada descrita na seção seguinte.
- [ ] E11.15 Associar cada critério final ao teste, evidência ou demonstração que o satisfaz.

**Aceite:** nenhuma capacidade exigida depende apenas de uma promessa do README; o conjunto de evidências demonstra o produto realmente operando.

## 5. Demonstração obrigatória de ponta a ponta

Usar o mesmo Outcome durante todo o cenário, com identidades e clientes distintos. Executar ao menos uma vez sem Woobe e outra integração de referência com Woobe por MCP.

1. Um humano autentica e cria Outcome, estado desejado e critérios verificáveis.
2. Ele cria Objectives/WorkItems e publica um Roadmap inicial.
3. Um agente descobre o Outcome por interface autorizada e obtém contexto sem receber o histórico de chat.
4. O agente consulta candidatos prontos e assume explicitamente um WorkItem.
5. O humano registra uma Decision e um Issue com Blocker; ambos aparecem na continuidade consultada pelo agente.
6. O Issue é resolvido, mas o WorkItem continua bloqueado enquanto o Blocker não for liberado explicitamente.
7. Uma execução perde a sessão e o lease expira; outro agente faz reclaim.
8. O executor antigo tenta concluir usando o fencing antigo e é rejeitado.
9. O novo executor registra Artifact/Evidence e conclui o trabalho permitido pelo contrato.
10. Um humano ou avaliador autorizado avalia os critérios; aprovação segue a política configurada.
11. Objectives e Outcome são concluídos explicitamente, com snapshots de comprovação preservados.
12. O plano é revisado em uma situação aplicável e a revisão histórica permanece legível e imutável.
13. Uma evidência relevante é retraída; a contestação aparece sem apagar a conclusão histórica nem reabrir silenciosamente o lifecycle.
14. O servidor reinicia; um terceiro cliente recupera estado, referências e histórico.
15. Um ciclo de backup/restauração em instalação limpa mantém a possibilidade de continuação.
16. Um sinal externo é entregue novamente; sua identidade permite ao consumidor deduplicar o processamento.
17. O Outcome é arquivado, continua consultável conforme o contrato e pode ser desarquivado para retomada autorizada.
18. Outro Namespace tenta acessar o cenário por ID, busca, cursor e recurso MCP e não obtém acesso.

Não se deve fabricar evidências ou respostas do agente para demonstrar essa jornada. O teste pode usar atores determinísticos para validar contratos; a demonstração de interoperabilidade deve usar clientes reais.

## 6. Sequência de execução recomendada

| Etapa | Entregas | Marco de saída |
|---|---|---|
| 1. Base confiável | E00 e fechamento de E01 | Roadmap integrado e documentação coerente. |
| 2. Estado utilizável e identidade | E02, contratos centrais de E03 e E06 | Continuidade completa, com escopo e autoria autorizados. |
| 3. Operação humano + agente | E04, E05 e primeira fatia de E09 | Jornada mista funcional sobre SQLite, ainda sem chamar a release completa de pronta. |
| 4. Standalone completo | E07, E08 e E10 | PostgreSQL, sinais, distribuição e recuperação atendem o escopo prometido. |
| 5. Fechamento da versão | E11 e conclusão de E09 | Matriz de aceite, documentação e release verificadas. |

A autenticação não deve ser deixada para depois de expor MCP remoto e interface multiusuário. Seus contratos precisam anteceder essas integrações. PostgreSQL e Triggers podem evoluir em paralelo quando os contratos e transações estiverem estabilizados.

## 7. O que não precisa entrar para concluir esse objetivo

Não são requisitos intrínsecos: planner autônomo; inferência de LLM dentro do WOS; executor próprio de Tools; regras pedagógicas da Lipo; runtime de agentes; banco vetorial; Redis obrigatório; barramento distribuído obrigatório; armazenamento de bytes de todos os Artifacts; um Jira completo; dezenas de SDKs; aplicações móveis nativas; editor visual extremamente sofisticado; cobrança/assinaturas; toda a plataforma de hospedagem comercial.

Essas extensões não devem substituir nem adiar as capacidades centrais prometidas. Triggers, MCP e PostgreSQL continuam parte do escopo técnico descrito no roadmap; um primeiro demo SQLite não os torna desnecessários para a versão completa planejada.

## 8. Fontes e rastreabilidade

As referências abaixo identificam o material efetivamente consultado. Caminhos são relativos a `A1b3rt0M3rcad0/wos`.

| Ref. | Fonte | Revisão/observação |
|---|---|---|
| S01 | Commit de `master` e PR #12 | `2f9967689fa954b7c371781bc7a00c1ae760825d` |
| S02 | PR #13 e arquivos alterados | Head `0bd7c538dfdc4eb4a0732f60c3ec57bd341da345`; aberto |
| S03 | CI #206 | Run `37064544568`; sucesso |
| S04 | CI #228 | Run `37340922999`, job `111867478636`; sucesso |
| S05 | `README.md` | Master; status textual desatualizado |
| S06 | `ROADMAP.md` | Master e head do PR #13 |
| S07 | `packages/wos-api/http/handler.go` | Master; rotas e identidade local do handler |
| S08 | `packages/wos-api/internal/server/runtime.go` | Master; SQLite/local obrigatórios, health/shutdown existentes |
| S09 | `packages/wos-api/authentication/local/local.go` | Master; Principal fixo, ActorKindHuman, privilégios locais restritos |
| S10 | `packages/wos-core/application/query.go` | Master; snapshot, históricos e contestações |
| S11 | `packages/wos-core/application/pipeline.go` | Master; transação, eventos e idempotência |
| S12 | `packages/wos-core/application/wave10_concurrency_contestation_test.go` | Master; avaliações concorrentes e Evidence retraída |
| S13 | `packages/wos-core/application/planning.go` | Head do PR #13; comandos, referências, publicação e hash |
| S14 | `packages/wos-core/application/wave11_completion_test.go` | Head do PR #13; concorrência e preservação histórica |
| S15 | `packages/wos-core/storage/sqlite/wave11_roadmap_restart_test.go` | Head do PR #13; persistência e alteração de entidades vivas |
| S16 | `.github/workflows/ci.yml` | Pipeline atual e escopo do smoke |
| S17 | Árvores de `packages/wos-api` e `packages/wos-core/storage` | Implementações presentes/ausentes na base auditada |
| S18 | `WOS_Design_Arquitetura_Planejamento_Atualizado.md` | Documento de arquitetura, revisão 2, 1/10/2026; proposta, não prova de implementação |
| S19 | Documentação oficial do MCP Go SDK | Consulta pública em 5/10/2026; suporte a clientes/servidores, stdio/HTTP e perfis de protocolo |

**Confiança:** alta sobre os fatos de branch, código consultado e CI; moderada sobre completude global de invariantes não cobertos por esta inspeção dirigida. Itens propostos de produto e política são recomendações de implementação/aceite, não alegações de defeitos reproduzidos.

**Veredito final:** o WOS já possui o núcleo de um sistema de Outcomes. Para finalizar a missão do produto, precisa entregar continuidade completa, operação padronizada por agentes, identidade/autorização reais, uma experiência humana utilizável e a prova operacional de que tudo permanece consistente entre atores, sessões, processos e reinícios.


## Evidência de execução — 2026-10-05

Base auditada: `master` 2f9967689fa954b7c371781bc7a00c1ae760825d (Waves 01–10), PR #13 `feat/wave-11-roadmap-planning-history` em 0bd7c538dfdc4eb4a0732f60c3ec57bd341da345; CI anterior: run 37340922999, sucesso. Esta implementação parte do head de #13.

A licença escolhida pelo mantenedor é Apache 2.0, registrada em LICENSE. A primeira entrega humana inclui cliente web oficial. Nenhum requisito do plano foi removido.

| Frente | Evidência atual | Estado de aceite |
| --- | --- | --- |
| E01 | testes existentes de planning, concorrência e restart; `TestWave11UpgradeExistingVersionSevenDatabase` | Revisão local realizada; integração ainda pendente |
| E02 | `continuity_contract_test.go`: descoberta, cursores, omissões, timeline, invalidação por revisão; Application graph/context | Parcial: medição e otimização de scans pendentes |
| E03 | `security_contract_test.go`: token vinculado ao Namespace, grants, revogação, replay autorizado antes do idempotency lookup | Parcial: sessões, contexto externo e histórico administrativo pendentes |
| E04 | `TestRealMCPClientSharesDurableStateWithHTTP`: cliente oficial real, descoberta, mutação, replay e HTTP; 71 comandos gerados | Parcial: integração Woobe e matriz completa pendentes |
| E06 | contestações estruturais por alteração de obrigações e Objective obrigatório | Teste específico adicional pendente |
| E07 | adaptador pgx, migrações e contratos compartilhados gerados; runtime seleciona PostgreSQL | Compila; só aceite após testes reais do job PostgreSQL |

Validação local: `go test ./...` passou com Go 1.27.1. Testes PostgreSQL foram explicitamente ignorados localmente por ausência de servidor; o CI fornece PostgreSQL 18 e não pode ignorá-los. Este documento não declara Release 0.1 concluída.
