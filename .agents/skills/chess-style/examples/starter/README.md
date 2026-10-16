# Starter local — chess-style

Este exemplo demonstra estrutura editorial, tokens, dois tamanhos de composição,
texto com realce progressivo, navegação compacta e modo estático. Ele não é uma
reconstrução pixel-perfect e não contém os assets de produção da referência.

## Executar

Abrir `index.html` diretamente ou servir a pasta raiz da skill com
`python -m http.server 8000` e acessar a rota `/examples/starter/`.

Não há pacotes npm, chamadas a API, fontes remotas, analytics ou dependências de CDN.
Os dois CSS são locais. O objeto de papel é uma construção esquemática em CSS; o
pequeno esquema de relações é SVG inline. Não representam uma série de ilustrações
editoriais finalizadas.

## Controles reais

A navegação conduz a seções existentes. No mobile, o menu abre e fecha com o botão;
Escape fecha e retorna o foco quando aberto. O controle “Leitura estática”, no rodapé,
ativa/desativa o realce. A preferência de movimento reduzido do sistema prevalece.
Sem JS, a navegação fica visível e o conteúdo permanece completo.

## Personalizar

Substituir o conteúdo antes de ajustar a escala final. Integrar fontes próprias com
permissão adequada usando os aliases de `tokens.css`. Substituir o objeto esquemático
e o desenho por assets específicos com dimensões e alternativas registradas. Alterar
o JSON de tokens e regenerar CSS em vez de criar valores concorrentes.

O caso de uso ilustrado é uma reflexão editorial sobre decisões. Não apresenta
métricas, clientes ou capacidades comerciais fictícias.

## Limites

O navegador escolhe uma fonte local disponível; a forma dos glifos varia por sistema.
O controle de aparência não é persistido entre visitas. Não há formulário, checkout,
carrossel, seleção de idioma ou backend. Acrescentar esses recursos somente quando
o projeto pedir e quando puderem funcionar de verdade.

Em uma SPA, use os mesmos contratos de inicialização e cleanup. O exemplo expõe
`window.chessStyle.destroy()` para demonstrar a limpeza; não é uma API estável de
biblioteca nem um substituto de integração com o ciclo de vida do framework.
