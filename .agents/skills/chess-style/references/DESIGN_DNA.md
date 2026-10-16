# Design DNA — o que faz a linguagem ser chess-style

## 1. Conceito organizador

A experiência parece uma página editorial ampliada para ocupar a tela, atravessada
por vestígios de pensamento e pequenas cenas desenhadas. O rigor vem de alinhamentos,
serifas e grid; a humanidade vem de personagens, escrita irregular e situações
visuais. A materialidade vem de recortes monocromáticos com volume aparente.

A unidade não depende de usar um tabuleiro. Ela depende da tensão entre **sistema e
gesto**, **enorme e minúsculo**, **texto plano e objeto material**, **ordem e desvio**.
Usar apenas uma fonte clássica e um fundo cinza preserva a paleta, mas perde a ideia.

## 2. Hierarquia de identidade

### Camada indispensável

A composição editorial e a tipografia dominam a atenção. Há grandes diferenças de
escala. O fundo é contínuo e discreto. O texto não fica sempre contido em caixas.
Mesmo uma página sem animação precisa ser reconhecível por essas decisões.

### Camada fortemente identitária

Grid fino, acento verde contido, marginalia manuscrita, pequenas vinhetas em tinta e
alternância entre regular e itálico. Esse conjunto estabelece uma linguagem própria,
e não apenas uma estética editorial genérica.

### Camada narrativa variável

Objetos, personagens, metáforas, coordenadas e gestos mudam com o conteúdo. Em um
produto de software, o assunto pode ser dependências e decisões; em uma exposição,
relações entre memória e matéria. Não transplantar dinheiro, peças ou personagens
sem relação semântica com a nova mensagem.

### Camada técnica substituível

Framework, biblioteca de animação, implementação do grid e formato de imagem não são
a identidade. Trocar CSS por SVG para desenhar uma linha não altera o DNA; trocar
toda a hierarquia por cards arredondados altera.

## 3. As oito relações fundamentais

| Relação | Como produzir | Como verificar |
|---|---|---|
| Monumental × íntimo | Título muito grande junto de figura pequena | A diferença de escala continua evidente em miniatura |
| Rígido × orgânico | Grid ortogonal e notas de inclinação irregular | A grade parece deliberada; os desenhos não parecem ícones |
| Tinta × papel | Fundo neutro, preto e cinzas; verde pontual | O fundo não ganha mais atenção que o texto |
| Plano × volumétrico | Letras planas atrás de objeto recortado material | O objeto não parece um card aplicado sobre outro card |
| Silêncio × ênfase | Vazios generosos alternados com manifesto denso | Há pausas perceptíveis entre argumentos |
| Literal × metafórico | Conteúdo claro acompanhado por imagem alusiva | Retirar a imagem não destrói a informação essencial |
| Ordem × interrupção | Personagem ocupa uma lacuna prevista na frase | A pausa visual não quebra a leitura nem a semântica |
| Estado × transformação | Texto e objetos mudam de maneira controlada | O estado final é legível e o modo estático é completo |

## 4. Gramática de combinação

Uma cena forte combina uma mensagem principal, uma relação de escala e um gesto
visual secundário. Como ponto de partida, usar **um protagonista por viewport**:
ou o título, ou o objeto, ou uma frase narrativa. Outro elemento pode atravessá-lo,
mas não deve criar uma segunda disputa igualmente intensa.

Exemplo de composição: título de duas linhas ocupa grande parte da largura;
objeto estreito sobrepõe a junção das linhas; uma figura de tinta aparece perto da
baseline; quatro notas ficam nas zonas de vazio. Cada elemento tem uma função.

Exemplo de composição ruim: título, oito moedas, muitos rabiscos, vídeo, gradiente,
mascote, botão chamativo e painel de métricas competindo simultaneamente. A origem
comum dos elementos não produz coerência por si só.

## 5. Regras de cor

Papel cinza-claro é superfície, preto é informação, verde é assinatura e cinza baixo
é ambiente. Não distribuir verde em todos os textos; sua eficácia depende de
contraste com o restante predominantemente acromático. Cor funcional não é idêntica
a cor ornamental: texto pequeno exige uma combinação apropriada de contraste.

O objeto principal permanece monocromático por padrão. Introduzir fotografia colorida
ou um segundo acento exige justificativa explícita e teste de dominância. Não fazer
essa expansão apenas para “deixar mais moderno”.

## 6. Regras de grafia

Há três vozes principais: display de grande escala, leitura editorial e pensamento
manuscrito. A itálica é uma inflexão da voz editorial, não uma nova família aleatória.
A voz utilitária pode aparecer em legendas pequenas e metadados, sempre secundária.

Pontuação, sublinhado, outline e quebras também fazem parte da gramática. Um trecho
em itálico altera o tom; uma palavra vazada pode sugerir transição; uma nota lateral
sugere pensamento fora do texto principal. Cada recurso deve ter papel definido.

## 7. Narrativa transferível

A abertura apresenta a tese por imagem e escala. O manifesto explicita uma tensão.
O desenvolvimento mostra uma mudança de compreensão. Demonstração e evidência
introduzem substância. A seção de credibilidade acrescenta contexto humano ou
institucional. O encerramento condensa a tese e oferece uma ação inequívoca.

Essa sequência é uma **proposta de construção**, não uma obrigação de inserir sete
seções. Em uma página curta, três blocos podem cumprir todas essas funções. A métrica
é cobertura da mensagem, não quantidade de telas.

## 8. Liberdades permitidas

Trocar assunto, símbolos, objeto central, personagens e idioma. Adaptar a família
serifada quando os ativos licenciados exigirem. Alterar intensidade do verde dentro
da mesma função cromática. Remover movimento. Reorganizar a ordem da narrativa.
Reduzir ornamentos. Fazer uma versão curta ou longa.

Preservar: contraste de escala, qualidade tipográfica, hierarquia, espaço negativo,
articulação entre texto e desenho e ausência de superfícies genéricas concorrentes.

## 9. Descaracterizações frequentes

**Tema literal de xadrez:** muitos ícones de peças e casas alternadas, mas sem
composição editorial. **SaaS genérico:** grid de features, badges em pílula, CTA
volumoso e mockup dentro de card. **Luxo genérico:** serifada dourada, fundo preto e
nenhum gesto manual. **Sketch genérico:** todos os elementos irregulares, perdendo
contraste com uma estrutura precisa. **Galeria de efeitos:** cada bloco anima de
um jeito e não há ritmo de leitura.

## 10. Teste de identidade sem conteúdo

Cobrir os nomes e trocar o texto por linhas da mesma extensão. A composição ainda
deveria revelar: tipografia dominante, desenho em escala pequena, cor pontual e
geometria discreta. Se a identificação depende de uma marca, do objeto de origem ou
de uma citação literal, o sistema ainda não foi extraído corretamente.
