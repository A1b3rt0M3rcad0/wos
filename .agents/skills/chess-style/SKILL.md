---
name: chess-style
description: >-
  Constrói e refina websites editoriais expressivos com tipografia serifada
  monumental de alto contraste, papel cinza-claro, grid delicado, acento verde,
  marginalia manuscrita, pequenas ilustrações em tinta, objetos monocromáticos
  recortados e narrativa por scroll. Use quando o usuário pedir chess-style,
  esta linguagem editorial ou uma experiência que combine rigor geométrico
  com desenho manual. Inclui direção de arte, implementação, responsividade,
  motion, tokens, especificações de assets e validação; não é um tema genérico
  de xadrez, um template de cards ou uma obrigação de usar peças e tabuleiros.
---

# chess-style

## Missão

Transformar o conteúdo real do usuário em uma experiência editorial na qual
**a tipografia constrói o espaço, a geometria organiza e o gesto manual humaniza**.
Entregar uma interface funcional quando houver pedido de implementação; entregar
especificações executáveis quando houver pedido de direção de arte ou planejamento.
Não limitar o resultado a uma análise estética nem implementar além do pedido.

O nome descreve uma gramática visual, não impõe o assunto da página. Adaptar símbolos,
metáforas e ilustrações ao domínio do projeto. Nunca reproduzir marcas, textos,
retratos, créditos ou personagens de referência como conteúdo do novo produto.

## Ativação e limites

Aplicar a landing pages narrativas, apresentações de produtos, manifestos,
portfólios, experiências culturais e editoriais. Em interfaces operacionais densas,
transferir apenas tokens, detalhes gráficos e hierarquia apropriados: não obrigar
tabelas, formulários ou dashboards a usar títulos gigantes e scroll coreografado.

Não chamar qualquer página cinza com uma fonte serifada de chess-style. O resultado
precisa preservar a tensão entre escala monumental, microrrepresentação desenhada,
objeto material, grade discreta e espaço negativo.

## Leitura progressiva obrigatória

1. Ler `references/DESIGN_DNA.md`, `references/VISUAL_FORENSICS.md` e
   `references/EVIDENCE_AND_LIMITS.md` antes da direção de arte.
2. Para construir: ler `references/TYPOGRAPHY_SYSTEM.md`,
   `references/LAYOUT_AND_COMPOSITION.md`, `references/GRAPHIC_LANGUAGE.md` e
   `references/REPRODUCTION_PLAYBOOK.md`.
3. Para implementar movimento e controles: ler `references/MOTION_AND_INTERACTIONS.md`,
   `references/IMPLEMENTATION_ARCHITECTURE.md` e `references/RESPONSIVE_ACCESSIBILITY.md`.
4. Para elaborar a página: consultar `references/SECTION_ATLAS.md` e
   `references/ASSET_INVENTORY.md`; não tratar seções opcionais como obrigatórias.
5. Antes de entregar: executar `scripts/validate_package.py` quando estiver
   verificando o pacote e aplicar `references/QUALITY_CHECKLIST.md` ao projeto.

Os tokens de `tokens/DESIGN_TOKENS.json` são a fonte de configuração. A folha
`tokens/tokens.css` é gerada por `scripts/build_tokens.py`. Nunca divergirem
silenciosamente. As evidências em `evidence/` servem à análise, não à produção.

## Invariantes de identidade

| Regra | Aplicação concreta | Rejeitar |
|---|---|---|
| Tipografia é estrutura | Uma composição display domina a abertura; textos narrativos têm papel espacial | Hero em duas colunas com título pequeno e imagem dentro de card |
| Papel neutro contínuo | Fundo-base `#E9E9E9`, sem superfícies concorrentes | Seções alternadas por gradientes e blocos coloridos |
| Geometria sussurrada | Grid de linhas finas abaixo do conteúdo | Tabuleiro preto e branco, bordas pesadas ou grid de cards |
| Cor escassa | Verde `#00846A` como acento gráfico; variante funcional mais escura | Paleta multicolorida de ícones e botões |
| Desenho com gesto | Personagens pequenos, traço irregular, manchas de tinta e ação legível | Mascotes 3D fofos, ícones uniformes, cartoon vetorial estéril |
| Materialidade | Um objeto recortado monocromático pode atravessar palavras | Mockup genérico dentro de retângulo com sombra |
| Marginalia com intenção | Pequenas anotações esparsas, relacionadas à metáfora | Ruído aleatório, símbolos sem função ou escrita sobre CTA |
| Movimento editorial | Revelar, conduzir leitura e pontuar transições | Scroll bloqueado, efeitos constantes e elementos competindo |
| Leitura preservada | Texto semântico, foco, contraste e versão sem movimento | Texto essencial apagado, cortado ou dependente de JS |

Os valores cromáticos acima foram medidos nas capturas, não no CSS de origem.
Todas as durações, breakpoints e métricas de implementação são propostas da skill.

## Workflow

### 1. Obter o problema e o conteúdo

Identificar público, ação principal, mensagem, evidências disponíveis, stack existente,
idioma, ativos e restrições. Usar `templates/CREATIVE_BRIEF.md`. Não inventar métricas,
clientes, resultados, preços, depoimentos ou integrações para preencher a página.
Perguntar somente o que bloqueia uma decisão material; registrar suposições restantes.

Respeitar o escopo: se o usuário pediu apenas planejamento, não modificar arquivos do
produto; se pediu implementação, inspecionar o projeto antes de escolher dependências.

### 2. Definir uma metáfora única

Escrever uma frase que conecte assunto, sistema espacial e imagens. Exemplo neutro:
“Decisões complexas tornam-se compreensíveis quando relações e consequências ficam
visíveis.” Derivar dela três famílias: notações, personagens em ação e objeto-síntese.
Não adicionar símbolos apenas por serem visualmente interessantes.

Preencher a matriz em `templates/ART_DIRECTION_MATRIX.md`: elemento, significado,
posição, escala, movimento, variante mobile e origem do asset.

### 3. Montar a narrativa antes dos componentes

Usar como esqueleto adaptável: abertura → tensão → explicação → demonstração/prova →
credibilidade → tese final → ação. Combinar ou remover etapas conforme o conteúdo.
Planejar uma alternância de momentos densos e respiros. Não converter cada etapa em
um card nem copiar a ordem de uma referência sem justificar sua função.

### 4. Resolver a tipografia antes dos ornamentos

Escolher uma família editorial adequada e testar título real, acentos, itálico,
algarismos, navegação e um manifesto de duas linhas. A família exata da referência
não foi identificada: não afirmar equivalência de uma substituta.

Ajustar largura, entrelinha e quebra de linha antes de ampliar a fonte. Desenhar
separadamente o hero largo e o hero móvel. Não usar `scaleX()` para fingir uma fonte
condensada nem depender de `white-space: nowrap` em títulos longos.

### 5. Construir a versão estática completa

Implementar semântica, navegação, tipografia, grid, imagens com dimensão reservada,
texto, CTA e rodapé. A página deve funcionar sem JavaScript e sem fonte externa.
O starter em `examples/starter/` demonstra a estrutura, não substitui direção de arte
nem inclui ilustrações de produção ou uma fonte editorial licenciada.

### 6. Integrar os ativos e as camadas

Usar assets específicos do projeto. Separar um grande objeto recortado de pequenas
vinhetas desenhadas. Manter as áreas interativas livres. Controlar camadas e recortes
localmente; não esconder todo overflow da página para mascarar erros de composição.
Registrar permissões, dimensões, finalidade e variantes em um manifesto de assets.

### 7. Adicionar movimento por melhoria progressiva

Um pequeno número de sequências fortes supera animação em todos os elementos.
Priorizar revelação tipográfica, transição do objeto e entrada discreta de vinhetas.
Não inventar durações observadas: usar receitas propostas e calibrar em execução.
Oferecer modo estático/reduzido; desabilitar parallax em touch quando prejudicar leitura.

### 8. Compor mobile como uma edição própria

Testar 1920×1080, 1440×900, 1280×800, 768×1024, 390×844 e 360×740. Testar também
320 CSS px para reflow e zoom de texto a 200%. As dimensões das imagens recebidas
não comprovam os viewports ou breakpoints da referência.

Manter tipografia expressiva e retirar excesso de ornamentos. Converter margem
lateral em conteúdo no fluxo. Diminuir sobreposição, não apenas a página inteira.

### 9. Validar e entregar

Comparar a página a `references/QUALITY_CHECKLIST.md`. Testar sem JS, com fonte
indisponível, teclado, movimento reduzido, textos longos e imagens ausentes.
Registrar o que foi efetivamente testado usando `templates/VALIDATION_REPORT.md`.
Não declarar teste de navegador, precisão pixel-perfect ou cobertura total sem evidência.

## Contrato de saída

Para implementação, entregar arquivos funcionais na stack do projeto, direção de arte
aplicada, estados responsivos, controles reais e relatório breve de validação. Não
entregar somente explicações ou uma captura de tela em lugar do código solicitado.

Para planejamento, entregar sequência de execução, mapa de seções, decisões tipográficas,
inventário de ativos, interações, requisitos de legibilidade e critérios de aceite.

Para uma nova skill ou pacote, preservar os caminhos relativos e a distinção entre:
**observação visual**, **medição em captura**, **inferência**, **prescrição** e
**informação não verificada**.

## Regras de precisão

Não atribuir framework, biblioteca, formato de imagem, fonte ou técnica de animação
à referência sem inspeção confirmatória. Não confundir objeto com aparência 3D e
modelo 3D interativo. Não usar uma captura intermediária para concluir que o texto
permanece cinza no estado final. Não atribuir ausência de algo na imagem à ausência
no site inteiro. Valores de acessibilidade e engenharia estão identificados como
critérios da implementação e acompanhados de referências técnicas próprias.

Não incorporar nomes ou créditos da referência nos artefatos novos. Não empacotar
fontes do sistema, arquivos de terceiros ou evidências como ativos distribuíveis do
produto. Não reutilizar a imagem de uma pessoa como identidade de outra organização.
