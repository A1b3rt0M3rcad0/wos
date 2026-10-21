import { expect, test } from "./ux-fixture.mjs";
import { seedOperationalCharacterization } from "./fixtures/operational-policy.mjs";

test("operational characterization fixture uses current persisted WOS facts", async ({ wos }) => {
  const fixture = await seedOperationalCharacterization(wos, {
    titlePrefix: "E03 fixture contract",
  });

  expect(fixture.outcome.lifecycle).toBe("active");
  expect(fixture.objectives.planned.lifecycle).toBe("planned");
  expect(fixture.objectives.inProgress.lifecycle).toBe("in_progress");
  expect(fixture.objectives.achieved.lifecycle).toBe("achieved");
  expect(fixture.objectives.cancelled.lifecycle).toBe("cancelled");

  expect(fixture.tasks.backlog.lifecycle).toBe("backlog");
  expect(fixture.tasks.todo.lifecycle).toBe("todo");
  expect(fixture.tasks.inProgress.lifecycle).toBe("in_progress");
  expect(fixture.tasks.accepted.lifecycle).toBe("done");
  expect(fixture.tasks.cancelled.lifecycle).toBe("cancelled");
  expect(fixture.tasks.hardDependent.lifecycle).toBe("backlog");
  expect(fixture.tasks.advisoryDependent.lifecycle).toBe("backlog");

  expect(fixture.dependencies.hard.strength).toBe("hard");
  expect(fixture.dependencies.advisory.strength).toBe("advisory");
  expect(fixture.blockers.direct.propagation).toBe("direct");
  expect(fixture.blockers.subtree.propagation).toBe("subtree");
  expect(fixture.issues.direct.objective_id).toBe(fixture.objectives.inProgress.id);
  expect(fixture.issues.contextual.objective_id).toBe(fixture.objectives.planned.id);

  expect(fixture.evidence.id).toBeTruthy();
  expect(fixture.evidenceLink.evidence_id).toBe(fixture.evidence.id);
  expect(fixture.credentials.samePrincipal[0].principal_id)
    .toBe(fixture.credentials.samePrincipal[1].principal_id);
  expect(fixture.credentials.samePrincipal[0].credential_id)
    .not.toBe(fixture.credentials.samePrincipal[1].credential_id);
  expect(fixture.credentials.independentReviewer.principal_id)
    .not.toBe(fixture.credentials.samePrincipal[0].principal_id);
});
