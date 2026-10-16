# Base de cores da Woobe aplicada ao WOS

> **Objetivo:** reutilizar a identidade cromática já implementada na Woobe como fundação do novo visual do **WOS**, sem copiar a composição da interface administrativa da Woobe. Este documento descreve a paleta existente e apresenta um mapeamento **proposto** para o WOS; o mapeamento não representa código já implementado no WOS.

## 1. Origem e escopo

A paleta operacional foi extraída dos seguintes arquivos do repositório **Woobe**:

- [`packages/woobe-ui/src/styles/theme.css`](https://github.com/A1b3rt0M3rcad0/woobe/blob/3dbcb3e1a7559f4a50ebca320e411933d46e8012/packages/woobe-ui/src/styles/theme.css): variáveis CSS operacionais, com valores Light e Dark.
- [`packages/woobe-ui/src/tokens/colors.ts`](https://github.com/A1b3rt0M3rcad0/woobe/blob/3dbcb3e1a7559f4a50ebca320e411933d46e8012/packages/woobe-ui/src/tokens/colors.ts): aliases de cores usados no Tailwind e tema de código.
- [`packages/woobe-ui/brand-kit/design-tokens/colors.css`](https://github.com/A1b3rt0M3rcad0/woobe/blob/3dbcb3e1a7559f4a50ebca320e411933d46e8012/packages/woobe-ui/brand-kit/design-tokens/colors.css): paleta de marca, incluindo o laranja `#FF5500`.

**Referência examinada:** commit `3dbcb3e1a7559f4a50ebca320e411933d46e8012` (branch `master`, 15/10/2026).

**Distinção essencial:** o laranja de marca **`#FF5500`** não é o mesmo que o cobre operacional **`#A7441D`** usado como cor de links e destaques no tema claro. A cor `primary` de botões da Woobe é, no tema claro, **carvão `#292620`**. A proposta para o WOS preserva essas funções distintas.

## 2. Paleta operacional — Light

| Papel | Token original da Woobe | Cor | Uso no WOS |
|---|---|---|---|
| Canvas | `--woobe-canvas` | `#EFEDE8` | Fundo estrutural da aplicação |
| Workspace | `--woobe-workspace` | `#F5F3EF` | Área de grafos, roadmaps e dependências |
| Surface | `--woobe-surface` | `#FFFDF9` | Cards de Outcomes, Tasks e painéis |
| Elevated | `--woobe-elevated` | `#FFFEFC` | Popovers, menus e superfícies elevadas |
| Panel band | `--woobe-panel-band` | `#F6F3EE` | Cabeçalhos e faixas discretas |
| Control | `--woobe-control` | `#EAE6DF` | Controles secundários e estados neutros |
| Border | `--woobe-border` | `#D4CEC4` | Bordas e divisores |
| Control border | `--woobe-control-border` | `#857A6C` | Limites de inputs e controles |
| Ink / Foreground | `--woobe-ink` | `#292620` | Texto principal, botões primários e ilustração |
| Secondary | `--woobe-secondary` | `#4B463F` | Texto secundário com maior ênfase |
| Muted | `--woobe-muted` | `#615B52` | Metadados, descrições e legendas |
| Link / Copper | `--woobe-link` | `#A7441D` | Destaque, links e caminho relevante |
| Primary | `--woobe-primary` | `#292620` | Fundo das ações primárias |
| Primary hover | `--woobe-primary-hover` | `#403A32` | Hover de ações primárias |
| On primary | `--woobe-on-primary` | `#FFFDF9` | Texto/ícones sobre `primary` |
| Selected | `--woobe-selected` | `#F1DFD0` | Nó, tarefa ou item selecionado |
| Focus | `--woobe-focus` | `#A7441D` | Contorno de foco por teclado |
| Grid | `--woobe-grid` | `#C2BCB2` | Malha sutil dos diagramas |
| Edge | `--woobe-edge` | `#857A6C` | Arestas/conexões padrão dos grafos |

### Cores semânticas — Light

| Papel | Token original | Cor | Uso no WOS |
|---|---|---|---|
| Success | `--woobe-success` | `#246846` | Task/Outcome concluído |
| Warning | `--woobe-warning` | `#865B13` | Pendência, atenção, validação parcial |
| Error | `--woobe-error` | `#B23834` | Erro e falha; bloqueio quando efetivamente representar estado negativo |
| Info | `--woobe-info` | `#365F85` | Informação complementar |
| Network | `--woobe-network` | `#6D41A8` | Identificação pontual de redes ou estruturas relacionadas |

### Cor de marca

| Papel | Origem | Cor | Regra |
|---|---|---|---|
| Brand orange | `woobe-ui/brand-kit` e alias `brand-mark` | `#FF5500` | Reservar para aplicações da marca; não substituir indiscriminadamente o cobre operacional |

## 3. Paleta operacional — Dark

| Papel | Token original da Woobe | Cor |
|---|---|---|
| Canvas | `--woobe-canvas` | `#191A17` |
| Workspace | `--woobe-workspace` | `#1D1F1B` |
| Surface | `--woobe-surface` | `#252721` |
| Elevated | `--woobe-elevated` | `#2D3028` |
| Panel band | `--woobe-panel-band` | `#20221D` |
| Control | `--woobe-control` | `#171915` |
| Border | `--woobe-border` | `#41453A` |
| Control border | `--woobe-control-border` | `#777E6C` |
| Ink / Foreground | `--woobe-ink` | `#F0EEE7` |
| Secondary | `--woobe-secondary` | `#D5D2C6` |
| Muted | `--woobe-muted` | `#C0BDAF` |
| Link / Copper | `--woobe-link` | `#F1A16F` |
| Primary | `--woobe-primary` | `#F0EEE7` |
| Primary hover | `--woobe-primary-hover` | `#FFFDF9` |
| On primary | `--woobe-on-primary` | `#1D1F1B` |
| Selected | `--woobe-selected` | `#3D2C21` |
| Focus | `--woobe-focus` | `#F6B68B` |
| Grid | `--woobe-grid` | `#4C5144` |
| Edge | `--woobe-edge` | `#929985` |
| Success | `--woobe-success` | `#8AC6A0` |
| Warning | `--woobe-warning` | `#E0BE73` |
| Error | `--woobe-error` | `#F09187` |
| Info | `--woobe-info` | `#90B7DC` |
| Network | `--woobe-network` | `#BEA4EB` |

> **Nota:** a barra de navegação da Woobe possui sobrescritas próprias e permanece escura mesmo no tema Light. O WOS pode adotar esse padrão, mas não é obrigatório para compartilhar a identidade cromática.

## 4. Hierarquia cromática proposta para o WOS

```text
IDENTIDADE COMPARTILHADA WOOBE + WOS
│
├── Neutros quentes
│   ├── Canvas ─────────── Fundo do produto
│   ├── Workspace ──────── Editor de grafos e roadmaps
│   ├── Surface ────────── Cards, painéis, tarefas e outcomes
│   ├── Elevated ───────── Menus e overlays
│   ├── Control ────────── Botões e controles secundários
│   ├── Border ─────────── Separação e estrutura
│   └── Ink / Muted ────── Hierarquia tipográfica e ilustrações
│
├── Interação
│   ├── Primary ────────── Ação principal (carvão / off-white)
│   ├── Copper ─────────── Destaque, links e caminhos importantes
│   ├── Selected ───────── Seleção de tarefa, outcome ou nó
│   └── Focus ──────────── Acessibilidade por teclado
│
├── Diagrama de dependências
│   ├── Workspace ──────── Fundo do canvas
│   ├── Grid ───────────── Malha discreta
│   ├── Edge ───────────── Conexão padrão
│   ├── Copper ─────────── Conexão destacada
│   └── Estados ────────── Conclusão, atenção, erro, informação
│
└── Identidade ilustrativa WOS
    ├── Traço monocromático
    └── Robozinho preservado, sem recolorização
```

## 5. Como aplicar por componente

| Componente WOS | Fundo | Texto | Bordas / Arestas | Realce |
|---|---|---|---|---|
| Layout geral | `canvas` | `foreground` | `border` | — |
| Canvas de dependências | `workspace` | `foreground` | `grid` + `edge` | `copper` |
| Outcome card | `surface` | `foreground` | `border` | `selected` ao selecionar |
| Task card | `surface` | `foreground` / `muted` | `border` | `selected` ao selecionar |
| Inspector de Task/Outcome | `surface` / `elevated` | `foreground` | `border` | `copper` para links/ações |
| Botão primário | `primary` | `on-primary` | `primary` | `primary-hover` |
| Botão secundário | `control` | `foreground` | `control-border` | `surface` / `selected`, conforme estado |
| Input | `surface` | `foreground` | `control-border` | `focus` |
| Conexão de dependência | — | — | `edge` | `copper` quando destacada |
| Task/Outcome concluído | Neutro | `success` | Neutro | Indicador `success` |
| Pendência ou atenção | Neutro | `warning` | Neutro | Indicador `warning` |
| Falha | Neutro | `error` | Neutro | Indicador `error` |
| Ilustração do robô | Transparente | `foreground` no traço | — | Sem recolorização automática |

**Regras de composição:**

1. **Neutros dominam a tela.** Superfícies e divisores transmitem hierarquia; cores saturadas não devem dominar o diagrama.
2. **Cobre é sinal de relevância, não decoração universal.** Reservar para links, foco de atenção, nós/caminhos destacados e ações apropriadas.
3. **Estado semântico não deve depender só de cor.** Exibir rótulos ou ícones para `completed`, `failed`, `blocked`, etc.
4. **Task e Outcome devem compartilhar a base de superfície.** A diferenciação deve resultar de conteúdo, ícone, posição e estados, não de uma paleta arbitrária por entidade.
5. **Manter a ilustração independente.** O robozinho monocromático é um elemento de identidade próprio do WOS.
6. **Dark não deve ser uma simples inversão automática.** Usar os valores semânticos escuros especificados.

## 6. Tokens CSS sugeridos para o WOS

Os nomes `--wos-*` abaixo são **novos aliases propostos**; seus valores refletem a base cromática real da Woobe. É possível implementá-los em um `tokens.css` ou módulo equivalente do WOS.

```css
:root {
  color-scheme: light;

  /* Estrutura */
  --wos-canvas: #efede8;
  --wos-workspace: #f5f3ef;
  --wos-surface: #fffdf9;
  --wos-elevated: #fffefc;
  --wos-panel-band: #f6f3ee;
  --wos-control: #eae6df;
  --wos-border: #d4cec4;
  --wos-control-border: #857a6c;

  /* Texto */
  --wos-foreground: #292620;
  --wos-secondary: #4b463f;
  --wos-muted: #615b52;

  /* Ação e interação */
  --wos-primary: #292620;
  --wos-primary-hover: #403a32;
  --wos-on-primary: #fffdf9;
  --wos-copper: #a7441d;
  --wos-selected: #f1dfd0;
  --wos-focus: #a7441d;

  /* Diagramas */
  --wos-grid: #c2bcb2;
  --wos-edge: #857a6c;

  /* Estados */
  --wos-success: #246846;
  --wos-warning: #865b13;
  --wos-error: #b23834;
  --wos-info: #365f85;
  --wos-network: #6d41a8;

  /* Identidade de marca, uso pontual */
  --wos-brand-orange: #ff5500;
}

:root[data-theme='dark'] {
  color-scheme: dark;

  --wos-canvas: #191a17;
  --wos-workspace: #1d1f1b;
  --wos-surface: #252721;
  --wos-elevated: #2d3028;
  --wos-panel-band: #20221d;
  --wos-control: #171915;
  --wos-border: #41453a;
  --wos-control-border: #777e6c;

  --wos-foreground: #f0eee7;
  --wos-secondary: #d5d2c6;
  --wos-muted: #c0bdaf;

  --wos-primary: #f0eee7;
  --wos-primary-hover: #fffdf9;
  --wos-on-primary: #1d1f1b;
  --wos-copper: #f1a16f;
  --wos-selected: #3d2c21;
  --wos-focus: #f6b68b;

  --wos-grid: #4c5144;
  --wos-edge: #929985;

  --wos-success: #8ac6a0;
  --wos-warning: #e0be73;
  --wos-error: #f09187;
  --wos-info: #90b7dc;
  --wos-network: #bea4eb;
}

/* Exemplos de composição */
.wos-workspace {
  background: var(--wos-workspace);
  color: var(--wos-foreground);
}

.wos-task-card {
  background: var(--wos-surface);
  border: 1px solid var(--wos-border);
  color: var(--wos-foreground);
}

.wos-task-card[aria-selected='true'] {
  background: var(--wos-selected);
  border-color: var(--wos-copper);
}

.wos-primary-button {
  background: var(--wos-primary);
  color: var(--wos-on-primary);
  border: 1px solid var(--wos-primary);
}

.wos-primary-button:hover {
  background: var(--wos-primary-hover);
}

.wos-graph-edge { stroke: var(--wos-edge); }
.wos-graph-edge[data-highlighted='true'] { stroke: var(--wos-copper); }

:where(button, a, input, select, textarea):focus-visible {
  outline: 2px solid var(--wos-focus);
  outline-offset: 2px;
}
```

## 7. Diretriz de identidade entre os produtos

A **Woobe** e o **WOS** devem parecer produtos do mesmo ecossistema devido a seus **neutros quentes, tipografia escura, superfícies claras e destaques em cobre**. Não precisam compartilhar a mesma navegação, densidade, layout ou componentes.

- **Woobe:** identidade de um painel operacional de sistemas de IA.
- **WOS:** identidade de um ambiente de organização de Outcomes, Tasks e dependências, com visual editorial e ilustrações monocromáticas.

**Decisão de design recomendada:** reutilizar a **paleta operacional semântica** da Woobe, manter os **aliases `--wos-*`** no WOS e conservar o robô/ilustrações monocromáticos como linguagem visual particular do produto.

---

**Status:** especificação cromática proposta para o WOS, baseada em valores verificados no repositório Woobe. Não é uma auditoria de implementação do WOS nem uma proposta de alteração dos arquivos da Woobe.
