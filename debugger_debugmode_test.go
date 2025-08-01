package goja

import (
	"testing"
)

func TestDebugModeVariables(t *testing.T) {
	const script = `
	function test(a, b) {
		var x = 10;
		var y = 20;
		let z = 30;
		const w = 40;
		debugger;
		return x + y + z + w + a + b;
	}
	test(1, 2);
	`

	// Test with debug mode enabled
	t.Run("DebugModeEnabled", func(t *testing.T) {
		r := NewWithOptions(RuntimeOptions{EnableDebugMode: true})
		if !r.IsDebugMode() {
			t.Fatal("Debug mode should be enabled")
		}
		
		debugger := r.EnableDebugger()
		
		var foundVariables map[string]Value
		
		debugger.SetHandler(func(state *DebuggerState) DebugCommand {
			if len(state.CallStack) > 0 {
				frame := &state.CallStack[0]
				locals := frame.GetLocalVariables()
				
				t.Logf("Frame: %s", frame.FuncName())
				t.Logf("Found %d local variables", len(locals))
				
				foundVariables = locals
				for name, value := range locals {
					t.Logf("  %s = %v", name, value)
				}
			}
			return DebugContinue
		})

		result, err := r.RunString(script)
		if err != nil {
			t.Fatal(err)
		}
		
		// Check result
		if result.Export() != int64(103) {
			t.Errorf("Expected result 103, got %v", result.Export())
		}
		
		// In debug mode, all variables should be in stash
		expectedVars := map[string]int64{
			"a": 1,
			"b": 2,
			"x": 10,
			"y": 20,
			"z": 30,
			"w": 40,
		}
		
		for name, expected := range expectedVars {
			if val, ok := foundVariables[name]; ok {
				if exported := val.Export(); exported != expected {
					t.Errorf("Variable %s: expected %d, got %v", name, expected, exported)
				}
			} else {
				t.Errorf("Variable %s not found in stash", name)
			}
		}
		
		if len(foundVariables) != len(expectedVars) {
			t.Errorf("Expected %d variables, found %d", len(expectedVars), len(foundVariables))
		}
	})

	// Test with debug mode disabled (default)
	t.Run("DebugModeDisabled", func(t *testing.T) {
		r := New()
		if r.IsDebugMode() {
			t.Fatal("Debug mode should be disabled by default")
		}
		
		debugger := r.EnableDebugger()
		
		var foundVariables map[string]Value
		
		debugger.SetHandler(func(state *DebuggerState) DebugCommand {
			if len(state.CallStack) > 0 {
				frame := &state.CallStack[0]
				locals := frame.GetLocalVariables()
				
				t.Logf("Frame: %s", frame.FuncName())
				t.Logf("Found %d local variables", len(locals))
				
				foundVariables = locals
				for name, value := range locals {
					t.Logf("  %s = %v", name, value)
				}
			}
			return DebugContinue
		})

		result, err := r.RunString(script)
		if err != nil {
			t.Fatal(err)
		}
		
		// Check result
		if result.Export() != int64(103) {
			t.Errorf("Expected result 103, got %v", result.Export())
		}
		
		// Without debug mode, most variables will be optimized to stack
		t.Logf("Variables found without debug mode: %d", len(foundVariables))
		if len(foundVariables) == 6 {
			t.Error("All variables found in stash without debug mode - optimization not working")
		}
	})
}

func TestDebugModeClosures(t *testing.T) {
	const script = `
	function outer() {
		var captured = "I am captured";
		var notCaptured = "I am not captured";
		
		function inner() {
			var innerVar = captured + " in closure";
			debugger;
			return innerVar;
		}
		
		return inner();
	}
	
	outer();
	`

	r := NewWithOptions(RuntimeOptions{EnableDebugMode: true})
	debugger := r.EnableDebugger()
	
	var foundVariables map[string]Value
	var frameCount int
	
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		frameCount = len(state.CallStack)
		if frameCount > 0 {
			frame := &state.CallStack[0]
			locals := frame.GetLocalVariables()
			
			t.Logf("Frame: %s", frame.FuncName())
			t.Logf("Found %d local variables", len(locals))
			
			foundVariables = locals
			for name, value := range locals {
				t.Logf("  %s = %v", name, value)
			}
		}
		return DebugContinue
	})

	_, err := r.RunString(script)
	if err != nil {
		t.Fatal(err)
	}
	
	// In debug mode, we should see all variables including innerVar
	if _, ok := foundVariables["innerVar"]; !ok {
		t.Error("innerVar not found in debug mode")
	}
}

func TestDebugModePerformance(t *testing.T) {
	// This test just ensures that debug mode doesn't crash
	// Performance testing should be done separately
	const script = `
	function fibonacci(n) {
		if (n <= 1) return n;
		var a = 0, b = 1, temp;
		for (var i = 2; i <= n; i++) {
			temp = a + b;
			a = b;
			b = temp;
		}
		return b;
	}
	
	fibonacci(10);
	`

	// Test with debug mode
	r1 := NewWithOptions(RuntimeOptions{EnableDebugMode: true})
	result1, err := r1.RunString(script)
	if err != nil {
		t.Fatal("Debug mode error:", err)
	}
	
	// Test without debug mode
	r2 := New()
	result2, err := r2.RunString(script)
	if err != nil {
		t.Fatal("Normal mode error:", err)
	}
	
	// Results should be the same
	if result1.Export() != result2.Export() {
		t.Errorf("Different results: debug=%v, normal=%v", result1.Export(), result2.Export())
	}
}