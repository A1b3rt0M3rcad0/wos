package commands

import (
	"reflect"
	"strings"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
)

type TransportDescription struct {
	Channels                      []string `json:"channels"`
	AuthenticatedIdentityRequired bool     `json:"authenticated_identity_required"`
	Authority                     string   `json:"authority"`
	UnsignedFallback              bool     `json:"unsigned_fallback"`
	ExecutionAuthorityGranted     bool     `json:"execution_authority_granted"`
}

func commandDescriptor(name string, commandType reflect.Type) Descriptor {
	coreName := strings.TrimSuffix(commandType.Name(), "Command")
	authorization, ok := application.CommandAuthorizationDescriptionForPresentation(coreName)
	if !ok {
		// A missing programmer-owned authorization registration must fail before
		// an executable command is exposed to discovery or any public transport.
		panic("public command has no authorization contract: " + coreName)
	}
	transport := TransportDescription{Channels: []string{"http", "mcp", "go_sdk"}, AuthenticatedIdentityRequired: true, Authority: "authenticated_command"}
	description := "Execute " + name + " through the canonical Application command. An idempotency key is required; reconcile uncertain results with the same intent and never silently replace versions or authority."
	if application.CommandRequiresSignedHostForPresentation(coreName) {
		transport.Authority = "signed_host"
		description += " Use an authorized signed host/profile; the server verifies the existing signed protocol and bindings. There is no unsigned fallback."
	}
	switch coreName {
	case "SubmitTaskForReview":
		transport.Authority = "current_lease_or_contract"
		description = "Deliver the current Task execution for review using exactly one credential-bound legacy lease or contracts_v1 authority/material pair. This ends execution authority and creates an immutable TaskDelivery identified by digest and round. It does not accept the Task. Signed-v2 work uses ReturnSignedWork."
	case "AcceptTaskDelivery":
		transport.Authority = "independent_reviewer"
		description = "Accept one current native TaskDelivery after reviewing the exact digest, round and applicable current assessments. Required reviewer permissions and independence are checked in the same transaction; delivery or linked Evidence alone does not establish acceptance. Signed-v2 review uses ReturnSignedReview."
	case "RequestTaskDeliveryChanges":
		transport.Authority = "independent_reviewer"
		description = "Request actionable changes on one exact current native TaskDelivery. Record the review decision and return the Task to To do without executor authority; a subsequent execution must acquire fresh authority and submit a new round. Signed-v2 review uses ReturnSignedReview."
	case "WithdrawTaskDelivery":
		transport.Authority = "authenticated_credential"
		description = "Withdraw one exact undecided native TaskDelivery using its current digest and round. Append withdrawal provenance and return the Task to To do without execution authority. Withdrawal is not a review decision; Backlog replanning requires a separate explicit DeferWorkItem operation."
	case "SetObjectiveCompletionMode":
		description = "Explicitly configure manual or delegated automatic Objective completion at the observed Objective version and Outcome revision. Record the authenticated delegator and current credential policy; this does not assess a Criterion or bypass completion requirements."
	case "ConfigureOutcomeOperationalProfile":
		description = "Explicitly migrate one existing Outcome to operational_v2 and strict_work using the observed Outcome version/revision. The Application reevaluates the migration gate; this cannot weaken the policy or configure Objective completion delegation."
	}
	return Descriptor{Name: name, Schema: Schema(commandType), Description: description, Authorization: authorization, Transport: transport}
}
