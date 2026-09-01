//go:build !cgo

package cloudhsm

import (
	"errors"

	"github.com/sergiorandria/hsm-go-client/hsm"
)

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
