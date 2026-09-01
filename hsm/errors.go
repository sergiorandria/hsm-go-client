package hsm

import "errors"

// Sentinel errors for production handling. Wrap with %w and check via errors.Is/As.
// Keeps drivers thin and callers scalable (retry, circuit-breaker).
var (
	ErrNotFound         = errors.New("hsm: not found")
	ErrAlreadyExists    = errors.New("hsm: already exists")
	ErrUnauthorized     = errors.New("hsm: unauthorized")
	ErrInvalidArgument  = errors.New("hsm: invalid argument")
	ErrNotSupported     = errors.New("hsm: not supported")
	ErrTimeout          = errors.New("hsm: timeout")
	ErrDeviceNotReady   = errors.New("hsm: device not ready")
	ErrMechanismInvalid = errors.New("hsm: mechanism invalid")
)

// Error wraps backend errors with operation context and preserves sentinel via %w.
type Error struct {
	Op      string
	KeyID   string
	Backend string
	Err     error
}

func (e *Error) Error() string {
	if e.KeyID != "" {
		return e.Backend + "/" + e.Op + " key=" + e.KeyID + ": " + e.Err.Error()
	}
	return e.Backend + "/" + e.Op + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

func wrapError(backend, op string, keyID string, err error, sentinel error) error {
	if err == nil {
		return nil
	}
	// Preserve sentinel for errors.Is
	if sentinel != nil {
		err = errors.Join(sentinel, err)
	}
	return &Error{Backend: backend, Op: op, KeyID: keyID, Err: err}
}
