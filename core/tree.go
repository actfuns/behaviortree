package core

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// Tree represents a behavior tree instance.
type Tree struct {
	Subtrees  []*TreeSubtree
	Manifests map[string]TreeNodeManifest

	wakeUp      *WakeUpSignal
	uidCounter  atomic.Uint64
	initialized bool
}

// TreeSubtree represents a subtree within a tree.
type TreeSubtree struct {
	Nodes        []TreeNode
	Blackboard   *Blackboard
	InstanceName string
	TreeID       string
}

// NewTree creates a new empty Tree.
func NewTree() *Tree {
	return &Tree{
		Subtrees:  nil,
		Manifests: make(map[string]TreeNodeManifest),
		wakeUp:    NewWakeUpSignal(),
	}
}

// Initialize prepares the tree for execution.
func (t *Tree) Initialize() {
	if t.initialized {
		return
	}

	for _, subtree := range t.Subtrees {
		for _, node := range subtree.Nodes {
			node.SetWakeUpInstance(t.wakeUp)
		}
	}

	t.initialized = true
}

// RootNode returns the root node of the main tree.
func (t *Tree) RootNode() TreeNode {
	if len(t.Subtrees) == 0 {
		return nil
	}
	if len(t.Subtrees[0].Nodes) == 0 {
		return nil
	}
	return t.Subtrees[0].Nodes[0]
}

// TickExactlyOnce ticks the root node exactly once.
// After the tick, if the root node's status is completed (SUCCESS or FAILURE),
// the root node's status is reset to IDLE, matching C++ behavior.
func (t *Tree) TickExactlyOnce() NodeStatus {
	root := t.RootNode()
	if root == nil {
		return FAILURE
	}
	status := root.ExecuteTick()
	if status.IsCompleted() {
		root.ResetStatus()
	}
	return status
}

// TickOnce ticks the root once, but if a wake-up signal was emitted,
// it will tick again until no wake-up is pending.
func (t *Tree) TickOnce() NodeStatus {
	return t.tickRoot(OnceUnlessWokenUp, 0)
}

// TickWhileRunning ticks the root until it returns a status other than RUNNING.
func (t *Tree) TickWhileRunning(sleepTime time.Duration) NodeStatus {
	return t.tickRoot(WhileRunning, sleepTime)
}

type tickOption int

const (
	ExactlyOnce       tickOption = 0
	OnceUnlessWokenUp tickOption = 1
	WhileRunning      tickOption = 2
)

func (t *Tree) tickRoot(opt tickOption, sleepTime time.Duration) NodeStatus {
	// Ensure initialized
	if !t.initialized {
		t.Initialize()
	}

	root := t.RootNode()
	if root == nil {
		return FAILURE
	}

	switch opt {
	case ExactlyOnce:
		return root.ExecuteTick()

	case OnceUnlessWokenUp:
		status := root.ExecuteTick()
		for t.wakeUp != nil {
			t.wakeUp.mu.Lock()
			fired := t.wakeUp.fired
			t.wakeUp.fired = false
			t.wakeUp.mu.Unlock()
			if !fired {
				break
			}
			status = root.ExecuteTick()
		}
		return status

	case WhileRunning:
		status := root.ExecuteTick()
		for status == RUNNING {
			// Wait for wake-up signal or timeout
			if t.wakeUp != nil {
				t.wakeUp.WaitFor(sleepTime)
			} else {
				time.Sleep(sleepTime)
			}
			status = root.ExecuteTick()
		}
		return status

	default:
		return root.ExecuteTick()
	}
}

// HaltTree halts all nodes in the tree.
func (t *Tree) HaltTree() {
	for _, subtree := range t.Subtrees {
		for _, node := range subtree.Nodes {
			node.Halt()
		}
	}
}

// Sleep sleeps for a duration, interruptible by wake-up signal.
func (t *Tree) Sleep(timeout time.Duration) bool {
	if t.wakeUp != nil {
		return t.wakeUp.WaitFor(timeout)
	}
	time.Sleep(timeout)
	return false
}

// EmitWakeUpSignal wakes up the tree.
func (t *Tree) EmitWakeUpSignal() {
	if t.wakeUp != nil {
		t.wakeUp.Emit()
	}
}

// WakeUpSignal returns the tree's wake-up signal.
func (t *Tree) WakeUpSignal() *WakeUpSignal {
	return t.wakeUp
}

// RootBlackboard returns the root blackboard.
func (t *Tree) RootBlackboard() *Blackboard {
	if len(t.Subtrees) == 0 {
		return nil
	}
	return t.Subtrees[0].Blackboard
}

// GetUID returns a new unique ID.
func (t *Tree) GetUID() uint16 {
	return uint16(t.uidCounter.Add(1))
}

// ApplyVisitor calls the visitor for each node in the tree recursively.
func (t *Tree) ApplyVisitor(visitor func(node TreeNode)) error {
	root := t.RootNode()
	if root != nil {
		return ApplyRecursiveVisitor(root, visitor)
	}
	return nil
}

// PrintTreeRecursively prints the tree hierarchy.
func PrintTreeRecursively(root TreeNode) {
	var recursivePrint func(indent int, node TreeNode)
	recursivePrint = func(indent int, node TreeNode) {
		if node == nil {
			fmt.Printf("%s!nullptr!\n", indentStr(indent))
			return
		}
		fmt.Printf("%s%s\n", indentStr(indent), node.Name())
		indent++

		if control, ok := interface{}(node).(interface{ Children() []TreeNode }); ok {
			for _, child := range control.Children() {
				recursivePrint(indent, child)
			}
		} else if deco, ok := interface{}(node).(interface{ Child() TreeNode }); ok {
			recursivePrint(indent, deco.Child())
		}
	}
	fmt.Println("----------------")
	recursivePrint(0, root)
	fmt.Println("----------------")
}

func indentStr(n int) string {
	return strings.Repeat("   ", n)
}

// ApplyRecursiveVisitor calls the visitor for each node in the tree recursively.
func ApplyRecursiveVisitor(node TreeNode, visitor func(TreeNode)) error {
	if node == nil {
		return fmt.Errorf("One of the children of a DecoratorNode or ControlNode is nullptr")
	}

	visitor(node)

	if control, ok := interface{}(node).(interface{ Children() []TreeNode }); ok {
		for _, child := range control.Children() {
			if err := ApplyRecursiveVisitor(child, visitor); err != nil {
				return err
			}
		}
	} else if deco, ok := interface{}(node).(interface{ Child() TreeNode }); ok {
		if deco.Child() != nil {
			if err := ApplyRecursiveVisitor(deco.Child(), visitor); err != nil {
				return err
			}
		}
	}
	return nil
}

// SerializedTreeStatus is a vector of UID/status pairs.
type SerializedTreeStatus []struct {
	UID    uint16
	Status uint8
}

// BuildSerializedStatusSnapshot creates a snapshot of all node statuses.
func BuildSerializedStatusSnapshot(root TreeNode) (SerializedTreeStatus, error) {
	var result SerializedTreeStatus
	if err := ApplyRecursiveVisitor(root, func(node TreeNode) {
		result = append(result, struct {
			UID    uint16
			Status uint8
		}{node.UID(), uint8(node.Status())})
	}); err != nil {
		return nil, err
	}
	return result, nil
}
