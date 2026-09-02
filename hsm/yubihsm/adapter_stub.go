//go:build !cgo

package yubihsm

import (
	"errors"

	"github.com/sergiorandria/hsm-go-client/hsm"
)

func init() {
	hsm.RegisterBackend("yubihsm", func(cfg hsm.DriverConfig, opts ...hsm.Option) (hsm.Driver, error) {
		return NewDriver(Config{
			LibraryPath: cfg.PKCS11.LibraryPath,
			TokenLabel:  cfg.PKCS11.TokenLabel,
			PIN:         cfg.PKCS11.PIN,
			SlotID:      cfg.PKCS11.SlotID,
		})
	})
}

type Config struct {
	LibraryPath  string
	TokenLabel   string
	PIN          string
	SlotID       *uint
	MaxSessions  int
	ConnectorURL string
}

func NewDriver(cfg Config) (hsm.Driver, error) {
	return nil, errors.New("yubihsm: cgo required; build with CGO_ENABLED=1")
}
