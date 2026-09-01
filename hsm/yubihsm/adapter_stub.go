//go:build !cgo

package yubihsm

import (
	"errors"

	"github.com/sergiorandria/hsm-go-client/hsm"
)

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
