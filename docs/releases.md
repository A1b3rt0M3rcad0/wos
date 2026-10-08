# Releases e instalação

## Produtos

`VERSION` é a versão SemVer coordenada dos dois pacotes: `@a1b3rt0m3rcad0/wos` (serviço Linux amd64) e `@a1b3rt0m3rcad0/wos-skill` (skills portáveis, Node >=22). Não confundir SemVer com HTTP `/v1`, versão de snapshot/evento, migração SQL ou protocolo MCP negociado.

O campo `wosCompatibility` do pacote de skills identifica a versão coordenada de serviço e os contratos alvo; não certifica outras versões ou runtimes. Um major Go >=2 também exige revisar o module path/imports conforme Go semantic import versioning antes de publicar a tag; o gerador de versões não migra imports ou contratos automaticamente.

O arquivo nativo e o container dispensam Node. O wrapper npm do serviço não tem download/postinstall, verifica o binário e encaminha sinais/exit status. Skills não iniciam agentes nem configuram MCP: usam o contrato do servidor e as ferramentas nativas do runtime. Veja [pacote de skills](../packages/wos-skill/README.md), [pacote do serviço](../packages/wos-npm/README.md), [plano e tradeoffs](distribution-plan.md) e [operação](operations.md).

## Fluxo automático

1. Cada mudança distribuível adiciona `.changes/<nome>.json` com bump explícito e resumo. Mudanças incompatíveis usam major, inclusive em 0.x.
2. Após merge em `master`, `prepare-release.yml` prepara versões/notas/manifesto e abre ou atualiza o PR reservado `release/next`. Esse branch pertence à automação; não use para outras mudanças. A primeira preparação mantém 0.1.0, seguintes escolhem o maior bump.
3. Revise e integre o PR. Um PR criado por GITHUB_TOKEN pode não disparar workflows adicionais; o workflow de publicação sempre executa os gates completos depois do merge, independentemente desse comportamento.
4. `release.yml` verifica o commit exato por três workflows reutilizáveis: Go/API/PostgreSQL/race/acceptance/container, navegador, e skills/distribuição/release contracts. Não há publicação em PRs de forks ou branch arbitrário.
5. Depois dos gates, constrói/testa tarballs e imagem, confere integridade e identidade, cria tag imutável/draft, publica ambos os npm, publica GHCR e finaliza GitHub Release. Tags versionadas não são substituídas. Não há deployment automático em instâncias de clientes, migrations ou reinício de dados.

Build/replay exige o mesmo commit e ferramenta: Go 1.27.1, Node 24.19.0 e npm 11.9.0. Source timestamp vem do commit, não do relógio do job. Os tarballs e arquivo nativo devem ser byte-idênticos entre caminhos de source, verificados por `tools/distribution/reproducible.mjs`. Artefatos são `wos-VERSION.tgz`, `wos-skill-VERSION.tgz`, `wos_VERSION_linux_amd64.tar.gz`, `distribution.json`, `SHA256SUMS`; imagem `ghcr.io/a1b3rt0m3rcad0/wos:VERSION`.

## Configuração pelo titular das contas

- Regularize billing/spending do GitHub Actions. Jobs não iniciaram nesta sessão; testes locais não significam publicação hosted.
- Confirme o escopo npm `@a1b3rt0m3rcad0`. O nome sem escopo `wos` pertence a outro produto. Configure autorização de publicação dos dois pacotes.
- Bootstrap/token: configure `NPM_TOKEN` em GitHub Actions Secrets com token npm granular adequado para publicar/criar esses pacotes e a política de 2FA da conta. Não coloque o valor em chat, skill ou Git. A sessão não tem npm autenticado e a integração não consegue listar Actions Secrets (403); isso não prova que o Secret já exista ou não.
- Trusted publishing: quando suportado pela conta/pacotes/repositório, associe cada pacote ao repositório `A1b3rt0M3rcad0/wos`, workflow `release.yml`, e configure Actions variable `NPM_AUTH_MODE=oidc`. O workflow concede id-token somente ao job de publicação. O primeiro cadastro pode requerer bootstrap pelo titular; vínculo OIDC não é configurado por um commit.
- Proveniência: `NPM_PROVENANCE=true` somente quando requisitos do npm e repositório fonte público forem atendidos. Este repositório é privado; a flag fica desligada por padrão e o workflow rejeita ativá-la aqui. Não tornar o repositório público como workaround. Consulte [npm trusted publishing](https://docs.npmjs.com/trusted-publishers/) e [provenance](https://docs.npmjs.com/generating-provenance-statements/).
- Permita que Actions criem PRs em Settings → Actions → General. Configure proteção de `master` e tags de release conforme o processo da equipe. GHCR precisa de packages:write e pode manter visibilidade privada; consumidores precisam acesso. Não há mudança automática de visibilidade de repositório, pacote ou imagem.

## Falha parcial e retomada

Registries não oferecem transação conjunta. Uma falha pode deixar um pacote npm publicado, tag e draft, sem concluir a distribuição. Corrigir a causa e executar `release.yml` por workflow_dispatch com `source_sha` do commit original. Todas as verificações e builds usam esse SHA, que precisa pertencer à história de master; não usar o master mais novo para reutilizar a versão antiga.

Retomada compara integridade npm, commit da tag, bytes dos assets e metadata da imagem. Versões existentes idênticas são reconciliadas; qualquer divergência interrompe a publicação. Assets não usam `--clobber`; tag não usa force. Uma publicação já visível no npm não é desfeita automaticamente. Uma correção deve usar nova versão. Se o registry ainda não expôs metadata após publicar, o job falha de forma explícita e pode ser retomado no mesmo SHA depois da propagação.

Quando master já avançou e `source_sha` é histórico, use publicação por token sem proveniência nessa retomada. O workflow rejeita OIDC/proveniência nesse caso: a identidade de execução do GitHub descreve o SHA atual, não o source histórico. Não gerar atestação que atribua artefatos ao commit errado. Retomadas no mesmo SHA podem usar o modo original suportado.

Mudanças novas após uma release preparada geram outro PR e versão; não editar a mesma versão publicada. Para rollback operacional, fixe uma versão anterior e restaure/valide dados conforme o guia de operações — baixar um binário antigo não reverte schema automaticamente.

## Validação local

```sh
node --test packages/wos-skill/test/*.test.mjs tools/distribution/test.mjs tools/releases/test.mjs
node tools/releases/version.mjs plan
node tools/distribution/build.mjs
node tools/distribution/verify.mjs
node tools/distribution/reproducible.mjs
# Após integrar o manifesto de release e com o source limpo:
node tools/releases/publish.mjs --dry-run
```

Use o compilador correto em PATH e caches graváveis no sandbox. `build` produz arquivos ignorados; `prepare` altera arquivos rastreados e deve ser executado no branch de release. Dry-run valida source/artifacts e mostra etapas; não acessa registries nem prova autenticação ou publicação. A publicação real exige o workflow autorizado, credenciais e gates.

## Matriz de evidência e melhorias

Instalação/atualização/remoção das quatro skills e destinos nativos/genéricos são testadas. O serviço instalado realmente inicia HTTP/MCP/UI, trata SIGTERM e rejeita corrupção. Replays e falhas de publicação são exercitados com adapters determinísticos; uma publicação real exige os serviços externos. Nenhum teste de arquivo prova execução em todas as versões de Claude/Codex/Hermes/OpenClaw.

Próximas melhorias: integração nativa executada em cada runtime; contexto focal mais seletivo por tarefa; medição de custo/token e retomada; arm64/macOS/Windows com smoke nativo; assinatura de assets e attestations da imagem; métricas/backup/deploy gerenciado por cliente. Versões independentes de serviço e skills só quando o acoplamento atual causar releases desnecessárias. Esses itens são backlog, não promessas implementadas.

## Contracts/client release 0.2.0

WOS npm includes Linux `wos` + `wosctl`; wos-skill contains five skills and installs
20 skill copies across four discovery roots when `--agent all` is selected.
Additional immutable assets are `wosctl_VERSION_linux_amd64.tar.gz` and
`wosctl_VERSION_windows_amd64.zip`, with schema/README/licenses and source metadata.
Docker includes the API-only client: `--entrypoint wosctl ... version --output json`.

`release.yml` requires the reusable native CLI matrix, in addition to core/browser/
packages. Native jobs run race/protocol/workspace and skill installer tests on their
host OS, build/execute the exact-source client and upload checksum receipts. The
publisher checks both host receipts against the generated binary SHA-256 identities;
missing, cross-built or wrong-source receipts block publication. Dry-run may inspect
prepared assets without pretending native Windows validation occurred. Gate receipts
are CI evidence, not secret material or agent authority. Matching source_ref is used
throughout historical retries.

Current local state: source/artifacts can be prepared and tested on Linux. Actual
Windows job is blocked by GitHub Actions billing/spending restrictions. Account-side
npm authorization remains separate. No released 0.2.0/npm/GHCR claim is made until
publication and immutable identities have actually been verified.
