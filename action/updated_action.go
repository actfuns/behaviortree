package action

import (
	"github.com/actfuns/behaviortree/core"
	"log/slog"
)

// EntryUpdatedAction checks the Timestamp in a blackboard entry
// to determine if the value was updated since the last time.
// SUCCESS if it was updated, FAILURE if not updated or doesn't exist.
type EntryUpdatedAction struct {
	core.SyncActionNode
	sequenceID uint64
	entryKey   string
}

func NewEntryUpdatedAction(name string, config core.NodeConfig) *EntryUpdatedAction {
	n := &EntryUpdatedAction{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("UpdatedAction")

	// Extract the entry key from config's input ports
	entryStr, hasEntry := config.InputPorts["entry"]
	if !hasEntry || entryStr == "" {
		slog.Error("Missing port [entry]", "node", name)
		return n
	}
	if ok, strippedKey := core.IsBlackboardPointer(entryStr); ok {
		n.entryKey = strippedKey
	} else {
		n.entryKey = entryStr
	}

	return n
}

func (n *EntryUpdatedAction) Tick() core.NodeStatus {
	if n.Config().Blackboard != nil {
		entry := n.Config().Blackboard.GetEntry(n.entryKey)
		if entry != nil {
			entry.Lock()
			currentID := entry.SequenceID()
			previousID := n.sequenceID
			n.sequenceID = currentID
			entry.Unlock()

			if previousID != currentID {
				return core.SUCCESS
			}
		}
	}
	return core.FAILURE
}
