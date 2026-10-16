# Grid, geometria e composição

## 1. Dois grids independentes

O **grid gráfico** é visível: linhas finas sobre o fundo contínuo. O **grid estrutural**
organiza conteúdo, margens e alinhamentos. Eles podem conversar, mas não precisam
compartilhar o mesmo número de colunas nem obrigar cada letra a tocar uma linha.

Nunca interpretar a referência como um tabuleiro alternado de 8×8. As capturas
mostram uma malha linear, não casas pretas e brancas. Também não comprovam uma regra
única de quadrados perfeitos em todas as resoluções.

## 2. Grid gráfico proposto

Usar uma camada própria atrás do conteúdo, sem eventos de ponteiro. Duas imagens
CSS de gradiente linear podem formar as linhas; SVG ou outro método equivalente é
aceitável. A técnica é substituível e não foi identificada na referência.

```css
.chess-page {
  --ch-grid-x: clamp(7rem, 12.5vw, 14rem);
  --ch-grid-y: clamp(8rem, 12.5vw, 14rem);
  background-color: var(--ch-color-paper);
  background-image:
    linear-gradient(to right, var(--ch-color-grid) 1px, transparent 1px),
    linear-gradient(to bottom, var(--ch-color-grid) 1px, transparent 1px);
  background-size: var(--ch-grid-x) var(--ch-grid-y);
}
```

Os gradientes acima desenham linhas, não introduzem superfícies coloridas. Se o
alinhamento precisar permanecer estável entre seções, aplicar ao wrapper contínuo
da página. Uma camada `fixed` é outra decisão: não atribuí-la à referência sem teste.

Evitar que o grid interfira com letras finas. Quando a região ficar muito densa,
ajustar o contraste ou a origem da malha. Não colocar retângulos brancos atrás de cada
frase só para “limpar” o fundo.

## 3. Grid estrutural proposto

| Contexto | Colunas | Margem externa inicial | Estratégia |
|---|---|---|---|
| Desktop amplo | 12 | 4–5vw, com limite máximo | Hero expansivo; spread com margem documental |
| Laptop | 12 | 3–4vw | Reduzir sobreposição e resguardar controles |
| Tablet | 6 | 24–32 px equivalentes | Duas zonas apenas onde o conteúdo permitir |
| Mobile | 4 | 20 px equivalentes | Fluxo principal de uma coluna |
| Compacto | 4 | 16 px equivalentes | Tipografia expressiva sem texto cortado |

Usar `rem` para margens nas regras finais quando isso beneficiar preferências de
fonte e zoom. A tabela é uma receita inicial, não uma extração de breakpoints.
O limite de conteúdo pode ficar em torno de 1600–1760 px em desktop amplo, mas títulos
de impacto podem ultrapassar esse limite de forma local e deliberada.

## 4. Hero: composição e não template

O hero consiste em camadas. O título oferece a massa visual principal. O objeto
atravessa a junção das linhas ou outra área com vazio suficiente. O subtítulo e a
assinatura de contexto são menores. Notas vivem em bolsões de espaço residual.
Um desenho pequeno perto da baseline estabelece escala humana.

Não colocar o objeto em uma coluna à direita apenas porque esse é o padrão comum.
Também não forçar sobreposição se o título real ficar ilegível. A solução depende da
geometria concreta das palavras e do recorte do objeto.

Propostas para calibração inicial:

| Relação | Faixa de partida |
|---|---|
| Largura do conjunto tipográfico | 84–96% da largura útil |
| Ocupação vertical do hero | 70–105svh, sem altura fixa obrigatória |
| Objeto central / largura útil | 18–30% no desktop |
| Objeto central / altura do conjunto | 55–88% |
| Personagem / altura de uma capital | 18–38% |
| Notações / altura de uma capital | 5–10% |

Essas proporções são **P**. Não usá-las como trava: em um título longo, a largura
do objeto pode precisar cair. O conteúdo deve continuar legível sem o objeto.

## 5. Ordem de camadas

| Camada | Índice sugerido | Função |
|---|---:|---|
| Fundo e grid | 0 | Superfície contínua |
| Notas ornamentais de fundo | 1 | Atmosfera |
| Texto e estrutura | 10 | Informação principal |
| Vinhetas e objetos | 20 | Narrativa visual |
| Links/controles que exigem proteção | 30 | Interação |
| Header e navegação móvel | 50 | Navegação |
| Overlays funcionais reais | 100 | Diálogo, se o projeto precisar |

Usar `isolation: isolate` quando um bloco tem sua própria composição. Índices não
substituem compreensão de stacking contexts: transformações, opacity e posicionamento
podem criar contextos adicionais. Evitar um sistema que dependa de `z-index: 999999`.

Elementos puramente decorativos recebem `pointer-events: none`. Não aplicar essa
regra a um objeto que realmente seja um controle interativo.

## 6. Oclusão e clipping

A sobreposição é uma escolha expressiva; ocultação de conteúdo é um defeito quando
retira informação necessária. Proteger preço, CTA, navegação, estado e partes críticas
de palavras. Testar o hero em uma captura estática: ele não pode exigir movimento
futuro para tornar a mensagem compreensível.

Manter máscaras nos contêineres decorativos, não no `body` como correção universal.
Não ocultar rolagem horizontal antes de identificar o elemento que causa overflow.
Um elemento ornamental pode sair da tela; o texto funcional não deve sair.

## 7. Manifesto e linhas narrativas

O manifesto é um bloco de poucas frases, não toda a documentação do produto ampliada.
Compor linhas com pausas significativas e vocabulário concreto. Integrar uma vinheta
em posição previamente reservada. Evitar uma vinheta em cada linha: o contraste
entre presença e ausência cria o efeito.

Uma composição centralizada funciona com linhas equilibradas. Para texto muito longo,
reduzir escala, dividir em dois argumentos ou mudar alinhamento. Não condensar a fonte
artificialmente para preservar uma única linha.

A largura em `ch` depende da fonte: usar esse valor como ferramenta, não como prova
exata de caracteres por linha. Corpo editorial pode começar entre 45–65ch; manifesto
em grande escala costuma exigir medida bem menor. O texto real decide.

## 8. Spread assimétrico de credibilidade

No desktop, reservar uma margem de aproximadamente 1–3 colunas para evidências e
uma área mais ampla para narrativa. Um elemento contextual no topo introduz a seção.
Usar regular e itálico para ritmo, sem empilhar todos os trechos em headings.

A margem não é publicidade lateral. Vincular cada imagem à informação que ela apoia.
Se não houver prova real, reduzir a seção ou substituí-la por contexto verificável.
Em mobile, intercalar evidências e texto, sem uma coluna estreita residual.

## 9. Ritmo vertical

Trabalhar com três intensidades: impacto, leitura e respiro. Impacto pode usar uma
frase grande; leitura precisa de continuidade; respiro permite ver objeto ou figura
sem competir com muita informação. Não repetir hero monumental em todo viewport.

Tokens de espaço propostos: 4, 8, 12, 16, 24, 32, 48, 64, 96 e 144 px equivalentes.
Usar unidades relativas no CSS e espaçamento fluido de seção. Esses tokens governam
o sistema; ajustes ópticos pequenos permanecem permitidos e devem ser documentados.

## 10. Mobile: nova composição

No mobile, o objeto pode ficar entre título e subtítulo, ou parcialmente ao fundo
de uma região de letras menos densa. A vinheta pode sair da frase. A coluna de
evidências vira fluxo. As notas diminuem. A grade pode aumentar a legibilidade ao
ficar mais espaçada, mesmo em uma tela menor.

Não remover todos os sinais identitários: manter o contraste de escalas, pelo menos
um gesto manual relevante, fundo contínuo e a família editorial. Em vez de achatar o
hero inteiro, recompor sua relação principal.

## 11. Breakpoints e conteúdo

Os breakpoints de 640, 900 e 1200 px nos tokens são decisões propostas. Inserir outros
apenas quando o conteúdo os justificar. Quando o header não couber, transformá-lo
antes que os links encostem; não esperar uma categoria de dispositivo arbitrária.
Testar todos os tamanhos intermediários, não somente os endpoints.

## 12. Condições de aceite espacial

Zero overflow horizontal acidental nas larguras de teste. Nenhum controle sob uma
imagem. Nada essencial cortado por máscara. Grid consistente entre seções quando a
continuidade for intencional. Espaço suficiente para a maior variante de texto e
para fonte fallback. Hero reconhecível em desktop e mobile sem depender da mesma
coordenada absoluta. Foco visível não cortado pelos contêineres.
