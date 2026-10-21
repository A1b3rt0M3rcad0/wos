package httptransport

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func createHTTPTestObjective(t *testing.T, client *http.Client, outcomeURL, key string) domain.Objective {
	t.Helper()
	created := doJSON[application.MutationResult[domain.Objective]](
		t, client, http.MethodPost, outcomeURL+"/objectives",
		[]byte(`{"title":"Task owner","priority":"normal"}`),
		map[string]string{"Idempotency-Key": key}, http.StatusCreated,
	)
	return created.Value.Value
}

func testWorkItemBody(objectiveID domain.ID, title, lifecycle string) []byte {
	body, _ := json.Marshal(struct {
		ObjectiveID domain.ID `json:"objective_id"`
		Title       string    `json:"title"`
		Priority    string    `json:"priority"`
		Lifecycle   string    `json:"lifecycle"`
	}{ObjectiveID: objectiveID, Title: title, Priority: "normal", Lifecycle: lifecycle})
	return body
}
