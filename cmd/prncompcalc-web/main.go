//go:build js && wasm

// Command prncompcalc-web is the WebAssembly side of the web page. It exposes
// the calculation to JavaScript and then waits, because the functions stop
// working if the Go program exits.
package main

import (
	"syscall/js"

	"github.com/rmkhl/prncompcalc/internal/version"
	"github.com/rmkhl/prncompcalc/internal/webapi"
)

func calculate(_ js.Value, args []js.Value) any {
	response, err := webapi.CalculateJSON(args[0].String())
	if err != nil {
		return js.Global().Get("Error").New(err.Error())
	}
	return response
}

func main() {
	js.Global().Set("prncompcalcCalculate", js.FuncOf(calculate))
	js.Global().Set("prncompcalcVersion", version.Version)
	select {}
}
