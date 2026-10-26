# EXEMPLO FICTÍCIO — auditoria de um board/canvas de trabalho B2B

**Objetivo didático:** demonstrar achados específicos, distinção entre observação/lei e critérios de aceite, sem afirmar que um produto real tem tais falhas. Todos os números são **classificações ilustrativas**; nenhum tempo de tarefa, ganho de conversão ou comportamento de usuários foi medido.

**Cenário simulado:** browser 1280×720 CSS px, viewport visual 100% de zoom, papel `editor`, quadro com 32 cards. O agente hipoteticamente consegue interagir com interface, inspecionar DOM e testar teclado; caso real, capturas, arquivos e resultados devem substituir essa ficção.

## Resumo e prioridades

| ID | Observação hipotética | Princípio pertinente | I | E | O | Confiança | Faixa |
|---|---|---|---:|---:|---:|---|---|
| UX-B01 | Header/toolbars deixam área do board insuficiente | Organização contextual; não atribuir automaticamente às 7 | 4.10 | 2.00 | 2.05 | medio | P1 |
| UX-B02 | Mover card só funciona com mouse drag, sem alternativa | WCAG 2.1.1/2.5.7 (não é Lei Fitts) | 3.90 | 2.33 | 1.67 | alto | P0* |
| UX-B03 | Critérios/evidências ficam em seções distantes | Miller (reconhecimento), Jakob (contexto) | 3.95 | 2.33 | 1.69 | medio | P2 |
| UX-B04 | Barra “85%” não possui definição de progresso | Gradiente de meta (aplicação imprópria) | 2.40 | 1.33 | 1.80 | medio | P3 |
| UX-B05 | Status crítico mostrado apenas por cor | WCAG 1.4.1; Von Restorff (contextual) | 3.40 | 1.67 | 2.04 | medio | P2 |

`P0*` é uma **exceção de bloqueio** no exemplo: uma função essencial, “mover card”, foi hipoteticamente verificada como inacessível por teclado e por alternativa sem drag; o índice não rebaixa a prioridade.

## UX-B01 — Área central do board parcialmente inutilizável em HD

- **Status:** `observed`; **local:** `/workspace/board`, viewport 1280×720.
- **Contexto e evidência simulado:** screenshot `board-1280x720.png` com medidas de 78px de header, 140px de resumo, 86px de tabs/toolbar e rodapé de 72px; área útil residual e botão de card cortado. Um relatório real forneceria artefato e medição DOM.
- **Passos:** 1. Abrir board; 2. selecionar card de uma coluna; 3. tentar abrir ações do fim do card.
- **Impacto no objetivo:** inspecionar e atualizar trabalho exige scroll externo e a ação não fica visível nesse estado.
- **Fundamento:** problemas de distribuição da área útil, visibilidade de estado e navegação contextual. Não usar Fitts como explicação principal sem verificar área/distância reais de apontamento.
- **Mudança:** recolher resumo não utilizado durante edição; mover detalhes secundários para drawer; `min-height:0` no container; reservar superfície central com overflow correto e ações acessíveis.
- **Impacto:** `S=4, C=5, R=3 → I=4.10`, alcance estimado. **Esforço:** `T=2,D=2,V=2 → E=2.00`; `O=2.05`.
- **Aceite:** no mesmo viewport a tarefa abre e salva com controle visível; em zoom 200% e desktop grande não há perda de conteúdo; teclado mantém foco visível.
- **Cautela:** eventual economia em segundos não pode ser reportada até ser medida.

## UX-B02 — Movimentação exclusiva por drag

- **Status:** `verified` **somente no cenário hipotético**; local: card no board.
- **Passos simulados:** navegar até card por Tab; pressionar Enter/Space, abrir menu; procurar ação de mover; não existe alternativa; arrastar via mouse funciona.
- **Evidência simulada:** gravação de tentativa com teclado e inspeção da UI; não se conclui que toda pessoa com deficiência seja incapaz sem testes adicionais, mas o requisito de método alternativo é verificável.
- **Fundamento:** WCAG 2.1.1 Keyboard e 2.5.7 Dragging Movements (avaliar as exceções); evitar imputar a Fitts, que descreve outra tarefa motora.
- **Mudança:** implementar menu sem drag “Mover para coluna…” e suporte de teclado com atualização de status e restauração de foco.
- **Impacto:** `S=5,C=4,R=2 → I=3.90`. **Esforço:** `T=2,D=2,V=3 → E≈2.33`; `O≈1.67`.
- **Prioridade:** **P0**, pela exclusão verificada de uma tarefa central no cenário simulado, não pelo valor do índice.
- **Aceite:** card move de A a B apenas por teclado e por ponteiro sem drag; feedback de sucesso é apresentado e foco não se perde; seleção, edição e drag originais continuam corretos.

## UX-B03 — Contexto de aceitação e evidência distante da tarefa

- **Status:** `observed` **simulado**; local: drawer de detalhes.
- **Evidência hipotética:** critérios em seção `/acceptance`, evidências em `/evidence` e ação “Avaliar” no card; é necessário alternar três páginas e relembrar qual regra estava pendente.
- **Fundamento:** Miller (reduzir demanda de recordação), Jakob (modelo local coerente). Isso **não** implica que todos os dados devam aparecer na mesma coluna visual.
- **Mudança:** no detalhe de uma tarefa, agrupar status de critérios, evidências associadas, autoria e próximo passo com links diretos às fontes; manter histórico acessível sem poluição visual.
- **Impacto:** `S=3,C=5,R=4 → I=3.95`. **Esforço:** `T=3,D=2,V=2 → E≈2.33`; `O≈1.69`.
- **Aceite:** participante ou avaliador consegue verificar um critério com sua evidência contextual sem copiar dados manualmente entre telas; persistência e autorização continuam corretas.

## UX-B04 — Progresso numérico sem regra

- **Status:** `observed` simulado; local: coluna “Meta”.
- **Evidência hipotética:** rótulo “85% concluído” embora existam critérios obrigatórios pendentes e nenhuma função de agregação documentada.
- **Fundamento:** gradiente de meta funciona em contextos específicos; usar porcentagem falsa compromete compreensão/trust.
- **Mudança:** mostrar “3 de 5 critérios verificados” e “2 pendentes” quando dado correto existir; para objetivos sem denominador, usar status qualitativo com próximos passos.
- **Impacto:** `S=3,C=2,R=2 → I=2.40`. `T=1,D=1,V=2 → E≈1.33`; `O≈1.80`.
- **Aceite:** indicador reflete sempre o estado persistido e diferencia critérios verificados de declarados.

## UX-B05 — Status expresso apenas por cor

- **Status:** `observed` simulado; local: tag visual de card.
- **Evidência hipotética:** cards bloqueados e livres possuem o mesmo rótulo “Aberto” e só mudam vermelho/cinza; uma pessoa em modo sem cores não diferencia.
- **Fundamento:** WCAG 1.4.1; Von Restorff explica distinção contextual de itens, **não** substitui redundância semântica.
- **Mudança:** texto “Bloqueado” e indicação textual de dependência, além de estilo cromático; revisar contraste e nome acessível.
- **Impacto:** `S=4,C=3,R=3 → I=3.40`; `T=1,D=2,V=2 → E≈1.67`; `O≈2.04`.
- **Aceite:** status interpretável sem cor e por leitor de tela (quando testado), sem quebrar densidade do card.

## Lições do exemplo

- Um relatório final **não** deve deixar valores divergentes entre tabela e detalhe: recompute a matriz antes da entrega e compare o índice com seus componentes.
- Não rotular como violação WCAG verificada sem evidência de execução.
- Uma visualização bidimensional pode ser exceção específica de reflow, mas sidebar e controles não ficam automaticamente isentos.
- Os indicadores de progresso dependem de regras reais do domínio, não de estética da barra.
