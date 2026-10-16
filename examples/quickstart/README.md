# A complete local lifecycle

Run one small example without configuring an agent or signing keys. It creates a real Outcome, acquires one Task, writes a file outside the server, records its delivery and Evidence, assesses one required criterion, and explicitly concludes the Outcome.

**This is one trusted operator using the unsigned HTTP API.** It demonstrates the state model and explicit lease/conclusion rules. It does not demonstrate a signed WorkContract, independent reviewer identity, agent execution, or the production setup for a shared service. For those, use the [agent guide](../../docs/agents.md) and [signed workspace guide](../../docs/signed-workspace-profiles.md).

## Run it

Requirements: Python 3.9+ and a native `wos` executable that includes the server. No Python packages or external services are required. From the repository root, after [building the native executable](../../docs/operations.md#local-installation):

```sh
python3 examples/quickstart/demo.py --binary ./bin/wos
```

If `wos` is already on your PATH:

```sh
python3 examples/quickstart/demo.py --binary wos
```

The unpaced example normally finishes in a few seconds. The script starts its own server on a free loopback port with temporary SQLite storage and a random credential that it never prints. It ignores inherited `WOS_*` configuration and does not connect to your running instance, use your `.env`, or change your CLI workspace. It stops only the process it started.

## What the example actually does

| Step | Real action |
| --- | --- |
| Define | Creates the Outcome **Installation instructions are ready for review** and required criterion **The guide contains installation and verification commands**. |
| Acquire | Creates **Write the installation guide**, then explicitly claims its five-minute lease. The response includes `in_progress` and a fencing token. |
| Deliver | The Python consumer writes `installation.md`; the server does not execute the Task. It registers an Artifact reference and checksum, then completes the Task with its claim ID and fencing token. |
| Inspect | Reads the file locally, checks for both command strings, and records inspection Evidence linked to the Artifact. It does not execute npm or validate package installation. |
| Review | Reads the Outcome to prove it remains `active`, then explicitly records a `met` Assessment using that Evidence and the current criterion revision. The same trusted operator performs this demonstration review. |
| Conclude | Explicitly requests achievement and checks that the returned Conclusion references the Assessment. |

The printed HTTP statuses and JSON summaries come from the live responses. The final summary includes `"outcome": "achieved"`, `"assessment_count": 1`, and a newly generated `conclusion_id`. Completing the Task and registering Evidence alone leave the Outcome `active`.

Successful runs remove their temporary database and sample file. To inspect the exact requests, responses, server log, SQLite database, and delivered file after the server stops:

```sh
python3 examples/quickstart/demo.py --binary ./bin/wos --keep
```

## Short recording

The checked-in [24-second terminal recording](demo.cast) contains a real paced execution of this script. Play it with an asciicast-compatible player, such as `asciinema play examples/quickstart/demo.cast`. It contains only sample identifiers and sanitized response summaries. Its [capture provenance](demo.provenance.json) identifies the source revision, script, binary, and recorded bytes.

Record another run to a **new** file:

```sh
python3 examples/quickstart/demo.py --binary ./bin/wos --record /tmp/wos-demo.cast --pace 4
```

`--pace` adds up to four seconds before each of six meaningful stages; the recording timestamps preserve the actual elapsed time. Neither the recording nor the request journal includes the bootstrap credential.

## If a run stops

The script retains its isolated directory and reports its path. Each request's destination, payload, idempotency key, and version precondition are saved before sending; a confirmed response is saved separately. It does not retry mutations, change expected versions, or compensate for an uncertain result. Inspect those files before deciding how to resume. This journal is a small demonstration aid, not the native CLI's signed recovery protocol.

If the binary is missing, build it or pass its actual path with `--binary`. If server startup fails, inspect the retained `server.log`. The script chooses a free loopback port; an authentication failure after a rare port collision stops the run rather than writing to another instance. Restarting the example creates a new isolated dataset.
