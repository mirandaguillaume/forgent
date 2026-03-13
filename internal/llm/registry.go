package llm

import "fmt"

// ProviderFactory creates a Provider given an API key.
type ProviderFactory func(apiKey string) Provider

var providers = make(map[string]ProviderFactory)

// RegisterProvider registers a named provider factory.
func RegisterProvider(name string, factory ProviderFactory) {
	providers[name] = factory
}

// GetProvider returns a Provider instance for the given name and API key.
func GetProvider(name, apiKey string) (Provider, error) {
	factory, ok := providers[name]
	if !ok {
		available := make([]string, 0, len(providers))
		for k := range providers {
			available = append(available, k)
		}
		return nil, fmt.Errorf("unknown provider %q (available: %v)", name, available)
	}
	return factory(apiKey), nil
}
