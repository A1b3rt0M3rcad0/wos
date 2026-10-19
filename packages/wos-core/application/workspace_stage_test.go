package application

import "testing"

func TestWorkspaceStagePreservesTerminalAndReviewMeaning(t *testing.T) {
	for _, c := range []struct{ lifecycle, display, want string }{
		{"done", "awaiting_review", "accepted"}, {"achieved", "", "accepted"},
		{"cancelled", "awaiting_review", "cancelled"}, {"in_progress", "awaiting_review", "verification"},
		{"in_progress", "corrections_requested", "in_progress"}, {"planned", "planned", "todo"},
		{"todo", "blocked", "todo"}, {"in_progress", "in_progress", "in_progress"},
	} {
		if got := workspaceKanbanStage(c.lifecycle, c.display); got != c.want {
			t.Errorf("%s/%s: %s != %s", c.lifecycle, c.display, got, c.want)
		}
	}
}
