# Acessibilidade: verificação orientada à WCAG 2.2

Este arquivo fornece uma **amostra operacional de critérios recorrentes**; não é uma auditoria WCAG 2.2 exaustiva. Uma análise completa exige verificar todos os critérios aplicáveis no nível contratado e fluxos/estados relevantes, inclusive com pessoas e tecnologias assistivas quando necessário. Na auditoria de UX, separe `falha verificada`, `suspeita`, `recomendação adicional` e `não testado`.

**Fonte normativa:** https://www.w3.org/TR/WCAG22/; explicações: https://www.w3.org/WAI/WCAG22/Understanding/; padrões de interação: https://www.w3.org/WAI/ARIA/apg/.

## Matriz de verificações (referência por critério)

| WCAG 2.2 SC | Nível | O que testar | Cautela |
|---|---|---|---|
| **1.1.1 Non-text Content** | A | Imagens informativas, botões de imagem, equivalentes textuais conforme função. | Elemento decorativo tem tratamento diferente de conteúdo significativo. |
| **1.3.1 Info and Relationships** | A | Estrutura semântica de headings, rótulos, listas, grupos e tabelas. | Formatação visual sozinha não estabelece semântica. |
| **1.3.2 Meaningful Sequence** | A | Ordem significativa de leitura DOM e navegação quando apresentação muda. | Ordem visual não prova ordem programática. |
| **1.3.5 Identify Input Purpose** | AA | Autocomplete/propósito programaticamente identificável de campos pessoais aplicáveis. | Nem todo input é elegível à taxonomia deste critério. |
| **1.4.1 Use of Color** | A | Estado/erro/ênfase não dependem só de diferenciação cromática. | Cor pode complementar texto, ícone e forma. |
| **1.4.3 Contrast (Minimum)** | AA | Texto normal `≥4.5:1`; texto grande `≥3:1`, exceções normativas. | Logomarcas, textos incidentais/inativos e texto grande seguem regras próprias; não arredondar valores abaixo do limiar. |
| **1.4.4 Resize Text** | AA | Ampliação de texto até 200% sem perda de conteúdo/funcionalidade, salvo exceções. | Diferenciar zoom do browser, resize text e device pixel ratio. |
| **1.4.10 Reflow** | AA | Conteúdo de leitura em `320 CSS px` de largura efetiva para scroll vertical sem precisar scroll em duas dimensões; 256 CSS px de altura para scroll horizontal. | Exceção para conteúdo cujo uso/significado exige dimensão dupla (mapas, canvas, certas tabelas). A exceção **não** se aplica indiscriminadamente ao resto da página. |
| **1.4.11 Non-text Contrast** | AA | Contraste `≥3:1` para limites/estados de controles e objetos gráficos necessários. | Componentes inativos e elementos com exceções exigem análise separada; não confundir com 1.4.3. |
| **1.4.12 Text Spacing** | AA | Regras de espaçamento de texto com valores de teste WCAG sem cortar texto/funcionalidade. | Não exigir que site use esse espaçamento por padrão; deve tolerá-lo. |
| **1.4.13 Content on Hover or Focus** | AA | Elementos auxiliares que surgem com hover/focus podem ser dispensados, acessados e persistem conforme exceções. | Aplica-se quando conteúdo adicional é mostrado nessas condições. |
| **2.1.1 Keyboard** | A | Todas as funcionalidades aplicáveis operáveis por teclado. | Drag de mouse sem alternativa pode excluir usuários; teste real obrigatório. |
| **2.1.2 No Keyboard Trap** | A | Entrada/saída de foco possível, salvo comportamento explicitamente documentado de componente modal conforme boas práticas. | Modal deve manter foco enquanto aberto e permitir fechar, quando apropriado; não confundir trap indevido com contenção de modal. |
| **2.4.3 Focus Order** | A | Sequência significativa para operação e entendimento. | Ordem de Tab = inspeção interativa, não screenshot. |
| **2.4.7 Focus Visible** | AA | Indicador visível de foco do teclado. | Não depende de hover com mouse. |
| **2.4.11 Focus Not Obscured (Minimum)** | AA | Foco não fica **totalmente oculto** por conteúdo criado pelo autor (sticky header, banner etc.). | Parcialmente encoberto não falha necessariamente este SC, embora possa ser ruim. |
| **2.4.13 Focus Appearance** | AAA | Área/contraste mínimos do indicador de foco conforme definição. | Não rotular como requisito AA. |
| **2.5.3 Label in Name** | A | Nome acessível inclui texto do rótulo visível em componentes com labels. | Importante para acionamento por voz. |
| **2.5.7 Dragging Movements** | AA | Funcionalidade baseada em arraste oferece meio alternativo com ponteiro único sem arraste, salvo exceções aplicáveis. | Não confundir automaticamente com requisito de teclado (que é outro SC). |
| **2.5.8 Target Size (Minimum)** | AA | Alvo de ponteiro `≥24×24 CSS px` **OU** pelo menos uma exceção: espaçamento calculado, equivalente, inline, controle do user agent, essencial. | **Não** marque falha apenas porque botão tem 20px; verifique a regra de espaçamento e demais exceções. |
| **2.5.5 Target Size (Enhanced)** | AAA | Alvo `≥44×44 CSS px` salvo exceções correspondentes. | **Não** equiparar este limiar ao SC 2.5.8 AA. |
| **3.3.1 Error Identification** | A | Erro detectado identificado e descrito ao usuário em texto. | Verificar mensagem na ação, não apenas presença do elemento no DOM. |
| **3.3.2 Labels or Instructions** | A | Formulários disponibilizam labels/instruções quando requeridos. | Placeholder sozinho não é substituto seguro de rótulo permanente. |
| **3.3.3 Error Suggestion** | AA | Sugestão de correção quando conhecida e compatível com segurança/finalidade. | Existem limites por segurança e contexto. |
| **3.3.4 Error Prevention (Legal, Financial, Data)** | AA | Nas transações elegíveis, reversão, verificação ou confirmação conforme regra. | Não se aplica indiscriminadamente a toda ação. |
| **3.3.7 Redundant Entry** | A | Evitar reintrodução redundante de informação já fornecida no mesmo processo, salvo exceções. | Estado, segurança e essencialidade podem alterar análise. |
| **3.3.8 Accessible Authentication (Minimum)** | AA | Autenticação sem testes cognitivos desnecessários, com alternativas ou mecanismos assistivos conforme norma. | Não presumir que todo CAPTCHA/2FA falha; verificar condições específicas. |
| **4.1.2 Name, Role, Value** | A | Componentes customizados expõem nome, papel, estado e valores programáticos. | HTML semântico costuma ser mais robusto que div clicável + ARIA incompleto. |
| **4.1.3 Status Messages** | AA | Mensagens de status relevantes expostas a tecnologias assistivas sem roubar foco. | `role=status`, live region etc. exigem comportamento correto, não só atributo presente. |

## Procedimento básico sem resultados falsos

1. Defina **nível WCAG-alvo** (A, AA ou AAA) e páginas/funcionalidades cobertas; se não houver alvo, use WCAG 2.2 AA como referência de triagem e **não declare certificação**.
2. Inspecione DOM/semântica e navegação por teclado. Registre ordem de foco, operação de botões, menus, overlays, tabs, drag/canvas e conteúdo dinâmico.
3. Compare contrastes com ferramenta apropriada de cores renderizadas; teste estados padrão, hover, selected, disabled quando aplicável. Um print sem cores verificáveis gera **suspeita**, não número de contraste.
4. Verifique dimensões clicáveis **efetivas**, não só aparência. Para 2.5.8 examine exceções, sobretudo espaçamento entre alvos pequenos.
5. Teste zoom/reflow respeitando distorções introduzidas por controles dimensionais, canvas e tabelas. Avalie conteúdo fora das exceções em separado.
6. Use ferramentas automatizadas como diagnóstico parcial e acrescente inspeção manual. Ferramenta com “0 issues” **não** significa WCAG conforme.
7. Documente **SC exato, nível, teste, evidência, exceções consideradas, estado e resultado**. Não cite “contraste WCAG” sem qual proporção/critério se aplica.

## Padrões frequentes em SaaS / boards / canvases

- **Modal:** nome acessível, foco inicial útil, Tab dentro do modal ativo, Escape/fechamento apropriado e retorno de foco. O comportamento específico deve seguir o APG do componente.
- **Card selecionável com ações internas:** separar semanticamente selecionar, abrir, editar, expandir e menu; teclado e nome acessível precisam mapear cada operação.
- **Canvas bidimensional:** espaço cartográfico é exceção possível à regra de reflow, mas toolbar, painel de detalhes e instruções precisam continuar acessíveis, podendo exigir visão alternativa listada.
- **Kanban arrastável:** incluir ação de mover via menu/botões para operações baseadas em arraste e suporte por teclado conforme requisitos distintos.
- **Sidebar colapsada:** garantir que navegação ainda seja descobrível e tenha nome acessível, sem depender somente de `title` ou hover.
- **Conflitos e erros:** exibir texto local que explique recuperação e anunciar mudanças importantes a quem usa leitor de tela.

## Modelo curto de registro WCAG

```text
ID: A11Y-007
SC: WCAG 2.2 — 2.4.7 Focus Visible (AA)
Escopo: modal "Editar item", campo "Prazo" após Tab
Método: inspeção interativa; teclado; Firefox; viewport 1280x720
Evidência: vídeo 00:12–00:21 + CSS `.field:focus { outline: none }` (linha 91)
Observado: nenhum indicador discernível do foco naquele estado
Exceções analisadas: não aplicáveis à falha constatada
Resultado: falha verificada no contexto testado
Remediação: adicionar focus-visible discernível, testar tema/estados
Aceite: foco percebido em todos os controles do formulário, sem perder ordem
```

**Importante:** exemplo fictício; não atribuir essa falha a um produto real sem executar o teste.
