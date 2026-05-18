package action

import (
	"sync"

	"github.com/actfuns/behaviortree/core"
)

// PopFromQueue pops an item from a queue on the blackboard and stores it in "popped_item".
// Returns FAILURE if the queue is empty, SUCCESS otherwise.
type PopFromQueue struct {
	core.SyncActionNode
}

func NewPopFromQueue(name string, config core.NodeConfig) *PopFromQueue {
	n := &PopFromQueue{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("PopFromQueue")
	return n
}

func (n *PopFromQueue) Tick() core.NodeStatus {
	// Read queue from blackboard as a pointer to QueueItem
	var queue *QueueItem
	if err := n.GetInput("queue", &queue); err != nil || queue == nil {
		return core.FAILURE
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	if len(queue.Items) == 0 {
		return core.FAILURE
	}

	val := queue.Items[0]
	queue.Items = queue.Items[1:]

	if err := n.SetOutput("popped_item", val); err != nil {
		return core.FAILURE
	}

	return core.SUCCESS
}

// QueueItem holds a queue with its mutex for blackboard access.
type QueueItem struct {
	mu    sync.Mutex
	Items []interface{}
}
