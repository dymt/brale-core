package e2e

import "fmt"

// Suite defines the interface all E2E test suites must implement.
type Suite interface {
	Name() string
	Run(ctx *Context) SuiteResult
}

// RegisteredSuites maps suite names to factory functions.
var registeredSuites = map[string]func() Suite{}

// Register adds a suite factory to the global registry.
func Register(name string, factory func() Suite) {
	registeredSuites[name] = factory
}

// GetSuite returns a suite instance by name.
func GetSuite(name string) (Suite, error) {
	factory, ok := registeredSuites[name]
	if !ok {
		return nil, fmt.Errorf("unknown suite: %s (available: %v)", name, SuiteNames())
	}
	return factory(), nil
}

// SuiteNames returns all registered suite names.
func SuiteNames() []string {
	names := make([]string, 0, len(registeredSuites))
	for name := range registeredSuites {
		names = append(names, name)
	}
	return names
}
