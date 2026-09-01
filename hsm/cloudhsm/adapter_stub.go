//go:build !cgo

package cloudhsm

import (
	"errors"

	"github.com/sergiorandria/hsm-go-client/hsm"
)

func init() {
	hsm.RegisterBackend("cloudhsm", func(cfg hsm.DriverConfig, opts ...hsm.Option) (hsm.Driver, error) {
		return NewDriver(Config{
			LibraryPath: cfg.PKCS11.LibraryPath,
			TokenLabel:  cfg.PKCS11.TokenLabel,
			PIN:         cfg.PKCS11.PIN,
		})
	})
	hsm.RegisterBackend("aws-cloudhsm", func(cfg hsm.DriverConfig, opts ...hsm.Option) (hsm.Driver, error) {
		return NewDriver(Config{
			LibraryPath: cfg.PKCS11.LibraryPath,
			TokenLabel:  cfg.PKCS11.TokenLabel,
			PIN:         cfg.PKCS11.PIN,
		})
	})
}

type Config struct {
	LibraryPath string
	TokenLabel  string
	PIN         string
	SlotID      *uint
	MaxSessions int
}

func NewDriver(cfg Config) (hsm.Driver, error) {
	return nil, errors.New("cloudhsm: cgo required; build with CGO_ENABLED=1")
}
