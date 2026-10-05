"""Run the audit's 18-step journey against a real standalone SQLite process.

The default client is an independent JSON-RPC consumer. --client woobe uses
unmodified production discovery/HTTP/MCP modules from an explicit Woobe source
directory. Identities are deterministic test actors; no LLM output is simulated.
"""
import argparse
import asyncio
import datetime as dt
import hashlib
import hmac
import json
import os
from pathlib import Path
import secrets
import socket
import sqlite3
import subprocess
import sys
import tempfile
import threading
import time
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.error import HTTPError
from urllib.request import Request, urlopen
from uuid import uuid4


def now():
    return dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z")


def free_port():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


async def journey(args):
    binary = str(Path(args.binary).resolve())
    with tempfile.TemporaryDirectory(prefix="wos-acceptance-") as directory:
        directory = Path(directory)
        ns = "0199d200-0000-7000-8000-000000000001"
        endpoint = "0199d200-0000-7000-8000-000000000002"
        bootstrap, secret = secrets.token_urlsafe(32), secrets.token_urlsafe(32)
        consumer_file = directory / "consumer.db"
        with sqlite3.connect(consumer_file) as db:
            db.execute("CREATE TABLE effects(id TEXT PRIMARY KEY, body BLOB)")
            db.execute("CREATE TABLE attempts(id TEXT)")

        class Consumer(BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def do_POST(self):
                body = self.rfile.read(int(self.headers["Content-Length"]))
                event_id, timestamp = self.headers["X-WOS-Event-ID"], self.headers["X-WOS-Timestamp"]
                expected = "sha256=" + hmac.new(secret.encode(), timestamp.encode() + b"." + event_id.encode() + b"." + body, hashlib.sha256).hexdigest()
                if not hmac.compare_digest(expected, self.headers["X-WOS-Signature"]) or abs(time.time() - int(timestamp)) > 300:
                    self.send_error(401)
                    return
                with sqlite3.connect(consumer_file) as db:
                    db.execute("BEGIN IMMEDIATE")
                    previous = db.execute("SELECT body FROM effects WHERE id=?", (event_id,)).fetchone()
                    if previous is not None and previous[0] != body:
                        self.send_error(409)
                        return
                    db.execute("INSERT OR IGNORE INTO effects VALUES (?,?)", (event_id, body))
                    db.execute("INSERT INTO attempts VALUES (?)", (event_id,))
                self.send_response(200)
                self.end_headers()

        consumer = HTTPServer(("127.0.0.1", 0), Consumer)
        thread = threading.Thread(target=consumer.serve_forever, daemon=True)
        thread.start()
        port = free_port()
        url = f"http://127.0.0.1:{port}"
        env = {**os.environ, "WOS_LISTEN": f"127.0.0.1:{port}", "WOS_STORAGE_DRIVER": "sqlite", "WOS_SQLITE_PATH": str(directory / "wos.db"),
               "WOS_AUTH_MODE": "api_token", "WOS_BOOTSTRAP_TOKEN": bootstrap, "WOS_BOOTSTRAP_NAMESPACE_ID": ns, "WOS_BOOTSTRAP_NAMESPACE_NAME": "Full acceptance",
               "WOS_LOCAL_PRINCIPAL_ID": "human", "WOS_MCP_ENABLED": "true", "WOS_DELIVERY_WORKER_ENABLED": "true", "WOS_WEBHOOK_ALLOW_LOOPBACK": "true",
               "WOS_WEBHOOK_ENDPOINTS": json.dumps([{"id": endpoint, "namespace_id": ns, "url": f"http://127.0.0.1:{consumer.server_port}/signals", "secret_ref": "acceptance-key", "key_id": "v1"}]),
               "WOS_WEBHOOK_SECRETS": json.dumps({"acceptance-key": secret})}
        process = None
        log = open(directory / "server.log", "ab")

        def request(path, token=bootstrap, body=None, prefix="/api/v1", expected=200):
            headers = {"Authorization": "Bearer " + token, "Content-Type": "application/json", "Accept": "application/json, text/event-stream", "MCP-Protocol-Version": "2025-06-18"}
            if body is not None:
                headers["Idempotency-Key"] = str(uuid4())
            req = Request(url + prefix + path, headers=headers, data=None if body is None else json.dumps(body).encode())
            try:
                with urlopen(req, timeout=15) as response:
                    status, raw = response.status, response.read()
            except HTTPError as error:
                status, raw = error.code, error.read()
            assert status == expected, f"{path}: expected {expected}, got {status}: {raw[:800]!r}"
            return json.loads(raw) if raw else None

        async def start():
            nonlocal process
            process = subprocess.Popen([binary, "server"], env=env, stdout=subprocess.DEVNULL, stderr=log)
            for _ in range(100):
                try:
                    request("/readyz", prefix="")
                    return
                except (OSError, AssertionError):
                    if process.poll() is not None:
                        raise RuntimeError("standalone process exited before readiness")
                await asyncio.sleep(0.05)
            raise RuntimeError("standalone readiness timeout")

        def stop():
            nonlocal process
            if process and process.poll() is None:
                process.terminate()
                process.wait(timeout=15)
            process = None

        def command(name, value):
            return request("/commands/" + name, body={"command": value})["value"]

        def rpc(token, method, params):
            value = request("/mcp", token, {"jsonrpc": "2.0", "id": str(uuid4()), "method": method, "params": params}, prefix="")
            assert "error" not in value, value
            return value["result"]

        def step(number, description):
            print(f"J{number:02d} passed: {description}", flush=True)

        def independent_state(token):
            reader = Path(__file__).with_name("read_state.py")
            output = subprocess.check_output([sys.executable, str(reader)], env={**os.environ, "WOS_READER_URL": url + "/api/v1", "WOS_READER_NAMESPACE": ns, "WOS_READER_TOKEN": token}, timeout=20)
            return json.loads(output)

        executor = None
        http_client = None
        try:
            await start()
            issued = []
            namespace_version = 1
            for principal in ("agent-a", "agent-b"):
                grant = request("/security/commands", body={"namespace_id": ns, "expected_namespace_version": namespace_version, "operation": "set_grant", "principal_id": principal, "permissions": ["state:read", "work:write", "records:write"]})
                namespace_version = grant["result"]["namespace_version"]
                credential = request("/security/commands", body={"namespace_id": ns, "expected_namespace_version": namespace_version, "operation": "issue_credential", "principal_id": principal, "actor_ref": {"kind": "agent", "provider": "acceptance", "id": principal}, "expires_at": (dt.datetime.now(dt.timezone.utc) + dt.timedelta(hours=1)).isoformat().replace("+00:00", "Z")})
                namespace_version = credential["result"]["namespace_version"]
                issued.append(credential["token"])

            woobe_configs = {}
            if args.client == "woobe":
                source = Path(args.woobe_core_source).resolve()
                manifest = json.loads((Path(__file__).parents[2] / "examples/woobe-reference/source-manifest.json").read_text())
                if args.woobe_fencing_fix:
                    for entry in manifest:
                        if entry["path"].endswith("shared/security/redaction.py"):
                            entry["sha"] = "16cc8935f34758a4a06c805d47d580329a26dd32"
                for entry in manifest:
                    data = (source / entry["path"].split("/src/", 1)[1]).read_bytes()
                    digest = hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest()
                    assert digest == entry["sha"], "Woobe source differs from the pinned commit: " + entry["path"]
                sys.path.insert(0, str(source))
                from modules.agent.application.mcp_discovery import DiscoverMcpToolsUseCase
                from modules.agent.application.dtos.mcp_discovery_dtos import McpDiscoverInput
                from modules.agent.infra.adapters.tool_runtime import McpToolRuntime
                from modules.agent.infra.adapters.tool_executor import HttpToolExecutor
                import httpx
                http_client = httpx.AsyncClient(trust_env=False, follow_redirects=False)
                executor = HttpToolExecutor(client=http_client, allow_local_networks=True)
                runtime = McpToolRuntime(executor)
                for token in issued:
                    discovered = await DiscoverMcpToolsUseCase(executor).execute(McpDiscoverInput(server_url=url + "/mcp", headers={"Authorization": "Bearer " + token}))
                    assert discovered.initialized and not discovered.error and len(discovered.tools) >= 96
                    config = {"server": {"url": url + "/mcp", "transport": "streamable_http"}, "server_headers": [{"key": "Authorization", "value": "Bearer " + token}], "discovered_tools": [t.model_dump() for t in discovered.tools]}
                    reviewed = await runtime.test(config, {"__mcp_tool_name": "wos_search_outcomes", "namespace_id": ns})
                    assert not reviewed.success and reviewed.output["error_code"] == "MCP_TOOL_REQUIRES_POLICY"
                    for tool in config["discovered_tools"]:
                        tool["permission"] = {"mode": "allow"}
                    woobe_configs[token] = config
                print("Woobe production source hashes verified; discovery and review policy passed", flush=True)

            async def tool(token, name, arguments, failure=False):
                if args.client == "woobe":
                    result = await runtime.test(woobe_configs[token], {"__mcp_tool_name": "wos_" + name, **arguments})
                    assert result.success, result.error
                    payload = result.output["mcp"]
                else:
                    rpc(token, "initialize", {"protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "wos-audit-independent", "version": "1"}})
                    request("/mcp", token, {"jsonrpc": "2.0", "method": "notifications/initialized", "params": {}}, prefix="", expected=202)
                    payload = rpc(token, "tools/call", {"name": "wos_" + name, "arguments": arguments})
                assert bool(payload.get("isError")) == failure, payload
                return payload if failure else payload["structuredContent"]

            async def agent_command(token, name, value, failure=False):
                result = await tool(token, name, {"idempotency_key": str(uuid4()), "command": value}, failure)
                return result if failure else result["value"]

            root = command("create_outcome", {"namespace_id": ns, "title": "Full shared acceptance", "desired_state": "A verified delivery can be continued independently", "priority": "normal"})
            scope = {"namespace_id": ns, "outcome_id": root["id"]}
            path = f"/namespaces/{ns}/outcomes/{root['id']}"
            ref = lambda kind, entity_id: {**scope, "kind": kind, "id": entity_id}
            outcome_ref = ref("outcome", root["id"])
            criterion = command("add_criterion", {"owner": outcome_ref, "expected_version": root["version"], "title": "Observed contract assertions reviewed", "required": True, "verification_mode": "evidence_review"})
            root = request(path)["value"]
            root = command("activate_outcome", {"scope": scope, "expected_version": root["version"]})
            step(1, "authenticated human creates and activates Outcome with required criterion")
            objective = command("create_objective", {"scope": scope, "title": "Deliver verified continuation", "priority": "normal", "required_for_outcome": True})
            objective_ref = ref("objective", objective["id"])
            obj_criterion = command("add_criterion", {"owner": objective_ref, "expected_version": objective["version"], "title": "Review actual assertion report", "required": True, "verification_mode": "evidence_review"})
            objective = request(path + "/objectives/" + objective["id"])["value"]
            objective = command("start_objective", {"scope": scope, "objective_id": objective["id"], "expected_version": objective["version"]})
            command("configure_trigger", {"scope": scope, "name": "Acceptance delivery", "event_types": ["work_item.created"], "predicate": {}, "target_endpoint_ids": [endpoint], "signal_type": "acceptance.work_available"})
            work = command("create_work_item", {"scope": scope, "title": "Execute assertions", "priority": "normal", "lifecycle": "todo", "objective_id": objective["id"]})
            work_ref = ref("work_item", work["id"])
            plan = command("create_roadmap", {"scope": scope, "plan_scope": {"kind": "outcome", "id": root["id"]}, "title": "Shared plan"})

            def publish(plan, label, base=None):
                opened = command("open_roadmap_draft", {"scope": scope, "roadmap_id": plan["id"], "expected_version": plan["version"], **({"base_revision_number": base} if base else {})})
                replaced = command("replace_roadmap_draft", {"scope": scope, "roadmap_id": plan["id"], "expected_version": opened["version"], "expected_draft_version": opened["draft"]["draft_version"], "nodes": [{"node_key": "delivery", "node_type": "reference", "target_ref": work_ref, "title": label, "position": 0}], "after_links": []})
                published = command("publish_roadmap_draft", {"scope": scope, "roadmap_id": plan["id"], "expected_version": replaced["version"], "expected_draft_version": replaced["draft"]["draft_version"]})
                command("activate_roadmap_revision", {"scope": scope, "roadmap_id": plan["id"], "expected_version": published["version"], "revision_number": len(published["revisions"])})
                return published

            plan = publish(plan, "Initial delivery")
            plan = request(path + "/roadmaps/" + plan["id"])["value"]
            first_revision = json.dumps(plan["revisions"][0], sort_keys=True)
            step(2, "required Objective, WorkItem and immutable initial plan published")
            found = await tool(issued[0], "search_outcomes", {"namespace_id": ns, "text": "Full shared"})
            assert found["items"][0]["id"] == root["id"]
            snapshot = await tool(issued[0], "get_continuity", {**scope, "limit": 25})
            assert snapshot["outcome"]["ref"]["id"] == root["id"]
            step(3, "agent discovers context through authorized MCP without chat history")
            ready = await tool(issued[0], "list_ready_work", scope)
            assert ready["items"][0]["work_item"]["ref"]["id"] == work["id"]
            claimed = await agent_command(issued[0], "claim_work_item", {"scope": scope, "work_item_id": work["id"], "expected_version": work["version"], "ttl_seconds": 30})
            assert type(claimed["current_lease"]["fencing_token"]) is int, "consumer redacted the public fencing_token: " + repr(claimed["current_lease"]["fencing_token"])
            step(4, "agent queries ready candidates and explicitly claims work")
            decision = command("propose_decision", {"scope": scope, "title": "Use durable state", "proposal": "Use WOS, not chat history", "chosen_alternative": "WOS", "alternatives": ["WOS", "chat"], "rationale": "Independent continuation is required"})
            command("accept_decision", {"scope": scope, "decision_id": decision["id"], "expected_version": decision["version"]})
            issue = command("create_issue", {"scope": scope, "title": "Review execution before proceeding", "severity": "minor", "affected_refs": [work_ref]})
            blocker = command("create_blocker", {"scope": scope, "blocked_ref": work_ref, "cause_ref": ref("issue", issue["id"]), "description": "Require explicit release", "propagation": "direct"})
            context = await tool(issued[0], "get_work_context", {**scope, "entity_id": work["id"]})
            assert "Use durable state" in json.dumps(context) and "Require explicit release" in json.dumps(context)
            step(5, "accepted Decision and active Issue/Blocker appear in agent context")
            command("resolve_issue", {"scope": scope, "issue_id": issue["id"], "expected_version": issue["version"], "resolution_summary": "Review completed"})
            operational = request(path + "/work-items/" + work["id"] + "/operational-state", issued[0])
            assert operational["state"]["is_blocked"]
            premature = {"scope": scope, "work_item_id": work["id"], "expected_version": claimed["version"], "claim_id": claimed["current_lease"]["claim_id"], "fencing_token": claimed["current_lease"]["fencing_token"], "result_summary": "Premature", "reason": "Must be rejected while blocked"}
            await agent_command(issued[0], "complete_work_item", premature, failure=True)
            command("resolve_blocker", {"scope": scope, "blocker_id": blocker["id"], "expected_version": blocker["version"], "resolution_summary": "Explicit release after review"})
            step(6, "Issue resolution keeps Blocker; completion rejected until explicit release")
            expiry = dt.datetime.fromisoformat(claimed["current_lease"]["expires_at"].replace("Z", "+00:00"))
            await asyncio.sleep(max(0, (expiry - dt.datetime.now(dt.timezone.utc)).total_seconds()) + 0.2)
            reclaimed = await agent_command(issued[1], "reclaim_work_item", {"scope": scope, "work_item_id": work["id"], "expected_version": claimed["version"], "ttl_seconds": 900})
            assert isinstance(reclaimed["current_lease"]["fencing_token"], int), "consumer redacted the public fencing_token: " + repr(reclaimed["current_lease"]["fencing_token"])
            assert reclaimed["current_lease"]["fencing_token"] > claimed["current_lease"]["fencing_token"]
            step(7, "real lease expiry and reclaim by a second authenticated agent")
            premature["expected_version"] = reclaimed["version"]
            await agent_command(issued[0], "complete_work_item", premature, failure=True)
            step(8, "old executor rejected with stale claim/fencing")
            report = json.dumps({"ready_claimed": True, "blocker_independent": True, "second_agent_reclaimed": True, "stale_executor_rejected": True}, sort_keys=True).encode()
            artifact_file = directory / "observed-report.json"
            artifact_file.write_bytes(report)
            checksum = hashlib.sha256(report).hexdigest()
            artifact = await agent_command(issued[1], "register_artifact", {"scope": scope, "artifact_type": "test_report", "name": "Observed acceptance assertions", "uri": artifact_file.as_uri(), "media_type": "application/json", "checksum": checksum})
            evidence = await agent_command(issued[1], "register_evidence", {"scope": scope, "evidence_type": "test_result", "description": "Actual assertions executed by this independent consumer", "source_ref": {"provider": "acceptance-harness", "uri": artifact_file.as_uri()}, "captured_at": now(), "artifact_id": artifact["id"], "checksum": checksum})
            for owner, requirement in ((outcome_ref, criterion), (objective_ref, obj_criterion)):
                await agent_command(issued[1], "create_evidence_link", {"scope": scope, "evidence_id": evidence["id"], "target_ref": owner, "criterion_id": requirement["id"], "stance": "supports", "rationale": "Inspect the recorded assertion report"})
            lease = reclaimed["current_lease"]
            await agent_command(issued[1], "complete_work_item", {"scope": scope, "work_item_id": work["id"], "expected_version": reclaimed["version"], "claim_id": lease["claim_id"], "fencing_token": lease["fencing_token"], "result_summary": "Assertions passed and report registered", "reason": "Observed and persisted execution"})
            assert request(path)["value"]["lifecycle"] == "active"
            step(9, "second agent registers real Artifact/Evidence and completes work without certifying Outcome")
            for owner, requirement, read_path in ((outcome_ref, criterion, path), (objective_ref, obj_criterion, path + "/objectives/" + objective["id"])):
                current = request(read_path)["value"]
                command("record_criterion_assessment", {"owner": owner, "expected_version": current["version"], "criterion_id": requirement["id"], "criterion_revision": requirement["criterion_revision"], "result": "met", "rationale": "Human fixture reviews the actual persisted assertion report", "evidence_ids": [evidence["id"]]})
            step(10, "authorized human separately reviews required criteria with Evidence")
            objective = request(path + "/objectives/" + objective["id"])["value"]
            objective = command("achieve_objective", {"scope": scope, "objective_id": objective["id"], "expected_version": objective["version"], "reason": "Required proof reviewed"})
            root = request(path)["value"]
            root = command("achieve_outcome", {"scope": scope, "expected_version": root["version"], "reason": "Required Objective and proof certified"})
            root = request(path)["value"]
            original_conclusion = json.dumps(root["conclusion"], sort_keys=True)
            assert root["conclusion"]["assessments"] and objective["conclusion"]["assessments"]
            step(11, "Objective and Outcome explicitly certified with proof snapshots")
            plan = request(path + "/roadmaps/" + plan["id"])["value"]
            plan = publish(plan, "Reviewed delivery", base=1)
            plan = request(path + "/roadmaps/" + plan["id"])["value"]
            assert len(plan["revisions"]) == 2 and json.dumps(plan["revisions"][0], sort_keys=True) == first_revision
            step(12, "revision 2 published; revision 1 remains readable and immutable")
            command("retract_evidence", {"scope": scope, "evidence_id": evidence["id"], "expected_version": evidence["version"], "reason": "Test a later evidence validity change"})
            state = request(path + "/state")
            assert state["conclusion_contested"] and state["outcome"]["lifecycle"] == "achieved"
            assert json.dumps(state["outcome"]["conclusion"], sort_keys=True) == original_conclusion
            step(13, "Evidence retraction contests the conclusion without implicit reopen or history rewrite")
            stop()
            await start()
            independent = independent_state(issued[1])
            assert independent["outcome"]["id"] == root["id"] and independent["conclusion_contested"] and len(independent["evidence"]) == 1
            step(14, "new HTTP consumer reads same state/history after standalone restart")
            backup, restored = directory / "backup.db", directory / "restored.db"
            subprocess.run([binary, "db", "backup", str(backup)], env={**env, "WOS_DELIVERY_WORKER_ENABLED": "false"}, check=True, stdout=subprocess.DEVNULL, stderr=log, timeout=30)
            stop()
            subprocess.run([binary, "db", "restore", str(backup), str(restored)], check=True, stdout=subprocess.DEVNULL, stderr=log, timeout=15)
            env["WOS_SQLITE_PATH"] = str(restored)
            await start()
            resumed = independent_state(issued[1])
            assert json.dumps(resumed["outcome"]["conclusion"], sort_keys=True) == original_conclusion
            assert resumed["conclusion_contested"] and len(resumed["active_roadmaps"]) == 1
            step(15, "consistent backup restored to a clean file; new consumer continues history and plan")
            deliveries = request(path + "/deliveries")["items"]
            assert len(deliveries) == 1
            delivery_id = deliveries[0]["id"]
            command("redeliver_delivery", {"scope": scope, "expected_version": resumed["outcome"]["version"], "delivery_id": delivery_id, "reason": "Verify durable consumer deduplication"})
            for _ in range(100):
                with sqlite3.connect(consumer_file) as db:
                    count = db.execute("SELECT count(*) FROM attempts").fetchone()[0]
                    effects = db.execute("SELECT count(*) FROM effects").fetchone()[0]
                if count >= 2:
                    break
                await asyncio.sleep(0.05)
            assert count == 2 and effects == 1
            step(16, "signed external signal delivered twice; durable consumer commits one effect")
            current = request(path)["value"]
            archived = command("archive_outcome", {"scope": scope, "expected_version": current["version"]})
            assert request(path)["value"]["archived_at"]
            unarchived = command("unarchive_outcome", {"scope": scope, "expected_version": archived["version"], "reason": "Authorized return from archive"})
            assert unarchived.get("archived_at") is None and unarchived["lifecycle"] == "achieved"
            step(17, "archive remains queryable; explicit unarchive preserves achieved lifecycle")
            other = request("/security/commands", body={"namespace_id": ns, "expected_namespace_version": namespace_version, "operation": "create_namespace", "namespace_name": "Isolated consumer"})
            other_ns, other_token = other["result"]["created_namespace"]["id"], other["token"]
            command("create_outcome", {"namespace_id": ns, "title": "Pagination witness", "desired_state": "Produce a bounded cursor", "priority": "normal"})
            page = request(f"/namespaces/{ns}/outcomes?limit=1")
            assert page["next_cursor"]
            request(path, other_token, expected=403)
            request(f"/namespaces/{other_ns}/outcomes/{root['id']}", other_token, expected=404)
            assert request(f"/namespaces/{other_ns}/outcomes?text=Full", other_token)["items"] == []
            request(f"/namespaces/{other_ns}/outcomes?limit=1&cursor=" + page["next_cursor"], other_token, expected=422)
            denied = request("/mcp", other_token, {"jsonrpc": "2.0", "id": "foreign-resource", "method": "resources/read", "params": {"uri": f"wos://namespaces/{ns}/outcomes/{root['id']}/continuity"}}, prefix="")
            assert "error" in denied
            step(18, "foreign Namespace denied by ID, discovery, bound cursor and MCP resource")
            print(f"PASS: all 18 steps, client={args.client}, same Outcome, real standalone process, signed delivery and clean restore", flush=True)
        finally:
            if executor:
                await executor.aclose()
            if http_client:
                await http_client.aclose()
            stop()
            consumer.shutdown()
            consumer.server_close()
            thread.join(timeout=5)
            log.close()


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", default="bin/wos")
    parser.add_argument("--client", choices=["http", "woobe"], default="http")
    parser.add_argument("--woobe-core-source", default="")
    parser.add_argument("--woobe-fencing-fix", action="store_true", help="explicitly use the proposed public integer fencing redaction fix")
    arguments = parser.parse_args()
    if arguments.client == "woobe" and not arguments.woobe_core_source:
        parser.error("--woobe-core-source is required for the real Woobe client")
    asyncio.run(journey(arguments))
