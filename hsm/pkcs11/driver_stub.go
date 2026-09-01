//go:build !cgo

package pkcs11

import (
	"errors"
)

type Config struct {
	LibraryPath string
	SlotID      *uint
	TokenLabel  string
	PIN         string
	MaxSessions int
	SO_PIN      string
}

func NewDriver(cfg Config) (interface{}, error) {
	return nil, errors.New("pkcs11: cgo required; build with CGO_ENABLED=1")
}
