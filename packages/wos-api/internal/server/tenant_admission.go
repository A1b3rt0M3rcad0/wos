package server

import (
	"net/http"
	"sync"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// tenantAdmissionHTTP bounds authenticated Tenant work in this process.
// Identity always comes from authentication, never a URL or caller-supplied
// tenant header. Global transport admission continues to bound authentication
// itself. Active-only entries avoid an unbounded tenant registry or wait queue.
func tenantAdmissionHTTP(maximum int, resolve func(*http.Request) (application.Identity, error), next http.Handler) http.Handler {
	if maximum == 0 || resolve == nil {
		return next
	}
	var mu sync.Mutex
	active := map[domain.ID]int{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := resolve(r)
		if err != nil {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeTransportError(w, http.StatusUnauthorized, "unauthorized", "valid authentication is required")
			return
		}
		mu.Lock()
		if active[id.TenantID] >= maximum {
			mu.Unlock()
			w.Header().Set("Retry-After", "1")
			writeTransportError(w, http.StatusTooManyRequests, "tenant_overloaded", "tenant request capacity is saturated; reconcile existing mutations before retrying")
			return
		}
		active[id.TenantID]++
		mu.Unlock()
		defer func() {
			mu.Lock()
			active[id.TenantID]--
			if active[id.TenantID] == 0 {
				delete(active, id.TenantID)
			}
			mu.Unlock()
		}()
		next.ServeHTTP(w, r.WithContext(application.WithIdentity(r.Context(), id)))
	})
}
