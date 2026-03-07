package primitive

import (
	"fmt"
	"sort"
	"sync"

	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

// Registry holds all registered primitives and their mutable state.
// Thread-safe for concurrent access.
type Registry struct {
	mu         sync.RWMutex
	primitives map[types.PrimitiveID]Primitive
	states     map[types.PrimitiveID]*mutableState
	ordered    []types.PrimitiveID // sorted by layer then ID
}

type mutableState struct {
	activation types.Activation
	lifecycle  types.LifecycleState
	state      map[string]any
	lastTick   types.Tick
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		primitives: make(map[types.PrimitiveID]Primitive),
		states:     make(map[types.PrimitiveID]*mutableState),
	}
}

// Register adds a primitive to the registry. Starts in Dormant lifecycle.
func (r *Registry) Register(p Primitive) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := p.ID()
	if _, exists := r.primitives[id]; exists {
		return fmt.Errorf("primitive %q already registered", id.Value())
	}

	r.primitives[id] = p
	r.states[id] = &mutableState{
		activation: types.MustActivation(0.0),
		lifecycle:  types.LifecycleDormant,
		state:      make(map[string]any),
		lastTick:   types.MustTick(0),
	}
	r.rebuildOrder()
	return nil
}

// Get returns a primitive by ID.
func (r *Registry) Get(id types.PrimitiveID) (Primitive, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.primitives[id]
	return p, ok
}

// All returns all primitives in layer order.
func (r *Registry) All() []Primitive {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Primitive, 0, len(r.ordered))
	for _, id := range r.ordered {
		result = append(result, r.primitives[id])
	}
	return result
}

// AllStates returns a snapshot of all primitive states (deep copy).
func (r *Registry) AllStates() map[types.PrimitiveID]PrimitiveState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[types.PrimitiveID]PrimitiveState, len(r.primitives))
	for id, p := range r.primitives {
		ms := r.states[id]
		stateCopy := deepCopyState(ms.state)
		result[id] = PrimitiveState{
			ID:         id,
			Layer:      p.Layer(),
			Lifecycle:  ms.lifecycle,
			Activation: ms.activation,
			Cadence:    p.Cadence(),
			state:      stateCopy,
			LastTick:   ms.lastTick,
		}
	}
	return result
}

// Lifecycle returns the current lifecycle state for a primitive.
func (r *Registry) Lifecycle(id types.PrimitiveID) types.LifecycleState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if ms, ok := r.states[id]; ok {
		return ms.lifecycle
	}
	return types.LifecycleDormant
}

// LastTick returns the last tick a primitive was invoked.
func (r *Registry) LastTick(id types.PrimitiveID) types.Tick {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if ms, ok := r.states[id]; ok {
		return ms.lastTick
	}
	return types.MustTick(0)
}

// SetLastTick records when a primitive was last invoked.
func (r *Registry) SetLastTick(id types.PrimitiveID, tick types.Tick) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ms, ok := r.states[id]; ok {
		ms.lastTick = tick
	}
}

// SetLifecycle transitions a primitive's lifecycle state.
func (r *Registry) SetLifecycle(id types.PrimitiveID, state types.LifecycleState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ms, ok := r.states[id]
	if !ok {
		return fmt.Errorf("primitive %q not found", id.Value())
	}
	newState, err := ms.lifecycle.TransitionTo(state)
	if err != nil {
		return err
	}
	ms.lifecycle = newState
	return nil
}

// SetActivation updates a primitive's activation level.
func (r *Registry) SetActivation(id types.PrimitiveID, level types.Activation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ms, ok := r.states[id]
	if !ok {
		return fmt.Errorf("primitive %q not found", id.Value())
	}
	ms.activation = level
	return nil
}

// UpdateState sets a key-value pair in a primitive's state.
func (r *Registry) UpdateState(id types.PrimitiveID, key string, value any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ms, ok := r.states[id]
	if !ok {
		return fmt.Errorf("primitive %q not found", id.Value())
	}
	ms.state[key] = deepCopyValue(value)
	return nil
}

// Activate transitions a primitive from Dormant → Activating → Active atomically.
func (r *Registry) Activate(id types.PrimitiveID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ms, ok := r.states[id]
	if !ok {
		return fmt.Errorf("primitive %q not found", id.Value())
	}
	activating, err := ms.lifecycle.TransitionTo(types.LifecycleActivating)
	if err != nil {
		return err
	}
	active, err := activating.TransitionTo(types.LifecycleActive)
	if err != nil {
		return err
	}
	ms.lifecycle = active
	return nil
}

// Count returns the number of registered primitives.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.primitives)
}

// deepCopyState creates a deep copy of a state map.
// Handles nested maps and slices; scalar values (string, int, float64, bool) are copied directly.
func deepCopyState(state map[string]any) map[string]any {
	if state == nil {
		return nil
	}
	if len(state) == 0 {
		return make(map[string]any)
	}
	cp := make(map[string]any, len(state))
	for k, v := range state {
		cp[k] = deepCopyValue(v)
	}
	return cp
}

func deepCopyValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return deepCopyState(val)
	case []any:
		cp := make([]any, len(val))
		for i, elem := range val {
			cp[i] = deepCopyValue(elem)
		}
		return cp
	case []string:
		cp := make([]string, len(val))
		copy(cp, val)
		return cp
	case []byte:
		cp := make([]byte, len(val))
		copy(cp, val)
		return cp
	case []int:
		cp := make([]int, len(val))
		copy(cp, val)
		return cp
	case []float64:
		cp := make([]float64, len(val))
		copy(cp, val)
		return cp
	default:
		return v // scalars are immutable
	}
}

func (r *Registry) rebuildOrder() {
	r.ordered = make([]types.PrimitiveID, 0, len(r.primitives))
	for id := range r.primitives {
		r.ordered = append(r.ordered, id)
	}
	sort.Slice(r.ordered, func(i, j int) bool {
		pi := r.primitives[r.ordered[i]]
		pj := r.primitives[r.ordered[j]]
		if pi.Layer().Value() != pj.Layer().Value() {
			return pi.Layer().Value() < pj.Layer().Value()
		}
		return r.ordered[i].Value() < r.ordered[j].Value()
	})
}
