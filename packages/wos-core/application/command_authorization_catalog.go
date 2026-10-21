package application

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

// CommandAuthorizationRequirement is the stable, presentation-safe part of a
// command's authorization contract. Conditional permissions are resolved from
// the command payload with CommandAuthorizationPermissions.
type CommandAuthorizationRequirement struct {
	PrimaryPermission ports.Permission `json:"permission"`
}

type commandAuthorizationEntry struct {
	primary         ports.Permission
	additional      func(any) []ports.Permission
	resolveResource commandAuthorizationResourceResolver
}

type commandAuthorizationResource struct {
	NamespaceID domain.ID
	Scope       domain.Scope
	Owner       *domain.EntityRef
}

type commandAuthorizationResourceResolver func(any) (commandAuthorizationResource, error)

// commandAuthorizationCatalog is intentionally an exact-name registry. Adding
// a command requires an explicit authorization decision; there is no
// prefix/substring inference and no default permission.
var commandAuthorizationCatalog = func() map[string]commandAuthorizationEntry {
	catalog := map[string]commandAuthorizationEntry{
		"AbandonOutcome":                     {primary: ports.PermissionOutcomeWrite},
		"AcceptDecision":                     {primary: ports.PermissionRecordsWrite},
		"AchieveObjective":                   {primary: ports.PermissionObjectiveConclude},
		"AchieveOutcome":                     {primary: ports.PermissionConclusionWrite},
		"AcquireNextSignedReviewContract":    {primary: ports.PermissionTaskReview},
		"AcquireNextSignedWorkContract":      {primary: ports.PermissionTaskClaim},
		"AcquireNextWorkContract":            {primary: ports.PermissionTaskClaim},
		"AcquireSignedReviewContract":        {primary: ports.PermissionTaskReview},
		"AcquireSignedWorkContract":          {primary: ports.PermissionTaskClaim},
		"AcquireWorkContract":                {primary: ports.PermissionTaskClaim},
		"ActivateOutcome":                    {primary: ports.PermissionOutcomeWrite},
		"ActivateRoadmapRevision":            {primary: ports.PermissionPlanningWrite},
		"ActivateWorkItem":                   {primary: ports.PermissionTaskActivate},
		"AddCriterion":                       {primary: ports.PermissionCriterionDefine},
		"AddDependency":                      {primary: ports.PermissionDependencyWrite},
		"AdministrativeCancelWorkItem":       {primary: ports.PermissionWorkAdminCancel},
		"AdministrativeCompleteWorkItem":     {primary: ports.PermissionWorkAdminComplete},
		"AppendEntityNote":                   {primary: ports.PermissionRecordsWrite},
		"AppendWorkActivity":                 {primary: ports.PermissionRecordsWrite},
		"ArchiveOutcome":                     {primary: ports.PermissionOutcomeWrite},
		"ArchiveRoadmap":                     {primary: ports.PermissionPlanningWrite},
		"AttestCriterion":                    {primary: ports.PermissionCriterionAssess},
		"CancelBlocker":                      {primary: ports.PermissionBlockerCancel},
		"CancelObjective":                    {primary: ports.PermissionObjectiveCancel},
		"CancelWorkItem":                     {primary: ports.PermissionTaskCancel},
		"ClaimWorkItem":                      {primary: ports.PermissionTaskClaim},
		"CompleteWorkItem":                   {primary: ports.PermissionTaskAccept},
		"ConfigureOutcomeOperationalProfile": {primary: ports.PermissionPolicyManage},
		"ConfigureTrigger":                   {primary: ports.PermissionIntegrationWrite},
		"CreateBlocker":                      {primary: ports.PermissionBlockerCreate},
		"CreateEvidenceLink":                 {primary: ports.PermissionRecordsWrite},
		"CreateIssue":                        {primary: ports.PermissionIssueCreate},
		"CreateObjective":                    {primary: ports.PermissionObjectiveCreate, additional: createObjectiveAuthorizationPermissions},
		"CreateOutcome":                      {primary: ports.PermissionOutcomeWrite, additional: createOutcomeAuthorizationPermissions},
		"CreatePlanItems":                    {primary: ports.PermissionPlanningWrite, additional: createPlanItemAuthorizationPermissions},
		"CreateRoadmap":                      {primary: ports.PermissionPlanningWrite},
		"CreateWorkItem":                     {primary: ports.PermissionTaskCreate, additional: createWorkItemAuthorizationPermissions},
		"DeactivateRoadmapRevision":          {primary: ports.PermissionPlanningWrite},
		"DeferWorkItem":                      {primary: ports.PermissionTaskDefer},
		"DiscardRoadmapDraft":                {primary: ports.PermissionPlanningWrite},
		"FailOutcome":                        {primary: ports.PermissionOutcomeWrite},
		"FinalizeWorkContract":               {primary: ports.PermissionTaskAccept},
		"InterveneSignedReviewCase":          {primary: ports.PermissionTaskIntervene},
		"InvestigateIssue":                   {primary: ports.PermissionIssueInvestigate},
		"LinkExternalReference":              {primary: ports.PermissionOutcomeWrite},
		"MarkIssueDuplicate":                 {primary: ports.PermissionIssueResolve},
		"MarkIssueWontFix":                   {primary: ports.PermissionIssueResolve},
		"OpenRoadmapDraft":                   {primary: ports.PermissionPlanningWrite},
		"ProposeDecision":                    {primary: ports.PermissionRecordsWrite},
		"ProposeDecisionInContext":           {primary: ports.PermissionRecordsWrite},
		"PublishRoadmapDraft":                {primary: ports.PermissionPlanningWrite},
		"ReclaimWorkItem":                    {primary: ports.PermissionTaskClaim},
		"ReconcileExpiredWorkContracts":      {primary: ports.PermissionTaskIntervene},
		"RecordCriterionAssessment":          {primary: ports.PermissionCriterionAssess, additional: criterionAssessmentAuthorizationPermissions},
		"RedeliverDelivery":                  {primary: ports.PermissionIntegrationWrite},
		"RegisterArtifact":                   {primary: ports.PermissionRecordsWrite},
		"RegisterArtifactInContext":          {primary: ports.PermissionRecordsWrite},
		"RegisterEvidence":                   {primary: ports.PermissionRecordsWrite},
		"RegisterEvidenceWithLinks":          {primary: ports.PermissionRecordsWrite},
		"RejectDecision":                     {primary: ports.PermissionRecordsWrite},
		"ReleaseWorkItem":                    {primary: ports.PermissionTaskRelease},
		"RemoveDependency":                   {primary: ports.PermissionDependencyWrite},
		"RemoveExternalReference":            {primary: ports.PermissionOutcomeWrite},
		"RenewSignedReviewContract":          {primary: ports.PermissionTaskReview},
		"RenewSignedWorkContract":            {primary: ports.PermissionTaskRenew},
		"RenewWorkContract":                  {primary: ports.PermissionTaskRenew},
		"RenewWorkItemLease":                 {primary: ports.PermissionTaskRenew},
		"ReopenIssue":                        {primary: ports.PermissionIssueReopen},
		"ReopenObjective":                    {primary: ports.PermissionObjectiveReopen},
		"ReopenOutcome":                      {primary: ports.PermissionConclusionWrite},
		"ReopenRoadmap":                      {primary: ports.PermissionPlanningWrite},
		"ReopenWorkItem":                     {primary: ports.PermissionTaskReopen},
		"ReplaceExternalContext":             {primary: ports.PermissionOutcomeWrite},
		"ReplaceRoadmapDraft":                {primary: ports.PermissionPlanningWrite},
		"ReportIssueWithBlocker":             {primary: ports.PermissionIssueCreate, additional: reportIssueWithBlockerPermissions},
		"ResolveBlocker":                     {primary: ports.PermissionBlockerRelease},
		"ResolveIssue":                       {primary: ports.PermissionIssueResolve},
		"ResolveIssueAndBlockers":            {primary: ports.PermissionIssueResolve, additional: resolveIssueAndBlockersPermissions},
		"ResolveSignedReturnIntent":          {primary: ports.PermissionStateRead},
		"ResumeSignedReviewContract":         {primary: ports.PermissionTaskReview},
		"ResumeSignedWorkContract":           {primary: ports.PermissionTaskRenew},
		"ResumeWorkContract":                 {primary: ports.PermissionTaskRenew},
		"RetireCriterion":                    {primary: ports.PermissionCriterionDefine},
		"RetractEvidence":                    {primary: ports.PermissionRecordsWrite},
		"RetractEvidenceLink":                {primary: ports.PermissionRecordsWrite},
		"ReturnSignedReview":                 {primary: ports.PermissionTaskReview, additional: signedReviewDecisionAuthorizationPermissions},
		"ReturnSignedWork":                   {primary: ports.PermissionTaskDeliver, additional: signedWorkReturnAuthorizationPermissions},
		"RevokeSignedContract":               {primary: ports.PermissionWorkContractRevoke},
		"RevokeWorkContract":                 {primary: ports.PermissionWorkContractRevoke},
		"ReviseCriterion":                    {primary: ports.PermissionCriterionDefine},
		"SetNamespaceWorkProtocol":           {primary: ports.PermissionPolicyManage},
		"SetObjectiveOwners":                 {primary: ports.PermissionObjectiveUpdate},
		"SetObjectiveParent":                 {primary: ports.PermissionObjectiveUpdate},
		"SetOutcomeOwners":                   {primary: ports.PermissionOutcomeWrite},
		"SetTriggerEnabled":                  {primary: ports.PermissionIntegrationWrite},
		"SetWorkItemAssignees":               {primary: ports.PermissionTaskAssign},
		"SetWorkItemExecutionSpec":           {primary: ports.PermissionTaskUpdateDefinition},
		"SetWorkItemNotBefore":               {primary: ports.PermissionTaskUpdateDefinition},
		"SetWorkItemObjective":               {primary: ports.PermissionTaskUpdateDefinition},
		"StartObjective":                     {primary: ports.PermissionObjectiveStart},
		"SubmitWorkResult":                   {primary: ports.PermissionTaskDeliver},
		"SupersedeDecision":                  {primary: ports.PermissionRecordsWrite},
		"SyncWorkContract":                   {primary: ports.PermissionTaskDeliver},
		"UnarchiveOutcome":                   {primary: ports.PermissionOutcomeWrite},
		"UpdateBlockerDescription":           {primary: ports.PermissionWorkWrite},
		"UpdateDecision":                     {primary: ports.PermissionRecordsWrite},
		"UpdateIssue":                        {primary: ports.PermissionIssueUpdate},
		"UpdateObjective":                    {primary: ports.PermissionObjectiveUpdate},
		"UpdateOutcome":                      {primary: ports.PermissionOutcomeWrite},
		"UpdateTrigger":                      {primary: ports.PermissionIntegrationWrite},
		"UpdateWorkItem":                     {primary: ports.PermissionTaskUpdateDefinition},
		"WithdrawArtifact":                   {primary: ports.PermissionRecordsWrite},

		// Application commands not currently included in the HTTP command catalog
		// retain the permissions previously inferred for their exact names.
		"ApplyDependencyChanges":      {primary: ports.PermissionDependencyWrite},
		"AssignIssue":                 {primary: ports.PermissionIssueAssign},
		"RelateIssueWorkItem":         {primary: ports.PermissionIssueUpdate},
		"RemoveIssueWorkItemRelation": {primary: ports.PermissionIssueUpdate},
		"ReportIssueWithBlockers":     {primary: ports.PermissionIssueCreate, additional: reportIssueWithBlockerPermissions},
		"StartIssueInvestigation":     {primary: ports.PermissionIssueInvestigate},
		"UnassignIssue":               {primary: ports.PermissionIssueAssign},
	}
	for name, entry := range catalog {
		entry.resolveResource = resolveCommandResourceFields
		catalog[name] = entry
	}
	for name, resolver := range map[string]commandAuthorizationResourceResolver{
		"ReturnSignedWork":   resolveSignedWorkReturnResource,
		"ReturnSignedReview": resolveSignedReviewReturnResource,
	} {
		entry := catalog[name]
		entry.resolveResource = resolver
		catalog[name] = entry
	}
	return catalog
}()

func commandAuthorizationRequirement(name string) (CommandAuthorizationRequirement, bool) {
	entry, ok := commandAuthorizationCatalog[name]
	if !ok || !entry.primary.Valid() {
		return CommandAuthorizationRequirement{}, false
	}
	return CommandAuthorizationRequirement{PrimaryPermission: entry.primary}, true
}

// CommandAuthorizationForPresentation exposes the registered primary
// requirement. The bool is false for names that are not in the command catalog.
func CommandAuthorizationForPresentation(name string) (CommandAuthorizationRequirement, bool) {
	return commandAuthorizationRequirement(name)
}

// CommandAuthorizationPermissions returns the primary permission followed by
// any payload-dependent requirements, de-duplicated in stable order.
func CommandAuthorizationPermissions(name string, command any) ([]ports.Permission, bool) {
	entry, ok := commandAuthorizationCatalog[name]
	if !ok || !entry.primary.Valid() || commandNameForAuthorization(command) != name {
		return nil, false
	}
	permissions := []ports.Permission{entry.primary}
	if entry.additional != nil {
		permissions = append(permissions, entry.additional(command)...)
	}
	seen := make(map[ports.Permission]struct{}, len(permissions))
	unique := permissions[:0]
	for _, permission := range permissions {
		if !permission.Valid() {
			return nil, false
		}
		if _, ok := seen[permission]; ok {
			continue
		}
		seen[permission] = struct{}{}
		unique = append(unique, permission)
	}
	return unique, true
}

func commandNameForAuthorization(command any) string {
	t := reflect.TypeOf(command)
	if t == nil {
		return ""
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	return strings.TrimSuffix(t.Name(), "Command")
}

func commandAuthorizationResourceFor(name string, command any) (commandAuthorizationResource, bool, error) {
	entry, ok := commandAuthorizationCatalog[name]
	if !ok || entry.resolveResource == nil || commandNameForAuthorization(command) != name {
		return commandAuthorizationResource{}, false, nil
	}
	resource, err := entry.resolveResource(command)
	if err != nil {
		return commandAuthorizationResource{}, true, err
	}
	if err := resource.NamespaceID.Validate(); err != nil {
		return commandAuthorizationResource{}, true, domain.WrapError(domain.ErrorCodeInvalidArgument, "command namespace cannot be resolved", err)
	}
	return resource, true, nil
}

func resolveCommandResourceFields(command any) (commandAuthorizationResource, error) {
	value := reflect.ValueOf(command)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return commandAuthorizationResource{}, domain.NewError(domain.ErrorCodeInvalidArgument, "command cannot be nil")
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return commandAuthorizationResource{}, domain.NewError(domain.ErrorCodeInvalidArgument, "command must be a struct")
	}
	resource := commandAuthorizationResource{}
	if field := value.FieldByName("NamespaceID"); field.IsValid() && field.Type() == reflect.TypeFor[domain.ID]() {
		resource.NamespaceID = field.Interface().(domain.ID)
	}
	if field := value.FieldByName("Scope"); field.IsValid() && field.Type() == reflect.TypeFor[domain.Scope]() {
		scope := field.Interface().(domain.Scope)
		if !resource.NamespaceID.IsZero() && !scope.NamespaceID.IsZero() && resource.NamespaceID != scope.NamespaceID {
			return commandAuthorizationResource{}, domain.NewError(domain.ErrorCodeInvalidScope, "command resource namespace fields disagree")
		}
		resource.Scope = scope
		resource.NamespaceID = resource.Scope.NamespaceID
	}
	if field := value.FieldByName("Owner"); field.IsValid() && field.Type() == reflect.TypeFor[domain.EntityRef]() {
		owner := field.Interface().(domain.EntityRef)
		if !resource.NamespaceID.IsZero() && !owner.NamespaceID.IsZero() && resource.NamespaceID != owner.NamespaceID ||
			!resource.Scope.OutcomeID.IsZero() && !owner.Scope.OutcomeID.IsZero() && resource.Scope.OutcomeID != owner.Scope.OutcomeID {
			return commandAuthorizationResource{}, domain.NewError(domain.ErrorCodeInvalidScope, "command scope and owner fields disagree")
		}
		resource.Owner = &owner
		resource.Scope = owner.Scope
		resource.NamespaceID = owner.NamespaceID
	}
	return resource, nil
}

func resolveSignedWorkReturnResource(command any) (commandAuthorizationResource, error) {
	var cmd ReturnSignedWorkCommand
	switch value := command.(type) {
	case ReturnSignedWorkCommand:
		cmd = value
	case *ReturnSignedWorkCommand:
		if value == nil {
			return commandAuthorizationResource{}, domain.NewError(domain.ErrorCodeInvalidArgument, "command cannot be nil")
		}
		cmd = *value
	default:
		return commandAuthorizationResource{}, domain.NewError(domain.ErrorCodeInvalidArgument, "signed work return command has an invalid type")
	}
	var hint signing.WorkReturnPayload[SignedReturnMaterial]
	if err := signedRequestHint(cmd.Envelope, &hint); err != nil {
		return commandAuthorizationResource{}, err
	}
	scope, err := signedRequestScope(hint.RequestBinding)
	if err != nil {
		return commandAuthorizationResource{}, err
	}
	return commandAuthorizationResource{NamespaceID: scope.NamespaceID, Scope: scope}, nil
}

func resolveSignedReviewReturnResource(command any) (commandAuthorizationResource, error) {
	var cmd ReturnSignedReviewCommand
	switch value := command.(type) {
	case ReturnSignedReviewCommand:
		cmd = value
	case *ReturnSignedReviewCommand:
		if value == nil {
			return commandAuthorizationResource{}, domain.NewError(domain.ErrorCodeInvalidArgument, "command cannot be nil")
		}
		cmd = *value
	default:
		return commandAuthorizationResource{}, domain.NewError(domain.ErrorCodeInvalidArgument, "signed review return command has an invalid type")
	}
	var hint signing.ReviewReturnPayload[SignedReviewMaterial]
	if err := signedRequestHint(cmd.Envelope, &hint); err != nil {
		return commandAuthorizationResource{}, err
	}
	scope, err := signedRequestScope(hint.RequestBinding)
	if err != nil {
		return commandAuthorizationResource{}, err
	}
	return commandAuthorizationResource{NamespaceID: scope.NamespaceID, Scope: scope}, nil
}

func createPlanItemAuthorizationPermissions(command any) []ports.Permission {
	var items []PlanItemInput
	switch cmd := command.(type) {
	case CreatePlanItemsCommand:
		items = cmd.Items
	case *CreatePlanItemsCommand:
		if cmd != nil {
			items = cmd.Items
		}
	default:
		return nil
	}
	permissions := make([]ports.Permission, 0, len(items)*3)
	for _, item := range items {
		if item.Kind == domain.EntityKindObjective {
			permissions = append(permissions, ports.PermissionObjectiveCreate)
		}
		if item.Kind == domain.EntityKindWorkItem {
			permissions = append(permissions, ports.PermissionTaskCreate)
		}
		if len(item.InitialCriteria) > 0 {
			permissions = append(permissions, ports.PermissionCriterionDefine)
		}
	}
	return permissions
}

func signedWorkReturnAuthorizationPermissions(command any) []ports.Permission {
	var envelope signing.Envelope
	switch cmd := command.(type) {
	case ReturnSignedWorkCommand:
		envelope = cmd.Envelope
	case *ReturnSignedWorkCommand:
		if cmd != nil {
			envelope = cmd.Envelope
		}
	default:
		return nil
	}
	var hint signing.WorkReturnPayload[json.RawMessage]
	if signedRequestHint(envelope, &hint) != nil || hint.CompletionIntent != "direct" {
		return nil
	}
	return []ports.Permission{ports.PermissionTaskAccept}
}

func signedReviewDecisionAuthorizationPermissions(command any) []ports.Permission {
	var envelope signing.Envelope
	switch cmd := command.(type) {
	case ReturnSignedReviewCommand:
		envelope = cmd.Envelope
	case *ReturnSignedReviewCommand:
		if cmd != nil {
			envelope = cmd.Envelope
		}
	default:
		return nil
	}
	var hint signing.ReviewReturnPayload[json.RawMessage]
	if signedRequestHint(envelope, &hint) != nil {
		return nil
	}
	switch hint.Decision {
	case "approved":
		return []ports.Permission{ports.PermissionTaskAccept}
	case "changes_requested":
		return []ports.Permission{ports.PermissionTaskRequestChanges}
	default:
		return nil
	}
}

func createOutcomeAuthorizationPermissions(command any) []ports.Permission {
	switch cmd := command.(type) {
	case CreateOutcomeCommand:
		if len(cmd.InitialCriteria) > 0 {
			return []ports.Permission{ports.PermissionCriterionDefine}
		}
	case *CreateOutcomeCommand:
		if cmd != nil && len(cmd.InitialCriteria) > 0 {
			return []ports.Permission{ports.PermissionCriterionDefine}
		}
	}
	return nil
}

func createObjectiveAuthorizationPermissions(command any) []ports.Permission {
	switch cmd := command.(type) {
	case CreateObjectiveCommand:
		if len(cmd.InitialCriteria) > 0 {
			return []ports.Permission{ports.PermissionCriterionDefine}
		}
	case *CreateObjectiveCommand:
		if cmd != nil && len(cmd.InitialCriteria) > 0 {
			return []ports.Permission{ports.PermissionCriterionDefine}
		}
	}
	return nil
}

func reportIssueWithBlockerPermissions(any) []ports.Permission {
	return []ports.Permission{ports.PermissionBlockerCreate}
}

func resolveIssueAndBlockersPermissions(any) []ports.Permission {
	return []ports.Permission{ports.PermissionBlockerRelease}
}

func createWorkItemAuthorizationPermissions(command any) []ports.Permission {
	switch cmd := command.(type) {
	case CreateWorkItemCommand:
		if len(cmd.InitialCriteria) > 0 {
			return []ports.Permission{ports.PermissionCriterionDefine}
		}
	case *CreateWorkItemCommand:
		if cmd != nil && len(cmd.InitialCriteria) > 0 {
			return []ports.Permission{ports.PermissionCriterionDefine}
		}
	}
	return nil
}

func criterionAssessmentAuthorizationPermissions(command any) []ports.Permission {
	switch cmd := command.(type) {
	case RecordCriterionAssessmentCommand:
		if cmd.Result == domain.AssessmentResultWaived {
			return []ports.Permission{ports.PermissionCriterionWaive}
		}
	case *RecordCriterionAssessmentCommand:
		if cmd != nil && cmd.Result == domain.AssessmentResultWaived {
			return []ports.Permission{ports.PermissionCriterionWaive}
		}
	}
	return nil
}

func (s *Service) authorizeCommandAdditionalPermissions(ctx context.Context, uow ports.UnitOfWork, cc domain.CommandContext, scope domain.Scope, command any) error {
	name := commandNameForAuthorization(command)
	permissions, ok := CommandAuthorizationPermissions(name, command)
	if !ok {
		return domain.NewError(domain.ErrorCodeForbidden, "command authorization requirement is not registered")
	}
	if len(permissions) <= 1 {
		return nil
	}
	additional := permissions[1:]
	if _, transactional := s.authorizer.(ports.TransactionalAuthorizer); !transactional {
		// Waivers are preauthorized in RecordCriterionAssessment before the
		// transaction; only transactional adapters need the replay-time check.
		filtered := additional[:0]
		for _, permission := range additional {
			if permission != ports.PermissionAssessmentWaive {
				filtered = append(filtered, permission)
			}
		}
		additional = filtered
	}
	return s.authorizeCreationPermissions(ctx, uow, cc, scope, additional...)
}
