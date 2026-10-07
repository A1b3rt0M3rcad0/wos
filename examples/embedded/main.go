// This trusted, in-process host supplies identity, time and IDs. It owns
// authentication and authorization; remote callers should use the server instead.
package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

type clock struct{}

func (clock) Now() time.Time { return time.Now().UTC() }

type ids struct{}

func (ids) NewID() (domain.ID, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	millis := uint64(time.Now().UnixMilli())
	for i := 5; i >= 0; i-- {
		raw[i] = byte(millis)
		millis >>= 8
	}
	raw[6] = raw[6]&0x0f | 0x70
	raw[8] = raw[8]&0x3f | 0x80
	return domain.ParseID(fmt.Sprintf("%x-%x-%x-%x-%x", raw[:4], raw[4:6], raw[6:8], raw[8:10], raw[10:]))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	service, err := application.NewService(memory.New(), clock{}, ids{})
	if err != nil {
		return err
	}
	ctx := context.Background()
	commandContext := func(key string) (domain.CommandContext, error) {
		id, err := (ids{}).NewID()
		return domain.CommandContext{
			PrincipalID: "trusted-example-human",
			Actor:       domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "example", ID: "reviewer"},
			CommandID:   id, IdempotencyKey: key,
		}, err
	}
	cc, err := commandContext("embedded-create-0001")
	if err != nil {
		return err
	}
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID: domain.MustParseID("0199d140-0000-7000-8000-000000000001"),
		Title:       "Embedded public Core", DesiredState: "The host records and certifies a verified result", Priority: domain.PriorityNormal,
	})
	if err != nil {
		return err
	}
	outcome := created.Value
	cc, err = commandContext("embedded-criterion-0001")
	if err != nil {
		return err
	}
	criterion, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner: outcome.Ref(), ExpectedVersion: outcome.Version, Title: "Host verified the public Core journey", Required: true, VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		return err
	}
	current, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		return err
	}
	cc, err = commandContext("embedded-activate-0001")
	if err != nil {
		return err
	}
	active, err := service.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{Scope: outcome.Scope(), ExpectedVersion: current.Value.Version})
	if err != nil {
		return err
	}
	cc, err = commandContext("embedded-attest-0001")
	if err != nil {
		return err
	}
	_, err = service.AttestCriterion(ctx, cc, application.AttestCriterionCommand{
		Owner: outcome.Ref(), ExpectedVersion: active.Value.Version, CriterionID: criterion.Value.ID,
		CriterionRevision: criterion.Value.Revision, Result: domain.AssessmentResultMet, Rationale: "The trusted host verified the required condition",
	})
	if err != nil {
		return err
	}
	current, err = service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		return err
	}
	cc, err = commandContext("embedded-achieve-0001")
	if err != nil {
		return err
	}
	achieved, err := service.AchieveOutcome(ctx, cc, application.AchieveOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: current.Value.Version, Reason: "The required criterion was assessed by the host",
	})
	if err != nil {
		return err
	}
	if achieved.Value.Lifecycle != domain.OutcomeLifecycleAchieved || achieved.Value.CurrentConclusion == nil {
		return fmt.Errorf("the host did not receive the certified conclusion")
	}
	fmt.Printf("lifecycle=%s revision=%d certified=true\n", achieved.Value.Lifecycle, achieved.OutcomeRevision)
	return nil
}
