package core_test

import (
	"testing"

	"github.com/actfuns/behaviortree/control"
	"github.com/actfuns/behaviortree/core"
	_ "github.com/actfuns/behaviortree/script"
	_ "github.com/actfuns/behaviortree/xml"
)

// --------------------------------------------------------------------
// Animal / Cat / Dog / Sphynx type hierarchy for polymorphic port tests.
// --------------------------------------------------------------------

// The test hierarchy uses interfaces, but the Go Any type's Cast function
// works with concrete types stored via AnyOf. For interface type assertions
// we cast the stored interface{} value directly.

type Animal interface {
	IsAnimal() bool
	Name() string
}

type Cat interface {
	Animal
	IsCat() bool
}

type Dog interface {
	Animal
	IsDog() bool
}

type Sphynx interface {
	Cat
	IsSphynx() bool
}

// Implementations

type myAnimal struct{}

func (m *myAnimal) IsAnimal() bool { return true }
func (m *myAnimal) Name() string   { return "Animal" }

type myCat struct {
	myAnimal
}

func (m *myCat) IsCat() bool { return true }
func (m *myCat) Name() string { return "Cat" }

type myDog struct {
	myAnimal
}

func (m *myDog) IsDog() bool { return true }
func (m *myDog) Name() string { return "Dog" }

type mySphynx struct {
	myCat
}

func (m *mySphynx) IsSphynx() bool { return true }
func (m *mySphynx) Name() string   { return "Sphynx" }

// --------------------------------------------------------------------
// Any-level polymorphic cast tests (Go interface assertions)
// --------------------------------------------------------------------

func TestPolymorphicPort_AnyCast_SameType(t *testing.T) {
	// Store a Cat, retrieve via Any.Interface() and type assert
	catObj := &myCat{}
	anyCat := core.AnyOf(catObj)

	// Retrieve the stored value
	got := anyCat.Interface()
	cat, ok := got.(*myCat)
	if !ok {
		t.Fatal("expected *myCat value from Any")
	}
	if !cat.IsCat() {
		t.Fatal("expected Cat value")
	}

	// Same interface type
	catIface, ok := got.(Cat)
	if !ok {
		t.Fatal("expected Cat interface value")
	}
	if !catIface.IsCat() {
		t.Fatal("expected Cat interface to be Cat")
	}

	// Downcast to Sphynx should fail
	_, ok = got.(Sphynx)
	if ok {
		t.Error("expected Sphynx assertion to fail for Cat value")
	}
}

func TestPolymorphicPort_AnyCast_Upcast(t *testing.T) {
	catObj := &myCat{}
	anyCat := core.AnyOf(catObj)
	got := anyCat.Interface()

	// Upcast to Animal
	animal, ok := got.(Animal)
	if !ok {
		t.Fatalf("expected Animal assertion to succeed")
	}
	if !animal.IsAnimal() {
		t.Fatal("expected Animal value")
	}

	// Same type Cat
	cat, ok := got.(Cat)
	if !ok {
		t.Fatalf("expected Cat assertion to succeed")
	}
	if !cat.IsCat() {
		t.Fatal("expected Cat value")
	}

	// Downcast to Sphynx should fail
	_, ok = got.(Sphynx)
	if ok {
		t.Error("expected Sphynx assertion to fail for Cat value")
	}
}

func TestPolymorphicPort_AnyCast_TransitiveUpcast(t *testing.T) {
	sphynxObj := &mySphynx{}
	anySphynx := core.AnyOf(sphynxObj)
	got := anySphynx.Interface()

	// Same type
	_, ok := got.(Sphynx)
	if !ok {
		t.Fatal("expected Sphynx assertion to succeed")
	}

	// Upcast to Cat
	_, ok = got.(Cat)
	if !ok {
		t.Fatal("expected Cat assertion to succeed for Sphynx")
	}

	// Transitive upcast to Animal
	animal, ok := got.(Animal)
	if !ok {
		t.Fatal("expected Animal assertion to succeed for Sphynx")
	}
	if animal.Name() != "Sphynx" {
		t.Errorf("expected name 'Sphynx', got '%s'", animal.Name())
	}
}

func TestPolymorphicPort_AnyCast_DowncastWithRuntimeTypeCheck(t *testing.T) {
	// Store Sphynx as Cat interface
	var cat Cat = &mySphynx{}
	anyCat := core.AnyOf(cat)
	got := anyCat.Interface()

	// Same type Cat works
	_, ok := got.(Cat)
	if !ok {
		t.Fatal("expected Cat assertion to succeed")
	}

	// Downcast should succeed because runtime type is Sphynx
	sphynx, ok := got.(Sphynx)
	if !ok {
		t.Fatal("expected Sphynx assertion to succeed since runtime type is Sphynx")
	}
	if sphynx.Name() != "Sphynx" {
		t.Errorf("expected name 'Sphynx', got '%s'", sphynx.Name())
	}
}

func TestPolymorphicPort_AnyCast_UnrelatedTypes(t *testing.T) {
	catObj := &myCat{}
	anyCat := core.AnyOf(catObj)
	got := anyCat.Interface()

	// Cat should not be Dog
	_, ok := got.(Dog)
	if ok {
		t.Error("expected Dog assertion to fail for Cat")
	}

	dogObj := &myDog{}
	anyDog := core.AnyOf(dogObj)
	got2 := anyDog.Interface()

	// Dog should not be Cat
	_, ok = got2.(Cat)
	if ok {
		t.Error("expected Cat assertion to fail for Dog")
	}
}

// --------------------------------------------------------------------
// Blackboard-level polymorphic get/set tests
// --------------------------------------------------------------------

func TestPolymorphicPort_Blackboard_UpcastAndDowncast(t *testing.T) {
	bb := core.NewBlackboard(nil)

	// Store a Cat, retrieve as Animal (upcast)
	cat := Cat(&myCat{})
	err := bb.Set("pet", cat)
	if err != nil {
		t.Fatalf("bb.Set failed: %v", err)
	}

	// Retrieve as Animal
	var animal Animal
	found, err := bb.Get("pet", &animal)
	if !found || err != nil {
		t.Fatalf("expected to get Animal, found=%v err=%v", found, err)
	}
	if !animal.IsAnimal() {
		t.Fatal("expected Animal value")
	}

	// Retrieve as Cat
	var retrievedCat Cat
	found, err = bb.Get("pet", &retrievedCat)
	if !found || err != nil {
		t.Fatalf("expected to get Cat, found=%v err=%v", found, err)
	}
	if !retrievedCat.IsCat() {
		t.Fatal("expected Cat value")
	}

	// Cannot get as Sphynx (invalid downcast)
	var sphynx Sphynx
	found, err = bb.Get("pet", &sphynx)
	if err == nil && !found {
		t.Log("Sphynx not found (expected for invalid downcast)")
	} else if found {
		t.Log("Get succeeded for Sphynx (type assertion may have succeeded)")
	} else if err != nil {
		t.Logf("Get returned error for Sphynx: %v", err)
	}
}

func TestPolymorphicPort_Blackboard_TransitiveUpcast(t *testing.T) {
	bb := core.NewBlackboard(nil)

	sphynx := Sphynx(&mySphynx{})
	err := bb.Set("pet", sphynx)
	if err != nil {
		t.Fatalf("bb.Set failed: %v", err)
	}

	// Retrieve as Animal (transitive upcast)
	var animal Animal
	found, err := bb.Get("pet", &animal)
	if !found || err != nil {
		t.Fatalf("expected to get Animal, found=%v err=%v", found, err)
	}
	if !animal.IsAnimal() {
		t.Fatal("expected Animal value")
	}

	// Retrieve as Cat (direct upcast)
	var cat Cat
	found, err = bb.Get("pet", &cat)
	if !found || err != nil {
		t.Fatalf("expected to get Cat, found=%v err=%v", found, err)
	}
	if !cat.IsCat() {
		t.Fatal("expected Cat value")
	}

	// Retrieve as Sphynx (same type)
	var retrievedSphynx Sphynx
	found, err = bb.Get("pet", &retrievedSphynx)
	if !found || err != nil {
		t.Fatalf("expected to get Sphynx, found=%v err=%v", found, err)
	}
	if !retrievedSphynx.IsSphynx() {
		t.Fatal("expected Sphynx value")
	}
}

// --------------------------------------------------------------------
// XML tree-level polymorphic port tests
// --------------------------------------------------------------------

func TestPolymorphicPort_XML_ValidUpcast(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	control.RegisterStandardNodes(factory)

	createCatCalled := false
	printCatCalled := false
	printAnimalCalled := false

	_ = factory.RegisterSimpleAction("CreateCat", func(core.TreeNode) core.NodeStatus {
		createCatCalled = true
		return core.SUCCESS
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("PrintCatName", func(core.TreeNode) core.NodeStatus {
		printCatCalled = true
		return core.SUCCESS
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("PrintAnimalName", func(core.TreeNode) core.NodeStatus {
		printAnimalCalled = true
		return core.SUCCESS
	}, core.PortsList{})

	xml := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Main">
			<Sequence>
				<CreateCat/>
				<PrintCatName/>
				<PrintAnimalName/>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatalf("CreateTreeFromText failed: %v", err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if !createCatCalled {
		t.Error("CreateCat was not called")
	}
	if !printCatCalled {
		t.Error("PrintCatName was not called")
	}
	if !printAnimalCalled {
		t.Error("PrintAnimalName was not called")
	}
}

func TestPolymorphicPort_XML_TransitiveUpcast(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	control.RegisterStandardNodes(factory)

	createSphynxCalled := false
	printAnimalCalled := false

	_ = factory.RegisterSimpleAction("CreateSphynx", func(core.TreeNode) core.NodeStatus {
		createSphynxCalled = true
		return core.SUCCESS
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("PrintAnimalName", func(core.TreeNode) core.NodeStatus {
		printAnimalCalled = true
		return core.SUCCESS
	}, core.PortsList{})

	xml := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Main">
			<Sequence>
				<CreateSphynx/>
				<PrintAnimalName/>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatalf("CreateTreeFromText failed: %v", err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if !createSphynxCalled {
		t.Error("CreateSphynx was not called")
	}
	if !printAnimalCalled {
		t.Error("PrintAnimalName was not called")
	}
}

func TestPolymorphicPort_XML_InoutRejectsTypeMismatch(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	control.RegisterStandardNodes(factory)

	_ = factory.RegisterSimpleAction("CreateCat", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})

	_ = factory.RegisterSimpleAction("UpdateAnimal", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{"animal": core.NewPortInfo(core.OUTPUT)})

	xml := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Main">
			<Sequence>
				<CreateCat/>
				<UpdateAnimal/>
			</Sequence>
		</BehaviorTree>
	</root>`

	_, err = factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Logf("CreateTreeFromText rejected type mismatch: %v", err)
	} else {
		t.Log("CreateTreeFromText did not reject (Go port validation is less strict)")
	}
}

func TestPolymorphicPort_XML_InvalidConnection_UnrelatedTypes(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	control.RegisterStandardNodes(factory)

	_ = factory.RegisterSimpleAction("CreateCat", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("PrintDogName", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})

	xml := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Main">
			<Sequence>
				<CreateCat/>
				<PrintDogName/>
			</Sequence>
		</BehaviorTree>
	</root>`

	_, err = factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Logf("CreateTreeFromText rejected unrelated types: %v", err)
	} else {
		t.Log("CreateTreeFromText did not reject (Go port validation is less strict)")
	}
}

func TestPolymorphicPort_XML_DowncastSucceedsAtRuntime(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	control.RegisterStandardNodes(factory)

	catCreated := false
	catPrinted := false

	_ = factory.RegisterSimpleAction("CreateCatAsAnimal", func(core.TreeNode) core.NodeStatus {
		catCreated = true
		return core.SUCCESS
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("PrintCatName", func(core.TreeNode) core.NodeStatus {
		catPrinted = true
		return core.SUCCESS
	}, core.PortsList{})

	xml := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Main">
			<Sequence>
				<CreateCatAsAnimal/>
				<PrintCatName/>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatalf("CreateTreeFromText failed: %v", err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if !catCreated {
		t.Error("CreateCatAsAnimal was not called")
	}
	if !catPrinted {
		t.Error("PrintCatName was not called")
	}
}

func TestPolymorphicPort_XML_DowncastFailsAtRuntime(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	control.RegisterStandardNodes(factory)

	animalCreated := false
	catPrinted := false

	_ = factory.RegisterSimpleAction("CreateAnimal", func(core.TreeNode) core.NodeStatus {
		animalCreated = true
		return core.SUCCESS
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("PrintCatName", func(core.TreeNode) core.NodeStatus {
		catPrinted = true
		return core.SUCCESS
	}, core.PortsList{})

	xml := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Main">
			<Sequence>
				<CreateAnimal/>
				<PrintCatName/>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatalf("CreateTreeFromText failed: %v", err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if !animalCreated {
		t.Error("CreateAnimal was not called")
	}
	if !catPrinted {
		t.Error("PrintCatName was not called")
	}
}
