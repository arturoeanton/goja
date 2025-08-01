# Debugger Native Function API Summary

This document describes the API additions for handling native functions in the Goja debugger, making it easier for DAP implementations to handle native function calls appropriately.

## Problem Statement

When debugging JavaScript code that calls native Go functions (like `console.log`, `Array.map`, etc.), the debugger generates multiple confusing events as it steps through the native implementation. This creates a poor debugging experience in UI debuggers.

## Solution

We've added API methods and state information to help DAP implementations detect and handle native function contexts appropriately.

## API Additions

### 1. DebuggerState Fields

```go
type DebuggerState struct {
    // ... existing fields ...
    
    // InNativeCall indicates if currently executing a native function
    InNativeCall bool
    
    // NativeFunctionName contains the name of the native function being executed
    // Returns "native" for anonymous functions, "<native>" if name cannot be determined
    NativeFunctionName string
}
```

### 2. Debugger Methods

```go
// IsInNativeCall returns true if currently executing native code
// This method can be called at any time to check the current state
func (d *Debugger) IsInNativeCall() bool

// GetNativeFunctionName returns the name of the currently executing native function
// Returns empty string if not in a native call
func (d *Debugger) GetNativeFunctionName() string

// ShouldStepInNativeCall returns true if debugger should process step events in native calls
// By default returns false to avoid confusing multiple events in native functions
// DAP implementations can use this to decide whether to show stepping in native functions
func (d *Debugger) ShouldStepInNativeCall() bool
```

## Usage Example

```go
debugger.SetHandler(func(state *DebuggerState) DebugCommand {
    if state.InNativeCall {
        // We're in a native function
        log.Printf("Native function: %s", state.NativeFunctionName)
        
        // DAP implementation might choose to:
        // 1. Skip showing this event to the user
        // 2. Show a simplified view
        // 3. Automatically continue execution
        
        if !debugger.ShouldStepInNativeCall() {
            // Skip processing this event in the UI
            return DebugContinue
        }
    }
    
    // Normal debugging logic for JavaScript code
    return handleNormalDebugging(state)
})
```

## Implementation Details

1. **Detection**: Native functions are detected by checking if `vm.prg == nil` during execution
2. **Events**: Debug events are still generated when entering native functions, but with `InNativeCall=true`
3. **Names**: Function names are extracted from the function object and cleaned up for readability
4. **No Flag Storage**: The implementation doesn't store flags; it checks the VM state in real-time

## Benefits for DAP Implementations

1. **Clear Context**: DAP can easily detect when in native code
2. **Better UX**: Can hide confusing internal stepping in native functions
3. **Flexibility**: DAP implementations can choose how to handle native calls
4. **Clean Names**: Native function names are cleaned up (e.g., "log" instead of "github.com/dop251/goja.TestFunc.func1")

## Example Scenarios

### Console.log
```javascript
console.log("Hello");  // Generates event with InNativeCall=true, NativeFunctionName="native"
```

### Array methods
```javascript
[1,2,3].map(x => x*2);  // map generates events with InNativeCall=true, NativeFunctionName="map"
```

### Custom native functions
```javascript
myNativeFunc();  // InNativeCall=true, NativeFunctionName="myNativeFunc" or "native"
```

## Testing

The implementation includes comprehensive tests in `debugger_native_test.go` and `debugger_native_extended_test.go` that verify:
- Native calls are properly detected
- Function names are correctly extracted
- The API methods work as expected
- Both built-in and custom native functions are handled