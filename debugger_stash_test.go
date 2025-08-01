package goja

import (
	"testing"
)

func TestDebuggerStashContent(t *testing.T) {
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

	r := New()
	debugger := r.EnableDebugger()
	
	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		vm := r.vm
		t.Logf("=== VM State ===")
		t.Logf("Current stash: %p", vm.stash)
		if vm.stash != nil {
			t.Logf("Stash names: %v", vm.stash.names)
			t.Logf("Stash values: %v", vm.stash.values)
			
			// Try manual extraction
			if vm.stash.names != nil {
				for name, idx := range vm.stash.names {
					if int(idx) < len(vm.stash.values) {
						t.Logf("Variable %s (idx %d) = %v", name.String(), idx, vm.stash.values[idx])
					}
				}
			}
		}
		
		// Check the stack
		t.Logf("\n=== Stack State ===")
		t.Logf("Stack pointer (sp): %d", vm.sp)
		t.Logf("Stack base (sb): %d", vm.sb)
		t.Logf("Args: %d", vm.args)
		
		// Print some stack values around sb
		if vm.sb >= 0 && vm.sb < len(vm.stack) {
			start := vm.sb - 5
			if start < 0 {
				start = 0
			}
			end := vm.sb + 10
			if end > len(vm.stack) {
				end = len(vm.stack)
			}
			for i := start; i < end; i++ {
				marker := ""
				if i == vm.sb {
					marker = " <-- sb"
				}
				if i == vm.sp {
					marker += " <-- sp"
				}
				t.Logf("Stack[%d]: %v%s", i, vm.stack[i], marker)
			}
		}
		
		// Also check the call stack
		t.Logf("\n=== Call Stack ===")
		for i, ctx := range vm.callStack {
			t.Logf("CallStack[%d]: prg=%v, stash=%p, sb=%d, args=%d", i, ctx.prg != nil, ctx.stash, ctx.sb, ctx.args)
			if ctx.stash != nil {
				t.Logf("  Stash names: %v", ctx.stash.names)
				t.Logf("  Stash values: %v", ctx.stash.values)
				if ctx.stash.names != nil {
					for name, idx := range ctx.stash.names {
						if int(idx) < len(ctx.stash.values) {
							t.Logf("  Variable %s (idx %d) = %v", name.String(), idx, ctx.stash.values[idx])
						}
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
}