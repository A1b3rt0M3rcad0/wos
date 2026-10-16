# Responsividade, leitura e acesso

Este documento prescreve como construir uma nova experiência utilizável. Não afirma
que a referência foi testada ou conforme aos critérios aqui estabelecidos.

## 1. Matriz de validação

| Viewport CSS proposto | Verificação principal |
|---|---|
| 1920 × 1080 | Limite de expansão, respiros, objeto e legibilidade de metadados |
| 1440 × 900 | Composição principal e equilíbrio das quebras do hero |
| 1280 × 800 | Header, altura disponível e sobreposições |
| 768 × 1024 | Conversão da margem documental e colunas |
| 390 × 844 | Navegação de toque, manifesto e CTA |
| 360 × 740 | Título longo, notas e áreas clicáveis |
| 320 CSS px de largura | Reflow, palavras longas e ausência de perda de conteúdo |

A matriz é um conjunto de testes, não os dispositivos confirmados das capturas.
Acrescentar paisagem, texto 200%, configurações maiores de fonte e idiomas reais do
projeto. Um screenshot a 740 px de largura não prova o comportamento mobile.

## 2. O que deve mudar no mobile

Recompor a abertura. Reduzir número e alcance das sobreposições. Colocar objetos
em uma posição que não oculte o título. Trazer artefatos marginais para o fluxo.
Mover vinhetas que estraguem a quebra da frase. Reduzir notas decorativas. Transformar
o header antes que os controles colidam. Desligar parallax que não contribua à leitura.

Manter: serifada de personalidade, relação entre escalas, cor escassa, continuidade
da superfície, um gesto desenhado legível e conclusão editorial. Mobile não deve
virar um template de cards porque o desktop ficou difícil de reduzir.

## 3. Contraste por função

Medições abaixo usam a fórmula de luminância relativa sRGB e o fundo sólido `#E9E9E9`.
São cálculos sobre tokens, não uma auditoria de contraste de todos os pixels renderizados.

| Cor sobre papel | Contraste aproximado | Decisão de projeto |
|---|---:|---|
| `#000000` | 17.30:1 | Texto principal |
| `#00846A` | 3.84:1 | Acento gráfico e texto grande; não padrão de texto pequeno |
| `#00745D` | 4.73:1 | Variante verde funcional para texto comum |
| `#686868` | 4.59:1 | Texto secundário legível e reveal funcional |
| `#ACACAC` | 1.87:1 | Só decoração ou camada não essencial |
| `#D1D1D1` | 1.26:1 | Grid puramente decorativo |

T4/T5 em `TECHNICAL_SOURCES.md` definem os critérios de referência: texto comum em
geral precisa de 4.5:1; texto grande, 3:1. Não usar o verde amostrado em legendas
pequenas só por fidelidade cromática. Manter dois tokens permite preservar a direção
de arte sem confundir cor de identidade e cor de interação.

Não baixar o contraste de uma informação importante para imitar uma captura de
transição. Se uma palavra ainda não “revelada” for necessária à compreensão, ela
precisa permanecer legível. Uma camada puramente decorativa pode manter a estética
fantasma enquanto o texto base continua acessível.

## 4. Texto, zoom e reflow

Definir tamanhos relativos e testar o comportamento final. Não desabilitar zoom
no viewport. Não usar altura fixa para parágrafos. Verificar acentos, descendentes,
itálicos e foco depois de ampliar texto.

`clamp()` ajuda a construir escala fluida, mas não dispensa inspeção. Um componente
pode precisar abandonar uma quebra de linha rígida sob zoom. `overflow-wrap` deve
resolver palavras realmente longas; não permitir fragmentação indiscriminada que
arruíne a composição em tamanhos normais.

Manter o texto essencial no DOM e selecionável. Não converter o manifesto inteiro
em SVG, canvas ou PNG apenas para obter encaixe exato. O hero também deve possuir
um título semântico apropriado.

## 5. Ordem de leitura

A ordem do DOM segue a narrativa, mesmo quando CSS produz posições assimétricas.
Não colocar uma evidência depois do rodapé e reposicioná-la visualmente no meio do
texto. Heading levels expressam estrutura, não tamanho: um manifesto grande pode
ser um parágrafo, e uma legenda não vira heading apenas para ficar em negrito.

Spans por palavra podem preservar a frase se espaços e elementos de ênfase forem
mantidos. Camadas de texto duplicadas para efeitos gráficos precisam ficar fora da
árvore acessível. Não esconder o texto real com `aria-hidden` e esquecer uma versão
semântica equivalente.

## 6. Imagens e decoração

Imagem decorativa recebe alternativa vazia quando apropriado. Elementos puramente
gráficos não entram na ordem de foco. Uma imagem informativa precisa de descrição
que capture seu papel, não de um relato de todos os pixels.

Não tratar qualquer personagem como decorativo por padrão: se sua ação comunica um
passo que não aparece no texto, fornecer equivalência textual. Não usar screenshot
da referência como banner sem explicar sua função e sem autorização adequada.

## 7. Navegação e controles

Incluir link para saltar ao conteúdo, foco visível e destinos reais. A ordem de Tab
é previsível. A navegação móvel informa abertura/fechamento. Controles de toque têm
meta de 44×44 CSS px; essa é uma decisão de projeto acima do mínimo AA de 24×24 com
exceções descrito em T5.

O tamanho visual de uma palavra pode continuar discreto enquanto padding amplia o
alvo. Não ampliar uma área clicável invisível sobre outro link. Header fixo ou sticky
precisa reservar espaço e usar `scroll-margin-top` nos destinos pertinentes.

Não adicionar custom cursor como requisito identitário: ele não é comprovado pelas
capturas e pode ser desnecessário. Se um projeto optar por enriquecimento do cursor,
manter o cursor nativo ou fallback inequívoco e não depender dele para indicar ações.

## 8. Movimento reduzido e modo estático

Respeitar a preferência do sistema e suas mudanças durante a sessão. Um controle
opcional de leitura estática pode oferecer uma escolha adicional, mas não deve
reativar movimentos quando a política do projeto exige respeitar a preferência
reduzida. O estado estático mostra texto completo, objeto em posição final e vinhetas
visíveis, sem espaços vazios aguardando animação.

Smooth scroll não é obrigatório. Em modo reduzido, usar navegação imediata ou uma
transição que não introduza movimento não essencial. Animações automáticas longas
ou repetitivas exigem avaliação adicional; esta skill prefere não introduzi-las.

## 9. Sem JavaScript e sob falhas

CSS base é a versão completa. Somente marcar a página como aprimorada depois que a
inicialização funcionar. Não usar `.js .content { opacity: 0 }` no início da página
sem uma forma robusta de recuperar conteúdo em falhas.

Sem JS, links continuam funcionando; navegação não fica permanentemente oculta;
manifesto fica completo; nenhuma imagem necessária depende de inserção tardia;
formulários reais, quando existirem, precisam de um caminho de submissão definido.
O starter não oferece um formulário fictício.

## 10. Teste manual necessário

Percorrer a página usando só teclado. Ativar texto maior. Bloquear fontes e imagens
remotas. Inspecionar com um leitor de tela representativo. Navegar para âncoras com
header ativo. Testar orientação horizontal. Selecionar e copiar uma frase animada.
Repetir com movimento reduzido e sem JS.

Testes automatizados e capturas ajudam, mas não substituem julgamento de leitura,
compreensão, foco real e acessibilidade completa. Registrar resultados como
“testado”, “não testado” ou “não aplicável”; não equiparar “sem erro de console” a
“acessível”.
