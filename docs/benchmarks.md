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
