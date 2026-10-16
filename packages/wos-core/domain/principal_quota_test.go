package domain

import (
	"encoding/json"
	"testing"
)

func TestPrincipalQuotaPolicyScopeBoundsAndLegacyDefaults(t *testing.T) {
	var p WorkAcceptancePolicy
	if e := json.Unmarshal([]byte(`{"namespace_id":"0199d180-0000-7000-8000-000000000001","revision":"1","acceptance_floor":"direct","max_active_work_contracts":20,"max_active_review_contracts":10}`), &p); e != nil {
		t.Fatal(e)
	}
	if e := p.Validate(); e != nil || p.PrincipalWorkLimit() != 3 || p.PrincipalReviewLimit() != 1 {
		t.Fatal("published policy defaults changed", e)
	}
	for _, v := range []int{-1, 0, 101} {
		p.MaxActiveWorkPerPrincipal = &v
		if p.Validate() == nil {
			t.Fatal("unsafe quota accepted", v)
		}
	}
	one, hundred := 1, 100
	p.MaxActiveWorkPerPrincipal = &hundred
	p.MaxActiveReviewPerPrincipal = &one
	if e := p.Validate(); e != nil {
		t.Fatal(e)
	}
	id := MustParseID("0199d180-0000-7000-8000-000000000002")
	p.OutcomeID = &id
	if p.Validate() == nil {
		t.Fatal("Outcome policy changed Principal ceiling")
	}
	p.WorkItemID = &id
	if p.Validate() == nil {
		t.Fatal("WorkItem policy changed Principal ceiling")
	}
	p.MaxActiveWorkPerPrincipal = nil
	p.MaxActiveReviewPerPrincipal = nil
	if e := p.Validate(); e != nil {
		t.Fatal("old scoped policy rejected", e)
	}
}
