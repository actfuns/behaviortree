package decorator

import (
	"sync"

	"github.com/actfuns/behaviortree/core"
)

// QueueItem holds a queue with its mutex for blackboard access.
// This is the Go equivalent of the C++ ProtectedQueue<T> template.
type QueueItem struct {
	mu    sync.Mutex
	Items []interface{}
}

// ConsumeQueueNode executes the child node as long as the queue is not empty.
// At each iteration, an item is popped from the "queue" and inserted in "popped_item".
// An empty queue will return SUCCESS.
//
// Deprecated: You are encouraged to use the LoopNode instead.
type ConsumeQueueNode struct {
	core.DecoratorNode
	runningChild bool
}

func NewConsumeQueueNode(name string, config core.NodeConfig) *ConsumeQueueNode {
	n := &ConsumeQueueNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("ConsumeQueue")
	return n
}

func (n *ConsumeQueueNode) Tick() core.NodeStatus {
	// By default, return SUCCESS, even if queue is empty
	statusToBeReturned := core.SUCCESS

	if n.runningChild {
		childState := n.Child().ExecuteTick()
		n.runningChild = (childState == core.RUNNING)
		if n.runningChild {
			return core.RUNNING
		}
		n.ResetChild()
		statusToBeReturned = childState
	}

	// Read queue from blackboard
	var queue *QueueItem
	if err := n.GetInput("queue", &queue); err != nil || queue == nil {
		return statusToBeReturned
	}

	queue.mu.Lock()
	items := queue.Items

	for len(items) > 0 {
		n.SetStatus(core.RUNNING)

		val := items[0]
		items = items[1:]
		queue.Items = items

		if err := n.SetOutput("popped_item", val); err != nil {
			queue.mu.Unlock()
			return core.FAILURE
		}

		queue.mu.Unlock()
		childState := n.Child().ExecuteTick()
		queue.mu.Lock()

		n.runningChild = (childState == core.RUNNING)
		if n.runningChild {
			queue.mu.Unlock()
			return core.RUNNING
		}
		n.ResetChild()
		if childState == core.FAILURE {
			queue.mu.Unlock()
			return core.FAILURE
		}
		statusToBeReturned = childState
	}
	queue.mu.Unlock()

	return statusToBeReturned
}
