# Protocolo de auditoria: rastreabilidade, execução e calibração

O objetivo não é produzir uma coleção de opiniões estéticas. Cada finding deve resistir à pergunta “como outra pessoa pode verificar isso?” e incluir uma intervenção que preserve o trabalho do usuário.

## 0. Preparação e restrições

Registre: nome e versão/build da interface (se disponíveis), ambiente, URLs/rotas, papéis/permissões, público, idioma, função crítica, dispositivos, viewport CSS, browser/OS, data, ferramentas permitidas, limitações de acesso, objetivos de negócio e restrições de design. Capture o estado de dados necessário à reprodução; anonimize informações pessoais e credenciais. Não force ações irreversíveis no ambiente produtivo para obter evidências.

Peça informação adicional somente quando necessária para executar um teste impossível sem ela. Em auditorias parciais, marque explicitamente lacunas e prossiga com o que é verificável.

## 1. Inventário de superfícies, jornadas e estados

| Entidade de auditoria | Exemplo | O que registrar |
|---|---|---|
| Página / tela | `/work`, modal de editar, detalhe | URL/estado, função, papel, viewport |
| Ponto de decisão | seleção da ação, confirmação de exclusão | alternativas disponíveis e consequências |
| Ação | criar, filtrar, selecionar, arrastar, salvar | trigger, indicador, validação, reversibilidade |
| Estado | vazio, carregando, sucesso, erro, conflito, sem permissão | mensagem, recuperação, próxima ação |
| Jornada | iniciar → editar → revisar → concluir | passos, saltos de contexto, bloqueios e retornos |
| Responsividade | 1280/1440/1920/2560 CSS px e 320/390 mobile | conteúdo visível, densidade, sobreposições, scroll |
| Acessibilidade | teclado, zoom, leitor de tela (se realmente testado) | método, dispositivo, SC WCAG, evidência |

Para cada jornada, examine: primeira visita, usuário experiente, conjunto de dados vazio, moderadamente denso e extremo, erros recuperáveis, conteúdo longo, acesso sem permissão, mudanças concorrentes e interrupção/reabertura.

## 2. Pirâmide de evidência local (do mais forte ao mais limitado)

1. **Uso real instrumentado / testes com participantes representativos:** conclusão de tarefa, erros, duração, relatos e contexto experimental. Ainda requer verificar amostra, qualidade do instrumento e causalidade.
2. **Execução reproduzida da interface:** passo a passo, viewport, estado, console/DOM quando disponíveis, captura/video. Demonstra comportamento observado no cenário, mas não prevalência populacional.
3. **Inspeção de código/DOM/testes:** propriedades programadas e casos cobertos; sem renderização real, é uma previsão incompleta de comportamento.
4. **Captura estática:** aparência e conteúdo visível apenas. Não demonstra hover, focus, touch target efetivo, cliques, persistência ou adaptação em outras resoluções.
5. **Hipótese do avaliador:** tentativa explicativa sem observação suficiente, claramente marcada como pendente de investigação.

Fontes bibliográficas sustentam **mecanismos e critérios**, não substituem evidência de que *esta* interface falha.

## 3. Matriz de capacidade por modo

| Pergunta | Visual | Código | Browser interativo |
|---|---|---|---|
| CTA perceptível? | Observação contextual | Parcial | Observação/teste |
| Hit area real? | Não | Possível, por estilo/evento | Sim, via inspeção |
| Ordem de foco e tabulação? | Não | Hipótese por DOM | Teste de teclado |
| Erro exibido após submit? | Não, exceto print específico | Fluxo implementado | Teste no estado real |
| Mobile preserva fluxos? | Apenas com prints mobile | Revisão de regras CSS | Teste por viewport |
| Leitor de tela utilizável? | Não | Indícios semânticos | Apenas se leitor/testador acessível |
| Tempo de conclusão melhorou? | Não | Não | Não sem baseline/teste apropriado |
| Cumprimento integral WCAG? | Não | Não | Auditoria completa dedicada ainda necessária |

Se nenhum acesso interativo estiver disponível, todas as hipóteses de interação devem constar como **“verificar no browser”** e nunca serem rotuladas confirmadas.

## 4. Ficha de observação reproduzível

Preencha, para cada achado:

- **Identidade:** `UX-001`, tela, elemento, role, estado e cenário de usuário.
- **Objetivo:** ação que uma pessoa tenta concluir e por que é importante.
- **Pré-condições:** dados, estado de login, largura/altura do viewport CSS, idioma/zoom, role.
- **Passos:** sequência mínima e determinística para reproduzir, inclusive o resultado.
- **Observado:** evento/desvio concreto (ex.: “a seção de detalhes ocupa a parte inferior da área útil e o botão Concluir não aparece sem rolar a janela de fora”).
- **Esperado:** critério funcional verificável, não “experiência bonita”.
- **Evidência:** caminho/link para print, vídeo, DOM node, trecho de código com linha, log ou resultado de teste. Dado pessoal anonimizável deve ser anonimizado.
- **Nexo causal:** problema → mecanismo de uso → lei(s) aplicáveis e restrições da analogia.
- **Remediação:** alteração de componente/copy/layout/estados e efeito funcional esperado.
- **Aceite:** casos observáveis + testes de regressão no mesmo cenário.
- **Confiança:** `alto` (reproduzido e causalidade direta), `medio` (observação forte, impacto ainda incerto), `baixo` (hipótese visual/informação insuficiente).

## 5. Checklist de execução para browser/repositório

1. Fixe viewport **CSS**, zoom, device pixel ratio (se relevante) e ambiente; evite confundir pixels físicos com pixels CSS.
2. Percorra tarefas sem ler código primeiro quando for possível, para não substituir discovery do usuário por conhecimento da implementação.
3. Repita com teclado: Tab/Shift+Tab, Enter/Space, setas segundo componente, Esc, foco após diálogos e mudanças de rota.
4. Percorra estados feliz, inválido, erro de rede, acesso negado, mudanças concorrentes, salvar e desfazer.
5. Teste overflow, sticky header e painéis sobrepostos em viewport verticalmente restrito; telas HD baixas (ex.: 1280×720) podem falhar mesmo com largura suficiente.
6. Inspecione contêineres flex/grid e min-width/min-height, overflow, breakpoints; detecte scroll duplo e conteúdo recortado.
7. Registre problemas únicos; se várias telas compartilham causa-raiz, abra finding-mãe com ocorrências. Evite contar o mesmo CSS bug sete vezes.
8. Não use métricas do browser como “tempo de decisão humana”; DOM render metrics e Core Web Vitals são outra dimensão.

## 6. Checagem de coerência antes de finalizar

- Se o título contém “violação”, a evidência mostra descumprimento de um requisito específico? Caso contrário, use “possível” ou “fricção observada”.
- O princípio selecionado explica a fricção observada ou foi adicionado porque o pedido exige sete leis?
- A solução muda realmente a tarefa ou apenas aparência?
- O critério de aceite permite comparar antes/depois sem interpretação subjetiva excessiva?
- A alternativa preserva leitura por tecnologias assistivas e ações por teclado?
- A prioridade segue impacto **e** esforço, não quantidade de leis citadas?
- O relatório explicita telas e estados não testados?
- O relatório separa conclusões confirmadas de proposições de pesquisa/experimento?

## Classificação dos achados

- `observed`: problema visível/reproduzido, mas impacto ainda não quantificado.
- `verified`: problema e seus efeitos diretamente verificados no cenário, sem extrapolação de prevalência.
- `hypothesis`: possível problema a validar com teste.
- `not-tested`: dimensão sem acesso ou sem condição para avaliar. **Não** criar finding conclusivo a partir dela.
- `not-applicable`: dimensão excluída por escopo/arquitetura, com justificativa.

A classificação descreve o grau da **evidência do problema**, não o grau de evidência científica da lei.
