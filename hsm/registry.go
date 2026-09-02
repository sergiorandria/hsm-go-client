package hsm

import "sync"

// Factory creates a Driver from DriverConfig and Options. Registered via RegisterBackend.
type Factory func(cfg DriverConfig, opts ...Option) (Driver, error)

var (
	registryMu sync.RWMutex
	registry   = map[string]Factory{}
)

// RegisterBackend registers a backend factory. Called from init() in backend packages.
// Thread-safe and allows third-party backends.
func RegisterBackend(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

func getFactory(name string) (Factory, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	f, ok := registry[name]
	return f, ok
}

// ListBackends returns registered backend names for observability.
func ListBackends() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	return out
}
