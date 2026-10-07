package sqlite

import (
	"context"
	"database/sql"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func syncExternalContextIndex(ctx context.Context, tx *sql.Tx, outcome domain.Outcome) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM outcome_context_entries WHERE namespace_id=? AND outcome_id=?`, outcome.NamespaceID.String(), outcome.ID.String()); err != nil {
		return mapSQLError("replace context index", err)
	}
	for key, value := range outcome.ExternalContext {
		canonical, err := domain.CanonicalContextValue(value)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO outcome_context_entries(namespace_id,outcome_id,context_key,canonical_value) VALUES(?,?,?,?)`, outcome.NamespaceID.String(), outcome.ID.String(), key, canonical); err != nil {
			return mapSQLError("index external context", err)
		}
	}
	return nil
}
