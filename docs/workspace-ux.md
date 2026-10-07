# Workspace humano do WOS

A interface oficial em `/app/` organiza a colaboração em torno de um resultado. Usa o mesmo estado persistido e os mesmos comandos que HTTP/MCP. A revisão de 7 de outubro de 2026 segue as referências enviadas pelo proprietário: Jira para navegação e fluxo de trabalho; Guild.ai para linguagem visual. [Galeria de telas reais](ui-workspace/README.md).

## Estrutura

- **Barra lateral:** contexto autorizado, busca/filtro/paginação de resultados, seleção persistente durante a navegação e administração. No celular, “Trocar resultado” expande essa navegação. A seleção é da sessão de navegação; não se promete restaurá-la após reload.
- **Resumo:** resultado esperado, cinco métricas com denominadores, distribuição do trabalho, objetivos/evidências/planos ativos, trabalho em foco e atenção. Os números vêm do snapshot do servidor, incluindo itens fora da página atual. Não são analytics históricos ou uma inferência de execução de agentes.
- **Lista:** trabalho, objetivos, disponibilidade, problemas/impedimentos, decisões, evidências, artefatos, planos, contestações, histórico e relações. Cada item abre seu detalhe; as coleções maiores mantêm os cursores da API.
- **Quadro:** Planejado reúne backlog/aguardo/programação; Pronto para execução, Em execução, Impedimentos e Concluído mostram as projeções operacionais do servidor. Cancelados têm coluna própria quando presentes. Um item impedido em execução aparece com sua projeção de impedimento, preservando o lifecycle no detalhe.
- **Detalhe lateral:** descrição, estado, prioridade, referências relacionadas, autoria da reserva, critérios, conclusões e revisões do plano em formato legível. Os dados técnicos permanecem recolhidos para auditoria. Ações rápidas são contextualizadas pelo tipo/estado, mas o servidor continua validando autorização e todas as precondições.
- **Formulários:** nomes em português, itens e evidências por nome, contexto explícito, versões/reservas preenchidas a partir da leitura e proveniência adicional recolhida. Critérios abertos pelo detalhe preenchem a identidade/revisão; não é necessário copiar UUIDs. A ação de confirmar permanece visível enquanto os campos rolam.

## Participar com agentes

1. Selecione o contexto e o resultado. Consulte o resumo e o plano.
2. Abra um item na lista ou no quadro. Reserve trabalho por comando antes de executar; a reserva possui duração e fencing.
3. Agentes podem continuar via MCP/HTTP. Atualize a interface para obter seu trabalho, provas e histórico. Não há uma sessão de conversa escondida como fonte do estado.
4. Registre evidências e avalie o critério pelo detalhe do seu proprietário. A evidência sozinha não comprova o critério.
5. Certifique Objective/Outcome explicitamente após cumprir suas obrigações. Concluir trabalho não certifica o resultado. Consulte contestações quando houver fatos posteriores.

O quadro é uma visualização. Não oferece arrastar cartões para gravar estados derivados ou ignorar reservas/provas. A mudança de estado passa por uma ação humana explícita e pelo Application service.

## Busca, paginação e concorrência

Busca de resultados usa o contrato de descoberta do servidor. Pesquisa/prioridade de itens filtra os registros **já carregados** na visualização; o aviso explicita esse limite. “Carregar mais” continua a página na lista ou na coluna do quadro. As contagens continuam sendo do snapshot completo. Não se promete busca global em itens omitidos.

Os cursores permanecem vinculados à revisão/tempo/escopo do servidor. Se o snapshot mudar, atualize a leitura; não se altera silenciosamente a intenção nem a versão esperada para fazer uma mutação passar. Recibos idempotentes continuam reconciliando respostas incertas.

Durante a troca de resultado, os controles anteriores ficam ocultos. Respostas atrasadas de outra seleção são descartadas. A revisão do detalhe raíz e do snapshot também deve coincidir para publicar a leitura na interface. A interface é atualizada manualmente; não se anuncia sincronização push ou presença em tempo real.

## Verificação de 7 de outubro de 2026

Go 1.27.1, Node 24.19.0, Playwright 1.62.1, Chromium 151.0.7922.173; Linux amd64. Três jornadas Playwright passaram no código final.

- Humano planeja/replaneja, agente executa via MCP, humano avalia e certifica; outro navegador retoma após restart.
- Humano registra evidência, resolve Issue/Blocker explicitamente, reconcilia conflito, contesta sem reabrir implicitamente, arquiva/retoma; teclado/foco/campos e viewport móvel.
- Workspace com 32 WorkItems: resumo/denominadores, quadro e lista paginados, pesquisa/prioridade, reserva real pelo formulário, publicação de plano, prova por nome, referências do impedimento, histórico legível, cancelamento distinto de conclusão, resposta atrasada entre Outcomes e viewport de 390 px.

`go test ./packages/wos-api/... ./tests/boundary` passou com PostgreSQL real configurado, além de `go vet ./packages/wos-api/...`, build, sintaxe dos módulos JavaScript e diff sem erros. A revisão é de apresentação e não altera Domain, contratos, migrações ou dependências. Não se repete o total de 437 testes da entrega anterior como se fosse uma nova execução completa.

Logs: `docs/audit/workspace-2026-10-07-*`. GitHub Actions não iniciou os jobs do PR #15 por cobrança/limite da conta (anotação explícita; nenhuma etapa executada). A validação local não representa CI hospedado verde. Regularize a conta e reexecute os workflows.

Não se certifica WCAG integral por estas jornadas. A interface não inclui workflows customizados, planner/runtime de agentes, analytics de produção, calendário editável ou arraste irrestrito de estados.
