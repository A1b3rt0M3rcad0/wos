package woscli

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func operationLimitV2(o options) (int, error) {
	if o.values["limit"] == "" {
		return 25, nil
	}
	n, e := strconv.Atoi(o.values["limit"])
	if e != nil || n < 1 || n > 100 {
		return 0, usage("--limit must be 1..100")
	}
	return n, nil
}

// Names and prefixes are conveniences, never authorities. Resolution fails
// closed when the bounded search cannot prove that a candidate is unique.
func matchesReferenceV2(ref, id, name string) bool {
	return ref == "" || strings.EqualFold(ref, id) || strings.EqualFold(ref, name) || len(ref) >= 4 && strings.HasPrefix(strings.ToLower(id), strings.ToLower(ref))
}

func resolveOutcomeV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, o options) (d.Scope, error) {
	project, e := w.LoadProjectV2()
	if e != nil {
		return d.Scope{}, e
	}
	scope := d.Scope{TenantID: profile.Binding.TenantID, OutcomeID: project.Scope.DefaultOutcomeID}
	if value := o.values["tenant"]; value != "" && value != scope.TenantID.String() {
		return scope, usage("profile Tenant is immutable")
	}
	ref := o.values["outcome"]
	if ref == "" && !scope.OutcomeID.IsZero() {
		return scope, scope.Validate()
	}
	if id, parse := d.ParseID(ref); parse == nil {
		scope.OutcomeID = id
		return scope, scope.Validate()
	}
	items := []d.ID{}
	cursor := ""
	for scan := 0; scan < 10; scan++ {
		page, e := client.SearchOutcomes(ctx, scope.TenantID, "", 100, cursor)
		if e != nil {
			return scope, e
		}
		for _, item := range page.Items {
			if matchesReferenceV2(ref, item.ID.String(), item.Title) {
				items = append(items, item.ID)
			}
		}
		if page.NextCursor == "" {
			if len(items) != 1 {
				return scope, usage(fmt.Sprintf("Outcome reference matched %d candidates; select --outcome with an exact ID or unique title", len(items)))
			}
			scope.OutcomeID = items[0]
			return scope, scope.Validate()
		}
		if page.NextCursor == cursor {
			return scope, fmt.Errorf("Outcome cursor did not advance; use an exact --outcome ID")
		}
		cursor = page.NextCursor
	}
	return scope, usage("Outcome discovery bound reached; use an exact --outcome ID")
}

func selectedContractV2(w *Workspace, profile ProfileV2, kind, ref string) (d.ID, error) {
	// Exact IDs retain compatibility, including read-only migrated records and
	// receipt recovery after the local contract has already been cleaned up.
	if id, e := d.ParseID(ref); e == nil {
		return id, nil
	}
	ids, e := contractIDsV2(w, profile.Name)
	if e != nil {
		return "", e
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	matches := []d.ID{}
	for _, id := range ids {
		_, legacy, e := readLegacyUnsignedV2(w, profile, id)
		if e != nil {
			return "", e
		}
		if legacy {
			continue
		}
		_, view, _, _, e := loadContractV2(w, profile, id)
		if e != nil {
			return "", e
		}
		if view.Authority.ContractKind != kind {
			continue
		}
		name := ""
		if view.WorkSpecification != nil {
			name = view.WorkSpecification.Spec.Title
		}
		if matchesReferenceV2(ref, id.String(), name) || ref != "" && matchesReferenceV2(ref, view.Authority.WorkItemID, "") {
			matches = append(matches, id)
		}
	}
	if len(matches) != 1 {
		return "", usage(fmt.Sprintf("contract reference matched %d %s contracts; use a contract ID, unique prefix, or work title (work list)", len(matches), kind))
	}
	return matches[0], nil
}

func contractOptionV2(w *Workspace, profile ProfileV2, kind string, o options, position int) (d.ID, error) {
	if len(o.args) > position+1 {
		return "", usage("at most one contract reference is accepted")
	}
	ref := o.values["contract"]
	if len(o.args) > position {
		if ref != "" {
			return "", usage("select a contract with either an argument or --contract")
		}
		ref = o.args[position]
	}
	return selectedContractV2(w, profile, kind, ref)
}

func outcomeCommandV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, o options) (Output, error) {
	result := Output{Operation: "outcome"}
	if len(o.args) < 2 {
		return result, usage("outcome list|get [reference]")
	}
	result.Operation += " " + o.args[1]
	switch o.args[1] {
	case "list":
		if len(o.args) != 2 {
			return result, usage("outcome list --limit N --cursor CURSOR")
		}
		limit, e := operationLimitV2(o)
		if e != nil {
			return result, e
		}
		page, e := client.SearchOutcomes(ctx, profile.Binding.TenantID, "", limit, o.values["cursor"])
		result.Data = page
		return result, e
	case "get":
		if len(o.args) > 3 {
			return result, usage("outcome get [reference]")
		}
		if len(o.args) == 3 {
			o.values["outcome"] = o.args[2]
		}
		scope, e := resolveOutcomeV2(ctx, w, profile, client, o)
		if e != nil {
			return result, e
		}
		value, e := client.GetContinuity(ctx, scope, 10)
		result.OutcomeID = scope.OutcomeID
		result.Data = value
		return result, e
	default:
		return result, usage("outcome list|get [reference]")
	}
}

func resolveAvailableWorkV2(ctx context.Context, client *sdk.Client, scope d.Scope, ref string) (d.ID, d.Version, error) {
	cursor := ""
	matches := []a.AvailableWork{}
	for scan := 0; scan < 10; scan++ {
		page, e := client.ListAvailableWork(ctx, scope, 100, cursor)
		if e != nil {
			return "", 0, e
		}
		for _, item := range page.Items {
			if matchesReferenceV2(ref, item.Candidate.ID.String(), item.Title) {
				matches = append(matches, item)
			}
		}
		if page.SearchComplete {
			if len(matches) != 1 {
				return "", 0, usage(fmt.Sprintf("work reference matched %d available items; select a unique title or ID", len(matches)))
			}
			return matches[0].Candidate.ID, matches[0].Candidate.Version, nil
		}
		if page.NextCursor == "" || page.NextCursor == cursor {
			return "", 0, fmt.Errorf("available-work search incomplete; use an exact work ID with --version")
		}
		cursor = page.NextCursor
	}
	return "", 0, usage("available-work discovery bound reached; use an exact work ID with --version")
}
