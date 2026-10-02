package httptransport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
	"github.com/A1b3rt0M3rcad0/wos/internal/authentication/local"
)

const maxJSONBodyBytes = 1 << 20

type Options struct {
	Prefix         string
	RequestTimeout time.Duration
	Service        *application.Service
	IDs            ports.IDGenerator
	LocalAuth      local.Resolver
}

type Handler struct {
	service        *application.Service
	ids            ports.IDGenerator
	auth           local.Resolver
	requestTimeout time.Duration
	prefix         string
	mux            *http.ServeMux
}

type contextKey string

const correlationKey contextKey = "wos-correlation-id"

func New(options Options) (*Handler, error) {
	if options.Service == nil || options.IDs == nil {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "http service and id generator are required")
	}
	if strings.TrimSpace(options.LocalAuth.Principal()) == "" {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "local auth resolver is required")
	}
	prefix := strings.TrimRight(strings.TrimSpace(options.Prefix), "/")
	if prefix == "" || !strings.HasPrefix(prefix, "/") {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "http prefix must be an absolute path")
	}
	if options.RequestTimeout <= 0 {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "http request timeout must be positive")
	}

	h := &Handler{
		service:        options.Service,
		ids:            options.IDs,
		auth:           options.LocalAuth,
		requestTimeout: options.RequestTimeout,
		prefix:         prefix,
		mux:            http.NewServeMux(),
	}
	h.routes()
	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	correlationID := strings.TrimSpace(r.Header.Get("X-Correlation-ID"))
	if correlationID == "" {
		correlationID = newCorrelationID()
	}
	w.Header().Set("X-Correlation-ID", correlationID)

	ctx, cancel := context.WithTimeout(r.Context(), h.requestTimeout)
	defer cancel()
	ctx = context.WithValue(ctx, correlationKey, correlationID)

	h.mux.ServeHTTP(w, r.WithContext(ctx))
}

func (h *Handler) routes() {
	base := h.prefix + "/namespaces/{namespace_id}/outcomes"

	h.mux.HandleFunc("POST "+base, h.createOutcome)

	outcome := base + "/{outcome_id}"
	h.mux.HandleFunc("GET "+outcome, h.getOutcome)
	h.mux.HandleFunc("GET "+outcome+"/state", h.getOutcomeState)
	h.mux.HandleFunc("POST "+outcome+"/actions/activate", h.activateOutcome)
	h.mux.HandleFunc("POST "+outcome+"/actions/achieve", h.achieveOutcome)
	h.mux.HandleFunc("POST "+outcome+"/actions/fail", h.failOutcome)
	h.mux.HandleFunc("POST "+outcome+"/actions/abandon", h.abandonOutcome)
	h.mux.HandleFunc("POST "+outcome+"/actions/reopen", h.reopenOutcome)
	h.mux.HandleFunc("POST "+outcome+"/actions/archive", h.archiveOutcome)
	h.mux.HandleFunc("POST "+outcome+"/actions/unarchive", h.unarchiveOutcome)

	h.mux.HandleFunc("POST "+outcome+"/criteria", h.addOutcomeCriterion)
	h.mux.HandleFunc("PATCH "+outcome+"/criteria/{criterion_id}", h.reviseOutcomeCriterion)
	h.mux.HandleFunc("POST "+outcome+"/criteria/{criterion_id}/actions/retire", h.retireOutcomeCriterion)
	h.mux.HandleFunc("POST "+outcome+"/criteria/{criterion_id}/assessments", h.assessOutcomeCriterion)

	h.mux.HandleFunc("POST "+outcome+"/objectives", h.createObjective)
	h.mux.HandleFunc("GET "+outcome+"/objectives", h.listObjectives)
	h.mux.HandleFunc("GET "+outcome+"/objectives/{objective_id}", h.getObjective)
	h.mux.HandleFunc("POST "+outcome+"/objectives/{objective_id}/actions/start", h.startObjective)
	h.mux.HandleFunc("POST "+outcome+"/objectives/{objective_id}/actions/achieve", h.achieveObjective)
	h.mux.HandleFunc("POST "+outcome+"/objectives/{objective_id}/actions/cancel", h.cancelObjective)
	h.mux.HandleFunc("POST "+outcome+"/objectives/{objective_id}/actions/reopen", h.reopenObjective)
	h.mux.HandleFunc("POST "+outcome+"/objectives/{objective_id}/criteria", h.addObjectiveCriterion)
	h.mux.HandleFunc("PATCH "+outcome+"/objectives/{objective_id}/criteria/{criterion_id}", h.reviseObjectiveCriterion)
	h.mux.HandleFunc("POST "+outcome+"/objectives/{objective_id}/criteria/{criterion_id}/actions/retire", h.retireObjectiveCriterion)
	h.mux.HandleFunc("POST "+outcome+"/objectives/{objective_id}/criteria/{criterion_id}/assessments", h.assessObjectiveCriterion)

	h.mux.HandleFunc("POST "+outcome+"/work-items", h.createWorkItem)
	h.mux.HandleFunc("GET "+outcome+"/work-items", h.listWorkItems)
	h.mux.HandleFunc("GET "+outcome+"/work-items/{work_item_id}", h.getWorkItem)
	h.mux.HandleFunc("POST "+outcome+"/work-items/{work_item_id}/actions/activate", h.activateWorkItem)
	h.mux.HandleFunc("POST "+outcome+"/work-items/{work_item_id}/actions/defer", h.deferWorkItem)
	h.mux.HandleFunc("POST "+outcome+"/work-items/{work_item_id}/actions/claim", h.claimWorkItem)
	h.mux.HandleFunc("POST "+outcome+"/work-items/{work_item_id}/actions/release", h.releaseWorkItem)
	h.mux.HandleFunc("POST "+outcome+"/work-items/{work_item_id}/actions/complete", h.completeWorkItem)
	h.mux.HandleFunc("POST "+outcome+"/work-items/{work_item_id}/actions/cancel", h.cancelWorkItem)
	h.mux.HandleFunc("POST "+outcome+"/work-items/{work_item_id}/actions/reopen", h.reopenWorkItem)
	h.mux.HandleFunc("POST "+outcome+"/work-items/{work_item_id}/criteria", h.addWorkItemCriterion)
	h.mux.HandleFunc("PATCH "+outcome+"/work-items/{work_item_id}/criteria/{criterion_id}", h.reviseWorkItemCriterion)
	h.mux.HandleFunc("POST "+outcome+"/work-items/{work_item_id}/criteria/{criterion_id}/actions/retire", h.retireWorkItemCriterion)
	h.mux.HandleFunc("POST "+outcome+"/work-items/{work_item_id}/criteria/{criterion_id}/assessments", h.assessWorkItemCriterion)
}

func (h *Handler) commandContext(r *http.Request) (domain.CommandContext, error) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if err := domain.ValidateIdempotencyKey(key); err != nil {
		return domain.CommandContext{}, err
	}
	commandID, err := h.ids.NewID()
	if err != nil {
		return domain.CommandContext{}, err
	}
	return domain.CommandContext{
		PrincipalID:    h.auth.Principal(),
		Actor:          h.auth.Actor(),
		CorrelationID:  correlationID(r.Context()),
		CommandID:      commandID,
		IdempotencyKey: key,
	}, nil
}

func correlationID(ctx context.Context) string {
	value, _ := ctx.Value(correlationKey).(string)
	return value
}

func newCorrelationID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "correlation-unavailable"
	}
	return hex.EncodeToString(raw[:])
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "JSON request body is required")
		}
		return domain.WrapError(domain.ErrorCodeInvalidArgument, "invalid JSON request body", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "request body must contain exactly one JSON value")
	}
	return nil
}

func parseScope(r *http.Request) (domain.Scope, error) {
	namespaceID, err := domain.ParseID(r.PathValue("namespace_id"))
	if err != nil {
		return domain.Scope{}, err
	}
	outcomeID, err := domain.ParseID(r.PathValue("outcome_id"))
	if err != nil {
		return domain.Scope{}, err
	}
	scope := domain.Scope{NamespaceID: namespaceID, OutcomeID: outcomeID}
	return scope, scope.Validate()
}

func parsePathID(r *http.Request, name string) (domain.ID, error) {
	return domain.ParseID(r.PathValue(name))
}

func optionalPathID(value *string) (*domain.ID, error) {
	if value == nil {
		return nil, nil
	}
	id, err := domain.ParseID(*value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func expectedVersion(r *http.Request, body *uint64, kind domain.EntityKind, id domain.ID) (domain.Version, error) {
	var bodyVersion *domain.Version
	if body != nil {
		v := domain.Version(*body)
		if err := v.Validate(); err != nil {
			return 0, err
		}
		bodyVersion = &v
	}

	header := strings.TrimSpace(r.Header.Get("If-Match"))
	var headerVersion *domain.Version
	if header != "" {
		expectedPrefix := "\"" + kind.String() + ":" + id.String() + ":v"
		if !strings.HasPrefix(header, expectedPrefix) || !strings.HasSuffix(header, "\"") {
			return 0, domain.NewError(domain.ErrorCodeVersionConflict, "If-Match does not identify the requested resource")
		}
		raw := strings.TrimSuffix(strings.TrimPrefix(header, expectedPrefix), "\"")
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return 0, domain.NewError(domain.ErrorCodeInvalidVersion, "If-Match contains an invalid version")
		}
		v := domain.Version(value)
		if err := v.Validate(); err != nil {
			return 0, err
		}
		headerVersion = &v
	}

	if headerVersion == nil && bodyVersion == nil {
		return 0, domain.NewError(domain.ErrorCodeInvalidArgument, "If-Match or expected_version is required")
	}
	if headerVersion != nil && bodyVersion != nil && *headerVersion != *bodyVersion {
		return 0, domain.NewError(domain.ErrorCodeVersionConflict, "If-Match and expected_version disagree")
	}
	if headerVersion != nil {
		return *headerVersion, nil
	}
	return *bodyVersion, nil
}

func setETag(w http.ResponseWriter, kind domain.EntityKind, id domain.ID, version domain.Version) {
	w.Header().Set("ETag", fmt.Sprintf(""%s:%s:v%d"", kind, id, version))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "internal server error"
	retryable := false

	if errors.Is(err, context.DeadlineExceeded) {
		status = http.StatusGatewayTimeout
		code = "deadline_exceeded"
		message = "request deadline exceeded"
		retryable = true
	} else if errors.Is(err, context.Canceled) {
		status = http.StatusRequestTimeout
		code = "request_cancelled"
		message = "request was cancelled"
		retryable = true
	} else if domainCode, ok := domain.ErrorCodeOf(err); ok {
		code = string(domainCode)
		message = err.Error()
		switch domainCode {
		case domain.ErrorCodeInvalidID,
			domain.ErrorCodeInvalidScope,
			domain.ErrorCodeInvalidEntityKind,
			domain.ErrorCodeInvalidEntityRef,
			domain.ErrorCodeInvalidActorKind,
			domain.ErrorCodeInvalidActorRef,
			domain.ErrorCodeInvalidVersion,
			domain.ErrorCodeInvalidCommandContext,
			domain.ErrorCodeInvalidArgument,
			domain.ErrorCodeCriterion,
			domain.ErrorCodeAssessment,
			domain.ErrorCodeInvalidEvent,
			domain.ErrorCodeInvalidIdempotencyKey:
			status = http.StatusBadRequest
		case domain.ErrorCodeNotFound:
			status = http.StatusNotFound
		case domain.ErrorCodeVersionConflict:
			status = http.StatusPreconditionFailed
		case domain.ErrorCodeAlreadyExists,
			domain.ErrorCodeInvalidTransition,
			domain.ErrorCodePreconditionFailed,
			domain.ErrorCodeLease,
			domain.ErrorCodeIdempotencyConflict,
			domain.ErrorCodeIdempotencyState:
			status = http.StatusConflict
		case domain.ErrorCodeInvalidConfig:
			status = http.StatusInternalServerError
		}
	}

	writeJSON(w, status, errorEnvelope{Error: errorBody{
		Code:          code,
		Message:       message,
		Retryable:     retryable,
		CorrelationID: correlationID(r.Context()),
	}})
}
