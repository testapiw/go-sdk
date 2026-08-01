// Package factory builds provider adapters by name. It is the composition
// root: it imports the built-in providers and pre-registers them, so the
// application just asks for a provider by name and gets a contract.Provider
// without any registration boilerplate.
package factory

import (
	"fmt"

	httpx "github.com/testapiw/go-sdk/http-sdk/client"

	"github.com/testapiw/go-sdk/provider-sdk/provider/coingecko"
	"github.com/testapiw/go-sdk/provider-sdk/provider/contract"
)

// Config is the generic configuration passed to a provider constructor.
// It carries only infrastructure that is common to every provider; each
// provider owns its own settings and reads credentials from the environment.
type Config struct {
	// Name selects the provider (e.g. "coingecko").
	Name string

	// Doer is the underlying HTTP client. Nil uses the default client.
	Doer httpx.Doer
}

// Constructor builds a provider adapter. The name is passed in so a
// constructor never hardcodes its own registration name.
type Constructor func(name string, cfg Config) (contract.Provider, error)

// Factory builds provider adapters by name.
type Factory struct {
	constructors map[string]Constructor
}

// New returns a factory pre-registered with the built-in providers.
func New() *Factory {
	f := &Factory{
		constructors: make(map[string]Constructor),
	}

	RegisterProvider(f, "coingecko", coingecko.New, coingecko.DefaultConfig)

	return f
}

// RegisterProvider binds a provider name to its adapter type. It adapts a
// provider's own constructor (which takes a provider-specific config and a
// default-config factory) into the generic Constructor, so registration is a
// single line with no closure. P is the concrete adapter type; it must
// satisfy contract.Provider.
func RegisterProvider[T any, P contract.Provider](
	f *Factory,
	name string,
	new func(name string, cfg T, doer httpx.Doer) (P, error),
	defaults func() T,
) {
	f.Register(name, func(name string, cfg Config) (contract.Provider, error) {
		return new(name, defaults(), cfg.Doer)
	})
}

// Register adds or replaces a provider constructor by name.
func (f *Factory) Register(name string, c Constructor) {
	f.constructors[name] = c
}

// Create builds a provider adapter for the named provider using its default
// configuration (credentials are read from the environment by the provider
// itself).
func (f *Factory) Create(name string) (contract.Provider, error) {
	return f.CreateWithConfig(Config{Name: name})
}

// CreateWithConfig builds a provider adapter for the named provider with an
// optional custom HTTP client.
func (f *Factory) CreateWithConfig(cfg Config) (contract.Provider, error) {
	c, ok := f.constructors[cfg.Name]
	if !ok {
		return nil, fmt.Errorf("factory: unknown provider %q", cfg.Name)
	}

	return c(cfg.Name, cfg)
}
