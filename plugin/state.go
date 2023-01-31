package plugin

import "sync/atomic"

// State represents a plugin state
type State int32

// A set of possible plugin states
const (
	StateIdle State = iota
	StateActive
)

// SetIdle changes the state to idle
func (s *State) SetIdle() {
	atomic.SwapInt32((*int32)(s), int32(StateIdle))
}

// SetActive changes the state to active
func (s *State) SetActive() {
	atomic.SwapInt32((*int32)(s), int32(StateActive))
}

// Get returns the current state
func (s *State) Get() State {
	return State(atomic.LoadInt32((*int32)(s)))
}
