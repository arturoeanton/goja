package goja

import (
	"fmt"
	"testing"
)

func TestDebuggerVariables(t *testing.T) {
	const script = `
	function testFunc(x, y) {
		var local1 = x + y;
		var local2 = "hello";
		var obj = {a: 1, b: "test"};
		var arr = [1, 2, 3];
		debugger; // Breakpoint here
		return local1;
	}
	
	var global1 = 42;
	var global2 = "global string";
	testFunc(10, 20);
	`

	r := New()
	debugger := r.EnableDebugger()
	
	var capturedState *DebuggerState
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		capturedState = state
		t.Logf("Debugger paused at PC: %d", state.PC)
		t.Logf("CallStack length: %d", len(state.CallStack))
		t.Logf("DebugStack length: %d", len(state.DebugStack))
		return DebugContinue
	})

	_, err := r.RunString(script)
	if err != nil {
		t.Fatal(err)
	}

	if capturedState == nil {
		t.Fatal("Debugger was not triggered")
	}

	// Test local scope
	if len(capturedState.DebugStack) == 0 {
		t.Fatal("Debug stack is empty")
	}

	frame := capturedState.DebugStack[0]
	if len(frame.Scopes) < 2 {
		t.Fatalf("Expected at least 2 scopes, got %d", len(frame.Scopes))
	}

	// Find local scope
	var localScope *Scope
	for _, scope := range frame.Scopes {
		if scope.Name == "Local" {
			localScope = &scope
			break
		}
	}

	if localScope == nil {
		t.Fatal("Local scope not found")
	}

	// Get local variables
	t.Logf("Getting variables for scope ref: %d", localScope.VariablesRef)
	localVars := debugger.GetVariables(localScope.VariablesRef)
	if localVars == nil {
		t.Fatal("Failed to get local variables")
	}
	t.Logf("Found %d local variables", len(localVars))

	// Check for expected variables
	varMap := make(map[string]*Variable)
	for i := range localVars {
		varMap[localVars[i].Name] = &localVars[i]
	}

	// Check x parameter
	if x, ok := varMap["x"]; !ok {
		t.Error("Variable 'x' not found in local scope")
	} else {
		if x.Type != "number" {
			t.Errorf("Expected 'x' to be number, got %s", x.Type)
		}
		if val, ok := x.Value.(valueInt); !ok || int64(val) != 10 {
			t.Errorf("Expected 'x' to be 10, got %v", x.Value)
		}
	}

	// Check y parameter
	if y, ok := varMap["y"]; !ok {
		t.Error("Variable 'y' not found in local scope")
	} else {
		if y.Type != "number" {
			t.Errorf("Expected 'y' to be number, got %s", y.Type)
		}
		if val, ok := y.Value.(valueInt); !ok || int64(val) != 20 {
			t.Errorf("Expected 'y' to be 20, got %v", y.Value)
		}
	}

	// Check local1
	if local1, ok := varMap["local1"]; !ok {
		t.Error("Variable 'local1' not found in local scope")
	} else {
		if local1.Type != "number" {
			t.Errorf("Expected 'local1' to be number, got %s", local1.Type)
		}
		if val, ok := local1.Value.(valueInt); !ok || int64(val) != 30 {
			t.Errorf("Expected 'local1' to be 30, got %v", local1.Value)
		}
	}

	// Check local2
	if local2, ok := varMap["local2"]; !ok {
		t.Error("Variable 'local2' not found in local scope")
	} else {
		if local2.Type != "string" {
			t.Errorf("Expected 'local2' to be string, got %s", local2.Type)
		}
	}

	// Check obj
	if obj, ok := varMap["obj"]; !ok {
		t.Error("Variable 'obj' not found in local scope")
	} else {
		if obj.Type != "object" {
			t.Errorf("Expected 'obj' to be object, got %s", obj.Type)
		}
		if obj.Ref == 0 {
			t.Error("Expected 'obj' to have a reference ID")
		}
		
		// Get object properties
		objProps := debugger.GetVariables(obj.Ref)
		if objProps == nil {
			t.Error("Failed to get object properties")
		}
		
		propMap := make(map[string]*Variable)
		for i := range objProps {
			propMap[objProps[i].Name] = &objProps[i]
		}
		
		if a, ok := propMap["a"]; !ok {
			t.Error("Property 'a' not found in object")
		} else if val, ok := a.Value.(valueInt); !ok || int64(val) != 1 {
			t.Errorf("Expected obj.a to be 1, got %v", a.Value)
		}
	}

	// Check arr
	if arr, ok := varMap["arr"]; !ok {
		t.Error("Variable 'arr' not found in local scope")
	} else {
		if arr.Type != "array" {
			t.Errorf("Expected 'arr' to be array, got %s", arr.Type)
		}
		if arr.Ref == 0 {
			t.Error("Expected 'arr' to have a reference ID")
		}
	}
}

func TestDebuggerGlobalVariables(t *testing.T) {
	const script = `
	var global1 = 42;
	var global2 = "global string";
	var globalObj = {prop: "value"};
	
	function test() {
		debugger;
	}
	
	test();
	`

	r := New()
	debugger := r.EnableDebugger()
	
	var capturedState *DebuggerState
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		capturedState = state
		return DebugContinue
	})

	_, err := r.RunString(script)
	if err != nil {
		t.Fatal(err)
	}

	if capturedState == nil {
		t.Fatal("Debugger was not triggered")
	}

	// Find global scope
	frame := capturedState.DebugStack[0]
	var globalScope *Scope
	for _, scope := range frame.Scopes {
		if scope.Name == "Global" {
			globalScope = &scope
			break
		}
	}

	if globalScope == nil {
		t.Fatal("Global scope not found")
	}

	// Get global variables
	globalVars := debugger.GetVariables(globalScope.VariablesRef)
	if globalVars == nil {
		t.Fatal("Failed to get global variables")
	}

	// Check for our global variables
	varMap := make(map[string]*Variable)
	for i := range globalVars {
		varMap[globalVars[i].Name] = &globalVars[i]
	}

	// Check global1
	if global1, ok := varMap["global1"]; !ok {
		t.Error("Variable 'global1' not found in global scope")
	} else {
		if global1.Type != "number" {
			t.Errorf("Expected 'global1' to be number, got %s", global1.Type)
		}
		if val, ok := global1.Value.(valueInt); !ok || int64(val) != 42 {
			t.Errorf("Expected 'global1' to be 42, got %v", global1.Value)
		}
	}

	// Check global2
	if global2, ok := varMap["global2"]; !ok {
		t.Error("Variable 'global2' not found in global scope")
	} else {
		if global2.Type != "string" {
			t.Errorf("Expected 'global2' to be string, got %s", global2.Type)
		}
	}

	// Check globalObj
	if globalObj, ok := varMap["globalObj"]; !ok {
		t.Error("Variable 'globalObj' not found in global scope")
	} else {
		if globalObj.Type != "object" {
			t.Errorf("Expected 'globalObj' to be object, got %s", globalObj.Type)
		}
	}
}

func TestDebuggerEvaluate(t *testing.T) {
	const script = `
	var x = 10;
	var y = 20;
	
	function test() {
		var local = 30;
		debugger;
		return local;
	}
	
	test();
	`

	r := New()
	debugger := r.EnableDebugger()
	
	var evalResult Value
	var evalErr error
	
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		// Evaluate expressions
		evalResult, evalErr = debugger.Evaluate("x + y", 0)
		return DebugContinue
	})

	_, err := r.RunString(script)
	if err != nil {
		t.Fatal(err)
	}

	if evalErr != nil {
		t.Fatalf("Evaluation failed: %v", evalErr)
	}

	if val, ok := evalResult.(valueInt); !ok || int64(val) != 30 {
		t.Errorf("Expected evaluation result to be 30, got %v", evalResult)
	}
}

func TestDebuggerScopes(t *testing.T) {
	const script = `
	function outer() {
		var outerVar = "outer";
		
		function inner() {
			var innerVar = "inner";
			debugger;
		}
		
		inner();
	}
	
	outer();
	`

	r := New()
	debugger := r.EnableDebugger()
	
	var capturedState *DebuggerState
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		capturedState = state
		return DebugContinue
	})

	_, err := r.RunString(script)
	if err != nil {
		t.Fatal(err)
	}

	if capturedState == nil {
		t.Fatal("Debugger was not triggered")
	}

	// Check that we have multiple frames
	if len(capturedState.DebugStack) < 2 {
		t.Fatalf("Expected at least 2 frames in call stack, got %d", len(capturedState.DebugStack))
	}

	// Check scopes for each frame
	for i, frame := range capturedState.DebugStack {
		t.Logf("Frame %d: %v", i, frame.StackFrame)
		
		for _, scope := range frame.Scopes {
			vars := debugger.GetVariables(scope.VariablesRef)
			t.Logf("  Scope %s: %d variables", scope.Name, len(vars))
			
			for _, v := range vars {
				t.Logf("    %s (%s) = %v", v.Name, v.Type, v.Value)
			}
		}
	}
}

func ExampleDebugger_GetVariables() {
	script := `
	function example(param1, param2) {
		var local = param1 + param2;
		var obj = {x: 10, y: 20};
		debugger;
		return local;
	}
	
	example(5, 10);
	`

	r := New()
	debugger := r.EnableDebugger()
	
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		// Get the first frame's scopes
		if len(state.DebugStack) > 0 {
			frame := state.DebugStack[0]
			
			// Find local scope
			for _, scope := range frame.Scopes {
				if scope.Name == "Local" {
					// Get variables
					vars := debugger.GetVariables(scope.VariablesRef)
					
					fmt.Println("Local variables:")
					for _, v := range vars {
						fmt.Printf("  %s (%s)\n", v.Name, v.Type)
					}
				}
			}
		}
		
		return DebugContinue
	})

	r.RunString(script)
	// Output:
	// Local variables:
	//   param1 (number)
	//   param2 (number)
	//   local (number)
	//   obj (object)
}