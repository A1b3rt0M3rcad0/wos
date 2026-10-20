# Priorização por impacto, esforço e qualidade de evidência

Este é um **modelo operacional de triagem**, não um algoritmo cientificamente calibrado de UX. Defina valores só após observar o cenário; se não houver dados de alcance, use faixa estimada e marque confiança baixa. Números não substituem julgamento em bloqueios graves nem requisitos normativos.

## 1. Impacto composto (I)

Pontue de 1 a 5, registrando justificativa:

| Nota | Severidade (`S`) — fricção concreta | Criticidade (`C`) da jornada | Alcance (`R`) |
|---|---|---|---|
| 1 | Cosmético/irrelevante para conclusão | Função periférica | Casos excepcionais |
| 2 | Ligeiro esforço extra, sem erro importante | Baixa frequência/valor | Pequena audiência específica |
| 3 | Retrabalho/erro recuperável em tarefa | Relevante, mas não central | Segmento material de usuários |
| 4 | Falhas recorrentes/alta fricção | Operação frequente ou de alto valor | Atinge muitos cenários relevantes |
| 5 | Impede tarefa crítica, perda de trabalho ou exclusão relevante | Jornada essencial do produto | Presente na maioria dos cenários/público |

Fórmula de ordenação: **`I = 0.40*S + 0.35*C + 0.25*R`** (entre 1 e 5). Pesos são escolhas de governança deste pacote, **não resultados de estudo empírico**. `R` exige dados de analytics ou estimativa explicitamente rotulada.

## 2. Esforço composto (E)

Pontue de 1 a 5:

| Nota | Complexidade técnica (`T`) | Dependências (`D`) | Custo de verificação (`V`) |
|---|---|---|---|
| 1 | Copy/estilo local, sem alterar comportamento | Independente | Teste local simples |
| 2 | Mudança de componente pequena | Uma área/owner | Testes rápidos por viewport/estado |
| 3 | Integração de componentes/fluxo | Várias telas ou squads | Regressão funcional e pesquisa curta |
| 4 | Refatoração estruturante | Múltiplos sistemas e contrato | Homologação em jornadas/roles extensos |
| 5 | Redesenho sistêmico/arquitetura | Migração e coordenação ampla | Experimentos, migração e aceites complexos |

Fórmula de ordenação: **`E = (T + D + V)/3`**. Não atribua estimativas em horas sem conhecer repositório, equipe e dependências. Esforço alto não diminui a gravidade, mas altera planejamento de entrega.

## 3. Índice de oportunidade e faixas

`O = I/E`. Interprete como **ordenação aproximada** dentro de grupos. Não comparar valores com precisão matemática falsa: uma diferença de 1,34 vs 1,32 é empate operacional. Registre notas/fundamentação.

**Quadro impacto×esforço** (limiares orientativos, não absolutos):

| | E ≤ 2: baixo esforço | E > 2: esforço médio/alto |
|---|---|---|
| I ≥ 4: alto impacto | **Quick win de alto impacto** | **Investimento prioritário**; quebrar em incrementos |
| I < 4: menor impacto | Ajuste oportunista, depois dos principais | Adiar ou avaliar quando surgir evidência |

## 4. Atribuição de prioridade (P0–P3)

- **P0 / bloqueador:** impossibilidade **verificada** de completar jornada crítica em cenário relevante, perda de trabalho grave, ou exclusão de acesso severa verificada. Exige justificativa específica e plano de resolução imediato. **Ignora o índice** `O` como fator decisório; pode ser caro.
- **P1 / alta:** impacto `I ≥ 4` com evidência ao menos observável e **confiança não baixa**; além de problemas normativos relevantes que não chegam a P0. Mesmo com esforço alto, priorize o plano, não a falsa sensação de facilidade.
- **P2 / média:** impacto `I ≥ 3` ou casos de impacto alto cuja hipótese causal ainda não foi verificada (investigar antes de alterar estruturalmente).
- **P3 / baixa:** restante; estética pura ou otimização de valor incerto sem bloqueio.

Essas faixas podem ser adaptadas ao produto, mantendo registro da decisão. **Não** classifique todas as suspeitas WCAG como falhas nem toda falha WCAG como P0; considere nível, requisito, impacto e escopo.

## 5. Confiança, independente da pontuação

- `alto`: cenário reproduzido com artefato rastreável + mecanismo direto/impacto constatado ou usuário observado.
- `medio`: desvio observado de forma inequívoca, mas magnitude/causa parcial ou prevalência desconhecida.
- `baixo`: inferência de screenshot, ausência de execução, desconhecimento de público/alcance, ou mecanismo de lei com transferência fraca.

Se confiança é baixa, anexar **teste de validação de hipótese** como próxima ação. Score alto com confiança baixa **não** autoriza refatoração cara sem investigação.

## 6. Exemplo aritmético (fictício)

- Observação: após abrir detalhe em 1280×720, footer sticky encobre ação “Salvar”; fluxo central exige scroll externo e perde contexto.
- `S=4`, `C=5`, `R=3` (alcance estimado, não medido). `I = 0.40*4 + 0.35*5 + 0.25*3 = 4.10`.
- `T=2`, `D=2`, `V=2` → `E=2.00`; `O=2.05`. Recomendar `P1`, **se o problema foi realmente observado** e confiança pelo menos média.
- Ação: limitar chrome vertical e manter botão visível dentro de painel/área com overflow correto; medir se fluxo completou sem scroll externo e não perde função com zoom.

## 7. Ordenação final com dependências

Primeiro P0 verificados. Depois P1 por dependências e oportunidade `O` aproximada. Depois P2 com testes e intervenções baratas; P3 por manutenção oportunista. Verifique se corrigir a causa-raiz resolve múltiplos achados e remova contagem dupla. Inclua plano por fases com owner **só se conhecido**, esforço relativo e critérios de aceite.

## 8. Regras de qualidade do índice

- Não chame escala ordinal 1–5 de “impacto esperado em porcentagem”.
- Não multiplique automaticamente por número de leis citadas; uma causa pode ter várias leis.
- Não presuma frequência sem dados; `R=3` pode ser suposição e deve ser assim marcado.
- Não estime ROI, aumento de conversão ou horas sem baseline e custo implementável conhecido.
- Se a ação é mandatória por critério normativo confirmado, não a descarte por índice baixo; identifique a obrigação e implemente no planejamento adequado.
