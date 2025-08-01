package goja

import (
	"testing"
)

func TestDebuggerBasic(t *testing.T) {
	const script = `
	var x = 10;
	debugger;
	x = 20;
	`

	r := New()
	debugger := r.EnableDebugger()
	
	triggered := false
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		triggered = true
		t.Logf("Debugger triggered at PC: %d", state.PC)
		t.Logf("Source position: %+v", state.SourcePos)
		t.Logf("Call stack length: %d", len(state.CallStack))
		return DebugContinue
	})

	result, err := r.RunString(script)
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}

	if !triggered {
		t.Fatal("Debugger was not triggered")
	}

	t.Logf("Script result: %v", result)
}

func TestDebuggerStepMode(t *testing.T) {
	const script = `
	var x = 10;
	var y = 20;
	var z = x + y;
	`

	r := New()
	debugger := r.EnableDebugger()
	
	steps := 0
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		steps++
		t.Logf("Step %d: PC=%d", steps, state.PC)
		if steps > 10 {
			return DebugContinue
		}
		return DebugStepInto
	})

	// Enable step mode
	debugger.SetStepMode(true)

	_, err := r.RunString(script)
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}

	if steps == 0 {
		t.Fatal("No steps were executed")
	}

	t.Logf("Total steps: %d", steps)
}