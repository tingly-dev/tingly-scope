package module

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// StateModule is the base interface for stateful modules
type StateModule interface {
	// StateDict returns the state dictionary for serialization
	StateDict() map[string]any

	// LoadStateDict loads the state from a dictionary
	LoadStateDict(ctx context.Context, state map[string]any) error
}

// StateModuleBase provides a base implementation for stateful modules
type StateModuleBase struct {
	mu       sync.RWMutex
	state    map[string]any
	initArgs map[string]any
}

// NewStateModuleBase creates a new state module base
func NewStateModuleBase() *StateModuleBase {
	return &StateModuleBase{
		state:    make(map[string]any),
		initArgs: make(map[string]any),
	}
}

// StateDict returns the state dictionary
func (m *StateModuleBase) StateDict() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a deep copy of the state
	result := make(map[string]any)
	for k, v := range m.state {
		result[k] = deepCopy(v)
	}

	// Add init args
	if len(m.initArgs) > 0 {
		result["_init_args"] = m.initArgs
	}

	return result
}

// LoadStateDict loads the state from a dictionary
func (m *StateModuleBase) LoadStateDict(ctx context.Context, state map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Extract init args if present
	if initArgs, ok := state["_init_args"].(map[string]any); ok {
		m.initArgs = initArgs
	}

	// Load state (exclude _init_args)
	for k, v := range state {
		if k != "_init_args" {
			m.state[k] = deepCopy(v)
		}
	}

	return nil
}

// Set sets a value in the state
func (m *StateModuleBase) Set(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state[key] = value
}

// Get gets a value from the state
func (m *StateModuleBase) Get(key string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.state[key]
	return v, ok
}

// Delete deletes a value from the state
func (m *StateModuleBase) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.state, key)
}

// Clear clears all state
func (m *StateModuleBase) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = make(map[string]any)
}

// SetInitArg sets an initialization argument
func (m *StateModuleBase) SetInitArg(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initArgs[key] = value
}

// GetInitArg gets an initialization argument
func (m *StateModuleBase) GetInitArg(key string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.initArgs[key]
	return v, ok
}

// deepCopy creates a deep copy of a value
func deepCopy(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any)
		for k, vv := range val {
			result[k] = deepCopy(vv)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, vv := range val {
			result[i] = deepCopy(vv)
		}
		return result
	case string, int, int64, float64, bool:
		return v
	default:
		// For other types, try to marshal and unmarshal as JSON
		data, err := json.Marshal(v)
		if err != nil {
			return v
		}
		var result any
		err = json.Unmarshal(data, &result)
		if err != nil {
			return v
		}
		return result
	}
}

// MergeStateModules merges multiple state modules into one state dict
func MergeStateModules(modules map[string]StateModule) (map[string]any, error) {
	result := make(map[string]any)

	for name, module := range modules {
		state := module.StateDict()
		if existing, ok := result[name]; ok {
			return nil, fmt.Errorf("duplicate module name: %s (existing: %v)", name, existing)
		}
		result[name] = state
	}

	return result, nil
}

// LoadStateModules loads state into multiple modules
func LoadStateModules(ctx context.Context, modules map[string]StateModule, state map[string]any) error {
	for name, module := range modules {
		moduleState, ok := state[name]
		if !ok {
			continue
		}

		stateDict, ok := moduleState.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid state format for module %s: expected map[string]any, got %T", name, moduleState)
		}

		if err := module.LoadStateDict(ctx, stateDict); err != nil {
			return fmt.Errorf("failed to load state for module %s: %w", name, err)
		}
	}

	return nil
}
