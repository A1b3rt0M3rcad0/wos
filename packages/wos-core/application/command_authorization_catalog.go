package application

import (
	"context"
	"encoding/base64"
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
		"AcceptTaskDelivery":                 {primary: ports.PermissionWorkReviewDecide, additional: acceptTaskDeliveryAuthorizationPermissions},
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
		"RequestTaskDeliveryChanges":         {primary: ports.PermissionWorkReviewDecide, additional: requestTaskDeliveryChangesAuthorizationPermissions},
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
		"SetObjectiveCompletionMode":         {primary: ports.PermissionObjectiveConfigureCompletion},
		"SetObjectiveOwners":                 {primary: ports.PermissionObjectiveUpdate},
		"SetObjectiveParent":                 {primary: ports.PermissionObjectiveUpdate},
		"SetOutcomeOwners":                   {primary: ports.PermissionOutcomeWrite},
		"SetTriggerEnabled":                  {primary: ports.PermissionIntegrationWrite},
		"SetWorkItemAssignees":               {primary: ports.PermissionTaskAssign},
		"SetWorkItemExecutionSpec":           {primary: ports.PermissionTaskUpdateDefinition},
		"SetWorkItemNotBefore":               {primary: ports.PermissionTaskUpdateDefinition},
		"SetWorkItemObjective":               {primary: ports.PermissionTaskUpdateDefinition},
		"StartObjective":                     {primary: ports.PermissionObjectiveStart},
		"SubmitTaskForReview":                {primary: ports.PermissionTaskDeliver},
		"SubmitWorkResult":                   {primary: ports.PermissionTaskDeliver},
		"SupersedeDecision":                  {primary: ports.PermissionRecordsWrite},
		"SyncWorkContract":                   {primary: ports.PermissionTaskDeliver, additional: syncWorkContractAuthorizationPermissions},
		"UnarchiveOutcome":                   {primary: ports.PermissionOutcomeWrite},
		"UpdateBlockerDescription":           {primary: ports.PermissionWorkWrite},
		"UpdateDecision":                     {primary: ports.PermissionRecordsWrite},
		"UpdateIssue":                        {primary: ports.PermissionIssueUpdate},
		"UpdateObjective":                    {primary: ports.PermissionObjectiveUpdate},
		"UpdateOutcome":                      {primary: ports.PermissionOutcomeWrite},
		"UpdateTrigger":                      {primary: ports.PermissionIntegrationWrite},
		"UpdateWorkItem":                     {primary: ports.PermissionTaskUpdateDefinition},
		"WithdrawArtifact":                   {primary: ports.PermissionRecordsWrite},
		"WithdrawTaskDelivery":               {primary: ports.PermissionTaskWithdrawDelivery},

		// Application commands not currently included in the HTTP command catalog
		// retain the permissions previously inferred for their exact names.
		"ApplyDependencyChanges":      {primary: ports.PermissionDependencyWrite},
		"AssignIssue":                 {primary: ports.PermissionIssueAssign},
		"RelateIssueToWorkItem":       {primary: ports.PermissionIssueUpdate},
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
	if signedRequestHint(envelope, &hint) != nil {
		return nil
	}
	permissions := []ports.Permission{ports.PermissionWorkContractReturn}
	material, err := signedWorkAuthorizationMaterial(hint.Material)
	if err != nil {
		return nil
	}
	if hasSignedDocumentaryMaterial(len(material.Artifacts), len(material.Evidence), len(material.EvidenceLinks)) {
		permissions = append(permissions, ports.PermissionRecordsWrite)
	}
	if hint.CompletionIntent == "direct" {
		permissions = append(permissions,
			ports.PermissionTaskAccept,
			ports.PermissionWorkCompleteDirect,
			ports.PermissionCriterionAssess,
			ports.PermissionAssessmentWrite,
			ports.PermissionConclusionWrite,
		)
	}
	for _, assessment := range material.Assessments {
		if assessment.Result == domain.AssessmentResultWaived {
			permissions = append(permissions, ports.PermissionAssessmentWaive)
			break
		}
	}
	return permissions
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
	permissions := []ports.Permission{ports.PermissionWorkReviewDecide}
	material, err := signedWorkAuthorizationMaterial(hint.Material)
	if err != nil {
		return nil
	}
	if hasSignedDocumentaryMaterial(len(material.Artifacts), len(material.Evidence), len(material.EvidenceLinks)) {
		permissions = append(permissions, ports.PermissionRecordsWrite)
	}
	switch hint.Decision {
	case "approved":
		permissions = append(permissions,
			ports.PermissionTaskAccept,
			ports.PermissionCriterionAssess,
			ports.PermissionAssessmentWrite,
			ports.PermissionConclusionWrite,
		)
	case "changes_requested":
		permissions = append(permissions, ports.PermissionTaskRequestChanges)
	}
	for _, assessment := range material.Assessments {
		if assessment.Result == domain.AssessmentResultWaived {
			permissions = append(permissions, ports.PermissionAssessmentWaive)
			break
		}
	}
	return permissions
}

func acceptTaskDeliveryAuthorizationPermissions(any) []ports.Permission {
	return []ports.Permission{ports.PermissionTaskAccept, ports.PermissionConclusionWrite}
}

func requestTaskDeliveryChangesAuthorizationPermissions(any) []ports.Permission {
	return []ports.Permission{ports.PermissionTaskRequestChanges}
}

type signedAuthorizationMaterial struct {
	Artifacts     []json.RawMessage `json:"artifacts"`
	Evidence      []json.RawMessage `json:"evidence"`
	EvidenceLinks []json.RawMessage `json:"evidence_links"`
	Assessments   []struct {
		Result domain.AssessmentResult `json:"result"`
	} `json:"criterion_assessments"`
}

func signedWorkAuthorizationMaterial(raw json.RawMessage) (signedAuthorizationMaterial, error) {
	var material signedAuthorizationMaterial
	if err := json.Unmarshal(raw, &material); err != nil {
		return material, err
	}
	return material, nil
}

func hasSignedDocumentaryMaterial(artifacts, evidence, links int) bool {
	return artifacts+evidence+links > 0
}

func syncWorkContractAuthorizationPermissions(command any) []ports.Permission {
	var cmd SyncWorkContractCommand
	switch value := command.(type) {
	case SyncWorkContractCommand:
		cmd = value
	case *SyncWorkContractCommand:
		if value == nil {
			return nil
		}
		cmd = *value
	default:
		return nil
	}
	if hasSignedDocumentaryMaterial(len(cmd.Artifacts), len(cmd.Evidence), len(cmd.EvidenceLinks)) {
		return []ports.Permission{ports.PermissionRecordsWrite}
	}
	return nil
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

func (s *Service) authorizeComposedCommandPermissions(ctx context.Context, uow ports.UnitOfWork, cc domain.CommandContext, scope domain.Scope, command any, fingerprint string) error {
	name := commandNameForAuthorization(command)
	permissions, ok := CommandAuthorizationPermissions(name, command)
	if !ok {
		return domain.NewError(domain.ErrorCodeForbidden, "command authorization requirement is not registered")
	}
	// Some commands, notably CreateOutcome, are namespace-scoped and do not
	// carry an Outcome Scope yet. Resolve the namespace from the command's
	// explicit authorization resource before checking any conditional
	// permissions (such as criterion:define for initial criteria).
	if scope.NamespaceID.IsZero() {
		resource, registered, err := commandAuthorizationResourceFor(name, command)
		if err != nil {
			return err
		}
		if !registered {
			return domain.NewError(domain.ErrorCodeForbidden, "command authorization requirement is not registered")
		}
		scope.NamespaceID = resource.NamespaceID
	}
	if name == "ReturnSignedWork" && s.requireIdentity {
		permissionsForDirectCompletion, err := s.signedAutoReturnCompletionPermissions(ctx, uow, cc, command, fingerprint)
		if err != nil {
			return err
		}
		permissions = append(permissions, permissionsForDirectCompletion...)
	}
	if len(permissions) <= 1 {
		return nil
	}
	additional := permissions[1:]
	return s.authorizeCreationPermissions(ctx, uow, cc, scope, name, additional...)
}

var signedDirectCompletionPermissions = []ports.Permission{
	ports.PermissionTaskAccept,
	ports.PermissionWorkCompleteDirect,
	ports.PermissionCriterionAssess,
	ports.PermissionAssessmentWrite,
	ports.PermissionConclusionWrite,
}

// An "auto" signed return may complete the Task directly or submit it for
// review. For a replay, derive its requirements from the original durable
// receipt, not today's policy: tightening the floor cannot make an old direct
// acceptance replay disclosure less protected, and a review delivery must not
// inherit completion permissions it never used. For a new request, resolve the
// current signed acceptance floor inside this UOW before idempotency receipts
// can be disclosed. Explicit direct returns are handled by the static payload
// resolver above.
func (s *Service) signedAutoReturnCompletionPermissions(ctx context.Context, uow ports.UnitOfWork, cc domain.CommandContext, command any, fingerprint string) ([]ports.Permission, error) {
	var envelope signing.Envelope
	switch cmd := command.(type) {
	case ReturnSignedWorkCommand:
		envelope = cmd.Envelope
	case *ReturnSignedWorkCommand:
		if cmd == nil {
			return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "signed work return command cannot be nil")
		}
		envelope = cmd.Envelope
	default:
		return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "signed work return command has an invalid type")
	}
	var hint signing.WorkReturnPayload[json.RawMessage]
	if err := signedRequestHint(envelope, &hint); err != nil {
		return nil, err
	}
	if hint.CompletionIntent != "auto" {
		return nil, nil
	}
	scope, err := signedRequestScope(hint.RequestBinding)
	if err != nil {
		return nil, err
	}
	identity, policy, err := s.signedAccess(ctx, uow, scope, ports.PermissionTaskDeliver, domain.ID(hint.SignerKeyID), legacyPermissionForCommand("ReturnSignedWork", ports.PermissionTaskDeliver))
	if err != nil {
		return nil, err
	}
	_, workContracts, err := signedRepository(uow)
	if err != nil {
		return nil, err
	}
	contract, err := workContracts.Get(ctx, scope, domain.ID(hint.ContractID))
	if err != nil {
		return nil, err
	}
	if contract.SignedBinding == nil || contract.SignedBinding.CredentialID != identity.CredentialID || contract.HolderPrincipalID != identity.PrincipalID || contract.WorkItemID != domain.ID(hint.WorkItemID) {
		return nil, domain.NewError(domain.ErrorCodeForbidden, "return does not own this credential authority")
	}
	if cc.IdempotencyKey != "" {
		payload, err := base64.StdEncoding.Strict().DecodeString(envelope.Payload)
		if err != nil {
			return nil, signingError(err)
		}
		if required, replay, err := signedReturnReplayCompletionPermissions(ctx, uow, scope, identity, cc, fingerprint, contract.ID, domain.ID(hint.RequestID), domain.ID(hint.WorkItemID), signing.Digest(payload)); err != nil {
			return nil, err
		} else if replay {
			return required, nil
		}
	}
	// Older idempotency entries may predate durable signed-operation records.
	// The execution contract's terminal completed state is durable evidence that
	// its signed return took the direct-completion branch; delivered contracts
	// remain eligible for ordinary review-delivery replay without these rights.
	if contract.Status == domain.ContractCompleted {
		return append([]ports.Permission(nil), signedDirectCompletionPermissions...), nil
	}
	floor, err := s.signedAcceptanceFloor(ctx, uow, scope, domain.ID(hint.WorkItemID), policy, contract.SignedBinding.AcceptanceFloor)
	if err != nil {
		return nil, err
	}
	if floor != domain.AcceptanceDirect {
		return nil, nil
	}
	return append([]ports.Permission(nil), signedDirectCompletionPermissions...), nil
}

func signedReturnReplayCompletionPermissions(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope, identity Identity, cc domain.CommandContext, fingerprint string, contractID, requestID, workItemID domain.ID, requestDigest string) ([]ports.Permission, bool, error) {
	repository, ok := uow.(ports.SignedOperationUnitOfWork)
	if !ok {
		return nil, false, domain.NewError(domain.ErrorCodeInvalidConfig, "durable signed operation repository required")
	}
	operations := repository.SignedOperations()
	previous, err := operations.Get(ctx, scope.NamespaceID, identity.PrincipalID, cc.IdempotencyKey)
	if err == nil {
		if previous.Scope != scope || previous.PrincipalID != identity.PrincipalID || previous.CredentialID != identity.CredentialID || previous.CommandName != "ReturnSignedWork" || previous.Fingerprint != fingerprint {
			return nil, false, domain.NewError(domain.ErrorCodeIdempotencyConflict, "signed intention belongs to different scope, credential or command")
		}
		permissions, err := signedReturnStoredCompletionPermissions(scope, contractID, requestID, cc.IdempotencyKey, workItemID, requestDigest, previous.Result)
		return permissions, true, err
	}
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeNotFound {
		return nil, false, err
	}
	legacy, err := operations.FindLegacy(ctx, scope.NamespaceID, identity.PrincipalID, cc.IdempotencyKey)
	if err != nil {
		if code, _ := domain.ErrorCodeOf(err); code == domain.ErrorCodeNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	if legacy.CommandName != "ReturnSignedWork" || legacy.Fingerprint != fingerprint {
		return nil, false, domain.NewError(domain.ErrorCodeIdempotencyConflict, "legacy signed intention differs from requested command")
	}
	view, err := legacySignedOperationView(ctx, uow, scope, identity.PrincipalID, cc.IdempotencyKey, legacy)
	if err != nil {
		return nil, false, err
	}
	if view.CredentialID != identity.CredentialID {
		return nil, false, domain.NewError(domain.ErrorCodeIdempotencyConflict, "legacy signed intention belongs to another credential")
	}
	permissions, err := signedReturnStoredCompletionPermissions(scope, contractID, requestID, cc.IdempotencyKey, workItemID, requestDigest, view.Result)
	return permissions, true, err
}

func signedReturnStoredCompletionPermissions(scope domain.Scope, contractID, requestID domain.ID, idempotencyKey string, workItemID domain.ID, requestDigest string, stored domain.StoredCommandResult) ([]ports.Permission, error) {
	var result MutationResult[SignedReturnResult]
	if err := json.Unmarshal(stored.ResponseJSON, &result); err != nil {
		return nil, domain.WrapError(domain.ErrorCodeIdempotencyState, "stored signed return result cannot be decoded", err)
	}
	var receipt signing.ReceiptPayload
	if err := json.Unmarshal(result.Value.Receipt.Payload, &receipt); err != nil {
		return nil, domain.WrapError(domain.ErrorCodeIdempotencyState, "stored signed return receipt cannot be decoded", err)
	}
	if receipt.NamespaceID != scope.NamespaceID.String() || receipt.OutcomeID != scope.OutcomeID.String() || receipt.ContractID != contractID.String() || receipt.WorkItemID != workItemID.String() || receipt.RequestID != requestID.String() || receipt.IdempotencyKey != idempotencyKey || receipt.RequestDigest != requestDigest || !receipt.Accepted {
		return nil, domain.NewError(domain.ErrorCodeIdempotencyState, "stored signed return receipt binding differs")
	}
	switch receipt.Disposition {
	case "completed":
		return append([]ports.Permission(nil), signedDirectCompletionPermissions...), nil
	case "delivered_for_review":
		return nil, nil
	default:
		return nil, domain.NewError(domain.ErrorCodeIdempotencyState, "stored signed return receipt disposition is invalid")
	}
}
