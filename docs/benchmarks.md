# Medições de continuidade — 2026-10-07

Go 1.27.1, Linux amd64, Intel Xeon Platinum 8573C, GOMAXPROCS=4. SQLite pelo driver WASM pinado; PostgreSQL 18.6 em contêiner local. Fixture: 101 Outcomes, dois com 100 WorkItems; o segundo tem um critério obrigatório e uma avaliação attestation por WorkItem. Página de 25. Cada operação inclui Application e serialização JSON. Benchtime mínimo 500 ms. Nenhuma consulta/teste concorrente externa foi executada durante a contagem SQL.

| Banco / consulta | Média (ms) | JSON (bytes) | SQL/op | Bytes alocados/op | Alocações/op |
| --- | ---: | ---: | ---: | ---: | ---: |
| SQLite / descoberta | 0,345 | 7.505 | 1 | 35.610 | 381 |
| SQLite / snapshot simples | 3,663 | 32.238 | 23 | 935.386 | 5.953 |
| SQLite / snapshot com avaliações | 7,535 | 32.627 | 32 | 1.320.093 | 12.514 |
| PostgreSQL / descoberta | 1,962 | 7.505 | 3 | 41.865 | 565 |
| PostgreSQL / snapshot simples | 15,135 | 32.238 | 25 | 972.341 | 7.070 |
| PostgreSQL / snapshot com avaliações | 19,682 | 32.627 | 34 | 1.405.476 | 15.817 |

Leituras em lote preservam base, autoria, critérios, revisões, Evidence, avaliações e conclusões. Nove queries de histórico substituem consultas por proprietário/assessment/conclusão nas listagens de Objectives/WorkItems. A fixture simples medida nesta mesma máquina antes da otimização usava 822 instruções SQL em SQLite. A diferença de SQL é verificável; tempos variam com hardware/carga e não são comparados à máquina de 5 de outubro.

| Quatro leitores, mesmo Outcome com avaliações | Throughput | Latência média por chamada | Espera média do pool/op |
| --- | ---: | ---: | ---: |
| SQLite, pool de uma conexão | 95,48 snapshots/s | 41,479 ms | 28,296 ms |
| PostgreSQL, pool de até 16 conexões | 43,17 snapshots/s | 90,454 ms | 0 ms |

O benchmark paralelo mede leitura no mesmo Outcome. PostgreSQL serializa o guard desse Outcome; portanto mais conexões não garantem ganho neste cenário. Zero espera de pool não significa zero contenção do guard. Não foram medidos percentis, throughput de escrita, múltiplos Outcomes ou capacidade de produção.

SQLite conta instruções com TRACE_STMT. PostgreSQL soma chamadas de `public.pg_stat_statements`, descontando a consulta de estatística; a extensão deve estar habilitada e acessível. Sem ela o benchmark omite essa métrica. Tempos/alocações incluem overhead de instrumentação. Leituras seriais não esperaram por conexão. Métricas de guard em runtime incluem preparação/aquisição e não apenas tempo bloqueado.

A resposta é limitada, mas o snapshot ainda reconstrói coleções completas; custo de histórico individual e algumas projeções cresce com o dataset. Estes números são uma fixture de aceitação e não metas de desempenho ou garantia para datasets arbitrários.

```sh
GOMAXPROCS=4 go test ./packages/wos-core/storage/sqlite ./packages/wos-core/storage/postgres -run '^$' -bench BenchmarkContinuity -benchmem -benchtime=500ms
```

Configure WOS_TEST_POSTGRES_DSN com um banco de testes. Resultado bruto: `docs/audit/release-2026-10-07-benchmarks.txt`. `BenchmarkContinuity` e a variante PostgreSQL são mantidos pelo gerador e verificados contra drift.

---

## Histórico de 5 de outubro

# Medições de continuidade — 2026-10-05

Estas são medições locais, não metas de desempenho nem um resultado de produção. Go 1.27.1, Linux amd64, AMD EPYC 9V74 (80-Core Processor), `GOMAXPROCS=8`, SQLite via driver WASM do projeto, diretório temporário. Fixture: 100 Outcomes; o primeiro tem 100 WorkItems; páginas de 25 itens. Cada iteração inclui consulta Application e serialização JSON. Duração mínima de cada benchmark: 500 ms.

| Consulta | Tempo médio | Resposta JSON | Memória por operação | Alocações |
| --- | ---: | ---: | ---: | ---: |
| Descoberta, 100 Outcomes | 195.194 ns (0,195 ms) | 7.402 bytes | 35.626 bytes | 380 |
| Snapshot, 100 WorkItems | 65.329.583 ns (65,330 ms) | 32.156 bytes | 2.768.710 bytes | 53.759 |

A descoberta usa leitura de metadados e índices. O snapshot limita a resposta, mas ainda reconstrói coleções inteiras e faz leituras por entidade. A medição evidencia trabalho de otimização; não estabelece que datasets grandes tenham custo limitado. Quantidade de queries, espera de guard, throughput concorrente, percentis e PostgreSQL continuam pendentes.

Reprodução:

```sh
go test ./packages/wos-core/storage/sqlite -run '^$' -bench BenchmarkContinuity -benchmem -benchtime=500ms
```

Fixture: `packages/wos-core/storage/sqlite/continuity_benchmark_test.go`. Saída original: `docs/audit/benchmark-sqlite.txt`.
