# Lei de Hick-Hyman — escolhas e tempo de reação

**Categoria:** modelo psicofísico de tempo de reação para seleção entre alternativas estímulo–resposta. **Confiança no fenômeno original:** alta nos paradigmas apropriados; **confiança na prescrição “menos opções = melhor UX”: baixa**, sem exame da tarefa.

## Fundamento e evidência

W. E. Hick (1952) e Ray Hyman (1953) estudaram relação entre informação/incerteza das alternativas e reação de escolha. A simplificação didática `RT = a + b log2(n + 1)` vale, em condições específicas, quando `n` representa alternativas equiprováveis; a distribuição de probabilidades, estímulos e o aprendizado alteram o cenário. **Não** use essa fórmula para prever duração de cadastros, tomada de decisão complexa, navegação ou visual search.

Uma análise revisada por pares no CHI 2020 (Liu et al., DOI 10.1145/3313831.3376878) argumenta que a lei costuma ser aplicada incorretamente em HCI e **não** implica o slogan “menos é melhor”. O tempo de escolha em interfaces envolve descoberta visual, familiaridade, significado das opções, restrições, custo de erro e exigências de decisão. Reduzir itens pode elevar navegação, revelar opções progressivamente ou ocultar funções úteis, piorando o fluxo.

## Perguntas de auditoria

- A pessoa está **identificando** opções, **buscando** um item, **comparando** propostas, ou fazendo escolha treinada de correspondência estímulo–resposta?
- Os agrupamentos seguem modelo mental do usuário e são mutuamente compreensíveis? As categorias possuem rótulos específicos?
- A opção crítica está visível quando importa? Mais um submenu introduz custo de descoberta e cliques sem benefício?
- A complexidade é intrínseca ao domínio? Esconder configurações necessárias compromete decisões ou controle?
- Usuários iniciantes e especialistas têm necessidades diferentes? Pesquisa, favoritos, atalhos ou recent items fazem diferença?

## Evidência exigida

Registre menu/step/rota, itens visíveis, conteúdo dos rótulos, hierarquia, número de passos até atingir o objetivo, estado inicial, critérios de acesso às opções ocultas. Não marque “violação” apenas porque há 8, 10 ou 20 opções. Se disponível, teste de árvore (`tree testing`) ou tarefas cronometradas e erro de caminho.

## Recomendações condicionais

1. Prefira **categorizar por tarefa** com rótulos claros a remover itens arbitrariamente.
2. Priorize acesso direto às ações frequentes, mantendo descoberta de funções avançadas.
3. Use busca, filtros e agrupamento escaneável para conjuntos grandes; revele opções secundárias apenas quando contexto justificar.
4. Evite decisões fragmentadas artificialmente em “uma pergunta por tela” quando é mais eficiente comparar lado a lado.
5. Em fluxos de alto risco, apresentar alternativas relevantes claramente pode importar mais que reduzir quantidade.

## Equívocos proibidos

- “Um menu com mais de sete itens viola Hick.” Não existe esse corte.
- “Remova metade dos botões para reduzir tempo pela metade.” O crescimento logarítmico nem se traduz diretamente ao cenário.
- “Mais cliques significa menos carga cognitiva.” Não necessariamente; total de esforço deve ser medido ao longo do fluxo.
- “Escolhas complexas sempre melhoram com divisão em etapas.” Há custo de contexto e comparação.

## Critérios de aceite exemplares

- Usuários descobrem a rota correta sem pistas externas, com nomenclatura compreensível; registrar sucesso/erro em teste representativo.
- Nenhuma função crítica fica inacessível ou perde visibilidade quando o contexto exige execução.
- A nova IA reduz busca improdutiva sem aumentar indevidamente os passos ou perder funções.

**Fontes:** Hick (1952) DOI 10.1080/17470215208416600; Hyman (1953) DOI 10.1037/h0056940; Liu et al. (2020) CHI DOI 10.1145/3313831.3376878.
