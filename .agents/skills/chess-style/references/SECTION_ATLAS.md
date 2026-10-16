# Atlas de seções e componentes

Cada blueprint separa a função observável de uma proposta nova de implementação.
Adaptar ou remover seções quando o conteúdo não as justificar. A experiência não
precisa vender um livro nem apresentar uma pessoa para conservar sua linguagem.

## A00 — Abertura silenciosa opcional

**Base visual:** S01; personagem pequeno, notas e grande vazio. **Função temporal:
não confirmada.** **Uso proposto:** breve prólogo visual ou estado de transição real,
nunca bloqueio artificial obrigatório.

Composição: uma vinheta central, pequena em relação ao viewport, sem card. O grid
permanece. Não adicionar texto “carregando” se não existe estado de carregamento.
Se a página já tem hero suficiente, remover o prólogo para preservar ritmo.

Em mobile, a figura precisa continuar reconhecível. Em modo estático, não criar
uma tela vazia que exija um evento para desaparecer. Aceite: acesso ao conteúdo não
depende de uma espera decorativa.

## A01 — Header editorial

**Base visual:** S02 e S04. Marca textual ou nome do projeto, navegação discreta,
metadados, ação sublinhada e eventual seletor compacto.

Objetivo: orientar sem parecer uma barra de produto corporativo. Tipografia menor
que o hero, mas legível. Fundo integrado à página; não introduzir blur ou sombra
pesada como padrão. Uma linha fina pode separar zonas quando necessária.

Na nova implementação, definir se o header é estático, sticky ou fixo conforme o
conteúdo. O fato de aparecer em capturas não decide essa escolha. Estados: normal,
hover, foco, seção atual e menu móvel aberto/fechado. Idioma só aparece se existir
conteúdo traduzido de verdade. Aceite: todos os destinos funcionam e o header não
oculta foco ou primeira linha das seções.

## A02 — Hero monumental

**Base visual:** S02/S03. **Mensagem:** uma tese curta e memorável.

Estrutura sugerida: linha de contexto → H1 de 1–3 linhas → objeto-síntese que
atravessa a composição → apoio editorial curto → indicação de continuidade ou CTA.
Não exigir que todos esses elementos existam. O H1 deve dizer algo específico.

Imagem: um grande recorte monocromático, não uma montagem de cinco mockups. Inserir
uma vinheta pequena em um ponto de pausa; distribuir notas nos vazios. O objeto não
esconde as palavras decisivas nem fica sobre a ação.

Motion proposto: apresentação curta e parallax moderado. Estado sem movimento:
composição final completa. Mobile: recompor a relação de objeto e título, reduzir
notas e testar palavras longas. Aceite: reconhecível sem animação, sem fonte externa
e sem rolagem horizontal involuntária.

## A03 — Primeiro manifesto: a tensão

**Base visual:** S04. **Função:** nomear um problema, crença ou situação.

Usar um ou dois parágrafos realmente curtos. Tipografia grande, entrelinha suficiente
e uma vinheta colocada em pausa semântica. É possível centralizar, desde que as linhas
permaneçam equilibradas. Não converter texto explicativo extenso em mural ilegível.

Os estados tonais observados inspiram reveal, mas o comportamento é uma proposta.
A implementação usa tinta secundária legível antes do realce. Um objeto lateral
pode atravessar uma área vazia, sem bloquear informação.

Mobile: reflow sem pinning; cena pode ir entre parágrafos. Aceite: a ideia é entendida
sem aguardar uma animação e sem precisar interpretar o desenho.

## A04 — Segundo manifesto: a mudança de entendimento

**Base visual:** S05. **Função:** apresentar uma nova lógica, não repetir o problema.

Continuar a superfície e a família tipográfica, alterando ritmo ou distribuição de
ênfase. Usar uma ação diferente para a vinheta, relacionada ao argumento. Um objeto
já introduzido pode reaparecer como continuidade, não como elemento aleatório.

Separar conceitualmente A03 e A04. Se o texto apenas reformular a mesma promessa,
fundir as seções. Aceite: cada bloco acrescenta informação e a passagem entre eles
faz sentido mesmo em leitura linear sem gráficos.

## A05 — Demonstração, produto ou evidência aplicada

**Status:** extensão de construção; a forma deste componente não é confirmada pelas
capturas. **Função:** dar substância à tese com material do novo projeto.

Uma prova pode ser uma página de amostra, diagrama, trecho de produto, processo em
etapas ou comparação real. Enquadrar como parte de um spread editorial, evitando
uma grade automática de features. Permitir legenda clara e observação específica.

Interações só entram quando necessárias: ampliar uma imagem, alternar duas vistas
ou navegar amostras. Não simular controles que não funcionam. Não inventar números,
clientes, resultados ou avaliações. Aceite: a evidência é verificável ou explicitamente
identificada como exemplo ilustrativo.

## A06 — Contexto e credibilidade

**Base visual:** organização de S06. **Função transferível:** mostrar trajetória,
origem, método, processo ou autoridade verificável do novo projeto.

Composição: marcador de contexto ao alto; narrativa ampla com itálicos seletivos;
artefatos pequenos nas margens com legendas. O marcador pode ser um objeto ou símbolo
do projeto. Nenhum retrato ou nome da referência deve ser transplantado.

Escolher evidências concretas próximas da afirmação que sustentam. Não substituir
isso por uma coleção de logos não autorizados. No mobile, trazer as evidências para
a ordem de leitura. Aceite: cada artefato tem função e nenhuma informação é inventada
para preencher a área lateral.

## A07 — Tese final e conjunto de personagens

**Base visual:** S07. **Função:** condensar a transformação desejada e encerrar o arco.

Usar frase curta em escala grande. Um trecho pode mudar de peso ou receber efeito
gráfico vazado, desde que o sentido permaneça acessível. Abaixo, uma fila de personagens
ou conjunto equivalente dá pluralidade e respiro. Variar silhuetas, gestos e espaços.

Não repetir o hero literalmente. A conclusão deve trazer o resultado do argumento.
A ação seguinte pode ficar nesta região ou no bloco imediatamente posterior.
Motion proposto: preenchimento de trecho e entrada discreta do conjunto. Mobile:
reorganizar personagens em composição menor, sem reduzir todos a pontos ilegíveis.

## A08 — Ação e rodapé editorial

**Base visual:** links textuais e acabamento de S07; lógica comercial específica
não foi inferida. **Função:** tornar o próximo passo evidente.

Usar um verbo real, destino real e contexto suficiente. Preço só aparece se for
fornecido. O rodapé reúne navegação secundária, informações próprias do projeto e
links existentes. Tipografia discreta e alinhamentos claros; sublinhado manual pode
ser um detalhe de assinatura visual sem reproduzir uma assinatura pessoal.

Não criar selos, redes sociais ou termos fictícios. Aceite: nenhuma âncora `#` usada
como substituto de ação, nenhum botão sem comportamento e nenhum crédito da
referência inserido no novo projeto.

## Como combinar o atlas

**Página curta:** A01 → A02 → A03/A04 fundidos → A05 → A07/A08.

**Página narrativa completa:** A01 → A02 → A03 → A04 → A05 → A06 → A07 → A08.

**Manifesto sem conversão comercial:** A01 reduzido → A02 → A03 → A04 → A07 → A08
com navegação editorial, sem CTA inventado.

A00 é sempre opcional. Não adicionar todas as seções para demonstrar que a skill
“capturou tudo”; completude da gramática não exige excesso na página final.
