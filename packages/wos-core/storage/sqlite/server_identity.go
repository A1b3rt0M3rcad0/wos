package sqlite

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type serverIdentityRepository struct{ u *unitOfWork }

func (u *unitOfWork) ServerIdentity() ports.ServerIdentityRepository {
	return serverIdentityRepository{u}
}
func (r serverIdentityRepository) Server(ctx context.Context) (d.ServerIdentity, error) {
	s, err := signingRead[d.ServerIdentity](ctx, r.u, `SELECT state_json FROM server_protocol_identity WHERE singleton=1`)
	if err == nil {
		err = s.Validate()
	}
	return s, err
}
func (r serverIdentityRepository) InsertServer(ctx context.Context, s d.ServerIdentity) error {
	if err := s.Validate(); err != nil {
		return err
	}
	raw, err := marshalJSON(s)
	if err != nil {
		return err
	}
	_, err = r.u.tx.ExecContext(ctx, `INSERT INTO server_protocol_identity(singleton,state_json) VALUES(1,?)`, raw)
	return mapSQLError("insert immutable server identity", err)
}
