# Protocolo de validação das melhorias (sem fabricar causalidade)

O diagnóstico de uma falha e a previsão de melhoria são proposições diferentes. Uma alteração pode corrigir um problema óbvio e introduzir regressões de descoberta, performance ou acessibilidade. Planeje critérios de aceite antes de alterar a interface.

## 1. Estabeleça a hipótese operacional

Exemplo: “Ao trazer critérios de aceite e evidências para o detalhe da tarefa, reduziremos alternância entre páginas durante a avaliação.” A hipótese é plausível (reconhecimento em vez de recordação), **mas não se sabe** a magnitude, nem se aumenta complexidade visual para usuários iniciantes.

Defina:

- **Objetivo e tarefa:** início, resultado observável e condição de sucesso.
- **População/role:** iniciante, frequente, especialista, pessoas que usam teclado e/ou leitor de tela quando aplicável.
- **Comparação:** baseline anterior, tarefa equivalente, ambiente consistente.
- **Medidas primárias:** sucesso de tarefa, erros, necessidade de ajuda, trabalho perdido, incidência de bloqueio, conformidade do critério específico.
- **Medidas secundárias:** duração, ações, alterações de contexto, satisfação, discoverability, erros de toque/teclado.
- **Guardrails:** regressão funcional, performance, acessibilidade, dados corretos, visibilidade de informação crítica.

## 2. Modalidades de verificação e grau da inferência

| Modalidade | Bom para | Não comprova sozinha |
|---|---|---|
| Checklist/revisão expert | Detectar problemas potenciais | Prevalência entre usuários |
| Browser com cenários determinísticos | Comportamento/regressão de fluxo | Melhoria de conversão populacional |
| Validação WCAG SC específico | Falha ou atendimento contextual daquele SC | Conformidade global WCAG |
| Teste de usabilidade moderado | Descoberta, compreensão, causas de erro | Tamanho de efeito geral sem amostragem adequada |
| Estudo quantitativo / analytics | Métricas observadas no uso real | Causalidade sem controle de confundidores |
| A/B com randomização adequada | Comparação causal sob hipótese e pressupostos | Recomendação universal ou extrapolação a outros públicos |

Não reportar porcentagens, ganhos em segundos ou ROI sem amostra, período, baseline, métrica, segmentação e dados auditáveis.

## 3. Modelo de experimento/pré-pós

```text
Finding: UX-XXX
Hipótese: [uma mudança deve alterar qual comportamento?]
Cenário: [role, dispositivo, dados e objetivo]
Baseline: [data; valor medido ou "não disponível"]
Tratamento: [mudança isolável, se possível]
Métrica primária e unidade: [p.ex. % de tarefas concluídas corretamente]
Métricas de segurança: [ex.: sem regressão de keyboard/zoom e sem perda de dados]
Método: [reprodução, pesquisa moderada, telemetria, A/B]
Critério de aceite: [condição testável, vinculada ao requisito]
Resultado observado: [em branco até teste]
Limitações: [amostra pequena, aprendizado, confusão, missing data]
Decisão: [manter, iterar, reverter, inconclusivo]
```

## 4. Medidas específicas conforme problema

**Fitts:** tempo e erros de ativação por dispositivo, posição e tamanho; compare comandos equivalentes. Instrumentação de cliques não mede automaticamente intenção.

**Hick/IA:** sucesso em encontrar ação; primeiro caminho escolhido; erros de navegação; tempo de descoberta. A complexidade da decisão real pode ser mais importante que contagem de alternativas.

**Jakob:** previsibilidade de função de rótulos/ícones; uso sem instrução; taxa de erro ao executar ação de padrão conhecido vs. alternativo.

**Miller:** saltos de contexto, recordação indevida de valores, necessidade de anotações, acerto de critérios com informações no mesmo contexto.

**Zeigarnik/continuidade:** porcentagem de rascunhos recuperados com sucesso e tempo até retomar ação. Não dizer que isso mede diretamente Zeigarnik.

**Gradiente de meta:** correção e compreensão do progresso, completion rate apenas com baseline e objetivo verdadeiro, sem usar gamificação manipulativa.

**Von Restorff:** detecção e compreensão de alerta ou ação; recall se realmente objeto de teste. Clique e conversão são métricas adicionais distintas.

**WCAG:** presença do SC, método e exceções; erros confirmados e rechecagem no mesmo estado, com ferramentas automáticas e operação humana.

## 5. Critérios para relatório pós-mudança

- O finding tinha evidência antes? A correção aborda a mesma causa?
- Os critérios de aceite foram percorridos nos viewports/roles necessários?
- O desenho antigo e o novo são comparados com mesmo cenário?
- Existem regressões funcionais, visuais, de navegação ou acessibilidade?
- Dados contraditórios estão explicitados, não excluídos da conclusão?
- Sem condições para mensurar, relatório diz **“hipótese ainda não validada”**.

## 6. Resultado mínimo aceitável sem ferramentas

Se só houver screenshot ou prompt descritivo, entregue propostas e testes que precisam acontecer; não declare “foi resolvido”, “conversão aumentará 25%” ou “100% acessível”. O plano de teste é parte do produto da skill, não substituto de execução.
