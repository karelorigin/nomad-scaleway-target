package plugin

import (
	"testing"
)

// TestState tests switching states
func TestState(t *testing.T) {
	var state State
	if state.Get() != StateIdle {
		t.Error("Expected default state to be idle")
	}

	state.SetActive()
	if state.Get() != StateActive {
		t.Error("Expected state to be active after SetActive")
	}

	state.SetIdle()
	if state.Get() != StateIdle {
		t.Error("Expected state to be idle after SetIdle")
	}
}
