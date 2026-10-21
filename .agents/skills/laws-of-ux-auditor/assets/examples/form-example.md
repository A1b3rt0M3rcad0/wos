# EXEMPLO FICTÍCIO — análise de formulário a partir de capturas

**Objetivo:** mostrar como a skill deve lidar com **evidência insuficiente** sem fabricar comportamento. Há dois prints simulados de um cadastro de organização, um com três grupos de campos e outro mostrando toast de erro. Nenhuma interface foi realmente fornecida ou executada para este exemplo.

## Achado UX-F01 — Validação parece depender de notificação transitória

- **Modo:** `visual`; **classificação:** `hypothesis`; **confiança:** `baixo`.
- **Observável na captura hipotética:** toast genérico “Não foi possível salvar”; não há erros de campo visíveis na captura.
- **O que NÃO se sabe:** se existem descrições acessíveis no DOM, se a mensagem persiste em leitor de tela, se o form preserva o rascunho, se o erro é realmente de validação ou de rede.
- **Lei aplicável com cautela:** Miller (se houver necessidade de recordar erro após desaparecer), Jakob (se padrões de feedback forem surpreendentes). Relevante considerar WCAG 3.3.1/3.3.3 **após confirmação da natureza do erro**.
- **Recomendação testável:** executar fluxo com dados inválidos e falha de rede; registrar mensagens e foco. **Somente se confirmado**, apresentar erro contextual, ação de recuperação e preservar dados; evitar substituir uma falha do servidor por aviso de “campo inválido”.
- **Impacto:** não atribuir score definitivo sem distinguir tipo de falha e alcance; lançar como hipótese `P2` para investigação, sem refatoração imediata.

## Achado UX-F02 — Muitos campos em uma tela

- **Modo:** `visual`; **classificação:** `hypothesis`; **confiança:** `baixo`.
- **Observável:** 14 campos distribuídos em uma página, alguns obrigatórios.
- **Conclusão correta:** **não existe violação de Miller ou Hick por ter 14 campos**. O desempenho depende de agrupamento, rótulos, necessidade e dados já existentes.
- **Investigação:** testar tempo de preenchimento, taxas de erro, agrupamento semântico e possibilidade de consultar informações necessárias ao lado.
- **Possível alteração:** grupos de identidade, faturamento e preferências, com subtítulos e resumo. Wizard só se houver vantagem comprovável de sequência; não introduzir passos apenas para reduzir campos visíveis.
- **Aceite:** completar cadastro sem perder valores e sem retornar a telas anteriores para conferir dados que já haviam sido informados; verificar teclado e zoom.

## Achado UX-F03 — CTA pequena aparente

- **Modo:** `visual`; **classificação:** `hypothesis`; **confiança:** `baixo`.
- **Observável:** ícone visual de confirmação de aproximadamente 16px na screenshot.
- **Não comprovado:** hit area real, espaçamento com outros targets, viewport CSS e exceções WCAG; não declarar violação 2.5.8 automaticamente.
- **Procedimento de confirmação:** inspecionar bounding box real e alvos adjacentes; testar mouse/touch, teclado e os casos do SC 2.5.8. Se pequeno sem exceção aplicável, abrir falha WCAG AA específica.
- **Remediação eventual:** aumentar área clicável e separar visualmente ações concorrentes; manter densidade adequada e rótulo acessível.

## Resumo prático

Em uma auditoria exclusivamente visual, o resultado pode conter mais **hipóteses verificáveis** do que problemas confirmados. Isso não é deficiência do relatório; é representação correta dos limites do material analisado.
