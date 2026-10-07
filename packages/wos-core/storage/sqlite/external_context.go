package sqlite

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type externalContextRepository struct{ uow *unitOfWork }

func (u *unitOfWork) ExternalContexts() ports.ExternalContextRepository {
	return externalContextRepository{uow: u}
}
func (r externalContextRepository) List(ctx context.Context, scope domain.Scope) ([]domain.ExternalReference, error) {
	rows, err := r.uow.tx.QueryContext(ctx, `SELECT provider,context_kind,external_id,url FROM outcome_external_references WHERE namespace_id=? AND outcome_id=? ORDER BY provider,context_kind,external_id`, scope.NamespaceID.String(), scope.OutcomeID.String())
	if err != nil {
		return nil, mapSQLError("list external context", err)
	}
	defer rows.Close()
	values := []domain.ExternalReference{}
	for rows.Next() {
		var v domain.ExternalReference
		if err = rows.Scan(&v.Provider, &v.Kind, &v.ExternalID, &v.URL); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}
func (r externalContextRepository) Add(ctx context.Context, scope domain.Scope, v domain.ExternalReference) error {
	_, err := r.uow.tx.ExecContext(ctx, `INSERT INTO outcome_external_references(namespace_id,outcome_id,provider,context_kind,external_id,url) VALUES(?,?,?,?,?,?)`, scope.NamespaceID.String(), scope.OutcomeID.String(), v.Provider, v.Kind, v.ExternalID, v.URL)
	return mapSQLError("link external reference", err)
}
func (r externalContextRepository) Remove(ctx context.Context, scope domain.Scope, v domain.ExternalReference) error {
	_, err := r.uow.tx.ExecContext(ctx, `DELETE FROM outcome_external_references WHERE namespace_id=? AND outcome_id=? AND provider=? AND context_kind=? AND external_id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), v.Provider, v.Kind, v.ExternalID)
	return mapSQLError("remove external reference", err)
}
