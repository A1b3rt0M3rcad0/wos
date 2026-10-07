# Continuidade para agentes

O WOS mantém estado e coordenação. O agente ou produto consumidor escolhe a ação e executa o trabalho fora do WOS. O caminho de referência independente é `tests/acceptance/journey.py`; o cliente Go executável é `examples/remote-client`.

## Ambiente local

Use Go 1.27.1 e o checkout existente:

```sh
go mod download
go build -trimpath -buildvcs=false -o bin/wos ./packages/wos-api/cmd/wos
WOS_MCP_ENABLED=true ./bin/wos server
```

O bind local padrão é loopback. A interface humana está em `/app/`, HTTP em `/api/v1` e Streamable HTTP MCP em `/mcp`. O Namespace local padrão é `0199d000-0000-7000-8000-000000000001`. O modo local usa uma identidade confiável configurada; para agentes com identidades distintas, use o perfil `api_token` e emita uma credencial vinculada a cada Principal/ActorRef conforme [operations.md](operations.md). O processo `wos mcp stdio` oferece stdio local. Leia [contracts.md](contracts.md) para os perfis MCP e os limites. Prefira o SDK oficial: o perfil 2026-07-28 usa `server/discover`, metadados por request e headers de método; o perfil 2025-06-18 usado pela referência mantém `initialize`. Não misture envelopes de perfis diferentes.

## Retomar um Outcome

1. Autentique e descubra Outcomes no Namespace autorizado (`wos_search_outcomes` ou `GET /api/v1/namespaces/{id}/outcomes`). Não deduza permissão de `tenant_id`, contexto externo ou ActorRef no payload.
2. Leia continuidade com `wos_get_continuity` ou `GET .../outcomes/{id}/continuity`. Guarde `outcome_revision`, `evaluated_at`, contagens e omissões. Expanda as seções pelos cursores quando necessário; `snapshot_changed` exige nova leitura.
3. Consulte trabalho pronto e o contexto focal, critérios, dependências, impedimentos, decisões e referências do plano ativo. Disponibilidade é uma projeção; não transforme `blocked` ou `ready` em status livre.
4. Escolha um WorkItem e adquira claim com sua `expected_version`, TTL explícito e uma chave idempotente. Guarde `claim_id` e `fencing_token`; renove enquanto estiver executando. Um reclaim incrementa fencing. O executor antigo deve parar ao perder o claim.
5. Registre Artifact/Evidence como referências com proveniência, não como substitutos de avaliação. Conclua WorkItem explicitamente com versão, claim e fencing atuais. Trabalho concluído não conclui automaticamente Objective/Outcome.
6. O participante autorizado avalia cada revisão de critério com resultado, justificativa e prova exigida. Conclusões registram as avaliações e obrigações usadas. Fatos posteriores podem contestar uma conclusão; nunca apague ou reabra automaticamente seu histórico.

O catálogo de comandos é `GET /api/v1/commands` e `tools/list`; MCP usa nomes `wos_{command_name}`. Argumentos incluem `idempotency_key` e `command`. Scope é explícito; Principal/ActorRef autenticados são resolvidos pelo servidor. Nunca invente IDs de entidades já existentes: descubra ou consulte o estado persistido.

## Falhas e recuperação

Uma resposta perdida pode esconder um commit bem-sucedido. Repita exatamente a intenção e a chave para reconciliar; HTTP e MCP compartilham os recibos. `version_conflict` exige leitura/revisão da intenção. `transaction_conflict` permite retry explícito da mesma intenção/versão/chave. WOS não altera a versão esperada para fazer um retry passar. Após expirar a retenção de recibos, confirme o estado antes de uma nova mutação.

Use IDs estáveis de Integration Event para deduplicar webhooks. WOS oferece entrega pelo menos uma vez, não exactly-once em efeitos externos. O receptor valida assinatura/timestamp e aplica sua própria transação/fencing/idempotência.

## Validar a integração

```sh
go test ./...
go test -race ./...
python3 tests/acceptance/journey.py --binary bin/wos --client http
```

Para o aceite PostgreSQL configure `WOS_TEST_POSTGRES_DSN` e, para clean restore, `WOS_TEST_POSTGRES_CONTAINER` com PostgreSQL 18.6 descartável. Os testes isolam schemas e restauram em banco vazio; nunca aponte essas variáveis para uma base de produção. Sem DSN os casos PostgreSQL são explicitamente ignorados. O navegador usa `npm ci`, `npx playwright install --with-deps chromium` e `npm test` em `tests/web`; `WOS_TEST_CHROMIUM_PATH` permite usar Chromium já instalado. Veja [verification-2026-10-07.md](verification-2026-10-07.md) para a evidência do candidato atual.
