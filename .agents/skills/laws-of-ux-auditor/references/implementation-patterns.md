# Catálogo de remediações de UX/UI — do achado ao critério de aceite

Use somente quando o problema correspondente for **observado ou validado**. Exemplos são genéricos e não descrevem nenhum sistema real. Não alterar identidade visual/estrutura de dados inadvertidamente.

## P01. Área de trabalho reduzida por chrome vertical

**Sintoma:** em viewport 1280×720, header global, título, tabs, barra de filtros e resumo ocupam altura e deixam o board/canvas com área insuficiente. **Evidência:** screenshot com viewport e dimensões CSS; DOM mostrando alturas; scroll para operar item.

**Alteração:** compactar ou contextualizar cabeçalhos redundantes, ajustar hierarquia de ferramentas, usar layout de `min-height: 0` em filhos flex/grid e reservar viewport útil ao workspace. Mover metadados extensos a painel/detalhe acessível, sem escondê-los permanentemente. **Teste:** criar/abrir/editar item em 1280×720, 1440×900, desktop grande e zoom; header/filtros continuam alcançáveis, nenhuma ação é cortada. **Princípios:** Miller (contexto/organização), Jakob (navegação previsível); Fitts apenas se posição real de alvos for problema.

Exemplo CSS **ilustrativo**, requer adaptação à arquitetura existente:

```css
.app-shell { min-height: 100dvh; display: grid; grid-template-rows: auto minmax(0, 1fr); }
.workspace { min-height: 0; display: grid; grid-template-columns: minmax(0, 1fr) minmax(18rem, 24rem); }
.workspace-main { min-height: 0; min-width: 0; overflow: auto; }
.workspace-detail { min-height: 0; min-width: 0; overflow: auto; }
@media (max-width: 60rem) {
  .workspace { grid-template-columns: minmax(0, 1fr); }
  /* Use drawer/modal acessível ou seção sequencial; não deixe painel inacessível. */
}
```

## P02. Alvo pequeno e ícones concorrentes

**Sintoma:** três botões de ícone de 16×16 CSS px sobrepostos ou colados em card. **Alteração:** aumentar hit area real e espaçamento; tooltip não substitui nome acessível; botões com rótulos apropriados; separar excluir de abrir. **Teste:** inspecionar bounding boxes e exceções WCAG 2.5.8, usar ponteiro, touch/teclado; cards não iniciam drag acidental. **Princípios:** Fitts; WCAG 2.5.8 e 2.1.1.

```css
.icon-action { display: inline-flex; align-items: center; justify-content: center;
  min-inline-size: 2rem; min-block-size: 2rem; }
.icon-action:focus-visible { outline: 2px solid currentColor; outline-offset: 3px; }
```

`2rem` é **proposta visual**, não garantia de WCAG em todos os casos: verifique área efetiva e geometria.

## P03. Navegação fragmentada

**Sintoma:** o usuário alterna entre lista, detalhe, “critérios”, “evidências” e “histórico” por páginas sem relação visível. **Alteração:** item como contexto primário e seções do detalhe relacionadas; permitir acesso à informação necessária ao decidir sem retornar ao menu global. Usar tabs/drawer apenas se corresponderem a tarefas distintas e preservarem estado. **Teste:** localizar proprietário, próximo passo e evidência relevante em uma jornada; verificar tempo/erros com baseline. **Princípios:** Miller, Jakob; Hick somente se custo de decisão observado.

## P04. Muitos controles e categorias pouco compreensíveis

**Sintoma:** menu com opções com nomes semelhantes e grupos sem semântica. **Alteração:** pesquisa de nomenclatura, agrupamento por tarefa, diferenciação de ações (criar, executar, avaliar, arquivar), busca/atalhos, preservar escolhas necessárias. **Teste:** tree testing ou sessão com 5–8 participantes por rodada como método prático de triagem qualitativa, **não** como garantia estatística; rotas e erros registrados. **Princípios:** Jakob; Hick com cautela (CHI 2020).

## P05. CTA visualmente dominante e semanticamente ambígua

**Sintoma:** botão de cor primária “Concluir” em item que ainda exige aprovação externa. **Alteração:** diferenciar “Enviar para revisão”, “Aprovar” e “Concluir”, com estado/permite explícitos. Prioridade visual acompanha semântica; não usar destaque para forçar decisão equivocada. **Teste:** pessoas interpretam corretamente a consequência sem ajuda, sistema impede transição inválida de modo claro. **Princípios:** Von Restorff, Jakob.

## P06. Indicador de progresso enganoso

**Sintoma:** barra “90%” sem cálculo definido em workflow aberto. **Alteração:** remover porcentagem não verificável e mostrar status, condições de aceite cumpridas/pendentes e próximos passos; usar `x de y` somente se `x` e `y` forem bem definidos. **Teste:** comparar indicador com estado de negócio, dependências e dados persistidos. **Princípios:** gradiente de meta, Miller; princípio ético de representação verdadeira.

## P07. Tarefa interrompida sem retomada

**Sintoma:** ao reabrir um formulário longo, alterações somem e progresso não é identificado. **Alteração:** rascunho/autosave condicionado ao domínio, estado `salvo/salvando/erro`, localização da última edição, próxima ação e recuperação de conflito. **Teste:** sair em estado intermediário, voltar após refresh e reabrir sessão; dados corretamente preservados, falhas recuperáveis. **Princípios:** continuidade operacional; a metanálise do Zeigarnik **não** prova a solução psicologicamente.

## P08. Formulário com erro global inespecífico

**Sintoma:** toast “Precondition failed” ao enviar; não indica campo/regra e some. **Alteração:** explicar regra em linguagem do domínio, localizar erro junto ao controle, foco no resumo/primeiro erro quando apropriado, indicar como corrigir e manter rascunho. **Teste:** submit inválido, teclado, screen reader (se acessível), resolução do erro e nova submissão. **Princípios:** Miller (não exigir memória), Jakob (padrões de erro), WCAG 3.3.1/3.3.3/4.1.3 quando aplicáveis.

## P09. Conflito de edição concorrente

**Sintoma:** item alterado por terceiro enquanto usuário edita, mensagem genérica impede continuar. **Alteração:** mostrar diferença entre rascunho local e versão atual, preservar texto, oferecer comparar/reaplicar/descartar sem sobrescrever silenciosamente. **Teste:** duas sessões concorrentes, permissão adequada, divergência de versões e restauração. **Princípios:** feedback, prevenção de erro, Miller (contexto presente). Cautela: garantir consistência transacional.

## P10. Responsividade de tela ampla perde densidade

**Sintoma:** largura grande aumenta vazios desproporcionais, ícones/textos permanecem minúsculos e informação útil ocupa fração da janela. **Alteração:** definir comportamento por breakpoint/container query, limites `min/max`, densidade proporcional, colunas adicionais quando beneficiam a tarefa, expansão progressiva do espaço de trabalho; preservar ordem e identidade. **Teste:** comparar telas com mesmo dataset entre 1280, 1440, 1920, 2560 CSS px, conferir hierarquia e número de decisões por scroll; não aumentar tudo indiscriminadamente. **Princípios:** ergonomia e legibilidade; não atribuir automaticamente às sete leis.

```css
:root { --content-gutter: clamp(1rem, 2vw, 2.5rem); }
.workspace-heading { padding-inline: var(--content-gutter); }
.cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 18rem), 1fr)); gap: clamp(.75rem, 1vw, 1.25rem); }
```

**Atenção:** regras de grid só são indicadas onde redistribuir cards faz sentido; não reordene colunas de kanban conceituais.

## P11. Canvas com conexões incompreensíveis

**Sintoma:** grafo de cards/arestas sem indicação de tipo de relação ou direção. **Alteração:** legenda semântica, rótulo da aresta/tooltip persistível, painel de detalhes com relações textuais, navegação por lista, breadcrumbs/contexto para seleção. **Teste:** usuário identifica dependência, responsável e próximo bloqueio sem interpretar apenas cor/posição; teclado consegue abrir relações. **Princípios:** Miller (reconhecimento), Jakob (sinais compreensíveis), Von Restorff (ênfase em estado relevante).

## P12. Arrastar é o único método de movimentação

**Sintoma:** para mover card entre colunas, só drag mouse. **Alteração:** menu “Mover para…” ou botões equivalentes, operável por ponteiro simples e teclado; anunciar status após mover e preservar foco. **Teste:** concluir mesma tarefa por teclado, touch e mouse sem drag; inspecionar WCAG 2.5.7 e 2.1.1. **Princípios:** WCAG e independência de modalidade; não usar Fitts como único critério.

## P13. Modais com excesso de informação / foco perdido

**Sintoma:** drawer ocupa tela HD e esconde ação principal; Escape fecha com alterações não salvas e foco volta ao topo da página. **Alteração:** área interna rolável, header/footer adaptativos, rascunho/sinalização de alterações, foco gerenciado na abertura e retorno ao trigger; ação destrutiva não é default de teclado. **Teste:** 720px de altura, 320px de largura, zoom, Tab, Shift+Tab, Esc, scroll, rascunho. **Princípios:** Miller + Fitts (casos específicos) + APG de modal.

## P14. Estado e seleção dependentes só de cor

**Sintoma:** tarefas atrasadas mudam de cinza para cobre sem texto/ícone; quem não distingue cores não compreende o status. **Alteração:** rótulo textual, formato/ícone redundante, contraste de texto e UI validado. **Teste:** leitura sem informação de cor; contraste dos estados aplicáveis; nome acessível. **Princípios:** Von Restorff (uso contextual) + WCAG 1.4.1/1.4.3/1.4.11.

## P15. Ação irreversível muito próxima da ação usual

**Sintoma:** “Excluir” ao lado de “Editar”, ambos em ícones pequenos e visual semelhante. **Alteração:** separação espacial, rótulos, estilo de risco, undo/confirm quando adequado ao domínio, atalho seguro. **Teste:** sem cliques acidentais no fluxo representativo, restauração possível quando prometida. **Princípios:** Fitts + prevenção de erro; Von Restorff apenas se contraste semântico coerente.

## Template de especificação de correção

```text
Componente/arquivo/rota:
Condição e estado afetados:
Comportamento atual (evidência):
Comportamento desejado:
Mudança visual + interação + copy:
Estados: default / focus / hover / disabled / loading / error / success
Restrições: domínio, identidade, performance, dados, permissões
Responsividade: largura e altura, zoom, overflow, touch
Acessibilidade: semântica, nome, teclado, foco, contraste, áreas de clique
Critério(s) de aceite:
Regressões a executar:
Como medir melhoria com dados reais:
```

## Ordem sugerida de implementação

Resolver bloqueios funcionais e exclusões severas; corrigir causas-raiz de navegação/contexto; melhorar formulários, feedback e estados; ajustar densidade/responsividade; otimizar microinterações sem mascarar problemas estruturais. A ordem final deriva do inventário real e da matriz de prioridade, **não** deste catálogo.
