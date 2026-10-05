# Operação do WOS

Esta documentação descreve a branch de conclusão, ainda em revisão no PR #14. Não constitui aceite de Release 0.1. Linux amd64 com Go 1.27.1 foi o ambiente de execução local; PostgreSQL 18.6 real passou nos contratos compartilhados e race do CI #232; a jornada visual de referência passou em Chromium padrão no CI. A matriz completa e o aceite de release permanecem abertos.

## Instalação local

```sh
go mod download
go build -trimpath -buildvcs=false -o bin/wos ./packages/wos-api/cmd/wos
./bin/wos config validate
./bin/wos server
```

Abra `http://127.0.0.1:8080/app/`. O modo `local` aceita somente bind e Host de loopback. Cria o Namespace configurado em `WOS_LOCAL_NAMESPACE_ID` (default `0199d000-0000-7000-8000-000000000001`), nome `WOS_LOCAL_NAMESPACE_NAME` (default `Local`). A identidade local é configuração de composição; não representa autenticação multiusuário. Privilégios administrativos especiais exigem `WOS_LOCAL_ADMIN_OVERRIDES=true`.

O Core permanece utilizável via `application.NewService` para composição confiável embedded. Um host que aceita chamadas de terceiros deve usar `NewAuthorizedService` e resolver a identidade fora do payload.

## Servidor compartilhado

Configure `WOS_AUTH_MODE=api_token`, `WOS_BOOTSTRAP_NAMESPACE_ID` (UUIDv7), `WOS_BOOTSTRAP_NAMESPACE_NAME`, `WOS_LOCAL_PRINCIPAL_ID` e `WOS_BOOTSTRAP_TOKEN` (segredo aleatório com no mínimo 32 caracteres). `WOS_LISTEN` pode então usar uma interface pública. Use HTTPS no ingress; o processo não termina TLS. Não registre tokens em comandos versionados, logs ou URLs.

O bootstrap é exclusivo da primeira instalação: havendo credenciais no banco, não restaura grants nem ressuscita token revogado. A perda de todas as credenciais administrativas exige recuperação operacional offline sob controle do administrador; não existe bypass de login via variável de ambiente.

Cada token é vinculado a um Namespace, Principal e ActorRef. O navegador troca a credencial por uma sessão HttpOnly, SameSite Strict, com validade máxima de 12 horas e revogação própria. Revogar/expirar a credencial de origem invalida suas sessões. Cookies Secure são usados fora de HTTP de loopback. Mutações rejeitam origens estrangeiras.

`GET /api/v1/namespaces/{namespace_id}/administration` retorna a versão administrativa, grants, metadados de credenciais e até 100 entradas recentes de auditoria por coleção; `truncated` impede interpretar omissões como completude. `POST /api/v1/security/commands` recebe o contrato `AdministrativeIntent`: `namespace_id`, `expected_namespace_version`, `operation` e os campos da intenção. Exige `Idempotency-Key`. Operações: `set_grant`, `issue_credential`, `revoke_credential`, `create_namespace`. A criação copia somente as permissões atuais do criador e emite uma credencial vinculada ao novo Namespace, válida por 12 horas. A versão devolvida refere-se ao Namespace de origem; o novo inicia em versão 1.

Segredos emitidos aparecem apenas na primeira resposta. O replay retorna o recibo e `token_omitted=true`; se a resposta original foi perdida, consulte o ID, revogue essa credencial e emita outra com nova intenção. Nenhum segredo recuperável é persistido. Namespace CAS, autorização atual, alteração, auditoria e recibo são atômicos. As mutações de Outcome revalidam grants e credenciais dentro da transação, antes de revelar replay ou persistir estado.

`WOS_INDEPENDENT_REVIEWER=true` exige que quem avalia/conclui não tenha executado trabalho no mesmo Outcome. A política compara Principal, incluindo todo o histórico, e não aceita troca de alias de ActorRef como novo revisor.

## PostgreSQL e containers

```sh
WOS_STORAGE_DRIVER=postgres WOS_POSTGRES_DSN='postgres://USER:PASSWORD@HOST:5432/DB?sslmode=require' ./bin/wos server
```

O DSN é configuração secreta. São usados pgx, transações Serializable, guard por Outcome, pool de até 16 conexões, lock timeout de 5 s e statement timeout de 30 s. Conflitos transitórios retornam `transaction_conflict`; nunca se altera a versão esperada silenciosamente. O CI usa PostgreSQL 18 real. SQLite aprovado não equivale a PostgreSQL aprovado.

`Dockerfile` produz binário CGO-free e imagem com usuário 10001, certificados CA e volume `/data`. `compose.yaml` expõe apenas loopback no host e exige configuração de bootstrap. Por padrão usa SQLite. Para PostgreSQL, configure senha não vazia, `WOS_STORAGE_DRIVER=postgres`, DSN com host `postgres` e inicie `docker compose --profile postgres up --build`. Aguarde a saúde do banco antes do servidor; containers não foram executados no ambiente desta auditoria. Segredos são fornecidos pelo operador, fora do repositório.

## MCP

Ative `WOS_MCP_ENABLED=true` para Streamable HTTP em `/mcp`. O servidor usa o SDK oficial Go v1.8.0 e respostas JSON stateless; estado operacional pertence ao banco. HTTP remoto recebe `Authorization: Bearer ...`. As ferramentas revalidam a identidade/autoridade a cada chamada.

Para stdio local: `WOS_MCP_ENABLED=true ./bin/wos mcp stdio`. Esse comando não abre socket HTTP. stdout é reservado ao protocolo; logs vão para stderr. O perfil OAuth/IdP não está implementado. Veja [contracts.md](contracts.md) e os testes de cliente real.

## Backup, restore e upgrade

SQLite: `WOS_SQLITE_PATH=./data/wos.db ./bin/wos db backup ./backup/wos.db`. O adapter usa `VACUUM INTO`, incluindo um snapshot consistente de todas as tabelas e evitando cópia incompleta de WAL. Nunca copie apenas o arquivo principal com o banco ativo.

Restore SQLite: pare o servidor e workers; `./bin/wos db restore ./backup/wos.db ./data/restored.db`. O destino deve ser novo; não se sobrescreve uma instalação existente. A restauração é offline e copia todo o banco. Configure o caminho restaurado, execute `db migrate`, consulte `/readyz` e valide a continuidade antes de aceitar tráfego.

PostgreSQL: use `pg_dump --format=custom` e `pg_restore --single-transaction` para banco novo, com a versão de cliente correspondente ao servidor. Inclua schema completo, histórico, grants, credenciais, idempotência e outbox. Nunca restaure só as entidades atuais. Restore PostgreSQL completo permanece sem execução de aceite nesta sessão.

Migrações numeradas têm checksum e são ascendentes. `db migrate` aplica as pendentes; o servidor migra ao abrir. Bancos com schema desconhecido não ficam ready. Não há downgrade SQL destrutivo suportado: rollback operacional usa o binário antigo e seu backup pré-upgrade, em banco separado, considerando explicitamente os fatos posteriores que precisam de reconciliação.

Leases mantêm tempos UTC absolutos e fencing. Após restore, sincronize relógios; leases expirados precisam de reclaim explícito. Uma restauração pode reenviar entregas que o consumidor já recebeu; deduplique por `integration_event_id`. Não confie em memória de sessão para decidir o que terminou.

## Integrações e observabilidade

Endpoints são configurados em `WOS_WEBHOOK_ENDPOINTS` como array de `{id,namespace_id,url,secret_ref,key_id}`. Segredos ficam no mapa `WOS_WEBHOOK_SECRETS`, com valores aleatórios de pelo menos 32 caracteres; nenhum valor secreto aparece em comandos/modelos de Trigger. `WOS_DELIVERY_WORKER_ENABLED=true` inicia o worker do Server. `WOS_WEBHOOK_ALLOW_LOOPBACK=true` é opt-in para testes locais.

Destinos HTTPS têm allowlist de hosts da configuração, DNS/IP verificados, bloqueio de endereços privados e de redirecionamentos, timeout de 10 s e limite de leitura de resposta de 64 KiB. Triggers não fazem requisições dentro da transação. Cada fato, firing e delivery é gravado atomicamente; o worker tem lease de 30 s e fencing. Há até seis tentativas, backoff exponencial a partir de 2 s, recuperação de crash e redelivery explícito autorizado. O consumidor recebe o mesmo ID estável em cada tentativa.

Assinatura: `X-WOS-Signature = sha256=<HMAC-SHA256(secret, timestamp + "." + eventID + "." + body)>`; headers adicionais `X-WOS-Event-ID`, `X-WOS-Timestamp` e `X-WOS-Key-ID`. Verifique o corpo bruto, janela de cinco minutos e dedupe durable. Use `integration.VerifySignature` no consumidor Go. Rotacione segredo mantendo o Key-ID necessário para tentativas já pendentes.

Logs JSON registram comando, Principal/ActorRef, correlação, revisão, duração, espera de aquisição da transação, commit, erro classificado e replay. O observer permanece fora de Domain. Requests registram método, rota/caminho sem query, duração, status e tamanho; não registram corpo, headers de autenticação, tokens ou payload documental. `/metrics` devolve contadores JSON de comandos, replay, conflitos e requisições; em modo remoto exige namespace:admin. Reiniciar o processo reinicia esses contadores. Histograma de guard, métricas completas de leases/deliveries e contagem detalhada de SQL ainda são trabalho de aceite, não capacidades declaradas como concluídas.

Retenção de idempotência de Outcome: sete dias por padrão. Registros históricos e recibos administrativos não são purgados automaticamente. Planeje espaço e backups; arquivar não exclui histórico. Consulte [contracts.md](contracts.md) para os limites das consultas.
