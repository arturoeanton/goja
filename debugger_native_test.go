package goja

import (
	"testing"
)

func TestDebuggerNativeFunctions(t *testing.T) {
	const script = `
	var x = 10;
	console.log("Before debugger");
	debugger;
	console.log("After debugger");
	x = 20;
	`

	r := New()
	
	// Add console.log as a native function
	console := r.NewObject()
	console.Set("log", func(call FunctionCall) Value {
		// Native function that just returns undefined
		// In a real implementation, this would print to console
		return _undefined
	})
	r.Set("console", console)
	
	debugger := r.EnableDebugger()
	
	pauseCount := 0
	stepEvents := 0
	
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		pauseCount++
		t.Logf("Paused at line %d, PC: %d", state.SourcePos.Line, state.PC)
		
		if pauseCount == 1 {
			// First pause should be at the debugger statement
			// Let's try stepping
			return DebugStepInto
		} else {
			stepEvents++
			// Continue after stepping
			return DebugContinue
		}
	})

	_, err := r.RunString(script)
	if err != nil {
		t.Fatal(err)
	}
	
	// We should have paused once at the debugger statement
	if pauseCount < 1 {
		t.Error("Debugger didn't pause at debugger statement")
	}
	
	// Step events should not include native function internals
	t.Logf("Total pause events: %d, step events after debugger: %d", pauseCount, stepEvents)
}

func TestDebuggerStepOverNative(t *testing.T) {
	const script = `
	function test() {
		var a = 1;
		console.log("message"); // Should step over this
		var b = 2;
		return a + b;
	}
	debugger;
	test();
	`

	r := New()
	
	// Add console.log as a native function
	console := r.NewObject()
	console.Set("log", func(call FunctionCall) Value {
		return _undefined
	})
	r.Set("console", console)
	
	debugger := r.EnableDebugger()
	
	var events []string
	inTest := false
	
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		stack := r.CaptureCallStack(0, nil)
		funcName := "global"
		if len(stack) > 0 {
			fn := stack[0].FuncName()
			if fn != "" {
				funcName = string(fn)
			} else if len(stack) > 1 {
				// Try parent frame
				fn = stack[1].FuncName()
				if fn != "" {
					funcName = string(fn)
				}
			}
		}
		
		events = append(events, funcName)
		t.Logf("Event in function: %s, line: %d, stack depth: %d", funcName, state.SourcePos.Line, len(stack))
		
		if funcName == "test" && !inTest {
			inTest = true
			// When we enter test(), start stepping
			return DebugStepOver
		}
		
		if inTest && funcName == "test" {
			// Continue stepping in test function
			return DebugStepOver
		}
		
		return DebugContinue
	})

	_, err := r.RunString(script)
	if err != nil {
		t.Fatal(err)
	}
	
	// Check that we didn't get events from inside console.log
	hasNativeEvent := false
	for _, event := range events {
		if event != "global" && event != "test" && event != "" {
			hasNativeEvent = true
			t.Logf("Unexpected event in function: %s", event)
		}
	}
	
	if hasNativeEvent {
		t.Error("Debugger generated events inside native functions")
	}
}

func TestDebuggerWithDebugMode(t *testing.T) {
	const script = `
	function test(x) {
		var local = x * 2;
		console.log("Value: " + local);
		debugger;
		return local;
	}
	test(5);
	`

	// Test with debug mode to ensure native function filtering works with debug mode
	r := NewWithOptions(RuntimeOptions{EnableDebugMode: true})
	
	// Add console.log as a native function
	console := r.NewObject()
	console.Set("log", func(call FunctionCall) Value {
		return _undefined
	})
	r.Set("console", console)
	
	debugger := r.EnableDebugger()
	
	var foundLocal bool
	
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		if len(state.CallStack) > 0 {
			frame := &state.CallStack[0]
			if frame.FuncName() == "test" {
				locals := frame.GetLocalVariables()
				if val, ok := locals["local"]; ok {
					foundLocal = true
					t.Logf("Found local variable: local = %v", val)
				}
			}
		}
		return DebugContinue
	})

	result, err := r.RunString(script)
	if err != nil {
		t.Fatal(err)
	}
	
	if result.Export() != int64(10) {
		t.Errorf("Expected result 10, got %v", result.Export())
	}
	
	if !foundLocal {
		t.Error("Failed to find local variable in debug mode")
	}
}