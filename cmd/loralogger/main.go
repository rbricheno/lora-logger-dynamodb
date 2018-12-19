package main

import "github.com/rbricheno/lora-logger/cmd/loralogger/cmd"

var version string // set by the compiler

func main() {
	cmd.Execute(version)
}
