package httptransport

import (
	"encoding/json"
	"fmt"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWorkspaceTypedPreparationErrorIsNotVersionConflict(t *testing.T) {
	response := httptest.NewRecorder()
	err := d.NewDetailedError(d.ErrorCodePreconditionFailed, "Add a success condition", d.ErrorDetails{ReasonCode: "missing_required_criterion", FieldPaths: []string{"criteria"}, Remedies: []d.ErrorRemedy{{ActionID: "add_criterion"}}})
	writeError(response, httptest.NewRequest(http.MethodPost, "/api/v1/commands/activate_outcome", nil), fmt.Errorf("wrapped: %w", err))
	var body struct {
		Error struct {
			Code    string         `json:"code"`
			Details d.ErrorDetails `json:"details"`
		} `json:"error"`
	}
	if e := json.Unmarshal(response.Body.Bytes(), &body); e != nil {
		t.Fatal(e)
	}
	if body.Error.Code != "precondition_failed" || body.Error.Details.ReasonCode != "missing_required_criterion" {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	if response.Code == http.StatusPreconditionFailed {
		t.Fatal("domain preparation was classified as optimistic edit conflict")
	}
}
