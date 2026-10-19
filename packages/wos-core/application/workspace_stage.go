package application

// A presentation stage is not a persisted lifecycle or a new review workflow.
// Only actual review facts can move nonterminal work into verification.
func workspaceKanbanStage(lifecycle, display string) string {
	switch lifecycle {
	case "done", "achieved":
		return "accepted"
	case "cancelled", "abandoned", "failed":
		return "cancelled"
	}
	if display == "awaiting_review" {
		return "verification"
	}
	if lifecycle == "in_progress" {
		return "in_progress"
	}
	return "todo"
}
