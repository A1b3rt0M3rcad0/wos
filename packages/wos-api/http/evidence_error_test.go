package httptransport

import (
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEvidenceValidationHasNonretryableClientStatus(t *testing.T) {
	response := httptest.NewRecorder()
	writeError(response, httptest.NewRequest(http.MethodPost, "/api/v1/commands/return_signed_work", nil), domain.NewError(domain.ErrorCodeEvidence, "measurement details require evidence_type measurement"))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid documentary material reported as server failure: %d", response.Code)
	}
}
