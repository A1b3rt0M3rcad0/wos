import { test, expect } from "playwright/test";
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { mkdtemp, mkdir, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { resolve, join } from "node:path";
import { createServer } from "node:net";
import { once } from "node:events";
import { randomUUID, randomBytes } from "node:crypto";

test("workspace summary, scoped list, operational board, pagination and real human action", async ({
  browser,
}) => {
  test.setTimeout(120000);
  const directory = await mkdtemp(join(tmpdir(), "wos-workspace-"));
  const socket = createServer();
  socket.listen(0, "127.0.0.1");
  await once(socket, "listening");
  const port = socket.address().port;
  await new Promise((done) => socket.close(done));
  const url = `http://127.0.0.1:${port}`;
  const namespace = "01a11820-0000-7000-8000-000000000001";
  const bootstrap = randomBytes(32).toString("base64url");
  let logs = "";
  const child = spawn(resolve("../../bin/wos"), ["server"], {
    env: {
      ...process.env,
      WOS_LISTEN: `127.0.0.1:${port}`,
      WOS_SQLITE_PATH: join(directory, "wos.db"),
      WOS_AUTH_MODE: "api_token",
      WOS_BOOTSTRAP_TOKEN: bootstrap,
      WOS_BOOTSTRAP_NAMESPACE_ID: namespace,
      WOS_BOOTSTRAP_NAMESPACE_NAME: "Produto & Engenharia",
      WOS_LOCAL_PRINCIPAL_ID: "marina",
      WOS_MCP_ENABLED: "true",
    },
    stdio: ["ignore", "ignore", "pipe"],
  });
  child.stderr.on("data", (b) => (logs = (logs + b.toString()).slice(-8000)));
  const context = await browser.newContext({
    viewport: { width: 1512, height: 982 },
    timezoneId: "America/Sao_Paulo",
  });
  const page = await context.newPage();
  const errors = [];
  const failedResources = [];
  page.on("response", (r) => {
    if (r.status() === 404) failedResources.push(r.url());
  });
  page.on("pageerror", (e) => errors.push(e.message));
  page.on("console", (msg) => {
    if (msg.type() === "error" && !msg.text().includes("401"))
      errors.push(msg.text());
  });
  async function command(name, cmd) {
    const response = await fetch(url + "/api/v1/commands/" + name, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${bootstrap}`,
        "Idempotency-Key": randomUUID(),
      },
      body: JSON.stringify({ command: cmd }),
    });
    const result = await response.json();
    assert.ok(response.ok, JSON.stringify(result));
    return result.value;
  }
  async function read(path) {
    const r = await fetch(url + "/api/v1" + path, {
      headers: { Authorization: `Bearer ${bootstrap}` },
    });
    assert.ok(r.ok);
    return r.json();
  }
  async function shot(name) {
    if (process.env.WOS_BROWSER_ARTIFACT_DIR) {
      await mkdir(process.env.WOS_BROWSER_ARTIFACT_DIR, { recursive: true });
      await page
        .locator(".board-column")
        .evaluateAll((nodes) => nodes.forEach((n) => (n.scrollTop = 0)));
      await page.screenshot({
        path: join(process.env.WOS_BROWSER_ARTIFACT_DIR, name + ".png"),
        fullPage: name !== "08-mobile",
      });
    }
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
      title: "Lançar o portal de colaboração",
      desired_state:
        "Clientes e agentes acompanham cada entrega, com evidências revisadas e uma experiência clara.",
      priority: "high",
    });
    const scope = { namespace_id: namespace, outcome_id: outcome.id };
    await command("add_criterion", {
      owner: { ...scope, kind: "outcome", id: outcome.id },
      expected_version: outcome.version,
      title: "Entrega aprovada pela revisão humana",
      required: true,
      verification_mode: "attestation",
    });
    await command("activate_outcome", {
      scope,
      expected_version: outcome.version + 1,
    });
    const objective = await command("create_objective", {
      scope,
      title: "Uma experiência pronta para clientes",
      description:
        "Navegação clara, entregas rastreáveis e critérios de aceite.",
      priority: "high",
      required_for_outcome: true,
    });
    await command("add_criterion", {
      owner: { ...scope, kind: "objective", id: objective.id },
      expected_version: objective.version,
      title: "Fluxo principal validado no navegador",
      required: true,
      verification_mode: "evidence_review",
    });
    await command("start_objective", {
      scope,
      objective_id: objective.id,
      expected_version: objective.version + 1,
    });
    const titles = [
      "Mapear a jornada de onboarding",
      "Refinar navegação do workspace",
      "Preparar checklist de acessibilidade",
      "Conectar agentes ao contexto",
      "Revisar estados vazios",
      "Organizar critérios de sucesso",
      "Documentar decisões do produto",
    ];
    const readyItems = [];
    for (let i = 0; i < 28; i++)
      readyItems.push(
        await command("create_work_item", {
          scope,
          title:
            titles[i] || `Revisar entrega ${String(i + 1).padStart(2, "0")}`,
          description:
            "Entrega delimitada do portal de colaboração. Validar com a equipe e registrar evidências.",
          priority: i === 0 ? "high" : i % 3 === 0 ? "low" : "normal",
          lifecycle: "todo",
          objective_id: objective.id,
        }),
      );
    const active = await command("create_work_item", {
      scope,
      title: "Implementar revisão de evidências",
      priority: "high",
      lifecycle: "todo",
      objective_id: objective.id,
    });
    await command("claim_work_item", {
      scope,
      work_item_id: active.id,
      expected_version: 1,
      ttl_seconds: 900,
    });
    const blocked = await command("create_work_item", {
      scope,
      title: "Publicar ambiente de homologação",
      priority: "critical",
      lifecycle: "todo",
    });
    const issue = await command("create_issue", {
      scope,
      title: "Acesso à homologação pendente",
      description: "Aguardar credencial do ambiente antes de publicar.",
      severity: "major",
    });
    await command("create_blocker", {
      scope,
      blocked_ref: { ...scope, kind: "work_item", id: blocked.id },
      cause_ref: { ...scope, kind: "issue", id: issue.id },
      description: "Aguardar acesso à homologação",
      propagation: "direct",
    });
    const done = await command("create_work_item", {
      scope,
      title: "Validar contrato HTTP e MCP",
      priority: "normal",
      lifecycle: "todo",
    });
    const claim = await command("claim_work_item", {
      scope,
      work_item_id: done.id,
      expected_version: 1,
      ttl_seconds: 900,
    });
    await command("complete_work_item", {
      scope,
      work_item_id: done.id,
      expected_version: claim.version,
      claim_id: claim.current_lease.claim_id,
      fencing_token: claim.current_lease.fencing_token,
      reason: "Contratos equivalentes verificados.",
      result_summary: "HTTP e MCP compartilham estado e recibos idempotentes.",
    });
    await command("create_work_item", {
      scope,
      title: "Planejar acompanhamento pós-lançamento",
      priority: "low",
      lifecycle: "backlog",
    });
    await command("register_evidence", {
      scope,
      evidence_type: "test_result",
      description: "Jornadas de colaboração verificadas no Chromium",
      source_ref: {
        provider: "playwright",
        uri: "https://example.test/runs/portal",
        display_name: "Aceite do portal",
      },
      captured_at: new Date().toISOString(),
    });
    const roadmap = await command("create_roadmap", {
      scope,
      title: "Portal de colaboração · primeira entrega",
      plan_scope: { kind: "outcome", id: outcome.id },
    });
    const draft = await command("open_roadmap_draft", {
      scope,
      roadmap_id: roadmap.id,
      expected_version: roadmap.version,
    });
    const updated = await command("replace_roadmap_draft", {
      scope,
      roadmap_id: roadmap.id,
      expected_version: draft.version,
      expected_draft_version: draft.draft.draft_version,
      nodes: [
        {
          node_key: "discovery",
          node_type: "phase",
          title: "Descobrir e planejar",
          position: 0,
        },
        {
          node_key: "journey",
          node_type: "reference",
          title: "Mapear a jornada",
          parent_node_key: "discovery",
          position: 1,
          target_ref: { ...scope, kind: "work_item", id: readyItems[0].id },
        },
        {
          node_key: "proof",
          node_type: "reference",
          title: "Revisar as evidências",
          position: 2,
          target_ref: { ...scope, kind: "work_item", id: active.id },
        },
        {
          node_key: "delivery",
          node_type: "milestone",
          title: "Validar com clientes",
          position: 3,
        },
      ],
      after_links: [],
    });
    const published = await command("publish_roadmap_draft", {
      scope,
      roadmap_id: roadmap.id,
      expected_version: updated.version,
      expected_draft_version: updated.draft.draft_version,
    });
    await command("activate_roadmap_revision", {
      scope,
      roadmap_id: roadmap.id,
      expected_version: published.version,
      revision_number: 1,
    });
    const other = await command("create_outcome", {
      namespace_id: namespace,
      title: "Melhorar a operação de suporte",
      desired_state: "Menos tempo de resposta e resolução com evidências.",
      priority: "normal",
    });
    await page.goto(url + "/app/");
    await shot("01-acesso");
    await page.getByLabel("Access credential", { exact: true }).fill(bootstrap);
    await page.getByRole("button", { name: "Sign in", exact: true }).click();
    await page
      .locator("#outcomes")
      .getByRole("button", { name: /Lançar o portal/ })
      .click();
    await expect(page.locator("#summary-view")).toBeVisible();
    await expect(page.locator("#metrics")).toContainText("1 of 32 items");
    await expect(page.locator("#summary-content")).toContainText(
      "Clientes e agentes",
    );
    await expect(page.locator("#summary-content")).toContainText(
      "Active roadmaps",
    );
    await shot("02-resumo");
    await page.getByRole("button", { name: "Work", exact: true }).click();
    await page.getByRole("button", { name: "Board", exact: true }).click();
    await expect(page.locator(".board-column")).toHaveCount(5);
    await expect(
      page.getByRole("region", { name: "In progress", exact: true }),
    ).toContainText("Implementar revisão de evidências");
    await expect(
      page.getByRole("region", { name: "Blockers", exact: true }),
    ).toContainText("Publicar ambiente de homologação");
    await expect(
      page.getByRole("region", { name: "Done", exact: true }),
    ).toContainText("Validar contrato HTTP e MCP");
    await expect(
      page.getByRole("region", { name: "Ready", exact: true }).locator(".item"),
    ).toHaveCount(25);
    await page
      .getByRole("button", {
        name: "Load more · Ready",
        exact: true,
      })
      .click();
    await expect(
      page.getByRole("region", { name: "Ready", exact: true }).locator(".item"),
    ).toHaveCount(28);
    await shot("03-quadro");
    await page.getByLabel("Search items", { exact: true }).fill("Mapear");
    await expect(page.locator(".board .item")).toHaveCount(1);
    await page.getByLabel("Priority", { exact: true }).selectOption("low");
    await expect(page.locator(".board .item")).toHaveCount(0);
    await page.getByLabel("Priority", { exact: true }).selectOption("");
    await page.getByLabel("Search items", { exact: true }).fill("");
    await page.getByRole("button", { name: "List", exact: true }).click();
    await page
      .getByLabel("Work status", { exact: true })
      .selectOption("work_items");
    await expect(page.locator(".list-row")).toHaveCount(25);
    await page.getByRole("button", { name: "Load more", exact: true }).click();
    await expect(page.locator(".list-row")).toHaveCount(32);
    await shot("04-lista");
    await page.getByLabel("Search items", { exact: true }).fill("Mapear");
    await page
      .locator("#content")
      .getByRole("button", { name: /Mapear a jornada/ })
      .click();
    await expect(page.locator("#detail-content")).toContainText(
      "Entrega delimitada",
    );
    await expect(page.locator("#detail-content pre")).not.toBeVisible();
    await shot("05-detalhe");
    await page
      .locator("#detail-actions")
      .getByRole("button", { name: "Reserve task", exact: true })
      .click();
    await expect(page.locator("#fields")).toContainText("Mapear a jornada");
    await expect(page.locator("#fields input[readonly]")).toHaveCount(0);
    await page.locator("#submit-command").click();
    await expect(page.locator("#command-dialog")).not.toBeVisible();
    const result = await read(
      `/namespaces/${namespace}/outcomes/${outcome.id}/work-items/${readyItems[0].id}`,
    );
    assert.equal(result.value.lifecycle, "in_progress");
    await page.getByRole("button", { name: "Work", exact: true }).click();
    await page.getByRole("button", { name: "Board", exact: true }).click();
    await expect(
      page.getByRole("region", { name: "In progress", exact: true }),
    ).toContainText("Mapear a jornada");
    await page.getByRole("button", { name: "List", exact: true }).click();
    await page
      .locator("#area-tabs")
      .getByRole("button", { name: "Plan", exact: true })
      .click();
    await page
      .locator("#tabs")
      .getByRole("button", { name: "All roadmaps", exact: true })
      .click();
    await page
      .locator("#content")
      .getByRole("button", { name: /Portal de colaboração/ })
      .click();
    await expect(page.locator("#detail-content")).toContainText(
      "Descobrir e planejar",
    );
    await expect(page.locator("#detail-content")).toContainText(
      "Validar com clientes",
    );
    await expect(page.locator("#detail-content pre")).not.toBeVisible();
    await expect(
      page.locator("#detail-actions").getByRole("button", {
        name: "Publish roadmap draft",
        exact: true,
      }),
    ).toHaveCount(0);
    await shot("06-plano");
    await page.keyboard.press("Escape");
    await page
      .locator("#area-tabs")
      .getByRole("button", { name: "Plan", exact: true })
      .click();
    await page
      .locator("#tabs")
      .getByRole("button", { name: "Objectives", exact: true })
      .click();
    await page
      .locator("#content")
      .getByRole("button", { name: /Uma experiência pronta/ })
      .click();
    await page
      .getByRole("button", { name: /Fluxo principal validado/ })
      .click();
    await page.keyboard.press("Escape");
    await page.locator("#developer-tools").click();
    await page
      .locator("#command-select")
      .selectOption("record_criterion_assessment");
    await expect(page.locator("#fields")).toContainText(
      "Fluxo principal validado no navegador",
    );
    await expect(
      page.getByLabel("Success criterion", { exact: true }),
    ).toHaveCount(0);
    await page
      .getByLabel("Rationale", { exact: true })
      .fill("Fluxo de colaboração verificado na jornada de aceite.");
    const proof = page
      .locator("fieldset")
      .filter({
        has: page.locator("legend").getByText("Evidence used", { exact: true }),
      })
      .last();
    await proof.getByRole("button", { name: "Add item", exact: true }).click();
    await proof
      .getByRole("combobox", { name: "Evidence", exact: true })
      .selectOption({
        label: "Jornadas de colaboração verificadas no Chromium",
      });
    await shot("09-avaliacao");
    await page.keyboard.press("Escape");
    await page
      .locator("#area-tabs")
      .getByRole("button", { name: "Issues", exact: true })
      .click();
    await page
      .locator("#tabs")
      .getByRole("button", { name: "Blockers", exact: true })
      .click();
    await page
      .locator("#content")
      .getByRole("button", { name: /Aguardar acesso/ })
      .click();
    await expect(page.locator("#detail-content")).toContainText(
      "Publicar ambiente de homologação",
    );
    await expect(page.locator("#detail-content")).toContainText(
      "Acesso à homologação pendente",
    );
    await page.keyboard.press("Escape");
    await page
      .locator("#area-tabs")
      .getByRole("button", { name: "Activity", exact: true })
      .click();
    await expect(page.locator(".timeline-entry").first()).toBeVisible();
    await expect(page.locator("#content pre")).toHaveCount(0);
    await shot("07-historico");
    await command("cancel_work_item", {
      scope,
      work_item_id: readyItems[1].id,
      expected_version: 1,
      reason: "Entrega removida do escopo por decisão explícita.",
    });
    await page.locator("#refresh").click();
    await page.getByRole("button", { name: "Work", exact: true }).click();
    await page.getByRole("button", { name: "Board", exact: true }).click();
    await expect(
      page.getByRole("region", { name: "Cancelled", exact: true }),
    ).toContainText("Refinar navegação do workspace");
    await expect(
      page.getByRole("region", { name: "Done", exact: true }),
    ).not.toContainText("Refinar navegação do workspace");
    // A delayed response for another Outcome cannot replace the current workspace.
    let release;
    const delayed = new Promise((resolve) => (release = resolve));
    let seen;
    const arrived = new Promise((resolve) => (seen = resolve));
    await page.route(`**/outcomes/${other.id}/continuity?*`, async (route) => {
      seen();
      await delayed;
      await route.continue();
    });
    await page
      .locator("#outcomes")
      .getByRole("button", { name: /operação de suporte/ })
      .click();
    await arrived;
    await expect(page.locator("#outcome")).not.toBeVisible();
    await page
      .locator("#outcomes")
      .getByRole("button", { name: /Lançar o portal/ })
      .click();
    await expect(page.locator("#title")).toHaveText(
      "Lançar o portal de colaboração",
    );
    release();
    await page.waitForResponse((r) =>
      r.url().includes(other.id + "/continuity"),
    );
    await expect(page.locator("#title")).toHaveText(
      "Lançar o portal de colaboração",
    );
    await page.setViewportSize({ width: 390, height: 844 });
    await page.getByRole("button", { name: "Work", exact: true }).click();
    await page.getByRole("button", { name: "Board", exact: true }).click();
    assert.ok(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
      "mobile horizontal overflow",
    );
    await page.locator(".board").scrollIntoViewIfNeeded();
    await shot("08-mobile");
    await page.locator("#actions").focus();
    await page.keyboard.press("Enter");
    await expect(page.locator("#action-dialog")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(page.locator("#actions")).toBeFocused();
    assert.deepEqual(errors, [], JSON.stringify(failedResources));
  } finally {
    await context.close();
    if (child.exitCode === null) {
      const exited = once(child, "exit");
      child.kill("SIGTERM");
      await exited;
    }
    await rm(directory, { recursive: true, force: true });
  }
});
