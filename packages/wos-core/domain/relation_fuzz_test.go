package domain

import (
	"fmt"
	"testing"
	"time"
)

// An independent transitive-closure oracle checks proposed edges in arbitrary
// small DAGs; it exercises indirect cycles, not just opposite pairs.
func FuzzDependencyCycleAgainstReachability(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5}, uint8(5), uint8(0))
	f.Add([]byte{0, 0, 0}, uint8(0), uint8(5))
	f.Fuzz(func(t *testing.T, edges []byte, from, to uint8) {
		const n = 6
		scope := Scope{NamespaceID: MustParseID("01a11770-0000-7000-8000-000000000001"), OutcomeID: MustParseID("01a11770-0000-7000-8000-000000000002")}
		ref := func(i int) EntityRef {
			return EntityRef{Scope: scope, Kind: EntityKindWorkItem, ID: MustParseID(fmt.Sprintf("01a11771-0000-7000-8000-%012x", i+1))}
		}
		reach := [n][n]bool{}
		existing := []Relation{}
		k := 0
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				if len(edges) > 0 && edges[k%len(edges)]%2 == 1 {
					r, e := NewDependencyRelation(MustParseID(fmt.Sprintf("01a11772-0000-7000-8000-%012x", k+1)), ref(i), ref(j), DependencyStrengthHard, time.Now())
					if e != nil {
						t.Fatal(e)
					}
					existing = append(existing, r)
					reach[i][j] = true
				}
				k++
			}
		}
		for k := 0; k < n; k++ {
			for i := 0; i < n; i++ {
				for j := 0; j < n; j++ {
					reach[i][j] = reach[i][j] || (reach[i][k] && reach[k][j])
				}
			}
		}
		a, b := int(from)%n, int(to)%n
		candidate, err := NewDependencyRelation(MustParseID("01a11772-0000-7000-8000-000000000100"), ref(a), ref(b), DependencyStrengthHard, time.Now())
		if a == b {
			if err == nil {
				t.Fatal("accepted self dependency")
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		err = ValidateDependencyAcyclic(existing, candidate, 100)
		if (err != nil) != reach[b][a] {
			t.Fatalf("cycle oracle differs for %d->%d: reachable=%v err=%v", a, b, reach[b][a], err)
		}
	})
}
