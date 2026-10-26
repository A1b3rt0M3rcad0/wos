# WOS — Plano completo de implementação do Outcome Board

**Destino:** PR #116, branch `fix/workspace-operational-density`.

**Baseline auditada:** `5d945d3081f85927b80c686d695849bd1df73a5d`.

**Conteúdo:** especificação funcional, contratos de backend, backlog com 24 unidades, 108 cenários de teste, instruções de execução e fontes. Todos os incrementos permanecem planejados; o código do repositório não foi alterado por este pacote.

**Ordem de implementação:** baseline → regras e escrita → leituras coerentes → inspector e operações → mapa e integração → qualificação.


---

## Parte 1 — WOS — Outcome Board compreensível e operacional


### Plano de implementação para o PR #116

**Repositório:** `A1b3rt0M3rcad0/wos`  
**Branch de destino:** `fix/workspace-operational-density`  
**Revisão auditada:** `5d945d3081f85927b80c686d695849bd1df73a5d`  
**Base observada:** `master`, `8f6ff6313d874c089c06d8e719901024a16ffe76`  
**Natureza deste documento:** especificação proposta para implementação; não é um relatório de funcionalidades já entregues.  
**Modo de entrega:** continuar o PR existente, com commits incrementais e verificáveis, sem merge ou publicação automática.

### 1. Resultado que esta implementação deve entregar

Um humano deve conseguir montar o mapa de um Outcome sem estudar a ontologia ou o catálogo de comandos do WOS. Ao abrir qualquer item, deve entender seu papel, suas relações, seu estado e suas pendências. Ao abrir uma Task, deve encontrar explicitamente: o que fazer, critérios, evidências, quem responde pelo trabalho, quem detém a execução atual, o que foi registrado ou entregue e o que falta para avançar.

O produto deve oferecer dois níveis complementares: **mapa de trabalho** para compreender a estrutura e **inspector operacional** para entender e atuar sobre um item. O mesmo inspector será usado no canvas e nas outras views. Não criar um segundo aplicativo ou um segundo domínio de Tasks para o dashboard.

A simplicidade será obtida por contexto preenchido, criação progressiva, agrupamento correto, operações compostas e leitura orientada a decisões. Não será obtida removendo verificação, falsificando autoria ou tratando uma evidência como aprovação automática.

#### Critérios centrais da entrega

- Construir um Outcome, dois objetivos e quatro Tasks sem abrir um console técnico, sem inserir IDs e sem procurar novamente o pai de um item criado dentro dele.
- Distinguir pertencimento, dependência, evidência, avaliação e bloqueio pela própria interface.
- Encontrar os cinco grupos de informação exigidos pelo usuário dentro da Task, com estados vazios e dados incompletos identificados corretamente.
- Registrar evidência, atualização documental e problema no contexto selecionado, sem navegar primeiro para coleções globais.
- Saber por que uma operação está indisponível e qual ação resolve a condição faltante.
- Preservar a autoridade de execução, a revisão independente e o significado de cada conclusão.

### 2. Escopo e limites definidos

Esta entrega inclui frontend, application/domain quando necessário, API, persistência das novas notas documentais, projeções de leitura, comandos de reorganização, compatibilidade, testes e documentação operacional. Atualizações do CLI, MCP e inventário de comandos serão limitadas ao necessário para expor ou documentar as novas capacidades sem duplicar regras.

Permanecem fora desta implementação: executar agentes pelo WOS; um scheduler de execução; integração proprietária com serviços externos; criação automática de planos por LLM; serviço de armazenamento de arquivos; inferência automática de comprovação; editor genérico de workflows; nova framework frontend; conversão silenciosa do protocolo de execução de um namespace.

**Execução assinada:** o dashboard continuará respeitando o protocolo vigente. Onde a operação exige um perfil autorizado no host, a interface oferecerá um encaminhamento claro, contextual e preciso. Não armazenará chaves privadas nem assinará em nome de um agente. Este plano não promete remover o CLI do fluxo `signed_contracts_v2`. Planejamento, registro documental, comprovação consultável e entendimento do trabalho não precisam depender dessa limitação. Um futuro mecanismo de assinatura humana nativa exige um desenho próprio e não deve ser improvisado como atalho nesta branch.

Preservar a identidade visual: papéis/post-its, robôs, cores, estilo editorial, traços e fontes existentes. Não copiar a aparência de Trello, Linear ou outro produto. Manter o inglês como idioma da interface na baseline atual; os textos em português deste plano explicam o comportamento e não autorizam uma migração parcial de idioma.

### 3. Correções de premissas consolidadas pela revisão

#### 3.1 Campos expostos não são necessariamente campos obrigatórios

Hoje os formulários de Objective e Task já permitem omitir parte dos campos. A dificuldade não é só validação excessiva: é exigir decisões sobre pai, estado, prioridade, critérios e autoridade fora do momento em que fazem sentido. A criação rápida deve reduzir decisões e mudanças de contexto, não apenas retirar a marca de obrigatório.

#### 3.2 Criar um item não é disponibilizá-lo para um agente

Uma Task com apenas título pode ser um rascunho de planejamento útil, mas isso não deve colocar trabalho incompleto na fila de aquisição. A criação rápida deste plano salva novas Tasks em `backlog`. A ação explícita de disponibilizar passa para `todo`, observando as regras atuais. O backend e clientes existentes preservam seus contratos de criação; a mudança de default se aplica ao novo fluxo humano e aparece visivelmente nele.

#### 3.3 Campos adiados exigem edição posterior real

A implementação precisa permitir editar instruções, restrições, entregáveis, contexto de execução, agendamento e associação quando o domínio permitir. O `UpdateWorkItemCommand` auditado só cobre título, descrição e prioridade. Não publicar um formulário mínimo que depois impeça completar a tarefa.

#### 3.4 Nota humana não é atividade sob contrato assinado

`AppendWorkActivity` exige contrato, chave, credencial, versão e autoridade corrente. Esses requisitos não serão removidos para encaixar um formulário humano. Introduzir uma nota documental pequena e tipada, separada de atividade contratada, porém apresentada no mesmo histórico de leitura. Isso corrige a proposta anterior excessivamente ampla de que nenhuma nova estrutura persistida seria necessária.

#### 3.5 Inspector universal não significa campos idênticos

Evidence não terá um executor fictício; Issue não terá critérios fictícios; Criterion continua pertencendo a um Outcome, Objective ou Task. A estrutura visual é comum, mas cada entidade responde às perguntas pertinentes à sua natureza.

#### 3.6 Um status não prova execução física

Um contrato vigente informa autoridade registrada. Não prova que um processo externo está neste instante usando CPU ou produzindo resultado. Mostrar responsável declarado, executor com autoridade atual, último executor e última atividade como informações distintas. Não fabricar presença online.

#### 3.7 Nem todo destino mostrado hoje é válido

O formulário de EvidenceLink oferece Blocker, mas a validação de backend auditada aceita Outcome, Objective, WorkItem, Issue e Decision. Corrigir o seletor e publicar elegibilidade tipada comum. Evidência relativa a um impedimento deve ser vinculada à Issue causal ou ao trabalho afetado; não ampliar os targets de EvidenceLink apenas para acomodar um erro da UI.

### 4. Modelo mental e hierarquia da informação

A estrutura principal será **Outcome → Objectives → Tasks**. Objectives podem conter subobjetivos. Tasks diretamente no Outcome continuam válidas para trabalho pequeno ou transversal. Não obrigar a criar objetivos vazios só para acomodar uma tarefa.

| Camada | Conteúdo principal | Apresentação padrão |
| --- | --- | --- |
| Estrutura | Outcome, Objectives, Tasks | Cards e relações de pertencimento |
| Operação | Estado, participantes, pré-requisitos, bloqueios, entregas e revisão | Indicadores do card e inspector |
| Comprovação e registros | Critérios, avaliações, evidências, artefatos, decisões e notas | Seções contextuais, com exploração das relações |

Critérios aparecem dentro de seus proprietários. A opção de explorar comprovação pode expandir uma representação visual de um critério, mas essa representação usa `owner_ref + criterion_id` e não cria uma nova entidade operacional global.

Relações estruturais organizam o mapa. Dependências dizem o que precisa ocorrer antes. EvidenceLinks dizem como uma observação se relaciona com uma afirmação. Assessments registram uma decisão. Blockers registram impedimentos. Não usar um único comando “Connect” como entrada ambígua para todos esses significados.

#### Obrigações do Outcome

No novo fluxo humano, criar um Objective pré-seleciona `required_for_outcome=true`, com o chip visível **Required for Outcome** e opção imediata de torná-lo opcional. A regra vale também para subobjetivos: o rótulo continua apontando para o Outcome, não para o pai. Não inventar herança de obrigatoriedade ou `required_for_parent`.

Esse é um default da experiência de criação, enviado explicitamente no comando. Não alterar objetivos existentes nem o default de clientes que omitem o campo. Na revisão para iniciar/concluir o Outcome, apresentar a lista exata de objetivos obrigatórios, inclusive os que não estão visíveis no ramo atual.

### 5. Nova experiência de criação e edição

#### 5.1 Criar Outcome

Apresentar dois campos: nome curto e resultado esperado. Abaixo, “Condições de sucesso”, com linhas opcionais e ação de adicionar. Descrição extensa, prioridade e responsáveis ficam acessíveis por expansão, sem ocupar a primeira tela.

Salvar como `draft`. Quando houver critérios preenchidos, salvar Outcome e critérios em uma única operação transacional. Não converter o texto de `desired_state` em critério automaticamente. Não gerar um critério genérico “Outcome concluído”. O usuário pode preparar o mapa antes de definir a comprovação, mas não pode ativar sem o critério obrigatório exigido.

O botão padrão é **Create Outcome**. A ativação é uma decisão posterior, em uma revisão curta de preparação. Essa separação evita a ambiguidade de “Salvar e começar” sem visualizar os gates.

#### 5.2 Criar Objective no mapa

Ao clicar em adicionar no Outcome ou Objective, abrir um editor inline ou popover junto à origem. Mostrar somente o título, o contexto de pertencimento já preenchido e o chip de obrigatoriedade. O pai não aparece como um campo de pesquisa em branco.

Descrição, critérios e prioridade ficam disponíveis por expansão. Salvar sem critérios é permitido como planejamento; o card indica “Definir condição de sucesso” enquanto faltar requisito para iniciar. Enter confirma uma linha; Escape fecha sem salvar; um erro mantém o texto e a localização.

#### 5.3 Criar Task no mapa

Título é o único texto obrigatório na criação rápida. O contexto mostra o Objective ou “Diretamente neste Outcome”. O item nasce em `backlog`, apresentado como planejado/não disponibilizado. O formulário não exige prioridade, data, responsável ou estado inicial para registrar a ideia.

Após criar, oferecer uma continuação discreta: “Adicionar detalhes” ou “Disponibilizar”. Não abrir compulsoriamente outro modal. Disponibilizar é explícito; não significa adquirir a execução. Uma Task simples pode ser disponibilizada sem critérios formais se isso for admitido pelo protocolo e sua política. “Melhorar a descrição” é uma sugestão editorial, não um gate de domínio inventado. Instruções vazias não bloqueiam por si só uma operação que o contrato real permite. Como atalho explícito, o modo completo pode oferecer “Criar como disponível”, enviando `todo`; isso não altera o default seguro da criação rápida nem adquire autoridade.

#### 5.4 Critérios em linhas compreensíveis

Criar o critério no próprio item com a pergunta “O que precisa ser verdadeiro para considerar este item atendido?”. O campo principal é uma condição verificável. `required=true` é visível. O novo fluxo utiliza `evidence_review` como sugestão visível, editável antes de salvar; não altera critérios já existentes nem os defaults de contratos públicos anteriores.

Oferecer os três modos reais com linguagem compreensível: revisão de evidências, declaração autorizada e avaliação externa. O modo externo pede a configuração pertinente; não é obrigatório conhecer esse modo para cadastrar um critério normal. Não degradar uma política mais exigente para declaração apenas para viabilizar um clique de conclusão.

Uma linha de critério não funciona como checkbox livre de aprovação. Clicar abre suas evidências, revisão e avaliação. “Atendido” exige a operação autorizada correspondente; “dispensado” permanece visualmente distinto.

#### 5.5 Completar a especificação depois da criação

Adicionar seções editáveis de contexto/descrição, instruções, restrições, entregáveis, referências e agendamento. Cada seção tem salvar/cancelar, versão observada e erros locais. Reutilizar a mesma definição de formulário nos modos rápido e completo.

As novas operações de alteração de especificação precisam respeitar contratos congelados e revisão pendente. Não “aplicar mesmo assim”, não invalidar silenciosamente a entrega e não revogar automaticamente um contrato. O inspector explica a condição bloqueadora e encaminha para a operação legítima de encerramento ou recuperação.

#### 5.6 Criar várias entradas e montar o mapa em sequência

Oferecer “Adicionar várias Tasks” e “Adicionar vários Objectives” no contexto atual: uma entrada por linha, prévia dos itens e confirmação única. Para hierarquia mista, um editor de estrutura opcional permite indicar explicitamente tipo e pai em cada linha; indentação visual não executa movimentos em registros já existentes.

Implementar `CreatePlanItems` para criação de novos itens em um único Outcome, com `local_key` e referências a pais existentes ou criados no mesmo lote. Limite inicial: 50 novos itens e 100 critérios no lote, sempre respeitando limites canônicos mais restritivos e o limite de bytes. Tasks do lote permanecem em backlog. Não criar um endpoint genérico para executar uma lista arbitrária de comandos.

A prévia mostra quantos objetivos obrigatórios, opcionais e Tasks serão criados. Rejeitar ciclos locais, pais ausentes, tipos incompatíveis e chaves repetidas antes de escrever. A falha de qualquer entrada desfaz o lote inteiro. Retry da mesma intenção não cria duplicatas.

### 6. Contrato de informação dos cards

Os cards precisam ser legíveis em zoom de trabalho, não pequenos relatórios. Mostrar título, tipo, estado principal, uma linha de participantes relevante e até duas linhas de resumo. Detalhes completos pertencem ao inspector.

| Card | Resumo recomendado |
| --- | --- |
| Outcome | Objetivos obrigatórios alcançados / total; critérios obrigatórios atendidos / total; indicação de impedimento |
| Objective | Critérios atuais; Tasks relacionadas; requisito obrigatório/opcional explícito |
| Task | Planejada, pronta, em execução, em revisão ou correções; executor atual quando conhecido; pendência mais relevante |
| Registros contextuais | Tipo, descrição curta, data/origem relevante e indicação do vínculo selecionado |

Os termos humanos são projeções do estado existente, não novos lifecycles persistidos. A UI deve manter acesso ao estado canônico e aos motivos. Uma Task pode estar em execução e bloqueada: o destaque do impedimento não apaga a informação de execução.

Contadores de progresso não misturam dimensões. Separar trabalho concluído, critérios atendidos, critérios dispensados e objetivos alcançados. Um denominador desconhecido aparece como desconhecido, não como zero. Critérios retirados e avaliações de outra revisão/entrega não aumentam o total comprovado atual.

O indicador de evidência deve distinguir observações únicas, vínculos e avaliações. A evidência reutilizada em três critérios não se torna três documentos. Evitar “100% concluído” quando apenas todas as Tasks terminaram.

### 7. Canvas, navegação e responsividade

#### 7.1 Visão padrão

Abrir no mapa de trabalho, não no grafo indiscriminado de todas as entidades. Mostrar o Outcome e os ramos operacionais. Problemas, evidências, decisões e artefatos aparecem como indicadores; podem ser expandidos sob demanda. As coleções completas continuam acessíveis e pesquisáveis.

Manter somente uma barra compacta com busca, mapa/lista, adicionar no contexto, filtros e organização visual. Recolher ferramentas globais e funções técnicas. Não recriar uma sequência de headers, tabs e resumos que consuma a metade da altura disponível.

#### 7.2 Relações visuais

Pertencimento deve ser lido como estrutura, sem parecer uma fila de execução. Dependência exibe claramente “A depende de B” e força hard/advisory. Problemas mostram “afeta”; impedimentos mostram “bloqueia”; evidências mostram sua stance real. Não depender só de cor ou de hover.

Ao selecionar uma relação, mostrar as duas pontas com nomes, significado, motivo e operação permitida. EvidenceLink e Relation são inspecionados como relações, não como Tasks. Critérios podem ser abertos sem perder o card proprietário.

#### 7.3 Escala e foco

Separar zoom do mapa de dimensionamento responsivo da interface. O tamanho de leitura de controles, inspector e formulários não acompanha o zoom do canvas. O aumento do monitor deve ampliar área útil, não afastar indefinidamente componentes ou reduzir sua importância visual.

A visão inicial usa um zoom legível e colapsa ramos quando necessário. Não executar “Fit all” automaticamente se isso tornar os títulos ilegíveis. “Ver mapa inteiro” é explícito. Ao selecionar um item fora da área atual, deslocar o mínimo necessário; não reorganizar todo o mapa a cada inspeção, atualização ou nova evidência.

Preservar posições assinadas/negativas, pan por roda/trackpad, zoom ao ponteiro, foco e acessibilidade já tratados pelo PR. Não usar escala global do documento para corrigir densidade.

#### 7.4 Inspector responsivo

Em desktop largo, inspector lateral com largura de leitura adequada e máximo definido por token. Em HD ou baixa altura, painel sobreposto à área do mapa, preservando contexto/câmera. Em mobile, painel de detalhe de largura disponível com retorno claro para o mapa/lista. O layout é orientado ao viewport CSS, não à resolução física ou a marcas de monitor.

Preferir um corpo de leitura com uma rolagem principal. Rodapé curto: uma ação principal, até duas secundárias e “Mais”. Não manter quinze ações em um rodapé com rolagem separada. Formulários longos não podem esconder salvar, cancelar ou erros atrás do teclado ou de barras fixas.

Aceitar navegação por teclado, nomes longos, zoom nativo de 125%, 150% e 200%, texto ampliado e movimento reduzido. Respeitar a área útil dos assets de papel sem cortar arte nem cobrir texto com dobras ou ilustrações.

### 8. Inspector operacional compartilhado

#### 8.1 Esqueleto comum

O topo identifica entidade, título, caminho hierárquico e estado. Logo depois, um resumo curto responde “o que falta agora” e apresenta a ação pertinente, quando existir. Em seguida vêm as seções específicas. Auditoria, IDs completos, versões técnicas, checksums e payloads ficam recolhidos, mas disponíveis a quem tiver acesso.

No caso da Task, os cinco grupos solicitados não podem ficar escondidos em cinco abas independentes. A visão de resumo mostra prévias de todos eles, mesmo que “Ver tudo” abra uma seção detalhada. Quando um grupo não possui informação, exibir isso com ação contextual; quando não foi carregado, exibir “não carregado”, não “vazio”.

#### 8.2 Task

Apresentar, nesta ordem funcional: contexto e trabalho solicitado; responsável e execução/revisão; pendência atual; critérios e comprovação; entrega mais recente e atividade; dependências/problemas; detalhes adicionais. A posição exata pode adaptar-se à largura, sem esconder os grupos.

“O que precisa fazer” vem de instruções e entregáveis definidos, condições de prontidão, critérios atuais, revisão pendente e correções registradas. Não inferir uma nova task list a partir de texto livre ou de uma narrativa do agente. “O que foi feito” distingue atualização declarada, artefato registrado, entrega recebida, revisão realizada e conclusão formal.

#### 8.3 Outcome e Objective

Mostrar resultado/condição desejada, critérios próprios, objetivos/Tasks associados, responsáveis declarados e principais pendências de conclusão. No Outcome, a lista de objetivos obrigatórios é explícita. No Objective, a lista de tarefas ajuda a acompanhar trabalho, mas não afirma que todas são gates de aceite se o domínio não as define assim.

Quando o usuário solicita conclusão com trabalho aberto, explicar quais itens permanecerão abertos e quais ficarão indisponíveis em decorrência do escopo terminal. Não cancelar, mover, concluir ou dispensar esses itens por inferência. A revisão de impacto precisa corresponder ao estado usado para confirmar a conclusão.

#### 8.4 Criterion e Assessment

Abrir Criterion dentro do proprietário. Exibir definição, obrigatoriedade, modo, revisão vigente, evidências relacionadas, avaliações e aplicabilidade. Indicar claramente avaliação desatualizada por mudança de critério ou por entrega diferente. Para contratos, preservar o vínculo à submissão e ao digest corretos.

Exibir estados distintos: sem avaliação, atendido, não atendido, inconclusivo, dispensado e não aplicável. “Tem evidência” não vira estado “Atendido”. EvidenceLink no critério também não passa a significar que a evidência foi aceita para a revisão atual.

#### 8.5 Evidence, Artifact e Decision

Evidence mostra observação, fonte, produtor declarado, principal que a registrou, data observada, data de registro, versão/checksum quando disponível e uso por alvo/critério. Mostrar retração e o efeito sobre avaliações correntes sem reescrever o passado.

Artifact mostra o material de referência, produtor, versão, URI segura e usos. Um link não significa upload nem verificação de conteúdo remoto. Decision mostra questão, alternativas, proposta/escolha, justificativa, autor, situação e decisões substituídas/substitutas. Aprovar decisão não libera automaticamente um Blocker associado.

#### 8.6 Issue e Blocker

Issue mostra problema, severidade, itens afetados, quem reportou, Tasks de investigação/correção relacionadas e resolução. Não confundir o autor da Issue com responsável de execução. Uma Task vinculada para investigar pode mostrar sua execução real, sem adicionar artificialmente autoridade à Issue.

Blocker mostra alvo exato, causa, alcance, condição de liberação e estado. Evidências relevantes são recuperadas pelos vínculos da causa ou do alvo, com origem explícita; não simular que existe EvidenceLink direto ao Blocker.

#### 8.7 Roadmap, relações e registros de execução

Roadmap permanece opcional, com Draft, revisões publicadas, visualização e ativação distintas. Mostrar marcos, fases e referências ao trabalho existente. Não duplicar a árvore operacional nem mudar ownership ao mover uma referência no plano.

Contratos, submissões e casos de revisão aparecem dentro da Task com linguagem funcional e acesso à auditoria. ActorRef, ExternalContext, Namespace, triggers e credenciais permanecem no contexto apropriado de participação, integração e configuração, não viram post-its do mapa principal.

### 9. Operações contextuais

#### 9.1 Adicionar evidência

A partir de Task, Objective, Outcome ou Criterion, preencher automaticamente o alvo. Pedir observação, origem e significado da relação. O usuário não precisa procurar novamente o item de origem. Ao abrir por Criterion, fixar seu proprietário e identificador; a alteração do alvo é uma ação explícita.

Oferecer três origens: referência externa; Artifact já registrado; observação humana registrada. A última cria uma nota/observação interna identificável e a Evidence correspondente, com autoria e timestamps reais, na mesma transação. Ela é declaração/observação documentada, não um teste executado pelo servidor.

Preservar a justificativa do vínculo: não gerar um texto de comprovação que o usuário não informou. O modo simples pode exibir um campo de observação e outro curto de relevância. Metadados de origem são preenchidos com dados verificáveis ou confirmados pelo usuário. Não buscar a URL automaticamente.

A operação `RegisterEvidenceWithLinks` registra Evidence, Artifact opcional e vínculos explícitos em uma única UnitOfWork. Permitir vincular Evidence existente sem copiar seu conteúdo. Os targets permitidos vêm do contrato de capacidades do servidor. O default visual é `context` para um item genérico e `supports` quando a intenção foi adicionar comprovação a um critério, sempre visível/editável. Contradição continua sendo opção de primeira classe.

Validar a revisão do critério e a versão do alvo observadas quando a intenção depende da definição. Um vínculo ao ID estável do critério não aprova automaticamente revisões futuras. Após salvar, retornar ao mesmo item com a evidência visível e a avaliação ainda pendente.

#### 9.2 Registrar atualização documental

No item, oferecer “Registrar atualização”: texto, tipo documental simples e referências opcionais. O tipo não executa transição. Escrever “concluído” não conclui a Task; selecionar “problema” não cria automaticamente Issue; uma nota de validação não é Assessment.

Persistir a nota como registro imutável, com owner, principal, ActorRef autorizado, timestamp de registro e referências validadas. Correção é nova nota com `supersedes_note_id`, não edição retroativa. Notas documentais são possíveis em itens concluídos não arquivados; mutações operacionais continuam sujeitas às regras de reabertura.

A nota humana e a WorkActivity contratada aparecem juntas na timeline, com origem distinguível. Não afrouxar os requisitos de `AppendWorkActivity`; quando a UI estiver mostrando material contratado, exibir a associação ao contrato real.

#### 9.3 Reportar problema

No item afetado, abrir um formulário curto com título, descrição opcional, severidade e pergunta “Impede continuar?”. Alvo e item afetado já estão preenchidos. O modo sem bloqueio chama a criação de Issue; com bloqueio utiliza a operação composta existente.

Propagação válida por alvo: Task apenas direta; Outcome conforme regra de alcance do Outcome; Objective permite as alternativas suportadas. Mostrar os itens impactados ou, para subárvore grande, uma prévia limitada com total/limitação honestos. Não oferecer opções que o backend rejeita por tipo.

Se a Issue for criada sobre trabalho terminal, não permitir um Blocker ativo até a reabertura legítima. Problema posterior à entrega pode contestar a conclusão, mas não altera o lifecycle silenciosamente.

#### 9.4 Resolver problema e liberar trabalho

Mostrar os bloqueios ativos relacionados e pedir confirmação dos alvos que realmente podem continuar. Reutilizar `ResolveIssueAndBlockers` com IDs e versões explícitos. Não liberar todos por default e não apagar o motivo histórico. Exibir bloqueios que permaneceram ativos.

#### 9.5 Registrar entregável ou decisão no contexto

Permitir registrar Artifact e proposta de Decision a partir do item, preservando uma relação canônica de contexto. Quando a intenção for “esta Task produziu este Artifact”, utilizar `produces`; quando for apenas referência contextual, utilizar a relação documental apropriada. Não usar EvidenceLink como associação genérica de qualquer coisa.

Se não existir comando composto adequado, extrair os primitivos transacionais existentes e criar um caso de uso pequeno. Nunca chamar dois handlers transacionais públicos em sequência e afirmar atomicidade. O registro do material continua separado de envio de uma entrega contratada.

### 10. Reorganização persistida do mapa

#### 10.1 Mover Task para Objective

Implementar `SetWorkItemObjective`, se não existir handler registrado na revisão de integração, com destino Objective ou `null` para diretamente no Outcome. Verificar escopo, existência, versões, lifecycle, autoridade contratada, revisão aberta e contexto de dependências. O comando não altera TaskID, evidências, critérios, autoria ou entregas anteriores.

O movimento padrão é uma ação no inspector com busca de destino. Pode ser complementado por um modo explícito de arrastar para associar; o drag livre existente continua mudando apenas posição visual. Antes de confirmar uma reassociação, mostrar o pai antigo, o novo e o impacto operacional observado.

#### 10.2 Mover Objective

Implementar `SetObjectiveParent` com validação de ciclos estruturais, destino não terminal e mesmo Outcome. A versão inicial permite reorganizar ramos ainda planejados e sem execução/aceite congelado que seria afetado. Uma subárvore em execução ou com revisão aberta recebe motivo claro e operações de preparação legítimas; não existe force move genérico.

Validar o efeito sobre Tasks descendentes e bloqueios herdados. Não modificar critérios, obrigatoriedade global ou dependências como efeito oculto de mudança de pai. Registros históricos conservam o contexto do acontecimento; o estado atual exibe a nova estrutura.

#### 10.3 Dependência não é pertencimento

Renomear “Connect” para “Add dependency” e mostrar uma frase completa com a direção antes de salvar. Rejeitar autodependência e ciclos conforme regras atuais. Adicionar regressão para uma Task depender obrigatoriamente do próprio Objective direto: se confirmada a incompatibilidade de prontidão, rejeitar esse vínculo com erro específico e orientação para usar associação.

A mesma verificação precisa ocorrer ao mover uma Task para um Objective de que já depende. Dados antigos nessa configuração devem ser identificados na inspeção, sem remoção automática. Dependências advisory não bloqueiam execução e não serão rejeitadas apenas por esse motivo.

### 11. Ações, preparação, execução e conclusão

Expor uma ação primária derivada das condições reais e poucas secundárias. Guardar operações destrutivas/administrativas no menu “Mais”, com consequência e confirmação. O nome da ação deve indicar seu alvo. O menu global não pode trocar silenciosamente o escopo de um formulário iniciado na Task.

| Situação | Ação ou explicação |
| --- | --- |
| Outcome sem critério obrigatório ativo | Definir condição de sucesso |
| Outcome preparado | Iniciar Outcome, após confirmação |
| Objective sem critério | Adicionar critério |
| Task em backlog | Completar detalhes ou disponibilizar |
| Task pronta e autoridade disponível no browser | Iniciar trabalho via operação permitida |
| Task pronta no protocolo assinado | Continuar com perfil autorizado, com contexto exato |
| Dependência não atendida | Abrir pré-requisito nomeado |
| Bloqueio ativo | Abrir impedimento e causa |
| Entrega recebida | Aguardar ou assumir revisão pelo canal autorizado |
| Correções solicitadas | Ver achados e obter autoridade de correção |
| Critérios elegíveis para revisão | Revisar comprovação |
| Condições de conclusão satisfeitas | Revisar e concluir explicitamente |

Não ocultar indisponibilidade como ausência inexplicável de botão. Permissões podem ocultar operações sensíveis; impedimentos de domínio de uma ação que o usuário conhece aparecem com motivos e remédios. Uma query de elegibilidade não concede autorização nem substitui a validação no commit.

Ao concluir Objective, permitir o caminho canônico `planned → achieved` quando elegível, sem exigir um Start artificial. Ao concluir Outcome, revisar critérios próprios e todos os Objectives obrigatórios. Critérios dispensados são mostrados como dispensados, não verificados. Tasks abertas ou conclusões contestadas recebem tratamento explícito na revisão; nenhum aceite automático por porcentagem.

Revisão de critérios, submissão de resultado, revisão independente e conclusão podem compartilhar uma superfície, mas continuam intenções identificáveis. Não criar um “concluir tudo” que aprova múltiplas camadas sem decisão.

### 12. Leituras operacionais e backend

Criar um modelo de leitura `EntityWorkspaceView`, com resumo do item, contexto, participantes, situação, critérios, comprovantes, atividade recente, relações, pendências e capacidades. Implementar facetas específicas por tipo, em vez de dezenas de campos nulos de uma entidade universal.

Reutilizar `WorkContext`, `WorkAttention`, projeções de prontidão, avaliações, contratos e revisão já existentes. Extrair leitura interna sob uma UnitOfWork/contexto de snapshot comum. Não chamar vários serviços que abrem transações independentes e depois atribuir uma revisão única ao conjunto.

Para o mapa, produzir resumo em lote dos cards selecionados ou da página estrutural. Não hidratar inspector, histórico e documentos completos de cada card. Índices e consultas precisam buscar filhos diretos, referências e atividade paginada; limitar o resultado HTTP não corrige uma leitura SQL que continua carregando o Outcome inteiro.

Separar capacidade disponível, motivo de impedimento e prioridade de apresentação. As mesmas funções de precondição usadas para explicação devem orientar a validação dos comandos, preservando verificações finais de versão, tempo, lease e autoridade. Não duplicar o motor de regras no JavaScript.

A API continua independente do dashboard: nenhuma regra deve depender de “veio do canvas” ou de uma flag enviada por um cliente não confiável.

### 13. Timeline unificada, sem nova verdade de domínio

A timeline é uma projeção que combina eventos de domínio, WorkActivity, EntityNote e referências a entrega/revisão. Não criar cópia editável de fatos já existentes.

Ordenar por sequência de commit do Outcome e desempate estável de evento; manter `occurred_at/captured_at` como informação, sem usá-los para inventar a ordem de commit. Um evento composto pode compartilhar uma revisão com outros fatos: não paginar só por `outcome_revision` sem desempate.

Preservar `origin`, referência canônica e correlation/causation. Um `work_activity.recorded` com o mesmo WorkActivity não aparece como duas atualizações idênticas. Agrupar visualmente os fatos de uma intenção composta sem apagar seus registros auditáveis.

Ao abrir o item, carregar a primeira página recente automaticamente: cinco entradas no resumo, até vinte na resposta inicial. O restante usa cursor. Outcome/Objective podem mostrar atividade própria e dos descendentes com origem explícita; não atribuir uma atividade de filho ao pai como se fosse ação do pai.

Cursor congela a fronteira histórica. Novos acontecimentos geram indicação de atualização, não alteram páginas antigas durante a navegação. O usuário pode atualizar o resumo preservando foco e formulário; dados novos não devem misturar estados incompatíveis de uma avaliação aberta.

### 14. Erros, concorrência e recuperação

Corrigir imediatamente a classificação de `precondition_failed`. Falta de critério, dependência, bloqueio ou protocolo não significa conflito de edição. Adicionar motivos tipados e referências relacionadas; a UI mostra a ação corretiva, não “Descartar minhas edições”.

Conservar `version_conflict` para desatualização real. Edições simples podem apresentar comparação de versão base, conteúdo local e estado atual, com confirmação de nova intenção. Avaliação, movimento estrutural, transição e decisão de revisão precisam reexaminar os dados relevantes, sem rebase automático.

Se uma resposta for incerta após envio, congelar bytes, idempotency key e escopo autenticado. Não permitir editar e reenviar com a mesma chave. Recuperar o resultado da intenção original antes de formar uma nova. Se o write foi confirmado e o refresh falhou, mostrar “Salvo; atualização da visualização pendente” e não oferecer uma segunda criação.

Adicionar navegação de correção com pilha curta de contexto: formulário original preservado, abrir o critério faltante, salvar, retornar e reavaliar a decisão. Não submeter a operação original automaticamente. Mudança de usuário/namespace/outcome cancela aplicação de respostas antigas e remove dados não autorizados.

### 15. Persistência e compatibilidade

Reutilizar tabelas/agregados existentes para ownership, critérios, participantes, evidências, relações e avaliação. Não manter arrays paralelos no frontend ou uma tabela genérica `dashboard_state` com regras operacionais.

A nova nota documental requer persistência pequena e append-only, com autorização, referências e índices de escopo. Utilizar migrations aditivas nos adapters persistentes e implementação equivalente no adapter de memória. A nota não cria uma nova entidade operacional que possa receber lease, virar dependência ou participar do grafo principal.

Novos campos opcionais e comandos são aditivos. Não tornar campos obrigatórios em contratos antigos silenciosamente. A exigência de motivo em uma nova operação deve estar em seu contrato; em operações antigas, a UI pode enviar justificativa adicional com mudança compatível, sem inventar motivos para eventos históricos.

A base existente não será convertida para novos defaults: não criar critérios por heurística, não marcar objetivos antigos como obrigatórios, não concluir itens, não retirar evidências e não reconstruir autoria sem fonte. Posições visuais permanecem preferências locais; ao persistir preferências, isolar por usuário/escopo e não tratá-las como estrutura operacional compartilhada.

Não executar binário antigo sobre schema novo sem teste e declaração de compatibilidade. Um rollback que deixa notas novas inacessíveis precisa ser descrito como tal; não apagar dados para facilitar downgrade.

### 16. Arquivos e responsabilidades

| Área existente | Papel na alteração |
| --- | --- |
| `packages/wos-core/domain/` | Tipos/regras de nota, alteração estrutural e condições compartilhadas; sem conceitos específicos do DOM |
| `packages/wos-core/application/update.go`, `lifecycle_commands.go` | Edição posterior, responsáveis e integração dos novos comandos |
| `application/work_contract_guards.go` | Cobertura das novas mutações e congelamento de obrigações |
| `application/work_attention.go`, `focal_current.go`, `focal_graph.go` | Composição de contexto e pendências sem segunda engine |
| `application/documentary_records.go`, `issues_blockers_compound.go` | Reuso de primitivos de registro/vínculo e fluxos compostos |
| `application/pipeline.go` | Inventário, autorização, idempotência, eventos, guard e registro dos novos casos de uso |
| `packages/wos-core/ports/` e adapters reais | Leituras em lote, paginação, nota e migrations |
| `packages/wos-api/` | Rotas, DTOs, schemas, capabilities e erros tipados |
| `web/assets/app.js` | Orquestração existente de autenticação, rotas, scope, intents e refresh |
| `web/assets/human-forms.js`, `actions.js` | Formulários progressivos e catálogo de ações coerente |
| `web/assets/canvas-data.js`, `workspace-canvas.js` | Resumos em lote, camadas, layout, foco e criação contextual |
| `web/assets/workspace-inspector.js`, `canvas-context.js`, `workspace-entity-context.js`, `contracts-view.js` | Inspector único e renderização especializada, sem duplicação |
| `web/assets/en-US.js` e CSS existentes | Vocabulário e responsividade preservando identidade |
| `tests/web/`, testes Go e contratos existentes | Qualificação funcional, compatibilidade e UX |

Novos módulos sugeridos: `workspace-view.js`, `workspace-quick-create.js`, `workspace-criteria.js`, `workspace-timeline.js`, `workspace-flow.js`; novos arquivos Go pequenos para views, notas e edição estrutural. Esses nomes são propostas, não arquivos afirmados como existentes. Preferir extrair responsabilidades existentes e evitar duas implementações de inspector mantidas em paralelo.

### 17. Estratégia de implementação e conclusão

A sequência obrigatória é: baseline reproduzível → contratos/decisões → correções de erro → primitivos de backend e migrations → leituras coerentes → inspector → criação e operações contextuais → relações/layout → qualificação integrada. O backlog e os commits detalhados estão no documento 03.

Cada incremento deve incluir seus testes. Não deixar autorização, protocolo e concorrência para o final. Uma primeira fatia vertical precisa demonstrar: criar uma Task em backlog, completar detalhes, abrir inspector, registrar uma nota, adicionar uma evidência ao critério e mostrar a avaliação ainda pendente.

Antes da entrega, remover o caminho antigo de detalhes duplicados, atualizar documentos operacionais e o escopo do PR. A descrição atual de que não há alteração de API/domínio/persistência deixará de ser verdadeira após esta implementação. Atualizar changelog/versionamento pela rotina do repositório; não presumir que a versão anteriormente planejada ainda é a correta.

O aceite final exige os fluxos de Task, Objective, Outcome e registros documentais completos, não somente screenshots do novo canvas. Testes automáticos comprovam contratos específicos; a meta de compreensão humana deve ser medida separadamente. Sem participantes de teste, marcar a validação de usabilidade como pendente, não como concluída por simulação.


---

## Parte 2 — Contratos propostos, persistência e integração


Este documento especifica extensões propostas. Os nomes novos abaixo não representam endpoints ou comandos já disponíveis. A implementação deve seguir o mecanismo de rotas, schemas, inventário e autorização realmente utilizado pelo WOS, preservando os contratos anteriores.

### 1. Separação de responsabilidades

`domain` mantém invariantes e tipos. `application` coordena autorização, guard, leitura consistente, casos de uso e transações. `ports` descreve as capacidades de persistência/consulta. Adapters implementam essas capacidades. A API traduz transporte. O frontend apresenta dados e coleta intenções explícitas.

Não adicionar `is_dashboard`, `skip_contract_validation` ou `force=true` para permitir atalhos humanos. Não colocar regras de prontidão no CSS, na posição do card ou no estado local do componente.

### 2. Vocabulário de endereçamento

Uma referência operacional continua contendo namespace, Outcome, kind e ID. Criterion usa proprietário e criterion ID, não um UUID de entidade global inventada. RoadmapRevision usa Roadmap mais revisão. Material de execução conserva seu contrato, submissão e digest.

As novas queries de inspector podem aceitar um seletor discriminado:

- `entity`: EntityRef persistida;
- `criterion`: EntityRef do proprietário + criterion_id;
- `roadmap_revision`: EntityRef do Roadmap + revision_number;
- `execution_record`: TaskRef + identidade de contrato/submissão/caso, com validação específica.

Não transformar esse seletor de leitura em um novo registry de entidades que possam receber dependências ou bloqueios.

### 3. EntityWorkspaceView

Modelo mínimo, com facetas tipadas e paginação:

| Campo | Contrato |
| --- | --- |
| `subject` | Referência selecionada e, quando necessário, sub-referência de Criterion/revisão |
| `entity_version` | Versão corrente pertinente à edição, preservada sem perda de precisão |
| `outcome_revision` | Revisão decimal exata do snapshot |
| `evaluated_at` | Instante de avaliação das condições, fornecido pelo servidor |
| `header` | Nome, tipo, lifecycle canônico, estado de apresentação e descrição curta |
| `placement` | Outcome, pai direto, caminho conhecido e limites de contexto |
| `people` | Papéis, autoria, responsabilidade declarada, autoridade vigente e última participação, com fontes |
| `work` | Instruções, restrições, entregáveis e escopo; apenas para tipos pertinentes |
| `criteria` | Critérios atuais, avaliação aplicável, evidências relacionadas e referências à revisão/entrega |
| `delivery` | Resumo da entrega corrente, identificação, estado de revisão e resultado, quando existente |
| `relationships` | Grupos tipados, referências resolvidas e cursores |
| `attention` | Obrigações derivadas, impedimentos e correções; sem lista pendente persistida duplicada |
| `activity` | Primeira página recente da timeline e fronteira histórica |
| `capabilities` | Ações e respectivos canais, disponibilidade, motivos e alvos |
| `coverage` | Completude por faceta: completo, parcial, não carregado, indisponível ou não aplicável |

Campos irrelevantes são omitidos ou marcados `not_applicable`; não produzir vinte seções vazias no inspector. Não retornar dados privados de assinatura, credenciais ou contexto fora da autorização do principal.

#### Read consistency

Abrir um contexto de consulta coerente, autorizar dentro dele e fixar revisão/tempo. Reutilizar funções internas de WorkContext e WorkAttention com esse contexto. Se cada serviço público abre sua própria UnitOfWork, extrair funções internas antes de compor. Uma igualdade acidental de revisão entre respostas não substitui um contrato transacional de leitura.

Resumos de entidades, status assinado, critérios e material precisam corresponder à mesma revisão relevante. Material histórico imutável pode ser acessado por identidade/digest; seu conteúdo não autoriza assumir que ainda é a entrega corrente. Retornar a associação corrente separadamente.

Manter versões/revisões de 64 bits como representação exata no transporte e no frontend, sem conversão insegura para `Number`. Não modificar silenciosamente formatos de endpoints antigos; padronizar as novas projeções e conservar os codecs existentes.

#### Coverage

Cada faceta usa `status`, `returned_count`, `total_count` somente quando conhecido, `next_cursor` e eventual motivo. `null`/ausente significa desconhecido, não zero. O backend só declara ausência de bloqueios, evidências ou dependências após consultar a fonte correta com cobertura suficiente.

Uma query restrita por profundidade continua restrita mesmo quando seu cursor terminou. Um `Load more` deve indicar qual coleção ou ramo está expandindo; não prometer carregar o banco inteiro.

### 4. Read model dos cards

Adicionar leitura em lote de resumos dos itens de uma página estrutural ou de um conjunto autorizado de refs. Limite inicial: 100 refs por consulta, sujeito ao limite de bytes da API. Aplicar uma única revisão por lote e validação de escopo para cada ref.

Cada resumo inclui título, tipo, estado humano e canônico, indicação de autoridade quando conhecida, participantes declarados, resumo de critérios e principal motivo de atenção. Não incluir histórico completo, bytes de evidência, payloads assinados ou cópias de todas as relações.

Se a projeção não consegue resolver autoridade em lote com o protocolo correto, retornar `authority_state=unknown` e permitir resolver na inspeção. Nunca preencher executor a partir de `assignee_refs` nem inferir expiração assinada de um lease legado.

Contar critérios correntes apenas quando avaliação, revisão, evidência e submissão satisfazem a aplicabilidade. `met` e `waived` têm contadores diferentes. Dependências e impedimentos têm dimensões separadas. O número de documentos distintos não é o número de EvidenceLinks.

### 5. Capabilities e precondições

Uma capability proposta possui:

```
action_id
subject
transport: browser_command | signed_host | read_only
availability: available | blocked | unavailable | unknown
reason_codes[]
related_refs[]
required_permissions[] (somente quando sua divulgação for autorizada)
observed_versions
confirmation_kind
presentation_priority
```

A prioridade é uma decisão de apresentação baseada no estado, não uma ordem que o WOS executa. O frontend conhece renderizadores seguros para action IDs permitidos; não executa uma URL ou comando arbitrário retornado em texto.

Query e mutação compartilham predicados de regra. O handler revalida autorização, lifecycle, versão, guard, contrato, revisão, relógio e evidências antes de gravar. Entre preview e commit pode haver mudança legítima; a capability não funciona como token de autorização.

A lista de alvos permitidos de EvidenceLink, propagação de Blocker, tipos de associação e transições deve ser gerada/derivada de uma definição canônica, com testes de paridade entre domínio, API e UI. Não manter quatro allowlists divergentes.

### 6. Intenções de escrita

| Operação proposta | Implementação | Garantias |
| --- | --- | --- |
| Criar Outcome com critérios iniciais opcionais | Extensão aditiva do comando existente, se compatível com o gerador de schemas; caso contrário, caso de uso composto específico | Uma transação; nenhum critério inventado; salva draft |
| Criar Objective/Task com critérios iniciais | Mesmo padrão, preservando a criação antiga sem esses campos | Pai validado; critérios pertencem ao novo agregado; não inicia execução |
| `CreatePlanItems` | Comando específico de criação em lote no mesmo Outcome | No máximo 50 itens/100 critérios; local keys; sem mutações em itens existentes; tudo ou nada |
| `SetWorkItemExecutionSpec` | Setter tipado, não patch JSON arbitrário | Versão; contrato congelado; revisão aberta; validação de refs |
| `SetWorkItemNotBefore` | Setter de data com clear explícito | Versão e estado elegível; horário não inicia execução |
| `SetWorkItemObjective` | Reassociação específica | Sem mudança de ID; mesmo Outcome; guard e verificações de dependência |
| `SetObjectiveParent` | Reparenting específico | Sem ciclos; validação da subárvore afetada e do estado de execução |
| `RegisterEvidenceWithLinks` | Primitivos documentais sob uma única UnitOfWork | Evidence/Artifact opcional/nota de origem/vínculos explícitos; rollback completo |
| `AppendEntityNote` | Registro documental append-only | Autoria corrente; refs autorizadas; sem transição, assinatura ou aceite |
| Registros contextuais de Artifact/Decision | Casos de uso pequenos reutilizando registro e Relation | Vínculo canônico; sem EvidenceLink genérico indevido |
| Reportar Issue com Blocker | Reutilizar o comando composto existente | Associação não é bloqueio implícito; alvo/alcance válidos |
| Resolver Issue e impactos selecionados | Reutilizar operação existente | IDs, versões e confirmação por impacto |
| Assessment e conclusão | Comandos existentes e seus gates | Sem aprovação por associação ou contador |

Não adicionar uma transação distribuída no browser. Não simular atomicidade com “salva registro, tenta vínculo e, se falhar, deleta o primeiro”. Registros imutáveis/auditáveis não são compensações descartáveis.

#### Pipeline

Toda nova mutação deve entrar no inventário tipado: schema, permissão, classificação de exposição, idempotência, receipt/replay, guard de Outcome, proteção de contratos, eventos e outbox. Não registrar somente a rota HTTP e esquecer a camada de autorização ou as proteções aplicadas por nome/tipo de comando.

Primitivos internos transacionais não chamam outros handlers públicos que abrem uma nova transação. O receipt de sucesso conserva os IDs criados e o resultado da intenção. O retorno de `CreatePlanItems` inclui o mapa `local_key → ID` e as referências/versões necessárias para continuar a edição.

Eventos compostos compartilham correlação e revisão, mas registram os fatos reais: item criado, critério adicionado, evidência registrada, vínculo criado. Não criar um evento fictício “objetivo comprovado” quando apenas se salvou uma evidência.

#### Atualizações posteriores

Definir semântica de ausência, substituição e limpeza. Para setters de lista, o comando substitui o conjunto explicitamente informado e não faz merge oculto. Para associação/data, `null` significa limpar; ausência não significa limpar. O transporte deve preservar essa distinção.

A alteração de obrigação ou contexto congelado exige o fluxo autorizado já existente. A versão inicial das novas operações não oferece override para contratos ou revisão pendente. A interface informa o impedimento e mantém o rascunho de edição.

### 7. EntityNote: extensão documental mínima

Modelo proposto:

| Campo | Semântica |
| --- | --- |
| `id` | Identidade criada pelo servidor |
| `scope` | Namespace e Outcome |
| `subject_ref` | Entidade dona do contexto |
| `criterion_id?` | Apenas para notas de Criterion dentro de proprietário válido |
| `type` | update, discovery ou observation; categoria documental, nunca transição |
| `body` | Texto validado e limitado a 16 KiB na versão inicial |
| `principal_id` | Identidade autenticada que registrou |
| `actor_ref?` | Autoria declarada/delegada conforme regras existentes |
| `recorded_at` | Tempo do servidor/UnitOfWork |
| `occurred_at?` | Tempo informado da observação; não controla ordem de commit |
| `artifact_refs[]`, `evidence_refs[]` | Referências autorizadas e verificadas |
| `supersedes_note_id?` | Correção explícita, mesmo sujeito e autor principal na versão inicial |
| `outcome_revision` | Fronteira de commit para timeline |

Permissão padrão: `records:write`, sujeita a acesso ao sujeito e às regras de arquivamento. Corrigir nota de outro autor não é uma edição permitida por possuir acesso de escrita: pode-se responder com outra nota, preservando ambas. Não inventar moderação administrativa genérica nesta entrega.

Adicionar repository/port, implementação de memória e migrations persistentes. Índices por `(namespace, outcome, subject, revision, id)` e por ID de correção; referências locais devem manter integridade de escopo. Registrar DomainEvent na mesma transação. Conteúdo antigo nunca é reescrito por uma alteração do card.

#### Observação interna como origem de Evidence

A opção explícita de observação humana cria uma nota `observation`, depois Evidence com SourceRef apontando para essa nota, e vínculos escolhidos. O servidor valida a origem interna existente; um nome de provider enviado pelo cliente não prova procedência interna. O inspector só identifica “observação registrada no WOS” quando a referência pode ser resolvida e validada.

O texto continua sendo declaração/observação do participante. Não transformar uma nota “rodei os testes” em resultado de testes produzido pelo sistema. A origem, autoria e natureza declarada permanecem visíveis.

### 8. Participantes sem uma segunda ontologia

Usar `SetOutcomeOwners`, `SetObjectiveOwners` e `SetWorkItemAssignees` já identificados na application, expondo-os de forma contextual e com o inventário/permissions apropriados. Não criar uma tabela universal de assignments que duplique owner_refs/assignee_refs.

No inspector, exibir papéis com sua fonte:

- responsabilidade declarada: owner/assignee;
- executor autorizado atual: contrato/lease vigente, com status de autoridade;
- último executor: histórico real;
- revisor atual: contrato/caso de revisão corrente;
- autor de atualização/evidência: principal de registro e ActorRef autorizado;
- produtor declarado de Artifact: referência de produção.

Issue pode mostrar quem reportou e as tarefas relacionadas que investigam/corrigem o problema. Não atribuir a ela executor próprio inexistente. Uma declaração de ActorRef não é automaticamente uma conta autenticada ou autorização. Names resolution não concede acesso.

### 9. Timeline e cursores

A projeção combina registros, sem copiá-los para uma tabela editável de atividade do dashboard. Cada entrada possui ID canônico, origem, assunto, principal/ator, descrição, instante de registro, revisão e referências.

Usar ordem estável por revisão de commit e desempate canônico. Quando o modelo de eventos possuir ordem intra-commit, reutilizá-la. Caso contrário, documentar desempate estável por origem/ID sem afirmar cronologia não registrada dentro da transação.

Cursor opaco inclui sujeito/filtro autorizado, fronteira histórica e última chave de ordenação. Deve funcionar quando vários eventos têm a mesma revisão. Novos registros não aparecem no meio de uma paginação congelada. Recomeçar a timeline é uma nova leitura explícita.

Deduplique a projeção de um WorkActivity e seu evento de registro, não dois fatos diferentes que por acaso têm textos iguais. Expor origem para itens de descendentes. Eventos de execução, revisão e conclusão continuam distinguíveis mesmo quando agrupados visualmente em uma mesma intenção.

### 10. Erros tipados

Preservar códigos públicos existentes e adicionar detalhes compatíveis, em vez de obrigar o frontend a analisar mensagens em inglês:

```
code: precondition_failed
reason_code: required_active_criterion_missing
subject_ref: ...
field_paths: ["criteria"]
related_refs: []
remedies: [{ action_id: "add_criterion", subject_ref: ... }]
```

Outros motivos necessários incluem dependência não atendida, Blocker aplicável, Objective terminal, autoridade necessária no host, revisão aberta, critério alterado, definição congelada e alvo de evidência inválido. A nomenclatura deve seguir o catálogo do projeto e ser coberta por schemas e testes.

No caso de dependência impossível com o próprio Objective, explicar a distinção de associação. Em uma falha de leitura limitada, não utilizar `required_active_criterion_missing` sem saber se o critério está ausente. Em timeout incerto, não atribuir erro definitivo de domínio.

### 11. HTTP, CLI, MCP e geração de contratos

Adotar rotas de leitura coerentes com o prefixo existente, por exemplo `GET O/workspace-view?...`, `POST O/workspace-summaries` para consulta em lote quando o padrão da API admitir esse transporte, e `GET O/timeline?...`, sendo `O` o path do Outcome. Estes nomes são propostas de interface, não rotas confirmadas.

Mutações seguem o transporte de comandos existente. Não criar uma API privada do dashboard que contorne o inventário utilizado pelo CLI/MCP. Para cada novo comando/query, decidir e registrar exposição: humano/documental, leitura para agentes ou operação técnica. Geradores, schemas, exemplos e testes de compatibilidade devem ser atualizados juntos.

O CLI continua sendo o canal de autoridade assinada; a UI não precisa abrir ou editar YAML. Encaminhamentos mostram perfil e alvo reais, mas jamais escolhem uma credencial automaticamente. Inputs humanos usados em snippets devem ser tratados como dados; não concatenar shell arbitrário. Manter os mecanismos de recuperação de intenção e aprovações de revisão separados.

### 12. Migração, desempenho e limites

Migrations de notas e índices são aditivas. Sem alteração automática de required_for_outcome, lifecycle ou critérios antigos. Não criar notas retrospectivas artificiais para preencher uma timeline vazia. Dados antigos sem procedência suficiente aparecem como desconhecidos.

Na primeira versão, não persistir uma segunda projeção de prontidão. Preferir consultas indexadas e agrupadas. Medir planos de consulta e volume lido nos adapters. Projeção futura materializada só após necessidade demonstrada, com invalidadores e fonte canônica definidos.

Os limites novos nunca ampliam silenciosamente limites existentes: respeitar o menor limite aplicável de payload, critérios por agregado, refs e grafo. Validar lotes antes e dentro do guard. Uma expansão excedida retorna erro/coverage específico e não aceita uma relação sem validar ciclos.

Os exemplos no diretório `exemplos/` são contratos conceituais válidos como JSON, com identificadores fictícios. Não são payloads garantidamente aceitos pela versão atual do WOS.


---

## Parte 3 — Backlog e sequência de commits


As 24 unidades abaixo são incrementos revisáveis no mesmo PR. A sugestão é um commit por unidade quando coeso; uma unidade pode exigir commits menores. Não condensar migrations, regras, UI e testes em um único commit opaco. Nenhuma tarefa está marcada como implementada.

### Ondas de integração

| Onda | Unidades | Resultado |
| --- | --- | --- |
| A — Baseline e invariantes | WOS-01 a WOS-05 | Base reproduzível, erros corretos e regras compartilhadas |
| B — Capacidade de escrita | WOS-06 a WOS-10 | Notas, criação completa, edição, estrutura e operações atômicas |
| C — Leitura operacional | WOS-11 a WOS-13 | Timeline, inspector view e cards em lote |
| D — Experiência contextual | WOS-14 a WOS-19 | Inspector, criação, comprovação, problemas, ações e movimentos |
| E — Mapa e integração | WOS-20 a WOS-22 | Canvas legível, responsividade e paridade dos consumidores |
| F — Qualificação | WOS-23 a WOS-24 | Aceite integrado e PR documentado |

### Trabalho paralelo

Após WOS-05, notas, criação e edição podem avançar em paralelo com responsáveis distintos. Depois dos contratos, UI pode preparar componentes com fixtures, mas aceite exige dados e comandos reais. Migrations, pipeline, schemas e `app.js` têm propriedade explícita por incremento para evitar alterações concorrentes incompatíveis. Nunca avançar um fluxo visível antes de existir sua autorização e escrita reais.

### Unidades de implementação

#### WOS-01 — Baseline e preservação da branch

**Depende de:** nenhuma.  
**Commit sugerido:** `docs(wos): baseline de fluxos e contratos do Outcome Board`.

**Implementação.** Conferir HEAD remoto/local, mudanças não commitadas, instruções do repositório e todos os contratos realmente registrados. Registrar o diff já existente no PR; preservar assets, testes e trabalho paralelo. Produzir baseline de comandos, schemas, páginas e screenshots com revisão, viewport e fixtures identificados.

**Pontos de alteração.** Metadados do PR, CI real, docs operacionais, inventário de comandos e diretórios de teste.

**Validação.** Executar suites existentes no ambiente de implementação, registrar falhas pré-existentes separadamente; comparar branch e SHA. Não afirmar compatibilidade a partir de apenas um build.

**Aceite.** Baseline reproduzível anexada ao PR e mapa das lacunas. Nenhum reset destrutivo, force push, merge ou publicação.

#### WOS-02 — ADR de experiência e contratos

**Depende de:** WOS-01.  
**Commit sugerido:** `docs(wos): contrato operacional e defaults da experiencia humana`.

**Implementação.** Fixar os defaults e as invariantes dos documentos 01/02: Task rápida em backlog, Objective obrigatório visível, critérios opcionais na criação e obrigatórios nos gates pertinentes, notas documentais separadas, inspector comum e protocolo assinado preservado. Especificar capabilities, errors e facetas.

**Pontos de alteração.** docs/adr/, documentação de UX, DTOs propostos e exemplos.

**Validação.** Revisão cruzada dos cenários: criar, editar depois, disponibilizar, adquirir, registrar, avaliar e concluir. Nenhum passo depende de um endpoint inexistente não incluído no backlog.

**Aceite.** Contratos e comportamento estão definidos; diferenças entre UI e default público explícitas.

#### WOS-03 — Fixtures e testes de caracterização

**Depende de:** WOS-02.  
**Commit sugerido:** `test(wos): fixtures e regressoes da operacao contextual`.

**Implementação.** Criar cenários determinísticos pequenos e grandes para legado, contracts_v1 quando suportado e signed_contracts_v2. Incluir critérios opcionais/retirados, evidência contraditória/retraída, revisões antigas, múltiplos atores, hierarquia profunda e respostas parciais.

**Pontos de alteração.** Testes Go, fixtures dos adapters, tests/web/.

**Validação.** Reproduzir a mensagem incorreta de precondition, selector EvidenceLink inválido, criação com contexto e inaplicabilidade de uma avaliação de outra submissão.

**Aceite.** Os problemas conhecidos possuem regressões executáveis, sem desabilitar testes antigos.

#### WOS-04 — Erros de precondição e preparação coerente

**Depende de:** WOS-03.  
**Commit sugerido:** `fix(wos): corrigir precondicoes e recuperacao de formularios`.

**Implementação.** Separar version_conflict de precondition_failed no frontend. Adicionar detalhes de motivo e remédios compatíveis; corrigir checklist para required && status active. Preservar rascunhos, resultados já confirmados e recuperação de resposta incerta.

**Pontos de alteração.** domain/application erros, DTO de erro, app.js, human-forms.js, en-US.js.

**Validação.** Falta de critério abre correção; conflito real preserva comparação; timeout repete a intenção exata; write confirmado e refresh falho não duplica o comando.

**Aceite.** O erro relatado pelo usuário não produz falso conflito de edição.

#### WOS-05 — Predicados, elegibilidade e proteção de comandos

**Depende de:** WOS-04.  
**Commit sugerido:** `feat(wos): unificar elegibilidade e guardas de operacao`.

**Implementação.** Extrair condições reutilizáveis de início, conclusão, comprovação e autoridade. Criar elegibilidade de relações/propagação derivada de regras canônicas. Registrar os novos tipos nas proteções por comando e garantir que composição não contorne o pipeline.

**Pontos de alteração.** application/work_contract_guards.go, work_attention.go, readiness, pipeline, contratos da API.

**Validação.** Paridade entre motivo da query e resultado da mutação sob mesmo snapshot; rejeição com mudança concorrente; targets corretos de EvidenceLink; políticas por protocolo.

**Aceite.** Nenhuma nova ação depende de uma engine paralela no browser e nenhuma composição burla um guard.

#### WOS-06 — Notas documentais persistentes

**Depende de:** WOS-05.  
**Commit sugerido:** `feat(wos): notas documentais persistentes e auditaveis`.

**Implementação.** Implementar EntityNote, port/repository, adapter de memória, migrations e índices persistentes. Append-only, autoria do contexto autenticado, sujeito e refs escopados, correção explícita do mesmo autor e mesmo sujeito. Não alterar WorkActivity contratada.

**Pontos de alteração.** domain/ports/application de nota; adapters e migrations reais; schemas e inventário.

**Validação.** Replay sem duplicação, atomismo com evento, isolamento, terminal não arquivado permitido, arquivado rejeitado, identidade não forjável, correção não altera a nota anterior.

**Aceite.** Nota humana utilizável via comando autorizado, com persistência e histórico, sem conferir autoridade de execução.

#### WOS-07 — Criação com critérios e lotes de planejamento

**Depende de:** WOS-05.  
**Commit sugerido:** `feat(wos): criacao de planejamento com criterios iniciais`.

**Implementação.** Extrair primitivos internos de criação; adicionar critérios iniciais opcionais de forma compatível. Implementar CreatePlanItems restrito a novos itens de um Outcome, chaves locais, pais existentes/locais e validação de lote. Atomicidade e IDs de resultado estáveis no replay.

**Pontos de alteração.** application de criação, pipeline, DTOs, schemas, event factories e testes.

**Validação.** Salvar draft/Objective/Task sem critérios; salvar critérios corretos; lote com 50 entradas; rollback na última entrada inválida; ciclos/chaves duplicadas; Task criada não inicia execução.

**Aceite.** Criação simples e em lote funciona sem dependências implícitas, critérios gerados ou writes parciais.

#### WOS-08 — Edição posterior de detalhes e participação

**Depende de:** WOS-05.  
**Commit sugerido:** `feat(wos): edicao progressiva de especificacao e participantes`.

**Implementação.** Expor setters legítimos de owners/assignees e implementar edição de execution_spec e not_before quando ausente. Definir limpar versus manter, versões e congelamento. Reutilizar campos atuais e não criar assignments paralelos.

**Pontos de alteração.** application/update.go, lifecycle_commands.go, setters específicos, guards, schemas.

**Validação.** Criar só título e completar depois; listas vazias limpam explicitamente; referências de outro Outcome rejeitadas; contrato ativo/revisão aberta não alterados; responsáveis não recebem lease.

**Aceite.** Todo campo adiado pela criação rápida pode ser preenchido depois quando o domínio permite.

#### WOS-09 — Reorganização persistente e dependências

**Depende de:** WOS-05, WOS-08.  
**Commit sugerido:** `feat(wos): reassociacao de trabalho e hierarquia validada`.

**Implementação.** Implementar SetWorkItemObjective e SetObjectiveParent por handler registrado, não apenas por nome previsto num guard. Validar versões, escopo, ciclos, destino, subárvore afetada, bloqueios herdados e autoridade. Cobrir a incompatibilidade da Task com dependência hard do próprio Objective.

**Pontos de alteração.** domain/application de estrutura, refs/relations, guards, índices e eventos.

**Validação.** Mover Task preserva ID/histórico; null remove associação; ciclos rejeitados; movimentos concorrentes; subárvore em execução; configurações legadas inválidas diagnosticadas sem limpeza automática.

**Aceite.** Estrutura muda somente por intenção explícita e os impactos são verificáveis.

#### WOS-10 — Operações documentais compostas

**Depende de:** WOS-06, WOS-05.  
**Commit sugerido:** `feat(wos): evidencias e registros contextuais atomicos`.

**Implementação.** Implementar registro de evidência com vínculos e Artifact/observação interna opcional em uma UnitOfWork. Cobrir reuso de Evidence existente e critério explicitamente escolhido. Adicionar registros contextuais de Artifact/Decision com relações canônicas; reutilizar Issue+Blocker existentes.

**Pontos de alteração.** application/documentary_records.go, primitivos/compostos, Issue/Blocker, pipeline e schemas.

**Validação.** Rollback de vínculo inválido; source interna verificada; Evidence retirada; versões de critério; Decision como target válido e Blocker inválido; relato humano não apresentado como teste do servidor.

**Aceite.** Uma intenção contextual cria os registros corretos, mantém autoria e não aprova critérios.

#### WOS-11 — Timeline coerente e paginada

**Depende de:** WOS-06, WOS-10.  
**Commit sugerido:** `feat(wos): timeline coerente com fontes e cursores`.

**Implementação.** Criar projeção de eventos, notas e WorkActivity com ordem estável, fronteira histórica e origem explícita. Deduplique projeções do mesmo registro, não fatos distintos. Incluir referências e entregas sem copiar payloads completos.

**Pontos de alteração.** application queries, ports de consulta, adapters/índices e API de timeline.

**Validação.** Vários eventos na mesma revisão, novo evento durante paginação, retry de leitura, alteração de principal, histórico parcial/legado, evento mais registro sem duplicação.

**Aceite.** Timeline tem cursor estável e não mistura afirmação, evidência, submissão, avaliação e conclusão.

#### WOS-12 — EntityWorkspaceView e capabilities

**Depende de:** WOS-05, WOS-08, WOS-09, WOS-10, WOS-11.  
**Commit sugerido:** `feat(wos): projecoes operacionais por entidade`.

**Implementação.** Implementar facetas tipadas e leitura sob contexto transacional único. Reutilizar WorkContext/Attention e vinculação de assessments à entrega/digest. Retornar motivos e canais das ações, pessoas com papel/origem, pendências e cobertura.

**Pontos de alteração.** application focal_current/work_attention/novas workspace views, ports, API e DTOs.

**Validação.** Coerência de revisão entre facetas; assinatura independente; desconhecido não vira vazio; nome/ID não vazam; false-positive de met em entrega antiga é rejeitado.

**Aceite.** Inspector recebe estado coerente e não precisa derivar gates de negócio em JavaScript.

#### WOS-13 — Resumo de cards em lote

**Depende de:** WOS-12.  
**Commit sugerido:** `feat(wos): resumos de mapa em lote`.

**Implementação.** Adicionar resumo estrutural paginado/em lote, com critério, participante e atenção pertinentes. Indexar consultas; evitar carregar domínio inteiro ou um inspector por card. Distinguir contadores completos de relações apenas carregadas.

**Pontos de alteração.** application query/progress, adapters, API e canvas-data.js.

**Validação.** Lote de 100 refs, mesmo scope/revisão, dados grandes, consultas e bytes medidos, authority unknown em vez de identidade inventada, contagem sem evidência duplicada.

**Aceite.** Resumo útil com custo medido, sem N+1 de hidratação e sem inferir total a partir da página.

#### WOS-14 — Inspector comum e navegação de contexto

**Depende de:** WOS-12.  
**Commit sugerido:** `feat(wos): inspector operacional compartilhado`.

**Implementação.** Extrair shell de inspector compartilhado no canvas e views especializadas. Topo claro, contexto, resumo e seções por tipo. Remover duplicação de leitura/propriedades técnicas; permitir abrir Criterion mantendo o proprietário e voltar com foco.

**Pontos de alteração.** app.js, workspace-inspector.js, novos módulos de view, CSS e rotas.

**Validação.** Task/Objective/Outcome/Evidence/Issue/Artifact/Decision/Roadmap/relações; link direto; back/forward; abertura concorrente; formulário preservado e invalidado corretamente no scope.

**Aceite.** Todas as entidades têm contexto legível, sem uma segunda implementação do detalhe por view.

#### WOS-15 — Critérios, comprovação e notas no inspector

**Depende de:** WOS-10, WOS-11, WOS-14.  
**Commit sugerido:** `feat(wos): criterios evidencias e atualizacoes no contexto`.

**Implementação.** Renderizar critérios atuais com avaliações aplicáveis e evidências; implementar formulário contextual de Evidence e nota. Mostrar cinco grupos essenciais na Task com prévias recentes, ações locais e estados vazios honestos.

**Pontos de alteração.** workspace-criteria.js, workspace-timeline.js, human-forms.js, view/copy.

**Validação.** Adicionar evidência não atende critério; nota não conclui; revisão/digest desatualizado; contradição; origem interna/externa; troca rápida de item não injeta conteúdo antigo.

**Aceite.** O usuário encontra critérios, evidências, participação, histórico e pendências dentro da Task.

#### WOS-16 — Criação rápida e editor de estrutura

**Depende de:** WOS-07, WOS-08, WOS-14.  
**Commit sugerido:** `feat(wos): criacao rapida e montagem progressiva do mapa`.

**Implementação.** Criar formulários inline com parent fixado, defaults visíveis e expansão de detalhes. Suportar criação repetida por teclado e lote com prévia. Atualizar mapa/seleção a partir de refs reais retornadas pelo servidor.

**Pontos de alteração.** workspace-quick-create.js, human-forms.js, canvas callbacks e copy.

**Validação.** Outcome+2 Objectives+4 Tasks; enter/escape; erro mantém draft; required_for_outcome explícito; backlog não disponível a agentes; completar especificação após salvar.

**Aceite.** Mapa criado sem IDs, pesquisa repetida de pai ou navegação compulsória para outras views.

#### WOS-17 — Fluxos de problemas e registros relacionados

**Depende de:** WOS-10, WOS-14.  
**Commit sugerido:** `feat(wos): problemas e bloqueios no item afetado`.

**Implementação.** Adicionar Report issue em Task/Objective/Outcome, com affected target preenchido e pergunta de impedimento. Resolver Issue mostrando impactos selecionados. Registrar entregável/decisão no item sem confundir produção, contexto e comprovação.

**Pontos de alteração.** human-forms/actions, workspace-flow, Issue/Blocker inspector.

**Validação.** Issue sem bloquear; compound bloqueia alvo correto; Task sem subtree inválido; Issue terminal com bloqueio exige reabertura; resolver sem liberar conserva impedimento.

**Aceite.** Problemas e seus impactos podem ser operados no contexto sem exigir conhecimento de nomes de comandos.

#### WOS-18 — Ação principal, protocolo e conclusão guiada

**Depende de:** WOS-12, WOS-14, WOS-15, WOS-17.  
**Commit sugerido:** `feat(wos): proximas acoes e conclusoes guiadas`.

**Implementação.** Substituir rodapé de comandos indiferenciados por ação principal, secundárias e menu. Mostrar indisponibilidades explicáveis e caminho de perfil assinado. Concluir Objective/Outcome com checklist de condições e impacto, sem clique de aprovação automático.

**Pontos de alteração.** actions.js, workspace-action-menu.js, contracts-view.js, workspace-flow/copy.

**Validação.** Matriz de papel×protocolo×estado; planned→achieved elegível; waiver explícita; owner não vira executor; material recebido não é aprovado; terminal deixa trabalho aberto identificado.

**Aceite.** Cada estágio tem uma próxima ação compreensível e legítima; o browser não adquire assinatura inexistente.

#### WOS-19 — Relações e movimentos na interface

**Depende de:** WOS-09, WOS-14, WOS-18.  
**Commit sugerido:** `feat(wos): relacoes explicitas e reorganizacao do mapa`.

**Implementação.** Substituir Connect por dependência explícita, direção e força. Adicionar Move to Objective/Parent com preview, confirmação e versões. Manter drag livre local; eventual gesto de associação tem modo explícito e fallback por teclado.

**Pontos de alteração.** workspace-canvas.js, canvas-data.js, formulários de relação e inspector.

**Validação.** Mover papel não muda pai; reassociar persiste; conflito não descarta texto; restrições explicadas; hard e advisory distinguíveis; seleção de aresta abre endpoints reais.

**Aceite.** O usuário consegue montar e corrigir a hierarquia sem criar dependências acidentalmente.

#### WOS-20 — Mapa principal, cards e camadas

**Depende de:** WOS-13, WOS-16, WOS-19.  
**Commit sugerido:** `feat(wos): canvas de trabalho com camadas contextuais`.

**Implementação.** Priorizar Outcome/Objectives/Tasks, introduzir indicadores semânticos e expansão contextual de provas/problemas. Trabalhar com zoom legível, ramo em foco e lista equivalente. Preservar arte original, pan, posições negativas e estados parciais.

**Pontos de alteração.** workspace-canvas.js, canvas-data.js, canvas.css/editorial tokens.

**Validação.** 100/1000 Tasks com colapso; auto-fit não obrigatório; não perder câmera; contadores honestos; sem posts duplicados para mesma Evidence; busca encontra registros fora da página.

**Aceite.** Mapa legível e explorável, sem transformar todas as entidades em uma árvore falsa.

#### WOS-21 — Responsividade, teclado e acabamento

**Depende de:** WOS-20, WOS-15, WOS-17, WOS-18.  
**Commit sugerido:** `feat(wos): refluxo responsivo e navegacao por teclado`.

**Implementação.** Ajustar inspector, forms e action bar para HD, QHD, ultrawide, mobile e baixa altura. Preferir rolagem de leitura clara e controles alcançáveis. Tratar nomes longos, erro com foco, zoom nativo, texto ampliado e redução de movimento.

**Pontos de alteração.** CSS existentes, inspector/canvas/form components e testes de navegador.

**Validação.** Matriz de viewports, foco e teclado; não cortar assets; contraste nos tokens preservados; tab order; modais vs painel não modal; screenshots vinculados à revisão.

**Aceite.** Fluxos principais utilizáveis por teclado e sem clipping em HD/mobile, mantendo a identidade.

#### WOS-22 — Paridade de API, CLI, MCP e compatibilidade

**Depende de:** WOS-07, WOS-08, WOS-09, WOS-10, WOS-12, WOS-13.  
**Commit sugerido:** `feat(wos): paridade de contratos e consumidores`.

**Implementação.** Fechar auditoria dos registros adicionados incrementalmente: schemas, inventário, geração, exemplos, permission labels, classificações de transporte e handoffs. Não esperar esta etapa para proteger comandos já expostos.

**Pontos de alteração.** API schemas/inventory, adaptadores CLI/MCP existentes, docs e testes de contrato.

**Validação.** Clientes antigos sem campos novos, novos defaults só na UI, string de revisão grande, perfil assinado e recuperação, handlers novos cobertos pelo guard.

**Aceite.** Uma única fonte de regras para humanos e agentes; nenhum cliente antigo depende do novo inspector.

#### WOS-23 — Qualificação integrada e usabilidade

**Depende de:** WOS-21, WOS-22.  
**Commit sugerido:** `test(wos): qualificacao integrada e aceite de usabilidade`.

**Implementação.** Executar a matriz completa em adapters suportados, concorrência, browser e fixtures grandes. Medir latência, bytes/consultas e compreensão das tarefas de usuário. Corrigir falhas antes do aceite, sem transformar pendências em pass.

**Pontos de alteração.** Suites Go/browser/compatibilidade, migrations, métricas e relatório de sessões.

**Validação.** Todas as jornadas do documento 04; upgrade do banco existente; passagem entre atores; exploração de mapa parcial; back/forward; incerteza após commit; sessões de usuários quando disponíveis.

**Aceite.** Relatório identifica exatamente revisão, ambiente, testes pass/fail/pending e resultados de compreensão, sem benchmark fictício.

#### WOS-24 — Consolidação do PR e documentação final

**Depende de:** WOS-23.  
**Commit sugerido:** `docs(wos): documentar e consolidar o escopo do PR 116`.

**Implementação.** Atualizar workspace guides, relationship guide, changelog e descrição do PR para o escopo real. Remover renderizadores e caminhos obsoletos, explicar compatibilidade/migrations e anexar capturas atuais. Seguir versionamento coordenado do repo sem publicar.

**Pontos de alteração.** docs/workspace-ux.md, workspace-relationships.md, ADRs, release metadata e PR existente.

**Validação.** Links/doc exemplos válidos; nenhuma promessa de API-only anterior contraditória; build/suites finais no mesmo HEAD; arquivos gerados atualizados e limpos.

**Aceite.** PR permanece draft até decisão do responsável, com commits verificáveis e sem merge ou release automática.


---

## Parte 4 — Matriz de testes e critérios de aceite


Os cenários abaixo especificam o que deve ser implementado e verificado. Não estão marcados como executados. Cada suíte deve registrar revisão do código, adapter, protocolo, identidade, fixtures, ambiente e resultado.

### 1. Estratégia de validação

Testar domínio e application antes da UI; transporte e schemas antes dos fluxos de produção; componentes com dados reais e jornadas de browser antes do aceite visual. Cada commit funcional inclui regressões relacionadas. Dados simulados podem desenvolver UI, mas não qualificam permissões, idempotência ou integração.

A suíte de caracterização preserva os comportamentos corretos anteriores e reproduz falhas encontradas. Depois das correções, os mesmos testes passam a comprovar a nova regra. Não apagar uma regressão porque revela uma restrição de contrato inconveniente.

### 2. Fixtures obrigatórias

| Fixture | Conteúdo |
| --- | --- |
| F01 — Planejamento mínimo | Outcome draft; sem critérios; criação contextual vazia |
| F02 — Mapa pequeno completo | 1 Outcome, 2 Objectives e 4 Tasks; critério próprio em cada nível pertinente |
| F03 — Comprovação | Critério obrigatório/opcional/retirado, modos distintos, Evidence suporte/contradição/contexto e waiver |
| F04 — Concorrência | Dois usuários, um agente, leases/contratos e writes simultâneos |
| F05 — Revisão assinada | Entrega recebida, revisão pendente, in_review, corrections_requested, approved e material antigo |
| F06 — Problemas | Issue afeta sem bloquear; múltiplos Blockers; efeitos diretos/herdados; Issue resolvida com impedimento ativo |
| F07 — Estrutura | Subobjetivos, Tasks diretas no Outcome, ciclos tentados, reassociação e dependência incompatível |
| F08 — Volume | 100 Objectives, 1.000 Tasks, 5.000 Evidence e 10.000 atividades/notas/eventos, distribuídos e com índices |
| F09 — Dados anteriores | Banco da revisão auditada, sem notas novas, sem novos defaults aplicados, histórico parcial conhecido |
| F10 — Fronteiras | Texto longo, refs sem nome carregado, permissão revogada, Outcome arquivado/terminal e leitura truncada |

Executar variantes de protocolos realmente suportados na branch, com identidades separadas de executor e revisor. Não criar revisão independente usando a mesma identidade apenas para facilitar o teste.

### 3. Cenários funcionais e de contrato

#### A — Criação progressiva (WOS-07, WOS-08, WOS-16)

| ID | Cenário e resultado exigido |
| --- | --- |
| T001 | Criar Outcome com nome e estado desejado: permanece draft e não ganha critério inventado. |
| T002 | Salvar Outcome e critérios iniciais: nenhum estado intermediário parcialmente persistido. |
| T003 | Criar Objective no Outcome: vínculo e default obrigatório visíveis e corretos. |
| T004 | Criar subobjetivo: pai persistido correto; obrigatoriedade refere-se ao Outcome, não a uma herança fictícia. |
| T005 | Tornar novo Objective opcional: flag false realmente enviada e respeitada. |
| T006 | Criar Task no Objective apenas com título: backlog, vínculo correto, nenhuma aquisição. |
| T007 | Criar Task diretamente no Outcome: associação null e leitura explícita, sem objetivo artificial. |
| T008 | Disponibilizar Task: passa a todo por intenção explícita; não concede lease automaticamente. |
| T009 | Adicionar instruções, constraints, deliverables e referências depois de criar: persistência e leitura corretas. |
| T010 | Limpar lista/data/Objective: distinguir clear explícito de campo omitido. |
| T011 | Lote de novos itens com local keys: IDs retornados estáveis e vínculos válidos entre itens novos. |
| T012 | Lote com último item inválido: rollback de todos os itens e eventos; repetição não deixa órfãos. |
| T013 | Chaves duplicadas, ciclos e pai local ausente: erro localizado antes de salvar, sem inferência silenciosa. |
| T014 | Limites de lote/bytes/critério: rejeição previsível no boundary, sem exceção ou write parcial. |
| T015 | Enter cria e permite continuar; Escape não salva; falha mantém texto e contexto. |

#### B — Critérios e comprovação (WOS-05, WOS-10, WOS-12, WOS-15)

| ID | Cenário e resultado exigido |
| --- | --- |
| T016 | Outcome só com critério opcional não está pronto para ativar. |
| T017 | Critério obrigatório retired não satisfaz preparação. |
| T018 | Critério obrigatório ativo ainda sem assessment permite preparação onde a regra pede definição, não prova. |
| T019 | Registrar Evidence já no Criterion: owner/id corretos, revisão observada validada e avaliação continua pendente. |
| T020 | Mesmo documento em dois alvos: uma Evidence, vínculos distintos, stances independentes. |
| T021 | Evidence contraditória: não é ocultada por filtro de suporte e não vira aprovação. |
| T022 | Evidence retraída: não aceita novo vínculo e não satisfaz comprovação vigente indevidamente. |
| T023 | Assessment da revisão antiga: aparece no histórico, não atende critério revisado. |
| T024 | Assessment de submissão/digest anterior: não aprova nova entrega mesmo com o mesmo criterion ID/revision. |
| T025 | Waiver autorizada: resultado explícito, motivo preservado e contador separado de comprovados. |
| T026 | Usuário sem waiver permission: não recebe nem consegue executar dispensa. |
| T027 | Modos attestation/evidence_review/external_evaluation: formulário e validação correspondem ao modo real. |
| T028 | Target Decision permitido; target Blocker proibido; Issue/Decision não recebem criterion_id. |
| T029 | Registros antigos com links incompletos: mostrar ausência de contexto conhecido, não inventar avaliação. |
| T030 | Evidência de observação interna: origem resolvível, autor real, sem alegar execução de teste pelo servidor. |
| T031 | URL de Artifact/Evidence: não buscar conteúdo no servidor nem executar dados como código. |
| T032 | Transação de Evidence com vínculo inválido: nenhum Artifact, nota ou Evidence parcial é criado. |
| T033 | Critério retirado/revisado enquanto formulário aberto: nova decisão exigida, entradas preservadas. |

#### C — Participação, execução e aceitação (WOS-05, WOS-08, WOS-12, WOS-18)

| ID | Cenário e resultado exigido |
| --- | --- |
| T034 | Owner/assignee atribuído: não se torna automaticamente executor, autenticado ou autorizado. |
| T035 | Contrato vigente de outro principal: UI não oferece iniciar como se estivesse livre. |
| T036 | Contrato expirado: não mostrar atividade física presumida; oferecer recuperação legítima quando suportada. |
| T037 | Resumo assinado sem autoridade suficiente: mostrar desconhecido, não usar expiração do lease legado. |
| T038 | Revisor corrente e executor anterior: papéis distintos e provenientes dos contratos/casos corretos. |
| T039 | Entrega recebida: não aparece como aprovada ou concluída por mero recebimento. |
| T040 | Correções solicitadas: achados atuais aparecem antes de sugerir nova conclusão. |
| T041 | Execução assinada: browser não recebe chave privada, não assina e não faz fallback para claim legado. |
| T042 | Snippet de perfil: alvo, versão e escopo corretos, sem interpolar shell não confiável. |
| T043 | Edição de especificação/criteria/estrutura congelada: bloqueada no servidor, não apenas escondida na UI. |
| T044 | Nova mutação registrada: coberta por autorização, guard, replay e event factory; nenhum bypass por tipo novo. |
| T045 | Objective planned já comprovado: conclusão canônica permitida sem Start artificial. |
| T046 | Todas as Tasks done, critério do Objective pendente: não marcar Objective alcançado. |
| T047 | Todos os Objectives alcançados, critério próprio do Outcome pendente: não concluir Outcome. |
| T048 | Objective opcional incompleto: não transformá-lo em gate obrigatório por posição na árvore. |
| T049 | Concluir com trabalho aberto: explicitar consequência sobre o escopo, sem cancelamento/conclusão dos filhos. |
| T050 | Reabertura: histórico preservado e estado/revisões atuais usados na próxima ação. |

#### D — Atualizações e timeline (WOS-06, WOS-11, WOS-15)

| ID | Cenário e resultado exigido |
| --- | --- |
| T051 | Nota “terminei a Task”: aparece como declaração, sem transição de lifecycle. |
| T052 | Nota “encontrei problema”: não cria Issue/Blocker sem intenção explícita. |
| T053 | AppendWorkActivity sem autoridade assinada válida: continua rejeitado. |
| T054 | Correção de nota do próprio autor: nova nota, mesmo contexto; conteúdo original preservado. |
| T055 | Tentar sobrescrever nota de outro autor: rejeitado; novo comentário não falsifica autoria. |
| T056 | Nota em item terminal não arquivado: documental permitida; novo trabalho operacional continua protegido. |
| T057 | Nota em Outcome arquivado: comportamento de bloqueio consistente com as regras do escopo. |
| T058 | Timeline carregada ao abrir item: primeiras entradas acessíveis sem caçar botão de cobertura. |
| T059 | WorkActivity e seu evento: uma entrada documental representativa, sem duplicação enganosa. |
| T060 | Vários fatos na mesma revisão: cursor não pula nem repete entradas. |
| T061 | Novo evento durante paginação: fronteira congelada e aviso de novidades, sem mistura invisível. |
| T062 | Atividade do descendente no pai: origem e caminho identificados; não atribuída ao pai. |
| T063 | Datas de observação diferentes da ordem de registro: ambas corretas, sem cronologia inventada. |

#### E — Problemas, vínculos e reorganização (WOS-09, WOS-10, WOS-17, WOS-19)

| ID | Cenário e resultado exigido |
| --- | --- |
| T064 | Issue criada na Task: affected_refs preenchido; nenhuma pesquisa repetida do alvo. |
| T065 | Issue sem “impede continuar”: não altera prontidão por si só. |
| T066 | Issue com impedimento: comando composto cria os dois registros com target correto. |
| T067 | Propagação incompatível com Task/Outcome: opção não oferecida e API rejeita entrada forjada. |
| T068 | Resolver Issue sem liberar Blocker: trabalho continua impedido e isso fica explícito. |
| T069 | Liberar um de vários Blockers: os restantes continuam aplicáveis. |
| T070 | Problema após conclusão: Issue documental possível, Blocker operacional exige reabertura conforme domínio. |
| T071 | Mover papel livremente: não envia comando de domínio nem altera parent/Objective. |
| T072 | Reassociar Task: preserva ID, critérios, evidências e histórico, atualiza vínculo atual. |
| T073 | Destino em outro Outcome/namespace: rejeitado antes de gravar; nenhuma informação privada revelada. |
| T074 | Ciclo de Objective: rejeitado, inclusive em operações concorrentes. |
| T075 | Movimento com subárvore em execução ou revisão aberta: motivo claro e nenhuma revogação implícita. |
| T076 | Dependência hard da Task em seu próprio Objective: reproduzir incompatibilidade; rejeitar criação/reassociação que a introduza. |
| T077 | Relação advisory: continua informativa, sem bloqueio operacional inventado. |
| T078 | Preview de impacto fica obsoleto: commit detecta nova situação e pede revisão, não usa preview como autorização. |
| T079 | Artifact produzido versus apenas relacionado: tipo de Relation correto, sem EvidenceLink indevido. |
| T080 | Decision aceita substitui outra: inspector mostra cadeia real; Blocker associado não se libera por inferência. |

#### F — Concorrência, autorização e recuperação (WOS-04, WOS-05, WOS-12, WOS-22)

| ID | Cenário e resultado exigido |
| --- | --- |
| T081 | Falta de critério: precondition correto, ação adicionar, sem mensagem de edição concorrente. |
| T082 | Conflito real: versão base/local/atual disponíveis e nenhuma perda do rascunho. |
| T083 | Timeout após commit de criação composta: retry retorna IDs originais, sem duplicatas. |
| T084 | Write confirmado e refresh falho: não oferecer repetir a mutação como nova intenção. |
| T085 | Duas criações/avaliações concorrentes: guard e versões preservam invariantes do Outcome. |
| T086 | Revogação de permissão durante formulário/query: nenhuma gravação ou conteúdo antigo reaplicado. |
| T087 | Mudança de scope ou identidade com resposta atrasada: geração antiga descartada. |
| T088 | Versão maior que Number.MAX_SAFE_INTEGER: comparação exata, cursor e payload preservados. |
| T089 | Facetas de revisões diferentes: view não publicada como coerente; leitura/refetch controlado. |
| T090 | Capability disponível seguida de mudança: comando revalida e pode recusar corretamente. |
| T091 | Referência maliciosa, ActorRef não autorizado ou source interna falsa: não ganha autoridade/procedência. |
| T092 | Leitura de página parcial: ausência de item/evidência não vira total zero ou prontidão verdadeira. |

#### G — Interface, desempenho e compatibilidade (WOS-13 a WOS-24)

| ID | Cenário e resultado exigido |
| --- | --- |
| T093 | Task revela critérios, evidências, pessoas, histórico e pendências no resumo sem navegação global. |
| T094 | Criterion/Evidence/Issue/Roadmap têm facetas pertinentes; não recebem executor ou critérios fictícios. |
| T095 | Abrir, voltar, copiar deep link e recarregar: sujeito correto e retorno ao contexto preservado. |
| T096 | Adicionar informação não reorganiza arbitrariamente o mapa nem perde câmera/seleção. |
| T097 | Busca server-backed encontra item fora do mapa carregado e indica seu contexto real. |
| T098 | Mapa grande usa lote/paginação; não faz uma sequência de chamadas completas por card. |
| T099 | Ramo colapsado/fit explícito/zoom de trabalho: títulos e controles continuam compreensíveis. |
| T100 | HD e telas baixas: mapa continua central, inspector não fica abaixo de uma página enorme. |
| T101 | Mobile com teclado: salvar, cancelar e erro permanecem alcançáveis. |
| T102 | Teclado sem drag: criar, abrir, mover por comando, avaliar, voltar e fechar são operáveis. |
| T103 | Zoom nativo/texto aumentado: sem títulos cortados, botões microscópicos ou foco perdido. |
| T104 | Assets originais: sem esticar papel, cortar robô, cobrir texto com dobra ou substituir identidade. |
| T105 | Upgrade de banco anterior: dados/contratos intactos, notas funcionam e nenhum default antigo muda. |
| T106 | Clientes antigos sem campos opcionais novos: continuam válidos e respeitam seus defaults. |
| T107 | Exposição CLI/MCP/inventário: permissões e semântica equivalentes à API, sem endpoint humano privilegiado. |
| T108 | Relatório final identifica pass/fail/pending e ambiente; não confunde teste automatizado com compreensão humana. |

### 4. Matriz de telas e acessibilidade

Validar pelo menos: 1280×720, 1366×768, 1920×1080, 2560×1440, 3840×2160; mobile de 320, 390 e 430 CSS px, incluindo altura reduzida e teclado aberto. Preservar cenários extremos já cobertos pela branch, em vez de substituí-los pela nova matriz mínima.

Testar zoom nativo de 125%, 150% e 200% e aumento de texto; DPR e viewport equivalente não substituem zoom nativo. Card, relação e ação precisam de nome acessível. Foco deve voltar à origem correta. Painel não modal não recebe focus trap de modal; modal verdadeiro deve ter semântica e fechamento consistentes.

A arte editorial pode continuar decorativa, sem duplicar o texto acessível. Cor não é o único sinal de tipo, estado, bloqueio, obrigatoriedade ou stance. Não declarar conformidade formal com uma norma apenas porque testes automatizados de acessibilidade passaram.

### 5. Metas de desempenho — propostas para medição

Ambiente de referência inicial: backend com 2 vCPU/4 GiB, storage SSD, navegador desktop, rede controlada com RTT de 50 ms, até 10 sessões simultâneas e fixture F08. Registrar versões, adapter, índices e estado de cache.

Alvos iniciais: resposta p95 de inspector/resumo em até 400 ms no ambiente descrito; primeiro mapa utilizável em até 2 s; feedback local de seleção em até 100 ms, sem esperar a rede para indicar carregamento. São metas de engenharia a medir, não números observados ou SLA do produto.

Hard gates: ausência de fan-out HTTP/SQL por card para hidratação completa; leitura e bytes limitados; paginação indexada da timeline; sem congelamento perceptível ao selecionar ramo; sem buscar todo o histórico para exibir cinco acontecimentos. Quando uma validação de grafo exige busca, medir o trabalho e respeitar limites em vez de afirmar custo constante.

Facetas extensas retornam previews, páginas e truncamento declarado. Não enviar vinte notas de 16 KiB na primeira tela. Definir limite de preview e preservar acesso ao registro completo autorizado.

### 6. Teste de compreensão humana

Roteiro formativo com pelo menos cinco participantes que não tenham participado do desenho, quando disponíveis. Fornecer o conteúdo dos itens para que o teste meça operação, não o tempo de inventar um plano. Não explicar comandos internos antes da tarefa.

| Tarefa | Meta inicial de aceite |
| --- | --- |
| Criar 1 Outcome, 2 Objectives, 4 Tasks e critérios de preparação | Mediana até 3 minutos; nenhum ID ou console técnico |
| Encontrar as cinco informações da Task | Pelo menos 4/5 participantes em até 30 segundos, sem mudar de view global |
| Adicionar Evidence ao critério correto | Pelo menos 4/5 em até 45 segundos, sem vínculo perdido |
| Reportar problema no item e escolher se bloqueia | Pelo menos 4/5 em até 45 segundos, com alvo/impacto corretos |
| Distinguir responsável, executor e revisor | Nenhum erro que conceda autoridade fictícia ou confunda autoria |
| Distinguir evidência vinculada, critério atendido e entrega aceita | Nenhum erro crítico de entendimento induzido pela UI no roteiro |

Registrar erros, desistências, ajuda recebida e tempo por tarefa. Cinco participantes permitem encontrar falhas, não estimar estatisticamente satisfação de todo o mercado. Se não houver participantes, executar revisão orientada a tarefas com o responsável, mas manter a validação com usuários como pendente.

### 7. Comandos de verificação conhecidos e integração

O `go.mod` da baseline define a toolchain. Usar a versão do projeto, sem upgrade incidental. Os comandos abaixo são para executar durante a implementação; não foram executados por este pacote:

```sh
go test ./...
go test -race ./packages/wos-core/...
go vet ./...
go build -o bin/wos ./packages/wos-api/cmd/wos
(cd tests/web && npm ci && npm test)
```

`tests/web/package.json` define `npm test` como unit tests seguidos de Playwright. Executar também suites de banco real, migração, geração e compatibilidade exatamente como a CI atual prescreve; variáveis de conexão devem vir dos scripts existentes, não de nomes inventados neste plano.

### 8. Definição final de pronto

A entrega técnica está pronta quando as operações têm persistência e autorização reais; os fluxos são completos; testes de domínio, application, adapters, API e browser pertinentes passam; migração e clientes antigos foram verificados; a documentação corresponde ao código; não existem caminhos duplicados de inspector/regra; e o PR descreve seu novo escopo corretamente.

O aceite de UX tem evidências próprias. Screenshots, contagem de testes e walkthrough do autor não são automaticamente aceitação por usuários. Qualquer item pendente deve aparecer no relatório final, junto da decisão explícita do responsável sobre avançar o PR. Nenhuma release ou merge é consequência automática de terminar um commit do plano.


---

## Parte 5 — Instruções para o agente ou equipe implementadora


### 1. Destino e autorização de trabalho

Implementar na branch `fix/workspace-operational-density`, continuando o PR #116. A revisão usada por este plano é `5d945d3081f85927b80c686d695849bd1df73a5d`. Antes de alterar arquivos, conferir o HEAD atual, o estado do working tree, alterações paralelas e instruções próprias do repositório. Comparar o diff posterior à revisão auditada e reconciliar capacidades já implementadas.

Não recriar o PR, não aplicar reset, não apagar arquivos locais não reconhecidos, não fazer force push e não sobrescrever trabalho do autor. Não fazer merge, tag, publicação de pacote ou release como efeito automático deste plano. A implementação deve permanecer revisável em commits incrementais.

O README deste pacote é um índice do plano: **não o copiar sobre o README.md do repositório WOS**. Documentar comportamento operacional nos guias/ADRs pertinentes, sem reescrever marketing ou identidade do produto como tarefa incidental.

### 2. Ordem de leitura

Ler `01-PLANO-DE-IMPLEMENTACAO.md`, `02-CONTRATOS-E-DADOS.md`, `03-BACKLOG-E-COMMITS.md`, `04-TESTES-E-ACEITE.md` e `06-FONTES-E-REVISAO.md`. `BACKLOG.json` contém as mesmas 24 unidades com dependências; seus status começam em `planned`.

Distinguir três categorias: comportamento auditado da baseline; extensão definida por este plano; meta ainda a medir. Nomes novos sugeridos para módulos, queries e comandos não são prova de que já existem. Nomes de setters mencionados em guards não comprovam que há handler/API correspondente.

### 3. Regras de construção

Preservar o stack e a arquitetura do projeto. Reutilizar componentes, tipos, transações, permissões, schemas e fontes de dados antes de criar outro mecanismo. Nunca criar “dashboard core”, segunda tabela de Tasks, segundo checklist de pending ou API privilegiada de UI.

Adicionar testes junto de cada alteração. Toda nova mutação entra no pipeline de autorização/idempotência/guard/eventos; não adiar isso para a auditoria final. Todos os novos campos adiados no formulário precisam ter edição posterior real. Toda ação visível precisa de backend/autoridade correspondente.

Não marcar a Task como concluída por nota, Evidence, Artifact ou entrega recebida. Não tratar assignee como executor. Não aproveitar o principal humano para assinar em nome do host. Não preencher UUID, credencial, critério ou prova com dados artificiais para atravessar validação.

Mantenha `AppendWorkActivity` vinculado à autoridade contratada. Use nota documental para atualizações no workspace. Não criar uploads, automações, presença em tempo real, compiladores de workflow ou executores de agentes neste escopo.

### 4. Primeira fatia vertical obrigatória

Entregar primeiro um caminho real: criar Task em backlog dentro de um Objective; completar especificação depois; abrir o inspector; registrar uma nota; cadastrar critério; adicionar evidência no critério; verificar que a avaliação continua pendente. Esse caminho deve funcionar com API e persistência reais, refresh e outro cliente lendo os registros.

A fatia não autoriza ignorar as dependências de migrations/pipeline. Ela é um ponto de integração precoce, não um protótipo que simula sucesso no navegador.

### 5. Contratos e geração

Descobrir e registrar os geradores, schemas, enums, rotas, inventários e adapters reais no início. Propagar extensões por todos os consumidores necessários. Nos exemplos deste pacote, IDs são fictícios e modelos novos são conceituais: não copiá-los como payload de uma API antiga.

Não mudar formatos de versão/revisão de contratos existentes sem análise de compatibilidade. Preservar precisão no browser. Queries compostas devem compartilhar um snapshot coerente; writes compostos devem compartilhar uma UnitOfWork.

### 6. Critérios de revisão por commit

Registrar no commit ou relatório: unidade WOS-XX atendida, arquivos alterados, comportamento visível, contrato afetado, testes executados e pendências reais. Se um teste não rodou por ambiente, registrar “não executado”, não “passou”. Se uma capacidade já surgiu no HEAD, adaptar o plano e reutilizá-la em vez de duplicá-la.

Manter o build e os testes pertinentes verdes a cada incremento. Não mover lógica entre módulos apenas para evitar uma regressão que revela condição de domínio. Não ampliar escopo com mudança de framework, identidade visual ou protocolo.

### 7. Relatório final esperado

O relatório de conclusão deve apontar SHA final, lista de commits, unidades atendidas, migrations, compatibilidade, comandos/schemas atualizados, resultados de teste por camada, métricas observadas, capturas por viewport e situação da avaliação humana.

Atualizar a descrição do PR para remover a afirmação anterior de que não há mudanças de API/domínio/persistência. Documentar que registro humano é documental e que execução assinada continua pelo perfil autorizado. Seguir o mecanismo de versionamento existente, sem escolher uma versão ou publicar só porque a baseline tinha uma release planejada.

A conclusão é uma decisão explícita do responsável pelo PR. Não usar a quantidade de commits, testes ou screenshots como substituto do aceite dos fluxos do usuário.


---

## Parte 6 — Fontes, revisão e grau de confirmação


### Base da análise

Revisão fixa: `5d945d3081f85927b80c686d695849bd1df73a5d`. Os links abaixo apontam para conteúdo imutável dessa revisão, exceto o link do PR, que é mutável. A análise é estática, cruzando código e documentos; não é execução da sua instância nem auditoria dinâmica de permissões do seu usuário.

Os testes citados pelo PR são resultados relatados para a baseline do próprio PR. Este pacote não executou a suíte do repositório nem mediu tempo de usuários. Contratos novos e critérios de desempenho neste plano são propostas.

### Índice de fontes

#### S01 — Metadados do PR

Fonte: [https://github.com/A1b3rt0M3rcad0/wos/pull/116](https://github.com/A1b3rt0M3rcad0/wos/pull/116).

Branch, estado draft e revisão da análise. A descrição atual afirma ausência de mudanças de domínio/API/persistência; terá de ser atualizada ao implementar este plano.

#### S02 — Formulários humanos

Fonte: [packages/wos-api/web/assets/human-forms.js](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-api/web/assets/human-forms.js).

Criação, critérios, evidência, problema e preenchimento de referências. Mais campos expostos não significa todos obrigatórios.

#### S03 — Ações e protocolos

Fonte: [packages/wos-api/web/assets/actions.js](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-api/web/assets/actions.js).

Contexts de ações, filtros de lifecycle/permission, operações legadas e bloqueio de ações de critérios de Task sob protocolo assinado.

#### S04 — Detalhe, timeline e erro de conflito

Fonte: [packages/wos-api/web/assets/app.js](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-api/web/assets/app.js).

showEntity, quickActions, loadCanvasHistory, checklist e tratamento que agrupa precondition_failed com version_conflict.

#### S05 — Contexto focal da Task

Fonte: [packages/wos-api/web/assets/canvas-context.js](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-api/web/assets/canvas-context.js).

Instruções, obrigações, bloqueios, links de evidência e carregamento explícito de histórico.

#### S06 — Relações da entidade

Fonte: [packages/wos-api/web/assets/workspace-entity-context.js](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-api/web/assets/workspace-entity-context.js).

Separação de referências registradas e mapa carregado; critérios não são um kind global de card.

#### S07 — Canvas e indicadores

Fonte: [packages/wos-api/web/assets/workspace-canvas.js](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-api/web/assets/workspace-canvas.js).

Representação visual, estrutura, ações Add/Connect, contadores de relações carregadas e posições locais.

#### S08 — Adapter de dados do canvas

Fonte: [packages/wos-api/web/assets/canvas-data.js](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-api/web/assets/canvas-data.js).

Paginação, revisão exata, expansão e fronteiras do grafo; resumo não deve extrapolar autoridade assinada.

#### S09 — Execução e revisão assinadas no browser

Fonte: [packages/wos-api/web/assets/contracts-view.js](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-api/web/assets/contracts-view.js).

Leitura de contrato/submissão/revisão e encaminhamento a perfil; browser não assina pelo host.

#### S10 — Atualização de agregados

Fonte: [packages/wos-core/application/update.go](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-core/application/update.go).

Campos atuais de UpdateOutcome/Objective/WorkItem; associação e execution_spec não fazem parte desse update genérico.

#### S11 — Comandos de lifecycle e participantes

Fonte: [packages/wos-core/application/lifecycle_commands.go](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-core/application/lifecycle_commands.go).

SetOutcomeOwners, SetObjectiveOwners e SetWorkItemAssignees; reutilizar em vez de criar assignment paralelo.

#### S12 — Proteções de contratos

Fonte: [packages/wos-core/application/work_contract_guards.go](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-core/application/work_contract_guards.go).

Guards por tipo/nome; congelamento e revisão aberta. Nomes previstos no guard não comprovam handler registrado.

#### S13 — Atividade sob autoridade assinada

Fonte: [packages/wos-core/application/work_activity.go](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-core/application/work_activity.go).

AppendWorkActivity exige autoridade, credencial, versão, principal e contrato; registros inline e idempotência.

#### S14 — Pendências operacionais

Fonte: [packages/wos-core/application/work_attention.go](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-core/application/work_attention.go).

Obrigações derivadas, cobertura de evidência, revisão/correção e aplicabilidade por criterion revision + submission digest.

#### S15 — Registros e elegibilidade de EvidenceLink

Fonte: [packages/wos-core/application/documentary_records.go](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-core/application/documentary_records.go).

Transações de registro/vínculo e validateEvidenceLinkTarget: Outcome, Objective, WorkItem, Issue, Decision; Blocker não é aceito.

#### S16 — Operações compostas de problemas

Fonte: [packages/wos-core/application/issues_blockers_compound.go](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/packages/wos-core/application/issues_blockers_compound.go).

Criação de Issue+Blocker e resolução de impactos escolhidos; reuso das operações existentes.

#### S17 — Guia operacional humano

Fonte: [docs/workspace-ux.md](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/docs/workspace-ux.md).

Limites de execução assinada, recuperação de intenção, views e ausência de validação de compreensão com usuários na baseline.

#### S18 — Guia de relações

Fonte: [docs/workspace-relationships.md](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/docs/workspace-relationships.md).

Ownership, dependências, comprovação, conclusão explícita e diferença de transição de Objective entre domínio e menu.

#### S19 — Arquitetura canônica

Fonte: [docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/docs/WOS_Design_Arquitetura_Planejamento_Atualizado.md).

Ontologia e invariantes; documentos podem divergir de código: comparar handlers nas decisões de implementação.

#### S20 — Galeria do PR

Fonte: [docs/design/editorial/board-communication/README.md](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/docs/design/editorial/board-communication/README.md).

Capturas e verificações relatadas para revisões identificadas; não tratar como medição de compreensão humana.

#### S21 — Testes do browser

Fonte: [tests/web/package.json](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/tests/web/package.json).

Scripts reais: npm test executa unit e Playwright; test:unit usa node --test.

#### S22 — Módulo Go

Fonte: [go.mod](https://github.com/A1b3rt0M3rcad0/wos/blob/5d945d3081f85927b80c686d695849bd1df73a5d/go.mod).

Toolchain/dependências definidos pelo repositório; não fazer upgrade de bibliotecas como parte incidental desta UX.

### Ajustes aplicados após revisar as propostas anteriores

1. Não confundir um formulário com muitos campos opcionais com a exigência de todos eles. O foco é reduzir decisões prematuras.

2. Não publicar criação com dados mínimos sem implementar edição posterior de especificação e relação estrutural.

3. Não transformar participantes declarados em executor autorizado ou presença online.

4. Não expor AppendWorkActivity no browser removendo seu contrato. Criar nota documental menor e separada.

5. Não representar Criterion como uma entidade global nem forçar critérios em Evidence/Issue.

6. Não pressupor que a lista de EvidenceLink do frontend coincide com o backend. Decision é válido, Blocker não.

7. Não tratar resumo de requisições distintas como uma única leitura coerente. Reutilizar uma UnitOfWork/contexto de query.

8. Não chamar handlers transacionais independentes para uma operação que promete atomicidade.

9. Não aprovar um novo resultado com assessment de outra submissão ou digest, mesmo que a revisão do critério seja a mesma.

10. Não colocar uma Task com título apenas em aquisição automática; criar em backlog e disponibilizar explicitamente.

11. Não alterar dados antigos ao escolher defaults melhores para a nova UI.

12. Não afirmar que a execução assinada se tornou totalmente web sem um mecanismo legítimo de assinatura.

13. Não simplificar o mapa removendo acesso a decisões, artefatos e provas: torná-los contextuais e exploráveis.

14. Não declarar o PR concluído apenas por screenshots e testes de DOM: exigir jornadas, compatibilidade e avaliação humana separada.

### Incertezas limitadas e tratamento na implementação

A combinação Task depende hard do próprio Objective foi identificada por leitura das condições. WOS-03/WOS-09 devem reproduzir o comportamento antes de aplicar a nova rejeição. A lista exata de arquivos de adapters, geradores e suites de compatibilidade será registrada em WOS-01 a partir da árvore atual, sem supor que nomes previstos neste pacote já existem. Esse levantamento não deixa decisões de produto abertas.

Uma mudança no HEAD após esta revisão exige leitura do diff e reconciliação. Não sobrescrever novas capacidades com implementações paralelas só porque o plano as descreveu como faltantes na baseline.
