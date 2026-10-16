# Documentação do WOS

Comece pelo [README do projeto](../README.md) para instalar o WOS e percorrer seu primeiro resultado. Escolha abaixo o guia correspondente à sua tarefa.

## Usar o workspace

| Quero…                                                | Leia                                                              |
| ----------------------------------------------------- | ----------------------------------------------------------------- |
| Criar Outcomes, planejar, executar e avaliar trabalho | [Guia do workspace](workspace-ux.md)                              |
| Ver a interface atual em desktop e mobile             | [Galeria e sistema visual](design/editorial/README.md#main-pages) |
| Entender o fluxo de pessoas e agentes                 | [Como o WOS opera](../README.md#como-o-wos-opera)                 |
| Entender a ilustração utilizada no README             | [Origem do arquivo](images/assets.json)                           |

## Conectar agentes e aplicações

| Quero…                                                   | Leia                                                                                         |
| -------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| Instalar skills e operar com um agente                   | [Guia para agentes](agents.md)                                                               |
| Consultar comandos do cliente                            | [CLI nativo](../packages/wos-cli/README.md)                                                  |
| Configurar credenciais e perfis para contratos assinados | [Perfis assinados](signed-workspace-profiles.md)                                             |
| Revisar entregas e recuperar respostas incertas          | [Retornos assinados](signed-cli-returns.md)                                                  |
| Preparar ou migrar um workspace/protocolo                | [Cutover](signed-protocol-cutover.md) e [migração](workspace-migration-v2.md)                |
| Usar a API ou conectar MCP                               | [HTTP](http.md), [contratos MCP](contracts.md) e [OpenAPI](../packages/wos-api/openapi.yaml) |
| Integrar um consumidor e tratar recibos                  | [Qualificação de clientes](client-qualification.md)                                          |
| Consumir eventos e webhooks                              | [Eventos de integração](integration-events.md)                                               |

## Operar o servidor

- [Operação](operations.md): ambiente local, serviço autenticado, bancos, backup e restore.
- [Recuperação do emissor](issuer-recovery.md): identidade e confiança em contratos assinados.
- [Release e distribuição](releases.md): instalação pública, plataformas e processo de publicação.
- [Referência operacional](stable/operations-reference.md): limites de capacidade, retenção e diagnóstico.
- [Medições](benchmarks.md): escopo e limites dos benchmarks; evidência técnica não equivale a uma garantia de produção.

## Desenvolver e entender o projeto

- [AGENTS.md](../AGENTS.md): contrato de desenvolvimento e verificações obrigatórias.
- [ROADMAP.md](../ROADMAP.md): estado real da implementação, trabalho atual e aceites pendentes.
- [Arquitetura canônica](WOS_Design_Arquitetura_Planejamento_Atualizado.md): modelo, invariantes e plano original.
- [Decisões de arquitetura](adr/): escolhas aceitas e sua justificativa.
- [Dependências](dependencies.md): toolchain e bibliotecas.
- [Validação da interface atual](design/editorial/validation.md): verificações automatizadas e limites.

## Registros históricos

Os relatórios datados, planos de implementação, ADRs e evidências em `audit/`, `ux/` e `stable/` documentam revisões específicas. Eles não substituem os guias atuais nem representam aprovação automática de uma versão posterior.

A antiga galeria foi substituída pela [galeria editorial](design/editorial/README.md#main-pages). As imagens anteriores e os dois relatórios de 5 de outubro removidos da documentação ativa continuam disponíveis no histórico Git; as referências de aceite apontam para o commit que os preserva.
