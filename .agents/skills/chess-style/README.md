# chess-style

Skill de direção de arte e construção de websites editoriais. Seu centro é a relação
entre serifas de alto contraste em escala monumental, fundo semelhante a papel,
grid discreto, escrita marginal, pequenas ilustrações em tinta e objetos recortados.

## Começar

Descompactar o ZIP mantendo a pasta `chess-style/` inteira. Em uma ferramenta que
carregue skills, adicionar essa pasta ao diretório de skills aceito pela ferramenta.
Como esse caminho depende do agente, este pacote não presume um instalador universal.

Em qualquer agente com acesso a arquivos, instruir:

> Leia `chess-style/SKILL.md` e aplique a skill ao projeto. Use o conteúdo real,
> preserve a stack existente e siga as referências necessárias antes de implementar.

Não há dependência de runtime para ler a skill. A documentação está em português;
o conteúdo de uma interface deve usar o idioma pedido pelo usuário.

## Estrutura

| Caminho | Função |
|---|---|
| `SKILL.md` | Entrada, gatilhos, workflow, invariantes e contrato de saída |
| `references/` | Manual visual, técnico, narrativo e de validação |
| `tokens/` | Tokens JSON com procedência e sua exportação CSS |
| `templates/` | Brief, plano, matriz de direção, manifesto de assets e relatório |
| `evidence/` | Recortes das capturas fornecidas, exclusivamente para estudo |
| `examples/starter/` | Base HTML/CSS/JS local e sem dependências de rede |
| `scripts/` | Gerador CSS, validação estrutural e smoke test opcional de navegador |
| `reports/` | Resultado dos testes do starter e capturas do exemplo |
| `PACKAGE_MANIFEST.json` | Inventário técnico com tamanho e hash dos arquivos |

## Rotas de leitura

**Direção de arte:** Design DNA → Anatomia visual → Tipografia → Linguagem gráfica →
Composição → Atlas de seções.

**Implementação:** Playbook → Tokens → Arquitetura → Motion → Responsividade e
acessibilidade → Checklist.

**Auditoria:** Evidências e limites → Inventário de ativos → Checklist → Relatório.

## Exemplo local

Abrir `examples/starter/index.html` no navegador. Opcionalmente, executar na pasta
raiz desta skill:

```bash
python -m http.server 8000
```

Então acessar a rota `/examples/starter/` nesse servidor local. O exemplo usa fontes
locais de fallback e marcadores gráficos deliberadamente esquemáticos. Não carrega
fontes, trackers, bibliotecas remotas nem assets da referência. Os controles de leitura
estática e movimento são funcionais; as âncoras conduzem a conteúdo real do exemplo.

## Atualizar tokens e validar

```bash
python scripts/build_tokens.py
python scripts/validate_package.py
```

Os dois scripts usam apenas a biblioteca padrão do Python. O smoke test opcional
`python scripts/browser_smoke.py --help` precisa de Playwright e de um Chromium local;
as instruções não instalam dependências automaticamente.

## O que foi observado e o que foi proposto

A linguagem está fundamentada nas sete capturas fornecidas. O arquivo de escopo
recebido orientou a cobertura: essência, grafia, tipografia, composição, movimento,
componentes, narrativa, documentação e critérios de aceite. Os documentos separam
observações da imagem de soluções propostas para construir uma nova interface.

A família tipográfica exata, os estilos computados, as durações originais e o
comportamento real em mobile não foram confirmados. O pacote não se apresenta como
cópia do código ou auditoria integral da implementação de origem.

Não estão incluídos nomes/créditos da referência, texto promocional original,
retratos pessoais, fontes binárias, um modelo 3D de produção ou um backend comercial.
Os recortes visuais são evidências de estudo e não devem ser publicados no novo site.

## Verificação incluída

O relatório `reports/VALIDATION.md` registra 11 cenários de Chromium com o starter
carregado em memória, incluindo sete larguras, ausência de JS, movimento reduzido
e ampliação do texto. O relatório explicita o que não foi testado.

Versão da skill: **1.0.0**.
