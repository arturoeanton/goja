module github.com/arturoeanton/goja/examples/debugger/goja-debug

go 1.23.0

toolchain go1.24.5

replace github.com/dop251/goja => ../../..

require (
	github.com/dop251/goja v0.0.0-00010101000000-000000000000
	github.com/fatih/color v1.16.0
	github.com/peterh/liner v1.2.2
	golang.org/x/term v0.33.0
)

require (
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.3 // indirect
	github.com/peterh/liner v1.2.2 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.3.8 // indirect
)
