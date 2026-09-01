//go:build !cgo

package luna

import (
	"errors"

	"github.com/sergiorandria/hsm-go-client/hsm"
)

type Config struct {
	LibraryPath       string
	TokenLabel        string
	PIN               string
	SlotID            *uint
	MaxSessions       int
	HASlotDescription string
}

func NewDriver(cfg Config) (hsm.Driver, error) {
	return nil, errors.New("luna: cgo required; build with CGO_ENABLED=1")
}
