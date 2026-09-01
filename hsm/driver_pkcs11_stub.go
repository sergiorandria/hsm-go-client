//go:build !cgo

package hsm

import (
	"errors"

	"github.com/sergiorandria/hsm-go-client/hsm/pkcs11"
)

func NewPKCS11Driver(cfg pkcs11.Config) (Driver, error) {
	return nil, errors.New("pkcs11: cgo required; build with CGO_ENABLED=1")
}
