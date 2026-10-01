package ports

import "time"

// Clock supplies time to application/domain workflows without coupling them to
// the process clock. Implementations should return UTC instants.
type Clock interface {
	Now() time.Time
}
