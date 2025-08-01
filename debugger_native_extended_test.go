package goja

import (
	"testing"
)

func TestDebuggerNativeFunctionNames(t *testing.T) {
	vm := New()
	debugger := vm.EnableDebugger()
	
	// Track native function calls
	nativeCalls := make(map[string]int)
	
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		if state.InNativeCall {
			nativeCalls[state.NativeFunctionName]++
		}
		return DebugContinue
	})
	
	// Enable step mode
	debugger.SetStepMode(true)
	
	// Add console with named function
	console := vm.NewObject()
	console.Set("log", func(call FunctionCall) Value {
		return _undefined
	})
	vm.Set("console", console)
	
	// Add named native function
	vm.Set("myNamedFunc", func(call FunctionCall) Value {
		return vm.ToValue("named result")
	})
	
	script := `
		console.log("test");
		myNamedFunc();
		[1,2,3].map(function(x) { return x * 2; });
	`
	
	_, err := vm.RunString(script)
	if err != nil {
		t.Fatal(err)
	}
	
	// Check results
	t.Logf("Native calls detected:")
	for name, count := range nativeCalls {
		t.Logf("  %s: %d calls", name, count)
	}
	
	// We should have detected at least console.log and myNamedFunc
	if nativeCalls["native"] == 0 && nativeCalls["log"] == 0 {
		t.Error("console.log not detected")
	}
	
	if nativeCalls["native"] == 0 && nativeCalls["myNamedFunc"] == 0 {
		t.Error("myNamedFunc not detected")
	}
}

func TestDebuggerArrayMethodsNative(t *testing.T) {
	vm := New()
	debugger := vm.EnableDebugger()
	
	// Track all events
	var events []struct {
		isNative bool
		name     string
		line     int
	}
	
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		events = append(events, struct {
			isNative bool
			name     string
			line     int
		}{
			isNative: state.InNativeCall,
			name:     state.NativeFunctionName,
			line:     state.SourcePos.Line,
		})
		return DebugContinue
	})
	
	debugger.SetStepMode(true)
	
	script := `
		var arr = [1, 2, 3];
		var result = arr.map(function(x) { 
			return x * 2; 
		});
		result;
	`
	
	_, err := vm.RunString(script)
	if err != nil {
		t.Fatal(err)
	}
	
	// Count native vs non-native events
	nativeCount := 0
	jsCount := 0
	for _, event := range events {
		if event.isNative {
			nativeCount++
		} else {
			jsCount++
		}
	}
	
	t.Logf("Total events: %d (native: %d, JS: %d)", len(events), nativeCount, jsCount)
	
	// We should have both native and JS events
	if nativeCount == 0 {
		t.Error("No native events detected for Array.map")
	}
	
	if jsCount == 0 {
		t.Error("No JS events detected")
	}
}

func TestDebuggerSkipNativeWithShouldStep(t *testing.T) {
	vm := New()
	debugger := vm.EnableDebugger()
	
	// Count events that would be shown vs skipped
	shownEvents := 0
	skippedEvents := 0
	
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		if state.InNativeCall {
			// Check if DAP implementation would skip this
			if !debugger.ShouldStepInNativeCall() {
				skippedEvents++
			} else {
				shownEvents++
			}
		} else {
			shownEvents++
		}
		return DebugContinue
	})
	
	debugger.SetStepMode(true)
	
	// Add some native functions
	vm.Set("nativeFunc", func(call FunctionCall) Value {
		// Simulate some work
		return vm.ToValue(42)
	})
	
	script := `
		var x = nativeFunc();
		var y = x + 1;
		var z = nativeFunc() + nativeFunc();
		z;
	`
	
	_, err := vm.RunString(script)
	if err != nil {
		t.Fatal(err)
	}
	
	t.Logf("Events shown: %d, skipped: %d", shownEvents, skippedEvents)
	
	// Verify that native events would be skipped by default
	if skippedEvents == 0 {
		t.Error("Expected some native events to be marked for skipping")
	}
	
	if shownEvents == 0 {
		t.Error("Expected some events to be shown")
	}
}