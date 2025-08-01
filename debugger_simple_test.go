package goja

import (
	"testing"
)

func TestDebuggerSimpleVars(t *testing.T) {
	const script = `
	var x = 10;
	var y = 20;
	debugger;
	`

	r := New()
	debugger := r.EnableDebugger()
	
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		t.Logf("=== Debugger State ===")
		t.Logf("PC: %d", state.PC)
		t.Logf("CallStack length: %d", len(state.CallStack))
		t.Logf("DebugStack length: %d", len(state.DebugStack))
		
		if len(state.DebugStack) > 0 {
			frame := state.DebugStack[0]
			t.Logf("Frame 0 scopes: %d", len(frame.Scopes))
			
			for i, scope := range frame.Scopes {
				t.Logf("  Scope %d: %s (ref=%d)", i, scope.Name, scope.VariablesRef)
				
				vars := debugger.GetVariables(scope.VariablesRef)
				t.Logf("    Variables: %d", len(vars))
				
				for _, v := range vars {
					t.Logf("      %s (%s) = %v", v.Name, v.Type, v.Value)
				}
			}
		}
		
		// Also try accessing via GetScopes
		scopes := debugger.GetScopes(0)
		t.Logf("\nGetScopes(0) returned %d scopes", len(scopes))
		
		return DebugContinue
	})

	_, err := r.RunString(script)
	if err != nil {
		t.Fatal(err)
	}
}