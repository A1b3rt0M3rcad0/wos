"""Create real v0.2 data, upgrade/restore, and reject an incompatible old writer.

Requires explicitly built pinned historical/current servers. Only generated,
owned test databases are dropped; report contains public provenance, not tokens.
"""
import argparse
import base64
from contextlib import closing
import hashlib
import json
import os
from pathlib import Path
import secrets
import socket
import sqlite3
import subprocess
import tempfile
import time
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit, urlunsplit
from urllib.request import Request, urlopen
from uuid import uuid4

OLD_SOURCE = "b75aa3e0ca6d649cafadf8450d9b5d606367dda4"
HISTORY = {
    "contracts": "SELECT id,spec_json,spec_digest FROM work_contracts ORDER BY id",
    "submissions": "SELECT id,submission_digest,payload_json FROM work_contract_submissions ORDER BY id",
    "criteria": "SELECT criterion_id,criterion_revision,definition_json FROM criterion_revisions ORDER BY criterion_id,criterion_revision",
    "receipts": "SELECT command_name,idempotency_key,request_hash,command_id,outcome_revision,response_json FROM idempotency_records ORDER BY command_name,idempotency_key",
}


def run(args, data=None):
    return subprocess.run(args, input=data, capture_output=True, check=True, timeout=60).stdout


def free_port():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def exercise(args, driver):
    with tempfile.TemporaryDirectory(prefix="wos-v020-upgrade-") as temporary:
        directory = Path(temporary)
        token = secrets.token_urlsafe(32)
        namespace = "01a11a00-0000-7000-8000-000000000001"
        port = free_port()
        origin = f"http://127.0.0.1:{port}"
        database = directory / "wos.db"
        base = {**os.environ, "WOS_LISTEN": f"127.0.0.1:{port}", "WOS_AUTH_MODE": "api_token", "WOS_LOCAL_PRINCIPAL_ID": "historical-operator", "WOS_BOOTSTRAP_TOKEN": token, "WOS_BOOTSTRAP_NAMESPACE_ID": namespace, "WOS_BOOTSTRAP_NAMESPACE_NAME": "Actual v0.2 upgrade", "WOS_STORAGE_DRIVER": driver, "WOS_SQLITE_PATH": str(database), "WOS_MCP_ENABLED": "false", "WOS_SERVER_SIGNING_SEED_ENV": "", "WOS_DELIVERY_WORKER_ENABLED": "false"}
        owned_databases = []
        process = None
        log = open(directory / "server.log", "ab")
        originals = []
        container = os.environ.get("WOS_TEST_POSTGRES_CONTAINER", "")
        parsed = urlsplit(os.environ.get("WOS_TEST_POSTGRES_DSN", ""))
        user = parsed.username or "postgres"
        database_name = None

        def pg(*arguments, data=None):
            return run(["docker", "exec", "-i", "-e", "PGOPTIONS=-c max_parallel_workers_per_gather=0", container, *arguments], data)

        def create_database():
            name = "wos_hist_v020_" + secrets.token_hex(10)
            pg("createdb", "-U", user, name)
            owned_databases.append(name)
            return name

        def choose_database(name):
            nonlocal database_name
            database_name = name
            base["WOS_POSTGRES_DSN"] = urlunsplit((parsed.scheme, parsed.netloc, "/" + name, parsed.query, parsed.fragment))

        def request(path, body=None, key=None, expected=200, health=False):
            headers = {"Authorization": "Bearer " + token, "Content-Type": "application/json"}
            if body is not None:
                headers["Idempotency-Key"] = key or str(uuid4())
            req = Request(origin + ("" if health else "/api/v1") + path, headers=headers, data=None if body is None else json.dumps(body).encode())
            try:
                with urlopen(req, timeout=15) as response:
                    status, raw = response.status, response.read()
            except HTTPError as error:
                status, raw = error.code, error.read()
            assert status == expected, f"{driver} {path}: expected {expected}, got {status}: {raw[:500]!r}"
            return json.loads(raw) if raw else None

        def command(name, value, remember=False, expected=200):
            key = "actual-v020-" + str(uuid4())
            body = {"command": value}
            result = request("/commands/" + name, body, key, expected)
            if remember:
                originals.append((name, key, body, result))
            return result.get("value", result)

        def stop():
            nonlocal process
            if process is not None:
                if process.poll() is None:
                    process.terminate()
                process.wait(timeout=15)
                process = None

        def start(binary, signed=False, ready=True):
            nonlocal process
            environment = dict(base)
            if signed:
                environment.update(WOS_SERVER_SIGNING_SEED_ENV="WOS_HISTORICAL_ISSUER", WOS_HISTORICAL_ISSUER=base64.b64encode(bytes(range(32))).decode())
            process = subprocess.Popen([str(Path(binary).resolve()), "server"], env=environment, stdout=log, stderr=log)
            for _ in range(150):
                assert process.poll() is None, f"{driver} server exited before readiness; see owned test log"
                try:
                    request("/readyz" if ready else "/livez", health=True)
                    return
                except (URLError, AssertionError, TimeoutError):
                    time.sleep(0.1)
            raise AssertionError(f"{driver} readiness timed out")

        def rows(sql):
            if driver == "sqlite":
                with closing(sqlite3.connect(database)) as db:
                    return [list(row) for row in db.execute(sql)]
            # JSON arrays preserve the exact text stored in historical columns.
            wrapped = "SELECT COALESCE(json_agg(row_to_json(h)), '[]'::json) FROM (" + sql + ") h"
            raw = pg("psql", "-X", "-A", "-t", "-U", user, "-d", database_name, "-c", wrapped)
            return [list(row.values()) for row in json.loads(raw)]

        def history():
            return {name: rows(sql) for name, sql in HISTORY.items()}

        def preserves(expected):
            actual = history()
            for name, records in expected.items():
                for record in records:
                    assert record in actual[name], f"{driver}: historical {name} row changed"
            assert rows("SELECT COUNT(*) FROM signed_work_contracts")[0][0] == 0
            assert rows("SELECT COUNT(*) FROM signed_protocol_facts")[0][0] == 0

        def replays():
            for name, key, body, original in originals:
                replay = request("/commands/" + name, body, key)
                assert replay["idempotent_replay"] is True, f"{name}: not an original replay"
                for field in ("command_id", "outcome_revision", "value"):
                    assert replay[field] == original[field], f"{name}: historic {field} changed"

        try:
            if driver == "postgres":
                assert container and parsed.scheme in ("postgres", "postgresql"), "actual PostgreSQL fixture configuration required"
                choose_database(create_database())
            start(args.old_binary)
            outcome = command("create_outcome", {"namespace_id": namespace, "title": "Historic v0.2 obligation", "desired_state": "Original unsigned material and receipts remain readable", "priority": "normal"}, True)
            scope = {"namespace_id": namespace, "outcome_id": outcome["id"]}
            owner = {**scope, "kind": "outcome", "id": outcome["id"]}
            command("add_criterion", {"owner": owner, "expected_version": outcome["version"], "title": "Historic criterion", "required": True, "verification_mode": "attestation"}, True)
            command("activate_outcome", {"scope": scope, "expected_version": outcome["version"] + 1}, True)
            command("set_namespace_work_protocol", {"scope": scope, "expected_protocol_version": 1, "phase": "draining", "reason": "Actual v0.2 fixture drains legacy"})
            command("set_namespace_work_protocol", {"scope": scope, "expected_protocol_version": 2, "phase": "contracts_v1", "writers_drained": True, "reason": "Actual v0.2 fixture activates unsigned contracts"})
            work = command("create_work_item", {"scope": scope, "title": "Original v0.2 Task", "priority": "normal", "lifecycle": "todo", "execution_spec": {"instructions": ["Existing obligation only"]}}, True)
            command("add_criterion", {"owner": {**scope, "kind": "work_item", "id": work["id"]}, "expected_version": work["version"], "title": "Original Task criterion", "required": True, "verification_mode": "attestation"}, True)
            acquired = command("acquire_work_contract", {"scope": scope, "work_item_id": work["id"], "expected_work_item_version": work["version"] + 1, "ttl_seconds": 900}, True)
            contract = acquired["contract"]
            authority = {name: contract[name] for name in ("execution_id", "fencing_token", "spec_digest")}
            submitted = command("submit_work_result", {"scope": scope, "contract_id": contract["id"], "authority": authority, "expected_contract_version": contract["version"], "material": {"contract_id": contract["id"], "work_item_id": work["id"], "spec_digest": contract["spec_digest"], "summary": "Actual original unsigned v0.2 material", "artifacts": [], "evidence_ids": [], "criterion_evidence": []}}, True)
            path = f"/namespaces/{namespace}/outcomes/{outcome['id']}"
            historic_work = request(path + "/work-items/" + work["id"])["value"]
            old_history = history()
            assert old_history["contracts"] and old_history["submissions"] and len(old_history["criteria"]) == 2
            old_schema = rows("SELECT MAX(version) FROM schema_migrations")[0][0]
            assert old_schema == 19, "fixture did not run the actual v0.2 schema"
            stop()
            # Restore the *original* database into a clean target before upgrade.
            if driver == "sqlite":
                backup = directory / "original.db"
                with closing(sqlite3.connect(database)) as source, closing(sqlite3.connect(backup)) as target:
                    source.backup(target)
                restored = directory / "restored.db"
                with closing(sqlite3.connect(backup)) as source, closing(sqlite3.connect(restored)) as target:
                    source.backup(target)
                database = restored
                base["WOS_SQLITE_PATH"] = str(restored)
                backup_digest = hashlib.sha256(backup.read_bytes()).hexdigest()
            else:
                backup = pg("pg_dump", "-U", user, "-d", database_name, "--format=custom", "--no-owner", "--no-acl")
                backup_digest = hashlib.sha256(backup).hexdigest()
                choose_database(create_database())
                pg("pg_restore", "-U", user, "-d", database_name, "--single-transaction", "--no-owner", "--no-acl", data=backup)
            start(args.binary, signed=True)
            preserves(old_history)
            assert request(path + "/work-items/" + work["id"])["value"] == historic_work
            replays()
            identity = request("/security/signing-identity")
            request("/security/signing-commands", {"namespace_id": namespace, "expected_namespace_version": identity["namespace_version"], "operation": "set_acceptance_policy", "acceptance_policy": {"namespace_id": namespace, "acceptance_floor": "independent_review", "max_active_work_contracts": 3, "max_active_review_contracts": 1}})
            command("set_namespace_work_protocol", {"scope": scope, "expected_protocol_version": 3, "phase": "draining_to_signed_v2", "reason": "Actual upgrade drains old contract"})
            command("set_namespace_work_protocol", {"scope": scope, "expected_protocol_version": 4, "phase": "signed_contracts_v2", "writers_drained": True, "reason": "Premature activation must fail"}, expected=409)
            command("revoke_work_contract", {"scope": scope, "contract_id": contract["id"], "expected_contract_version": submitted["contract"]["version"], "reason": "Explicitly retire original v0.2 authority"})
            command("set_namespace_work_protocol", {"scope": scope, "expected_protocol_version": 4, "phase": "signed_contracts_v2", "writers_drained": True, "reason": "Validated historical restore and retired old writer"})
            preserves(old_history)
            replays()
            final_history = history()
            before_work = request(path + "/work-items/" + work["id"])["value"]
            stop()
            # v0.2 lacks the new startup refusal: it may bind, but is not ready
            # and its protocol parser rejects incompatible mutations.
            start(args.old_binary, ready=False)
            request("/readyz", expected=503, health=True)
            rejected = command("acquire_work_contract", {"scope": scope, "work_item_id": work["id"], "expected_work_item_version": before_work["version"], "ttl_seconds": 300}, expected=422)
            assert rejected.get("error", {}).get("code") == "invalid_argument" and "unknown work protocol" in rejected["error"]["message"], "old writer refusal was unrelated to compatibility"
            assert history() == final_history, "rejected old writer changed committed history"
            stop()
            start(args.binary, signed=True)
            preserves(old_history)
            assert request(path + "/work-items/" + work["id"])["value"] == before_work
            replays()
            return {"driver": driver, "old_schema": old_schema, "current_schema": rows("SELECT MAX(version) FROM schema_migrations")[0][0], "backup_sha256": backup_digest, "historical_receipts_replayed": len(originals), "contracts_preserved": len(old_history["contracts"]), "submissions_preserved": len(old_history["submissions"]), "criteria_preserved": len(old_history["criteria"]), "unsigned_history_not_resigned": True, "active_v1_cutover_refused": True, "old_binary_may_bind": True, "old_binary_ready_status": 503, "incompatible_old_mutation_refused": True}
        finally:
            stop()
            log.close()
            for name in reversed(owned_databases):
                pg("dropdb", "-U", user, "--force", name)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", required=True)
    parser.add_argument("--old-binary", required=True)
    parser.add_argument("--report", required=True)
    parser.add_argument("--source", required=True)
    parser.add_argument("--driver", choices=("sqlite", "postgres", "both"), default="both")
    args = parser.parse_args()
    old = run([str(Path(args.old_binary).resolve()), "version"]).decode().strip()
    assert f"0.2.0 (commit {OLD_SOURCE}," in old, "historical binary must declare pinned actual source"
    assert len(args.source) == 40 and all(c in "0123456789abcdef" for c in args.source), "exact source required"
    current_build = run([str(Path(args.binary).resolve()), "version"]).decode().strip()
    assert f"(commit {args.source}," in current_build, "current binary source metadata differs"
    results = [exercise(args, driver) for driver in (["sqlite", "postgres"] if args.driver == "both" else [args.driver])]
    report = {"schema": 1, "historical_source": OLD_SOURCE, "historical_binary_sha256": hashlib.sha256(Path(args.old_binary).read_bytes()).hexdigest(), "current_binary_sha256": hashlib.sha256(Path(args.binary).read_bytes()).hexdigest(), "current_source": args.source, "current_build": current_build, "results": results, "native_windows_executed": False, "signed_pending_restore_executed": False}
    Path(args.report).write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    main()
