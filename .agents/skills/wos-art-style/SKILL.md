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
