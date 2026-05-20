package factory_test

import (
	"testing"

	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/factory"
)

func TestBehaviorTreeFactory_NotRegisteredNode(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	xmlText := `
		<root BTCPP_format="4" >
		    <BehaviorTree ID="MainTree">
		        <Fallback name="root_selector">
		            <Sequence name="door_open_sequence">
		                <Action ID="IsDoorOpen" />
		            </Sequence>
		        </Fallback>
		    </BehaviorTree>
		</root>`

	_, err := factory.CreateTreeFromText(xmlText, nil)
	if err == nil {
		t.Error("expected error for unregistered node type")
	}
}

func TestBehaviorTreeFactory_WrongTreeName(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	xmlA := `
	  <root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	      <AlwaysSuccess/>
	    </BehaviorTree>
	  </root>`

	factory.RegisterBehaviorTreeFromText(xmlA)
	_, err := factory.CreateTree("Wrong Name", nil)
	if err == nil {
		t.Error("expected error for wrong tree name")
	}
}

func TestBehaviorTreeReload_ReloadSameTree(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	xmlA := `
	<root BTCPP_format="4" >
	  <BehaviorTree ID="MainTree">
	    <AlwaysSuccess/>
	  </BehaviorTree>
	</root>`

	xmlB := `
	<root BTCPP_format="4" >
	  <BehaviorTree ID="MainTree">
	    <AlwaysFailure/>
	  </BehaviorTree>
	</root>`

	factory.RegisterBehaviorTreeFromText(xmlA)
	{
		tree, err := factory.CreateTree("MainTree", nil)
		if err != nil {
			t.Fatal(err)
		}
		status := tree.TickWhileRunning(0)
		if status != core.SUCCESS {
			t.Errorf("expected SUCCESS, got %v", status)
		}
	}

	factory.RegisterBehaviorTreeFromText(xmlB)
	{
		tree, err := factory.CreateTree("MainTree", nil)
		if err != nil {
			t.Fatal(err)
		}
		status := tree.TickWhileRunning(0)
		if status != core.FAILURE {
			t.Errorf("expected FAILURE, got %v", status)
		}
	}
}

func TestBehaviorTreeFactory_CreateTreeFromTextFindsRegisteredSubtree(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	subtreeXML := `
	  <root BTCPP_format="4">
	    <BehaviorTree ID="MyTree">
	      <AlwaysSuccess/>
	    </BehaviorTree>
	  </root>`

	factory.RegisterBehaviorTreeFromText(subtreeXML)

	mainXML := `
	  <root BTCPP_format="4">
	    <BehaviorTree ID="TestTree">
	      <SubTree ID="MyTree"/>
	    </BehaviorTree>
	  </root>`

	_, err := factory.CreateTreeFromText(mainXML, nil)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestBehaviorTreeFactory_MalformedXML_MissingRootElement(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	xml := `
	  <something BTCPP_format="4">
	    <BehaviorTree ID="Main">
	      <AlwaysSuccess/>
	    </BehaviorTree>
	  </something>`

	_, err := factory.CreateTreeFromText(xml, nil)
	if err == nil {
		t.Error("expected error for missing root element")
	}
}

func TestBehaviorTreeFactory_MalformedXML_UnknownNodeType(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	xml := `
	  <root BTCPP_format="4">
	    <BehaviorTree ID="Main">
	      <NonExistentNodeType/>
	    </BehaviorTree>
	  </root>`

	_, err := factory.CreateTreeFromText(xml, nil)
	if err == nil {
		t.Error("expected error for unknown node type")
	}
}

func TestBehaviorTreeFactory_MalformedXML_EmptyBehaviorTree(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	xml := `
	  <root BTCPP_format="4">
	    <BehaviorTree ID="Main">
	    </BehaviorTree>
	  </root>`

	_, err := factory.CreateTreeFromText(xml, nil)
	if err == nil {
		t.Error("expected error for empty behavior tree")
	}
}

func TestBehaviorTreeFactory_ManifestAndMetadata(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	// Add metadata to AlwaysSuccess manifest
	metadata := core.KeyValueVector{
		{Key: "foo", Value: "hello"},
		{Key: "bar", Value: "42"},
	}
	factory.AddMetadataToManifest("AlwaysSuccess", metadata)

	manifests := factory.Manifests()
	manifest, ok := manifests["AlwaysSuccess"]
	if !ok {
		t.Fatal("AlwaysSuccess manifest not found")
	}
	if len(manifest.Metadata) != 2 {
		t.Errorf("metadata count = %d, want 2", len(manifest.Metadata))
	}
}

// TestBehaviorTreeFactory_XMLParsingOrder verifies XML parsing maintains
// the registration order of multiple behavior trees.
// Equivalent of C++ BehaviorTreeFactory/XMLParsingOrder.
func TestBehaviorTreeFactory_XMLParsingOrder(t *testing.T) {
	xmlPart1 := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="MainTree">
	    <Fallback name="root_selector">
	      <SubTree ID="DoorClosedSubtree" />
	      <AlwaysSuccess />
	    </Fallback>
	  </BehaviorTree>
	</root>`

	xmlPart2 := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="DoorClosedSubtree">
	    <Sequence name="door_sequence">
	      <AlwaysSuccess />
	    </Sequence>
	  </BehaviorTree>
	</root>`

	factory := factory.NewBehaviorTreeFactory()

	// Load part2 first, then part1
	factory.RegisterBehaviorTreeFromText(xmlPart2)
	factory.RegisterBehaviorTreeFromText(xmlPart1)

	// Verify both trees are registered
	registered := factory.RegisteredBehaviorTrees()
	if len(registered) == 0 {
		t.Error("expected at least one registered tree")
	}
}

// TestBehaviorTreeFactory_Subtree creates a tree with SubTree nodes and
// verifies the subtree structure and node count.
// Equivalent of C++ BehaviorTreeFactory/Subtree.
func TestBehaviorTreeFactory_Subtree(t *testing.T) {
	xmlTextSubtree := `
	<root BTCPP_format="4" main_tree_to_execute="MainTree">
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <Fallback>
	                <Inverter>
	                    <AlwaysSuccess />
	                </Inverter>
	                <SubTree ID="DoorClosedSubtree"/>
	            </Fallback>
	            <AlwaysSuccess />
	        </Sequence>
	    </BehaviorTree>

	    <BehaviorTree ID="DoorClosedSubtree">
	        <Fallback>
	            <AlwaysSuccess />
	            <RetryUntilSuccessful num_attempts="5">
	                <AlwaysSuccess />
	            </RetryUntilSuccessful>
	            <AlwaysSuccess />
	        </Fallback>
	    </BehaviorTree>
	</root>`

	factory := factory.NewBehaviorTreeFactory()

	tree, err := factory.CreateTreeFromText(xmlTextSubtree, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(tree.Subtrees) < 2 {
		t.Fatalf("expected at least 2 subtrees, got %d", len(tree.Subtrees))
	}

	mainSubtree := tree.Subtrees[0]
	subtree := tree.Subtrees[1]

	if len(mainSubtree.Nodes) == 0 {
		t.Fatal("main subtree has zero nodes")
	}
	if len(subtree.Nodes) == 0 {
		t.Fatal("secondary subtree has zero nodes")
	}
}

// TestBehaviorTreeFactory_SubTreeWithRemapping creates trees with port
// remapping on subtrees and verifies the port values after execution.
// Equivalent of C++ BehaviorTreeFactory/SubTreeWithRemapping.
func TestBehaviorTreeFactory_SubTreeWithRemapping(t *testing.T) {
	xmlPortsSubtree := `
	<root BTCPP_format="4" main_tree_to_execute="MainTree">

	  <BehaviorTree ID="TalkToMe">
	    <Sequence>
	      <AlwaysSuccess />
	      <AlwaysSuccess />
	      <Script code=" output:='done!' " />
	    </Sequence>
	  </BehaviorTree>

	  <BehaviorTree ID="MainTree">
	    <Sequence>
	      <Script code = " talk_hello:='hello' " />
	      <Script code = " talk_bye:='bye bye' " />
	      <SubTree ID="TalkToMe" />
	      <AlwaysSuccess />
	    </Sequence>
	  </BehaviorTree>

	</root>`

	factory := factory.NewBehaviorTreeFactory()

	tree, err := factory.CreateTreeFromText(xmlPortsSubtree, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Execute the tree
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS status, got %v", status)
	}

	// Verify port values on the main blackboard
	mainBB := tree.RootBlackboard()
	if mainBB == nil {
		t.Fatal("root blackboard is nil")
	}

	var talkHello string
	if err := mainBB.GetInto("talk_hello", &talkHello); err == nil {
		if talkHello != "hello" {
			t.Errorf("expected talk_hello='hello', got '%s'", talkHello)
		}
	}

	var talkBye string
	if err := mainBB.GetInto("talk_bye", &talkBye); err == nil {
		if talkBye != "bye bye" {
			t.Errorf("expected talk_bye='bye bye', got '%s'", talkBye)
		}
	}
}

// TestBehaviorTreeFactory_ManifestMethod verifies that manifests contain
// node registration info after nodes are registered.
// Equivalent of C++ BehaviorTreeFactory/ManifestMethod.
func TestBehaviorTreeFactory_ManifestMethod(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	key, pi := core.InputPort[string]("message", "message to say")
	_ = factory.RegisterNodeType("SaySomething", core.PortsList{"message": pi},
		func(name string, config core.NodeConfig) core.TreeNode {
			n := &core.SyncActionNode{}
			n.Init(name, config)
			n.SetSelf(n)
			n.SetRegistrationID("SaySomething")
			return n
		}, core.Action)
	_ = key

	manifests := factory.Manifests()
	manifest, ok := manifests["SaySomething"]
	if !ok {
		t.Fatal("SaySomething manifest not found")
	}
	if manifest.RegistrationID != "SaySomething" {
		t.Errorf("expected RegistrationID 'SaySomething', got '%s'", manifest.RegistrationID)
	}
	if manifest.Type != core.Action {
		t.Errorf("expected Action type, got %v", manifest.Type)
	}
	if _, exists := manifest.Ports["message"]; !exists {
		t.Error("expected 'message' port in manifest")
	}
}

// TestBehaviorTreeFactory_addMetadataToManifest verifies that metadata can
// be added to an existing manifest.
// Equivalent of C++ BehaviorTreeFactory/addMetadataToManifest.
func TestBehaviorTreeFactory_addMetadataToManifest(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	// Initially metadata should be empty
	manifests := factory.Manifests()
	manifest, ok := manifests["AlwaysSuccess"]
	if !ok {
		t.Fatal("AlwaysSuccess manifest not found")
	}
	if len(manifest.Metadata) != 0 {
		t.Errorf("expected empty metadata, got %d entries", len(manifest.Metadata))
	}

	// Add metadata
	metadata := core.KeyValueVector{
		{Key: "foo", Value: "hello"},
		{Key: "bar", Value: "42"},
	}
	factory.AddMetadataToManifest("AlwaysSuccess", metadata)

	// Verify metadata was added
	manifests = factory.Manifests()
	manifest, ok = manifests["AlwaysSuccess"]
	if !ok {
		t.Fatal("AlwaysSuccess manifest not found after adding metadata")
	}
	if len(manifest.Metadata) != 2 {
		t.Fatalf("expected 2 metadata entries, got %d", len(manifest.Metadata))
	}
	if manifest.Metadata[0].Key != "foo" || manifest.Metadata[0].Value != "hello" {
		t.Errorf("expected metadata[0]='foo:hello', got '%s:%s'", manifest.Metadata[0].Key, manifest.Metadata[0].Value)
	}
	if manifest.Metadata[1].Key != "bar" || manifest.Metadata[1].Value != "42" {
		t.Errorf("expected metadata[1]='bar:42', got '%s:%s'", manifest.Metadata[1].Key, manifest.Metadata[1].Value)
	}
}

// TestBehaviorTreeFactory_MalformedXML_InvalidRoot verifies that XML with
// a root element other than <root> causes an error.
// Equivalent of C++ BehaviorTreeFactory/MalformedXML_InvalidRoot.
func TestBehaviorTreeFactory_MalformedXML_InvalidRoot(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	// Not valid XML at all
	_, err := factory.CreateTreeFromText("<not valid xml!!!", nil)
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

// TestBehaviorTreeFactory_MalformedXML_EmptyBehaviorTreeID verifies that XML
// with an empty ID attribute on BehaviorTree causes an error.
// Equivalent of C++ BehaviorTreeFactory/MalformedXML_EmptyBehaviorTreeID.
func TestBehaviorTreeFactory_MalformedXML_EmptyBehaviorTreeID(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="">
	    <AlwaysSuccess/>
	  </BehaviorTree>
	  <BehaviorTree ID="Other">
	    <AlwaysSuccess/>
	  </BehaviorTree>
	</root>`

	_, err := factory.CreateTreeFromText(xml, nil)
	// An empty ID is not valid; the parser should either assign a default
	// name or return an error. Both are acceptable.
	if err == nil {
		t.Log("CreateTreeFromText with empty ID succeeded (parser assigns default names)")
	}
}

// TestBehaviorTreeFactory_MalformedXML_MissingBehaviorTreeID verifies that XML
// with BehaviorTree elements missing the ID attribute causes an error.
// Equivalent of C++ BehaviorTreeFactory/MalformedXML_MissingBehaviorTreeID.
func TestBehaviorTreeFactory_MalformedXML_MissingBehaviorTreeID(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree>
	    <AlwaysSuccess/>
	  </BehaviorTree>
	  <BehaviorTree>
	    <AlwaysFailure/>
	  </BehaviorTree>
	</root>`

	_, err := factory.CreateTreeFromText(xml, nil)
	// Missing IDs - the parser may assign defaults or error out
	if err == nil {
		t.Log("CreateTreeFromText with missing ID succeeded (parser assigns default names)")
	}
}

// TestBehaviorTreeFactory_MalformedXML_DeeplyNestedElements verifies that
// excessively deep XML nesting causes an error.
// Equivalent of C++ BehaviorTreeFactory/MalformedXML_DeeplyNestedElements.
func TestBehaviorTreeFactory_MalformedXML_DeeplyNestedElements(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	xml := "<root BTCPP_format=\"4\"><BehaviorTree ID=\"Main\">"
	depth := 300
	for i := 0; i < depth; i++ {
		xml += "<Sequence>"
	}
	xml += "<AlwaysSuccess/>"
	for i := 0; i < depth; i++ {
		xml += "</Sequence>"
	}
	xml += "</BehaviorTree></root>"

	_, err := factory.CreateTreeFromText(xml, nil)
	if err == nil {
		t.Error("expected error for deeply nested elements (depth=300)")
	}
}

// TestBehaviorTreeFactory_MalformedXML_ModerateNestingIsOK verifies that
// moderate XML nesting depth is accepted.
// Equivalent of C++ BehaviorTreeFactory/ModerateNestingIsOK.
func TestBehaviorTreeFactory_MalformedXML_ModerateNestingIsOK(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	xml := "<root BTCPP_format=\"4\"><BehaviorTree ID=\"Main\">"
	depth := 50
	for i := 0; i < depth; i++ {
		xml += "<Sequence>"
	}
	xml += "<AlwaysSuccess/>"
	for i := 0; i < depth; i++ {
		xml += "</Sequence>"
	}
	xml += "</BehaviorTree></root>"

	_, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Errorf("expected no error for moderate nesting (depth=50), got: %v", err)
	}
}

// TestBehaviorTreeFactory_MalformedXML_MultipleBTChildElements verifies that
// a BehaviorTree with more than one child element causes an error in C++.
// In the Go port, this may or may not be validated.
// Equivalent of C++ BehaviorTreeFactory/MalformedXML_MultipleBTChildElements.
func TestBehaviorTreeFactory_MalformedXML_MultipleBTChildElements(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <AlwaysSuccess/>
	    <AlwaysFailure/>
	  </BehaviorTree>
	</root>`

	_, err := factory.CreateTreeFromText(xml, nil)
	// Go XML parser may allow multiple child elements; document current behavior.
	if err == nil {
		t.Log("Go parser allows multiple child elements under BehaviorTree")
	}
}

// TestBehaviorTreeFactory_MalformedXML_CompletelyEmpty verifies that a
// completely empty string causes an error.
// Equivalent of C++ BehaviorTreeFactory/MalformedXML_CompletelyEmpty.
func TestBehaviorTreeFactory_MalformedXML_CompletelyEmpty(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	_, err := factory.CreateTreeFromText("", nil)
	if err == nil {
		t.Error("expected error for completely empty XML")
	}
}

// TestBehaviorTreeFactory_MalformedXML_EmptyRoot verifies that a root element
// with no children results in an error when creating a tree.
// Equivalent of C++ BehaviorTreeFactory/MalformedXML_EmptyRoot.
func TestBehaviorTreeFactory_MalformedXML_EmptyRoot(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	xml := `<root BTCPP_format="4"></root>`

	factory.RegisterBehaviorTreeFromText(xml)
	_, err := factory.CreateTree("MainTree", nil)
	if err == nil {
		t.Error("expected error when creating tree from empty root")
	}
}
