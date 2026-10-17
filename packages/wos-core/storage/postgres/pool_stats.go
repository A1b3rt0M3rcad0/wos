package postgres

import "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"

func (s *Store) ConnectionStats() ports.StoragePoolStats {
	v := s.db.Stats()
	return ports.StoragePoolStats{MaxOpenConnections: v.MaxOpenConnections, OpenConnections: v.OpenConnections, InUse: v.InUse, Idle: v.Idle, WaitCount: v.WaitCount, WaitDurationNS: int64(v.WaitDuration)}
}
