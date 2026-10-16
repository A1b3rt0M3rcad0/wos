package ports

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"testing"
)

func TestContinuityBoundedBuilderMatchesFullSort(t *testing.T) {
	keys := make([]string, 5000)
	for i := range keys {
		keys[i] = fmt.Sprintf("%06d", i)
	}
	rng := rand.New(rand.NewPCG(7, 31))
	rng.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
	for _, after := range []string{"", "000049", "004994", "009999"} {
		b, e := NewContinuityPageBuilder(after, 7)
		if e != nil {
			t.Fatal(e)
		}
		want := []string{}
		for _, key := range keys {
			if key > after {
				want = append(want, key)
			}
			if e = b.Add(key, struct {
				ID string `json:"id"`
			}{key}); e != nil {
				t.Fatal(e)
			}
			if len(b.items) > 7 {
				t.Fatal("page retained the collection")
			}
		}
		sort.Strings(want)
		page := b.Page()
		if page.Count != 5000 || page.Remaining != max(0, len(want)-7) || len(page.Items) != min(len(want), 7) {
			t.Fatal("counts", page.Count, page.Remaining, len(page.Items))
		}
		for i, item := range page.Items {
			if item.Key != want[i] || string(item.Value) != `{"id":"`+want[i]+`"}` {
				t.Fatal("sort/payload mismatch", item)
			}
		}
		if b.HasAfter() != (after == "" || after <= "004999") {
			t.Fatal("missing cursor was accepted")
		}
	}
}
