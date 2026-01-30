package message

import (
	"context"
)

// Injector transforms a message before sending to LLM
// Each injector can prepend content blocks to provide additional context
type Injector interface {
	// Inject adds content blocks to the beginning of the message
	// Returns the modified message (or original if no injection)
	Inject(ctx context.Context, msg *Msg) *Msg

	// Name returns the injector's name for debugging
	Name() string
}

// InjectorChain manages a sequence of injectors
// Injectors are applied in order, each prepending to the previous result
type InjectorChain struct {
	injectors []Injector
}

// NewInjectorChain creates a new injection chain with the given injectors
func NewInjectorChain(injectors ...Injector) *InjectorChain {
	return &InjectorChain{
		injectors: injectors,
	}
}

// ApplyAll runs all injectors in sequence on the message
// Each injector receives the output of the previous injector
func (c *InjectorChain) ApplyAll(ctx context.Context, msg *Msg) *Msg {
	if c == nil || len(c.injectors) == 0 {
		return msg
	}

	result := msg
	for _, injector := range c.injectors {
		result = injector.Inject(ctx, result)
	}
	return result
}

// Add appends an injector to the end of the chain
// Returns the chain for method chaining
func (c *InjectorChain) Add(injector Injector) *InjectorChain {
	if c == nil {
		return NewInjectorChain(injector)
	}
	c.injectors = append(c.injectors, injector)
	return c
}

// Insert inserts an injector at a specific position
// If index is out of bounds, appends to the end
func (c *InjectorChain) Insert(index int, injector Injector) *InjectorChain {
	if c == nil {
		return NewInjectorChain(injector)
	}
	if index < 0 || index >= len(c.injectors) {
		return c.Add(injector)
	}

	c.injectors = append(c.injectors[:index], append([]Injector{injector}, c.injectors[index:]...)...)
	return c
}

// Remove removes an injector by name
func (c *InjectorChain) Remove(name string) *InjectorChain {
	if c == nil {
		return nil
	}

	newInjectors := make([]Injector, 0, len(c.injectors))
	for _, inj := range c.injectors {
		if inj.Name() != name {
			newInjectors = append(newInjectors, inj)
		}
	}
	c.injectors = newInjectors
	return c
}

// Clear removes all injectors from the chain
func (c *InjectorChain) Clear() *InjectorChain {
	if c == nil {
		return nil
	}
	c.injectors = nil
	return c
}

// Count returns the number of injectors in the chain
func (c *InjectorChain) Count() int {
	if c == nil {
		return 0
	}
	return len(c.injectors)
}

// Names returns the names of all injectors in the chain
func (c *InjectorChain) Names() []string {
	if c == nil {
		return nil
	}

	names := make([]string, 0, len(c.injectors))
	for _, inj := range c.injectors {
		names = append(names, inj.Name())
	}
	return names
}
