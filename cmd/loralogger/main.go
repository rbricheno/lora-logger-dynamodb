package main

import "github.com/rbricheno/lora-logger-dynamodb/cmd/loralogger/cmd"

var version string // set by the compiler

func main() {
	cmd.Execute(version)
}
