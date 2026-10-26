# Auditoria UX/UI — {{PRODUCT}}

**Escopo:** {{SCOPE}}  
**Modo:** {{MODE}}  
**Data:** {{DATE}}  
**Responsável:** [informar se conhecido]

> Este relatório registra somente resultados sustentados pelos insumos acessíveis. “Não testado” não equivale a “sem problemas”. Estudos de UX são contexto teórico, não provas de falhas específicas nesta interface.

## 1. Resumo executivo

**Objetivo da auditoria:** [tarefa de negócio e resultado esperado]  
**Jornadas centrais:** [lista breve]  
**Situação geral:** [síntese objetiva sem score arbitrário]  
**Impedimentos observados:** [referências a IDs]  
**Melhorias de maior valor:** [IDs + por quê]  
**Cobertura/limitação principal:** [explicitar o que não foi possível verificar]

## 2. Insumos e condições de teste

| Parâmetro | Valor |
|---|---|
| URLs / arquivos / build | [informar] |
| Papéis / personas | [informar] |
| Browser, OS, dispositivo | [informar ou não testado] |
| Viewports CSS e zoom | [informar ou não testado] |
| Dados/estados | [informar] |
| Insumos de suporte | [prints, DOM, código, logs, telemetria se disponíveis] |
| Limitações de acesso | [informar explicitamente] |

## 3. Inventário de jornadas

| Jornada / tarefa | Início → fim | Perfil | Estados inspecionados | Resultado e evidência | Lacunas |
|---|---|---|---|---|---|
| [identificador] | [passos] | [role] | [estados] | [resultado] | [não testado] |

## 4. Matriz de cobertura

Status permitidos: `ok_verificado`, `problema_verificado`, `hipotese`, `nao_testado`, `nao_aplicavel`.

| Eixo | Status | Telas/fluxos testados | Evidência | Lacunas |
|---|---|---|---|---|
| A. Orientação/navegação | nao_testado | — | — | — |
| B. Hierarquia/densidade | nao_testado | — | — | — |
| C. Fluxos/modelos mentais | nao_testado | — | — | — |
| D. Interações/feedback | nao_testado | — | — | — |
| E. Formulários | nao_testado | — | — | — |
| F. Responsividade/viewport | nao_testado | — | — | — |
| G. Acessibilidade | nao_testado | — | — | — |
| H. Estados/recuperação | nao_testado | — | — | — |
| I. Board/canvas/edição rica | nao_testado | — | — | — |
| J. Conteúdo/idioma/confiança | nao_testado | — | — | — |

## 5. Priorização e decisão

**Método:** `I = 0.40*S + 0.35*C + 0.25*R`; `E=(T+D+V)/3`; `O=I/E`; notas de 1 a 5, sem significado estatístico. P0 prevalece para bloqueios relevantes verificados. Confiança é avaliada em separado.

| ID | Finding | Evidência | I | E | O | Confiança | Faixa | Motivo da prioridade |
|---|---|---|---:|---:|---:|---|---|---|
| UX-001 | [descrição] | [link] | [valor] | [valor] | [valor] | [alto/medio/baixo] | [P0–P3] | [motivo] |

### Quick wins

[Achados de alto impacto e menor esforço, explicitamente comparados com correções fundamentais necessárias]

### Investimentos estruturantes

[Alterações de maior impacto e dependências, faseadas]

## 6. Achados detalhados

Para **cada** finding use `assets/templates/finding.md`. Não agrupe problemas distintos em uma única descrição genérica. Se duas ocorrências compartilham causa-raiz, referencie finding-mãe com evidências adicionais.

### UX-001 — [nome preciso]

- Classificação: observado / verificado / hipótese.
- Evidência: [caminho, repro, estado, viewport, observação].
- Objetivo afetado e custo: [descrição contextual].
- Fundamento: [lei(s) e por que se aplicam; limitação].
- Proposta: [componente, copy, estado, visual, navegação, acessibilidade].
- Impacto/esforço/confiança: [notas + justificativa].
- Aceite: [testes verificáveis antes/depois].
- Limite: [o que não foi medido].

## 7. Plano de implementação

| Fase | Finding(s) | Mudança precisa | Dependências | Evidência de aceite | Owner (se conhecido) |
|---|---|---|---|---|---|
| 0. Bloqueios | [IDs] | [ação] | [dependências] | [teste] | [se conhecido] |
| 1. Fluxos essenciais | [IDs] | [ação] | [dependências] | [teste] | [se conhecido] |
| 2. Ergonomia e polimento | [IDs] | [ação] | [dependências] | [teste] | [se conhecido] |

## 8. Plano de validação

[Baseline real se existir; hipóteses; tarefa; métrica; método; participantes/roles; critérios de aceite; guardrails; regressões].

## 9. Não testados e perguntas abertas

- [função / estado / motivo / teste recomendado]
- [documentar explicitamente quando não foi possível executar teclado, reflow, WCAG, dispositivo real ou coleta de métricas]

## 10. Conclusão decisória

[O que corrigir primeiro, o que experimentar, o que não alterar por ausência de evidência, e o que caracteriza conclusão.]

## 11. Fontes

Referenciar estudos/SC **efetivamente utilizados** com DOI/URL em `references/sources.md`; não anexar leis irrelevantes apenas para cumprir lista.
