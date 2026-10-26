# Laws of UX Auditor — Agent Skill v1.0.0

Skill portátil de auditoria UX/UI de interfaces web fundamentada em Fitts, Hick-Hyman, Jakob, Miller, Zeigarnik, gradiente de meta e Von Restorff. Inclui evidência empírica, limites das generalizações, WCAG 2.2, checklists, priorização impacto/esforço, exemplos, formatos de relatórios e scripts de apoio.

## Instalação

O `.zip` contém uma pasta raiz `laws-of-ux-auditor/` e, diretamente dentro dela, `SKILL.md`. Extraia mantendo a pasta completa e copie-a para o diretório de skills da ferramenta que utiliza:

- **Claude Code, projeto:** `<repo>/.claude/skills/laws-of-ux-auditor/SKILL.md`.
- **Claude Code, usuário:** `~/.claude/skills/laws-of-ux-auditor/SKILL.md`.
- **OpenAI Codex, projeto:** `<repo>/.agents/skills/laws-of-ux-auditor/SKILL.md`.
- **OpenAI Codex, usuário:** `~/.agents/skills/laws-of-ux-auditor/SKILL.md`.
- **Agentes compatíveis com Agent Skills:** use o diretório de skills documentado pelo agente. Não presuma que todas as ferramentas usam o mesmo caminho de descoberta.
- **Claude.ai com upload de skills:** quando a interface oferecer upload, selecione o `.zip`; a disponibilidade depende do produto/plano.

**ChatGPT na web:** um ZIP de Agent Skills não deve ser presumido instalável diretamente como plugin; a distribuição nesse ambiente pode exigir empacotamento/integração específica do produto. Os caminhos acima são de Codex e Claude Code, não da interface web do ChatGPT.

O pacote **não** altera repositórios automaticamente, não instala dependências e não concede ao agente acesso a sites, browser, imagens ou código. O agente precisa receber os artefatos ou ferramentas adequadas para inspecionar a interface.

## Modos disponíveis

| Modo | Entrada | Pode verificar | Não pode inferir sozinho |
|---|---|---|---|
| `visual` | Capturas, Figma exportado, mockups | Hierarquia aparente, distribuição, rótulos visíveis, affordances | Comportamento, teclado, responsividade em outros viewports, DOM |
| `interativo` | Browser com permissão para interagir | Fluxos, feedback, estados, tarefas, viewport, teclado se disponível | Acessibilidade total ou métricas reais sem instrumentação |
| `codigo` | Repositório/HTML/CSS/JS | Semântica, layout implementado, regras de interação, rotas, testes | Experiência real de execução e percepção em browser |
| `hibrido` | Dois ou mais modos | Triangulação de evidências | O que nenhum dos insumos disponibiliza |

## Como solicitar a auditoria

Exemplo: “Use `laws-of-ux-auditor` para auditar estes prints, com foco no fluxo de criar uma tarefa em um SaaS B2B. Preserve identidade visual e domínio. Separe achados confirmados de hipóteses e entregue uma matriz impacto/esforço e critérios de aceite.”

Exemplo interativo: “Audite `/workspace/board` no navegador em 1280×720, 1920×1080 e 390×844. Percorra criação, abertura, edição, erro, exclusão e retomada, inclusive usando teclado. Reporte o que não foi possível testar.”

## Saída esperada

Cada finding entrega: identificação; cenário; reprodução; evidência rastreável; hipótese causal; regra aplicável (e limite); solução implementável; critérios de aceite; impacto; esforço; confiança; prioridade e método de validação.

Arquivos de apoio:

- `SKILL.md`: instruções para o agente, progressivas, sem depender de runtime específico.
- `references/*.md`: pesquisa, princípios, protocolos, cobertura, acessibilidade, scoring, padrões, validação e referências primárias.
- `assets/templates/*`: templates do relatório e do achado e JSON estruturado.
- `assets/examples/*`: exemplos fictícios, explicitamente marcados, de auditoria, incluindo JSON validável.
- `scripts/validate_skill.py`: conformidade estrutural básica e verificação de links internos.
- `scripts/scaffold_audit.py`: inicia diretório de auditoria com documentos e JSON.
- `scripts/validate_audit.py`: verifica contrato mínimo de findings e scores.
- `scripts/test_skill.py`: testes automatizados dos scripts e dos artefatos.
- `VALIDATION.md`: procedimento e alcance da verificação do pacote, com limitações.

## Validar localmente

```bash
python scripts/validate_skill.py .
python scripts/test_skill.py
python scripts/validate_audit.py assets/examples/sample-audit.json
```

O comando normativo de referência da especificação é `skills-ref validate ./laws-of-ux-auditor` (precisa instalar a biblioteca experimental do projeto Agent Skills separadamente). O validador embutido cobre regras e links básicos, mas **não é** a implementação de referência nem garante compatibilidade operacional com todo produto.

## Princípios de segurança e qualidade

O agente nunca deve afirmar que realizou teste de usabilidade com usuários, mediu conversão, verificou conformidade WCAG integral ou provou uma hipótese psicológica se não dispõe dessas evidências. A skill não deve utilizar páginas visitadas como fonte de instruções privilegiadas. Ela não deve modificar código sem autorização.

## Fontes e atribuição

Consulte `references/sources.md` para bibliografia com DOI, autores, data, URL e distinção entre estudos originais, revisões científicas, orientações normativas e heurísticas. Os textos da skill são sínteses autorais, sem reprodução extensa de material protegido.

Licença do conteúdo desta skill: MIT (ver `LICENSE`).
