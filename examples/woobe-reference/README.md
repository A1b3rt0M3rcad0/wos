# Woobe reference consumer

This reference executes the audit's same-Outcome 18-step journey against the standalone WOS server, using Woobe's actual `DiscoverMcpToolsUseCase`, `HttpToolExecutor` and `McpToolRuntime`. It does not substitute a fake ToolExecutionPort, fabricate model responses, or import Woobe into WOS production code. The actors are deterministic authenticated fixtures; this tests client interoperability rather than a complete deployed Woobe Agent/Network environment.

The baseline Woobe source is commit `42e6667008db4e1306c05714af7a0df5b3ce7dd2`. `source-manifest.json` pins the Git blob hash of every imported Woobe module. Use `packages/woobe-core/src` from that checkout and install its Python dependencies (the local check used Python 3.12, httpx 0.28.1, uuid6 2025.0.1 and Pydantic 2).

The baseline redaction policy removes the public integer `fencing_token` from MCP results, preventing valid work completion. The focused fix is proposed in [Woobe PR #178](https://github.com/A1b3rt0M3rcad0/woobe/pull/178), commit `9161eecc0afac5a4e2736bf71ddedba28d781b45`. It preserves only a non-negative integer under that exact name; credentials, strings, booleans, containers and sensitive parents retain redaction. Three focused security regressions passed. The fix is a draft and is not integrated in Woobe master.

For the proposed compatibility profile, use the source directory at that fix commit:

```bash
go build -trimpath -o bin/wos ./packages/wos-api/cmd/wos
python3 tests/acceptance/journey.py --client woobe \
  --woobe-core-source /absolute/path/to/woobe/packages/woobe-core/src \
  --woobe-fencing-fix
```

The flag selects the exact alternative redaction blob hash, not a runtime monkey patch. Other production modules must retain their baseline hashes. Discovery defaults to `review`; the test checks that policy blocks execution, then explicitly allows the fixture's discovered tools. `allow_local_networks=True` is scoped to the ephemeral loopback fixture; redirect and DNS/IP validation still run. The injected httpx client disables environment proxies for this local fixture.

Without Woobe, run `python3 tests/acceptance/journey.py --client http`. That independent consumer uses real JSON-RPC initialize/initialized/tools/call and the same assertions. Both profiles create a fresh temporary installation, real HTTP process and signed webhook consumer with SQLite deduplication. A real 30-second lease expiry is required. The harness always shuts down processes and removes temporary fixture files.

Full Woobe backend/security validation and integration of PR #178 remain required before declaring deployed Woobe compatibility. The test keeps MCP `isError` distinct from HTTP transport success; consumers must preserve that distinction when interpreting tool results.
