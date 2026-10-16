# WOS · Web Brand Kit

**Símbolo-base:** a MESMA silhueta entregue anteriormente (`194 × 82`), com o mesmo `path d` em todos os assets. Não há redesenho da geometria: módulos e conectores permanecem inalterados.

**Cor da marca:** `#FF5500`, obtida do favicon.svg da Woobe.

**Formato:** SVG vetorial, com fundo transparente sempre que o asset não for explicitamente de superfície, cartão, avatar ou social.

> As versões WOS + texto são propostas de lockup para web, usando Inter Display Bold convertido em caminhos vetoriais. Não foram fornecidas fontes oficiais ou uma versão tipográfica homologada. Não há arquivos de fonte incluídos. As cores neutras auxiliam a aplicação e não foram extraídas do favicon da Woobe.

## Estrutura

- **01-marca/** — Símbolo-mestre, cores e versão quadrada com respiro. 8 SVG.
- **02-logotipos/** — Assinaturas opcionais com a palavra WOS, texto vetorizado. 9 SVG.
- **03-favicon-app/** — Favicon, Apple Touch, PWA e ícones de aplicativos. 12 SVG.
- **04-navegacao/** — Elementos de navbar, footer, sidebar, card e avatar. 14 SVG.
- **05-badges/** — Selos e chips para interface. 6 SVG.
- **06-loading/** — Loaders SVG com animações e suporte reduced-motion. 4 SVG.
- **07-estados-vazios/** — Ilustrações de estados vazios e feedback. 7 SVG.
- **08-hero/** — Ilustrações e grafismos para landing e hero. 7 SVG.
- **09-patterns/** — Padrões repetíveis, ornamentos e marcas d’água. 8 SVG.
- **10-social/** — Capas OG e quadradas em SVG. 4 SVG.
- **11-sprites/** — Sprite SVG para uso com <use>. 1 SVG.
- **12-png-web/** — PNGs para navegadores e redes sociais. 0 SVG e 13 PNG.

## Escolha prática para cada situação

| Uso | Arquivo recomendado |
| --- | --- |
| Ícone WOS sem fundo | `01-marca/wos-symbol-primary.svg` |
| Logotipo de cabeçalho claro | `02-logotipos/wos-horizontal-light.svg` |
| Logotipo de cabeçalho escuro | `02-logotipos/wos-horizontal-dark.svg` |
| Sidebar estreita | `04-navegacao/wos-sidebar-light.svg` |
| App icon / avatar | `03-favicon-app/app-icon-orange.svg` |
| Favicon moderno | `03-favicon-app/favicon.svg` |
| Apple Touch | `12-png-web/apple-touch-icon-180.png` |
| PWA adaptativa | `12-png-web/pwa-maskable-512.png` |
| Overlay para hero | `08-hero/wos-hero-watermark-light.svg` |
| Hero ilustrativo | `08-hero/wos-hero-illustration-light.svg` |
| Fundo repetível | `09-patterns/wos-pattern-tile-light.svg` |
| Estado vazio - tarefas | `07-estados-vazios/wos-empty-tasks.svg` |
| Carregamento | `06-loading/wos-loader-pulse.svg` |
| Redes sociais / Open Graph | `12-png-web/wos-open-graph-dark.png` |
| React com customização da cor | `WosIcon.tsx` |

## Integração

- Use `USO_HTML.html` como exemplo completo de tags HTML, ícones e aplicações.
- Use `WosIcon.tsx` para embutir o desenho em React/TypeScript sem depender de fontes, imagens externas ou fetch.
- Use `wos-tokens.css` como tokens de referência (neutros opcionais).
- Use `site.webmanifest` **apenas se** o site for PWA; o manifest já referencia PNGs na mesma estrutura da pasta.
- Os arquivos de logotipo com texto têm letras convertidas em paths. Não dependem da instalação de Inter no navegador.
- Os assets com animação respeitam `prefers-reduced-motion`.
- Para meta `og:image`, use os PNGs fornecidos, com URL absoluta e pública no ambiente de produção.
- Em `<img src="...wos-symbol-currentcolor.svg">`, `currentColor` não herda a cor do documento; utilize o SVG inline (ou WosIcon.tsx) para controle via CSS.

## Padrões de qualidade

- Geometria exata do arquivo-mestre WOS reaproveitada em todas as variações.
- Sem dependências de arquivos de fontes.
- Sem imagens base64 embutidas nos SVG.
- `viewBox` presente em todos os SVG.
- Todos os SVGs parseados por XML e revisados por rasterização de amostra.
- Cor de marca da Woobe, `#FF5500`, permanece sólida no símbolo.
- Nos componentes onde aparece branco/preto em vez de laranja, trata-se apenas de versão monocromática para contraste.

## Catálogo interativo

Abra `CATALOGO.html` em um navegador para visualizar, pesquisar e baixar os SVGs individualmente.

Para compatibilidade adicional, `12-png-web/favicon.ico` inclui vários tamanhos.
