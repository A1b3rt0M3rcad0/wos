#!/usr/bin/env python3
"""Run the public HTTP lifecycle against an isolated, loopback-only WOS server."""

import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class Demo:
    def __init__(self, directory, origin, token, record, pace):
        self.directory, self.origin, self.token = directory, origin, token
        self.record, self.pace = record, pace
        self.started = time.monotonic()
        self.opener = urllib.request.build_opener(
            urllib.request.ProxyHandler({}), NoRedirect()
        )
        self.sequence = 0
        self.scope_path = None

    def emit(self, message):
        print(message, flush=True)
        if self.record:
            event = [round(time.monotonic() - self.started, 6), "o",
                     (message + "\n").replace("\n", "\r\n")]
            self.record.write(json.dumps(event) + "\n")
            self.record.flush()

    def stage(self, title):
        if self.pace:
            time.sleep(self.pace)
        self.emit("\n" + title)

    def request(self, method, path, payload=None, etag=None, expect=200):
        self.sequence += 1
        headers = {"Content-Type": "application/json"}
        if method != "GET":
            headers["Idempotency-Key"] = "quickstart-" + secrets.token_hex(16)
        if etag:
            headers["If-Match"] = etag
        # Save exact destination, payload, key and precondition before sending.
        # Authorization is excluded from both journal and terminal output.
        stem = self.directory / f"{self.sequence:02d}"
        intent = {"method": method, "url": self.origin + path,
                  "headers": headers, "payload": payload}
        stem.with_suffix(".request.json").write_text(json.dumps(intent, indent=2) + "\n")
        data = None if payload is None else json.dumps(payload).encode()
        request = urllib.request.Request(
            self.origin + path, data=data, method=method,
            headers={**headers, "Authorization": "Bearer " + self.token},
        )
        shown_path = path.removeprefix(self.scope_path) if self.scope_path else path
        self.emit(f"> {method} {shown_path or '/'}")
        try:
            with self.opener.open(request, timeout=5) as response:
                status = response.status
                raw = response.read().decode()
                response_etag = response.headers.get("ETag")
        except urllib.error.HTTPError as error:
            status, raw, response_etag = error.code, error.read().decode(), None
        stem.with_suffix(".response.json").write_text(
            json.dumps({"status": status, "etag": response_etag,
                        "body": json.loads(raw)}, indent=2) + "\n"
        )
        self.emit(f"< HTTP {status}")
        if status != expect:
            raise RuntimeError(f"Expected HTTP {expect}; inspect {stem}.response.json")
        return json.loads(raw), response_etag

    def show(self, **values):
        self.emit(json.dumps(values, indent=2))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", default="bin/wos", help="native wos executable")
    parser.add_argument("--keep", action="store_true", help="retain isolated data on success")
    parser.add_argument("--record", type=Path, help="write actual stdout as asciicast v2")
    parser.add_argument("--pace", type=float, default=0, help="seconds between stages (0–4)")
    args = parser.parse_args()
    if not 0 <= args.pace <= 4:
        parser.error("--pace must be between 0 and 4 seconds")
    located = shutil.which(args.binary)
    binary = Path(located or args.binary).resolve()
    if not binary.is_file():
        parser.error("native executable not found; build bin/wos or pass --binary")

    directory = Path(tempfile.mkdtemp(prefix="wos-quickstart-"))
    token = secrets.token_urlsafe(32)
    namespace = "0199d000-0000-7000-8000-000000000001"
    # Authentication prevents writes to another instance if another process wins
    # the small gap between reserving a free port and WOS binding it.
    with socket.socket() as reservation:
        reservation.bind(("127.0.0.1", 0))
        port = reservation.getsockname()[1]
    origin = f"http://127.0.0.1:{port}"
    environment = {key: value for key, value in os.environ.items()
                   if not key.startswith("WOS_")}
    environment.update({
        "WOS_LISTEN": f"127.0.0.1:{port}", "WOS_STORAGE_DRIVER": "sqlite",
        "WOS_SQLITE_PATH": str(directory / "wos.db"), "WOS_AUTH_MODE": "api_token",
        "WOS_BOOTSTRAP_TOKEN": token, "WOS_BOOTSTRAP_NAMESPACE_ID": namespace,
        "WOS_BOOTSTRAP_NAMESPACE_NAME": "Quickstart", "WOS_LOCAL_PRINCIPAL_ID": "demo-operator",
        "WOS_MCP_ENABLED": "false", "WOS_DELIVERY_WORKER_ENABLED": "false",
    })
    record = None
    process = None
    succeeded = False
    try:
        if args.record:
            record = args.record.open("x", encoding="utf-8")
            record.write(json.dumps({"version": 2, "width": 140, "height": 32,
                                     "timestamp": int(time.time()),
                                     "title": "One trusted operator: unsigned HTTP lifecycle"}) + "\n")
        with (directory / "server.log").open("w") as log:
            process = subprocess.Popen([str(binary), "server"], cwd=directory,
                                       env=environment, stdout=log, stderr=log)
        demo = Demo(directory, origin, token, record, args.pace)
        ready_opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
        deadline = time.monotonic() + 20
        while True:
            if process.poll() is not None:
                raise RuntimeError(f"server stopped; inspect {directory / 'server.log'}")
            try:
                with ready_opener.open(origin + "/readyz", timeout=0.5) as response:
                    if response.status == 200:
                        break
            except (OSError, urllib.error.URLError):
                pass
            if time.monotonic() >= deadline:
                raise RuntimeError("server readiness timed out")
            time.sleep(0.1)

        demo.emit("One trusted local operator; unsigned HTTP, not independent signed review.")
        demo.emit("A real server, isolated SQLite, no agent runtime or external service.")
        base = f"/api/v1/namespaces/{namespace}/outcomes"
        demo.stage("1. Define the Outcome and one required criterion")
        outcome, outcome_etag = demo.request("POST", base, {
            "title": "Installation instructions are ready for review",
            "desired_state": "A small installation guide includes an install command and verification step.",
            "priority": "normal",
        }, expect=201)
        outcome_path = base + "/" + outcome["value"]["id"]
        demo.scope_path = outcome_path
        demo.emit("Outcome: " + outcome["value"]["id"])
        demo.emit("Following request paths are relative to this Outcome.")
        criterion, outcome_etag = demo.request("POST", outcome_path + "/criteria", {
            "title": "The guide contains installation and verification commands",
            "required": True, "verification_mode": "evidence_review",
        }, etag=outcome_etag, expect=201)
        criterion_id = criterion["value"]["id"]
        demo.show(criterion=criterion["value"]["title"], required=criterion["value"]["required"])
        _, outcome_etag = demo.request("POST", outcome_path + "/actions/activate", {}, etag=outcome_etag)

        demo.stage("2. Create and explicitly acquire one Task")
        task, task_etag = demo.request("POST", outcome_path + "/work-items", {
            "title": "Write the installation guide", "priority": "normal", "lifecycle": "todo",
        }, expect=201)
        task_path = outcome_path + "/work-items/" + task["value"]["id"]
        task, task_etag = demo.request("POST", task_path + "/actions/claim", {
            "lease_ttl_seconds": 300,
        }, etag=task_etag)
        lease = task["value"]["current_lease"]
        demo.show(task=task["value"]["lifecycle"], fencing_token=lease["fencing_token"])

        demo.stage("3. Execute outside WOS and record the delivered file")
        delivered = directory / "installation.md"
        delivered.write_text("# Installation\n\nInstall: npm install -g wosctl@latest\n\nVerify: wos version --output json\n")
        digest = "sha256:" + hashlib.sha256(delivered.read_bytes()).hexdigest()
        artifact, _ = demo.request("POST", outcome_path + "/artifacts", {
            "artifact_type": "document", "name": "Installation guide", "uri": delivered.as_uri(),
            "media_type": "text/markdown", "checksum": digest,
        }, expect=201)
        task, _ = demo.request("POST", task_path + "/actions/complete", {
            "claim_id": lease["claim_id"], "fencing_token": lease["fencing_token"],
            "result_summary": "Wrote installation.md with installation and version commands.",
            "reason": "The local operator created and registered the file.",
        }, etag=task_etag)
        observed = delivered.read_text()
        if "npm install -g wosctl@latest" not in observed or "wos version --output json" not in observed:
            raise RuntimeError("delivered guide does not satisfy the example criterion")
        evidence, _ = demo.request("POST", outcome_path + "/evidence", {
            "evidence_type": "inspection", "description": "Read the file; both required commands are present.",
            "source_ref": {"provider": "quickstart-local-inspection", "uri": delivered.as_uri()},
            "artifact_id": artifact["value"]["id"], "checksum": digest,
            "captured_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
        }, expect=201)
        demo.show(task=task["value"]["lifecycle"], evidence=evidence["value"]["evidence_type"])

        demo.stage("4. Confirm: delivery and Evidence did not achieve the Outcome")
        pending, outcome_etag = demo.request("GET", outcome_path)
        if pending["value"]["lifecycle"] != "active":
            raise RuntimeError("expected the Outcome to remain active before assessment")
        demo.show(outcome=pending["value"]["lifecycle"])

        demo.stage("5. Explicitly assess the criterion against the inspected material")
        assessment, outcome_etag = demo.request("POST", outcome_path + "/criteria/" + criterion_id + "/assessments", {
            "criterion_revision": criterion["value"]["criterion_revision"], "result": "met",
            "rationale": "The local inspection found both commands in the registered delivery.",
            "evidence_ids": [evidence["value"]["id"]],
        }, etag=outcome_etag, expect=201)
        demo.show(assessment=assessment["value"]["result"])

        demo.stage("6. Explicitly conclude the Outcome")
        achieved, _ = demo.request("POST", outcome_path + "/actions/achieve", {
            "reason": "The required criterion has a current positive assessment of the delivery.",
        }, etag=outcome_etag)
        conclusion = achieved["value"]["conclusion"]
        if achieved["value"]["lifecycle"] != "achieved" or not conclusion["assessments"]:
            raise RuntimeError("expected an achieved Outcome with an assessment-bound Conclusion")
        demo.show(outcome=achieved["value"]["lifecycle"], conclusion_id=conclusion["id"],
                  assessment_count=len(conclusion["assessments"]))
        demo.emit("The operator wrote and inspected the file; WOS stored state and checked transition rules.")
        succeeded = True
    except (OSError, RuntimeError, ValueError, KeyError, KeyboardInterrupt) as error:
        print(f"Demo stopped: {error}\nOriginal requests, responses and SQLite retained at {directory}", file=sys.stderr)
    finally:
        if process and process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
        if record:
            record.close()
        if succeeded and not args.keep:
            shutil.rmtree(directory)
        elif succeeded:
            print(f"Isolated data retained at {directory}")
    return 0 if succeeded else 1


if __name__ == "__main__":
    sys.exit(main())
