# Efeito Zeigarnik — interrupção, lembrança e retomada

**Categoria:** hipótese de memória de tarefas interrompidas; distinguir do efeito Ovsiankina (tendência de retomar atividade). **Força empírica atual:** efeito de memória geral **não estabelecido**; tendência de retomada tem suporte metanalítico.

## Fundamento e estado da evidência

Trabalhos clássicos associados a Bluma Zeigarnik (1927) sugeriram melhor memória de tarefas interrompidas em relação às concluídas. Contudo, a metanálise de Ghibellini & Meier (2025), publicada em *Humanities and Social Sciences Communications*, concluiu que **não havia vantagem geral de memória para tarefas inacabadas** nas evidências sintetizadas; observou-se uma tendência geral de retomada (Ovsiankina). Os resultados são influenciados por contexto, demandas situacionais, engajamento e diferenças individuais.

Há uma distinção relevante: Wendsche, Weigelt & Syrek (2026) sintetizaram outra literatura, sobre **tarefas profissionais pendentes e pensamentos relacionados ao trabalho fora do expediente**. Encontraram associações positivas (entre pessoas: `ρ=.382`; dentro das mesmas pessoas ao longo do tempo: `ρ=.247`), especialmente com ruminação afetiva. Isso **não contradiz** diretamente a conclusão de 2025 sobre recordação em tarefas experimentais: são variáveis, contextos e perguntas diferentes. Também não demonstra que indicadores de pendência numa interface causem mais produtividade ou recordação.

Por isso, a skill **não** deve alegar que deixar um fluxo pendente fará o usuário voltar nem usar tensão psicológica artificial para forçar retorno. Interfaces de trabalho persistente devem oferecer continuidade e contexto **porque isso facilita execução e reduz perda de trabalho**, independentemente de um mecanismo mnemônico universal.

## O que auditar

- Tarefas interrompidas podem ser retomadas com estado salvo, sem retrabalho?
- A pessoa identifica com precisão `em andamento`, `bloqueado`, `pendente de aprovação` e `concluído`?
- Há distinção explícita entre status do trabalho, definição de pronto, último evento e **próxima ação**?
- É possível localizar rapidamente atividades iniciadas e não finalizadas, inclusive após logout, atualização e troca de dispositivo?
- A UI mostra o que ainda falta sem gatilhos manipulativos de incompletude?

## Evidência exigida

Verificar fluxo real (abrir → alterar → sair/interromper → retornar), persistência, aviso de mudanças não salvas, retomar foco, trilha de atividade, critério e evidência associada. Não tomar um indicador “60% completo” como progresso verdadeiro sem saber o denominador e a definição.

## Remediações práticas

1. Guardar rascunhos e exibir estado salvo/não salvo, quando compatível com o domínio.
2. Apresentar **“retomar de onde parei”**, última mudança e próxima ação concreta.
3. Mostrar pendências fundamentadas em critérios/regras do produto, sem exigir que a pessoa memorize o contexto.
4. Para recuperação de falhas, preservar o texto digitado e explicar conflito de versão com opções seguras de reconciliação.
5. Avisos de progresso são ferramentas de informação e orientação; não prometer efeito psicológico automático.

## Anti-padrões

- Criar interrupções propositais para produzir lembrança.
- Falsificar contadores de trabalho incompleto para aumentar retorno.
- Associar todo abandono à ausência do efeito Zeigarnik.
- Usar o “efeito Zeigarnik” como justificativa exclusiva de push notifications recorrentes.

## Critérios de aceite

- Ao retornar à tarefa, a pessoa recupera estado e contexto necessários (documente tempo e sucesso com baseline se houver medição).
- Dados não salvos recebem sinalização e recuperação adequadas.
- Progresso/pending são definidos por estados reais do negócio.

**Fontes:** Zeigarnik (1927), DOI 10.1007/BF02409755; Ghibellini & Meier (2025), DOI 10.1057/s41599-025-05000-w (memória/retomada); Wendsche et al. (2026), DOI 10.1080/10615806.2026.2616302 (pensamentos sobre trabalho). Ver `sources.md`.
