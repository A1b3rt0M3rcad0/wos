package sqlite

import "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"

func (s *Store) ConnectionStats() ports.StoragePoolStats {
	v := s.db.Stats()
	if s.readDB != nil {
		readers := s.readDB.Stats()
		v.MaxOpenConnections += readers.MaxOpenConnections
		v.OpenConnections += readers.OpenConnections
		v.InUse += readers.InUse
		v.Idle += readers.Idle
		v.WaitCount += readers.WaitCount
		v.WaitDuration += readers.WaitDuration
	}
	return ports.StoragePoolStats{MaxOpenConnections: v.MaxOpenConnections, OpenConnections: v.OpenConnections, InUse: v.InUse, Idle: v.Idle, WaitCount: v.WaitCount, WaitDurationNS: int64(v.WaitDuration)}
}
