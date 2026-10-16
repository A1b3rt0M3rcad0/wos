# Sistema tipográfico e grafias

## 1. O que a referência permite afirmar

As capturas mostram serifas editoriais de alto contraste, hastes verticais marcadas,
fios muito finos, capitais grandes e proporções relativamente estreitas em vários
textos narrativos. Há regular, itálico, pesos mais fortes, outline e escrita manual.
O desenho é compatível com uma direção de serifas modernas/display. Essa descrição
morfológica não identifica uma família comercial nem comprova que exista uma única
família por trás de todos os tratamentos.

**Fonte exata: não verificada.** Os arquivos de fonte não foram inspecionados. Não
listar uma família como a original com base apenas em semelhança. Alternativas devem
ser avaliadas como candidatas, nunca equivalências confirmadas.

## 2. As quatro vozes

### A. Display monumental

Capitais de grande escala, contraste forte, pouca distância entre linhas e
espaçamento óptico compacto. A fonte precisa sobreviver a tamanhos muito grandes:
terminais e serifas ganham protagonismo. A composição pode ter duas linhas
cuidadosamente distintas, mas não deve depender de deformação horizontal.

A largura da palavra determina a escala viável. Um título de nove letras não pode
receber automaticamente o mesmo tamanho de uma palavra de cinco. Ajustar a direção
do título real, e não um placeholder curto que será substituído depois.

### B. Editorial de leitura expressiva

Serifada regular, ritmo alongado e presença de ascendentes/descendentes. O texto
narrativo funciona como imagem e informação. Em desktop admite linhas curtas em
grande escala; em mobile vira uma composição mais compacta, não uma linha infinita.

A centralização é um recurso recorrente da referência, não uma obrigação universal.
Para parágrafos longos ou informação técnica densa, escolher alinhamento à esquerda
e uma medida adequada preserva a voz editorial sem impor leitura sinuosa.

### C. Inflexões: itálico, peso e contorno

Itálico enfatiza um conceito ou trecho sem criar uma caixa. Bold acrescenta peso à
tese final. Outline introduz um estado gráfico, mas não é o tratamento adequado para
um parágrafo informativo inteiro. Usar a variante itálica real quando disponível;
a inclinação sintética não reproduz o desenho dos glifos cursivos.

### D. Escrita marginal e utilitária

Notas têm aparência rápida, irregular, levemente inclinada e pouco contraste.
Metadados e legendas podem empregar serifada menor ou sans-serif discreta. Não usar
uma fonte handwriting em textos funcionais extensos: ela deve parecer anotação,
não se tornar a voz de toda a interface.

## 3. Anatomia para avaliar uma candidata

Testar `M N W R Q C S a e g f i t 1 2 4 7 0 & ? ! ( )` e acentos de português.
Observar largura de M/W, perna de R, curva de S, contraformas de e/a, descendentes,
serifas de N e terminais em itálico. Examinar pares `AV`, `To`, `Wa`, `fi`, `fl`,
`Ta` e palavras longas do projeto. Esses testes são critérios de seleção, não
características já confirmadas da fonte de referência.

Avaliar quatro contextos simultaneamente: display de 180–280 px; frase de 64–96 px;
parágrafo de 20–28 px; navegação de 16–20 px. Uma única fonte não precisa vencer os
quatro. Uma segunda família utilitária é aceitável se permanecer subordinada.

## 4. Escala de implementação proposta

Todos os valores abaixo são **P**, não medidas de CSS original. Os tokens expressam
pontos de partida e devem ser calibrados após carregar a fonte real.

| Papel | Expressão inicial | Entrelinha | Tracking | Uso |
|---|---|---|---|---|
| Hero | `clamp(3.25rem, 1rem + 12.5vw, 19rem)` | 0.82–0.94 | −0.025em a −0.055em | Composição de 1–3 linhas |
| Manifesto | `clamp(2rem, 1rem + 5vw, 6.75rem)` | 1.08–1.2 | −0.025em | Argumento curto e expressivo |
| Tese final | `clamp(2.5rem, 1.25rem + 6vw, 8rem)` | 1.02–1.14 | −0.035em | Conclusão |
| Título de seção | `clamp(2rem, 1.1rem + 3vw, 4.25rem)` | 1.08–1.18 | −0.025em | Hierarquia secundária |
| Corpo editorial | `clamp(1.15rem, 0.95rem + 0.7vw, 1.75rem)` | 1.42–1.65 | −0.01em a 0 | Explicação contínua |
| Navegação | `clamp(1rem, 0.9rem + 0.25vw, 1.2rem)` | 1.3 | 0 | Links e ações |
| Legenda | `0.8125rem` a `0.9375rem` | 1.4–1.55 | 0 | Evidência e metadados |
| Nota manual | `clamp(1rem, 0.75rem + 1vw, 1.75rem)` | 1 | Variável por asset | Elemento marginal |

Faixas não dispensam teste de zoom. Uma fórmula que mistura `rem` e `vw` pode ainda
produzir resultados inadequados em determinadas larguras. Validar a composição com
texto ampliado, configurações de fonte do usuário e palavras extensas. Ver T2/T3 em
`TECHNICAL_SOURCES.md`.

## 5. Matriz de propriedades

Declarar deliberadamente `font-family`, `font-weight`, `font-style`, `font-size`,
`line-height`, `letter-spacing`, `text-align` e `text-transform`. Não adicionar
propriedades sofisticadas sem um objetivo verificável.

`font-stretch`: apenas para uma família que ofereça os desenhos correspondentes.
`font-variation-settings`: somente após verificar os eixos disponíveis no arquivo.
`font-feature-settings`: preservar ligaduras comuns; avaliar alternativas caso a fonte
as exponha. `font-synthesis: none` pode evitar falsos pesos/itálicos, mas exige que os
estilos corretos realmente existam. Não ativar isso sem testar o fallback.

`text-shadow`: não usar para engordar toda a família artificialmente. `text-stroke`:
restrito ao recurso outline. `text-decoration`: adequado para links e ênfase textual;
para rabisco visual, utilizar uma camada decorativa separada do estado funcional.

## 6. Font stack e carregamento

O pacote não distribui fontes. O starter usa:

```css
--ch-font-editorial: "Chess Editorial", "Bodoni 72", "Iowan Old Style",
  "Times New Roman", serif;
--ch-font-utility: Arial, Helvetica, sans-serif;
--ch-font-hand: "Chess Hand", "Segoe Print", "Bradley Hand", cursive;
```

`Chess Editorial` e `Chess Hand` são aliases de integração, não nomes de fontes
incluídas. O resultado varia por sistema até o projeto configurar arquivos próprios.
Quando houver fonte adequada e liberada, criar `@font-face` com URLs reais, estilos
corretos e estratégia de carregamento testada. Não manter URLs fictícias em produção.
Não extrair fontes do sistema, não anexar arquivos binários ao pacote e não
considerar a disponibilidade no computador como autorização de redistribuição.

Aguardar `document.fonts.ready` quando medidas JavaScript dependem dos glifos. Reservar
altura suficiente para evitar que a troca de fonte cubra o CTA. O fallback deve manter
o conteúdo acessível mesmo quando a composição não for visualmente idêntica.

## 7. Quebra de linha e alinhamento óptico

Desenhar as quebras de um hero como parte da direção de arte. Em títulos editoriais,
`<br>` pode expressar uma decisão válida; em parágrafos, não inserir quebras para
imitar um único screenshot. Revalidar em todos os idiomas e tamanhos.

Antes de alterar tracking, verificar se a palavra já excede a largura por causa da
fonte. Evitar tracking menor que −0.06em sem inspeção cuidadosa: letras podem parecer
fundidas. Esse limite é uma heurística de projeto, não uma regra universal tipográfica.

Ajustar margens ópticas de uma inicial larga, descendentes e pontuação final. Não
usar margens negativas globais para alinhar todas as fontes como se suas métricas
fossem idênticas. Uma linha com itálico pode precisar de respiro lateral extra.

## 8. Tratamentos e grafismos tipográficos

| Tratamento | Mecanismo | Restrição |
|---|---|---|
| Capitais gigantes | Tamanho, tracking e quebra controlados | Manter significado legível |
| Itálico intercalado | `<em>` quando a ênfase é semântica | Não inclinar parágrafos inteiros sem necessidade |
| Bold editorial | Variante real de peso | Não competir com o hero em toda seção |
| Outline | Camada decorativa com `-webkit-text-stroke` e fallback | Nunca única forma de texto informativo pequeno |
| Mudança tonal | Cor ou camada de preenchimento | Estado inativo deve continuar legível |
| Texto sobreposto a imagem | Empilhamento e área de oclusão planejada | Não ocultar ação, números ou conteúdo indispensável |
| Texto parcialmente recortado | Máscara local em material decorativo | Não cortar instruções ou links |
| Nota marginal | Asset vetorial manuscrito ou família própria | `aria-hidden` se estritamente decorativa |
| Sublinhado irregular | SVG/path separado ou pseudo-elemento | Foco e link continuam reconhecíveis sem o rabisco |
| Letras separadas | Spans visuais com semântica preservada | Não fragmentar a leitura do leitor de tela |

Não há obrigação de usar todos os tratamentos. A combinação de regular, itálico,
monumental e notas já estabelece uma base forte.

## 9. Vinhetas dentro de frases

Inserir em uma pausa real. A vinheta deve ter tamanho intrínseco reservado e
`vertical-align` calibrado. Uma largura inicial de 0.8–1.6em pode funcionar para uma
figura simples; cenas maiores precisam de uma lacuna própria ou de um bloco fora
da frase. Esses valores são **P**.

Quando a figura for decorativa, manter a sentença como texto contínuo e usar imagem
com `alt=""`; uma quebra de linha não deve transformar a figura em palavra. Quando
o desenho comunicar informação exclusiva, oferecer alternativa textual apropriada.
Não substituir substantivos essenciais por ilustrações sem equivalência acessível.

No mobile, mover a vinheta para depois da frase se a pausa gerar linhas órfãs.
Não diminuir a imagem até ela virar ruído ilegível.

## 10. Testes de aceitação tipográfica

Verificar: nenhuma colisão entre acentos e linha anterior; nenhuma descendente
cortada; itálico não ultrapassa o contêiner; peso e fonte efetivamente carregados;
texto selecionável; números e símbolos consistentes; navegação não excessivamente
pequena; zoom de 200%; idioma com strings 30% maiores; fallback sem rolagem horizontal.

Capturar a tela com o título real. A decisão final deve se apoiar na renderização,
não apenas em valores corretos no arquivo CSS.
