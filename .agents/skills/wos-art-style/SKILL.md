---
name: wos-art-style
description: >-
  Direção de arte local do projeto WOS para criar, selecionar ou editar ilustrações,
  personagens e assets. Usar quando o trabalho envolver ilustrações do projeto ou
  quando o usuário mencionar wos-art-style. Define desenho editorial manual em
  tinta preta, monocromático, com personagens expressivos e fundo transparente.
---

# Direção de arte das ilustrações WOS

## Prompt canônico do usuário

Minimalist hand-drawn black ink cartoon illustration, organic slightly imperfect outlines, expressive pen-and-ink linework, subtle cross-hatching and diagonal hatching for shadows, playful character proportions, oversized rounded head, thin expressive limbs, sketchbook aesthetic, monochrome black and white, editorial illustration style, clean silhouette, isolated character, transparent background, no gradients, no colors, no photorealism.

## Aplicação no projeto

- Usar o prompt canônico como base e acrescentar o assunto, a ação e a composição necessários à tarefa.
- Preservar contornos orgânicos ligeiramente imperfeitos, traço expressivo de caneta e hachuras sutis para sombras.
- Usar proporções lúdicas, cabeça grande e arredondada e membros finos e expressivos.
- Manter silhueta legível, personagem isolado, tinta preta e fundo transparente; branco pode compor detalhes internos do desenho.
- Não usar gradientes, cores ou fotorrealismo nas ilustrações.
- Ao gerar ou editar imagens, solicitar transparência real no arquivo, sem fundo branco ou padrão quadriculado desenhado.
- Os SVGs existentes ficam em `packages/wos-api/web/assets/robots/`, a partir da raiz do projeto. Reutilizar quando adequados e preservar seus originais.
- Esta direção se aplica às ilustrações. Não altera automaticamente cores da interface, tipografia, layout ou comportamento do produto.
- A skill `chess-style` continua disponível para composição editorial; em trabalhos que combinem ambas, usar esta direção específica para as ilustrações.
- Por decisão explícita do proprietário, a interface usa a paleta operacional clara da Woobe em `packages/wos-api/web/assets/colors.css`: neutros quentes, carvão e cobre, com laranja reservado à marca e cores semânticas para estados. Essa decisão substitui as sugestões de cinza/verde da `chess-style`, mantendo sua composição editorial. Consultar `docs/design/editorial/woobe-palette-source.md` para a especificação recebida. Não recolorir os robôs.
- Criar ou modificar assets somente quando solicitado; registrar o estilo não implica redesenhar os assets existentes.
- A marca do produto usa o símbolo modular fornecido em `packages/wos-api/web/assets/brand/`, com proporção mestre 194 × 82 e laranja `#FF5500`. Preservar os paths, a proporção e os respiros dos originais; não substituir o símbolo por letras ou monogramas improvisados. Por decisão explícita do proprietário, o nome “WOS” deve ser texto em Instrument Serif, a mesma tipografia editorial dos títulos, no header e nas demais menções da interface. Os lockups com Inter convertido em paths ficam como referências originais, sem uso como logotipo na interface. Consultar `docs/design/editorial/brand-kit/README.md` para as aplicações. A marca não substitui os robôs.

## Fluxo obrigatório de arte no desenvolvimento local

Esta skill é parte do repositório: não é preciso instalar pacote ou serviço externo para registrar a direção de arte. Agentes de desenvolvimento devem consultar `AGENTS.md` e esta skill **antes** de criar, selecionar, editar ou integrar ilustrações, inclusive quando a solicitação for feita apenas em linguagem natural. No Cursor, a regra de projeto `.cursor/rules/wos-art-style.mdc` também orienta arquivos de arte relevantes.

### 1. Delimitar a intenção

- Identificar o assunto, a ação, a expressão, o enquadramento e o contexto operacional WOS que a ilustração deve comunicar.
- Verificar primeiro se algum dos 20 robôs existentes em `packages/wos-api/web/assets/robots/` atende à intenção.
- Tratar o robô fornecido pelo proprietário como referência de personagem: silhueta, cabeça arredondada, proporções, traço e expressões. Não trocá-lo por um mascote genérico.
- Não criar assets novos quando a solicitação for apenas documentar ou utilizar a direção de arte.

### 2. Compor o prompt sem alterar o estilo

Sempre começar com **o texto integral e literal** da seção «Prompt canônico do usuário», sem encurtar ou reescrever seus qualificadores. Acrescentar somente parâmetros específicos da cena, por exemplo:

- **Sujeito:** o robô WOS existente, mantendo sua identidade.
- **Ação:** revisando evidências de uma tarefa diante de um pequeno quadro.
- **Composição:** figura isolada, silhueta inteiramente contida no enquadramento, respiro em todos os lados, legível em tamanho reduzido.
- **Formato:** transparência alfa real; SVG vetorial apenas quando o arquivo resultante contiver vetores reais.

Restrições negativas: cores, gradientes, brilho, renderização 3D, fotorrealismo, traço geometricamente perfeito, cenário preenchendo toda a imagem, fundo opaco, padrão quadriculado desenhado, texto ilegível, mudança de identidade do robô e elementos cortados. Essas restrições aplicam-se às ilustrações, **não** ao sistema cromático da interface ou à marca.

### 3. Produzir, revisar e integrar

1. Preservar o asset original; uma variante autorizada recebe nome próprio, preferencialmente `NN_papel_acao.ext`.
2. Conferir transparência real, personagem inteiramente visível e ausência de resíduos ou retângulos de fundo.
3. Inspecionar visualmente em 64 px, 112 px e no tamanho final da tela, observando hachuras, proporções e expressão.
4. Manter a proporção original e não usar `object-fit: cover` quando cortar a silhueta.
5. Um SVG que contém PNG embutido é raster encapsulado. Para entregar um SVG genuinamente vetorial, usar paths, strokes e outras primitivas SVG reais.
6. Integrar somente onde necessário em `packages/wos-api/web/assets/`: arte decorativa com `alt=""` e arte informativa com texto alternativo apropriado.
7. Conferir as telas afetadas em desktop e mobile, sem alterar estados, permissões, fluxos ou endpoints do WOS.

### Checklist de aceite

- [ ] Prompt original aplicado literalmente e identidade do robô preservada.
- [ ] Monocromia preta, traço manual orgânico e hachuras sutis; sem cor, gradiente ou fotorrealismo.
- [ ] Transparência real e nenhuma parte da ilustração cortada.
- [ ] Formato de arquivo correspondente à natureza efetiva do asset, sem falso vetor.
- [ ] Leitura correta no tamanho de destino, com margens seguras.
- [ ] Marca WOS, paleta operacional e layout editorial preservados quando fora do escopo.
- [ ] Acessibilidade e telas afetadas revisadas.
