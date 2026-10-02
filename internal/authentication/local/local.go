package local

import (
	"strings"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

type Resolver struct {
	PrincipalID string
}

func New(principalID string) (Resolver, error) {
	principalID = strings.TrimSpace(principalID)
	if principalID == "" {
		return Resolver{}, domain.NewError(domain.ErrorCodeInvalidConfig, "local principal id is required")
	}
	return Resolver{PrincipalID: principalID}, nil
}

func (r Resolver) Principal() string {
	return r.PrincipalID
}

func (r Resolver) Actor() domain.ActorRef {
	return domain.ActorRef{
		Kind:     domain.ActorKindHuman,
		Provider: "local",
		ID:       r.PrincipalID,
	}
}
