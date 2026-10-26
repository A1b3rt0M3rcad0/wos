package application

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

func TestCommandAuthorizationCatalogResolvesNamespaceScopeAndOwner(t *testing.T) {
	namespace := domain.MustParseID("018f0a10-0000-7000-8000-000000000001")
	outcome := domain.MustParseID("018f0a10-0000-7000-8000-000000000002")
	entity := domain.MustParseID("018f0a10-0000-7000-8000-000000000003")
	scope := domain.Scope{NamespaceID: namespace, OutcomeID: outcome}
	owner := domain.EntityRef{Scope: scope, Kind: domain.EntityKindObjective, ID: entity}

	tests := []struct {
		name    string
		command any
		want    commandAuthorizationResource
	}{
		{
			name:    "CreateOutcome",
			command: CreateOutcomeCommand{NamespaceID: namespace},
			want:    commandAuthorizationResource{NamespaceID: namespace},
		},
		{
			name:    "AchieveOutcome",
			command: AchieveOutcomeCommand{Scope: scope},
			want:    commandAuthorizationResource{NamespaceID: namespace, Scope: scope},
		},
		{
			name:    "AddCriterion",
			command: AddCriterionCommand{Owner: owner},
			want:    commandAuthorizationResource{NamespaceID: namespace, Scope: scope, Owner: &owner},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, registered, err := commandAuthorizationResourceFor(test.name, test.command)
			if err != nil {
				t.Fatal(err)
			}
			if !registered {
				t.Fatal("command has no explicit resource resolver")
			}
			if got.NamespaceID != test.want.NamespaceID || got.Scope != test.want.Scope {
				t.Fatalf("resource = %+v, want %+v", got, test.want)
			}
			if (got.Owner == nil) != (test.want.Owner == nil) || got.Owner != nil && *got.Owner != *test.want.Owner {
				t.Fatalf("owner = %+v, want %+v", got.Owner, test.want.Owner)
			}
		})
	}
}

func TestCommandResourceResolverRejectsConflictingScopeAndOwner(t *testing.T) {
	namespace := domain.MustParseID("018f0a10-0000-7000-8000-000000000021")
	otherOutcome := domain.MustParseID("018f0a10-0000-7000-8000-000000000022")
	ownerOutcome := domain.MustParseID("018f0a10-0000-7000-8000-000000000023")
	ownerID := domain.MustParseID("018f0a10-0000-7000-8000-000000000024")
	command := struct {
		Scope domain.Scope
		Owner domain.EntityRef
	}{
		Scope: domain.Scope{NamespaceID: namespace, OutcomeID: otherOutcome},
		Owner: domain.EntityRef{Scope: domain.Scope{NamespaceID: namespace, OutcomeID: ownerOutcome}, Kind: domain.EntityKindObjective, ID: ownerID},
	}
	if _, err := resolveCommandResourceFields(command); err == nil {
		t.Fatal("conflicting scope and owner were silently resolved")
	}
}

func TestSignedReturnAuthorizationResolversUseEnvelopeScope(t *testing.T) {
	namespace := domain.MustParseID("018f0a10-0000-7000-8000-000000000011")
	outcome := domain.MustParseID("018f0a10-0000-7000-8000-000000000012")
	scope := domain.Scope{NamespaceID: namespace, OutcomeID: outcome}
	binding := signing.RequestBinding{
		Binding: signing.Binding{
			ProtocolVersion: 2,
			ServerID:        "server-1",
			NamespaceID:     namespace.String(),
			OutcomeID:       outcome.String(),
			PrincipalID:     "agent-1",
			SignerKeyID:     "018f0a10-0000-7000-8000-000000000013",
		},
		OperationKind:           "work_return",
		RequestID:               "018f0a10-0000-7000-8000-000000000014",
		IdempotencyKey:          "request-identifier-001",
		ContractID:              "018f0a10-0000-7000-8000-000000000015",
		WorkItemID:              "018f0a10-0000-7000-8000-000000000016",
		ExecutionID:             "018f0a10-0000-7000-8000-000000000017",
		FencingToken:            1,
		SpecDigest:              testSignedDigest("spec"),
		AuthorityDigest:         testSignedDigest("authority"),
		ExpectedContractVersion: 1,
		ExpectedLeaseVersion:    1,
		ExpectedWorkItemVersion: 1,
		PolicyRevision:          1,
	}
	workPayload, err := json.Marshal(signing.WorkReturnPayload[SignedReturnMaterial]{
		RequestBinding:   binding,
		CompletionIntent: "complete",
		Material: SignedReturnMaterial{Result: domain.SignedResultMaterial{
			ContractID: domain.ID(binding.ContractID),
			WorkItemID: domain.ID(binding.WorkItemID),
			SpecDigest: binding.SpecDigest,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	reviewBinding := binding
	reviewBinding.OperationKind = "review_return"
	reviewPayload, err := json.Marshal(signing.ReviewReturnPayload[SignedReviewMaterial]{
		RequestBinding:   reviewBinding,
		ReviewCaseID:     "018f0a10-0000-7000-8000-000000000018",
		SubmissionID:     "018f0a10-0000-7000-8000-000000000019",
		SubmissionDigest: testSignedDigest("submission"),
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		command any
	}{
		{name: "ReturnSignedWork", command: ReturnSignedWorkCommand{Envelope: signing.Envelope{Payload: base64.StdEncoding.EncodeToString(workPayload)}}},
		{name: "ReturnSignedReview", command: ReturnSignedReviewCommand{Envelope: signing.Envelope{Payload: base64.StdEncoding.EncodeToString(reviewPayload)}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, registered, err := commandAuthorizationResourceFor(test.name, test.command)
			if err != nil {
				t.Fatal(err)
			}
			if !registered || got.NamespaceID != namespace || got.Scope != scope {
				t.Fatalf("resource = %+v, registered = %v; want scope %+v", got, registered, scope)
			}
		})
	}
}

func testSignedDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
