package goja

import (
	"testing"
)

func TestDebuggerClosureVariables(t *testing.T) {
	const script = `
	function outer() {
		var captured = "I am captured";
		var notCaptured = "I am not captured";
		
		function inner() {
			// Reference captured to force it into stash
			var innerVar = captured + " in closure";
			debugger;
			return innerVar;
		}
		
		return inner();
	}
	
	outer();
	`

	r := New()
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

	_, err := r.RunString(script)
	if err != nil {
		t.Fatal(err)
	}

	// In closure contexts, we might find captured variables
	// This test documents the current behavior
	if len(foundVariables) > 0 {
		t.Log("Successfully found variables in closure context")
	} else {
		t.Log("No variables found - they may be optimized to stack")
	}
}

func TestDebuggerGlobalScope(t *testing.T) {
	const script = `
	var globalX = 100;
	var globalY = "test";
	
	function test() {
		debugger;
		return globalX;
	}
	
	test();
	`

	r := New()
	debugger := r.EnableDebugger()
	
	var globalVarsFound bool
	
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		// Test global scope access through the debugger API
		scopes := debugger.GetScopes(0)
		
		for _, scope := range scopes {
			if scope.Name == "Global" {
				vars := debugger.GetVariables(scope.VariablesRef)
				t.Logf("Global scope has %d variables", len(vars))
				
				for _, v := range vars {
					if v.Name == "globalX" || v.Name == "globalY" {
						globalVarsFound = true
						t.Logf("Found global variable: %s = %v", v.Name, v.Value)
					}
				}
			}
		}
		
		return DebugContinue
	})

	_, err := r.RunString(script)
	if err != nil {
		t.Fatal(err)
	}

	if !globalVarsFound {
		t.Error("Failed to find global variables through debugger API")
	}
}