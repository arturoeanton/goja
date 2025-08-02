# Goja Debug Console

A comprehensive interactive debugger console for Goja that demonstrates all debugging capabilities.

## Features

### 🎨 Adaptive Two-Column Layout
- **Responsive Design**: Automatically adapts to terminal size
- **Code Panel** (60% width): Shows source code with line numbers and breakpoints
- **Variables Panel** (40% width): Shows local/global variables and call stack
- **Console Output**: Dedicated panel for console.log output
- **Compact Commands**: Two-column command reference at bottom

### 🔍 Variable Inspection
- **Local Variables**: View all variables in the current scope
- **Global Variables**: Inspect global scope variables
- **Closure Variables**: See variables captured in closures
- **Complex Types**: Navigate through objects and arrays with reference IDs
- **This Context**: View the current `this` value

### 🎯 Breakpoint Management
- Set breakpoints at any line: `b <line>`
- Delete breakpoints: `d <line>`
- List all breakpoints: `lb`
- Visual indicators in source view (● for breakpoints)

### 🚶 Execution Control
- **Continue** (`c`): Resume normal execution
- **Step Over** (`n`): Execute current line, skip function calls
- **Step Into** (`s`): Step into function calls (see limitations below)
- **Step Out** (`o`): Complete current function and return

#### ⚠️ Step-Into Limitation with Closures
Step-into does not work with closures (functions stored in variables). For example:
```javascript
let myFunc = createClosure();
myFunc(); // Step-into won't enter the closure
```
**Workaround**: Set breakpoints inside closure functions. See the included `test_closure.js` for examples.

### 💻 REPL During Debug
- **Interactive REPL**: Type any JavaScript expression directly (no command prefix needed)
- **Explicit evaluation**: Use `p <expression>` command for explicit evaluation
- **Access variables**: Read and modify variables in current scope
- **Call functions**: Execute functions and inspect results
- **Object exploration**: Navigate complex objects with formatted output
- **Full JavaScript support**: All JavaScript expressions and statements work

### 🔧 Native Function Support
- Detects when executing native functions
- Shows native function name in UI
- Properly handles stepping through native calls

### 🎨 Intuitive Console UI
- Color-coded interface for better readability
- Current line highlighting with arrow (→)
- Source code context (±5 lines)
- Call stack visualization
- Clear command help
- **Two-column console output**: Program output | Debug messages
  - Left column: console.log() and program output
  - Right column: Debug info, errors, and system messages

## Installation

```bash
cd examples/debugger/goja-debug
go mod download
go build
```

## Usage

```bash
./goja-debug <script.js>
```

Or with the test script:

```bash
./goja-debug test_debug.js
```

## Commands

| Command | Aliases | Description |
|---------|---------|-------------|
| `c` | `continue` | Continue execution |
| `n` | `next`, `over` | Step over (next line) |
| `s` | `step`, `into` | Step into function |
| `o` | `out` | Step out of function |
| `b <line>` | | Set breakpoint at line |
| `d <line>` | | Delete breakpoint at line |
| `lb` | | List all breakpoints |
| `p <expr>` | | Evaluate expression |
| `v <ref>` | | Inspect variable by reference |
| `locals` | | Show all local variables |
| `globals` | | Show global variables |
| `l` | `L` | Switch to local variables display |
| `g` | `G` | Switch to global variables display |
| `st` | `stack` | Toggle stack trace display |
| `h` | `help` | Show command help |
| `q` | `quit` | Exit debugger |
| `<expression>` | | REPL mode: evaluate any JS expression |

## New UI Layout

```
╔═════════════════════ GOJA DEBUG - demo_ui.js ══════════════════════╗
┌─────────── Source Code ───────────┐┌───────── Variables ─────────┐
│   →  12 function processUser(user) ││─ Local ─                    │
│      13     console.log("Processing││user: {...} #1001            │
│      14                            ││fullName: "John Doe"         │
│      15     var fullName = user.fi││age: undefined               │
│      16     var age = calculateAge││status: undefined            │
│      17     var status = age >= 18││                             │
│      18                            ││─ Call Stack ─               │
│      19     console.log("Full name││→ processUser                │
│      20     console.log("Age:", age││  <main>                     │
└────────────────────────────────────┘└─────────────────────────────┘
┌─────────────────── Console Output ──────────────────────────────────┐
│console.log: === Starting Goja Debug UI Demo v 2 ===                 │
│console.log: Config: {"theme":"dark","layout":"two-column"...}       │
│console.log: Processing user: johndoe                                │
└──────────────────────────────────────────────────────────────────────┘
─── Commands ───
c/continue   Continue          b <line>     Set breakpoint
n/next       Step over         d <line>     Delete breakpoint  
s/step       Step into         p <expr>     Evaluate
o/out        Step out          sl           Show locals
h/help       Help              sg           Show globals
REPL: Type any JavaScript expression when paused

debug> b 3
Breakpoint 1 set at line 3
debug> c

[... paused at breakpoint ...]

debug> n
[... execution continues ...]

debug> result
12
debug> n * 3
30
debug> {test: "value", arr: [1,2,3]}
{
  test: "value",
  arr: [1, 2, 3]
}
debug> locals
Local variables:
  n: 10
  result: 20
```

## Testing

Run the unit tests:

```bash
go test -v
```

The tests cover:
- Debugger console creation
- File loading
- Breakpoint management
- Variable printing
- Command processing
- Integration with Goja runtime
- Native function detection
- Variable scope handling