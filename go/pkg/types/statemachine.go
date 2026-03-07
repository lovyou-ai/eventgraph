package types

// LifecycleState represents a primitive's lifecycle state.
// Transitions are enforced — invalid transitions return an error.
type LifecycleState string

const (
	LifecycleDormant      LifecycleState = "dormant"
	LifecycleActivating   LifecycleState = "activating"
	LifecycleActive       LifecycleState = "active"
	LifecycleProcessing   LifecycleState = "processing"
	LifecycleEmitting     LifecycleState = "emitting"
	LifecycleDeactivating LifecycleState = "deactivating"
)

var validLifecycleTransitions = map[LifecycleState][]LifecycleState{
	LifecycleDormant:      {LifecycleActivating},
	LifecycleActivating:   {LifecycleActive, LifecycleDormant},
	LifecycleActive:       {LifecycleProcessing, LifecycleDeactivating},
	LifecycleProcessing:   {LifecycleEmitting, LifecycleActive},
	LifecycleEmitting:     {LifecycleActive},
	LifecycleDeactivating: {LifecycleDormant},
}

var validLifecycleStates = map[LifecycleState]bool{
	LifecycleDormant: true, LifecycleActivating: true, LifecycleActive: true,
	LifecycleProcessing: true, LifecycleEmitting: true, LifecycleDeactivating: true,
}

// IsValid returns true if the lifecycle state is a known state.
func (s LifecycleState) IsValid() bool { return validLifecycleStates[s] }

// TransitionTo attempts to transition to the target state.
// Returns the new state on success, or an error if the transition is invalid.
func (s LifecycleState) TransitionTo(target LifecycleState) (LifecycleState, error) {
	valid := validLifecycleTransitions[s]
	for _, v := range valid {
		if v == target {
			return target, nil
		}
	}
	targets := make([]LifecycleState, len(valid))
	copy(targets, valid)
	return s, &InvalidLifecycleTransitionError{
		From:         s,
		To:           target,
		ValidTargets: targets,
	}
}

// ValidTransitions returns the list of valid target states from this state.
func (s LifecycleState) ValidTransitions() []LifecycleState {
	valid := validLifecycleTransitions[s]
	result := make([]LifecycleState, len(valid))
	copy(result, valid)
	return result
}

// ActorStatus represents an actor's status in the system.
// Memorial is terminal — once memorialised, the actor's graph is preserved forever
// but the actor can never emit new events.
type ActorStatus string

const (
	ActorStatusActive    ActorStatus = "active"
	ActorStatusSuspended ActorStatus = "suspended"
	ActorStatusMemorial  ActorStatus = "memorial"
)

var validActorTransitions = map[ActorStatus][]ActorStatus{
	ActorStatusActive:    {ActorStatusSuspended, ActorStatusMemorial},
	ActorStatusSuspended: {ActorStatusActive, ActorStatusMemorial},
	ActorStatusMemorial:  {}, // terminal
}

var validActorStatuses = map[ActorStatus]bool{
	ActorStatusActive: true, ActorStatusSuspended: true, ActorStatusMemorial: true,
}

// IsValid returns true if the actor status is a known status.
func (s ActorStatus) IsValid() bool { return validActorStatuses[s] }

// TransitionTo attempts to transition to the target status.
// Returns the new status on success, or an error if the transition is invalid.
func (s ActorStatus) TransitionTo(target ActorStatus) (ActorStatus, error) {
	valid := validActorTransitions[s]
	for _, v := range valid {
		if v == target {
			return target, nil
		}
	}
	targets := make([]ActorStatus, len(valid))
	copy(targets, valid)
	return s, &InvalidActorTransitionError{
		From:         s,
		To:           target,
		ValidTargets: targets,
	}
}

// ValidTransitions returns the list of valid target statuses from this status.
func (s ActorStatus) ValidTransitions() []ActorStatus {
	valid := validActorTransitions[s]
	result := make([]ActorStatus, len(valid))
	copy(result, valid)
	return result
}
