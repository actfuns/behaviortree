package decorator

import (
	"github.com/actfuns/behaviortree/core"
	"log/slog"
)

// UpdatedDecorator checks the Timestamp in a blackboard entry to determine
// if the value was updated since the last time (true the first time).
// If it is, the child will be executed, otherwise [if_not_updated] value is returned.
type UpdatedDecorator struct {
	core.DecoratorNode
	sequenceID          uint64
	entryKey            string
	stillExecutingChild bool
	ifNotUpdated        core.NodeStatus
}

func NewUpdatedDecorator(name string, config core.NodeConfig, ifNotUpdated core.NodeStatus) *UpdatedDecorator {
	n := &UpdatedDecorator{
		ifNotUpdated: ifNotUpdated,
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("UpdatedDecorator")

	// Extract the entry key from config's input ports
	entryStr, hasEntry := config.InputPorts["entry"]
	if !hasEntry || entryStr == "" {
		slog.Error("Missing port [entry]", "node", name)
		return nil
	}
	if ok, strippedKey := core.IsBlackboardPointer(entryStr); ok {
		n.entryKey = strippedKey
	} else {
		n.entryKey = entryStr
	}
	return n
}

func (n *UpdatedDecorator) Tick() core.NodeStatus {
	// Continue executing an asynchronous child
	if n.stillExecutingChild {
		status := n.Child().ExecuteTick()
		n.stillExecutingChild = (status == core.RUNNING)
		return status
	}

	if n.Config().Blackboard != nil {
		entry := n.Config().Blackboard.GetEntry(n.entryKey)
		if entry != nil {
			currentID := entry.SequenceID()
			prevID := n.sequenceID
			n.sequenceID = currentID

			if prevID == currentID {
				return n.ifNotUpdated
			}
		} else {
			return n.ifNotUpdated
		}
	} else {
		return n.ifNotUpdated
	}

	status := n.Child().ExecuteTick()
	n.stillExecutingChild = (status == core.RUNNING)
	return status
}

func (n *UpdatedDecorator) Halt() {
	n.stillExecutingChild = false
	n.DecoratorNode.Halt()
}
