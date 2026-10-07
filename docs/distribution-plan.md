# Distribuição, skills e releases do WOS

Status: plano aprovado para implementação pelo proprietário em 2026-10-07. O ROADMAP distingue código verificado de publicação externa. Extensão às Waves 13/17/18, sem mudar o motor de domínio.

## Produtos e limites

1. `@a1b3rt0m3rcad0/wos`: distribuição npm do serviço Go, com o executável Linux amd64 embarcado, UI, HTTP e MCP existentes. Não é um novo serviço Node nem SDK. Também será distribuído como arquivo Linux e imagem GHCR.
2. `@a1b3rt0m3rcad0/wos-skill`: instalador npm sem dependências de runtime, contendo skills portáveis e referências carregadas sob demanda. Não inicia agentes nem altera regras de domínio.

O nome npm sem escopo `wos` já pertence a outro projeto. O escopo proposto deve ser controlado pela conta publicadora. As duas distribuições seguem uma versão SemVer coordenada em `VERSION`; o pacote Go continua no módulo atual. URLs, token, Namespace e Outcome são configuração da instância, nunca conteúdo do pacote.

Compatibilidade significa arquivos no formato Agent Skills e operação pelos contratos MCP/HTTP. Não significa que todos os runtimes existentes foram executados ou que todos oferecem subagentes. Claude Code, Codex, Hermes e OpenClaw recebem destinos documentados; agentes adicionais usam um destino explícito. A matriz registra instalação testada separadamente de execução nativa não verificada.

## Etapas e PRs

| Etapa | Entrega | Aceite |
| --- | --- | --- |
| D1 | Plano, ADR e backlog | Limites, benefícios, riscos, sequência e pré-requisitos explícitos |
| D2 | Pacote de skills | Instalação npm empacotada, quatro destinos, destino genérico, conflito seguro, atualização e remoção verificadas |
| D3 | Pacote do serviço | Binário real versionado, hash, instalação offline do tarball, health e MCP/HTTP funcionais, processo e erros propagados |
| D4 | Versionamento e CI/CD | Changes, PR de release automático, gates existentes, release draft, npm/GHCR, retomada sem sobrescrever versões e testes de falhas |

Cada etapa tem commits semânticos e PR próprio integrado sobre `master`. Mudanças novas incluem uma entrada `.changes/*.json` com bump e descrição. Infraestrutura de CI indisponível não autoriza remover gates.

## Skills

- `wos-coordination`: descoberta limitada, contexto focal, claim, renovação, versões, fencing, idempotência, registro de resultado e parada por impedimento.
- `wos-delegation`: supervisor cria tarefas delimitadas; runtime lança subagentes com contexto mínimo explicitamente; cada executor opera com autoridade apropriada; alterações concorrentes em arquivos são coordenadas fora do WOS.
- `wos-continuity`: checkpoints persistidos, resumo ativo pequeno, expansão sob demanda, retomada após perda de sessão e tratamento de omissões. Não promete apagar mensagens ou compactar o runtime.
- `wos-review`: critérios revisionados, prova, revisão independente quando exigida, certificação explícita e contestação preservada.

O instalador tem escopo de projeto por padrão. Destinos pessoais exigem `--global`; um diretório customizado exige `--dir`. Mostra destino em dry-run, reconhece conteúdo gerenciado por hashes, nunca substitui modificações locais automaticamente e não escreve credenciais/configurações MCP. Instalação não executa código do projeto. Acesso MCP deve ser configurado no cliente conforme sua versão; não inventar APIs universais de subagentes.

## Política de release

Uma fila de changes gera PR revisável com versões de ambos os pacotes, manifesto e notas. A primeira versão é 0.1.0; seguintes usam o maior bump declarado. Breaking changes são explícitas, inclusive em 0.x; não inferir bump somente de títulos de commit. Após merge do PR de release, a pipeline verifica Core/API, PostgreSQL, race, navegador e distribuição antes de publicar.

Builds usam Go fixado, CGO desativado, paths removidos e metadata derivada do commit. Assets contêm licença, notas, checksums, serviço e skills npm. O executável é embarcado: nenhum postinstall baixa binários ou inicia serviços. O wrapper verifica sua integridade e propaga sinais/exit status.

GitHub Release começa como draft. Publicam-se tarballs já testados no npm e imagem GHCR com tag SemVer; finaliza-se a release somente após todas as publicações. Publicação entre registries não é transacional: uma falha pode deixar um pacote disponível. Retomada compara versão e integridade; nunca substitui uma versão ou reutiliza tag para outro commit. Rollback seleciona uma versão anterior; correções geram nova versão. Atualização não executa migrations ou substitui dados automaticamente.

Preferir npm Trusted Publishing/OIDC e proveniência. O primeiro cadastro de pacote e o vínculo ao repositório/workflow precisam ser feitos pelo titular npm; há bootstrap via token granular em GitHub Secrets, nunca no repositório. Não usar `pull_request_target` ou secrets em PRs externos. Permissões de escrita ficam somente nos jobs de publicação, após gates.

## Vantagens, desvantagens e melhorias

| Escolha | Vantagem | Desvantagem / ação |
| --- | --- | --- |
| npm para serviço e skills | Instalação/versionamento conhecidos; fácil fixar uma versão | Serviço npm requer Node; arquivo nativo e container atendem quem não usa Node |
| Binário embarcado | Funciona sem download na instalação; conteúdo verificável | Tarball maior; plataforma inicial somente Linux amd64 |
| SemVer coordenado | Contrato simples entre skills e serviço | Release documental também versiona serviço; separar versões quando houver necessidade comprovada |
| Skills portáveis | Mesma disciplina entre agentes | Descoberta e delegação variam; adaptadores documentados e fallback explícito |
| Contexto focal + checkpoint | Menos histórico repetido e melhor retomada | Economia total de tokens depende do runtime; medir em piloto antes de prometer percentual |
| Release com gates | Evidência antes de distribuição | Billing, credenciais e permissões podem impedir publicação; não contornar testes |
| Instância dedicada | Isolamento operacional claro para clientes | Exige operação: backup, restore, atualização, autenticação e monitoramento por cliente |

Melhorias posteriores, sem apresentar como existentes: focar decisões do contexto por tarefa (hoje pode incluir decisões aceitas do Outcome); integração nativa executada em cada versão de agente; orçamento de contexto no consumidor; matriz arm64/macOS/Windows com smoke nativo antes de suporte; catálogo MCP seletivo no cliente; assinatura de assets além de checksums; deploy gerenciado por cliente com migration/backup gate. CDN, autoexecução e orquestrador de agentes dentro do Core ficam fora deste escopo.

## Pré-requisitos externos observados

Em 2026-10-07: repositório privado, sem releases/tags; GitHub API/push disponíveis; leitura de Actions Secrets negada pelo token da integração; npm sem identidade autenticada. Actions não iniciam jobs por billing. Pacotes públicos npm não tornam o repositório público; imagens GHCR podem exigir acesso. É necessário regularizar billing e provisionar npm pelo titular. Código e testes locais podem ser concluídos independentemente; publicação externa permanece bloqueada até esses requisitos, sem alegar release publicada.
