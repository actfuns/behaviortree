package core

import "time"

// WakeUpSignal provides a way to signal a waiting tree to tick again.
// Mirrors C++ BT::WakeUpSignal.
//
// Uses a buffered channel so Emit is non-blocking and WaitFor can use
// a simple select without creating background goroutines.
type WakeUpSignal struct {
	signal chan struct{}
}

// NewWakeUpSignal creates a new WakeUpSignal.
func NewWakeUpSignal() *WakeUpSignal {
	return &WakeUpSignal{
		signal: make(chan struct{}, 1),
	}
}

// Emit signals that the tree should be ticked again.
// Non-blocking: if a signal is already pending, this is a no-op.
func (w *WakeUpSignal) Emit() {
	select {
	case w.signal <- struct{}{}:
	default:
	}
}

// WaitFor waits for a signal with a timeout.
// If timeout <= 0, this is a non-blocking poll: returns true if a signal
// was pending (and consumes it), false otherwise.
// If timeout > 0, blocks until either a signal arrives (returns true) or
// the timeout elapses (returns false).
func (w *WakeUpSignal) WaitFor(timeout time.Duration) bool {
	if timeout <= 0 {
		select {
		case <-w.signal:
			return true
		default:
			return false
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-w.signal:
		return true
	case <-timer.C:
		// Timer fired first, but a signal may have arrived just before.
		// Check one more time to avoid losing it.
		select {
		case <-w.signal:
			return true
		default:
			return false
		}
	}
}

// Reset clears any pending signal.
func (w *WakeUpSignal) Reset() {
	select {
	case <-w.signal:
	default:
	}
}
