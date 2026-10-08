package woscli

import keyring "github.com/zalando/go-keyring"

// OSKeyring uses the operating system credential service. Errors never select
// an unencrypted file backend or cause secret values to enter diagnostics.
type OSKeyring struct{}

func (OSKeyring) Get(service, user string) (string, error) { return keyring.Get(service, user) }
func (OSKeyring) Set(service, user, value string) error    { return keyring.Set(service, user, value) }
func (OSKeyring) Delete(service, user string) error        { return keyring.Delete(service, user) }
