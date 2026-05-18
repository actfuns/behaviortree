package core

import (
	"sync"
	"time"
)

// WakeUpSignal provides a way to signal a waiting tree to tick again.
// It mirrors C++ BT::WakeUpSignal.
type WakeUpSignal struct {
	mu       sync.Mutex
	fired    bool
	signalCh chan struct{}
}

// NewWakeUpSignal creates a new WakeUpSignal.
func NewWakeUpSignal() *WakeUpSignal {
	return &WakeUpSignal{
		signalCh: make(chan struct{}, 1),
	}
}

// Emit signals that the tree should be ticked again.
func (w *WakeUpSignal) Emit() {
	w.mu.Lock()
	w.fired = true
	w.mu.Unlock()

	select {
	case w.signalCh <- struct{}{}:
	default:
	}
}

// WaitFor waits for a signal with a timeout.
// Returns true if the signal was received, false on timeout.
func (w *WakeUpSignal) WaitFor(timeout time.Duration) bool {
	w.mu.Lock()
	if w.fired {
		w.fired = false
		w.mu.Unlock()
		return true
	}
	w.mu.Unlock()

	select {
	case <-w.signalCh:
		w.mu.Lock()
		fired := w.fired
		w.fired = false
		w.mu.Unlock()
		return fired
	case <-time.After(timeout):
		return false
	}
}

// Reset clears the fired flag.
func (w *WakeUpSignal) Reset() {
	w.mu.Lock()
	w.fired = false
	w.mu.Unlock()
}
