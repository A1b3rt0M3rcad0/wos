# Lei de Miller — memória de curto prazo e chunking

**Categoria:** achados de capacidade limitada de memória/processamento de informação. **Evidência:** forte para limitações de memória sob paradigmas definidos; **regra de 7 elementos visíveis na UI:** não respaldada.

## Fundamento e evidência

George A. Miller (1956) descreveu limites de julgamento absoluto e extensão de memória imediata, com o número “7±2” como síntese evocativa de resultados distintos. O próprio trabalho aborda recodificação e agrupamento (`chunking`). Uma revisão influente de Nelson Cowan (2001) distingue componentes de memória e discute um limite de cerca de **3–5 chunks**, frequentemente descrito como aproximadamente quatro sob condições controladas.

Esses resultados **não** especificam quantos itens de menu, cartões, ações, campos ou abas devem aparecer em uma interface. A pessoa pode **reconhecer** opções presentes sem precisar mantê-las na memória de curto prazo. O problema de UX geralmente é memória de trabalho exigida por fluxos fragmentados, não o simples número de opções mostradas.

## Perguntas de auditoria

- É necessário **lembrar informação de tela anterior** para completar a atual? O sistema permite consultar os dados originais?
- Critérios, evidências, proprietário e próximos passos ficam próximos do item a que se referem ou estão espalhados por seções desconectadas?
- O usuário precisa memorizar códigos, filtros, status, campos ou nomes para trocar entre telas?
- Há agrupamento semântico real, rótulos, resumo de contexto, ordenação e revelação progressiva útil?
- Indicadores, instruções e mensagens de erro permanecem disponíveis durante a correção?

## Evidência exigida

Reproduza a navegação e mostre informação que desaparece quando necessária. Conte trocas de contexto, passos de ida/volta, cópias manuais ou consultas à memória. Quando possível, observe erros de lembrança com pessoas reais; não quantifique “carga cognitiva” apenas pela estética de densidade.

## Remediações condicionais

1. Projete para **reconhecimento, não recordação**: rótulos claros, valores persistidos, breadcrumbs, histórico, resumo e prévias.
2. Disponibilize contexto de uma tarefa no momento da ação (ex.: critérios, evidências e responsável visíveis no painel lateral ao editar).
3. Agrupe itens por significado e sequência de trabalho, não por cotas numéricas arbitrárias.
4. Em cadastros longos, persistência e revisão de dados existentes podem ser melhores que dividir em cinco telas.
5. Em interfaces densas para especialistas, prefira filtros, hierarquia informativa e shortcuts a eliminar informação crítica.

## Equívocos proibidos

- “Mais de sete itens viola Miller” (falso).
- “Quatro é o novo limite exato para qualquer interface” (também falso).
- “Toda densidade visual equivale a sobrecarga da memória de trabalho” (inferência não demonstrada).

## Critérios de aceite

- Fluxos centrais podem ser completados sem memorizar informações que já foram fornecidas ao sistema.
- Ao abrir um item, os dados necessários à decisão estão legíveis no mesmo contexto ou a um acesso previsível.
- Alterações preservam eficiência de especialistas e acessibilidade cognitiva de iniciantes.

**Fontes:** Miller (1956), Psychological Review DOI 10.1037/h0043158; Cowan (2001), Behavioral and Brain Sciences DOI 10.1017/S0140525X01003922.
