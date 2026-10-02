package ports

import "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"

// IDGenerator creates identifiers that satisfy the public domain ID contract.
// Production implementations are expected to generate UUIDv7 values.
type IDGenerator interface {
	NewID() (domain.ID, error)
}
