# Checklist operacional de cobertura de UX/UI web

Use o checklist como **roteiro de investigação**, não como lista de violações presumidas. Para cada item registre: `OK verificado`, `problema verificado`, `hipótese`, `não testado` ou `não aplicável`. Cada problema vira finding com evidência específica e hipótese de mecanismo. Em caso de dúvida, não marque reprovação.

## A. Orientação, arquitetura da informação e navegação

- [ ] Usuário compreende finalidade da tela, contexto ativo, onde está e o próximo passo possível.
- [ ] Navegação principal, secundária, breadcrumbs, voltar e atalhos não mudam de semântica entre telas sem razão.
- [ ] Rótulos de menus representam linguagem e tarefas do usuário; ícones sem rótulo são compreensíveis no público-alvo.
- [ ] Busca, filtros e categorias funcionam com coleções grandes e resultados vazios.
- [ ] Ações frequentes ficam descobríveis sem esconder funções críticas em menus obscuros.
- [ ] Mudança entre entidades relacionadas preserva contexto (seleção, filtros, item ativo, origem/retorno).
- [ ] URLs, tabs e painéis têm comportamento previsível no botão Voltar e em links diretos.
- [ ] Há caminhos de recuperação de rota sem permissão, item não encontrado, link quebrado e estados inconsistentes.
- [ ] Avaliar Jakob para padrões aprendidos; Hick para decisões apenas quando aplicável; Miller para contexto não memorizado.

## B. Hierarquia, conteúdo, legibilidade e densidade

- [ ] Título, descrição, resultado, ações primárias/secundárias e informações de status são diferenciáveis.
- [ ] Controles importantes têm tamanho, posição e rótulo apropriados ao uso; ações secundárias não competem desnecessariamente.
- [ ] Texto, alinhamento, contraste, espaçamento, tamanho de fonte e comprimento de linha preservam legibilidade.
- [ ] Layout mantém proporções intencionais em telas 1280–2560 CSS px, inclusive altura de 720/900; não desperdiça espaço crítico nem estica conteúdo arbitrariamente.
- [ ] Conteúdo mais usado está disponível sem overlays supérfluos; colunas e detalhes podem ser compreendidos com os dados à vista.
- [ ] Cores semânticas diferenciam estado e não dependem exclusivamente de cor.
- [ ] Dados densos têm agrupamento, filtros, hierarquia e navegação eficiente sem perda de precisão.
- [ ] Elementos vazios não deslocam contexto nem criam “salas vazias” sem orientação.
- [ ] Von Restorff para distinção significativa; Miller para reconhecimento/contexto; Fitts para aquisição de controles.

## C. Fluxos, tarefas e modelos mentais

- [ ] Cada jornada tem entrada, objetivo, ação principal, confirmação e saída claros.
- [ ] Tarefas centrais não exigem saltos repetidos entre seções nem reentrada de dados existentes.
- [ ] Etapas e dependências representam o negócio real, sem wizard artificial nem omissão de escolhas necessárias.
- [ ] Os caminhos diferenciam criar, editar, cancelar, salvar, desfazer, arquivar e excluir.
- [ ] A sequência para usuário iniciante é descobrível; atalhos de usuário experiente permanecem disponíveis.
- [ ] Estados de “em andamento”, “aguardando”, “bloqueado”, “concluído” e “validado” não se confundem.
- [ ] Progresso exibido corresponde à regra efetiva de conclusão; não se usa barra inventada para tarefas abertas.
- [ ] Hick/Jakob para decisões, Miller para transferência de contexto, gradiente de meta para metas verificáveis.

## D. Componentes interativos e feedback

- [ ] Hover, focus, active, pressed, selected, loading, disabled e error possuem sinais distinguíveis.
- [ ] A área clicável real, o espaçamento e o alvo visual correspondem à intenção da ação.
- [ ] Botões e links usam elementos semânticos e operações próprias; toda ação informa resultado ou falha.
- [ ] Menus, popovers, dialogs, tooltips, dropdowns e tabs abrem, fecham e devolvem foco coerentemente.
- [ ] Teclado executa ações sem depender de hover ou drag; ponteiro não executa ação destrutiva por acidente.
- [ ] Notificações temporárias não são única fonte de erro ou resultado crítico; feedback durável é localizável.
- [ ] Ações assíncronas não permitem submissão duplicada ou estados de loading intermináveis sem recuperação.
- [ ] Fitts para targets; Jakob para controles familiares; heurísticas de status e prevenção de erro complementares.

## E. Formulários e entradas

- [ ] Rótulos persistentes e valores esperados estão claros; placeholder não substitui label.
- [ ] Campos obrigatórios, opcionais, limites, formatos e dependências são evidentes antes do erro quando relevante.
- [ ] Ordem do formulário corresponde ao fluxo mental; longos grupos têm seções significativas.
- [ ] Erro identifica campo, causa e como corrigir; mantém valores já digitados.
- [ ] Mensagens funcionam para teclado/leitor de tela; aria-describedby e associação estão corretas quando implementadas.
- [ ] Há prevenção de duplicidade, confirmação de ações irreversíveis, autosave/rascunho quando apropriado.
- [ ] Data/hora/número/currency respeitam locale e usam controles apropriados, sem impor memorização.
- [ ] Revisão do formulário mostra o que será submetido em ações importantes; não são ocultadas informações necessárias.
- [ ] Miller para contexto; Hick para escolhas relevantes; gradiente de meta apenas com progresso verdadeiro.

## F. Responsividade, viewport e expansão

- [ ] Testar pelo menos desktop 1280×720, 1440×900, 1920×1080 e grande 2560×1440 quando disponível.
- [ ] Testar mobile 390×844, viewport estreito 320 CSS px e zoom 200%/400% no browser; registrar impossibilidades.
- [ ] Verificar alturas curtas: sticky header + navigation + workspace deixam área **utilizável** para conteúdo principal.
- [ ] Não há recorte, sobreposição, controls offscreen irreversíveis, campos ilegíveis ou scrolling inesperado.
- [ ] Títulos longos, 0/1/centenas de itens, erros extensos e traduções longas não quebram layout.
- [ ] Quadros, mapas, canvas e tabelas usam scroll bidimensional **apenas quando estruturalmente necessário**; o resto da página reflowa.
- [ ] Componentes ajustam densidade com critérios contextuais (min/max, clamp quando apropriado), mantendo proporções intencionais.
- [ ] Há rolagem/hit area/teclado em contextos com nested scroll, sidebar e drawer.
- [ ] Comparar requisitos específicos WCAG 1.4.10, 1.4.4, 2.4.11, 2.5.8 (ver referência normativa).

## G. Acessibilidade e uso inclusivo (conjunto inicial, não auditoria total WCAG)

- [ ] Semântica estrutural: headings, landmarks, labels, alt/nomes acessíveis.
- [ ] Contraste textual e não textual verificado por computação de cores e estados aplicáveis, não apenas por percepção.
- [ ] Navegação por teclado e foco visível sem trap ou foco encoberto.
- [ ] Alvos de ponteiro cumprem o critério WCAG pertinente, com exceções verificadas.
- [ ] Erros, mensagens de status, mudanças dinâmicas e estados são anunciados apropriadamente.
- [ ] Nível de zoom, reflow e redimensionamento de texto não perdem funcionalidades.
- [ ] Movimento/animation é controlável quando necessário, e redução de movimento respeita preferência do sistema.
- [ ] Autenticação e introdução redundante de informações verificadas contra WCAG aplicável.
- [ ] Fluxos baseados em arraste oferecem alternativa sem arraste quando aplicável (SC 2.5.7).
- [ ] Não declarar conformidade WCAG global com checklist parcial ou automação isolada.

## H. Estados do sistema e recuperação

- [ ] Empty state responde “o que é isto / como começar / o que aparece aqui?”.
- [ ] Loading mantém estrutura suficiente, anuncia progresso relevante e possui timeout/recuperação quando pertinente.
- [ ] Erros de rede, permissão, validação, conflito de edição e item movido oferecem ação de recuperação.
- [ ] O estado local do usuário é preservado quando há conflito, quando seguro, sem sobrescrever trabalho recente.
- [ ] Undo, histórico, versões e confirmação aparecem conforme reversibilidade e valor da informação.
- [ ] A interface deixa claro quando houve salvamento automático, quando ainda falta salvar e quando houve falha.
- [ ] A retomada de trabalho não se baseia na suposição de memória melhorada por Zeigarnik.

## I. Componentes ricos: board, kanban, canvas, editor e dados

- [ ] O espaço central de trabalho ocupa a área principal em viewports limitados; chrome não domina a janela.
- [ ] O item expõe seu título, status, proprietário, próximo passo, critérios, evidências e histórico conforme relevância ao caso de uso.
- [ ] Cards distinguem seleção, abertura, drag, menu contextual e ações inline sem ambiguidades.
- [ ] Conexões/dependências são navegáveis, rotuladas e compreensíveis fora de uma visualização puramente espacial.
- [ ] Zoom e pan não causam perda de foco; existem alternativas por teclado e modos de navegação não-espacial.
- [ ] Seleção de item preserva contexto do quadro (scroll/zoom/filtros), e detalhe pode ser fechado sem perder posição.
- [ ] Controles de pan/drag/drop não substituem ações por clique/teclado/touch de forma exclusiva.
- [ ] Altas quantidades de nós/cards, bordas da tela e nomes longos permanecem operáveis.
- [ ] Rótulos e prioridades não dependem só de cor/posição; agrupamento e filtragem são acessíveis.
- [ ] Os dados são consistentes em board, lista e detalhe; conflitos de versão são explicados de modo recuperável.

## J. Conteúdo, idioma e confiança

- [ ] Microcopy descreve resultado concreto e evita jargão ambíguo não compreendido pelo público.
- [ ] Promessas sobre progresso, prazo, sucesso e estado têm fundamento.
- [ ] Termos sensíveis (“publicar”, “confirmar”, “aprovar”, “excluir”) comunicam irreversibilidade real.
- [ ] Idiomas e textos longos não comprometem truncamento sem alternativa de expansão.
- [ ] Feedback de validação usa linguagem de ação e distingue falha de regra de negócio de erro temporário.
- [ ] Nenhum mecanismo depende de coerção, urgência falsa, caixa pré-marcada enganosa ou ocultação de informações relevantes.

## Cobertura mínima a constar no relatório

Apresente uma tabela por eixo (A–J) com status, telas representativas testadas, evidência, lacunas, motivo da lacuna e nível de confiança. Não confunda “não observei bug” com “cobertura completa”. Em auditoria de apenas uma screenshot, o relatório pode conter maioria de itens como **não testados** e deve ser tratado como revisão visual parcial, não auditoria exaustiva.
