package core

import (
	"container/heap"
	"sync"
	"time"
)

// TimerID is a unique identifier for a timer.
type TimerID uint64

// timerItem holds a scheduled timer in the priority queue.
type timerItem struct {
	end     time.Time
	id      TimerID
	handler func(aborted bool)
	index   int
}

// timerHeap implements heap.Interface, ordered by expiration time.
type timerHeap []*timerItem

func (h timerHeap) Len() int           { return len(h) }
func (h timerHeap) Less(i, j int) bool { return h[i].end.Before(h[j].end) }
func (h timerHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i]; h[i].index = i; h[j].index = j }
func (h *timerHeap) Push(x any)        { it := x.(*timerItem); it.index = len(*h); *h = append(*h, it) }
func (h *timerHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	old[n-1] = nil
	it.index = -1
	*h = old[:n-1]
	return it
}

// TimerQueue schedules callbacks on a background goroutine.
// Handlers run on the background goroutine and must be thread-safe.
// The aborted parameter is true if Cancel was called before the timer fired.
type TimerQueue struct {
	mu      sync.Mutex
	items   timerHeap
	nextID  TimerID
	finish  bool
	notify  chan struct{}
	stopped chan struct{}
}

// NewTimerQueue creates and starts a TimerQueue.
func NewTimerQueue() *TimerQueue {
	tq := &TimerQueue{
		notify:  make(chan struct{}, 1),
		stopped: make(chan struct{}),
	}
	go tq.run()
	return tq
}

// Add schedules a handler after duration d. Returns the timer ID.
func (tq *TimerQueue) Add(d time.Duration, handler func(aborted bool)) TimerID {
	tq.mu.Lock()
	tq.nextID++
	id := tq.nextID
	heap.Push(&tq.items, &timerItem{
		end:     time.Now().Add(d),
		id:      id,
		handler: handler,
	})
	tq.mu.Unlock()
	tq.wake()
	return id
}

// Cancel removes a pending timer. Returns true if found and cancelled.
func (tq *TimerQueue) Cancel(id TimerID) bool {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	for _, it := range tq.items {
		if it.id == id && it.handler != nil {
			it.handler = nil // mark cancelled; cleaned up when it reaches top
			tq.wakeLocked()
			return true
		}
	}
	return false
}

// CancelAll cancels all pending timers.
func (tq *TimerQueue) CancelAll() {
	tq.mu.Lock()
	for _, it := range tq.items {
		it.handler = nil
	}
	tq.items = nil
	tq.wakeLocked()
	tq.mu.Unlock()
}

// Stop terminates the background goroutine.
func (tq *TimerQueue) Stop() {
	tq.mu.Lock()
	tq.finish = true
	tq.mu.Unlock()
	tq.wake()
	<-tq.stopped
}

func (tq *TimerQueue) wake() {
	select {
	case tq.notify <- struct{}{}:
	default:
	}
}

func (tq *TimerQueue) wakeLocked() {
	select {
	case tq.notify <- struct{}{}:
	default:
	}
}

func (tq *TimerQueue) run() {
	defer close(tq.stopped)

	for {
		tq.mu.Lock()

		if tq.finish {
			tq.mu.Unlock()
			return
		}

		// Pop cancelled items from the top of the heap.
		for tq.items.Len() > 0 {
			if it := tq.items[0]; it.handler == nil {
				heap.Pop(&tq.items)
			} else {
				break
			}
		}

		if tq.items.Len() == 0 {
			// No timers pending — wait for new work.
			tq.mu.Unlock()
			select {
			case <-tq.notify:
			case <-tq.stopped:
				return
			}
			continue
		}

		it := tq.items[0]
		wait := time.Until(it.end)

		if wait <= 0 {
			// Expired — pop and fire.
			heap.Pop(&tq.items)
			handler := it.handler
			tq.mu.Unlock()
			if handler != nil {
				handler(false)
			}
			continue
		}

		tq.mu.Unlock()

		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
		case <-tq.notify:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-tq.stopped:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		}
	}
}

// defaultTimerQueue is a package-level TimerQueue used when nodes are
// created without a Tree (e.g., in tests). Tree.Initialize() replaces
// this with a tree-specific instance.
var defaultTimerQueue = NewTimerQueue()
