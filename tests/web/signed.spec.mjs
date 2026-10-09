import { test, expect } from "playwright/test";
import assert from "node:assert/strict";
import { spawn, execFileSync } from "node:child_process";
import {
  mkdtemp,
  rm,
  mkdir,
  writeFile,
  readFile,
  readdir,
} from "node:fs/promises";
import { tmpdir } from "node:os";
import { resolve, join } from "node:path";
import { createServer } from "node:net";
import { once } from "node:events";
import {
  randomUUID,
  randomBytes,
  createPrivateKey,
  createPublicKey,
  createHash,
} from "node:crypto";

for (const decision of ["approved", "changes_requested"])
  test(`signed work shows accepted delivery, independent review and protected profile actions: ${decision}`, async ({
    browser,
  }) => {
    const directory = await mkdtemp(join(tmpdir(), "wos-signed-ui-"));
    const socket = createServer();
    socket.listen(0, "127.0.0.1");
    await once(socket, "listening");
    const port = socket.address().port;
    await new Promise((done) => socket.close(done));
    const url = `http://127.0.0.1:${port}`,
      namespace = "01a11991-0000-7000-8000-000000000001",
      token = randomBytes(32).toString("base64url");
    const issuerSeed = randomBytes(32).toString("base64");
    let logs = "";
    const child = spawn(resolve("../../bin/wos"), ["server"], {
      env: {
        ...process.env,
        WOS_LISTEN: `127.0.0.1:${port}`,
        WOS_SQLITE_PATH: join(directory, "wos.db"),
        WOS_AUTH_MODE: "api_token",
        WOS_BOOTSTRAP_TOKEN: token,
        WOS_BOOTSTRAP_NAMESPACE_ID: namespace,
        WOS_BOOTSTRAP_NAMESPACE_NAME: "Signed workspace",
        WOS_LOCAL_PRINCIPAL_ID: "human",
        WOS_SERVER_SIGNING_SEED_ENV: "WOS_UI_ISSUER_SEED",
        WOS_UI_ISSUER_SEED: issuerSeed,
        WOS_SIGNED_ACCEPTANCE_FLOOR: "independent_review",
      },
      stdio: ["ignore", "ignore", "pipe"],
    });
    child.stderr.on("data", (b) => (logs = (logs + b.toString()).slice(-8000)));
    const context = await browser.newContext(),
      page = await context.newPage(),
      errors = [],
      mutations = [];
    page.on("pageerror", (e) => errors.push(e.message));
    page.on("request", (r) => {
      if (r.method() === "POST" && r.url().includes("/commands/"))
        mutations.push(r.url());
    });
    async function api(path, credential, body) {
      const response = await fetch(url + "/api/v1" + path, {
        method: body ? "POST" : "GET",
        headers: {
          Authorization: `Bearer ${credential}`,
          "Content-Type": "application/json",
          ...(body ? { "Idempotency-Key": randomUUID() } : {}),
        },
        ...(body ? { body: JSON.stringify(body) } : {}),
      });
      const data = await response.json();
      assert.ok(response.ok, JSON.stringify(data));
      return data;
    }
    const command = async (name, value) =>
      (await api("/commands/" + name, token, { command: value })).value;
    const signing = async (value) =>
      api("/security/signing-commands", token, value);
    const cli = (workspace, profile, environment, ...args) =>
      JSON.parse(
        execFileSync(
          resolve("../../bin/wosctl"),
          [
            ...args,
            "--workspace",
            workspace,
            "--profile",
            profile,
            "--output",
            "json",
          ],
          {
            env: { ...process.env, ...environment },
            encoding: "utf8",
            timeout: 20000,
          },
        ),
      );
    async function editDraft(path, section, changes) {
      const original = await readFile(path, "utf8"),
        lines = original.split("\n");
      const start = lines.indexOf(section + ":");
      assert.ok(start >= 0, "editable draft exists");
      let end = start + 1;
      while (end < lines.length && (lines[end].startsWith(" ") || !lines[end]))
        end++;
      const editable = changes(lines.slice(start + 1, end));
      const result = [
        ...lines.slice(0, start + 1),
        ...editable,
        ...lines.slice(end),
      ].join("\n");
      assert.equal(
        result.slice(0, result.indexOf("\n" + section + ":")),
        original.slice(0, original.indexOf("\n" + section + ":")),
        "issued and local sections remain exact",
      );
      await writeFile(path, result);
    }
    try {
      let ready = false;
      for (let i = 0; i < 100; i++) {
        try {
          if ((await fetch(url + "/readyz")).ok) {
            ready = true;
            break;
          }
        } catch {}
        await new Promise((done) => setTimeout(done, 50));
      }
      assert.ok(ready, logs);
      const outcome = await command("create_outcome", {
          namespace_id: namespace,
          title: "Entrega assinada e revisada",
          desired_state: "Execução e revisão distintas",
          priority: "normal",
        }),
        scope = { namespace_id: namespace, outcome_id: outcome.id };
      await command("add_criterion", {
        owner: { ...scope, kind: "outcome", id: outcome.id },
        expected_version: outcome.version,
        title: "Resultado validado",
        required: true,
        verification_mode: "attestation",
      });
      await command("activate_outcome", {
        scope,
        expected_version: outcome.version + 1,
      });
      let identity = await api("/security/signing-identity", token);
      const adminOperations = [
        "state:read",
        "namespace:admin",
        "work:write",
        "outcome:write",
        "planning:write",
        "assessment:write",
        "identity.signing_key.enroll",
        "actor:delegate",
      ];
      await signing({
        namespace_id: namespace,
        expected_namespace_version: identity.namespace_version,
        operation: "set_credential_policy",
        credential_policy: {
          namespace_id: namespace,
          credential_id: identity.credential_id,
          principal_id: identity.principal_id,
          acceptance_floor: "independent_review",
          max_active_work_contracts: 3,
          max_active_review_contracts: 1,
          permitted_operations: adminOperations,
        },
      });
      identity = await api("/security/signing-identity", token);
      await signing({
        namespace_id: namespace,
        expected_namespace_version: identity.namespace_version,
        operation: "set_acceptance_policy",
        acceptance_policy: {
          namespace_id: namespace,
          acceptance_floor: "independent_review",
          max_active_work_contracts: 3,
          max_active_review_contracts: 1,
        },
      });
      for (const [index, phase] of [
        "draining",
        "contracts_v1",
        "draining_to_signed_v2",
        "signed_contracts_v2",
      ].entries())
        await command("set_namespace_work_protocol", {
          scope,
          expected_protocol_version: index + 1,
          phase,
          writers_drained: ["contracts_v1", "signed_contracts_v2"].includes(
            phase,
          ),
          reason: "Explicit signed browser fixture drain",
        });
      const work = await command("create_work_item", {
        scope,
        title: "Tarefa com avaliação independente",
        priority: "normal",
        lifecycle: "todo",
        execution_spec: { instructions: ["Executar somente esta obrigação"] },
      });
      const trust = await api(`/namespaces/${namespace}/signed-trust`, token);
      async function provision(name, permissions) {
        const own = await api("/security/signing-identity", token);
        const granted = await api("/security/commands", token, {
          namespace_id: namespace,
          expected_namespace_version: own.namespace_version,
          operation: "set_grant",
          principal_id: name,
          permissions,
        });
        const issued = await api("/security/commands", token, {
          namespace_id: namespace,
          expected_namespace_version: granted.result.namespace_version,
          operation: "issue_credential",
          principal_id: name,
          actor_ref: {
            kind: "agent",
            provider: "signed-browser-fixture",
            id: name,
          },
          expires_at: new Date(Date.now() + 3600000).toISOString(),
        });
        const self = await api("/security/signing-identity", issued.token),
          admin = await api("/security/signing-identity", token);
        const configured = await signing({
          namespace_id: namespace,
          expected_namespace_version: admin.namespace_version,
          operation: "set_credential_policy",
          credential_policy: {
            namespace_id: namespace,
            credential_id: self.credential_id,
            principal_id: name,
            acceptance_floor: "independent_review",
            max_active_work_contracts: 3,
            max_active_review_contracts: 1,
            permitted_operations: permissions,
          },
        });
        const seed = randomBytes(32),
          privateKey = createPrivateKey({
            key: Buffer.concat([
              Buffer.from("302e020100300506032b657004220420", "hex"),
              seed,
            ]),
            format: "der",
            type: "pkcs8",
          }),
          publicKey = createPublicKey(privateKey)
            .export({ format: "der", type: "spki" })
            .subarray(-32),
          fingerprint =
            "sha256:" + createHash("sha256").update(publicKey).digest("hex");
        const enrollment = await signing({
          namespace_id: namespace,
          expected_namespace_version: configured.namespace_version,
          operation: "create_enrollment",
          credential_id: self.credential_id,
          principal_id: name,
          expected_fingerprint: fingerprint,
        });
        const workspace = join(directory, name);
        await mkdir(workspace);
        const environment = {
          WOS_UI_PROFILE_TOKEN: issued.token,
          WOS_UI_PROFILE_KEY: seed.toString("base64"),
        };
        cli(
          workspace,
          name,
          environment,
          "init",
          "--workspace-schema",
          "2",
          "--server",
          url,
          "--server-id",
          trust.server.server_id,
          "--namespace",
          namespace,
          "--outcome",
          outcome.id,
        );
        await writeFile(
          join(workspace, "proposal.yaml"),
          JSON.stringify({
            schema_version: 2,
            kind: "WOSProfile",
            name,
            binding: {
              server_id: trust.server.server_id,
              server_origin: url,
              namespace_id: namespace,
              principal_id: name,
              credential_id: self.credential_id,
              issuer_keys: [
                {
                  key_id: trust.server.issuer_key_id,
                  public_key: trust.server.public_key,
                  fingerprint: trust.server.fingerprint,
                },
              ],
            },
            authentication: { credential_ref: "env:WOS_UI_PROFILE_TOKEN" },
            signing: {
              key_id: enrollment.enrollment.key_id,
              private_key_ref: "env:WOS_UI_PROFILE_KEY",
              public_key_fingerprint: fingerprint,
            },
            lease: { requested_ttl_seconds: 300 },
            output: { default_format: "json" },
            _local: {
              schema_version: 1,
              pending_mac: "",
              binding_mac: "",
              pending_operations: [],
            },
          }),
        );
        cli(
          workspace,
          name,
          environment,
          "profile",
          "create",
          name,
          "--file",
          "proposal.yaml",
          "--server",
          url,
          "--server-id",
          trust.server.server_id,
          "--issuer-fingerprint",
          trust.server.fingerprint,
          "--enrollment",
          enrollment.enrollment.id,
        );
        return { workspace, environment, name };
      }
      const executor = await provision("executor", [
        "state:read",
        "identity.signing_key.enroll",
        "work.contract.acquire",
        "work.contract.return",
        "work:write",
        "assessment:write",
        "conclusion:write",
      ]);
      const reviewer = await provision("reviewer", [
        "state:read",
        "identity.signing_key.enroll",
        "work.review.acquire",
        "work.review.decide",
        "assessment:write",
        "conclusion:write",
      ]);
      cli(
        executor.workspace,
        executor.name,
        executor.environment,
        "work",
        "checkout",
        work.id,
        "--version",
        String(work.version),
      );
      const executionDirectory = join(
          executor.workspace,
          ".wos/profiles/executor/contract",
        ),
        executionFile = (await readdir(executionDirectory)).find((p) =>
          p.endsWith(".yaml"),
        );
      assert.ok(executionFile);
      await editDraft(
        join(executionDirectory, executionFile),
        "execution",
        (lines) => {
          let summaries = 0;
          const changed = lines.map((line) => {
            if (/^\s+summary:/.test(line)) {
              summaries++;
              return line.replace(
                /summary:.*/,
                'summary: "Material exato entregue pelo executor"',
              );
            }
            return line;
          });
          assert.ok(summaries > 0);
          const material = changed.findIndex((line) =>
            /^\s+material:$/.test(line),
          );
          assert.ok(material >= 0);
          const indent = changed[material].match(/^\s*/)[0];
          changed.splice(
            material + 1,
            0,
            indent + '  reason: "Entrega exige avaliação independente"',
          );
          return changed;
        },
      );
      cli(
        executor.workspace,
        executor.name,
        executor.environment,
        "work",
        "finish",
        executionFile.slice(0, -5),
      );
      assert.deepEqual(
        await readdir(executionDirectory),
        [],
        "executor obligation cleanup is separate from Task approval",
      );
      await page.goto(url + "/app/");
      await page.getByLabel("Access credential", { exact: true }).fill(token);
      await page.getByRole("button", { name: "Sign in", exact: true }).click();
      await page.locator("#signed-protocol-label").click();
      await expect(
        page.getByText("Workspace identity", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText(trust.server.fingerprint, { exact: true }).first(),
      ).toBeVisible();
      await expect(
        page.getByText(trust.server.server_id, { exact: true }),
      ).toBeVisible();
      await page
        .locator("#outcomes")
        .getByRole("button")
        .filter({ hasText: outcome.title })
        .click();
      await page.getByRole("button", { name: "Work", exact: true }).click();
      await page.getByRole("button", { name: "Board", exact: true }).click();
      await page
        .getByRole("button")
        .filter({ hasText: work.title })
        .first()
        .click();
      await expect(
        page.getByText("Awaiting independent review", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole("button", { name: "Review submission", exact: true }),
      ).toHaveCount(0);
      await page
        .getByRole("button", { name: "Load accepted material", exact: true })
        .click();
      await expect(
        page.getByText("Material exato entregue pelo executor", {
          exact: true,
        }),
      ).toBeVisible();
      await page
        .getByText("Execute or review with a profile", { exact: true })
        .click();
      await context.grantPermissions(["clipboard-read", "clipboard-write"]);
      await page
        .getByRole("button", { name: "Copy reviewer commands", exact: true })
        .click();
      const handoff = await page.evaluate(() => navigator.clipboard.readText());
      assert.ok(handoff.includes("wosctl --profile reviewer review checkout"));
      assert.ok(!handoff.includes(token));
      await expect(
        page.getByRole("button", {
          name: "Copy executor commands",
          exact: true,
        }),
      ).toHaveCount(0);
      await page.locator("#entity-actions").click();
      await page.keyboard.press("Escape");
      await page.keyboard.press("Escape");
      await page.locator("#developer-tools").click();
      const choices = await page
        .locator("#command-select option")
        .evaluateAll((options) => options.map((o) => o.value));
      assert.ok(!choices.includes("record_criterion_assessment"));
      assert.ok(!choices.includes("finalize_work_contract"));
      assert.ok(!choices.includes("return_signed_review"));
      await page
        .locator("#command-dialog")
        .getByRole("button", { name: "Close", exact: true })
        .click();
      cli(
        reviewer.workspace,
        reviewer.name,
        reviewer.environment,
        "review",
        "checkout",
        "--next",
      );
      const reviewDirectory = join(
          reviewer.workspace,
          ".wos/profiles/reviewer/contract",
        ),
        reviewFile = (await readdir(reviewDirectory)).find((p) =>
          p.endsWith(".yaml"),
        );
      assert.ok(reviewFile);
      await editDraft(join(reviewDirectory, reviewFile), "review", (lines) => {
        let decisions = 0,
          reason = 0;
        const changed = lines.map((line) => {
          if (/^\s+decision:/.test(line)) {
            decisions++;
            return line.replace(/decision:.*/, "decision: " + decision);
          }
          if (/^\s+reason:/.test(line)) {
            reason++;
            return line.replace(
              /reason:.*/,
              'reason: "Avaliação independente do material exato"',
            );
          }
          return line;
        });
        assert.equal(decisions, 1);
        assert.equal(reason, 1);
        if (decision === "changes_requested") {
          const material = changed.findIndex((line) =>
            /^\s+material:$/.test(line),
          );
          assert.ok(material >= 0);
          const indent = changed[material].match(/^\s*/)[0];
          changed.splice(
            material + 1,
            0,
            indent + "  findings:",
            indent + "    - id: 01a11991-0000-7000-8000-000000000002",
            indent + "      requirement_ref: task.title",
            indent +
              '      description: "Confirmar cumprimento da tarefa original"',
          );
        }
        return changed;
      });
      cli(
        reviewer.workspace,
        reviewer.name,
        reviewer.environment,
        "review",
        "finish",
        reviewFile.slice(0, -5),
      );
      assert.deepEqual(await readdir(reviewDirectory), []);
      await expect(page.locator("#detail-dialog")).not.toBeVisible();
      await page.locator("#refresh").click();
      await page
        .getByRole("button")
        .filter({ hasText: work.title })
        .first()
        .click();
      await expect(
        page.getByText(
          decision === "approved" ? "Review approved" : "Changes requested",
          { exact: true },
        ),
      ).toBeVisible();
      if (decision === "changes_requested") {
        await expect(
          page.getByText("Confirmar cumprimento da tarefa original", {
            exact: true,
          }),
        ).toBeVisible();
        await expect(
          page.getByText("Requirement: task.title", { exact: true }),
        ).toBeVisible();
        await expect(
          page.locator(".signed-next-step pre").first(),
        ).toContainText("--previous-review");
      }
      assert.deepEqual(
        mutations,
        [],
        "browser projections never submit an unsigned substitute or fabricate a signature",
      );
      assert.deepEqual(errors, []);
      if (process.env.WOS_BROWSER_ARTIFACT_DIR)
        await page.screenshot({
          path: join(
            process.env.WOS_BROWSER_ARTIFACT_DIR,
            `10-signed-review-${decision}.png`,
          ),
          fullPage: true,
        });
    } finally {
      await context.close();
      if (child.exitCode === null) {
        const ended = once(child, "exit");
        child.kill("SIGTERM");
        await ended;
      }
      await rm(directory, { recursive: true, force: true });
    }
  });
