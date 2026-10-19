package application

import (
	"bytes"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"reflect"
	"strings"
	"time"
)

// WorkspaceSubject selects a read surface, not a new globally writable entity.
// A Criterion always remains addressed through its actual aggregate owner.
type WorkspaceSubject struct {
	Kind           string      `json:"kind"`
	Ref            d.EntityRef `json:"ref"`
	CriterionID    *d.ID       `json:"criterion_id,omitempty"`
	RevisionNumber uint64      `json:"revision_number,omitempty,string"`
	RecordKind     string      `json:"record_kind,omitempty"`
	RecordID       *d.ID       `json:"record_id,omitempty"`
}
type WorkspaceViewQuery struct {
	Subject                  WorkspaceSubject              `json:"subject"`
	PlacementPreview         *PlacementPreviewQuery        `json:"placement_preview,omitempty"`
	BlockerPreview           *WorkspaceBlockerPreviewQuery `json:"blocker_preview,omitempty"`
	CriteriaCursor           string                        `json:"criteria_cursor,omitempty"`
	RequiredObjectivesCursor string                        `json:"required_objectives_cursor,omitempty"`
	RelationCursor           string                        `json:"relation_cursor,omitempty"`
	Limit                    int                           `json:"limit,omitempty"`
}
type WorkspaceCoverage struct {
	Status        string `json:"status"`
	ReturnedCount *int   `json:"returned_count,omitempty"`
	TotalCount    *int   `json:"total_count,omitempty"`
	NextCursor    string `json:"next_cursor,omitempty"`
	Reason        string `json:"reason,omitempty"`
}
type WorkspaceHeader struct {
	Title        string `json:"title"`
	Kind         string `json:"kind"`
	Lifecycle    string `json:"lifecycle,omitempty"`
	DisplayState string `json:"display_state,omitempty"`
	Description  string `json:"description,omitempty"`
}
type WorkspacePlacement struct {
	Outcome EntitySummary   `json:"outcome"`
	Parent  *EntitySummary  `json:"parent,omitempty"`
	Path    []EntitySummary `json:"path"`
}
type WorkspacePerson struct {
	Role           string      `json:"role"`
	Actor          *d.ActorRef `json:"actor_ref,omitempty"`
	PrincipalID    string      `json:"principal_id,omitempty"`
	Source         string      `json:"source"`
	AuthorityState string      `json:"authority_state,omitempty"`
	ContractID     *d.ID       `json:"contract_id,omitempty"`
	At             *time.Time  `json:"at,omitempty"`
}
type WorkspaceCriterion struct {
	Definition           d.SuccessCriterion     `json:"definition"`
	Assessment           *d.CriterionAssessment `json:"assessment,omitempty"`
	AssessmentApplicable bool                   `json:"assessment_applicable"`
	EvidenceLinked       bool                   `json:"evidence_linked"`
}
type WorkspaceDelivery struct {
	ContractID       *d.ID  `json:"contract_id,omitempty"`
	SubmissionID     *d.ID  `json:"submission_id,omitempty"`
	SubmissionDigest string `json:"submission_digest,omitempty"`
	ReviewCaseID     *d.ID  `json:"review_case_id,omitempty"`
	Status           string `json:"status"`
	Summary          string `json:"summary,omitempty"`
	Current          bool   `json:"current"`
}
type WorkspaceRelationships struct {
	Items []WorkspaceRelationship `json:"items"`
	Nodes []EntitySummary         `json:"nodes"`
}
type WorkspaceRelationship struct {
	GraphEdge
	Strength    d.DependencyStrength `json:"strength,omitempty"`
	Stance      d.EvidenceStance     `json:"stance,omitempty"`
	CriterionID *d.ID                `json:"criterion_id,omitempty"`
	Rationale   string               `json:"rationale,omitempty"`
}
type WorkspaceCapability struct {
	ActionID             string           `json:"action_id"`
	Subject              WorkspaceSubject `json:"subject"`
	Transport            string           `json:"transport"`
	Availability         string           `json:"availability"`
	ReasonCodes          []string         `json:"reason_codes"`
	RelatedRefs          []d.EntityRef    `json:"related_refs"`
	Remedies             []d.ErrorRemedy  `json:"remedies,omitempty"`
	ObservedVersion      d.Version        `json:"observed_version,string"`
	ConfirmationKind     string           `json:"confirmation_kind,omitempty"`
	PresentationPriority int              `json:"presentation_priority"`
}
type WorkspaceConclusionImpact struct {
	OpenObjectives int             `json:"open_objectives"`
	OpenTasks      int             `json:"open_tasks"`
	OpenItems      []EntitySummary `json:"open_items"`
	Consequences   []string        `json:"consequences"`
}
type EntityWorkspaceView struct {
	Subject         WorkspaceSubject  `json:"subject"`
	EntityVersion   d.Version         `json:"entity_version,string"`
	OutcomeRevision d.OutcomeRevision `json:"outcome_revision,string"`
	EvaluatedAt     time.Time         `json:"evaluated_at"`
	Header          WorkspaceHeader   `json:"header"`
	// Entity is the selected canonical record, with historical collections omitted.
	// Forms still load their full editable record through the existing resources.
	Entity             any                          `json:"entity"`
	Observation        *WorkspaceObservationSource  `json:"observation,omitempty"`
	Placement          WorkspacePlacement           `json:"placement"`
	People             []WorkspacePerson            `json:"people"`
	Work               *d.ExecutionSpec             `json:"work,omitempty"`
	Criteria           []WorkspaceCriterion         `json:"criteria,omitempty"`
	RequiredObjectives []EntitySummary              `json:"required_objectives,omitempty"`
	ConclusionImpact   *WorkspaceConclusionImpact   `json:"conclusion_impact,omitempty"`
	PlacementPreview   *WorkspacePlacementPreview   `json:"placement_preview,omitempty"`
	BlockerPreview     *WorkspaceBlockerPreview     `json:"blocker_preview,omitempty"`
	Delivery           *WorkspaceDelivery           `json:"delivery,omitempty"`
	Roadmap            *WorkspaceRoadmapFacet       `json:"roadmap,omitempty"`
	Relationships      WorkspaceRelationships       `json:"relationships"`
	Attention          *WorkAttention               `json:"attention,omitempty"`
	Activity           *WorkspaceTimelinePage       `json:"activity,omitempty"`
	Capabilities       []WorkspaceCapability        `json:"capabilities"`
	Coverage           map[string]WorkspaceCoverage `json:"coverage"`
	Rules              WorkspaceRules               `json:"rules"`
}
type WorkspaceRoadmapFacet struct {
	Draft                *d.RoadmapDraft     `json:"draft,omitempty"`
	Revisions            []d.RoadmapRevision `json:"revisions"`
	SelectedRevision     *d.RoadmapRevision  `json:"selected_revision,omitempty"`
	ActiveRevisionNumber *uint64             `json:"active_revision_number,omitempty,string"`
	IsActive             bool                `json:"is_active"`
	NodesStatus          string              `json:"nodes_status"`
}
type WorkspaceCardsQuery struct {
	Refs      []d.EntityRef `json:"refs,omitempty"`
	ParentRef *d.EntityRef  `json:"parent_ref,omitempty"`
	Limit     int           `json:"limit,omitempty"`
	Cursor    string        `json:"cursor,omitempty"`
}
type WorkspaceCriteriaSummary struct {
	Active   int `json:"active"`
	Required int `json:"required"`
	Met      int `json:"met"`
	Waived   int `json:"waived"`
	Pending  int `json:"pending"`
}
type WorkspaceCardSummary struct {
	Ref                d.EntityRef               `json:"ref"`
	Version            d.Version                 `json:"version,string"`
	Title              string                    `json:"title"`
	Lifecycle          string                    `json:"lifecycle"`
	DisplayState       string                    `json:"display_state"`
	Description        string                    `json:"description,omitempty"`
	Priority           d.Priority                `json:"priority,omitempty"`
	ParentRef          *d.EntityRef              `json:"parent_ref,omitempty"`
	RequiredForOutcome bool                      `json:"required_for_outcome,omitempty"`
	Participants       []WorkspacePerson         `json:"participants"`
	Criteria           *WorkspaceCriteriaSummary `json:"criteria,omitempty"`
	AuthorityState     string                    `json:"authority_state"`
	AttentionReason    string                    `json:"attention_reason,omitempty"`
}
type WorkspaceCardsPage struct {
	Items           []WorkspaceCardSummary `json:"items"`
	OutcomeRevision d.OutcomeRevision      `json:"outcome_revision,string"`
	EvaluatedAt     time.Time              `json:"evaluated_at"`
	Coverage        WorkspaceCoverage      `json:"coverage"`
}

// New workspace projections quote every version/revision/fencing integer,
// including integers nested in canonical records. Decoder.UseNumber prevents a
// float64 round trip. Legacy endpoints retain their existing codecs.
func marshalWorkspaceProjection(value any) ([]byte, error) {
	raw, e := json.Marshal(value)
	if e != nil {
		return nil, e
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var node any
	if e = decoder.Decode(&node); e != nil {
		return nil, e
	}
	rawType := reflect.TypeOf(json.RawMessage{})
	authored := func(key string) bool {
		switch key {
		case "external_context", "execution_context", "metadata", "measurement", "expected_measurement", "payload":
			return true
		}
		return false
	}
	var rewrite func(any, reflect.Value) any
	rewrite = func(node any, v reflect.Value) any {
		if !v.IsValid() {
			return node
		}
		for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return node
			}
			v = v.Elem()
		}
		if v.Type() == rawType {
			return node
		}
		if v.Kind() == reflect.Uint64 {
			if n, ok := node.(json.Number); ok {
				return n.String()
			}
			return node
		}
		switch v.Kind() {
		case reflect.Struct:
			object, ok := node.(map[string]any)
			if !ok {
				return node
			}
			t := v.Type()
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				if f.PkgPath != "" {
					continue
				}
				tag := strings.Split(f.Tag.Get("json"), ",")[0]
				if tag == "-" {
					continue
				}
				if f.Anonymous && tag == "" {
					rewrite(node, v.Field(i))
					continue
				}
				if tag == "" {
					tag = f.Name
				}
				if authored(tag) {
					continue
				}
				if item, ok := object[tag]; ok {
					object[tag] = rewrite(item, v.Field(i))
				}
			}
			return object
		case reflect.Map:
			object, ok := node.(map[string]any)
			if !ok || v.Type().Key().Kind() != reflect.String {
				return node
			}
			iterator := v.MapRange()
			for iterator.Next() {
				key := iterator.Key().String()
				if authored(key) {
					continue
				}
				if item, ok := object[key]; ok {
					object[key] = rewrite(item, iterator.Value())
				}
			}
			return object
		case reflect.Slice, reflect.Array:
			list, ok := node.([]any)
			if !ok {
				return node
			}
			for i := range list {
				if i < v.Len() {
					list[i] = rewrite(list[i], v.Index(i))
				}
			}
			return list
		}
		return node
	}
	node = rewrite(node, reflect.ValueOf(value))
	return json.Marshal(node)
}
func (v EntityWorkspaceView) MarshalJSON() ([]byte, error) {
	type plain EntityWorkspaceView
	return marshalWorkspaceProjection(plain(v))
}
func (v WorkspaceCardsPage) MarshalJSON() ([]byte, error) {
	type plain WorkspaceCardsPage
	return marshalWorkspaceProjection(plain(v))
}
func unmarshalWorkspaceProjection(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if e := decoder.Decode(&value); e != nil {
		return e
	}
	var normalize func(any, reflect.Type)
	normalize = func(value any, t reflect.Type) {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		switch t.Kind() {
		case reflect.Struct:
			record, ok := value.(map[string]any)
			if !ok {
				return
			}
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				tag := field.Tag.Get("json")
				parts := strings.Split(tag, ",")
				name := parts[0]
				if name == "" {
					name = field.Name
				}
				item, ok := record[name]
				if !ok {
					continue
				}
				ft := field.Type
				for ft.Kind() == reflect.Pointer {
					ft = ft.Elem()
				}
				quoted := false
				for _, option := range parts[1:] {
					quoted = quoted || option == "string"
				}
				if !quoted && (ft.Kind() >= reflect.Int && ft.Kind() <= reflect.Uint64) {
					if text, ok := item.(string); ok {
						record[name] = json.Number(text)
					}
				} else {
					normalize(item, field.Type)
				}
			}
		case reflect.Slice:
			if list, ok := value.([]any); ok {
				for _, item := range list {
					normalize(item, t.Elem())
				}
			}
		case reflect.Map:
			if record, ok := value.(map[string]any); ok {
				for key, item := range record {
					if t.Elem().Kind() >= reflect.Int && t.Elem().Kind() <= reflect.Uint64 {
						if text, ok := item.(string); ok {
							record[key] = json.Number(text)
						}
					} else {
						normalize(item, t.Elem())
					}
				}
			}
		}
	}
	normalize(value, reflect.TypeOf(target))
	encoded, e := json.Marshal(value)
	if e != nil {
		return e
	}
	return json.Unmarshal(encoded, target)
}
func (v *EntityWorkspaceView) UnmarshalJSON(raw []byte) error {
	type plain EntityWorkspaceView
	return unmarshalWorkspaceProjection(raw, (*plain)(v))
}
func (v *WorkspaceCardsPage) UnmarshalJSON(raw []byte) error {
	type plain WorkspaceCardsPage
	return unmarshalWorkspaceProjection(raw, (*plain)(v))
}
func workspaceCount(n int) *int { return &n }
