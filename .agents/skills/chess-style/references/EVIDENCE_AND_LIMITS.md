# Evidências, procedência e limites

## Convenção obrigatória

| Código | Significado | Exemplo |
|---|---|---|
| O | Observação direta da captura | Há um grid fino ao fundo |
| M | Medição dos pixels do arquivo recebido | O tom sólido dominante é `#E9E9E9` |
| I | Inferência plausível, não comprovada | O texto cinza pode fazer parte de uma revelação |
| P | Prescrição para uma nova implementação | Usar seis viewports de validação |
| N | Não verificado com a evidência disponível | Nome da fonte, easing e framework originais |

Confiança descreve a sustentação da afirmação, não a qualidade estética da proposta.
Uma prescrição pode ser específica sem ser uma medição. Nunca converter `P` em `M`
apenas porque foi escrita como um número exato.

## Material de base

Sete imagens foram fornecidas. As dimensões abaixo são **do raster**, não uma
certificação do viewport CSS, da resolução física da tela ou do zoom do navegador.

| ID | Dimensões do arquivo | Conteúdo relevante |
|---|---|---|
| S01 | 1409 × 833 | Personagem central pequeno, grid, espaço negativo e notas |
| S02 | 2048 × 1040 | Abertura editorial, serifas verdes monumentais e objeto central |
| S03 | 740 × 769 | Detalhe ampliado de letras, personagem, moeda e marginalia |
| S04 | 1865 × 1106 | Narrativa grande, vinhetas entre palavras e estados tonais |
| S05 | 1839 × 991 | Continuação narrativa, objeto recortado e personagem em ação |
| S06 | 1679 × 1011 | Spread de credibilidade, itálicos e documentos laterais |
| S07 | 2048 × 1104 | Tese final, outline, fila de personagens e rodapé |

Os recortes em `../evidence/` removem zonas com nomes, créditos e retratos. Cada
retângulo de extração, fonte e hash do arquivo de origem consta em
`../evidence/manifest.json`. Os IDs são locais ao pacote e não links para um site.

O texto de escopo fornecido foi usado para organizar a cobertura documental. Ele é
uma lista de tarefas e critérios, não uma prova de que cada componente ou tecnologia
mencionada existe na implementação observada. Por exemplo, pedir que se investiguem
carrosséis não comprova que um carrossel esteja presente nas capturas.

## Medições cromáticas

A contagem de pixels dos rasters identificou superfícies extensas nas seguintes cores:

| Papel | Valor | Fonte e limite |
|---|---|---|
| Fundo | `#E9E9E9` | Cor mais frequente nas sete imagens |
| Acento verde | `#00846A` | Segunda cor mais frequente em S02 e S03 |
| Tinta | `#000000` | Cor sólida recorrente nos textos de S04–S07 |
| Texto de baixo contraste | `#ACACAC` | Recorrente em S04 e S05 |
| Grid | `#D1D1D1` | Amostra recorrente em S03–S06; há antialiasing e variações |

A medição é válida para os pixels recebidos. Não prova o valor especificado em CSS,
a opacidade original, o perfil de cor do asset ou a técnica de composição.

## O que não foi confirmado

Não foram obtidos estilos computados, arquivos de fontes, código-fonte executável,
medidas DOM, gravações temporais, navegação de teclado da referência ou sessões
reais em seis dispositivos. A abertura do endereço em um leitor textual não forneceu
DOM/CSS suficiente para resolver essas lacunas. Não há alegação de inspeção dinâmica
completa ou pixel-perfect.

O site móvel não é deduzido da captura S03: ela pode ser apenas um detalhe recortado.
A posição repetida de cabeçalhos sugere persistência, mas não distingue `fixed`,
`sticky` ou composições capturadas em estados diferentes. Um objeto volumétrico na
imagem não demonstra WebGL. O aspecto de tinta não identifica o programa de desenho.

## Movimento: distinção indispensável

As capturas mostram texto preto, cinza e vazado. Elas não informam por si só a ordem
temporal, o ponto de ativação ou a duração. As receitas de reveal, parallax e transição
aqui documentadas são propostas compatíveis com esses estados visuais. Elas não
reivindicam reproduzir os parâmetros de uma implementação de origem.

Quando um agente futuro tiver acesso ao navegador, poderá adicionar observações de
execução mantendo data, viewport, estado, seletor, propriedade medida e ferramenta.
Não apagar as limitações anteriores nem substituir evidência por memória.

## Fontes técnicas externas

As recomendações sobre CSS, observação de interseção, preferências de movimento e
acessibilidade têm fontes próprias em `TECHNICAL_SOURCES.md`. Elas fundamentam a
**nova implementação**, não corrigem silenciosamente o que aparece nas capturas.

## Distribuição dos materiais

A skill contém princípios e instruções novas, mais recortes de estudo do material
fornecido. Não há autorização implícita para publicar esses recortes ou personagens
no produto final. Os ativos de produção devem ser novos, fornecidos para essa
finalidade ou devidamente liberados. Nenhuma fonte binária é distribuída.
