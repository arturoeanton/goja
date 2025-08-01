package goja

import (
	"testing"
)

func TestStackFrameGetLocalVariables(t *testing.T) {
	const script = `
	function testFunc(x, y) {
		var local1 = x + y;
		var local2 = "hello";
		var obj = {a: 1, b: "test"};
		debugger;
		return local1;
	}
	
	testFunc(10, 20);
	`

	r := New()
	debugger := r.EnableDebugger()
	
	var capturedStack []StackFrame
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		capturedStack = state.CallStack
		return DebugContinue
	})

	_, err := r.RunString(script)
	if err != nil {
		t.Fatal(err)
	}

	if len(capturedStack) == 0 {
		t.Fatal("No stack frames captured")
	}

	// Get the top frame (inside testFunc)
	frame := &capturedStack[0]
	
	// Debug: print frame info
	t.Logf("Frame: %s at %s:%d", frame.FuncName(), frame.SrcName(), frame.Position().Line)
	t.Logf("Frame has ctx: %v", frame.ctx != nil)
	if frame.ctx != nil {
		t.Logf("Frame ctx has stash: %v", frame.ctx.stash != nil)
		if frame.ctx.stash != nil {
			t.Logf("Stash has names: %v (count: %d)", frame.ctx.stash.names != nil, len(frame.ctx.stash.names))
			t.Logf("Stash values count: %d", len(frame.ctx.stash.values))
			t.Logf("Stash has outer: %v", frame.ctx.stash.outer != nil)
			// Try to print stash names
			if frame.ctx.stash.names != nil {
				t.Logf("Stash names:")
				for name, idx := range frame.ctx.stash.names {
					t.Logf("  %s -> idx %d", name.String(), idx)
				}
			}
		}
	}
	
	// Test GetLocalVariables
	locals := frame.GetLocalVariables()
	if locals == nil {
		t.Fatal("GetLocalVariables returned nil")
	}
	
	t.Logf("Found %d local variables", len(locals))
	for name, value := range locals {
		t.Logf("  %s = %v (type: %T)", name, value, value)
	}

	// Check for expected variables
	if x, ok := locals["x"]; !ok {
		t.Error("Variable 'x' not found")
	} else if val, ok := x.(valueInt); !ok || int64(val) != 10 {
		t.Errorf("Expected x=10, got %v", x)
	}

	if y, ok := locals["y"]; !ok {
		t.Error("Variable 'y' not found")
	} else if val, ok := y.(valueInt); !ok || int64(val) != 20 {
		t.Errorf("Expected y=20, got %v", y)
	}

	if local1, ok := locals["local1"]; !ok {
		t.Error("Variable 'local1' not found")
	} else if val, ok := local1.(valueInt); !ok || int64(val) != 30 {
		t.Errorf("Expected local1=30, got %v", local1)
	}

	if local2, ok := locals["local2"]; !ok {
		t.Error("Variable 'local2' not found")
	} else if str, ok := local2.(String); !ok || str.String() != "hello" {
		t.Errorf("Expected local2='hello', got %v", local2)
	}

	if obj, ok := locals["obj"]; !ok {
		t.Error("Variable 'obj' not found")
	} else if _, ok := obj.(*Object); !ok {
		t.Errorf("Expected obj to be an object, got %T", obj)
	}
}

func TestStackFrameGetArguments(t *testing.T) {
	const script = `
	function testFunc(a, b, c) {
		debugger;
		return a + b + c;
	}
	
	testFunc(1, 2, 3);
	`

	r := New()
	debugger := r.EnableDebugger()
	
	var capturedStack []StackFrame
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		capturedStack = state.CallStack
		return DebugContinue
	})

	_, err := r.RunString(script)
	if err != nil {
		t.Fatal(err)
	}

	if len(capturedStack) == 0 {
		t.Fatal("No stack frames captured")
	}

	// Get the top frame
	frame := &capturedStack[0]
	
	// Test GetArguments
	args := frame.GetArguments()
	if args == nil {
		t.Fatal("GetArguments returned nil")
	}

	if len(args) != 3 {
		t.Fatalf("Expected 3 arguments, got %d", len(args))
	}

	// Check argument values
	for i, expected := range []int64{1, 2, 3} {
		if val, ok := args[i].(valueInt); !ok || int64(val) != expected {
			t.Errorf("Expected args[%d]=%d, got %v", i, expected, args[i])
		}
	}
}

func TestDebuggerEvaluateInFrame(t *testing.T) {
	const script = `
	var globalVar = 100;
	
	function outer() {
		var outerVar = 50;
		
		function inner() {
			var innerVar = 25;
			var x = 10;
			var y = 20;
			debugger;
			return innerVar;
		}
		
		return inner();
	}
	
	outer();
	`

	r := New()
	debugger := r.EnableDebugger()
	
	var evalResults []struct {
		expr   string
		result Value
		err    error
	}
	
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		// Test evaluating expressions in the current frame context
		expressions := []string{
			"x + y",
			"innerVar * 2",
			"x",
			"y",
			"innerVar",
		}
		
		for _, expr := range expressions {
			result, err := debugger.EvaluateInFrame(expr, 0)
			evalResults = append(evalResults, struct {
				expr   string
				result Value
				err    error
			}{expr, result, err})
		}
		
		return DebugContinue
	})

	_, err := r.RunString(script)
	if err != nil {
		t.Fatal(err)
	}

	// Check evaluation results
	expectedResults := map[string]int64{
		"x + y":       30,
		"innerVar * 2": 50,
		"x":           10,
		"y":           20,
		"innerVar":    25,
	}

	for _, res := range evalResults {
		if res.err != nil {
			t.Errorf("Error evaluating '%s': %v", res.expr, res.err)
			continue
		}
		
		if expected, ok := expectedResults[res.expr]; ok {
			if val, ok := res.result.(valueInt); !ok || int64(val) != expected {
				t.Errorf("Expression '%s': expected %d, got %v", res.expr, expected, res.result)
			}
		}
	}
}

func TestStackFrameMethodsWithNoContext(t *testing.T) {
	// Test that methods handle frames with no context gracefully
	frame := &StackFrame{
		funcName: newStringValue("test").string(),
		// ctx and vm are nil
	}
	
	// These should return nil/empty without panicking
	locals := frame.GetLocalVariables()
	if locals != nil {
		t.Error("Expected nil for GetLocalVariables with no context")
	}
	
	args := frame.GetArguments()
	if args != nil {
		t.Error("Expected nil for GetArguments with no context")
	}
	
	thisVal := frame.GetThis()
	if thisVal != nil {
		t.Error("Expected nil for GetThis with no context")
	}
}