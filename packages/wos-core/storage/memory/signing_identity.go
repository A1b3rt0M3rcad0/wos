package memory

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"reflect"
	"sort"
	"strings"
	"time"
)

type memoryReceipt struct {
	Fingerprint string
	Payload     json.RawMessage
}
type memorySecurityState struct {
	Namespaces  map[d.ID]ports.Namespace
	Versions    map[d.ID]d.Version
	Credentials map[string]ports.Credential
	Grants      map[string]ports.NamespaceGrant
	Keys        map[string]d.SigningKey
	Enrollments map[string]d.SigningEnrollment
	Policies    map[string]d.CredentialPolicy
	Acceptance  map[string]d.WorkAcceptancePolicy
	Groups      map[string]string
	Receipts    map[string]memoryReceipt
	Audit       map[d.ID][]ports.SecurityAudit
}

func newMemorySecurityState() memorySecurityState {
	return memorySecurityState{Namespaces: map[d.ID]ports.Namespace{}, Versions: map[d.ID]d.Version{}, Credentials: map[string]ports.Credential{}, Grants: map[string]ports.NamespaceGrant{}, Keys: map[string]d.SigningKey{}, Enrollments: map[string]d.SigningEnrollment{}, Policies: map[string]d.CredentialPolicy{}, Acceptance: map[string]d.WorkAcceptancePolicy{}, Groups: map[string]string{}, Receipts: map[string]memoryReceipt{}, Audit: map[d.ID][]ports.SecurityAudit{}}
}
func deepCopy[T any](v T) T {
	raw, _ := json.Marshal(v)
	var out T
	_ = json.Unmarshal(raw, &out)
	return out
}
func cloneMemorySecurityState(s memorySecurityState) memorySecurityState {
	out := deepCopy(s)
	// Credential JSON intentionally omits secret digests; preserve them privately.
	out.Credentials = map[string]ports.Credential{}
	for digest, c := range s.Credentials {
		out.Credentials[digest] = c
	}
	return out
}
func securityKey(ns d.ID, id string) string { return ns.String() + "/" + id }
func missingSigning() error {
	return d.NewError(d.ErrorCodeNotFound, "signing/security state not found")
}

type signingIdentityRepository struct{ tx *transaction }

func (tx *transaction) SigningIdentity() ports.SigningIdentityRepository {
	return signingIdentityRepository{tx}
}
func (r signingIdentityRepository) LockNamespace(ctx context.Context, ns d.ID) (d.Version, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	v, ok := r.tx.security.Versions[ns]
	if !ok {
		return 0, missingSigning()
	}
	return v, nil
}
func (r signingIdentityRepository) Credential(ctx context.Context, ns, id d.ID) (ports.Credential, error) {
	for _, c := range r.tx.security.Credentials {
		if c.NamespaceID == ns && c.ID == id {
			return c, ctx.Err()
		}
	}
	return ports.Credential{}, missingSigning()
}
func (r signingIdentityRepository) CredentialByDigest(ctx context.Context, digest string) (ports.Credential, error) {
	c, ok := r.tx.security.Credentials[digest]
	if !ok {
		return c, missingSigning()
	}
	return c, ctx.Err()
}
func (r signingIdentityRepository) Key(ctx context.Context, ns, id d.ID) (d.SigningKey, error) {
	k, ok := r.tx.security.Keys[securityKey(ns, id.String())]
	if !ok {
		return k, missingSigning()
	}
	return deepCopy(k), ctx.Err()
}
func (r signingIdentityRepository) Keys(ctx context.Context, ns d.ID, principal string, limit int) ([]d.SigningKey, error) {
	if limit < 1 || limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "key list limit outside 1..101")
	}
	out := []d.SigningKey{}
	for _, k := range r.tx.security.Keys {
		if k.NamespaceID == ns && k.PrincipalID == principal {
			out = append(out, deepCopy(k))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, ctx.Err()
}
func (r signingIdentityRepository) SaveKey(ctx context.Context, k d.SigningKey, expected d.Version) error {
	if err := k.Validate(); err != nil {
		return err
	}
	key := securityKey(k.NamespaceID, k.ID.String())
	old, exists := r.tx.security.Keys[key]
	if (exists && expected == 0) || (!exists && expected > 0) || k.Version != expected+1 || (exists && old.Version != expected) {
		return d.NewError(d.ErrorCodeVersionConflict, "key version changed")
	}
	if exists {
		o, n := old, k
		o.Status = n.Status
		o.Version = n.Version
		o.UpdatedAt = n.UpdatedAt
		o.Reason = n.Reason
		if !reflect.DeepEqual(o, n) || k.Status == "active" || old.Status == "revoked" {
			return d.NewError(d.ErrorCodeInvalidTransition, "public key identity/history immutable")
		}
	}
	for id, other := range r.tx.security.Keys {
		if id != key && other.NamespaceID == k.NamespaceID && other.Fingerprint == k.Fingerprint {
			return d.NewError(d.ErrorCodeAlreadyExists, "key fingerprint already registered")
		}
	}
	r.tx.security.Keys[key] = deepCopy(k)
	return ctx.Err()
}
func (r signingIdentityRepository) Enrollment(ctx context.Context, ns, id d.ID) (d.SigningEnrollment, error) {
	e, ok := r.tx.security.Enrollments[securityKey(ns, id.String())]
	if !ok {
		return e, missingSigning()
	}
	return deepCopy(e), ctx.Err()
}
func (r signingIdentityRepository) SaveEnrollment(ctx context.Context, e d.SigningEnrollment, expected d.Version) error {
	if err := e.Validate(); err != nil {
		return err
	}
	key := securityKey(e.NamespaceID, e.ID.String())
	old, exists := r.tx.security.Enrollments[key]
	if (exists && expected == 0) || (!exists && expected > 0) || e.Version != expected+1 || (exists && old.Version != expected) {
		return d.NewError(d.ErrorCodeVersionConflict, "enrollment version changed")
	}
	if exists {
		o, n := old, e
		o.Version = n.Version
		o.Consumed = n.Consumed
		o.AcceptedDigest = n.AcceptedDigest
		if old.Consumed || !e.Consumed || e.AcceptedDigest == "" || !reflect.DeepEqual(o, n) {
			return d.NewError(d.ErrorCodeInvalidTransition, "enrollment challenge immutable/one-use")
		}
	}
	c, err := r.Credential(ctx, e.NamespaceID, e.CredentialID)
	if err != nil || c.PrincipalID != e.PrincipalID {
		return d.NewError(d.ErrorCodeInvalidScope, "enrollment credential differs")
	}
	r.tx.security.Enrollments[key] = deepCopy(e)
	return ctx.Err()
}
func (r signingIdentityRepository) CredentialPolicy(ctx context.Context, ns, id d.ID) (d.CredentialPolicy, error) {
	p, ok := r.tx.security.Policies[securityKey(ns, id.String())]
	if !ok {
		return p, missingSigning()
	}
	return deepCopy(p), ctx.Err()
}
func (r signingIdentityRepository) SaveCredentialPolicy(ctx context.Context, p d.CredentialPolicy, expected d.Version) error {
	if err := p.Validate(); err != nil {
		return err
	}
	c, err := r.Credential(ctx, p.NamespaceID, p.CredentialID)
	if err != nil || c.PrincipalID != p.PrincipalID {
		return d.NewError(d.ErrorCodeInvalidScope, "policy credential differs")
	}
	key := securityKey(p.NamespaceID, p.CredentialID.String())
	old, exists := r.tx.security.Policies[key]
	if (exists && expected == 0) || (!exists && expected > 0) || p.Version != expected+1 || (exists && old.Version != expected) {
		return d.NewError(d.ErrorCodeVersionConflict, "policy version changed")
	}
	r.tx.security.Policies[key] = deepCopy(p)
	return ctx.Err()
}
func acceptanceKey(ns d.ID, o, w *d.ID) string {
	key := ns.String() + "/"
	if o != nil {
		key += o.String()
	}
	key += "/"
	if w != nil {
		key += w.String()
	}
	return key
}
func (r signingIdentityRepository) AcceptancePolicy(ctx context.Context, ns d.ID, o, w *d.ID) (d.WorkAcceptancePolicy, error) {
	p, ok := r.tx.security.Acceptance[acceptanceKey(ns, o, w)]
	if !ok {
		return p, missingSigning()
	}
	return deepCopy(p), ctx.Err()
}
func (r signingIdentityRepository) SaveAcceptancePolicy(ctx context.Context, p d.WorkAcceptancePolicy, expected d.Version) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if p.OutcomeID != nil {
		if _, err := r.tx.Outcomes().Get(ctx, p.NamespaceID, *p.OutcomeID); err != nil {
			return err
		}
		if p.WorkItemID != nil {
			if _, err := r.tx.WorkItems().Get(ctx, d.Scope{NamespaceID: p.NamespaceID, OutcomeID: *p.OutcomeID}, *p.WorkItemID); err != nil {
				return err
			}
		}
	}
	key := acceptanceKey(p.NamespaceID, p.OutcomeID, p.WorkItemID)
	old, exists := r.tx.security.Acceptance[key]
	if (exists && expected == 0) || (!exists && expected > 0) || p.Version != expected+1 || (exists && old.Version != expected) {
		return d.NewError(d.ErrorCodeVersionConflict, "acceptance policy version changed")
	}
	r.tx.security.Acceptance[key] = deepCopy(p)
	return ctx.Err()
}
func (r signingIdentityRepository) PrincipalGroup(ctx context.Context, ns d.ID, principal string) (string, error) {
	return r.tx.security.Groups[securityKey(ns, principal)], ctx.Err()
}
func (r signingIdentityRepository) SetPrincipalGroup(ctx context.Context, ns d.ID, principal, group string) error {
	if len(group) > 256 || strings.TrimSpace(principal) == "" {
		return d.NewError(d.ErrorCodeInvalidArgument, "invalid Principal review group")
	}
	if _, ok := r.tx.security.Grants[securityKey(ns, principal)]; !ok {
		return missingSigning()
	}
	r.tx.security.Groups[securityKey(ns, principal)] = group
	return ctx.Err()
}
func (r signingIdentityRepository) Receipt(ctx context.Context, ns d.ID, principal, key string) (string, json.RawMessage, error) {
	v, ok := r.tx.security.Receipts[securityKey(ns, principal)+"/"+key]
	if !ok {
		return "", nil, missingSigning()
	}
	return v.Fingerprint, append(json.RawMessage(nil), v.Payload...), ctx.Err()
}
func (r signingIdentityRepository) SaveReceipt(ctx context.Context, ns d.ID, principal, key, fingerprint string, payload json.RawMessage) error {
	k := securityKey(ns, principal) + "/" + key
	if _, ok := r.tx.security.Receipts[k]; ok {
		return d.NewError(d.ErrorCodeAlreadyExists, "security intent exists")
	}
	r.tx.security.Receipts[k] = memoryReceipt{fingerprint, append(json.RawMessage(nil), payload...)}
	return ctx.Err()
}
func (r signingIdentityRepository) Audit(ctx context.Context, ns d.ID, principal string, actor d.ActorRef, op, target string, now time.Time) (d.Version, error) {
	v, err := r.LockNamespace(ctx, ns)
	if err != nil {
		return 0, err
	}
	next, err := v.Next()
	if err != nil {
		return 0, err
	}
	r.tx.security.Versions[ns] = next
	r.tx.security.Audit[ns] = append(r.tx.security.Audit[ns], ports.SecurityAudit{NamespaceVersion: next, PrincipalID: principal, Actor: actor, Operation: op, Target: target, RecordedAt: now})
	return next, ctx.Err()
}

var _ ports.SigningIdentityUnitOfWork = (*transaction)(nil)
