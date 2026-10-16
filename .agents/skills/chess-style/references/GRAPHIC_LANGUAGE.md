# Linguagem gráfica, desenho e materialidade

## 1. Três famílias que não devem ser confundidas

**Desenho em tinta:** personagens, pequenas cenas e gestos. Traço irregular,
preenchimentos pretos e alguns sinais de chão ou movimento. **Objeto recortado:**
volume, textura, luz e perspectiva, predominantemente em escala de cinza.
**Marginalia:** notas, notações, pequenos riscos e símbolos aparentemente rápidos.

A força da linguagem está na convivência dessas famílias, não em transformá-las
num único estilo de ícone. Uma vinheta não deve ganhar o mesmo acabamento do objeto
escultórico; uma nota não deve parecer um label de dashboard.

## 2. Anatomia do traço manual

Nas capturas, o contorno não tem uniformidade mecânica. Há pequenas oscilações,
interrupções e áreas de pressão visual diferente. A linha pode ser fina em braços,
mais carregada em cabelo e muito densa em roupas. O acabamento parece desenhado à
mão; sua ferramenta de origem não foi confirmada.

Para produzir novos assets, definir no próprio desenho:

| Parâmetro | Direção proposta |
|---|---|
| Contorno | Irregular e legível; não serrilhado artificialmente |
| Espessura | Hierarquia de linha, com variação de aproximadamente 1× a 2× |
| Preenchimento | Poucas massas pretas bem localizadas |
| Sombra | Hachura curta ou mancha; evitar gradiente liso nas vinhetas |
| Textura | Local ao traço, não um filtro de ruído sobre toda a imagem |
| Perspectiva | Frontal, lateral ou três-quartos simples |
| Fundo | Transparente ou exatamente compatível com a superfície |
| Silhueta | Reconhecível em tamanho pequeno |

As proporções numéricas acima são orientação de produção. Não são medições das
ilustrações observadas. Renderizar a arte no tamanho final para decidir a espessura,
em vez de aprová-la apenas em um zoom confortável de desenho.

## 3. Personagens

Os personagens têm diferença clara entre cabeça, tronco, pernas e acessórios. Há
variação de altura, postura, massa corporal e roupa. Os rostos são econômicos; a ação
é comunicada sobretudo pela silhueta. Pequenos desvios de anatomia dão expressão sem
transformar os personagens em volumes infantis de proporções uniformes.

A unidade vem de linha, paleta e nível de detalhe. Não vem de repetir o mesmo rosto,
a mesma inclinação de cabeça e o mesmo corpo com um acessório diferente.

### Matriz mínima de variação

Criar diferenças em pelo menos três eixos por personagem: silhueta, gesto, orientação,
acessório, distribuição de tinta, altura relativa e expressão. Manter o mesmo regime
de acabamento. Uma fila de dez personagens deve parecer um conjunto coerente, não
a duplicação de um único asset.

## 4. Repertório de ações e funções narrativas

| Ação | Papel editorial | Cuidado de composição |
|---|---|---|
| Observar algo pequeno | Curiosidade, descoberta | Deixar espaço na direção do olhar |
| Caminhar com carga | Progresso e esforço | Chão curto e gesto legível |
| Empurrar uma massa | Resistência, problema | A massa não pode encobrir o texto |
| Ler ou inspecionar | Compreensão | Evitar detalhes impossíveis em microescala |
| Conectar dois pontos | Coordenação ou relação | Usar poucas linhas, não virar organograma |
| Escolher uma direção | Decisão | Integrar à mudança de argumento |
| Equilibrar objetos | Prioridade e trade-off | Silhueta primeiro, acessório depois |
| Organizar pequenas peças | Ordem | Evitar objetos numerosos demais |
| Cooperar com outro | Trabalho compartilhado | Diferenciar postura dos dois |
| Descansar após a tarefa | Respiro narrativo | Reservar uma zona menos densa |
| Olhar para trás | Revisão ou memória | Direção deve apoiar a narrativa |
| Formar uma fila | Pluralidade e fechamento | Baseline comum e espaçamento variável |

Esse repertório é novo e transferível. Não corresponde a uma autorização para copiar
personagens ou a um inventário completo de todos os gestos existentes na referência.

## 5. Adaptação para personagens não humanos

A mesma linguagem admite pequenos robôs, ferramentas personificadas ou criaturas
abstratas quando o assunto justifica. Conservar desenho em tinta, proporções esguias
ou assimétricas, articulações simples e gestos claros. Não substituir por plástico
brilhante, neon, olhos luminosos, cabeças esféricas idênticas ou render 3D de mascote.

Construir diferenças mecânicas reais: um modelo sobre rodas, outro com pernas finas,
outro com braço telescópico; um corpo alto e outro baixo. Dar a cada um uma tarefa,
não apenas uma pose. A metáfora e as capacidades devem ser definidas pelo conteúdo
do produto, nunca inventadas pela skill.

## 6. Prompt-base para encomendar novos desenhos

> Criar uma ilustração editorial monocromática original de [SUJEITO] realizando
> [AÇÃO], com silhueta [DESCRIÇÃO] e orientação [DIREÇÃO]. Traço preto de tinta com
> pequenas irregularidades orgânicas, contornos finos, algumas massas de preto,
> hachura contida e ausência de sombreado suave. Personagem pequeno, gesto legível,
> detalhes econômicos, aparência de desenho editorial observacional. Sem cenário
> extenso; apenas um apoio de chão discreto se necessário. Fundo transparente.
> A imagem será usada a [LARGURA FINAL] em uma composição tipográfica sobre papel
> cinza-claro. Não incluir texto, marca, fotografia, neon, cor, moldura ou efeito 3D.

Antes de gerar, preencher sujeito, ação, direção e tamanho. Pedir expressão específica
em vez de “expressivo”. Para uma série, fornecer uma matriz de diferenças e a mesma
referência de traço. Avaliar todos os assets juntos e em tamanho final.

## 7. Marginalia e notações

A escrita marginal parece um pensamento periférico. Na evidência há combinações
curtas de letras, números, hífens e sinais. Sua rotação e densidade variam. Não é uma
segunda camada de informação indispensável à navegação.

Em um projeto sobre estratégia, notações algébricas podem ser coerentes. Em outro
assunto, substituir por uma gramática equivalente: IDs curtos de experimento,
sequências, coordenadas de pesquisa ou marcas de revisão. Definir o conjunto antes
de espalhar símbolos. Não usar credenciais, dados pessoais ou IDs reais de cliente
como decoração.

Proposta inicial: 3–7 notas visíveis em uma composição larga, com tamanho de 16–28 px
e rotação entre −12° e +12°. Em mobile, reduzir para 1–3. São limites de partida;
a densidade deve ser avaliada pelo vazio disponível e não por uma meta de contagem.

Se forem ornamentais, podem ter baixo contraste e `aria-hidden="true"`. Se carregarem
informação real, devem sair da camada decorativa e ter contraste e semântica adequados.

## 8. Linhas, riscos e sublinhados

A grade pertence à estrutura gráfica de fundo. O pequeno chão pertence à vinheta.
O sublinhado de link pertence à interação. O rabisco de ênfase pertence à composição.
Não usar a mesma linha padronizada para essas quatro funções.

Um sublinhado desenhado pode conter duas passagens não coincidentes e uma ligeira
curva, mas precisa continuar discreto. Usar uma versão vetorial própria; não extrair
assinaturas ou marcas pessoais da imagem de referência.

Setas, estrelas, círculos, marcadores e asteriscos são expansões possíveis, não
presenças confirmadas em todos os materiais. Cada adição deve justificar seu papel.

## 9. Objetos recortados de aparência material

O objeto principal deve condensar o conceito. Pode ser uma construção física,
assemblage, papel dobrado ou instrumento relacionado ao assunto. O recorte apresenta
volume próprio e contorno que rompe a regularidade das letras. A iluminação deve
ser coerente internamente, sem transformar a página em uma cena 3D inteira.

Proposta de produção: enquadramento frontal ou três-quartos, saturação muito baixa,
textura legível, sombras internas mais importantes que sombras projetadas,
transparência de fundo e margem suficiente para rotação. Evitar retângulo branco
embutido. Um asset estático bem produzido basta; não exigir WebGL para simular volume.

Um grande objeto pode ter pequenos parentes narrativos em seções posteriores, mas
não repetir literalmente o mesmo recorte dez vezes como um padrão de fundo.

## 10. Artefatos documentais

Imagens pequenas na margem podem funcionar como prova: páginas, diagramas, objetos de
processo, resultados ou documentação. Manter legenda honesta e curta. Não aplicar
uma estética tão forte que torne a evidência ilegível. Não inventar selos, prêmios,
marcas de clientes ou resultados para obter aparência de credibilidade.

No mobile, trazer cada artefato para o fluxo próximo da afirmação correspondente.
Não ocultar prova importante apenas porque a margem deixou de existir.

## 11. Especificação de exportação

Para cada asset registrar: finalidade, orientação, dimensões intrínsecas, proporção,
zona de segurança, variantes, formato, transparência, alternativa textual,
procedência e permissão de uso. SVG é adequado quando o desenho vetorial preserva
qualidade; raster pode ser melhor para textura real de tinta. O formato depende do
material, não de uma suposição sobre a ferramenta de origem.

Não expandir cegamente um raster pequeno. Preparar a resolução a partir do tamanho
de exibição e densidade esperada, oferecendo variantes responsivas quando necessário.
Remover metadados desnecessários e verificar contornos sobre `#E9E9E9`, não só no
fundo branco do editor.

## 12. Critérios de rejeição

Rejeitar: linhas perfeitamente uniformes em todas as peças; traço tão fino que some;
ruído global usado como atalho para “manual”; personagens indecifráveis em pequena
escala; fundo opaco incompatível; peso de tinta que rouba o protagonismo; ícones de
bibliotecas misturados sem adaptação; gestos idênticos; objeto sem relação com a tese;
e composição que usa assets para esconder problemas de tipografia.
