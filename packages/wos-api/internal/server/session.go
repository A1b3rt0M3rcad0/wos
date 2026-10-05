package server

import (
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const sessionCookie = "wos_session"

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && strings.EqualFold(u.Host, r.Host)
}
func loopbackHost(host string) bool {
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host
	}
	ip := net.ParseIP(h)
	return h == "localhost" || (ip != nil && ip.IsLoopback())
}
func sessionHandler(security *application.SecurityService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !sameOrigin(r) {
			http.Error(w, "origin rejected", http.StatusForbidden)
			return
		}
		secure := r.TLS != nil || !loopbackHost(r.Host)
		if r.Method == "DELETE" {
			if cookie, err := r.Cookie(sessionCookie); err == nil {
				_ = security.EndSession(r.Context(), cookie.Value)
			}
			http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode, MaxAge: -1})
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		var b struct {
			Token string `json:"token"`
		}
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		if d.Decode(&b) != nil {
			http.Error(w, "invalid session request", http.StatusBadRequest)
			return
		}
		credential, token, err := security.CreateSession(r.Context(), b.Token)
		if err != nil {
			http.Error(w, "invalid credential or grant", http.StatusUnauthorized)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode, MaxAge: 12 * 3600, Expires: credential.ExpiresAt})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"namespace_id": credential.NamespaceID, "principal_id": credential.PrincipalID, "actor_ref": credential.Actor, "expires_at": credential.ExpiresAt})
	})
}
