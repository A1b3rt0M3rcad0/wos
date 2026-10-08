package server

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"os"
	"regexp"
)

// localIssuer never performs IO. Its private seed is loaded by composition and
// is absent from the public identity, configuration response and WOS database.
type localIssuer struct {
	identity d.ServerIdentity
	private  ed25519.PrivateKey
}

func (s *localIssuer) Identity() d.ServerIdentity { return s.identity }
func (s *localIssuer) SignCanonical(purpose string, payload []byte) (json.RawMessage, error) {
	envelope, err := signing.SignCanonical(signing.PayloadType(purpose), payload, s.identity.IssuerKeyID.String(), s.private)
	if err != nil {
		return nil, err
	}
	document, err := signing.ToDocument(envelope)
	if err != nil {
		return nil, err
	}
	return json.Marshal(document.Proof)
}
func configureSignedRuntime(ctx context.Context, config Config, store runtimeStore, service *application.Service, security *application.SecurityService, ids ports.IDGenerator) error {
	uow, err := store.Begin(ctx)
	if err != nil {
		return err
	}
	repo, ok := uow.(ports.ServerIdentityUnitOfWork)
	if !ok {
		_ = uow.Rollback()
		return d.NewError(d.ErrorCodeInvalidConfig, "persistent signed server identity unavailable")
	}
	pinned, readErr := repo.ServerIdentity().Server(ctx)
	_ = uow.Rollback()
	if readErr != nil {
		if code, _ := d.ErrorCodeOf(readErr); code != d.ErrorCodeNotFound {
			return readErr
		}
	}
	if config.Signing.SeedEnv == "" {
		if readErr == nil && security != nil {
			security.ServerID = pinned.ID.String()
		}
		return nil
	}
	if security == nil || config.Auth.Mode != AuthModeAPIToken {
		return d.NewError(d.ErrorCodeInvalidConfig, "signed runtime requires scoped API credentials")
	}
	if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(config.Signing.SeedEnv) {
		return d.NewError(d.ErrorCodeInvalidConfig, "signing seed reference must name an environment variable")
	}
	encoded := os.Getenv(config.Signing.SeedEnv)
	seed, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(seed) != ed25519.SeedSize || base64.StdEncoding.EncodeToString(seed) != encoded {
		return d.NewError(d.ErrorCodeInvalidConfig, "signing seed reference is missing or not canonical base64 Ed25519 seed")
	}
	private := ed25519.NewKeyFromSeed(seed)
	clear(seed)
	public := private.Public().(ed25519.PublicKey)
	fp, err := signing.Fingerprint(public)
	if err != nil {
		return err
	}
	if readErr == nil {
		if pinned.PublicKey != base64.StdEncoding.EncodeToString(public) || pinned.Fingerprint != fp {
			clear(private)
			return d.NewError(d.ErrorCodeInvalidConfig, "signing seed differs from persisted public trust; explicit recovery required")
		}
	} else {
		serverID, e := ids.NewID()
		if e != nil {
			return e
		}
		keyID, e := ids.NewID()
		if e != nil {
			return e
		}
		pinned = d.ServerIdentity{ID: serverID, IssuerKeyID: keyID, PublicKey: base64.StdEncoding.EncodeToString(public), Fingerprint: fp, CreatedAt: systemClock{}.Now().UTC()}
	}
	floor := config.Signing.AcceptanceFloor
	if floor == "" {
		floor = d.AcceptanceDirect
	}
	if config.Auth.IndependentReviewer {
		floor = d.AcceptanceIndependentReview
	}
	if err = service.ConfigureSignedIssuer(ctx, &localIssuer{identity: pinned, private: private}, floor); err != nil {
		clear(private)
		return err
	}
	security.ServerID = pinned.ID.String()
	return nil
}
