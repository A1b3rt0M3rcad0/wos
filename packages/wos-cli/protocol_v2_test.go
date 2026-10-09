package woscli

import (
	"context"
	"encoding/json"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProtocolV2FreezesExactCounterAndGetFailureRemainsPrintable(t *testing.T) {
	project, profile, token := profileFixtureV2(t, "operator")
	command := a.SetNamespaceWorkProtocolCommand{Scope: d.Scope{NamespaceID: project.Scope.NamespaceID, OutcomeID: project.Scope.DefaultOutcomeID}, ExpectedProtocolVersion: 9007199254740995, Phase: d.WorkProtocolSignedDraining, Reason: "retire old writers"}
	raw, e := freezeProtocolV2(command)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(raw), `"expected_protocol_version":"9007199254740995"`) {
		t.Fatal("protocol CAS was not frozen as a decimal string")
	}
	restored, e := decodeProtocolV2(raw)
	if e != nil || restored.ExpectedProtocolVersion != command.ExpectedProtocolVersion {
		t.Fatal("protocol CAS rounded", e)
	}
	if _, e = decodeProtocolV2([]byte(strings.Replace(string(raw), `"9007199254740995"`, `9007199254740995`, 1))); e == nil {
		t.Fatal("unsafe numeric v2 CAS accepted")
	}
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer endpoint.Close()
	client, e := sdk.New(endpoint.URL, string(token), endpoint.Client())
	if e != nil {
		t.Fatal(e)
	}
	workspace, e := OpenWorkspace(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer workspace.Close()
	if e = workspace.CreateV2(".wos/project.yaml", project); e != nil {
		t.Fatal(e)
	}
	result, e := protocolCommandV2(context.Background(), workspace, profile, client, token, a.SigningIdentityView{}, options{args: []string{"protocol", "get"}, values: map[string]string{}})
	if e == nil {
		t.Fatal("failed protocol query reported success")
	}
	result.Error = e.Error()
	if _, e = json.Marshal(result); e != nil {
		t.Fatal("failed query produced an unprintable zero-ID result", e)
	}
}

func TestProtocolConfirmedAcceptanceStillRequiresOriginalTargetAndVersion(t *testing.T) {
	project, _, _ := profileFixtureV2(t, "operator")
	command := a.SetNamespaceWorkProtocolCommand{Scope: d.Scope{NamespaceID: project.Scope.NamespaceID, OutcomeID: project.Scope.DefaultOutcomeID}, ExpectedProtocolVersion: 3, Phase: d.WorkProtocolSignedDraining}
	protocol := d.DefaultWorkProtocol(command.Scope.NamespaceID)
	protocol.Version = 4
	protocol.Phase = command.Phase
	value := sdk.CommandResult[a.NamespaceWorkProtocolResult]{CommandID: project.Connection.ExpectedServerID, Value: a.NamespaceWorkProtocolResult{Protocol: protocol}}
	if e := validateProtocolAcceptanceV2(command, value); e != nil {
		t.Fatal(e)
	}
	value.Value.Protocol.Version = 5
	if validateProtocolAcceptanceV2(command, value) == nil {
		t.Fatal("confirmed response with a different CAS was cleared")
	}
	value.Value.Protocol.Version = 4
	value.CommandID = ""
	if validateProtocolAcceptanceV2(command, value) == nil {
		t.Fatal("response without original command ID was accepted")
	}
}
