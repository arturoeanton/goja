package goja

import (
	"fmt"
	"log"
	"os"
	"strings"
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
	Name             string // "Local", "Closure", "Global"
	VariablesRef     int    // Reference ID to get variables
	Expensive        bool   // If true, variables should be retrieved lazily
	NamedVariables   int    // Number of named variables
	IndexedVariables int    // Number of indexed variables (for arrays)
}

// DebugStackFrame extends StackFrame with debug information
type DebugStackFrame struct {
	StackFrame
	Scopes []Scope
	This   *Variable // The 'this' value in the context
}

// DebuggerState represents the current state when paused
type DebuggerState struct {
	PC                 int
	SourcePos          Position
	CallStack          []StackFrame
	DebugStack         []DebugStackFrame // Extended stack frames with variable info
	Breakpoint         *Breakpoint       // Current breakpoint if stopped at one
	StepMode           bool
	InNativeCall       bool   // True when executing native function
	NativeFunctionName string // Name of the native function being executed
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
	lastPC        int                 // Previous PC for step-over flow control
	lastSourceLine int                // Previous source line for step-over flow control
	continueStepAfterCall bool         // Continue stepping after a call instruction

	// Variable reference management for DAP
	variableRefs map[int]interface{} // Maps reference IDs to values or scopes
	nextVarRef   int
	
	// Logger for debugging
	logger *log.Logger
}

// NewDebugger creates a new debugger for the runtime
func (r *Runtime) NewDebugger() *Debugger {
	// Initialize logger
	var logger *log.Logger
	logFile, err := os.OpenFile("goja.debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		logger = log.New(logFile, "[GOJA_CORE] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	} else {
		logger = log.New(os.Stderr, "[GOJA_CORE] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	}
	
	d := &Debugger{
		runtime:       r,
		breakpoints:   make(map[int]*Breakpoint),
		pcBreakpoints: make(map[int]*Breakpoint),
		variableRefs:  make(map[int]interface{}),
		nextVarRef:    1000, // Start from 1000 to avoid conflicts with frame IDs
		logger:        logger,
	}
	
	d.logger.Println("=== Debugger created ===")
	return d
}

// SetHandler sets the debug handler that will be called when execution pauses
func (d *Debugger) SetHandler(handler DebugHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handler = handler
	d.logger.Println("SetHandler: Debug handler set")
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
	d.logger.Printf("AddBreakpoint: Added breakpoint #%d at %s:%d:%d\n", bp.id, filename, line, column)

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
		d.logger.Printf("SetStepMode: Enabled with mode=%v\n", d.stepMode)
	} else {
		d.flags &^= FlagStepMode
		d.logger.Println("SetStepMode: Disabled")
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
	d.lastPC = -1 // Reset lastPC to allow first step
	d.lastSourceLine = -1 // Reset lastSourceLine to allow first step
	d.logger.Printf("StepOver: flags=%b, stepMode=%v, stepDepth=%d\n", d.flags, d.stepMode, d.stepDepth)
}

// StepInto executes the next line, stepping into function calls
func (d *Debugger) StepInto() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.flags |= FlagStepMode
	d.flags &^= FlagPaused
	d.stepMode = DebugStepInto
	d.logger.Printf("StepInto: flags=%b, stepMode=%v\n", d.flags, d.stepMode)
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
		d.logger.Printf("resolveBreakpoint: Cannot resolve BP #%d - no program loaded\n", bp.id)
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
					d.logger.Printf("resolveBreakpoint: Resolved BP #%d to PC=%d\n", bp.id, pc)
					break
				}
			}
		}
	}
	if bp.pc < 0 {
		d.logger.Printf("resolveBreakpoint: Failed to resolve BP #%d at %s:%d\n", bp.id, bp.SourcePos.Filename, bp.SourcePos.Line)
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

	// For native functions, we still want to track step events
	// but the handler can decide whether to process them
	if vm.prg == nil {
		// Check if already paused or in step mode
		shouldPause := d.flags&FlagPaused != 0 || d.flags&FlagStepMode != 0
		if shouldPause {
			d.logger.Printf("checkBreakpoint: Native function, flags=%b, pausing=%v\n", d.flags, shouldPause)
		}
		return shouldPause
	}
	
	// Special handling for step-into with call instructions
	if d.flags&FlagStepMode != 0 && d.stepMode == DebugStepInto {
		// Check if the current instruction is a call
		if vm.pc >= 0 && vm.pc < len(vm.prg.code) {
			// Log the instruction type for debugging
			instr := vm.prg.code[vm.pc]
			d.logger.Printf("checkBreakpoint: StepInto - instruction type: %T\n", instr)
			
			if _, isCall := instr.(call); isCall {
				// We're about to execute a call instruction
				// Don't pause now, let it execute and pause at the first instruction of the called function
				d.logger.Printf("checkBreakpoint: StepInto on call instruction, deferring pause\n")
				// Mark that we should continue stepping after the call
				d.continueStepAfterCall = true
				// Keep step mode active but don't pause yet
				return false
			}
		}
	}

	// Log current state
	currentLine := -1
	if vm.prg.srcMap != nil && vm.pc < len(vm.prg.srcMap) {
		item := vm.prg.srcMap[vm.pc]
		if item.srcPos >= 0 && vm.prg.src != nil {
			pos := vm.prg.src.Position(item.srcPos)
			currentLine = pos.Line
		}
	}
	d.logger.Printf("checkBreakpoint: PC=%d, Line=%d, flags=%b, stepMode=%v, callStackLen=%d, stepDepth=%d\n", 
		vm.pc, currentLine, d.flags, d.stepMode, len(vm.callStack), d.stepDepth)

	// Check if paused
	if d.flags&FlagPaused != 0 {
		d.logger.Println("checkBreakpoint: Already paused, returning true")
		return true
	}

	// Check breakpoints
	if bp, exists := d.pcBreakpoints[vm.pc]; exists && bp.enabled {
		bp.hit++
		d.flags |= FlagPaused
		d.logger.Printf("checkBreakpoint: Hit breakpoint #%d at PC=%d, hits=%d\n", bp.id, vm.pc, bp.hit)
		return true
	}

	// Check step mode
	if d.flags&FlagStepMode != 0 {
		switch d.stepMode {
		case DebugStepInto:
			d.flags |= FlagPaused
			d.logger.Printf("checkBreakpoint: StepInto - pausing at PC=%d, Line=%d\n", vm.pc, currentLine)
			return true
		case DebugStepOver:
			if len(vm.callStack) <= d.stepDepth {
				// Get current source line
				currentLine := -1
				if vm.prg != nil && vm.prg.srcMap != nil && vm.pc < len(vm.prg.srcMap) {
					item := vm.prg.srcMap[vm.pc]
					if item.srcPos >= 0 && vm.prg.src != nil {
						pos := vm.prg.src.Position(item.srcPos)
						currentLine = pos.Line
					}
				}
				
				// Check if we've moved to a different source line
				isFirstStep := d.lastSourceLine == -1
				isDifferentLine := currentLine > 0 && currentLine != d.lastSourceLine
				isReturning := len(vm.callStack) < d.stepDepth
				
				// Special handling: if we had a valid line before and now have an invalid line,
				// continue stepping (we're probably in a transition state)
				wasValidLine := d.lastSourceLine > 0
				isInvalidLine := currentLine <= 0
				
				d.logger.Printf("checkBreakpoint: StepOver - currentLine=%d, lastLine=%d, firstStep=%v, diffLine=%v, returning=%v\n",
					currentLine, d.lastSourceLine, isFirstStep, isDifferentLine, isReturning)
				
				if isFirstStep || isDifferentLine || isReturning {
					// Only pause if we have a valid source position
					if currentLine > 0 || isReturning {
						d.flags |= FlagPaused
						d.logger.Printf("checkBreakpoint: StepOver - pausing (firstStep=%v, diffLine=%v, returning=%v)\n",
							isFirstStep, isDifferentLine, isReturning)
						return true
					}
				}
				
				// If we were at a valid line and now at invalid, keep stepping
				if wasValidLine && isInvalidLine {
					// Don't pause, keep stepping
					d.logger.Println("checkBreakpoint: StepOver - transitioning from valid to invalid line, continuing")
					return false
				}
			} else {
				d.logger.Printf("checkBreakpoint: StepOver - callStack depth %d > stepDepth %d, continuing\n",
					len(vm.callStack), d.stepDepth)
			}
		case DebugStepOut:
			if len(vm.callStack) < d.stepDepth {
				d.flags |= FlagPaused
				d.logger.Printf("checkBreakpoint: StepOut - pausing, callStack=%d < stepDepth=%d\n", 
					len(vm.callStack), d.stepDepth)
				return true
			}
		}
	}

	return false
}

// handlePause is called when the VM pauses
func (d *Debugger) handlePause(vm *vm) {
	// No need to skip native functions anymore - the handler can decide
	d.logger.Printf("handlePause: Called, PC=%d, prg=%v\n", vm.pc, vm.prg != nil)

	d.mu.RLock()
	handler := d.handler
	d.mu.RUnlock()

	if handler == nil {
		// No handler, just continue
		d.logger.Println("handlePause: No handler set, continuing")
		d.Continue()
		return
	}

	// Build debug state
	// Check if we're in a native function call
	isNative := vm.prg == nil
	nativeName := ""
	if isNative {
		// We're in a native function - try to get its name
		if vm.sb > 0 && vm.sb-1 < len(vm.stack) {
			if fn, ok := vm.stack[vm.sb-1].(*Object); ok && fn != nil && fn.self != nil {
				if name := fn.self.getStr("name", nil); name != nil {
					nameStr := name.String()
					// Clean up Go function names
					if idx := strings.LastIndex(nameStr, "."); idx > 0 && idx < len(nameStr)-1 {
						// Extract just the function name part for anonymous functions
						if strings.Contains(nameStr, "func") {
							// For anonymous functions, try to get a more meaningful name
							nativeName = "native"
						} else {
							nativeName = nameStr[idx+1:]
						}
					} else {
						nativeName = nameStr
					}
				} else {
					nativeName = "<native>"
				}
			} else {
				nativeName = "<native>"
			}
		} else {
			nativeName = "<native>"
		}
	}

	state := &DebuggerState{
		PC:                 vm.pc,
		StepMode:           d.flags&FlagStepMode != 0,
		InNativeCall:       isNative,
		NativeFunctionName: nativeName,
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
	d.logger.Printf("handlePause: Calling handler at Line=%d, PC=%d, InNative=%v, NativeName=%s\n",
		state.SourcePos.Line, state.PC, state.InNativeCall, state.NativeFunctionName)
	
	// Update lastSourceLine before calling handler
	if state.SourcePos.Line > 0 {
		d.mu.Lock()
		d.lastSourceLine = state.SourcePos.Line
		d.mu.Unlock()
	}
	
	cmd := handler(state)
	d.logger.Printf("handlePause: Handler returned command: %v\n", cmd)
	
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
		d.logger.Println("handlePause: Already paused, no action")
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
	
	if d.runtime.vm == nil {
		d.mu.RUnlock()
		return nil, fmt.Errorf("no active execution context")
	}

	// Get the current call stack
	stack := d.runtime.CaptureCallStack(0, nil)
	if frameIndex < 0 || frameIndex >= len(stack) {
		d.mu.RUnlock()
		return nil, fmt.Errorf("invalid frame index: %d", frameIndex)
	}

	frame := &stack[frameIndex]

	// If the frame has no context, evaluate in global scope
	if frame.ctx == nil || frame.ctx.stash == nil {
		d.mu.RUnlock()
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

	// Release lock before evaluating
	d.mu.RUnlock()

	// Evaluate the expression
	result, err := d.runtime.RunString(evalCode)

	// Restore global state
	d.runtime.globalObject = savedGlobal

	return result, err
}

// IsInNativeCall returns true if currently executing native code
func (d *Debugger) IsInNativeCall() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Check if VM is currently in native code
	if d.runtime != nil && d.runtime.vm != nil {
		return d.runtime.vm.prg == nil
	}

	return false
}

// GetNativeFunctionName returns the name of the currently executing native function
func (d *Debugger) GetNativeFunctionName() string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.runtime == nil || d.runtime.vm == nil || d.runtime.vm.prg != nil {
		return ""
	}

	// Try to get the function name from the call stack
	if len(d.runtime.vm.callStack) > 0 && d.runtime.vm.sb > 0 {
		// Look for the function being called
		stack := d.runtime.vm.stack
		sb := d.runtime.vm.sb
		if sb-1 < len(stack) && sb-1 >= 0 {
			if fn, ok := stack[sb-1].(*Object); ok && fn != nil && fn.self != nil {
				if name := fn.self.getStr("name", nil); name != nil {
					return name.String()
				}
			}
		}
	}

	return "<native>"
}

// ShouldStepInNativeCall returns true if debugger should process step events in native calls
// This is useful for DAP implementations to decide whether to show stepping in native functions
// By default, returns false to avoid confusing multiple events in native functions
func (d *Debugger) ShouldStepInNativeCall() bool {
	// For now, we skip stepping in native calls to avoid confusion
	// DAP implementations can override this behavior based on user preferences
	return false
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
