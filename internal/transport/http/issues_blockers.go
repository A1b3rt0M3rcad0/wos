package httptransport

import (
	"net/http"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func (h *Handler) createIssue(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request createIssueRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	affected, err := requestEntityRefs(scope, request.AffectedRefs)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.CreateIssue(r.Context(), cc, application.CreateIssueCommand{
		Scope: scope, Title: request.Title, Description: request.Description,
		Severity: request.Severity, AffectedRefs: affected,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindIssue, result.Value.ID, result.Value.Version)
	w.Header().Set("Location", issueLocation(h, scope, result.Value.ID))
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) getIssue(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.issueRequestScope(w, r)
	if !ok {
		return
	}
	result, err := h.service.GetIssue(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindIssue, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listIssues(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items, revision, err := h.service.ListIssues(r.Context(), scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, collectionResponse[domain.Issue]{Items: items, OutcomeRevision: revision})
}

func (h *Handler) updateIssue(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.issueRequestScope(w, r)
	if !ok {
		return
	}
	var request updateIssueRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindIssue, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var affected *[]domain.EntityRef
	if request.AffectedRefs != nil {
		values, err := requestEntityRefs(scope, *request.AffectedRefs)
		if err != nil {
			writeError(w, r, err)
			return
		}
		affected = &values
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.UpdateIssue(r.Context(), cc, application.UpdateIssueCommand{
		Scope: scope, IssueID: id, ExpectedVersion: expected,
		Title: request.Title, Description: request.Description, Severity: request.Severity, AffectedRefs: affected,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindIssue, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) investigateIssue(w http.ResponseWriter, r *http.Request) {
	h.issueVersionAction(w, r, func(scope domain.Scope, id domain.ID, expected domain.Version, cc domain.CommandContext) (application.MutationResult[domain.Issue], error) {
		return h.service.InvestigateIssue(r.Context(), cc, application.InvestigateIssueCommand{Scope: scope, IssueID: id, ExpectedVersion: expected})
	})
}

func (h *Handler) reopenIssue(w http.ResponseWriter, r *http.Request) {
	h.issueVersionAction(w, r, func(scope domain.Scope, id domain.ID, expected domain.Version, cc domain.CommandContext) (application.MutationResult[domain.Issue], error) {
		return h.service.ReopenIssue(r.Context(), cc, application.ReopenIssueCommand{Scope: scope, IssueID: id, ExpectedVersion: expected})
	})
}

func (h *Handler) resolveIssue(w http.ResponseWriter, r *http.Request) {
	h.issueResolutionAction(w, r, func(scope domain.Scope, id domain.ID, expected domain.Version, summary string, cc domain.CommandContext) (application.MutationResult[domain.Issue], error) {
		return h.service.ResolveIssue(r.Context(), cc, application.ResolveIssueCommand{
			Scope: scope, IssueID: id, ExpectedVersion: expected, ResolutionSummary: summary,
		})
	})
}

func (h *Handler) markIssueWontFix(w http.ResponseWriter, r *http.Request) {
	h.issueResolutionAction(w, r, func(scope domain.Scope, id domain.ID, expected domain.Version, summary string, cc domain.CommandContext) (application.MutationResult[domain.Issue], error) {
		return h.service.MarkIssueWontFix(r.Context(), cc, application.MarkIssueWontFixCommand{
			Scope: scope, IssueID: id, ExpectedVersion: expected, ResolutionSummary: summary,
		})
	})
}

func (h *Handler) markIssueDuplicate(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.issueRequestScope(w, r)
	if !ok {
		return
	}
	var request duplicateIssueRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindIssue, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	duplicateID, err := domain.ParseID(request.DuplicateOfIssueID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.MarkIssueDuplicate(r.Context(), cc, application.MarkIssueDuplicateCommand{
		Scope: scope, IssueID: id, ExpectedVersion: expected, DuplicateOfID: duplicateID,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindIssue, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) reportIssueWithBlocker(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request reportIssueWithBlockerRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	affected, err := requestEntityRefs(scope, request.Issue.AffectedRefs)
	if err != nil {
		writeError(w, r, err)
		return
	}
	blocked, err := relationEndpoint(scope, request.Blocker.BlockedRef)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.ReportIssueWithBlocker(r.Context(), cc, application.ReportIssueWithBlockerCommand{
		Scope: scope, Title: request.Issue.Title, Description: request.Issue.Description,
		Severity: request.Issue.Severity, AffectedRefs: affected, BlockedRef: blocked,
		BlockerDescription: request.Blocker.Description, Propagation: request.Blocker.Propagation,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) resolveIssueAndBlockers(w http.ResponseWriter, r *http.Request) {
	scope, issueID, ok := h.issueRequestScope(w, r)
	if !ok {
		return
	}
	var request resolveIssueAndBlockersRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	issueVersion, err := requiredBodyVersion(request.ExpectedIssueVersion, "expected_issue_version")
	if err != nil {
		writeError(w, r, err)
		return
	}
	blockers := make([]application.BlockerResolution, 0, len(request.Blockers))
	for _, item := range request.Blockers {
		id, err := domain.ParseID(item.BlockerID)
		if err != nil {
			writeError(w, r, err)
			return
		}
		version, err := requiredBodyVersion(item.ExpectedVersion, "blockers[].expected_version")
		if err != nil {
			writeError(w, r, err)
			return
		}
		blockers = append(blockers, application.BlockerResolution{
			BlockerID: id, ExpectedVersion: version, ResolutionSummary: item.ResolutionSummary, ReleaseConfirmed: item.ReleaseConfirmed,
		})
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.ResolveIssueAndBlockers(r.Context(), cc, application.ResolveIssueAndBlockersCommand{
		Scope: scope, IssueID: issueID, IssueExpectedVersion: issueVersion,
		IssueResolutionSummary: request.IssueResolutionSummary, Blockers: blockers,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) createBlocker(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request createBlockerRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	blocked, err := relationEndpoint(scope, request.BlockedRef)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var cause *domain.EntityRef
	if request.CauseRef != nil {
		value, err := relationEndpoint(scope, *request.CauseRef)
		if err != nil {
			writeError(w, r, err)
			return
		}
		cause = &value
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.CreateBlocker(r.Context(), cc, application.CreateBlockerCommand{
		Scope: scope, BlockedRef: blocked, CauseRef: cause, ExternalCause: request.ExternalCause,
		Description: request.Description, Propagation: request.Propagation,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindBlocker, result.Value.ID, result.Value.Version)
	w.Header().Set("Location", blockerLocation(h, scope, result.Value.ID))
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) getBlocker(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.blockerRequestScope(w, r)
	if !ok {
		return
	}
	result, err := h.service.GetBlocker(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindBlocker, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listBlockers(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items, revision, err := h.service.ListBlockers(r.Context(), scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, collectionResponse[domain.Blocker]{Items: items, OutcomeRevision: revision})
}

func (h *Handler) updateBlocker(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.blockerRequestScope(w, r)
	if !ok {
		return
	}
	var request updateBlockerRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	if request.Description == nil {
		writeError(w, r, domain.NewError(domain.ErrorCodeInvalidArgument, "blocker patch has no editable fields"))
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindBlocker, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.UpdateBlockerDescription(r.Context(), cc, application.UpdateBlockerDescriptionCommand{
		Scope: scope, BlockerID: id, ExpectedVersion: expected, Description: *request.Description,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindBlocker, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) resolveBlocker(w http.ResponseWriter, r *http.Request) {
	h.blockerResolutionAction(w, r, func(scope domain.Scope, id domain.ID, expected domain.Version, summary string, cc domain.CommandContext) (application.MutationResult[domain.Blocker], error) {
		return h.service.ResolveBlocker(r.Context(), cc, application.ResolveBlockerCommand{
			Scope: scope, BlockerID: id, ExpectedVersion: expected, ResolutionSummary: summary,
		})
	})
}

func (h *Handler) cancelBlocker(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.blockerRequestScope(w, r)
	if !ok {
		return
	}
	var request reasonRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindBlocker, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.CancelBlocker(r.Context(), cc, application.CancelBlockerCommand{
		Scope: scope, BlockerID: id, ExpectedVersion: expected, Reason: request.Reason,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindBlocker, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) issueVersionAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(domain.Scope, domain.ID, domain.Version, domain.CommandContext) (application.MutationResult[domain.Issue], error),
) {
	scope, id, ok := h.issueRequestScope(w, r)
	if !ok {
		return
	}
	var request versionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindIssue, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := action(scope, id, expected, cc)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindIssue, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) issueResolutionAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(domain.Scope, domain.ID, domain.Version, string, domain.CommandContext) (application.MutationResult[domain.Issue], error),
) {
	scope, id, ok := h.issueRequestScope(w, r)
	if !ok {
		return
	}
	var request resolutionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindIssue, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := action(scope, id, expected, request.ResolutionSummary, cc)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindIssue, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) blockerResolutionAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(domain.Scope, domain.ID, domain.Version, string, domain.CommandContext) (application.MutationResult[domain.Blocker], error),
) {
	scope, id, ok := h.blockerRequestScope(w, r)
	if !ok {
		return
	}
	var request resolutionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindBlocker, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := action(scope, id, expected, request.ResolutionSummary, cc)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindBlocker, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) issueRequestScope(w http.ResponseWriter, r *http.Request) (domain.Scope, domain.ID, bool) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return domain.Scope{}, domain.ID{}, false
	}
	id, err := parsePathID(r, "issue_id")
	if err != nil {
		writeError(w, r, err)
		return domain.Scope{}, domain.ID{}, false
	}
	return scope, id, true
}

func (h *Handler) blockerRequestScope(w http.ResponseWriter, r *http.Request) (domain.Scope, domain.ID, bool) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return domain.Scope{}, domain.ID{}, false
	}
	id, err := parsePathID(r, "blocker_id")
	if err != nil {
		writeError(w, r, err)
		return domain.Scope{}, domain.ID{}, false
	}
	return scope, id, true
}

func requestEntityRefs(scope domain.Scope, requests []relationEndpointRequest) ([]domain.EntityRef, error) {
	refs := make([]domain.EntityRef, 0, len(requests))
	for _, request := range requests {
		ref, err := relationEndpoint(scope, request)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func requiredBodyVersion(raw *uint64, field string) (domain.Version, error) {
	if raw == nil {
		return 0, &contractError{status: http.StatusPreconditionRequired, code: "precondition_required", message: field + " is required"}
	}
	version := domain.Version(*raw)
	if err := version.Validate(); err != nil {
		return 0, err
	}
	return version, nil
}

func issueLocation(h *Handler, scope domain.Scope, id domain.ID) string {
	return h.prefix + "/namespaces/" + scope.NamespaceID.String() + "/outcomes/" + scope.OutcomeID.String() + "/issues/" + id.String()
}

func blockerLocation(h *Handler, scope domain.Scope, id domain.ID) string {
	return h.prefix + "/namespaces/" + scope.NamespaceID.String() + "/outcomes/" + scope.OutcomeID.String() + "/blockers/" + id.String()
}
