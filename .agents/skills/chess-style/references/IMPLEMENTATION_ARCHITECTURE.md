# Arquitetura de implementação

## 1. Princípio

A direção de arte é independente de framework. Preservar a stack existente quando
ela puder atender a semântica, composição e interações. Não introduzir uma biblioteca
ou canvas apenas porque a página tem movimento e objetos de aparência 3D.

O mínimo funcional é HTML semântico, CSS com tokens e JavaScript progressivo. O
starter demonstra esse nível. Uma aplicação React, Vue ou outra pode mapear os mesmos
papéis para componentes, respeitando os contratos e a limpeza de efeitos.

## 2. Modelo de conteúdo

Separar conteúdo editorial, configuração de composição e assets. Um título não deve
ficar duplicado dentro de um SVG, dentro do componente e dentro de um arquivo de
animação. Centralizar a verdade textual e renderizar seus tratamentos visuais.

Modelo proposto:

```text
Page
  identity: nome próprio do projeto, idioma, links reais
  narrative: seções ordenadas com função e conteúdo
  typography: aliases e variantes disponíveis
  composition: breakpoints, escalas e pontos de ancoragem
  assets: manifesto com origem, dimensões e alternativas
  motion: política estática/reduzida e receitas habilitadas
```

As capacidades reais do produto vêm do brief. A skill decide apresentação, não
inventa recursos, preços ou resultados.

## 3. Contratos de componentes

| Componente | Responsabilidade | Não deve fazer |
|---|---|---|
| `EditorialPage` | Superfície, grid e ordem narrativa | Coordenar cada frame de todos os filhos |
| `EditorialHeader` | Links, seção atual e navegação compacta | Reimplementar histórico ou fingir tradução |
| `MonumentalHero` | Título e relação com objeto-síntese | Hardcode de palavras de referência |
| `NarrativeParagraph` | Texto contínuo e realce opcional | Renderizar texto essencial somente em canvas |
| `InlineVignette` | Asset, tamanho e alinhamento semântico | Alterar a frase ou deslocar layout ao carregar |
| `MarginalNote` | Ornamento local e não interativo | Carregar informação obrigatória escondida |
| `ArtifactRail` | Evidências próximas de afirmações | Inventar provas ou prender coluna no mobile |
| `ClosingStatement` | Tese final e conjunto visual | Copiar conteúdo do hero automaticamente |
| `MotionPolicy` | Preferência do sistema e modo estático | Reativar efeitos contra a política escolhida |

Esses nomes são sugestão de organização, não dependência de uma biblioteca.

## 4. Tokens como contrato único

`tokens/DESIGN_TOKENS.json` contém valores, classificação de evidência, confiança e
notas. `scripts/build_tokens.py` produz as custom properties de
`tokens/tokens.css`. Alterar o JSON e regenerar; não editar silenciosamente os dois
arquivos em direções diferentes.

As regras locais de composição podem criar tokens adicionais. Documentar o papel e
o motivo. Não espalhar variações quase iguais de verde ou espaçamento por dezenas
de componentes quando um token resolve a intenção.

## 5. Texto com realce progressivo

O texto base chega completo no HTML ou na renderização inicial. Somente após init
bem-sucedida é permitido adicionar wrappers visuais. Preservar espaços, `<em>`, links
e a ordem de leitura. Não usar `innerHTML` com conteúdo arbitrário para dividir texto.

Se usar uma cópia decorativa para outline/preenchimento, mantê-la fora da árvore
acessível e sincronizada com a fonte textual única. O modo estático remove a
necessidade dessas camadas; não deve esconder o conteúdo.

O starter faz divisão de nós textuais em spans e preserva os espaços. O recurso é
aplicado somente a parágrafos explicitamente marcados, evitando interferir com
controles, formulários e navegação.

## 6. Agendamento e geometria

Agrupar leituras de `getBoundingClientRect()` antes de escrever estilos. Em scroll,
agendar no máximo um `requestAnimationFrame` pendente. Sem evento, não precisa
existir frame. Uma mudança de fonte, resize ou preferência pode solicitar nova
atualização.

Para páginas longas, observar proximidade de seções com IntersectionObserver e
limitar trabalho às relevantes. Não supor que a API fornece a timeline precisa de
leitura sem uma função adicional de progresso. Ver T2 em `TECHNICAL_SOURCES.md`.

Em SPA, cancelar frames, remover listeners e desconectar observadores na desmontagem.
Não registrar efeitos repetidos em cada render. Definir um caminho de teardown e
testá-lo ao sair e voltar à rota.

## 7. Isolamento de transformações

Um wrapper controla posicionamento e rotação-base; outro controla deslocamento
animado; o asset interno controla recorte e dimensões. Alternativamente, usar custom
properties em um `transform` composto. O objetivo é impedir que uma atualização
apague outra decisão de direção de arte.

Não aplicar `will-change` permanentemente em centenas de nós. A promoção de camadas
não é um substituto universal para medir performance. Não animar sombras e filtros
pesados em toda a página sem demonstrar necessidade e custo aceitável.

## 8. Estratégia de mídia

Reservar dimensões intrínsecas de imagens. A imagem principal do hero merece uma
estratégia de carregamento prioritário quando comprovadamente necessária; imagens
fora da primeira dobra podem ser carregadas depois. Testar o impacto real, sem
atribuir um resultado de performance antes de medir.

Escolher vetor ou raster conforme o material. A textura de um desenho pode perder
qualidade ao ser vetorizada automaticamente. Um único arquivo enorme não é uma
estratégia responsiva: preparar variantes quando a resolução e o peso justificarem.

WebGL e modelos 3D são opcionais, só quando interação volumétrica real estiver no
escopo. Aparência de volume em uma captura não exige essa infraestrutura.

## 9. Orçamento inicial de engenharia

Propostas para uma landing editorial simples, não métricas medidas da referência:

| Recurso | Meta inicial de projeto |
|---|---|
| JavaScript específico de motion | Preferir até cerca de 30 KB comprimidos |
| Imagem principal | Tentar ficar abaixo de 300–450 KB na variante adequada |
| Vinheta pequena | Buscar dezenas de KB, não centenas, quando o material permitir |
| Dependências novas | Zero por padrão; justificar cada inclusão |
| Layout | Nenhum deslocamento causado por ausência de dimensão do asset |
| Atualização por frame | Nenhuma medição por caractere em loops de scroll |

Não sacrificar a qualidade do desenho para cumprir um número arbitrário. Medir,
comparar alternativas e documentar decisões. Testar um dispositivo de capacidade
intermediária além de uma máquina de desenvolvimento rápida.

## 10. Formulários, compra e funcionalidades externas

A skill não oferece checkout ou backend pronto. Quando o projeto pedir um formulário,
integrar endpoint real, validação, envio, feedback e estados de falha. Não colocar
um formulário decorativo que sempre exibe “sucesso”. Quando não houver infraestrutura,
explicitar isso no plano em vez de fingir uma integração.

No starter, as ações são âncoras locais reais e um controle de apresentação. Não
há compra, coleta de dados, links inventados ou simulação de pagamento.

## 11. Instrumentação e validação

Separar erros funcionais, problemas de composição e avaliação artística. O smoke
test pode detectar overflow, falha de JS e regressão de navegação; ele não decide se
a ilustração é coerente com o gesto editorial. Capturas devem registrar viewport,
modo de movimento e fonte usada.

Não reportar Lighthouse, Core Web Vitals, FPS, teste de Safari ou conformidade WCAG
se essas medições não foram executadas. Um resultado de Chromium local só vale para
a configuração testada.

## 12. Entregável técnico

Código legível, componentes com responsabilidade clara, tokens centralizados,
conteúdo real, asset manifest, interações completas, fallback e relatório do que foi
testado. Não entregar apenas uma descrição de “como ficaria” quando o pedido era
implementar a interface.
