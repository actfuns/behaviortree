package core

import (
	"fmt"
)

// TestTick returns SUCCESS and increments the counter.
// Equivalent of C++ TestTick function in test_helper.hpp.
func TestTick(tickCounter *int) NodeStatus {
	*tickCounter++
	return SUCCESS
}

// RegisterTestTick registers simple actions named prefix+"A", prefix+"B", etc.
// Equivalent of C++ RegisterTestTick template in test_helper.hpp.
// Each action increments its corresponding counter in tickCounters and returns SUCCESS.
func RegisterTestTick(factory BehaviorTreeFactory, namePrefix string, tickCounters []int) {
	for i := 0; i < len(tickCounters); i++ {
		tickCounters[i] = 0
		actionName := fmt.Sprintf("%s%c", namePrefix, 'A'+rune(i))
		counterPtr := &tickCounters[i]
		_ = factory.RegisterSimpleAction(actionName, func(TreeNode) NodeStatus {
			return TestTick(counterPtr)
		}, PortsList{})
	}
}
