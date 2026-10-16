# Playbook de construção

## Fase 1 — Entendimento e escopo

Ler o conteúdo fornecido e, quando necessário, o projeto existente. Extrair uma tese,
uma tensão, uma prova e uma ação principal. Registrar restrições de marca, stack,
idioma e ativos. Não começar por uma lista de componentes de biblioteca.

Saída: `templates/CREATIVE_BRIEF.md` preenchido. Critério de avanço: mensagem e ação
estão claras; afirmações sem fonte estão identificadas, não transformadas em fatos.

## Fase 2 — Direção de arte

Formular uma metáfora que gere decisões de composição. Escolher as vozes tipográficas
e uma única função para a cor de acento. Definir objeto, notas e personagens em função
do conteúdo. Usar `templates/ART_DIRECTION_MATRIX.md`.

Saída: gramática específica da página. Critério de avanço: cada elemento visual tem
uma função; não há dependência de nomes, texto ou retratos da referência.

## Fase 3 — Storyboard

Mapear seções usando o atlas. Para cada uma, escrever objetivo, conteúdo real,
protagonista visual, pausa, transição e ação. Desenhar desktop e mobile como composições
relacionadas, não uma única imagem escalada.

Saída: mapa de seções com quebras tipográficas preliminares. Critério de avanço:
a página faz sentido em texto puro; a progressão não repete a mesma promessa.

## Fase 4 — Prova tipográfica

Testar o H1 real, uma frase narrativa, um trecho com itálico e a navegação. Avaliar
acentos, largura, peso e fallback. Somente depois ajustar tamanho final e oclusão.
Se a fonte escolhida não sustentar o contraste entre voz monumental e leitura,
trocar a família antes de adicionar ornamentos.

Saída: specimen no navegador e decisões registradas. Critério de avanço: sem clipping,
sem distorção horizontal e sem depender de uma fonte que não será disponibilizada.

## Fase 5 — Implementação estática

Construir a ordem semântica, grid, header, hero, narrativa, evidências, conclusão e
rodapé. Integrar links e ações reais. Reservar dimensões de mídia. Verificar mobile
antes do movimento.

Saída: página completa sem JS. Critério de avanço: todas as mensagens e ações estão
presentes; falha de assets não deixa a experiência inoperante.

## Fase 6 — Produção e integração dos assets

Encomendar ou criar material novo conforme o inventário. Avaliar o conjunto em
miniatura e no tamanho final. Ajustar linhas, manchas, direção do olhar e recortes.
Definir variantes quando o mobile exigir outro enquadramento.

Saída: manifesto com assets aprovados. Critério de avanço: arquivo real, finalidade,
alternativa textual, dimensões e origem resolvidos. Não usar recortes de evidência
como ativos temporários que acabam publicados por esquecimento.

## Fase 7 — Movimento e estados

Adicionar uma sequência de cada vez. Primeiro realce de leitura; depois objeto;
por último microinterações. Testar reversão, resize, troca de fonte, teclado e
preferência reduzida. Remover efeitos que não adicionam significado.

Saída: receita de motion por componente. Critério de avanço: modo estático completo,
nenhum texto obrigatório invisível e nenhum controle inacessível durante transições.

## Fase 8 — Revisão visual

Primeira passagem: comparar macrocomposição, escala, hierarquia e ritmo. Segunda:
revisar grafias, itálicos, acentos, baseline das vinhetas, notas, bordas, recortes,
legendas, foco e comportamento intermediário. Não tentar corrigir problema de
hierarquia adicionando um asset maior.

Saída: capturas identificadas e ajustes realizados. Critério de avanço: o resultado
preserva o DNA sem depender do conteúdo de origem.

## Fase 9 — Entrega

Executar o checklist e preencher o relatório. Distinguir testes executados de
recomendações pendentes. Entregar arquivos e instruções de execução no formato
pedido pelo usuário. Não afirmar réplica exata de uma implementação não inspecionada.

## Ordem de correção quando algo parecer errado

1. Mensagem, escala e composição.
2. Família tipográfica, largura e quebras.
3. Densidade, espaço negativo e posição do objeto.
4. Qualidade dos desenhos, notas e materialidade.
5. Timing, estados e pequenos acabamentos.

Corrigir em ordem inversa costuma polir uma estrutura errada. Uma animação melhor não
salva um H1 sem hierarquia ou um produto cujo conteúdo foi escondido por ornamentação.

## Ajustes rápidos por sintoma

| Sintoma | Diagnóstico provável | Primeiro ajuste |
|---|---|---|
| Parece um template SaaS | Cards e superfícies dominam | Remover caixas e reorganizar por narrativa |
| Parece uma revista genérica | Falta contraste de escalas e gesto | Introduzir microvinheta e objeto com função |
| Parece bagunçado | Protagonistas demais | Escolher um foco e reduzir ornamentos |
| Parece infantil | Personagens e acabamento inadequados | Reduzir fofura, uniformidade e volume plástico |
| Parece ilegível | Fonte, contraste ou oclusão | Corrigir leitura antes de preservar efeito |
| Mobile perdeu a identidade | Redução indiscriminada ou remoção de tudo | Recompor uma relação forte, não miniaturizar |
| Está pesado | Assets e animação sem orçamento | Medir e simplificar as camadas mais caras |
| Movimento cansa | Efeitos constantes ou amplitude excessiva | Reduzir frequência, amplitude e simultaneidade |
