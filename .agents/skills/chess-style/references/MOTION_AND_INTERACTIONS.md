# Motion Design e interações

## 1. Natureza desta especificação

As capturas mostram estados visuais, não timelines. As receitas deste documento
são **prescrições novas** compatíveis com a linguagem observada. Nenhuma duração,
curva, posição de scroll ou biblioteca abaixo é atribuída à implementação de origem.

O objetivo é fazer a leitura parecer progressiva, introduzir profundidade moderada
e construir pausas. Não “animar tudo”. A versão estática precisa preservar a mesma
tese, a mesma ordem e todas as ações.

## 2. Três classes de movimento

**Narrativo:** conduz a leitura ou uma mudança de compreensão. Exemplo: palavras
ficam mais escuras conforme a frase entra na zona de leitura.

**Espacial:** articula a relação entre texto e objeto. Exemplo: pequeno deslocamento
do objeto durante a passagem do hero.

**Funcional:** comunica estado de um controle. Exemplo: sublinhado de link, abertura
de navegação ou indicação de seleção. Essa classe precisa funcionar por teclado,
não apenas no hover.

Movimento ornamental sem papel claro deve ser removido. Não atribuir duração e easing
iguais a todas as classes: um estado de botão e uma passagem narrativa têm funções
e escalas de tempo diferentes.

## 3. Tokens propostos

| Token | Valor inicial | Aplicação |
|---|---:|---|
| Instantâneo | 0 ms | Modo estático, correção de estado |
| Micro | 140 ms | Cor, sublinhado e estados pequenos |
| Curto | 220 ms | Navegação e controles |
| Editorial | 600 ms | Entrada curta de vinheta ou bloco |
| Cena | 900 ms | Apresentação inicial opcional do objeto |
| Easing de saída | `cubic-bezier(.22, 1, .36, 1)` | Entrada que desacelera |
| Easing simétrico | `cubic-bezier(.65, 0, .35, 1)` | Transições bidirecionais |
| Parallax máximo largo | 24 px | Amplitude total pequena de ornamento |
| Parallax móvel | 0 px por padrão | Evitar deslocamento gratuito em touch |

Não aumentar uma duração apenas para tornar um efeito “cinematográfico”. Em
scroll-linked, progresso é uma função de posição; uma duração em milissegundos
não substitui a definição de distância e zona de leitura.

## 4. M01 — Apresentação inicial do hero

**Estado inicial:** página já legível; título no fluxo, objeto em posição reservada.
**Gatilho:** conteúdo crítico pronto, sem esperar todos os assets do documento.
**Propriedades:** uma translação de até 12 px e mudança de opacidade apenas em elemento
decorativo; o H1 não depende de terminar a animação para ser lido.
**Final:** composição estática exata. **Duração:** 600–900 ms, uma vez por navegação.
**Reversão:** não necessária. **Mobile:** menor deslocamento ou apenas fade decorativo.
**Reduzido:** estado final imediato.

Preloader não é obrigatório. Não bloquear acesso por um timer artificial. Quando
houver carregamento real que precise de representação, informar o estado sem prender
conteúdo já disponível nem simular progresso inexistente.

## 5. M02 — Leitura progressiva por palavras

**Estado inicial funcional:** texto inteiro presente, selecionável e com contraste
suficiente. A variante recomendada vai de `#686868` a `#000000` sobre `#E9E9E9`.
**Gatilho:** posição do parágrafo em relação a uma zona de leitura do viewport.
**Propriedade:** cor, eventualmente uma camada decorativa de preenchimento.
**Final:** todas as palavras com tinta sólida. **Reversão:** pode acompanhar o scroll,
mas nunca tornar texto invisível. **Mobile:** funcionar sem pinning. **Reduzido:**
todo texto em tinta sólida.

Uma função inicial possível:

```text
vh = altura útil do viewport
h  = altura do parágrafo
t  = posição do topo do parágrafo no viewport
p  = clamp((0.82 * vh - t) / (h + 0.40 * vh), 0, 1)
limiar_da_palavra_i = i / max(1, total_de_palavras - 1)
ativa = p >= limiar_da_palavra_i
```

Essa fórmula define uma proposta, não reproduz uma medição. Testar textos curtos e
longos. A primeira e a última linha devem ficar completas enquanto ainda podem ser
lidas confortavelmente; ajustar a zona quando o header ou a densidade alterar isso.

Para um preenchimento contínuo, distribuir uma pequena faixa de progresso por
palavra. Preservar texto semântico; não usar uma imagem do parágrafo. Não medir o
DOM de cada palavra em todo frame. Fazer leituras de geometria agrupadas e só depois
as escritas. Não exigir smooth scroll para calcular o progresso.

## 6. M03 — Outline para tinta

O estado vazado é observado no encerramento, mas a transformação temporal é proposta.
Usar uma camada de contorno e uma de preenchimento quando uma transição gradual for
necessária. O texto informativo base permanece disponível; camadas repetidas de
apresentação devem ser `aria-hidden="true"` para evitar leitura duplicada.

Preferir esse efeito em uma frase curta de conclusão, não em todo o corpo. Garantir
fallback sólido quando a propriedade de contorno não for suportada e no modo reduzido.
Não aplicar stroke grosso que altere a anatomia da fonte. Calibrar algo próximo de
0.6–1 px em texto grande como ponto de partida, avaliando a rasterização real.

## 7. M04 — Parallax do objeto

**Estado inicial:** objeto já visível e com área reservada. **Gatilho:** passagem da
seção pelo viewport. **Propriedade:** `transform` em um wrapper separado do recorte.
**Amplitude:** pequena, normalmente até ±12 px; rotação-base pode ser estática.
**Reversão:** sim, acompanha a posição. **Mobile:** desligado por padrão.
**Reduzido:** desligado. **Final:** o objeto não sai de uma área que exponha cortes
inesperados ou cubra controles.

Não aplicar movimento no mesmo nó que controla outras transformações sem compor os
valores. Um `transform` escrito por JS pode apagar a rotação definida em CSS. Usar
wrappers ou custom properties explícitas.

## 8. M05 — Vinhetas e fila de personagens

Uma vinheta pode aparecer com 4–8 px de deslocamento vertical e opacidade do próprio
asset, em 400–600 ms. O espaço deve estar reservado antes da entrada. Não deslocar
palavras quando a imagem aparece.

No encerramento, uma fila pode surgir como conjunto ou com pequenos atrasos de
30–60 ms entre grupos. Não animar cada membro, cabelo e acessório continuamente.
O usuário deve perceber a diversidade das silhuetas, não um looping distrativo.
No modo reduzido, mostrar o conjunto completo e estático.

## 9. M06 — Marginalia

Posição estável por padrão. Um pequeno deslocamento pode sugerir profundidade, mas
não girar notações indefinidamente. A escrita já comunica gesto sem precisar mexer.
Se houver parallax no objeto, deixar a maioria das notas quieta para criar contraste.

## 10. M07 — Links e ações

| Estado | Aparência | Comportamento |
|---|---|---|
| Normal | Texto claro, sublinhado ou distinção inequívoca | Link tem destino real; botão tem ação |
| Hover | Acento funcional e/ou mudança discreta do sublinhado | Não mover a área clicável |
| Focus-visible | Contorno visível com offset | Não depender apenas de troca de cor |
| Active | Contraste ou detalhe breve | Não comprimir o texto a ponto de parecer instável |
| Disabled | Aparência e semântica apropriadas ao controle | Não usar um link sem destino fingindo estar desabilitado |
| Loading | Label estável e estado informado | Evitar clique duplicado quando necessário |

Usar `<a>` para navegação e `<button>` para ações. Não recriar links com spans e
handlers. Sublinhado manuscrito pode ser um enriquecimento visual; o comportamento
precisa continuar reconhecível sem o asset.

## 11. M08 — Navegação compacta

Em mobile, uma abertura simples é preferível a uma transição longa. O botão possui
nome acessível, `aria-expanded` e `aria-controls`. Escape fecha e devolve foco ao
botão quando isso fizer sentido. Ao seguir uma âncora, não deixar o menu sobre o
conteúdo de destino. O header não pode esconder o elemento focado.

Um menu de navegação não é automaticamente um diálogo modal. Não implementar focus
trap em uma navegação não modal. Se o design realmente usar um diálogo, tratar o
foco e a semântica como diálogo, não apenas acrescentar atributos de forma decorativa.

## 12. M09 — Idioma, preview e controles opcionais

O controle de idioma visível em uma referência não obriga o novo projeto a ter
tradução. Só incluí-lo se houver conteúdo e rotas válidas para as opções oferecidas.
Preservar seção atual quando possível; não fazer o seletor parecer ativo sem mudar
nada. Testar strings mais longas.

Preview/carrossel é uma extensão opcional, não um componente confirmado pelas
capturas usadas neste pacote. Se necessário, fornecer controles anterior/próximo,
posição textual, teclado, toque e conteúdo acessível sem arrastar. Não habilitar
autoplay por padrão. Uma transição simples pode substituir uma simulação de folha
quando não houver necessidade narrativa ou orçamento de performance.

## 13. Máquina de estados operacional

```text
static (sem JS ou preferência reduzida)
  └─ init bem-sucedida + movimento permitido → enhanced

enhanced
  ├─ preferência reduzida / controle de leitura estática → static
  ├─ seção próxima do viewport → active
  └─ desmontagem / falha → cleanup + static

active
  ├─ scroll/resize → atualizar progresso em requestAnimationFrame
  ├─ sai da região útil → enhanced
  └─ leitura estática / desmontagem → static
```

Não rodar um loop infinito apenas para esperar eventos. Agendar um frame quando
scroll, resize, troca de fonte ou preferência exigir atualização. Cancelar listeners,
observadores e frames pendentes na desmontagem em aplicações SPA.

## 14. Como registrar um efeito realmente observado no futuro

Usar `templates/COMPONENT_SPEC.md`: seletor, evidência, viewport, estado inicial,
intermediário e final, gatilho, propriedades, unidade de progresso, reversão,
fallback e origem dos parâmetros. Um vídeo pode apoiar o timing; o estilo computado
pode apoiar a técnica. Um screenshot isolado não apoia os dois.
