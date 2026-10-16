<p align="center">
  <img src="packages/wos-api/web/assets/brand/hand-drawn-chain/Hand-Drawn%20Orange%20Chain%20Icon.svg" width="144" alt="WOS" />
</p>

# WOS

**Organize o trabalho de pessoas e agentes em torno de um resultado que possa ser verificado.**

O WOS guarda o que precisa ser feito, quem pode trabalhar em cada tarefa, o que foi entregue e como essa entrega foi avaliada. Pessoas, agentes de IA e aplicações consultam o mesmo estado e podem continuar o trabalho em outra sessão.

[Começar pelo navegador](#começar-pelo-navegador) · [Trabalhar com agentes](#trabalhar-com-agentes) · [Documentação](docs/README.md)

![Robô no balcão entregando uma tarefa a outro robô, com três participantes aguardando diante de um quadro.](docs/images/robots-queuing-for-paperwork.png)

## Como o WOS opera

Pense no quadro da ilustração como o estado compartilhado do trabalho. Cada papel representa uma tarefa com instruções e condições de entrega. Um participante solicita trabalho, recebe a autorização correspondente, executa fora do WOS e registra o resultado para revisão.

1. **Defina o resultado.** Crie um **Outcome** com o estado que você quer alcançar e os critérios que demonstram sucesso.
2. **Organize o trabalho.** Adicione **Objectives** para condições intermediárias e **Tasks** para ações concretas. Use um Roadmap quando precisar organizar o plano em revisões.
3. **Execute uma tarefa.** A pessoa ou agente consulta o contexto e adquire a autorização exigida pelo protocolo do workspace. O WOS registra essa aquisição e sua validade.
4. **Registre a entrega.** Associe evidências, artefatos e o que foi realizado à tarefa. Em contratos assinados, a entrega fica vinculada ao contrato e à submissão correspondente.
5. **Revise e conclua.** Um participante autorizado avalia a entrega e os critérios. Se houver pedido de correção, o trabalho continua por uma nova aquisição explícita.
6. **Retome quando precisar.** O próximo participante consulta o estado persistido, as pendências e o histórico antes de continuar.

O WOS mantém esse registro e aplica suas regras de coordenação. A execução dos agentes acontece no seu ambiente: **o servidor não inicia agentes nem executa suas ferramentas**.

**Uma tarefa concluída não conclui automaticamente um Objective ou Outcome. Evidência registrada também não significa aprovação.** Cada resultado tem sua própria avaliação e conclusão explícita.

## Começar pelo navegador

Você precisa de Git e Docker com Compose. A configuração abaixo inicia um ambiente local com SQLite persistente:

```sh
git clone https://github.com/A1b3rt0M3rcad0/wos.git
cd wos
cp .env.example .env
docker compose up --build -d
```

Abra **http://127.0.0.1:8080/app/**. Em **Access credential**, informe o valor de `WOS_BOOTSTRAP_TOKEN` do arquivo `.env` e clique em **Sign in**.

O token de exemplo serve para essa demonstração local. Para compartilhar uma instância, configure credenciais próprias e permissões por participante conforme o [guia de operação](docs/operations.md#shared-service).

### Seu primeiro resultado

Exemplo: criar um fluxo de login que permita entrar no workspace e retomar um resultado salvo.

1. Na sidebar, clique no **+** ao lado de **Outcomes**. Preencha o nome e o estado desejado; **Save draft** salva um rascunho.
2. Adicione critérios de sucesso: o que uma pessoa precisa verificar para aceitar esse resultado. A checklist de preparação ajuda a encontrar as ações.
3. Em **Plan**, adicione os Objectives necessários. Em **Work**, crie as tarefas, suas instruções e os vínculos com Objectives.
4. Quando a preparação estiver pronta, use a ação de ativar o Outcome. Salvar o rascunho não o ativa.
5. Abra uma tarefa para consultar contexto, dependências, impedimentos e ações permitidas. Execute com a autorização mostrada; em um workspace assinado, siga o comando de execução pelo perfil local do agente.
6. Registre a entrega em **Evidence**, avalie os critérios aplicáveis e conclua explicitamente os resultados aceitos.

### Onde encontrar cada coisa

| Área         | Use para                                                          |
| ------------ | ----------------------------------------------------------------- |
| **Overview** | Entender o resultado desejado, o progresso e as pendências        |
| **Plan**     | Organizar Objectives e revisões do Roadmap                        |
| **Work**     | Consultar tarefas na lista ou no quadro e acompanhar sua execução |
| **Issues**   | Registrar problemas e os impedimentos que afetam o trabalho       |
| **Evidence** | Reunir observações, entregáveis e material para avaliação         |
| **Activity** | Consultar os acontecimentos e o histórico do Outcome              |

Em **Choose an action** e nos detalhes dos itens, as ações disponíveis seguem suas permissões e o estado atual. A interface usa nomes em inglês; títulos e descrições podem ser escritos no seu idioma.

No navegador, **Workspace** é o espaço de acesso e permissões chamado **Namespace** na API. Cada Outcome pertence a esse espaço.

Veja o [guia completo do workspace](docs/workspace-ux.md) e as [telas atuais](docs/design/editorial/README.md#main-pages).

## Trabalhar com agentes

O pacote público `wosctl` instala o comando nativo `wos`. O instalador npm exige Node.js 22 ou superior e oferece os clientes Linux e Windows x64:

```sh
npm install -g wosctl@latest
wos version
wos skill install --agent codex
```

Troque `codex` por `claude`, `hermes` ou `openclaw` conforme seu ambiente. Use `--agent all` para os quatro, ou `--agent generic --dir /caminho/skills` para indicar outro diretório. Você também pode baixar os binários nas [releases](https://github.com/A1b3rt0M3rcad0/wos/releases/latest), sem npm, ou [compilar o código Go](docs/operations.md#local-installation).

O comando `wos skill` copia instruções que já vêm dentro do binário. Essa operação funciona offline, preserva alterações locais e não precisa de um servidor WOS em execução.

| Skill              | Papel                                                        |
| ------------------ | ------------------------------------------------------------ |
| `wos-coordination` | Consultar contexto, coordenar trabalho e registrar progresso |
| `wos-delegation`   | Decompor e delegar usando os recursos do ambiente do agente  |
| `wos-continuity`   | Retomar uma sessão ou recuperar uma intenção interrompida    |
| `wos-review`       | Avaliar material e fazer revisão independente                |
| `wos-workspace`    | Operar contratos assinados e sua recuperação pelo CLI        |

**As skills são instruções para o seu agente.** Configure o ambiente de execução, a conexão MCP, as credenciais e o protocolo separadamente.

### Executar um contrato assinado

Primeiro, um administrador configura o protocolo do Namespace, as credenciais, a confiança no servidor e o perfil do executor. Siga [a preparação de perfis assinados](docs/signed-workspace-profiles.md); o nome `executor` abaixo pressupõe um perfil já autorizado.

```sh
wos --profile executor work checkout --next --count 1 --output json
wos --profile executor work show CONTRACT_UUID --for-agent --output json

# Execute fora do WOS e preencha o rascunho de execução autorizado.
wos --profile executor work finish CONTRACT_UUID --output json
```

`CONTRACT_UUID` é o identificador retornado pela aquisição. O contexto mostra instruções, limites e critérios do trabalho. A entrega pode exigir revisão independente; enviá-la não equivale a aprová-la. Cada executor e revisor deve usar sua própria identidade autorizada.

Se uma resposta se perder, reconcilie a intenção original:

```sh
wos --profile executor work recover --all-pending --output json
```

Não recrie a mesma operação com outra chave ou uma versão alterada para forçar sua conclusão. Leia o [guia para agentes](docs/agents.md) para o fluxo e as referências de recuperação.

## Retomar e operar o ambiente

```sh
docker compose logs -f wos   # acompanhar o serviço
docker compose stop         # parar
docker compose start        # retomar os mesmos dados
docker compose down         # remover containers, mantendo os volumes
```

O volume `wos_data` guarda o SQLite. Não use `down -v` se quiser preservar os dados. Ao voltar, entre no mesmo workspace e consulte o Outcome; o histórico e as pendências continuam no servidor. Um link compartilhado ajuda a abrir o contexto, mas não concede acesso.

Para um serviço compartilhado, consulte [autenticação, PostgreSQL, backup e restore](docs/operations.md). O servidor é validado para Linux amd64; o suporte do cliente Windows não implica suporte ao servidor Windows.

<details>
<summary>Executar diretamente a partir do código Go</summary>

Use a versão de Go indicada em [go.mod](go.mod), atualmente 1.27:

```sh
go build -trimpath -o bin/wos ./packages/wos-api/cmd/wos
./bin/wos config validate
./bin/wos server
```

Nesse modo local padrão, abra `http://127.0.0.1:8080/app/`, expanda **Local development server**, informe o Namespace `0199d000-0000-7000-8000-000000000001` e escolha **Open local workspace**. A identidade é local e confiável; para múltiplos usuários, use a configuração autenticada do guia de operação.

O binário inclui o servidor, a aplicação web e as cinco skills. `wosctl` permanece como alias de compatibilidade do cliente.

</details>

## Integrar com sua aplicação

O mesmo estado pode ser usado por estas interfaces:

| Interface          | Referência                                                                    |
| ------------------ | ----------------------------------------------------------------------------- |
| Workspace web      | [Uso pelo navegador](docs/workspace-ux.md)                                    |
| CLI                | [Comandos e protocolos](packages/wos-cli/README.md)                           |
| HTTP               | [API](docs/http.md) e [OpenAPI](packages/wos-api/openapi.yaml)                |
| MCP                | [Perfis e transportes](docs/contracts.md) — `/mcp` quando habilitado          |
| Go                 | [Core embutido](examples/embedded) e [cliente remoto](examples/remote-client) |
| Eventos e webhooks | [Integração com consumidores](docs/integration-events.md)                     |

O WOS funciona sem depender da Woobe ou de um provedor de IA. Agentes, aplicações e ambientes de execução continuam responsáveis por decidir e realizar as ações.

## Documentação e estado do projeto

O código desta branch é a referência para as instruções e telas apresentadas aqui. A versão do código está em [VERSION](VERSION); o pacote `@latest` acompanha a versão publicada no npm. O trabalho para a versão estável 1.0 e os aceites pendentes estão no [ROADMAP](ROADMAP.md).

- [Índice da documentação](docs/README.md): escolha um guia pelo que precisa fazer.
- [Operação do servidor](docs/operations.md): configuração, permissões, persistência e recuperação.
- [Desenvolvimento](docs/README.md#desenvolver-e-entender-o-projeto): arquitetura, contratos e regras do repositório.
- [Release e distribuição](docs/releases.md): plataformas e critérios de publicação.

Licença [Apache 2.0](LICENSE).
