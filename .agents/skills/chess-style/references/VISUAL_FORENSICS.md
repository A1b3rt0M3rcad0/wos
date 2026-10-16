# Anatomia visual consolidada

Este documento descreve o visível. Medidas recomendadas estão nos manuais de
construção. Os códigos de evidência são definidos em `EVIDENCE_AND_LIMITS.md`.

## 1. Superfície

**O/M, confiança alta.** O fundo das capturas é cinza-claro, de aparência uniforme,
com cor sólida dominante `#E9E9E9`. Não há evidência suficiente de textura de papel
aplicada ao fundo. O efeito editorial resulta principalmente da cor, do desenho e
da tipografia, e não exige uma textura de ruído global.

Linhas ortogonais finas atravessam a superfície, inclusive zonas vazias. Trata-se de
um grid gráfico. Sua presença não comprova que todas as letras, imagens ou seções
obedeçam à mesma malha estrutural. Em S03 há intervalos de aproximadamente 222 px
no raster; em outras capturas a escala aparente é diferente.

## 2. Abertura de baixa densidade

**O, confiança alta.** Em S01, um personagem desenhado pequeno fica próximo ao centro,
rodeado por muito vazio e algumas notações. O personagem tem contorno manual,
roupa clara e áreas pretas compactas. O estado lembra uma abertura ou transição,
mas sua função temporal e eventual associação a carregamento são **I/N**.

Não concluir que a experiência exija um preloader. Um novo projeto pode usar essa
composição como vinheta estática sem atrasar acesso ao conteúdo.

## 3. Abertura monumental

**O, confiança alta.** Em S02, duas linhas de capitais verdes ocupam a maior parte
da largura. As serifas são finas e o contraste de hastes é alto. O eixo central é
atravessado por um objeto monocromático vertical de aparência escultórica, com
elementos associados a papel, relevo e textura. O objeto passa à frente das letras.

O topo contém uma navegação textual, um detalhe de edição, um controle compacto de
idioma e uma ação sublinhada com informação secundária menor. Não há uma barra
pesada preenchida por cor. O hero combina precisão editorial com uma escala de
objeto incomum e pequenos desenhos quase escondidos na composição.

## 4. Microescala e letra

**O, confiança alta.** S03 torna visível o contraste entre letras enormes e uma figura
humana muito pequena. A figura tem pernas finas, corpo escuro e traços irregulares.
Uma curta linha horizontal sugere chão. O desenho não está dentro de um badge ou
círculo: compartilha diretamente o espaço da baseline.

Há uma moeda recortada com textura e iluminação próprias, notas manuscritas em cinza
e a grade ao fundo. Esses elementos pertencem a três linguagens distintas que se
mantêm coesas pela ausência de cores concorrentes e pela hierarquia de escala.

## 5. Narrativa tipográfica

**O, confiança alta.** Em S04 e S05, frases longas são apresentadas em serifada de
grande escala. Pequenas vinhetas interrompem visualmente trechos entre palavras.
Figuras em ação se relacionam a esforço, deslocamento e situações cotidianas.

O texto combina preto sólido e cinza claro no mesmo bloco. Em alguns pontos o estado
é intermediário entre escuro e claro. Isso é compatível com uma progressão de leitura,
mas a mecânica temporal não é medida. A imagem também corta palavras nas bordas:
sem o contexto de captura, não é possível atribuir esse corte ao design final, a uma
translação ou ao próprio recorte da imagem.

Não transportar clipping de texto essencial para uma nova página como “regra de
fidelidade”. Conservar a escala e a composição, resolvendo a leitura no viewport.

## 6. Imagens materiais atravessando o fluxo

**O, confiança alta.** Objetos monocromáticos relacionados a papel aparecem perto de
bordas, em diagonais e sobre algumas linhas de texto. A perspectiva e as dobras
produzem sensação de profundidade. Sua presença em mais de uma região estabelece
continuidade temática. Movimento, trajetória e velocidade permanecem **N**.

## 7. Spread de credibilidade

**O, confiança alta.** S06 organiza um bloco editorial amplo com texto centralizado,
itálicos seletivos e pequenas imagens documentais na margem. Um marcador visual
circular aparece no alto; o rosto não é necessário à gramática extraída e não é
incluído nos recortes. Há legendas pequenas, aparentemente de voz utilitária, sob
os documentos laterais.

A distinção é espacial: o texto principal conta uma trajetória e as imagens marginais
funcionam como evidência. Não é uma grade de logos, uma ficha de currículo ou um
conjunto de cartões de depoimentos.

## 8. Encerramento

**O, confiança alta.** S07 contém uma frase de grande escala e peso mais marcado,
com palavras pretas, cinza e de contorno. Uma fila de personagens pequenos aparece
abaixo. A diversidade se dá por silhueta, altura, postura, acessórios e expressões,
não por paleta multicolorida. O rodapé mantém texto e links discretos, além de um
sublinhado de aparência desenhada.

## 9. O que define a unidade

A unidade vem da repetição de cinco relações: fundo contínuo; tipografia sempre
importante; desenhos muito menores que as letras; detalhes manuscritos nas margens;
e recortes volumétricos acromáticos. A alternância de regular, itálico, bold e outline
adiciona mudança de tom sem transformar cada seção em outra identidade.

## 10. Tradução para construção

Preservar essas relações e redesenhar seu conteúdo. Os exemplos em `SECTION_ATLAS.md`
são blueprints novos. O conjunto não pressupõe a existência de formulários, modal,
carrossel, menu móvel específico, loading obrigatório, custom cursor ou checkout.
Esses recursos só entram quando o projeto realmente precisa deles.
