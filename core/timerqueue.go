package core

import (
	"container/heap"
	"sync"
	"time"
)

// TimerItem represents a scheduled timer event.
type TimerItem struct {
	ID       uint64
	Deadline time.Time
	Callback func(aborted bool)
	Index    int // index in the heap
}

// TimerQueue is a priority queue of timer events.
type TimerQueue struct {
	mu      sync.Mutex
	items   timerHeap
	counter uint64
	cond    *sync.Cond
	running bool
}

type timerHeap []*TimerItem

func (h timerHeap) Len() int           { return len(h) }
func (h timerHeap) Less(i, j int) bool { return h[i].Deadline.Before(h[j].Deadline) }
func (h timerHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

func (h *timerHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*TimerItem)
	item.Index = n
	*h = append(*h, item)
}

func (h *timerHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*h = old[0 : n-1]
	return item
}

// NewTimerQueue creates a new TimerQueue.
func NewTimerQueue() *TimerQueue {
	tq := &TimerQueue{
		running: true,
	}
	tq.cond = sync.NewCond(&tq.mu)
	heap.Init(&tq.items)
	return tq
}

// Add schedules a callback after the given duration.
// Returns the timer ID that can be used to cancel it.
func (tq *TimerQueue) Add(d time.Duration, callback func(aborted bool)) uint64 {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	tq.counter++
	id := tq.counter
	item := &TimerItem{
		ID:       id,
		Deadline: time.Now().Add(d),
		Callback: callback,
	}
	heap.Push(&tq.items, item)
	tq.cond.Signal()
	return id
}

// Cancel removes a timer by ID.
func (tq *TimerQueue) Cancel(id uint64) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	for i, item := range tq.items {
		if item.ID == id {
			heap.Remove(&tq.items, i)
			if item.Callback != nil {
				item.Callback(true) // aborted
			}
			return
		}
	}
}

// ProcessExpired fires callbacks for all timers whose deadline has passed.
// Returns the number of timers processed.
// Callbacks are invoked outside the lock to avoid deadlocks when callbacks
// acquire locks held by the caller of ProcessExpired.
func (tq *TimerQueue) ProcessExpired() int {
	tq.mu.Lock()
	now := time.Now()
	var expired []*TimerItem
	for tq.items.Len() > 0 && !tq.items[0].Deadline.After(now) {
		item := heap.Pop(&tq.items).(*TimerItem)
		expired = append(expired, item)
	}
	tq.mu.Unlock()

	for _, item := range expired {
		if item.Callback != nil {
			item.Callback(false)
		}
	}
	return len(expired)
}

// CancelAll removes all timers.
func (tq *TimerQueue) CancelAll() {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	for _, item := range tq.items {
		if item.Callback != nil {
			item.Callback(true) // aborted
		}
	}
	tq.items = nil
	heap.Init(&tq.items)
}
