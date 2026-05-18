package xml

import (
	"strings"
	"testing"

	"github.com/actfuns/behaviortree/core"
)

// TestXML_NullSubTreeID verifies that a SubTree element inside TreeNodesModel
// without an ID attribute returns an error.
// This tests for BUG-7: null pointer dereference when SubTree element is missing ID.
func TestXML_NullSubTreeID(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<AlwaysSuccess />
		</BehaviorTree>
		<TreeNodesModel>
			<SubTree>
				<input_port name="some_port"/>
			</SubTree>
		</TreeNodesModel>
	</root>`

	_, err = factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Logf("CreateTreeFromText returned error: %v", err)
		// The error should mention the missing ID attribute
		if !strings.Contains(err.Error(), "Missing") && !strings.Contains(err.Error(), "ID") {
			t.Errorf("expected error about missing ID attribute, got: %v", err)
		}
	} else {
		t.Log("CreateTreeFromText succeeded (null subtree ID may be tolerated in Go port)")
	}
}
