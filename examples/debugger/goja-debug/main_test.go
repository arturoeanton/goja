package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestDebuggerConsoleCreation(t *testing.T) {
	dc := NewDebuggerConsole()
	if dc == nil {
		t.Fatal("Failed to create DebuggerConsole")
	}
	
	if dc.breakpoints == nil {
		t.Error("Breakpoints map not initialized")
	}
	
	if dc.variableRefs == nil {
		t.Error("Variable refs map not initialized")
	}
}

func TestLoadFile(t *testing.T) {
	dc := NewDebuggerConsole()
	
	// Create a test file
	testContent := `
var x = 10;
var y = 20;
console.log(x + y);
`
	
	// Write test file
	err := ioutil.WriteFile("test_temp.js", []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("test_temp.js")
	
	// Load the file
	err = dc.LoadFile("test_temp.js")
	if err != nil {
		t.Fatalf("Failed to load file: %v", err)
	}
	
	// Check that source was loaded
	if dc.source != testContent {
		t.Error("Source content mismatch")
	}
	
	// Check that lines were split correctly
	expectedLines := 5 // Including empty lines
	if len(dc.sourceLines) != expectedLines {
		t.Errorf("Expected %d lines, got %d", expectedLines, len(dc.sourceLines))
	}
	
	// Check runtime and debugger were initialized
	if dc.runtime == nil {
		t.Error("Runtime not initialized")
	}
	
	if dc.debugger == nil {
		t.Error("Debugger not initialized")
	}
}

func TestBreakpointManagement(t *testing.T) {
	dc := NewDebuggerConsole()
	
	// Initialize runtime
	dc.runtime = goja.New()
	dc.debugger = dc.runtime.EnableDebugger()
	dc.filename = "test.js"
	
	// Test setting breakpoint
	dc.setBreakpoint("5")
	
	// Check that breakpoint was recorded
	if _, ok := dc.breakpoints[5]; !ok {
		t.Error("Breakpoint not recorded in map")
	}
	
	// Test deleting breakpoint
	dc.deleteBreakpoint("5")
	
	// Check that breakpoint was removed
	if _, ok := dc.breakpoints[5]; ok {
		t.Error("Breakpoint not removed from map")
	}
}

func TestVariablePrinting(t *testing.T) {
	dc := NewDebuggerConsole()
	
	testCases := []struct {
		name     string
		variable goja.Variable
		expected string
	}{
		{
			name: "String variable",
			variable: goja.Variable{
				Name:  "str",
				Type:  "string",
			},
		},
		{
			name: "Number variable",
			variable: goja.Variable{
				Name:  "num",
				Type:  "number",
			},
		},
		{
			name: "Undefined variable",
			variable: goja.Variable{
				Name: "undef",
				Type: "undefined",
			},
		},
		{
			name: "Object variable",
			variable: goja.Variable{
				Name: "obj",
				Type: "object",
				Ref:  1001,
			},
		},
	}
	
	// Just test that printVariable doesn't panic
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This test just ensures the function doesn't panic
			dc.printVariable(&tc.variable)
		})
	}
}

func TestCommandProcessing(t *testing.T) {
	dc := NewDebuggerConsole()
	// Initialize runtime and debugger to avoid nil pointer errors
	dc.runtime = goja.New()
	dc.debugger = dc.runtime.EnableDebugger()
	dc.filename = "test.js"
	
	testCases := []struct {
		input    string
		expected int
	}{
		{"c", int(goja.DebugContinue)},
		{"continue", int(goja.DebugContinue)},
		{"n", int(goja.DebugStepOver)},
		{"next", int(goja.DebugStepOver)},
		{"over", int(goja.DebugStepOver)},
		{"s", int(goja.DebugStepInto)},
		{"step", int(goja.DebugStepInto)},
		{"into", int(goja.DebugStepInto)},
		{"o", int(goja.DebugStepOut)},
		{"out", int(goja.DebugStepOut)},
		{"b 10", -1},      // Breakpoint commands return -1
		{"d 10", -1},      // Delete breakpoint returns -1
		{"lb", -1},        // List breakpoints returns -1
		{"p x + y", -1},   // Print expression returns -1
		{"v 1001", -1},    // Inspect variable returns -1
		{"locals", -1},    // Show locals returns -1
		{"globals", -1},   // Show globals returns -1
		{"invalid", -1},   // Invalid command returns -1
	}
	
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := dc.processCommand(tc.input)
			if result != tc.expected {
				t.Errorf("Command %q: expected %d, got %d", tc.input, tc.expected, result)
			}
		})
	}
}

func TestDebuggerIntegration(t *testing.T) {
	// Create a simple test script
	script := `
var counter = 0;
function increment() {
    counter++;
    return counter;
}
var result = increment();
console.log("Result:", result);
`
	
	runtime := goja.New()
	// Register console for testing
	console := runtime.NewObject()
	console.Set("log", func(args ...interface{}) {
		// Simple console.log implementation for testing
	})
	runtime.Set("console", console)
	debugger := runtime.EnableDebugger()
	
	// Test state tracking
	var capturedState *goja.DebuggerState
	stepCount := 0
	
	debugger.SetHandler(func(state *goja.DebuggerState) goja.DebugCommand {
		capturedState = state
		stepCount++
		
		// Step through first 5 instructions then continue
		if stepCount < 5 {
			return goja.DebugStepInto
		}
		return goja.DebugContinue
	})
	
	debugger.SetStepMode(true)
	
	_, err := runtime.RunString(script)
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}
	
	// Verify we stepped through the code
	if stepCount < 5 {
		t.Errorf("Expected at least 5 steps, got %d", stepCount)
	}
	
	// Verify we captured state
	if capturedState == nil {
		t.Error("No debugger state captured")
	}
}

func TestNativeFunctionDetection(t *testing.T) {
	script := `
var arr = [1, 2, 3];
var doubled = arr.map(function(x) { return x * 2; });
`
	
	runtime := goja.New()
	debugger := runtime.EnableDebugger()
	
	nativeFunctionDetected := false
	
	debugger.SetHandler(func(state *goja.DebuggerState) goja.DebugCommand {
		if state.InNativeCall {
			nativeFunctionDetected = true
			if !strings.Contains(state.NativeFunctionName, "map") {
				t.Errorf("Expected native function name to contain 'map', got %s", state.NativeFunctionName)
			}
		}
		return goja.DebugStepInto
	})
	
	debugger.SetStepMode(true)
	
	_, err := runtime.RunString(script)
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}
	
	if !nativeFunctionDetected {
		t.Error("Native function call was not detected")
	}
}

func TestVariableScopes(t *testing.T) {
	script := `
var globalVar = "global";
function testFunc(param) {
    var localVar = "local";
    return localVar + param;
}
testFunc("test");
`
	
	runtime := goja.New()
	debugger := runtime.EnableDebugger()
	
	scopesFound := map[string]bool{
		"Global": false,
		"Local":  false,
	}
	
	debugger.SetHandler(func(state *goja.DebuggerState) goja.DebugCommand {
		if len(state.DebugStack) > 0 {
			frame := state.DebugStack[0]
			for _, scope := range frame.Scopes {
				scopesFound[scope.Name] = true
			}
		}
		return goja.DebugContinue
	})
	
	debugger.SetStepMode(true)
	
	_, err := runtime.RunString(script)
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}
	
	// We should have seen at least the global scope
	if !scopesFound["Global"] {
		t.Error("Global scope not found")
	}
}