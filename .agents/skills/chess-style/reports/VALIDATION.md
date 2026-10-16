# Validação do pacote e do starter

Versão: 1.0.0. Data: 15/10/2026. Escopo: **arquivos da skill e exemplo incluído**.
Nenhum resultado deste relatório deve ser atribuído à implementação de referência.

## Resultado

Validação estrutural concluída: arquivos obrigatórios, frontmatter, JSON, nomes de
tokens, correspondência JSON/CSS, recursos locais e âncoras. Sintaxe JavaScript
verificada com `node --check`. Os recortes foram inspecionados em uma prancha conjunta.
Não há arquivos binários de fontes no pacote.

O starter passou em **11 cenários**, com **75 verificações automatizadas**
distribuídas conforme a tabela. Nenhum erro de página ou console foi registrado nos
cenários. Não foi detectado overflow horizontal nos pontos de scroll examinados.

## Ambiente e método

Navegador: Chromium 144.0.7559.96, headless, neste ambiente de geração.
HTML, CSS e JavaScript foram carregados em memória para renderização local. A navegação
para URLs locais estava bloqueada pela configuração do ambiente; portanto não houve
validação de hospedagem HTTP ou abertura direta via URL de arquivo. Os caminhos de
recursos foram verificados separadamente por inspeção estrutural.

A inspeção da fonte usada no H1 de desktop retornou **Tinos**. É um fallback
local: não é identificação da fonte das capturas. Nenhum arquivo de fonte foi copiado.

Comando usado para o smoke test:

```bash
python scripts/browser_smoke.py --browser /usr/bin/chromium --transport inline --out DIRETORIO_DE_RESULTADOS
```

O caminho do executável depende do ambiente. Os testes com texto 200% aumentam o
`font-size` da raiz para 200%; isso não equivale a todas as formas de zoom nativo do
navegador nem a uma auditoria completa do critério de redimensionamento.

## Cenários

| Viewport | Modo inicial do teste | Resultado | Verificações |
|---|---|---|---:|
| 1920×1080 | default | Aprovado | 6 |
| 1440×900 | default | Aprovado | 6 |
| 1280×800 | default | Aprovado | 6 |
| 768×1024 | default | Aprovado | 9 |
| 390×844 | default | Aprovado | 9 |
| 360×740 | default | Aprovado | 9 |
| 320×740 | default | Aprovado | 9 |
| 390×844 | no-js | Aprovado | 5 |
| 390×844 | reduced | Aprovado | 8 |
| 390×844 | root-text-200 | Aprovado | 4 |
| 1440×900 | root-text-200 | Aprovado | 4 |

Nos cenários padrão, o teste inicia com movimento permitido e aciona a leitura
estática. As capturas de viewport são feitas depois das interações, no topo da
página; o estado exato de cada captura está em `results.json`.

## Verificações executadas

Presença de um H1 e conteúdo; inicialização progressiva; overflow em quatro seções;
controle de leitura estática e tinta integral; abertura do menu compacto; Escape e
retorno do foco; fechamento ao seguir âncora; navegação sem JavaScript; preferência
reduzida. Nem toda verificação se aplica a todo cenário; a tabela e o JSON explicitam
o conjunto executado.

A revisão visual identificou e corrigiu sobreposição do objeto com a legenda no
hero e overflow de uma legenda lateral com texto ampliado. Houve reteste após as
correções. As capturas finais de desktop e mobile foram inspecionadas.

## Limites

Não foram testados Safari, Firefox, dispositivos físicos, leitores de tela, zoom
nativo em todas as modalidades, integração com backend, carga de fontes externas,
fontes licenciadas de produção ou desempenho em rede real. Não foram medidos
Lighthouse, Core Web Vitals ou FPS. Não há declaração de conformidade WCAG completa.

O starter contém objeto e esquema gráficos deliberadamente simplificados. A direção
de arte final exige a fonte e as ilustrações próprias especificadas na skill.

## Capturas

[Desktop completo em modo estático](screenshots/starter-1440-full-static.png)

[Mobile completo em modo estático](screenshots/starter-390-full-static.png)

Os demais arquivos em `screenshots/` correspondem aos cenários listados no JSON.
