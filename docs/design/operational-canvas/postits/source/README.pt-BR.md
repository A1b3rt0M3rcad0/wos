# WOS — Post-its para Canvas

Coleção de post-its e papéis ilustrados **em branco**, para cards de Plan, Work, Issues, tarefas, objetivos e dependências do WOS.

## Conteúdo

- **10 estilos** de papel/cartão desenhados à mão.
- **8 combinações de cores** (exclusivamente tokens oficiais fornecidos).
- **2 proporções**: `square` (800 × 800) e `wide` (1000 × 730).
- **160 artes SVG editáveis** e **160 PNGs transparentes**, renderizados a 1,5× (1200 × 1200 / 1500 × 1095).
- `preview/`: exemplos da coleção, sem texto dentro dos post-its.
- `manifest.json`: inventário completo, nomes, caminhos, cores e margens seguras para conteúdo dinâmico.
- `tokens.css`: variáveis CSS de cores oficiais.
- `examples/canvas.html`: catálogo local e exemplo de aplicação de texto pela camada HTML, **nunca embutido no SVG**.

## Paleta exata

| Token | Cor | Uso recomendado |
|---|---|---|
| Canvas | `#EFEDE8` | Canvas e variante de papel neutro |
| Workspace | `#F5F3EF` | Papel neutro alternativo |
| Surface | `#FFFDF9` | Papel claro e contraste com Canvas |
| Ink | `#292620` | Contornos/tipografia e opção de fundo escuro |
| Copper | `#A7441D` | Papel de ênfase, status importante |
| Selected | `#F1DFD0` | Papel destacado sem saturação |
| Grid | `#C2BCB2` | Papel cinza neutro, elementos secundários |
| Brand | `#FF5500` | Papel de alta ênfase visual |

**Não foram introduzidas novas cores**. Os desenhos não usam gradiente, sombras realistas nem texto ou checkboxes dentro do papel.

## Estilos

- **`curled`** — Canto inferior enrolado.
- **`classic`** — Post-it clássico irregular.
- **`rounded`** — Bordas arredondadas orgânicas.
- **`dogear`** — Dobra no canto superior.
- **`torn`** — Borda inferior rasgada.
- **`taped`** — Fita adesiva no topo.
- **`pinned`** — Alfinete superior.
- **`tabbed`** — Aba de organização.
- **`notebook`** — Bloco perfurado.
- **`stacked`** — Folhas sobrepostas.


## Como aplicar no Canvas

O **SVG é a imagem de fundo**, enquanto o conteúdo (títulos, tarefas, estados, badges, interações) fica na camada HTML/React, mantendo legibilidade, acessibilidade e dados dinâmicos.

```html
<article class="wos-note">
  <img src="svg/wide/curled/selected.svg" alt="" class="wos-note__art" />
  <div class="wos-note__content">
    <h3>Título do objetivo</h3>
    <p>Informações do Plan, Work, Issue ou tarefa</p>
  </div>
</article>
```

```css
.wos-note {
  position: relative;
  width: 360px;
  aspect-ratio: 1000 / 730;
}
.wos-note__art {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: fill;
  pointer-events: none;
}
.wos-note__content {
  position: absolute;
  inset: 16% 14% 15%;
  overflow: auto;
  color: var(--wos-ink, #292620);
}
```

As margens seguras recomendadas por estilo estão no `manifest.json`; para `pinned`, `taped`, `tabbed` e `notebook`, amplie o recuo superior. Para `curled`, preserve a dobra inferior direita. Em `copper` e `ink`, use texto `Surface (#FFFDF9)`; as demais variações usam `Ink (#292620)` como preferência visual (confirme contraste WCAG conforme tamanho efetivo do texto).

**Importante para front-end:** imagens rasterizadas não podem ser redimensionadas sem limites; prefira os SVGs como fonte em produção. Para grades densas, use `wide` como card principal, `square` como objetivo/nó grande. As proporções são intencionais; não estique uma forma 1:1 para 4:3. Não coloque títulos ou controles dentro do arquivo de imagem.

## Convenção de arquivos

`svg/<proporção>/<estilo>/<token>.svg` e `png/<proporção>/<estilo>/<token>.png`.

Exemplos:
- `svg/wide/curled/selected.svg`
- `svg/wide/taped/surface.svg`
- `svg/square/pinned/brand.svg`
- `png/wide/stacked/copper.png`

Licenciamento: assets originais criados para esta solicitação, prontos para integrar ao projeto.
