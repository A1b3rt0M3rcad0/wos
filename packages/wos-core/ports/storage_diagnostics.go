package ports

// StoragePoolStats contains process-local connection pressure, never credentials
// or Namespace data. Aggregate budgets still belong to the installation operator.
type StoragePoolStats struct {
	MaxOpenConnections int   `json:"max_open_connections"`
	OpenConnections    int   `json:"open_connections"`
	InUse              int   `json:"in_use"`
	Idle               int   `json:"idle"`
	WaitCount          int64 `json:"wait_count"`
	WaitDurationNS     int64 `json:"wait_duration_ns"`
}

type StoragePoolObserver interface{ ConnectionStats() StoragePoolStats }
