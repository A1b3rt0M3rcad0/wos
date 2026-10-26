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

// Tasks use the validated source projection rather than a display string or
// historical pointer. Terminal lifecycle always wins, even with review history.
func taskVerificationStage(lifecycle string, verification TaskVerificationView) string {
	if lifecycle == "done" || lifecycle == "cancelled" {
		return workspaceKanbanStage(lifecycle, "")
	}
	if verification.Pending && verification.SourceID != nil && (verification.SourceKind == "task_delivery" || verification.SourceKind == "signed_review_case") {
		return "verification"
	}
	if taskVerificationCorrectionPending(verification) {
		return "todo"
	}
	return workspaceKanbanStage(lifecycle, "")
}

// Signed changes-requested keeps its historical execution lifecycle. The ended
// authority is not an expired live execution, and correction has not yet begun.
func taskVerificationCorrectionPending(v TaskVerificationView) bool {
	return v.CorrectionRequired && !v.Pending && v.Lifecycle != "done" && v.Lifecycle != "cancelled" &&
		(v.ExecutorAuthority == "none" || v.ExecutorAuthority == "ended")
}
