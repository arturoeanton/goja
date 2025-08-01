package goja

import (
	"fmt"
	"sync"
)

// DebugFlags controls the debugging behavior
type DebugFlags uint32

const (
	// FlagStepMode enables step-by-step execution
	FlagStepMode DebugFlags = 1 << iota
	// FlagPaused indicates the VM is currently paused
	FlagPaused
)

// Position represents a position in source code
type Position struct {
	Filename string
	Line     int
	Column   int
}

// Breakpoint represents a breakpoint in the code
type Breakpoint struct {
	id        int
	SourcePos Position // Position in source code
	pc        int      // Program counter position (-1 if not resolved)
	enabled   bool
	hit       int // Number of times this breakpoint was hit
}

// ID returns the breakpoint ID
func (b *Breakpoint) ID() int {
	return b.id
}

// Variable represents a variable in the debug context
type Variable struct {
	Name  string
	Value Value
	Type  string
	Ref   int // Reference ID for complex types (objects, arrays)
}

// Scope represents a variable scope
type Scope struct {
	Name          string // "Local", "Closure", "Global"
	VariablesRef  int    // Reference ID to get variables
	Expensive     bool   // If true, variables should be retrieved lazily
	NamedVariables int   // Number of named variables
	IndexedVariables int // Number of indexed variables (for arrays)
}

// DebugStackFrame extends StackFrame with debug information
type DebugStackFrame struct {
	StackFrame
	Scopes []Scope
	This   *Variable // The 'this' value in the context
}

// DebuggerState represents the current state when paused
type DebuggerState struct {
	PC         int
	SourcePos  Position
	CallStack  []StackFrame
	DebugStack []DebugStackFrame // Extended stack frames with variable info
	Breakpoint *Breakpoint // Current breakpoint if stopped at one
	StepMode   bool
}

// DebugHandler is called when the debugger pauses execution
type DebugHandler func(state *DebuggerState) DebugCommand

// DebugCommand represents commands that can be sent to the debugger
type DebugCommand int

const (
	DebugContinue DebugCommand = iota
	DebugStepOver
	DebugStepInto
	DebugStepOut
	DebugPause
)

// Debugger provides debugging capabilities for the Runtime
type Debugger struct {
	mu          sync.RWMutex
	runtime     *Runtime
	breakpoints map[int]*Breakpoint
	nextID      int
	flags       DebugFlags
	handler     DebugHandler

	// Internal state
	pcBreakpoints map[int]*Breakpoint // PC to breakpoint mapping for fast lookup
	stepDepth     int                 // Call stack depth for step over/out
	stepMode      DebugCommand
	
	// Variable reference management for DAP
	variableRefs map[int]interface{} // Maps reference IDs to values or scopes
	nextVarRef   int
}

// NewDebugger creates a new debugger for the runtime
func (r *Runtime) NewDebugger() *Debugger {
	return &Debugger{
		runtime:       r,
		breakpoints:   make(map[int]*Breakpoint),
		pcBreakpoints: make(map[int]*Breakpoint),
		variableRefs:  make(map[int]interface{}),
		nextVarRef:    1000, // Start from 1000 to avoid conflicts with frame IDs
	}
}

// SetHandler sets the debug handler that will be called when execution pauses
func (d *Debugger) SetHandler(handler DebugHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handler = handler
}

// AddBreakpoint adds a breakpoint at the specified source position
func (d *Debugger) AddBreakpoint(filename string, line, column int) int {
	d.mu.Lock()
	defer d.mu.Unlock()

	bp := &Breakpoint{
		id: d.nextID,
		SourcePos: Position{
			Filename: filename,
			Line:     line,
			Column:   column,
		},
		pc:      -1,
		enabled: true,
	}

	d.nextID++
	d.breakpoints[bp.id] = bp

	// Try to resolve the breakpoint to a PC if we have a program loaded
	if d.runtime.vm != nil && d.runtime.vm.prg != nil {
		d.resolveBreakpoint(bp)
	}

	return bp.id
}

// RemoveBreakpoint removes a breakpoint by ID
func (d *Debugger) RemoveBreakpoint(id int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	bp, exists := d.breakpoints[id]
	if !exists {
		return false
	}

	delete(d.breakpoints, id)
	if bp.pc >= 0 {
		delete(d.pcBreakpoints, bp.pc)
	}

	return true
}

// EnableBreakpoint enables or disables a breakpoint
func (d *Debugger) EnableBreakpoint(id int, enabled bool) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	bp, exists := d.breakpoints[id]
	if !exists {
		return false
	}

	bp.enabled = enabled
	return true
}

// GetBreakpoints returns all breakpoints
func (d *Debugger) GetBreakpoints() []*Breakpoint {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]*Breakpoint, 0, len(d.breakpoints))
	for _, bp := range d.breakpoints {
		result = append(result, bp)
	}

	return result
}

// SetStepMode enables or disables step mode
func (d *Debugger) SetStepMode(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if enabled {
		d.flags |= FlagStepMode
		d.stepMode = DebugStepInto // Default to step into
	} else {
		d.flags &^= FlagStepMode
	}
}

// Continue resumes execution
func (d *Debugger) Continue() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.flags &^= FlagPaused
	d.stepMode = DebugContinue
}

// StepOver executes the next line, stepping over function calls
func (d *Debugger) StepOver() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.flags |= FlagStepMode
	d.flags &^= FlagPaused
	d.stepMode = DebugStepOver
	d.stepDepth = len(d.runtime.vm.callStack)
}

// StepInto executes the next line, stepping into function calls
func (d *Debugger) StepInto() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.flags |= FlagStepMode
	d.flags &^= FlagPaused
	d.stepMode = DebugStepInto
}

// StepOut continues execution until the current function returns
func (d *Debugger) StepOut() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.flags &^= FlagPaused
	d.stepMode = DebugStepOut
	d.stepDepth = len(d.runtime.vm.callStack) - 1
}

// Pause pauses execution at the next opportunity
func (d *Debugger) Pause() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.flags |= FlagPaused
}

// resolveBreakpoint tries to resolve a source position to a PC
func (d *Debugger) resolveBreakpoint(bp *Breakpoint) {
	prg := d.runtime.vm.prg
	if prg == nil || prg.src == nil {
		return
	}

	// Find the PC that corresponds to this source position
	// We need to search through srcMap items
	for pc := 0; pc < len(prg.code); pc++ {
		if pc < len(prg.srcMap) {
			item := prg.srcMap[pc]
			if item.srcPos >= 0 {
				pos := prg.src.Position(item.srcPos)
				if pos.Filename == bp.SourcePos.Filename &&
					pos.Line == bp.SourcePos.Line {
					// Found a match - we accept any column on the same line
					bp.pc = pc
					d.pcBreakpoints[pc] = bp
					break
				}
			}
		}
	}
}

// resolvePendingBreakpoints resolves all breakpoints that haven't been resolved yet
func (d *Debugger) resolvePendingBreakpoints() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, bp := range d.breakpoints {
		if bp.pc < 0 {
			d.resolveBreakpoint(bp)
		}
	}
}

// checkBreakpoint is called by the VM to check if we should pause
func (d *Debugger) checkBreakpoint(vm *vm) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Skip debugger events when in native functions
	// This prevents confusing debug events when executing native code like console.log
	// But still allow paused state and explicit breakpoints
	if vm.prg == nil {
		// Only check if already paused (from debugger statement for example)
		return d.flags&FlagPaused != 0
	}

	// Check if paused
	if d.flags&FlagPaused != 0 {
		return true
	}

	// Check breakpoints
	if bp, exists := d.pcBreakpoints[vm.pc]; exists && bp.enabled {
		bp.hit++
		d.flags |= FlagPaused
		return true
	}

	// Check step mode
	if d.flags&FlagStepMode != 0 {
		switch d.stepMode {
		case DebugStepInto:
			d.flags |= FlagPaused
			return true
		case DebugStepOver:
			if len(vm.callStack) <= d.stepDepth {
				d.flags |= FlagPaused
				return true
			}
		case DebugStepOut:
			if len(vm.callStack) < d.stepDepth {
				d.flags |= FlagPaused
				return true
			}
		}
	}

	return false
}

// handlePause is called when the VM pauses
func (d *Debugger) handlePause(vm *vm) {
	// Skip if in native function
	if vm.prg == nil {
		return
	}
	
	d.mu.RLock()
	handler := d.handler
	d.mu.RUnlock()

	if handler == nil {
		// No handler, just continue
		d.Continue()
		return
	}

	// Build debug state
	state := &DebuggerState{
		PC:       vm.pc,
		StepMode: d.flags&FlagStepMode != 0,
	}

	// Get source position
	if vm.prg != nil && vm.prg.srcMap != nil && vm.pc < len(vm.prg.srcMap) {
		item := vm.prg.srcMap[vm.pc]
		if item.srcPos >= 0 && vm.prg.src != nil {
			pos := vm.prg.src.Position(item.srcPos)
			state.SourcePos = Position{
				Filename: pos.Filename,
				Line:     pos.Line,
				Column:   pos.Column,
			}
		}
	}

	// Get current breakpoint if any
	d.mu.RLock()
	if bp, exists := d.pcBreakpoints[vm.pc]; exists {
		state.Breakpoint = bp
	}
	d.mu.RUnlock()

	// Capture call stack
	state.CallStack = d.runtime.CaptureCallStack(0, nil)
	
	// Build debug stack with variable information
	state.DebugStack = d.buildDebugStack(vm)

	// Call handler and process command
	cmd := handler(state)
	switch cmd {
	case DebugContinue:
		d.Continue()
	case DebugStepOver:
		d.StepOver()
	case DebugStepInto:
		d.StepInto()
	case DebugStepOut:
		d.StepOut()
	case DebugPause:
		// Already paused, do nothing
	}
}

// GetScopes returns the scopes for a given stack frame
func (d *Debugger) GetScopes(frameID int) []Scope {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.runtime.vm == nil || frameID >= len(d.runtime.vm.callStack) {
		return nil
	}

	// For simplicity, we'll start with just local scope
	// In the future, we can add closure and global scopes
	scopes := []Scope{
		{
			Name:         "Local",
			VariablesRef: d.createVariableRef(frameID, "local"),
			Expensive:    false,
		},
	}

	// Add global scope
	scopes = append(scopes, Scope{
		Name:         "Global",
		VariablesRef: d.createVariableRef(frameID, "global"),
		Expensive:    true,
	})

	return scopes
}

// createVariableRef creates a reference for variables lookup
func (d *Debugger) createVariableRef(frameID int, scopeType string) int {
	// Note: This function must be called with write lock held
	ref := d.nextVarRef
	d.nextVarRef++
	d.variableRefs[ref] = map[string]interface{}{
		"frameID": frameID,
		"type":    scopeType,
	}
	return ref
}

// GetVariables returns variables for a given reference (scope or object)
func (d *Debugger) GetVariables(variablesRef int) []Variable {
	// Handle lazy scope references (negative refs)
	if variablesRef < 0 {
		return d.resolveLazyScope(variablesRef)
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	refData, exists := d.variableRefs[variablesRef]
	if !exists {
		// Debug: log what references we have
		if d.runtime.vm != nil {
			for k, v := range d.variableRefs {
				_ = k
				_ = v
				// fmt.Printf("Debug: Have ref %d -> %T\n", k, v)
			}
		}
		return nil
	}

	switch data := refData.(type) {
	case map[string]interface{}:
		if scopeType, ok := data["type"].(string); ok {
			frameID, _ := data["frameID"].(int)
			return d.getVariablesForScope(frameID, scopeType)
		}
	case *Object:
		// Handle object properties
		return d.getObjectProperties(data)
	}

	return nil
}

// getVariablesForScope retrieves variables for a specific scope
func (d *Debugger) getVariablesForScope(frameID int, scopeType string) []Variable {
	var variables []Variable
	
	// Get the current call stack
	stack := d.runtime.CaptureCallStack(0, nil)
	if frameID < 0 || frameID >= len(stack) {
		return nil
	}
	
	switch scopeType {
	case "local":
		// Use the new GetLocalVariables method
		frame := &stack[frameID]
		localVars := frame.GetLocalVariables()
		if localVars != nil {
			variables = make([]Variable, 0, len(localVars))
			for name, value := range localVars {
				variable := Variable{
					Name:  name,
					Value: value,
					Type:  d.getValueType(value),
				}
				
				// For complex types, create a reference
				if obj, ok := value.(*Object); ok {
					d.mu.Lock()
					ref := d.nextVarRef
					d.nextVarRef++
					d.variableRefs[ref] = obj
					d.mu.Unlock()
					variable.Ref = ref
				}
				
				variables = append(variables, variable)
			}
		}
	case "global":
		// Get global variables
		variables = d.extractGlobalVariables()
	}

	return variables
}

// extractStashVariables extracts variables from a stash
func (d *Debugger) extractStashVariables(s *stash) []Variable {
	if s == nil || s.names == nil {
		return nil
	}

	variables := make([]Variable, 0, len(s.names))
	
	for name, idx := range s.names {
		if int(idx) < len(s.values) && s.values[idx] != nil {
			v := s.values[idx]
			variable := Variable{
				Name:  name.String(),
				Value: v,
				Type:  d.getValueType(v),
			}
			
			// For complex types, create a reference
			if obj, ok := v.(*Object); ok {
				ref := d.nextVarRef
				d.nextVarRef++
				d.variableRefs[ref] = obj
				variable.Ref = ref
			}
			
			variables = append(variables, variable)
		}
	}

	return variables
}

// extractGlobalVariables extracts global variables
func (d *Debugger) extractGlobalVariables() []Variable {
	globalObj := d.runtime.globalObject
	if globalObj == nil {
		return nil
	}

	var variables []Variable
	
	// Iterate through global object properties
	for item, next := globalObj.self.iterateStringKeys()(); next != nil; item, next = next() {
		nameStr := item.name.string()
		prop := globalObj.self.getStr(nameStr, nil)
		if prop != nil {
			variable := Variable{
				Name:  nameStr.String(),
				Value: prop,
				Type:  d.getValueType(prop),
			}
			
			// For complex types, create a reference
			if obj, ok := prop.(*Object); ok {
				ref := d.nextVarRef
				d.nextVarRef++
				d.variableRefs[ref] = obj
				variable.Ref = ref
			}
			
			variables = append(variables, variable)
		}
	}

	return variables
}

// getObjectProperties extracts properties from an object
func (d *Debugger) getObjectProperties(obj *Object) []Variable {
	if obj == nil {
		return nil
	}

	var variables []Variable
	
	// Iterate through object properties
	for item, next := obj.self.iterateStringKeys()(); next != nil; item, next = next() {
		nameStr := item.name.string()
		prop := obj.self.getStr(nameStr, nil)
		if prop != nil {
			variable := Variable{
				Name:  nameStr.String(),
				Value: prop,
				Type:  d.getValueType(prop),
			}
			
			// For nested objects, create a reference
			if nestedObj, ok := prop.(*Object); ok {
				ref := d.nextVarRef
				d.nextVarRef++
				d.variableRefs[ref] = nestedObj
				variable.Ref = ref
			}
			
			variables = append(variables, variable)
		}
	}

	return variables
}

// getValueType returns a string representation of the value's type
func (d *Debugger) getValueType(v Value) string {
	if v == nil {
		return "undefined"
	}
	
	switch val := v.(type) {
	case valueNull:
		return "null"
	case valueUndefined:
		return "undefined"
	case valueInt, valueFloat, *valueBigInt:
		return "number"
	case String:
		return "string"
	case valueBool:
		return "boolean"
	case *Object:
		// Check for specific object types
		className := val.self.className()
		switch className {
		case "Array":
			return "array"
		case "Function":
			return "function"
		case "Date":
			return "date"
		case "RegExp":
			return "regexp"
		case "Error":
			return "error"
		default:
			return "object"
		}
	default:
		return "unknown"
	}
}

// Evaluate evaluates an expression in the context of a frame
func (d *Debugger) Evaluate(expression string, frameID int) (Value, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// This is a simplified implementation
	// A full implementation would need to:
	// 1. Parse the expression
	// 2. Evaluate it in the context of the specified frame
	// 3. Handle errors appropriately
	
	// For now, we'll use the runtime's RunString in the global context
	// In the future, this should evaluate in the frame's context
	return d.runtime.RunString(expression)
}

// EvaluateInFrame evaluates an expression in the context of a specific stack frame
func (d *Debugger) EvaluateInFrame(expression string, frameIndex int) (Value, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.runtime.vm == nil {
		return nil, fmt.Errorf("no active execution context")
	}

	// Get the current call stack
	stack := d.runtime.CaptureCallStack(0, nil)
	if frameIndex < 0 || frameIndex >= len(stack) {
		return nil, fmt.Errorf("invalid frame index: %d", frameIndex)
	}

	frame := &stack[frameIndex]
	
	// If the frame has no context, evaluate in global scope
	if frame.ctx == nil || frame.ctx.stash == nil {
		return d.runtime.RunString(expression)
	}

	// Create a temporary evaluation context with the frame's variables
	// This is a simplified implementation that creates a new scope with the frame's variables
	evalCode := "(function() {\n"
	
	// Inject local variables from the frame's stash
	if frame.ctx.stash.names != nil {
		for name, idx := range frame.ctx.stash.names {
			if int(idx) < len(frame.ctx.stash.values) && frame.ctx.stash.values[idx] != nil {
				// Create a variable declaration for each local variable
				evalCode += fmt.Sprintf("var %s = this._%s;\n", name.String(), name.String())
			}
		}
	}
	
	// Add the expression to evaluate
	evalCode += "return (" + expression + ");\n"
	evalCode += "}).call(this)"
	
	// Create an object with the frame's variables
	varsObj := d.runtime.NewObject()
	if frame.ctx.stash.names != nil {
		for name, idx := range frame.ctx.stash.names {
			if int(idx) < len(frame.ctx.stash.values) && frame.ctx.stash.values[idx] != nil {
				varsObj.Set("_"+name.String(), frame.ctx.stash.values[idx])
			}
		}
	}
	
	// Save current global state
	savedGlobal := d.runtime.globalObject
	
	// Temporarily set the variables object as 'this'
	d.runtime.globalObject = varsObj
	
	// Evaluate the expression
	result, err := d.runtime.RunString(evalCode)
	
	// Restore global state
	d.runtime.globalObject = savedGlobal
	
	return result, err
}

// buildDebugStack builds the debug stack with variable information
func (d *Debugger) buildDebugStack(vm *vm) []DebugStackFrame {
	if vm == nil {
		return nil
	}

	// Get the regular call stack first
	callStack := d.runtime.CaptureCallStack(0, nil)
	debugStack := make([]DebugStackFrame, len(callStack))

	// Create scope references that will be populated lazily
	for i, frame := range callStack {
		debugFrame := DebugStackFrame{
			StackFrame: frame,
		}

		// Build scopes for this frame
		// Use negative refs for lazy loading - will be resolved in GetScopes
		scopes := []Scope{
			{
				Name:         "Local",
				VariablesRef: -(i*10 + 1), // Negative to indicate it needs resolution
				Expensive:    false,
			},
		}

		// Only add global scope for the first frame to avoid duplication
		if i == 0 {
			scopes = append(scopes, Scope{
				Name:         "Global",
				VariablesRef: -(i*10 + 2), // Negative to indicate it needs resolution
				Expensive:    true,
			})
		}

		debugFrame.Scopes = scopes

		// Get 'this' value for the frame
		// For now, we'll leave this as nil
		// TODO: Extract 'this' value from the context

		debugStack[i] = debugFrame
	}

	return debugStack
}

// resolveLazyScope resolves a lazy scope reference and returns its variables
func (d *Debugger) resolveLazyScope(lazyRef int) []Variable {
	// Extract frame ID and scope type from the negative reference
	// lazyRef = -(frameID*10 + scopeID)
	// scopeID: 1 = local, 2 = global
	absRef := -lazyRef
	frameID := absRef / 10
	scopeID := absRef % 10
	
	var scopeType string
	switch scopeID {
	case 1:
		scopeType = "local"
	case 2:
		scopeType = "global"
	default:
		return nil
	}
	
	return d.getVariablesForScope(frameID, scopeType)
}
