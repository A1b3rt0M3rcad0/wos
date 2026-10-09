import { message } from "./en-US.js";
export function contractViews({
  state,
  api,
  request,
  inventory,
  $,
  el,
  copy,
  date,
  display,
  badge,
  readable,
  referenceButton,
  outcomeBase,
  notice,
  report,
  openCommands,
  shortId,
}) {
  async function contractSection(detail, entity, generation) {
    if (state.protocol?.phase === "signed_contracts_v2") {
      return signedContractSection(detail, entity, generation);
    }

    const section = el(
      "section",
      undefined,
      copy.text.detailSectionContractSection,
    );
    section.append(el("h3", copy.text.workContract));
    detail.append(section);
    let loadGeneration = 0;
    const load = async (id, expand) => {
      const requested = ++loadGeneration;
      const response = await request(`${outcomeBase()}/work-contracts/${id}`);
      if (generation !== state.detailGeneration || requested !== loadGeneration)
        return;
      const view = response.value || response,
        c = view.contract;
      if (expand && view.omitted?.[expand]) {
        const expanded = await request(
          `${outcomeBase()}/${view.omitted[expand]}`,
        );
        if (
          generation !== state.detailGeneration ||
          requested !== loadGeneration
        )
          return;
        view[expand] = expanded.value || expanded;
        delete view.omitted[expand];
      }
      state.contractView = view;
      state.submission = view.latest_submission || null;
      const status = {
        active: copy.text.activeReservation,
        expired: copy.text.expiredReservation,
        revoked: copy.text.revokedContract,
        completed: copy.text.deliveryCompleted,
      };
      section.replaceChildren(
        el("h3", copy.text.workContract),
        el("strong", status[view.effective_status] || view.effective_status),
      );
      const info = el("dl", undefined, "detail-properties");
      for (const [label, value] of [
        [copy.text.holder, c.holder_principal_id],
        [copy.text.reservationExpiry, date(c.expires_at)],
        [copy.text.checkpointVersion, String(c.version)],
        [copy.text.leaseVersion, String(c.lease_version)],
        [
          copy.text.executionPermitted,
          view.execution_allowed ? copy.text.yes : copy.text.no,
        ],
      ])
        info.append(el("dt", label), el("dd", value));
      section.append(info);
      if (view.reasons?.length)
        section.append(
          el(
            "p",
            view.reasons
              .map((reason) => reason.message || display(reason.code))
              .join(copy.text.message),
            "contract-notice",
          ),
        );
      for (const [key, label] of [
        ["latest_checkpoint", copy.text.loadLatestCheckpoint],
        ["latest_submission", copy.text.loadLatestSubmission],
      ])
        if (view.omitted?.[key]) {
          const button = el("button", label, "quiet");
          button.onclick = () =>
            load(c.id, key).catch((error) => report(error));
          section.append(button);
        }
      if (view.latest_checkpoint) {
        const checkpoint = view.latest_checkpoint;
        const progress = el("div", undefined, "contract-progress");
        progress.append(
          el("h4", copy.text.latestCheckpoint),
          el("p", checkpoint.summary),
        );
        if (checkpoint.next_action)
          progress.append(
            el("p", message("nextAction", checkpoint.next_action)),
          );
        if (checkpoint.pending?.length)
          progress.append(
            el(
              "p",
              message("pending", checkpoint.pending.join(copy.text.message)),
            ),
          );
        if (checkpoint.dirty)
          progress.append(
            el("small", copy.text.localChangesHaveNotYetBeenTransferredToWos),
          );
        if (checkpoint.working_commit)
          progress.append(
            el("small", message("codeReference", checkpoint.working_commit)),
          );
        section.append(progress);
      }
      if (view.latest_submission) {
        const submission = view.latest_submission;
        const delivery = el("div", undefined, "contract-delivery");
        delivery.append(
          el("h4", copy.text.submissionForAssessment),
          el("p", submission.material.summary),
          el(
            "small",
            message(
              "artifactsEvidenceSubmitted",
              submission.material.artifacts?.length || 0,
              submission.material.evidence_ids?.length || 0,
              date(submission.submitted_at),
            ),
          ),
        );
        const review = el("button", copy.text.reviewSubmission, "quiet");
        review.disabled =
          !state.permissions.includes("assessment:write") ||
          view.effective_status !== "active" ||
          c.id !== entity.current_contract_id;
        review.onclick = () => {
          state.submission = submission;
          state.criterion =
            entity.criteria?.items?.find((x) => x.required) ||
            entity.criteria?.items?.[0];
          if (!state.criterion) {
            notice(
              copy.text
                .thisTaskRequiresNoCriterionAssessmentFinalizationRemainsExplicit,
            );
            return;
          }
          openCommands("record_criterion_assessment");
        };
        delivery.append(review);
        section.append(delivery);
      }
      if (
        view.effective_status === "active" &&
        c.id === entity.current_contract_id
      ) {
        const revoke = el(
          "button",
          copy.text.revokeContract,
          copy.text.quietDanger,
        );
        revoke.onclick = () => openCommands("revoke_work_contract");
        if (state.permissions.includes("work.contract.revoke")) {
          const advanced = el("details");
          advanced.append(el("summary", copy.text.advanced), revoke);
          section.append(advanced);
        }
      }
      const audit = el("details", undefined, "technical");
      audit.append(
        el("summary", copy.text.contractIdentityAndSpecification),
        el(
          "pre",
          JSON.stringify(
            {
              contract_id: c.id,
              execution_id: c.execution_id,
              fencing_token: c.fencing_token,
              spec_digest: c.spec_digest,
              spec: c.spec,
            },
            null,
            2,
          ),
        ),
      );
      section.append(audit);
    };
    try {
      if (entity.current_contract_id) await load(entity.current_contract_id);
      else
        section.append(
          el(
            "p",
            entity.lifecycle === "in_progress"
              ? copy.text.recoverableWorkExplicitlyAcquireANewContractToExecute
              : copy.text.noActiveReservationAcquisitionIsExplicit,
          ),
        );
      const history = el("details");
      history.append(el("summary", copy.text.contractHistory));
      const body = el("div");
      history.append(body);
      let cursor = "",
        loaded = false;
      const more = el("button", copy.text.loadContracts, "quiet");
      const loadHistory = async () => {
        const page = await request(
          `${outcomeBase()}/work-contracts?work_item_id=${encodeURIComponent(entity.id)}&limit=10&cursor=${encodeURIComponent(cursor)}`,
        );
        if (generation !== state.detailGeneration) return;
        loaded = true;
        for (const contract of page.items || []) {
          const button = el(
            "button",
            `${shortId(contract.id)} · ${contract.status} · ${date(contract.acquired_at)}`,
            "quiet",
          );
          button.onclick = () =>
            load(contract.id).catch((error) => report(error));
          body.append(button);
        }
        cursor = page.next_cursor || "";
        more.hidden = !cursor;
        if (!page.items?.length)
          body.append(el("p", copy.text.noContractsRecorded));
      };
      more.onclick = () => loadHistory().catch((error) => report(error));
      history.ontoggle = () => {
        if (history.open && !loaded)
          loadHistory().catch((error) => report(error));
      };
      history.append(more);
      detail.append(history);
    } catch (error) {
      section.append(
        el(
          "p",
          message("couldNotLoadTheContract", error.message),
          "contract-notice",
        ),
      );
    }
  }

  const unsignedExecutionActions = new Set([
    "claim_work_item",
    "release_work_item",
    "renew_work_item_lease",
    "reclaim_work_item",
    "complete_work_item",
    "administrative_complete_work_item",
    "acquire_work_contract",
    "acquire_next_work_contract",
    "renew_work_contract",
    "resume_work_contract",
    "sync_work_contract",
    "submit_work_result",
    "finalize_work_contract",
    "revoke_work_contract",
  ]);
  function commandAvailableInUI(name) {
    // Browser forms never hold an agent private key or manufacture a proof.
    if (inventory[name]?.exposure === "S") return false;
    if (
      state.protocol?.phase === "signed_contracts_v2" &&
      state.selected?._kind === "work_item" &&
      [
        "attest_criterion",
        "record_criterion_assessment",
        "waive_criterion",
      ].includes(name)
    )
      return false;
    return (
      state.protocol?.phase !== "signed_contracts_v2" ||
      !unsignedExecutionActions.has(name)
    );
  }
  async function refreshNamespaceProtocol() {
    const namespace = state.namespace,
      generation = state.generation;
    const protocol = await request(
      `${api}/namespaces/${namespace}/work-protocol`,
    );
    if (generation !== state.generation || namespace !== state.namespace)
      return;
    state.protocol = protocol;
    renderProtocolBanner();
  }
  function renderProtocolBanner() {
    const panel = $("signed-protocol-info");
    const phase = state.protocol?.phase;
    panel.hidden = !["signed_contracts_v2", "draining_to_signed_v2"].includes(
      phase,
    );
    $("signed-protocol-label").textContent =
      phase === "signed_contracts_v2"
        ? copy.text.signedContractsIndependentExecutionAndReview
        : copy.text.preparingSignedContractsNewLegacyReservationsSuspended;
  }
  function bindProtocolEvents() {
    $("signed-protocol-info").ontoggle = () => {
      if ($("signed-protocol-info").open) loadSignedIdentity().catch(report);
    };
  }

  async function loadSignedIdentity() {
    const namespace = state.namespace,
      generation = state.generation;
    const target = $("signed-protocol-content");
    target.replaceChildren(el("p", copy.text.loadingIdentityAndPolicies));
    const [trust, identity] = await Promise.all([
      request(`${api}/namespaces/${namespace}/signed-trust`),
      request(`${api}/security/signing-identity`),
    ]);
    if (generation !== state.generation || namespace !== state.namespace)
      return;
    const identityCard = el("section", undefined, "signed-identity-card");
    identityCard.append(el("h3", copy.text.workspaceIdentity));
    const properties = el("dl", undefined, "detail-properties");
    for (const [label, value] of [
      [copy.text.account, identity.principal_id],
      [copy.text.credential, identity.credential_id],
      [copy.text.persistentServer, trust.server?.server_id],
      [copy.text.currentIssuer, trust.server?.fingerprint],
      [
        copy.text.credentialAcceptanceRequirement,
        display(
          identity.credential_policy?.acceptance_floor ||
            copy.text.notConfigured,
        ),
      ],
    ])
      properties.append(
        el("dt", label),
        el("dd", value || copy.text.notAvailable),
      );
    identityCard.append(properties);
    const keys = el("section", undefined, "signed-identity-card");
    keys.append(el("h3", copy.text.identityKeys));
    for (const key of identity.keys || []) {
      const row = el("p", undefined, "signed-key");
      row.append(
        badge(key.status),
        el("code", shortId(key.id)),
        el("small", key.fingerprint),
      );
      keys.append(row);
    }
    if (!identity.keys?.length)
      keys.append(el("p", copy.text.noKeysReturnedByThisQuery));
    if (identity.keys_truncated)
      keys.append(
        el("small", copy.text.partialListContinueThroughApiPagesForOtherKeys),
      );
    const onboarding = el(
      "section",
      undefined,
      copy.text.signedIdentityCardSignedOnboarding,
    );
    onboarding.append(
      el("h3", copy.text.connectAProfile),
      el(
        "p",
        copy.text
          .askAnAdministratorForACredentialAndEnrollmentForYourRoleVeri4e28b08c,
      ),
    );
    const example = `wosctl init --workspace-schema 2 --server ${location.origin} --server-id ${trust.server?.server_id || "SERVER_UUID"} --namespace ${namespace} --outcome ${state.outcome?.id || "OUTCOME_UUID"}`;
    onboarding.append(
      el("pre", example),
      el(
        "p",
        copy.text
          .tokensAndPrivateKeysStayOnTheHostInProtectedReferencesInstal90b2e966,
      ),
    );
    const history = el("details", undefined, "technical");
    history.append(el("summary", copy.text.retainedPublicIssuers));
    for (const issuer of trust.issuer_history || [])
      history.append(
        el("p", `${shortId(issuer.issuer_key_id)} · ${issuer.fingerprint}`),
      );
    if (trust.search_complete === false)
      history.append(
        el("small", copy.text.partialHistoryContinueWithTheApiCursor),
      );
    target.replaceChildren(identityCard, keys, onboarding, history);
  }
  function decodeSignedProjection(encoded) {
    const bytes = Uint8Array.from(atob(encoded), (c) => c.charCodeAt(0));
    return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
  }
  function cliHandoff(profile, command) {
    const group = el("section", undefined, "cli-handoff");
    group.append(el("h4", `${profile} profile`), el("pre", command));
    const b = el(
      "button",
      message("copyCommands", profile.toLowerCase()),
      "quiet",
    );
    b.type = "button";
    b.onclick = async () => {
      try {
        await navigator.clipboard.writeText(command);
        notice(
          message(
            "commandsCopiedReplaceContractPlaceholdersOnlyAfterAnAuthorized",
            profile,
          ),
        );
      } catch (error) {
        report(error);
      }
    };
    group.append(b);
    return group;
  }
  async function signedContractSection(detail, entity, generation) {
    state.contractView = null;
    state.submission = null;
    const section = el(
      "section",
      undefined,
      copy.text.detailSectionContractSectionSignedContractSection,
    );
    section.append(
      el("h3", copy.text.signedExecutionAndReview),
      el(
        "p",
        copy.text
          .signaturesEstablishOriginAndIntegrityQualityRequiresExplicitd2f2854a,
        "contract-notice",
      ),
    );
    detail.append(section);
    const current = () => generation === state.detailGeneration;
    try {
      if (entity.current_contract_id) {
        const result = await request(
          `${outcomeBase()}/signed-state/execution/${entity.current_contract_id}`,
        );
        if (!current()) return;
        const contract = result.contract;
        const names = {
          active: copy.text.executionReserved,
          delivered: copy.text.executorSubmissionAccepted,
          completed: copy.text.requirementCompleted,
          revoked: copy.text.revokedContract,
          expired: copy.text.expiredReservation,
        };
        section.append(
          el(
            "strong",
            names[contract.effective_status] ||
              display(contract.effective_status),
          ),
        );
        const info = el("dl", undefined, "detail-properties");
        for (const [label, value] of [
          [copy.text.holder, contract.holder_principal_id],
          [copy.text.expiry, date(contract.expires_at)],
          [
            copy.text.reservationCurrentlyValid,
            contract.lease_valid ? copy.text.yes : copy.text.no,
          ],
          [copy.text.contractVersion, contract.contract_version],
          [copy.text.leaseVersion, contract.lease_version],
        ])
          info.append(el("dt", label), el("dd", String(value)));
        section.append(info);
      }
      let pendingReviewVersion;
      const caseIDs = [
        ...new Set(
          [
            entity.pending_review_case_id,
            entity.correction_review_case_id,
            entity.latest_review_case_id,
          ].filter(Boolean),
        ),
      ];
      for (const id of caseIDs) {
        const result = await request(
          `${outcomeBase()}/signed-state/case/${id}`,
        );
        if (!current()) return;
        const review = result.review_case;
        if (review.id === entity.pending_review_case_id)
          pendingReviewVersion = review.version;
        const card = el("section", undefined, "signed-review-card");
        const names = {
          pending: copy.text.awaitingIndependentReview,
          in_review: copy.text.reviewInProgress,
          approved: copy.text.reviewApproved,
          changes_requested: copy.text.changesRequested,
          cancelled: copy.text.reviewCancelled,
          superseded: copy.text.reviewSuperseded,
        };
        card.append(
          el("h4", names[review.status] || display(review.status)),
          el("p", message("rodadaCaso", review.round, shortId(review.id))),
        );
        card.append(
          el(
            "p",
            message(
              "entregaAcceptanceRequirement",
              shortId(review.submission_id),
              display(review.acceptance_floor),
            ),
          ),
        );
        const materialButton = el(
          "button",
          copy.text.loadAcceptedMaterial,
          "quiet",
        );
        materialButton.onclick = async () => {
          materialButton.disabled = true;
          try {
            const accepted = await request(
              `${outcomeBase()}/signed-state/submission/${review.submission_id}?digest=${encodeURIComponent(review.submission_digest)}`,
            );
            if (!current()) return;
            const material = decodeSignedProjection(accepted.material_payload);
            const delivery = el("section", undefined, "contract-delivery");
            delivery.append(
              el("h4", copy.text.acceptedExecutorMaterial),
              el("p", material.summary),
              el(
                "small",
                message(
                  "artifactsEvidence",
                  material.artifacts?.length || 0,
                  material.evidence_ids?.length || 0,
                ),
              ),
            );
            const canonical = el("details", undefined, "technical");
            canonical.append(
              el("summary", copy.text.materialReferences),
              el("pre", JSON.stringify(material, null, 2)),
            );
            delivery.append(canonical);
            card.insertBefore(delivery, materialButton);
            materialButton.remove();
          } catch (error) {
            if (current()) notice(error.message, true);
            materialButton.disabled = false;
          }
        };
        card.append(materialButton);
        if (review.close_reason) card.append(el("p", review.close_reason));
        if (
          review.status === "changes_requested" &&
          review.latest_decision_id
        ) {
          const acceptedDecision = await request(
            `${outcomeBase()}/signed-state/correction/${review.id}`,
          );
          if (!current()) return;
          const decision = decodeSignedProjection(
            acceptedDecision.envelope.payload,
          );
          card.append(
            el("h4", copy.text.requiredCorrections),
            el("p", decision.material.reason),
          );
          for (const finding of decision.material.findings || []) {
            const item = el("div", undefined, "signed-finding");
            item.append(
              el("p", finding.description),
              el(
                "small",
                message(
                  "requirement",
                  finding.requirement_ref || shortId(finding.criterion_id),
                ),
              ),
            );
            card.append(item);
          }
          card.append(
            el(
              "p",
              copy.text
                .theNextExecutionMustAddressEveryFindingOnExistingRequirement480c9676,
            ),
          );
        }
        if (review.current_contract_id) {
          const reserved = await request(
            `${outcomeBase()}/signed-state/review/${review.current_contract_id}`,
          );
          if (!current()) return;
          card.append(
            el(
              "small",
              message(
                "reviewerReservation",
                reserved.contract.holder_principal_id,
                reserved.contract.lease_valid
                  ? copy.text.valid
                  : copy.text.noCurrentAuthority,
              ),
            ),
          );
        }
        const audit = el("details", undefined, "technical");
        audit.append(
          el("summary", copy.text.reviewedMaterialIdentity),
          el(
            "pre",
            JSON.stringify(
              {
                review_case_id: review.id,
                submission_id: review.submission_id,
                submission_digest: review.submission_digest,
                acceptance_policy_revision: review.policy_revision,
                latest_decision_id: review.latest_decision_id,
              },
              null,
              2,
            ),
          ),
        );
        card.append(audit);
        section.append(card);
      }
      if (!entity.current_contract_id && !caseIDs.length)
        section.append(
          el(
            "p",
            copy.text
              .noReservedExecutionOrReviewForThisTaskAcquisitionIsExplicit,
          ),
        );
      if (entity.pending_review_case_id)
        section.append(
          el(
            "p",
            copy.text
              .theExecutorSubmissionWasAcceptedThisTaskAwaitsIndependentRev88610774,
          ),
        );
      const instructions = el("details", undefined, "signed-next-step");
      instructions.append(el("summary", copy.text.executeOrReviewWithAProfile));
      instructions.append(
        el(
          "p",
          copy.text
            .useYourAuthorizedProfileOnTheHostTheBrowserNeverStoresAPriva48737f91,
        ),
      );
      if (
        entity.lifecycle !== "done" &&
        entity.lifecycle !== "cancelled" &&
        !entity.pending_review_case_id
      )
        instructions.append(
          cliHandoff(
            copy.text.executor,
            `wosctl --profile executor work checkout ${entity.id} --version ${entity.version}${entity.correction_review_case_id ? ` --previous-review ${entity.correction_review_case_id}` : ""}
wosctl --profile executor work show CONTRACT_UUID --for-agent
wosctl --profile executor work finish CONTRACT_UUID`,
          ),
        );
      if (entity.pending_review_case_id && pendingReviewVersion !== undefined)
        instructions.append(
          cliHandoff(
            copy.text.reviewer,
            `wosctl --profile reviewer review checkout ${entity.pending_review_case_id} --version ${pendingReviewVersion}
wosctl --profile reviewer review show REVIEW_CONTRACT_UUID --for-agent
wosctl --profile reviewer review finish REVIEW_CONTRACT_UUID`,
          ),
        );
      instructions.append(
        el(
          "p",
          copy.text
            .afterAnUncertainResponseUseWorkRecoverOrReviewRecoverWithThefd343e16,
        ),
      );
      section.append(instructions);
    } catch (error) {
      if (current())
        section.append(
          el(
            "p",
            message("couldNotLoadSignedState", error.message),
            "contract-notice",
          ),
        );
    }
  }

  return {
    contractSection,
    commandAvailableInUI,
    refreshNamespaceProtocol,
    renderProtocolBanner,
    bindProtocolEvents,
  };
}
