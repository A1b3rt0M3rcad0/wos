# Inventário visual e contrato de assets

## 1. Inventário de elementos observados

`E` = estrutural; `ID` = identitário; `NAR` = narrativo; `INT` = interativo;
`ORN` = ornamental. A evidência aponta para uma captura, não um arquivo de produção.
Dimensões exatas e formatos de origem não foram verificados salvo onde indicado.

| ID | Elemento | Evidência | Classes | Papel e observação |
|---|---|---|---|---|
| V01 | Fundo cinza contínuo | S01–S07 | E/ID | Pixels predominantes `#E9E9E9` |
| V02 | Grid ortogonal fino | S01–S07 | E/ID | Linhas visíveis; não prova o grid DOM |
| V03 | Notações manuscritas curtas | S01–S07 | ID/ORN | Cinza, pequena escala e inclinação variável |
| V04 | Personagem isolado inicial | S01 | NAR/ID | Figura central pequena em campo vazio |
| V05 | Identificação textual no header | S02 | E/INT | Hierarquia compacta; conteúdo específico não reutilizado |
| V06 | Informação de edição | S02 | ORN/NAR | Texto menor junto à identificação |
| V07 | Links de navegação | S02 | INT | Estados de execução não testados |
| V08 | Controle compacto de idioma | S02 | INT | Aparência de cápsula contornada; opções não verificadas |
| V09 | Ação textual sublinhada | S02 | INT | Informação secundária menor ao lado |
| V10 | Linha de contexto do hero | S02 | NAR | Serifada menor acima da composição |
| V11 | Capitais verdes monumentais | S02/S03 | E/ID | Cor sólida amostrada `#00846A` |
| V12 | Objeto central escultórico | S02 | ID/NAR | Monocromático, vertical e sobreposto |
| V13 | Figura pequena na baseline | S02/S03 | ID/NAR | Contraste de escala com a letra |
| V14 | Linha de chão da vinheta | S03 | ORN | Ancoragem leve, irregular |
| V15 | Recorte circular material | S03 | NAR/ORN | Textura e relevo próprios |
| V16 | Indicação desenhada de scroll | S02 | ORN/INT | Aparência visível; clicabilidade não confirmada |
| V17 | Texto narrativo em escala grande | S04/S05 | E/NAR | Preto e cinza no mesmo conjunto |
| V18 | Figura caminhando entre palavras | S04 | NAR/ID | Vinheta integrada à composição tipográfica |
| V19 | Cena com papéis | S04 | NAR/ID | Pequena ação com maior largura visual |
| V20 | Cena de interação entre figuras | S04 | NAR/ID | Pouca tinta e várias silhuetas |
| V21 | Figura empurrando massa | S05 | NAR/ID | Gesto comunica esforço |
| V22 | Figura em deslocamento com objeto | S05 | NAR/ID | Ação lateral, traço manual |
| V23 | Papel recortado em diagonal | S04/S05 | NAR/ID | Dobras e profundidade aparente |
| V24 | Texto claro/intermediário | S04/S05 | ID | Estado visual; timing não confirmado |
| V25 | Marcador circular de contexto | S06 | NAR | Apenas função composicional é extraída |
| V26 | Identificação contextual em capitais menores | S06 | NAR | Hierarquia abaixo do marcador |
| V27 | Texto editorial com itálicos | S06 | ID/NAR | Ênfases entre trechos regulares |
| V28 | Artefatos documentais laterais | S06 | NAR | Evidência na margem, fora de cards |
| V29 | Legendas pequenas de artefatos | S06 | NAR | Voz utilitária subordinada |
| V30 | Tese final em peso mais forte | S07 | E/NAR | Escala grande e centralização |
| V31 | Letras de contorno | S07 | ID | Outline visível; técnica não confirmada |
| V32 | Fila de personagens variados | S07 | NAR/ID | Silhuetas, roupas e posturas diferentes |
| V33 | Links textuais de rodapé | S07 | INT | Organização discreta no fundo contínuo |
| V34 | Sublinhado de aparência manual | S07 | ID/ORN | Risco gráfico distinto da linha do grid |

O inventário cobre os tipos visíveis nas capturas, não todos os elementos do site.
Não há confirmação, a partir dessas imagens, de modal, formulário, arraste, carrossel,
custom cursor, microinteração específica ou técnica de carregamento.

## 2. Assets necessários para uma nova implementação

| ID de produção | Necessidade | Variante |
|---|---|---|
| A-HERO-01 | Um objeto-síntese original e monocromático | Largo e mobile, quando o recorte exigir |
| A-VIN-01…N | Vinhetas específicas da narrativa | Diferentes gestos e orientações |
| A-NOTE-01…N | Notações curtas da gramática escolhida | SVG próprio ou tipografia apropriada |
| A-PROOF-01…N | Evidências reais do produto/projeto | Tamanho legível e legenda própria |
| A-CAST-01…N | Conjunto de personagens para o fechamento | Reorganização responsiva, não só escala |
| A-MARK-01 | Pequeno risco/sublinhado original | Versão que não substitui foco funcional |

A quantidade depende da narrativa. Não encomendar dez figuras quando duas resolvem
a página. A skill deve descrever o necessário e registrar o que falta, não preencher
lacunas com material indevido da referência.

## 3. Contrato de asset

Cada entrada de `templates/ASSET_MANIFEST.json` possui ID, finalidade, status,
arquivos, dimensões, proporção, origem, permissão, texto alternativo, uso decorativo,
pontos de ancoragem e variantes. Um asset só vira “ready” quando o arquivo existe,
a finalidade está correta e a permissão foi resolvida.

Campos desconhecidos devem permanecer `null` ou explicitamente pendentes. Não
preencher dimensões imaginárias nem declarar autoria/permissão sem confirmação.

## 4. Camadas de documentação versus produção

`evidence/` contém recortes das imagens fornecidas para estudo. O starter não depende
deles. Nenhum recorte é apresentado como asset original criado pela skill.
O pipeline de produção deve manter seus próprios assets fora desse diretório.

Os exemplos de desenho descritos nos prompts são instruções de encomenda, não arquivos
gráficos completos. A fonte editorial é uma escolha a ser integrada pelo projeto;
nenhum binário de fonte é incluído. Isso evita prometer que um fallback terá o mesmo
aspecto em todos os sistemas.

## 5. Verificação de recortes e transparência

Inspecionar arestas sobre o fundo real. Um halo branco pode ser invisível no editor
e evidente sobre `#E9E9E9`. Conferir recorte de mãos, pés, linhas de chão e detalhes
finos. Não cortar extremidades para caber em uma caixa antes de testar outra escala.

Objetos que serão rotacionados precisam de margem de segurança. Imagens documentais
não devem receber um recorte que esconda o dado que justificava sua inclusão.

## 6. Nomes de arquivo

Usar nomes sem dados pessoais: `hero-object-desktop.webp`, `vignette-inspect-left.svg`,
`vignette-connect-front.svg`, `proof-process-step-02.webp`, `note-sequence-03.svg`.
Evitar nomes genéricos como `final2-new.png` ou nomes de pessoas da referência.
