// Package agent implements the 28 agent primitives and 8 named compositions.
// Agent primitives define what an agent is (structural), what an agent does
// (operational), how agents relate (relational), and how operations are
// modified (modal). All primitives operate at Layer 1 (Agency).
package agent

import "fmt"

// OperationalState represents the agent's current operational state.
// Follows a strict FSM with enforced valid transitions.
type OperationalState int

const (
	StateIdle       OperationalState = iota // ready for work
	StateProcessing                         // actively working
	StateWaiting                            // waiting for external input
	StateEscalating                         // passing issue upward
	StateRefusing                           // declining an action
	StateSuspended                          // temporarily disabled
	StateRetiring                           // graceful shutdown in progress
	StateRetired                            // terminal state
)

var operationalStateNames = map[OperationalState]string{
	StateIdle:       "Idle",
	StateProcessing: "Processing",
	StateWaiting:    "Waiting",
	StateEscalating: "Escalating",
	StateRefusing:   "Refusing",
	StateSuspended:  "Suspended",
	StateRetiring:   "Retiring",
	StateRetired:    "Retired",
}

func (s OperationalState) String() string {
	if name, ok := operationalStateNames[s]; ok {
		return name
	}
	return fmt.Sprintf("OperationalState(%d)", int(s))
}

// validTransitions defines the FSM from the spec:
//
//	Idle → {Processing, Suspended, Retiring}
//	Processing → {Idle, Waiting, Escalating, Refusing, Retiring}
//	Waiting → {Processing, Idle, Retiring}
//	Escalating → {Waiting, Idle}
//	Refusing → {Idle}
//	Suspended → {Idle, Retiring}
//	Retiring → {Retired}
//	Retired → {} (terminal)
var validTransitions = map[OperationalState][]OperationalState{
	StateIdle:       {StateProcessing, StateSuspended, StateRetiring},
	StateProcessing: {StateIdle, StateWaiting, StateEscalating, StateRefusing, StateRetiring},
	StateWaiting:    {StateProcessing, StateIdle, StateRetiring},
	StateEscalating: {StateWaiting, StateIdle},
	StateRefusing:   {StateIdle},
	StateSuspended:  {StateIdle, StateRetiring},
	StateRetiring:   {StateRetired},
	StateRetired:    {},
}

// TransitionTo validates and returns the new state if the transition is valid.
func (s OperationalState) TransitionTo(target OperationalState) (OperationalState, error) {
	valid, ok := validTransitions[s]
	if !ok {
		return s, fmt.Errorf("unknown operational state: %v", s)
	}
	for _, v := range valid {
		if v == target {
			return target, nil
		}
	}
	return s, fmt.Errorf("invalid transition: %s → %s", s, target)
}

// IsTerminal returns true if this is a terminal state.
func (s OperationalState) IsTerminal() bool {
	return s == StateRetired
}

// CanAct returns true if the agent can perform actions in this state.
func (s OperationalState) CanAct() bool {
	return s == StateProcessing
}
