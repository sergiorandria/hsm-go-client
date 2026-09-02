package hsm

import (
	"fmt"
	"strings"
)

// Validate checks DriverConfig for required fields. Scalable config validation.
func (c DriverConfig) Validate() error {
	if strings.TrimSpace(c.Backend) == "" {
		return fmt.Errorf("%w: Backend required (http, pkcs11, yubihsm, luna, cloudhsm)", ErrInvalidArgument)
	}
	switch c.Backend {
	case "http", "microcontroller", "mcu", "esp32", "yubihsm", "yubihsm2", "luna", "cloudhsm", "aws-cloudhsm", "pkcs11":
		// known
	default:
		// Allow third-party via registry; but still check if registered
		if _, ok := getFactory(c.Backend); !ok {
			return fmt.Errorf("%w: unknown backend %q not registered (registered: %v)", ErrInvalidArgument, c.Backend, ListBackends())
		}
	}
	if c.Backend == "http" || c.Backend == "mcu" || c.Backend == "esp32" || c.Backend == "microcontroller" {
		if err := c.HTTP.Validate(); err != nil {
			return err
		}
	}
	if c.Backend == "pkcs11" || c.Backend == "yubihsm" || c.Backend == "yubihsm2" || c.Backend == "luna" || c.Backend == "cloudhsm" || c.Backend == "aws-cloudhsm" {
		if err := c.PKCS11.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c HTTPConfig) Validate() error {
	if c.TimeoutSeconds < 0 {
		return fmt.Errorf("%w: HTTP TimeoutSeconds must be >=0", ErrInvalidArgument)
	}
	if c.ChunkSize < 0 {
		return fmt.Errorf("%w: HTTP ChunkSize must be >=0", ErrInvalidArgument)
	}
	return nil
}

func (c PKCS11Config) Validate() error {
	if strings.TrimSpace(c.LibraryPath) == "" {
		return fmt.Errorf("%w: PKCS11 LibraryPath required", ErrInvalidArgument)
	}
	return nil
}

func (s KeySpec) Validate() error {
	if strings.TrimSpace(s.Label) == "" {
		return fmt.Errorf("%w: KeySpec.Label required", ErrInvalidArgument)
	}
	if s.Bits != 0 && s.Bits != 2048 && s.Bits != 3072 && s.Bits != 4096 {
		return fmt.Errorf("%w: KeySpec.Bits must be 2048/3072/4096", ErrInvalidArgument)
	}
	return nil
}

func (id KeyID) Validate() error {
	if strings.TrimSpace(id.Label) == "" && len(id.ID) == 0 {
		return fmt.Errorf("%w: KeyID.Label or ID required", ErrInvalidArgument)
	}
	return nil
}
