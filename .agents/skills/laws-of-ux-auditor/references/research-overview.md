# Revisão crítica das sete leis — síntese para decisões de UX

**Data do levantamento:** 19/10/2026. Este documento distingue pesquisa básica, validade externa e tradução para design de produto. Não implica que a revisão seja uma revisão sistemática completa, nem que as sete leis esgotem usabilidade ou acessibilidade.

## Matriz de evidências

| Modelo | O que o trabalho sustenta | Tipo de suporte | Limites de uso no produto | Força da recomendação de UI |
|---|---|---|---|---|
| **Fitts** | Tempo de movimento se relaciona a amplitude e tolerância de alvo em tarefas motoras. | Experimentos originais (1954); análise de HCI (1992). | Prever ganho exige calibrar dispositivo/tarefa; não explica busca visual, decisão ou teclado. | Alta **quando** o gargalo é apontamento; outras etapas exigem dados próprios. |
| **Hick-Hyman** | Tempo de reação em escolhas estímulo–resposta correlaciona-se com informação/incerteza sob condições adequadas. | Estudos originais (1952–1953); revisão crítica CHI 2020. | Não significa “menus menores sempre melhores”; visual search e interpretação podem dominar. | Condicional; auditar rótulos, IA, descoberta e performance total. |
| **Jakob** | Convenções familiares tendem a reduzir aprendizado desnecessário e surpresa. | Heurística/observações especializadas de Nielsen. | Não é lei física ou ensaio experimental que proíba inovação; público e domínio são moderadores. | Boa hipótese de design, validar em jornadas reais. |
| **Miller** | Memória imediata e julgamento mostram limites que dependem de organização/recodificação. | Estudo original 1956 + revisão Cowan 2001. | `7±2` e ~4 chunks **não** são limites de quantidade de itens visíveis. | Forte para reduzir necessidade de recordar, não para contar botões. |
| **Zeigarnik** | Vantagem de memória para tarefas interrompidas **não foi generalizada** em metanálise recente; tendência à retomada existe. | Metanálise 2025; metanálise 2026 sobre ruminação (desfecho distinto). | Não prometer retenção por deixar coisas “pendentes”; suporte à retomada deve se justificar diretamente. | Forte para exigir cautela; design de continuidade é inferência prática, não prova do efeito de memória. |
| **Gradiente de meta** | Em determinados programas de incentivo, aproximação percebida de recompensa se associa a maior esforço. | Estudos de campo/laboratório e dados de fidelidade, 2006. | Sem evidência de efeito universal em workflows longos, ambíguos ou sem recompensa comparável. | Condicional para progresso **verdadeiro** e metas explícitas. |
| **Von Restorff** | Um item distinto pode ser mais lembrado; contexto e tipo de distinção importam. | Pesquisa experimental e revisão/experimentos 2017. | Memória do item não prova compreensão, acesso, clique ou conversão; destaque pode distrair. | Condicional para hierarquia contextual bem justificada. |

## Como converter ciência em auditoria sem extrapolação

1. **Identifique a tarefa real.** Exemplo: “selecionar e editar uma task” envolve localizar, reconhecer, decidir, apontar, editar e validar. Uma lei pode descrever só uma dessas etapas.
2. **Defina mecanismo plausível e falsificável.** “O alvo tem 16×16 CSS px com três vizinhos próximos, então a precisão do clique é uma hipótese plausível (Fitts), mas só observar erros/tempo demonstrará impacto real.”
3. **Verifique a cadeia causal.** Não atribuir queda de conversão a cor, número de opções ou barra de progresso sem testar fatores alternativos.
4. **Selecione a fonte certa:** WCAG para exigência normativa, estudos para explicação mecanística, dados de uso para contexto, heurísticas para busca de problemas, pesquisa com usuários para validação local.
5. **Evite conflito artificial entre leis.** Fitts pode favorecer alvo grande; Von Restorff, destaque; Jakob, convenções; Miller, contexto; Hick, descoberta. O trade-off resolve-se pelo objetivo da tarefa, restrições e testes, não pela “lei mais importante”.
6. **Registre o estado da evidência.** `observado`, `confirmado_por_execucao`, `confirmado_por_usuario`, `hipotese`, `nao_testado`. Se a lei é discutível no cenário, não a cite como prova.

## Três exemplos de inferência inválida

- “14 itens em navegação violam Hick + Miller; reduzir para 7.” **Inválido**: número de itens não basta. Testar busca visual, arquitetura da informação, comparação e custos de navegação; manter visíveis opções necessárias.
- “Usuários memorizarão tarefas inacabadas por Zeigarnik.” **Inválido**: síntese metanalítica de 2025 não sustenta efeito geral de memória; oferecer continuidade e lembretes contextuais é recomendação independente.
- “CTA laranja aumenta cliques por Von Restorff.” **Inválido**: saliência relativa pode mudar memorização, não estabelece taxa de clique; verificar hierarquia, contraste, usabilidade e métricas reais.

## Interpretação de níveis de evidência

- **Alta confiança no fenômeno original** não é sinônimo de alta confiança no efeito de uma mudança concreta de interface.
- **Norma testável** (WCAG) não é “hipótese psicológica”. Pode haver exceções e escopo de conformidade, que devem ser documentados.
- **Diagnóstico de especialista** é útil para triagem, mas não substitui medição de comportamento nem estudo com pessoas com deficiência.
- **Evidência da interface** exige execução/captura rastreável; somente bibliografia não demonstra falha em um site específico.

Detalhes e limitações: `fitts.md`, `hick.md`, `jakob.md`, `miller.md`, `zeigarnik.md`, `goal-gradient.md`, `von-restorff.md`. Referências e acesso em `sources.md`.
