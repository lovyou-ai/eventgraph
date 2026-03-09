package agent

import "testing"

func TestOperationalStateString(t *testing.T) {
	tests := []struct {
		state OperationalState
		want  string
	}{
		{StateIdle, "Idle"},
		{StateProcessing, "Processing"},
		{StateWaiting, "Waiting"},
		{StateEscalating, "Escalating"},
		{StateRefusing, "Refusing"},
		{StateSuspended, "Suspended"},
		{StateRetiring, "Retiring"},
		{StateRetired, "Retired"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", int(tt.state), got, tt.want)
		}
	}
}

func TestValidTransitions(t *testing.T) {
	tests := []struct {
		from OperationalState
		to   OperationalState
		ok   bool
	}{
		// Idle transitions
		{StateIdle, StateProcessing, true},
		{StateIdle, StateSuspended, true},
		{StateIdle, StateRetiring, true},
		{StateIdle, StateWaiting, false},
		{StateIdle, StateRetired, false},

		// Processing transitions
		{StateProcessing, StateIdle, true},
		{StateProcessing, StateWaiting, true},
		{StateProcessing, StateEscalating, true},
		{StateProcessing, StateRefusing, true},
		{StateProcessing, StateRetiring, true},
		{StateProcessing, StateSuspended, false},

		// Waiting transitions
		{StateWaiting, StateProcessing, true},
		{StateWaiting, StateIdle, true},
		{StateWaiting, StateRetiring, true},
		{StateWaiting, StateEscalating, false},

		// Escalating transitions
		{StateEscalating, StateWaiting, true},
		{StateEscalating, StateIdle, true},
		{StateEscalating, StateProcessing, false},

		// Refusing transitions
		{StateRefusing, StateIdle, true},
		{StateRefusing, StateProcessing, false},

		// Suspended transitions
		{StateSuspended, StateIdle, true},
		{StateSuspended, StateRetiring, true},
		{StateSuspended, StateProcessing, false},

		// Retiring transitions
		{StateRetiring, StateRetired, true},
		{StateRetiring, StateIdle, false},

		// Retired — terminal, no transitions
		{StateRetired, StateIdle, false},
		{StateRetired, StateRetired, false},
	}

	for _, tt := range tests {
		result, err := tt.from.TransitionTo(tt.to)
		if tt.ok {
			if err != nil {
				t.Errorf("%s → %s: unexpected error: %v", tt.from, tt.to, err)
			}
			if result != tt.to {
				t.Errorf("%s → %s: got %s, want %s", tt.from, tt.to, result, tt.to)
			}
		} else {
			if err == nil {
				t.Errorf("%s → %s: expected error, got nil", tt.from, tt.to)
			}
			if result != tt.from {
				t.Errorf("%s → %s: on error, state should remain %s, got %s", tt.from, tt.to, tt.from, result)
			}
		}
	}
}

func TestIsTerminal(t *testing.T) {
	if !StateRetired.IsTerminal() {
		t.Error("Retired should be terminal")
	}
	for _, s := range []OperationalState{StateIdle, StateProcessing, StateWaiting, StateEscalating, StateRefusing, StateSuspended, StateRetiring} {
		if s.IsTerminal() {
			t.Errorf("%s should not be terminal", s)
		}
	}
}

func TestCanAct(t *testing.T) {
	if !StateProcessing.CanAct() {
		t.Error("Processing should allow acting")
	}
	for _, s := range []OperationalState{StateIdle, StateWaiting, StateEscalating, StateRefusing, StateSuspended, StateRetiring, StateRetired} {
		if s.CanAct() {
			t.Errorf("%s should not allow acting", s)
		}
	}
}

func TestFullLifecycle(t *testing.T) {
	// Idle → Processing → Waiting → Processing → Idle → Retiring → Retired
	s := StateIdle
	var err error

	s, err = s.TransitionTo(StateProcessing)
	if err != nil {
		t.Fatalf("Idle → Processing: %v", err)
	}

	s, err = s.TransitionTo(StateWaiting)
	if err != nil {
		t.Fatalf("Processing → Waiting: %v", err)
	}

	s, err = s.TransitionTo(StateProcessing)
	if err != nil {
		t.Fatalf("Waiting → Processing: %v", err)
	}

	s, err = s.TransitionTo(StateIdle)
	if err != nil {
		t.Fatalf("Processing → Idle: %v", err)
	}

	s, err = s.TransitionTo(StateRetiring)
	if err != nil {
		t.Fatalf("Idle → Retiring: %v", err)
	}

	s, err = s.TransitionTo(StateRetired)
	if err != nil {
		t.Fatalf("Retiring → Retired: %v", err)
	}

	if !s.IsTerminal() {
		t.Error("should be terminal after retiring")
	}
}

func TestEscalationCycle(t *testing.T) {
	// Processing → Escalating → Waiting → Processing → Idle
	s := StateProcessing
	var err error

	s, err = s.TransitionTo(StateEscalating)
	if err != nil {
		t.Fatalf("Processing → Escalating: %v", err)
	}

	s, err = s.TransitionTo(StateWaiting)
	if err != nil {
		t.Fatalf("Escalating → Waiting: %v", err)
	}

	s, err = s.TransitionTo(StateProcessing)
	if err != nil {
		t.Fatalf("Waiting → Processing: %v", err)
	}

	s, err = s.TransitionTo(StateIdle)
	if err != nil {
		t.Fatalf("Processing → Idle: %v", err)
	}
}

func TestRefusalCycle(t *testing.T) {
	// Processing → Refusing → Idle
	s := StateProcessing
	var err error

	s, err = s.TransitionTo(StateRefusing)
	if err != nil {
		t.Fatalf("Processing → Refusing: %v", err)
	}

	s, err = s.TransitionTo(StateIdle)
	if err != nil {
		t.Fatalf("Refusing → Idle: %v", err)
	}

	if s != StateIdle {
		t.Errorf("expected Idle, got %s", s)
	}
}
