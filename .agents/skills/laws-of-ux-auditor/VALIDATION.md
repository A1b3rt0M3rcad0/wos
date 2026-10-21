# Verificação e limitações de instalação

**Versão:** 1.0.0. **Data:** 19/10/2026.

## Estrutura de uma Agent Skill

- A raiz da skill é `laws-of-ux-auditor/` e contém `SKILL.md` com frontmatter YAML.
- `name` em kebab-case confere com o diretório. `description` explica quando ativar.
- O corpo do `SKILL.md` define procedimento e carrega os documentos auxiliares sob demanda.
- Arquivos de apoio separados em `references/`, `assets/templates/`, `assets/examples/` e `scripts/`.
- Nenhum arquivo de apoio exige serviço de rede, dependência Python externa ou recursos de outro repositório.

## Testes locais reproduzíveis

Execute a partir da raiz da skill:

```bash
python scripts/validate_skill.py .
python scripts/test_skill.py
python scripts/validate_audit.py assets/examples/sample-audit.json
python scripts/scaffold_audit.py --product "Site exemplo" --scope "Dashboard" --mode visual --out /tmp/exemplo-ux-audit
python scripts/validate_audit.py /tmp/exemplo-ux-audit/audit-record.json
```

Esses testes verificam arquivos obrigatórios, parsing do frontmatter (YAML completo se `PyYAML` estiver instalado; fallback limitado sem a dependência), links internos, JSON de exemplo, geração de scaffold e consistência aritmética das pontuações.

## Validação oficial e ressalvas

A especificação oficial está em https://agentskills.io/specification e documenta a ferramenta de referência:

```bash
skills-ref validate ./laws-of-ux-auditor
```

**O comando de referência acima não foi executado no ambiente usado para construir este pacote, porque a ferramenta `skills-ref` não estava instalada e a instalação online estava indisponível.** A conformidade apresentada resulta de comparação documental com a especificação e de validação estrutural local, não de uma certificação externa.

Além disso, a inspeção do **ZIP** verifica apenas integridade do arquivo e que a skill extraída mantém estrutura/testes. A aceitação final pela ferramenta hospedeira depende de sua versão, caminho de instalação e permissões.

## Escopo da validação de relatórios

`validate_audit.py` rejeita itens sem campo de observação, evidência, critério de aceite ou prioridade, além de notas fora do intervalo 1–5 e índices incoerentes. O script **não prova veracidade das evidências, causalidade psicológica, acessibilidade real, impacto em usuários ou conformidade normativa**. Esses pontos exigem auditoria e testes contextualizados.
