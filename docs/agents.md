# Operar o WOS com agentes

Pessoas, agentes e aplicações compartilham o estado persistido pelo WOS. O agente executa no próprio ambiente; o WOS registra tarefas, autoridade, entregas e avaliações. Veja o [fluxo ilustrado](../README.md#como-o-wos-opera) antes de configurar a integração.

## 1. Instalar as instruções do agente

Com o executável `wos` disponível:

```sh
wos skill install --agent codex
wos skill list --agent codex
wos skill doctor --agent codex
```

Use `claude`, `hermes` ou `openclaw` para outro ambiente. `--agent all` instala nos quatro layouts; `--agent generic --dir /caminho/skills` seleciona um diretório explícito. `--project /caminho/projeto`, `--global` e `--dry-run` controlam o destino e a prévia.

As cinco skills vêm dentro do binário: coordenação, delegação, continuidade, revisão e workspace. A instalação é offline e preserva arquivos modificados pelo usuário. Ela não inicia agentes, configura MCP, emite credenciais, cadastra chaves ou ativa protocolos. O ambiente do agente pode precisar recarregar as instruções instaladas.

## 2. Conectar uma identidade autorizada

O servidor precisa estar em execução. Cada executor ou revisor utiliza uma identidade com acesso ao Namespace e às operações necessárias. Para um serviço compartilhado, configure autenticação e permissões pelo [guia de operação](operations.md#shared-service).

Para contratos assinados, um administrador prepara a identidade persistente do servidor, o protocolo do Namespace, as permissões e o cadastro das chaves dos agentes. Configure o projeto e os perfis conforme [perfis assinados](signed-workspace-profiles.md). Confira a identidade do servidor e o fingerprint por um canal confiável; use referências protegidas para os segredos.

O executor e o revisor independente precisam de Principals elegíveis distintos. Trocar apenas o nome de um perfil ou a chave não estabelece independência. Instalar uma skill ou criar um perfil não altera o protocolo do Namespace.

## 3. Adquirir e executar trabalho

Com o perfil `executor` já configurado e autorizado:

```sh
wos --profile executor auth status --output json
wos --profile executor doctor --output json
wos --profile executor work checkout --next --count 1 --output json
wos --profile executor work list --output json
wos --profile executor work show CONTRACT_UUID --for-agent --output json
```

Use o identificador retornado pela aquisição no lugar de `CONTRACT_UUID`. O contexto do contrato apresenta instruções, limites, critérios e referências. A ausência de trabalho em uma página limitada não prova que todo o Outcome esteja sem tarefas elegíveis; respeite os cursores e o escopo informados na resposta.

Execute fora do WOS e preencha somente o rascunho de execução permitido em `.wos/profiles/executor/contract/CONTRACT_UUID.yaml`. Preserve a especificação emitida e a intenção assinada. Observe a validade da autoridade; uma cópia local não renova uma aquisição expirada.

```sh
wos --profile executor work finish CONTRACT_UUID --output json
```

A entrega pode ficar pendente de revisão. O sucesso do envio ou de um teste não equivale à aprovação da tarefa, de seus critérios ou do Outcome.

## 4. Revisar a entrega

Um revisor com seu próprio perfil autorizado adquire um contrato de revisão:

```sh
wos --profile reviewer review checkout --next --count 1 --output json
wos --profile reviewer review show REVIEW_CONTRACT_UUID --for-agent --output json

# Examine a submissão e preencha o rascunho de revisão permitido.
wos --profile reviewer review finish REVIEW_CONTRACT_UUID --output json
```

A revisão considera a submissão exata e as revisões de critérios vinculadas. `changes_requested` registra um pedido de correção; a correção exige uma nova aquisição explícita e não altera a entrega histórica. Consulte [retornos e revisão assinados](signed-cli-returns.md).

## 5. Recuperar uma sessão ou resposta interrompida

```sh
wos --profile executor work recover --all-pending --output json
wos --profile reviewer review recover --all-pending --output json
```

A recuperação reconcilia as intenções e os recibos originais. Não troque a chave, o destino, o payload ou a versão esperada para tentar fazer uma escrita passar. Preserve rascunhos e material local ainda não confirmado. Se a autoridade expirou ou foi revogada, siga o resultado atual do servidor e a recuperação documentada; não substitua outra identidade para contornar o vínculo.

Ao retomar um Outcome, releia seu estado e pendências. Concluir uma tarefa não conclui automaticamente um Objective ou Outcome; evidências e avaliações continuam sendo registros distintos.

## Outros caminhos de integração

- [MCP e seus perfis](contracts.md): conexão por Streamable HTTP ou stdio, sem transformar o WOS em executor de ferramentas.
- [HTTP](http.md) e [cliente Go](../examples/remote-client): comandos, consultas e recuperação explícita.
- [Qualificação de clientes](client-qualification.md): limites, paginação, idempotência e recibos.
- [Protocolos legados](../packages/wos-cli/README.md): recuperação de schema 1 e `contracts_v1`; não substituem a autoridade de um Namespace assinado.
- [Workspace humano](workspace-ux.md) e [telas atuais](design/editorial/README.md#main-pages): o mesmo estado acessível pelo navegador.

Para desenvolver ou validar uma integração, consulte [AGENTS.md](../AGENTS.md), [o índice técnico](README.md#desenvolver-e-entender-o-projeto) e o estado de implementação no [ROADMAP](../ROADMAP.md).
