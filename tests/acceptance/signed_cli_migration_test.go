package acceptance_test

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strconv"
	"testing"

	cli "github.com/A1b3rt0M3rcad0/wos/packages/wos-cli"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func signedCLILegacyMigrationJourney(t *testing.T, destination, origin string, scope d.Scope, token string) {
	t.Helper()
	ctx := context.Background()
	source := t.TempDir()
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	run := func(root string, args ...string) cli.Output {
		t.Helper()
		var out, diagnostic bytes.Buffer
		code := cli.Run(ctx, append(args, "--workspace", root, "--output", "json"), &out, &diagnostic)
		if code != 0 {
			t.Fatalf("migration CLI %v exit %d: %s %s", args, code, out.String(), diagnostic.String())
		}
		var result cli.Output
		must(json.Unmarshal(out.Bytes(), &result))
		return result
	}
	operator, e := sdk.New(origin, token, nil)
	must(e)
	task, e := operator.CreateWorkItem(ctx, "migration-legacy-task-0001", a.CreateWorkItemCommand{Scope: scope, Title: "Preserve a legacy draft", Priority: d.PriorityNormal, Lifecycle: d.WorkItemLifecycleTodo})
	must(e)
	run(source, "init", "--server", origin, "--namespace", scope.NamespaceID.String(), "--outcome", scope.OutcomeID.String(), "--credential-env", "WOS_CLI_CUTOVER_TOKEN")
	acquired := run(source, "work", "checkout", task.Value.ID.String(), "--version", strconv.FormatUint(uint64(task.Value.Version), 10))
	if acquired.ContractID.IsZero() {
		t.Fatal("legacy CLI acquisition omitted original contract")
	}
	old, e := cli.OpenWorkspace(source)
	must(e)
	defer old.Close()
	dir := filepath.Join(".wos/work", task.Value.ID.String(), acquired.ContractID.String())
	var document cli.ContractDocument
	must(old.ReadDocument(filepath.Join(dir, "contract.yaml"), &document))
	var draft cli.ResultDraft
	must(old.ReadDocument(filepath.Join(dir, "result.yaml"), &draft))
	draft.Material.Summary = "Unconfirmed legacy draft retained by migration"
	must(old.WriteDocument(filepath.Join(dir, "result.yaml"), draft))
	original, e := old.ReadV2(filepath.Join(dir, "result.yaml"))
	must(e)
	dry := run(source, "workspace", "migrate", "--to", "2", "--profile", "operator", "--dry-run")
	var inventory cli.MigrationInventoryV2
	raw, e := json.Marshal(dry.Data)
	must(e)
	must(json.Unmarshal(raw, &inventory))
	if len(inventory.UnresolvedIntentions) != 0 || len(inventory.UnmaterializedAcquisitions) != 0 {
		t.Fatal("confirmed legacy materialization was not recognized")
	}
	_, e = operator.RevokeWorkContract(ctx, "migration-revoke-legacy-0001", a.RevokeWorkContractCommand{Scope: scope, ContractID: document.Contract.ID, ExpectedContractVersion: document.Contract.Version, Reason: "retire old execution authority before signed activation"})
	must(e)
	run(source, "workspace", "migrate", "--to", "2", "--profile", "operator", "--destination", destination)
	run(source, "workspace", "migrate", "--to", "2", "--profile", "operator", "--destination", destination)
	fresh, e := cli.OpenWorkspace(destination)
	must(e)
	defer fresh.Close()
	target := filepath.Join(".wos/profiles/operator/contract", acquired.ContractID.String()+".yaml")
	var imported cli.LegacyUnsignedFileV2
	copied, e := fresh.ReadV2(target)
	must(e)
	must(cli.DecodeV2Document(copied, &imported))
	if imported.Protocol != "legacy_unsigned" || imported.Contract.Contract.SignedBinding != nil || imported.Result.Material.Summary != draft.Material.Summary {
		t.Fatal("migration lost a draft or fabricated signed authority")
	}
	after, e := old.ReadV2(filepath.Join(dir, "result.yaml"))
	must(e)
	if signing.Digest(after) != signing.Digest(original) {
		t.Fatal("migration changed original legacy draft")
	}
	shown := run(destination, "work", "show", acquired.ContractID.String(), "--profile", "operator", "--for-agent")
	if shown.RequiresAction == "" {
		t.Fatal("legacy record was shown as operational signed authority")
	}
	listed := run(destination, "work", "list", "--profile", "operator")
	encoded, e := json.Marshal(listed.Data)
	must(e)
	if !bytes.Contains(encoded, []byte("legacy_unsigned")) {
		t.Fatal("legacy import disappeared from bounded work list")
	}

	var rejected, diagnostic bytes.Buffer
	code := cli.Run(ctx, []string{"work", "sign", acquired.ContractID.String(), "--workspace", destination, "--profile", "operator", "--output", "json"}, &rejected, &diagnostic)
	if code == 0 || !bytes.Contains(rejected.Bytes(), []byte("legacy_unsigned")) {
		t.Fatalf("unsigned migration authorized a signed return: %d %s %s", code, rejected.String(), diagnostic.String())
	}
	preserved, e := fresh.ReadV2(target)
	must(e)
	if !bytes.Equal(preserved, copied) {
		t.Fatal("rejected mutation changed imported draft")
	}
	current, e := fresh.LoadProfileV2("operator")
	must(e)
	if len(current.Local.PendingOperations) != 0 {
		t.Fatal("verified conversion retained pending operation")
	}
}
