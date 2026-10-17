package postgres

import (
	"context"
	"database/sql"
)

// Live replicas must verify the complete compiled migration inventory without
// running DDL or repairing the ledger. Upgrades use the suspended catalog path.
func (s *Store) validateTenantMigrations(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := validatePlacementMigrations(ctx, tx, s.path); err != nil {
		return err
	}
	return tx.Commit()
}
