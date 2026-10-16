# Checklist de qualidade e critérios de aceite

Classificar cada item como aprovado, reprovado, não testado ou não aplicável. Um
checklist preenchido sem evidência não equivale a teste executado.

## 1. Direção de arte

- [ ] A tese da página é específica e aparece na abertura.
- [ ] Tipografia, desenho e objeto têm papéis distintos e coerentes.
- [ ] Existe contraste claro entre escala monumental e microescala.
- [ ] O fundo é contínuo e o grid permanece subordinado.
- [ ] A cor de acento é escassa e funcionalmente consistente.
- [ ] A página não foi convertida em uma grade genérica de cards.
- [ ] Os símbolos pertencem ao assunto do novo projeto.
- [ ] Nenhum nome, crédito, retrato ou conteúdo promocional da referência foi transplantado.

## 2. Tipografia e grafias

- [ ] Família utilizada e variantes disponíveis estão documentadas.
- [ ] Nenhuma substituta é anunciada como a fonte exata sem comprovação.
- [ ] Hero foi testado com o título real e não só com placeholder.
- [ ] Itálicos, pesos, acentos e descendentes não colidem nem são cortados.
- [ ] Quebras fazem sentido na largura principal e no mobile.
- [ ] Links, metadados e legendas continuam legíveis.
- [ ] Outline e estados claros não são a única forma de informação essencial.
- [ ] Texto é selecionável e a ordem semântica é íntegra.
- [ ] Texto ampliado e fonte fallback não causam perda de conteúdo.

## 3. Desenhos, objetos e notas

- [ ] Personagens têm gesto e silhueta legíveis no tamanho final.
- [ ] Uma série varia postura, proporção e direção, não só acessórios.
- [ ] A tinta parece orgânica sem depender de ruído global artificial.
- [ ] Objeto central tem relação direta com a tese.
- [ ] Recortes não possuem halo incompatível ou fundo opaco inesperado.
- [ ] Notas não cobrem CTA, preço, navegação ou informação principal.
- [ ] Notas ornamentais não carregam dados pessoais, segredos ou identificadores reais.
- [ ] Dimensões, origem, finalidade e permissões dos assets estão registradas.
- [ ] Evidências do pacote não estão sendo usadas como assets publicados.

## 4. Composição e responsividade

- [ ] Não há overflow horizontal acidental em toda a matriz.
- [ ] Masks e overflow são locais e não escondem defeitos globais.
- [ ] O hero funciona sem esperar movimento para revelar palavras decisivas.
- [ ] Margem documental se transforma em fluxo legível no mobile.
- [ ] O objeto não encobre controles em nenhuma largura intermediária.
- [ ] O grid mantém continuidade quando essa é a intenção.
- [ ] Espaços de impacto, leitura e pausa se alternam.
- [ ] O mobile preserva a gramática sem miniaturizar todos os elementos.
- [ ] Reflow a 320 CSS px e texto a 200% foram avaliados.

## 5. Motion e interações

- [ ] Parâmetros propostos não são apresentados como medições da referência.
- [ ] Toda animação possui estado final e função claros.
- [ ] O modo reduzido apresenta tudo sem deslocamento não essencial.
- [ ] O controle de leitura estática, quando presente, funciona de verdade.
- [ ] Não há scroll hijacking ou pinning obrigatório para acessar conteúdo comum.
- [ ] Nenhum estado inativo torna texto essencial ilegível.
- [ ] Links e botões têm normal, hover, foco e ativação coerentes.
- [ ] Menu móvel abre, fecha e responde a Escape adequadamente.
- [ ] Idioma, formulário ou carrossel só aparecem se realmente implementados.
- [ ] Navegação por âncoras não deixa conteúdo ou foco sob o header.

## 6. Engenharia e fallback

- [ ] CSS base apresenta uma página completa sem JavaScript.
- [ ] Fontes externas indisponíveis não quebram navegação nem conteúdo.
- [ ] Imagens têm espaço reservado e alternativa adequada.
- [ ] Não existem links `#` e botões mortos usados como aparência de funcionalidade.
- [ ] Não há erros de console ou requisições involuntárias de assets fictícios.
- [ ] Listeners, observadores e frames têm cleanup em aplicações com desmontagem.
- [ ] Tokens JSON e CSS correspondem.
- [ ] Dependências novas têm justificativa.
- [ ] Medidas de performance citadas foram realmente obtidas.

## 7. Revisão em duas passagens

**Primeira passagem — macro:** olhar a página inteira e miniaturas. Avaliar hierarquia,
ritmo, comprimento da narrativa, repetição e intensidade visual. Não começar por
ajustar um sublinhado enquanto a ordem da página está errada.

**Segunda passagem — micro:** inspecionar tipografia, notas, bordas, vinhetas, recortes,
legendas, estados, foco e mobile. Repetir a comparação depois dos ajustes para evitar
que a correção local prejudique o conjunto.

## 8. Rubrica de direção de arte

A rubrica é uma ferramenta de revisão da skill, não uma métrica científica.

| Dimensão | Peso |
|---|---:|
| Tipografia e hierarquia editorial | 25 |
| Composição, grid e ritmo | 20 |
| Linguagem gráfica e materialidade | 20 |
| Narrativa e adequação ao conteúdo | 15 |
| Motion e interações | 10 |
| Responsividade e leitura | 10 |

Atribuir de 0 a 5 por dimensão; calcular `peso × nota / 5`. Meta proposta: pelo menos
85/100, sem dimensão abaixo de 3. Uma página estática pode receber nota máxima em
motion se a ausência de animação for deliberada, coerente e cumprir o pedido.
Não penalizar simplicidade que serve ao conteúdo.

## 9. Bloqueadores de entrega

Conteúdo essencial ilegível ou inacessível; ações sem comportamento; overflow que
corta informação; uso indevido de material da referência; métricas ou provas
inventadas; alegação de testes não executados; skill sem arquivos citados necessários.
Esses problemas bloqueiam a entrega independentemente da nota visual.

## 10. Critério final

Um agente que nunca viu as capturas deve conseguir construir uma experiência
reconhecível a partir da documentação, mas não uma cópia de marca ou conteúdo.
Um usuário deve conseguir compreender, navegar e agir sem depender de ornamentos,
fontes remotas ou animações. Esse é o teste conjunto de identidade e execução.
