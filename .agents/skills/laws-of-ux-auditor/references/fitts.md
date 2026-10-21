# Lei de Fitts — aquisição de alvos

**Categoria:** modelo quantitativo de desempenho motor em tarefas de apontar/selecionar. **Confiança no fenômeno:** alta nas condições apropriadas; **transferibilidade:** dependente de dispositivo, tarefa, geometria, experiência e contexto.

## Fundamento e evidência

Paul M. Fitts (1954) estudou velocidade/precisão de movimentos sob diferentes amplitudes e tolerâncias de alvo. Uma formulação amplamente usada em HCI é `MT = a + b log2(D/W + 1)`, em que `MT` é tempo de movimento, `D` é distância até o alvo e `W` é a largura efetiva do alvo ao longo da direção do movimento; `a` e `b` são estimados empiricamente. A forma matemática varia entre modelos e aplicações; não trate constantes como universais. MacKenzie (1992) sistematiza a transposição ao apontamento em interfaces, os parâmetros empíricos e seus limites.

O modelo diz respeito **à aquisição motora**, não ao tempo de localizar mentalmente a ação ou entender a opção. Um botão próximo pode ser invisível; um alvo grande pode ter rótulo ambíguo. Área clicável importa mais do que área apenas desenhada. Em touch, precisão, oclusão, tamanho do dedo e contato diferem de mouse/trackpad; para teclado e tecnologia assistiva, modelos específicos substituem a tarefa de apontamento.

## Perguntas de auditoria

- Qual é a ação frequente ou crítica? Qual a distância até ela **no contexto do fluxo**, não em pixels absolutos da página?
- Qual a área interativa efetiva (`getBoundingClientRect`, hit area, padding), não só o tamanho do ícone? Existem ícones pequenos agrupados ou alvos perigosamente próximos?
- O usuário precisa cruzar grandes regiões repetidamente (ex.: painel lateral → toolbar → rodapé) em ações seriadas?
- Os controles persistem acessíveis ao fazer scroll, zoom, abrir overlay ou trocar dispositivo? Aumentar alvo sobrepõe outro?
- Há ações destrutivas próximas do alvo principal sem separação/rotulagem suficiente?

## Evidência exigida

**Ideal:** passos para repetir o apontamento; viewport/dispositivo; dimensões CSS e espaçamento dos alvos; quantidade de deslocamentos/reoperações; erros acidentais observados, se medidos. **Só captura:** registre apenas “ícone visualmente pequeno/próximo”, marque hit area como não verificada.

## Recomendações condicionais

1. Amplie a **região interativa** de ações repetidas antes de aumentar indiscriminadamente o elemento visual; preserve densidade sem sacrificar precisão.
2. Agrupe controles ao contexto e favoreça ações primárias de maior frequência em locais fáceis de adquirir.
3. Separe ações destrutivas, usando verbo explícito, confirmação quando necessária e undo quando aplicável.
4. Em cartões e canvases, diferencie área clicável do card de seus controles internos e alças de arraste. Evite colocar alvos minúsculos em arestas se não houver alternativa equivalente.
5. Compare com WCAG: SC 2.5.8 AA recomenda alvo `24×24 CSS px` **ou** exceção aplicável (inclui regra de espaçamento); SC 2.5.5 AAA exige `44×44 CSS px`, salvo exceções. Essas medidas vêm de WCAG, **não** da fórmula de Fitts.

## Contraexemplos e falhas comuns

- “Todo botão deve ter 44 px para cumprir WCAG AA” é **incorreto**: 44×44 é AAA; 24×24/condições é AA em WCAG 2.2.
- “Mover a CTA ao centro sempre melhora” pode aumentar deslocamentos nos fluxos reais e destruir consistência.
- “Tornar o card inteiro clicável” pode gerar conflitos de drag, seleção de texto, checkbox e links internos.
- “A lei comprova ganho de X ms” é inválido sem calibração experimental de `a`, `b`, dispositivo, `D`, `W` e tarefa.

## Critérios de aceite exemplares

- Área efetiva do botão foi inspecionada no navegador e não colide com controles adjacentes no zoom/viewport testado.
- Com teclado, cada controle possui ordem lógica, nome discernível e indicador de foco (ver WCAG); não extrapolar Fitts para esses testes.
- No teste com participantes, tempo de apontamento e erros de seleção não pioram na tarefa crítica; metas devem ser definidas com baseline.

**Fontes:** Fitts (1954), DOI 10.1037/h0055392; MacKenzie (1992), DOI 10.1207/s15327051hci0701_3; W3C WCAG 2.2 SC 2.5.8 e SC 2.5.5. Links em `sources.md`.
