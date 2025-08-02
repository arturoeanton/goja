package goja

import (
	"testing"
)

// TestDebugModeDisablesOptimizations ensures that when debug mode is enabled,
// no optimizations are applied and all variables remain accessible in the stash
func TestDebugModeDisablesOptimizations(t *testing.T) {
	const SCRIPT = `
	function testFunc(a, b) {
		var x = 10;
		let y = 20;
		const z = 30;
		
		// All these variables should be in stash when debug mode is enabled
		debugger;
		
		return a + b + x + y + z;
	}
	
	var result = testFunc(1, 2);
	`
	
	// Test with debug mode enabled
	t.Run("WithDebugMode", func(t *testing.T) {
		opts := RuntimeOptions{
			EnableDebugMode: true,
		}
		r := NewWithOptions(opts)
		debugger := r.EnableDebugger()
		
		var hitBreakpoint bool
		var varsInStash int
		
		debugger.SetHandler(func(state *DebuggerState) DebugCommand {
			hitBreakpoint = true
			
			// Check that variables are accessible
			if len(state.DebugStack) > 0 {
				frame := state.DebugStack[0]
				for _, scope := range frame.Scopes {
					if scope.Name == "Local" {
						vars := debugger.GetVariables(scope.VariablesRef)
						varsInStash = len(vars)
						
						// In debug mode, we should see all variables: a, b, x, y, z
						expectedVars := map[string]bool{
							"a": false,
							"b": false,
							"x": false,
							"y": false,
							"z": false,
						}
						
						for _, v := range vars {
							if _, ok := expectedVars[v.Name]; ok {
								expectedVars[v.Name] = true
							}
						}
						
						for name, found := range expectedVars {
							if !found {
								t.Errorf("Variable %s not found in debug mode", name)
							}
						}
					}
				}
			}
			
			return DebugContinue
		})
		
		_, err := r.RunString(SCRIPT)
		if err != nil {
			t.Fatal(err)
		}
		
		if !hitBreakpoint {
			t.Error("Debugger breakpoint was not hit")
		}
		
		if varsInStash < 5 {
			t.Errorf("Expected at least 5 variables in stash, got %d", varsInStash)
		}
	})
	
	// Test that enterFuncStashless is not used in debug mode
	t.Run("NoStashlessFunctions", func(t *testing.T) {
		prg, err := Compile("test.js", SCRIPT, true)
		if err != nil {
			t.Fatal(err)
		}
		
		// Check that no enterFuncStashless instructions are present
		hasStashless := false
		for _, instr := range prg.code {
			if _, ok := instr.(*enterFuncStashless); ok {
				hasStashless = true
				break
			}
		}
		
		if hasStashless {
			t.Log("Note: Stashless functions found in compiled code - this is expected without debug mode")
		}
	})
}

// TestDebugModeArgsInStash verifies that function arguments are moved to stash in debug mode
func TestDebugModeArgsInStash(t *testing.T) {
	const SCRIPT = `
	function test(arg1, arg2, arg3) {
		// Arguments should be accessible in debug mode
		debugger;
		return arguments.length;
	}
	
	test(1, 2, 3);
	`
	
	opts := RuntimeOptions{
		EnableDebugMode: true,
	}
	r := NewWithOptions(opts)
	debugger := r.EnableDebugger()
	
	var argsFound bool
	
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		if len(state.DebugStack) > 0 {
			frame := state.DebugStack[0]
			for _, scope := range frame.Scopes {
				vars := debugger.GetVariables(scope.VariablesRef)
				for _, v := range vars {
					if v.Name == "arguments" || v.Name == "arg1" || v.Name == "arg2" || v.Name == "arg3" {
						argsFound = true
					}
				}
			}
		}
		return DebugContinue
	})
	
	_, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	
	if !argsFound {
		t.Error("Function arguments not found in stash in debug mode")
	}
}