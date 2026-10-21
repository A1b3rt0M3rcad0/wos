package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/google/jsonschema-go/jsonschema"
)

func operationalSchemaID(n byte) domain.ID {
	const prefix = "0199ed00-0000-7000-8000-0000000000"
	const hex = "0123456789abcdef"
	return domain.MustParseID(prefix + string([]byte{hex[n/16], hex[n%16]}))
}

func operationalSchemaScope() domain.Scope {
	return domain.Scope{NamespaceID: operationalSchemaID(1), OutcomeID: operationalSchemaID(2)}
}

func resolveOperationalSchema(t *testing.T, schema map[string]any) *jsonschema.Resolved {
	t.Helper()
	raw, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	var typed jsonschema.Schema
	if err := json.Unmarshal(raw, &typed); err != nil {
		t.Fatal(err)
	}
	resolved, err := typed.Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func operationalWire(t *testing.T, command any) map[string]any {
	t.Helper()
	raw, err := Encode(command)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func operationalLegacySubmission() application.SubmitTaskForReviewCommand {
	claim := operationalSchemaID(4)
	return application.SubmitTaskForReviewCommand{
		Scope: operationalSchemaScope(), WorkItemID: operationalSchemaID(3), ExpectedWorkItemVersion: domain.Version(math.MaxUint64), ClaimID: &claim, FencingToken: math.MaxUint64,
		LegacyMaterial: &domain.TaskDeliveryMaterial{Summary: "The requested result is ready for independent review"},
	}
}

func operationalContractSubmission() application.SubmitTaskForReviewCommand {
	contract, version := operationalSchemaID(5), domain.Version(math.MaxUint64)
	return application.SubmitTaskForReviewCommand{
		Scope: operationalSchemaScope(), WorkItemID: operationalSchemaID(3), ExpectedWorkItemVersion: version,
		ContractID: &contract, Authority: &application.ContractAuthority{ExecutionID: operationalSchemaID(6), FencingToken: domain.FencingToken(math.MaxUint64), SpecDigest: "sha256:" + strings.Repeat("a", 64)},
		ExpectedContractVersion: &version, SubmissionMaterial: &domain.WorkResultMaterial{ContractID: contract, WorkItemID: operationalSchemaID(3), SpecDigest: "sha256:" + strings.Repeat("a", 64), Summary: "Exact contracts_v1 submission"},
	}
}

func TestOperationalNativeSubmissionSchemaRequiresExactlyOneActualAuthority(t *testing.T) {
	typ := reflect.TypeFor[application.SubmitTaskForReviewCommand]()
	validator := resolveOperationalSchema(t, Schema(typ))
	legacy, contract := operationalWire(t, operationalLegacySubmission()), operationalWire(t, operationalContractSubmission())
	for name, input := range map[string]map[string]any{"legacy": legacy, "contracts_v1": contract} {
		if err := validator.Validate(input); err != nil {
			t.Fatalf("%s valid submission rejected: %v", name, err)
		}
	}
	for _, tc := range []struct {
		name   string
		base   application.SubmitTaskForReviewCommand
		change func(map[string]any)
	}{
		{"neither_authority", application.SubmitTaskForReviewCommand{Scope: operationalSchemaScope(), WorkItemID: operationalSchemaID(3), ExpectedWorkItemVersion: 1}, func(map[string]any) {}},
		{"mixed_authorities", operationalLegacySubmission(), func(v map[string]any) {
			v["contract_id"], v["authority"], v["expected_contract_version"], v["submission_material"] = contract["contract_id"], contract["authority"], contract["expected_contract_version"], contract["submission_material"]
		}},
		{"missing_inline_material", operationalLegacySubmission(), func(v map[string]any) { delete(v, "legacy_material") }},
		{"zero_legacy_fence", operationalLegacySubmission(), func(v map[string]any) { v["fencing_token"] = "0" }},
		{"missing_contract_version", operationalContractSubmission(), func(v map[string]any) { delete(v, "expected_contract_version") }},
		{"legacy_supersedes_submission", operationalLegacySubmission(), func(v map[string]any) { v["supersedes_submission_id"] = operationalSchemaID(7).String() }},
		{"unknown_field", operationalContractSubmission(), func(v map[string]any) { v["actor_ref"] = map[string]any{"id": "client authority"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := operationalWire(t, tc.base)
			tc.change(input)
			if err := validator.Validate(input); err == nil {
				t.Fatal("invalid authority/material combination passed the public schema")
			}
		})
	}
	// contracts_v1 accepts no legacy authority; an explicitly zero omitted fence
	// is equivalent to its zero Go field, rather than a live legacy lease.
	contract["fencing_token"] = "0"
	if err := validator.Validate(contract); err != nil {
		t.Fatal(err)
	}
}

func TestOperationalNewCountersRoundTripAsExactDecimalStrings(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	for _, command := range []any{
		operationalLegacySubmission(), operationalContractSubmission(),
		application.AcceptTaskDeliveryCommand{Scope: operationalSchemaScope(), WorkItemID: operationalSchemaID(3), ExpectedWorkItemVersion: math.MaxUint64, DeliveryID: operationalSchemaID(8), ExpectedDeliveryDigest: digest, ExpectedDeliveryRound: math.MaxUint64, Reason: "Reviewed the exact submitted round"},
		application.RequestTaskDeliveryChangesCommand{Scope: operationalSchemaScope(), WorkItemID: operationalSchemaID(3), ExpectedWorkItemVersion: math.MaxUint64, DeliveryID: operationalSchemaID(8), ExpectedDeliveryDigest: digest, ExpectedDeliveryRound: math.MaxUint64, Reason: "A concrete correction is required", Findings: []string{"Correct the documented behavior"}},
		application.WithdrawTaskDeliveryCommand{Scope: operationalSchemaScope(), WorkItemID: operationalSchemaID(3), ExpectedWorkItemVersion: math.MaxUint64, DeliveryID: operationalSchemaID(8), ExpectedDeliveryDigest: digest, ExpectedDeliveryRound: math.MaxUint64, Reason: "Replan this exact undecided delivery"},
	} {
		t.Run(reflect.TypeOf(command).Name(), func(t *testing.T) {
			raw, err := Encode(command)
			if err != nil {
				t.Fatal(err)
			}
			wire := operationalWire(t, command)
			for _, field := range []string{"expected_work_item_version", "expected_delivery_round", "fencing_token", "expected_contract_version"} {
				if value, exists := wire[field]; exists && value != "18446744073709551615" {
					t.Fatalf("%s lost exact uint64 precision: %s", field, raw)
				}
			}
			typ := reflect.TypeOf(command)
			validator := resolveOperationalSchema(t, Schema(typ))
			if err := validator.Validate(wire); err != nil {
				t.Fatal(err)
			}
			normalized, err := Normalize(raw, typ)
			if err != nil {
				t.Fatal(err)
			}
			decoded := reflect.New(typ)
			if err := Decode(normalized, decoded.Interface()); err != nil || !reflect.DeepEqual(decoded.Elem().Interface(), command) {
				t.Fatalf("exact counter round trip differs: %s %v", normalized, err)
			}
			for _, invalid := range []any{"0", "01", "18446744073709551616", "99999999999999999999", float64(9007199254740992)} {
				wire["expected_work_item_version"] = invalid
				if validator.Validate(wire) == nil {
					t.Fatalf("invalid version accepted: %v", invalid)
				}
			}
		})
	}
	// Reusing the historical WorkItem and material types does not change their
	// existing numeric JSON counters or any signed/canonical representation.
	legacy := Schema(reflect.TypeFor[application.ClaimWorkItemCommand]())["properties"].(map[string]any)
	if legacy["expected_version"].(map[string]any)["type"] != "integer" {
		t.Fatal("historical lease version schema changed")
	}
	result := application.TaskDeliveryResult{ExpectedWorkItemVersion: math.MaxUint64, ExpectedContractVersion: math.MaxUint64,
		WorkItem: domain.WorkItem{ID: operationalSchemaID(3), Scope: operationalSchemaScope(), Version: 9007199254740993, Lifecycle: domain.WorkItemLifecycleTodo, Priority: domain.PriorityNormal},
		Delivery: domain.TaskDelivery{ID: operationalSchemaID(8), Scope: operationalSchemaScope(), WorkItemID: operationalSchemaID(3),
			Authority: domain.TaskDeliveryAuthority{Kind: domain.TaskDeliveryAuthorityLegacyLease, PrincipalID: "executor", Actor: domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "executor"}}},
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]json.RawMessage
	_ = json.Unmarshal(raw, &values)
	if string(values["expected_work_item_version"]) != `"18446744073709551615"` || string(values["expected_contract_version"]) != `"18446744073709551615"` || !strings.Contains(string(values["work_item"]), `"version":9007199254740993`) {
		t.Fatal("new exact counters or historical nested encoding changed", string(raw))
	}
}

func TestOperationalObservedRevisionsAreRequiredNonnullWithoutChangingNullableSetters(t *testing.T) {
	zero, version := domain.OutcomeRevision(0), domain.Version(math.MaxUint64)
	commands := []any{
		application.SetObjectiveCompletionModeCommand{Scope: operationalSchemaScope(), ObjectiveID: operationalSchemaID(3), Mode: domain.ObjectiveCompletionModeManual, ExpectedVersion: version, ExpectedOutcomeRevision: &zero, Reason: "Explicit completion configuration"},
		application.ConfigureOutcomeOperationalProfileCommand{Scope: operationalSchemaScope(), ExpectedVersion: version, ExpectedOutcomeRevision: &zero, Reason: "Explicit operational migration"},
		application.ReportIssueWithBlockersCommand{Scope: operationalSchemaScope(), ObjectiveID: operationalSchemaID(3), Title: "Current problem", Severity: domain.IssueSeverityMajor, ObservedOutcomeRevision: &zero, Blockers: []application.IssueBlockerInput{{BlockedRef: domain.EntityRef{Scope: operationalSchemaScope(), Kind: domain.EntityKindWorkItem, ID: operationalSchemaID(4)}}}},
	}
	for _, command := range commands {
		t.Run(reflect.TypeOf(command).Name(), func(t *testing.T) {
			typ := reflect.TypeOf(command)
			field := "expected_outcome_revision"
			if typ == reflect.TypeFor[application.ReportIssueWithBlockersCommand]() {
				field = "observed_outcome_revision"
			}
			wire := operationalWire(t, command)
			validator := resolveOperationalSchema(t, Schema(typ))
			if err := validator.Validate(wire); err != nil {
				t.Fatal("revision zero is a valid initial Outcome snapshot", err)
			}
			for _, missing := range []bool{false, true} {
				if missing {
					delete(wire, field)
				} else {
					wire[field] = nil
				}
				raw, _ := json.Marshal(wire)
				if validator.Validate(wire) == nil {
					t.Fatal("schema accepted missing/null optimistic revision")
				}
				if _, err := Normalize(raw, typ); err == nil {
					t.Fatal("normalizer accepted missing/null optimistic revision")
				}
			}
		})
	}
	nilRevision := application.ConfigureOutcomeOperationalProfileCommand{Scope: operationalSchemaScope(), ExpectedVersion: version, Reason: "Missing observed snapshot"}
	if _, err := Encode(nilRevision); err == nil {
		t.Fatal("encoder emitted a nil required revision as a string")
	}
	setter := application.SetWorkItemObjectiveCommand{Scope: operationalSchemaScope(), WorkItemID: operationalSchemaID(3), ExpectedVersion: version}
	raw, err := Encode(setter)
	if err != nil || !strings.Contains(string(raw), `"objective_id":null`) {
		t.Fatal("explicit nullable setter semantics changed", string(raw), err)
	}
}

func TestOperationalMaterialAndFindingsSchemasMatchImplementedBounds(t *testing.T) {
	submission := resolveOperationalSchema(t, Schema(reflect.TypeFor[application.SubmitTaskForReviewCommand]()))
	for _, tc := range []struct {
		name   string
		change func(map[string]any)
	}{
		{"oversized_summary", func(m map[string]any) { m["summary"] = strings.Repeat("x", 16385) }},
		{"too_many_evidence", func(m map[string]any) {
			ids := make([]string, 501)
			for i := range ids {
				ids[i] = fmt.Sprintf("0199ed01-0000-7000-8000-%012x", i+1)
			}
			m["evidence_ids"] = ids
		}},
		{"unversioned_artifact", func(m map[string]any) {
			m["artifacts"] = []any{map[string]any{"artifact_id": operationalSchemaID(8).String()}}
		}},
		{"empty_criterion_evidence", func(m map[string]any) {
			m["criterion_evidence"] = []any{map[string]any{"criterion_id": operationalSchemaID(9).String(), "criterion_revision": 1, "evidence_ids": []any{}}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire := operationalWire(t, operationalLegacySubmission())
			tc.change(wire["legacy_material"].(map[string]any))
			if submission.Validate(wire) == nil {
				t.Fatal("invalid material passed schema")
			}
		})
	}
	wire := operationalWire(t, operationalLegacySubmission())
	wire["legacy_material"].(map[string]any)["artifacts"] = []any{map[string]any{"artifact_id": operationalSchemaID(8).String(), "checksum": "sha256:exact-source"}}
	if err := submission.Validate(wire); err != nil {
		t.Fatal("an exact checksum alone is a valid artifact source", err)
	}
	evidenceIDs := make([]string, 500)
	for i := range evidenceIDs {
		evidenceIDs[i] = fmt.Sprintf("0199ed01-0000-7000-8000-%012x", i+1)
	}
	wire["legacy_material"].(map[string]any)["evidence_ids"] = evidenceIDs
	if err := submission.Validate(wire); err != nil {
		t.Fatal("the implemented 500-Evidence boundary was narrowed", err)
	}
	changes := application.RequestTaskDeliveryChangesCommand{Scope: operationalSchemaScope(), WorkItemID: operationalSchemaID(3), ExpectedWorkItemVersion: 1, DeliveryID: operationalSchemaID(8), ExpectedDeliveryDigest: "sha256:" + strings.Repeat("a", 64), ExpectedDeliveryRound: 1, Reason: "Explain the requested correction", Findings: []string{"Correct the behavior"}}
	validator := resolveOperationalSchema(t, Schema(reflect.TypeOf(changes)))
	for _, findings := range []any{nil, []string{}, []string{strings.Repeat("x", 4097)}, make([]string, 101)} {
		wire := operationalWire(t, changes)
		wire["findings"] = findings
		if validator.Validate(wire) == nil {
			t.Fatal("invalid findings passed schema")
		}
	}
}

func TestOperationalCatalogPublishesCurrentPermissionsAndTransportWithoutGeneration(t *testing.T) {
	catalog := newCatalog(nil)
	register[application.SubmitTaskForReviewCommand](catalog, "submit_task_for_review", func(context.Context, domain.CommandContext, application.SubmitTaskForReviewCommand) (any, error) {
		return nil, nil
	})
	register[application.AcceptTaskDeliveryCommand](catalog, "accept_task_delivery", func(context.Context, domain.CommandContext, application.AcceptTaskDeliveryCommand) (any, error) {
		return nil, nil
	})
	register[application.RequestTaskDeliveryChangesCommand](catalog, "request_task_delivery_changes", func(context.Context, domain.CommandContext, application.RequestTaskDeliveryChangesCommand) (any, error) {
		return nil, nil
	})
	register[application.WithdrawTaskDeliveryCommand](catalog, "withdraw_task_delivery", func(context.Context, domain.CommandContext, application.WithdrawTaskDeliveryCommand) (any, error) {
		return nil, nil
	})
	register[application.SetObjectiveCompletionModeCommand](catalog, "set_objective_completion_mode", func(context.Context, domain.CommandContext, application.SetObjectiveCompletionModeCommand) (any, error) {
		return nil, nil
	})
	if len(catalog.Descriptors()) != 5 {
		t.Fatal("five authored command contracts were not registered")
	}
	for _, descriptor := range catalog.Descriptors() {
		if !descriptor.Authorization.PrimaryPermission.Valid() || !descriptor.Authorization.RecheckedOnExecution || descriptor.Description == "" || !descriptor.Transport.AuthenticatedIdentityRequired || descriptor.Transport.ExecutionAuthorityGranted || descriptor.Transport.UnsignedFallback {
			t.Fatalf("catalog metadata misstates authority: %+v", descriptor)
		}
		if len(descriptor.Transport.Channels) != 3 {
			t.Fatal("public command channels are incomplete")
		}
	}
	accept := catalog.descriptors["accept_task_delivery"].Authorization
	if !reflect.DeepEqual(accept.AlwaysRequired, []ports.Permission{ports.PermissionWorkReviewDecide, ports.PermissionTaskAccept, ports.PermissionConclusionWrite}) {
		t.Fatalf("native acceptance requirements differ from Core: %+v", accept)
	}
	signed := commandDescriptor("return_signed_work", reflect.TypeFor[application.ReturnSignedWorkCommand]())
	if signed.Transport.Authority != "signed_host" || signed.Transport.UnsignedFallback || len(signed.Authorization.ConditionalRequirements) < 4 {
		t.Fatal("signed host/conditional requirements were lost")
	}
	for _, descriptor := range NewCatalog(nil, nil).Descriptors() {
		if !descriptor.Authorization.PrimaryPermission.Valid() {
			t.Fatal("a discovered command lacks an authorization contract", descriptor.Name)
		}
	}
}

type unregisteredOperationalCommand struct{}

func TestOperationalCatalogNeverDiscoversACommandWithoutCoreAuthorization(t *testing.T) {
	catalog := newCatalog(nil)
	defer func() {
		if recover() == nil {
			t.Fatal("an unregistered command bypassed the required authorization contract")
		}
		if len(catalog.Descriptors()) != 0 {
			t.Fatal("an unauthorized command appeared in public discovery")
		}
	}()
	register[unregisteredOperationalCommand](catalog, "unregistered_operational", func(context.Context, domain.CommandContext, unregisteredOperationalCommand) (any, error) {
		return nil, nil
	})
}

func TestOperationalResultAndReadRegistriesDescribeImplementedWireAndRedactSecrets(t *testing.T) {
	results := WorkspaceCommandResults()
	for _, name := range []string{"submit_task_for_review", "accept_task_delivery", "request_task_delivery_changes", "withdraw_task_delivery", "set_objective_completion_mode"} {
		if results[name] == nil {
			t.Fatal("implemented result is missing", name)
		}
	}
	result := results["submit_task_for_review"]["anyOf"].([]any)[0].(map[string]any)
	value := result["properties"].(map[string]any)["value"].(map[string]any)["properties"].(map[string]any)
	if value["expected_work_item_version"].(map[string]any)["type"] != "string" || value["work_item"].(map[string]any)["properties"].(map[string]any)["version"].(map[string]any)["type"] != "integer" {
		t.Fatal("result contract changed historical nested versions or lost new exact counters")
	}
	reads := OperationalReadSchemas()
	for _, name := range []string{"TaskVerificationView", "ObjectiveCompletionReport", "DependencyChangesPreview", "DependencyImpactPage", "CredentialManagementPage", "CredentialRevocationImpact", "OperationalErrorDetails"} {
		if reads[name] == nil {
			t.Fatal("implemented read contract is missing", name)
		}
	}
	credential := reads["CredentialAdministrativeRecord"]["properties"].(map[string]any)["credential"].(map[string]any)["properties"].(map[string]any)
	for _, secret := range []string{"Digest", "digest", "parent_digest", "token"} {
		if _, exists := credential[secret]; exists {
			t.Fatal("credential schema publishes secret material", secret)
		}
	}
}
