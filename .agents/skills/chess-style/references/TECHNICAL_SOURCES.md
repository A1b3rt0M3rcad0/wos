# Referências técnicas da implementação

Estas referências fundamentam critérios adicionados pela skill. Não são evidências
sobre a tecnologia usada nas capturas. Consulta documental: 15/10/2026. A validade
operacional deve ser retestada no ambiente real do projeto.

## T1 — Preferência por movimento reduzido

MDN: a media feature `prefers-reduced-motion` permite detectar a preferência de
reduzir movimento não essencial. A skill usa essa preferência para neutralizar
parallax, reveals e deslocamentos ornamentais, mantendo o conteúdo completo.

```text
https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/At-rules/%40media/prefers-reduced-motion
```

## T2 — Interseção com o viewport

MDN: `IntersectionObserver` observa mudanças na interseção entre elementos e um
ancestral ou viewport. A skill usa isso como alternativa para saber quais seções
merecem atualização; a API não fornece, sozinha, uma timeline contínua precisa por
palavra para todos os layouts.

```text
https://developer.mozilla.org/en-US/docs/Web/API/Intersection_Observer_API
```

## T3 — Tamanho fluido

MDN: `clamp()` define limites inferior e superior para um valor preferido. Na skill,
essa técnica é uma ferramenta de composição, não garantia automática de zoom ou
reflow acessível. O resultado renderizado continua exigindo teste.

```text
https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/Values/clamp
```

## T4 — Contraste de texto

W3C, WCAG 2.2, critério 1.4.3: em geral, contraste mínimo de 4.5:1 para texto comum
e 3:1 para texto grande, com as exceções definidas no critério. O limiar de texto
grande corresponde a pelo menos 18 pt regular ou 14 pt bold, aproximadamente 24 e
18.67 CSS px. Hairlines muito finas podem exigir um tratamento mais forte mesmo
quando o cálculo nominal passa.

```text
https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html
https://www.w3.org/WAI/WCAG22/Techniques/general/G18
```

## T5 — Critérios completos de acessibilidade

W3C, WCAG 2.2: base para testes de teclado, redimensionamento de texto, reflow,
contraste, foco e tamanho de alvos. O critério 2.5.8 estabelece 24×24 CSS px para
alvos de ponteiro, com exceções. A skill adota 44×44 CSS px como meta de projeto
para controles compactos de toque, não como transcrição do mínimo AA desse critério.

```text
https://www.w3.org/TR/WCAG22/
https://www.w3.org/WAI/WCAG22/quickref/
```

## T6 — Contraste de componentes não textuais

W3C, entendimento de 1.4.11: elementos visuais necessários para identificar componentes
e estados precisam de contraste apropriado, tipicamente 3:1 nas relações descritas
no critério. Linhas puramente decorativas de fundo não têm a mesma função de um
contorno necessário para identificar um controle.

```text
https://www.w3.org/WAI/WCAG21/Understanding/non-text-contrast.html
```

## Regra de interpretação

A implementação desta skill pode satisfazer vários testes e ainda não constituir
uma auditoria completa WCAG. Não declarar conformidade AA a partir apenas do
checklist parcial ou de um smoke test automatizado.
