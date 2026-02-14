package main

import _ "embed"

//go:embed instructions.md
var defaultInstructions string

// CompletionSignal is the marker the agent outputs when all tasks are complete
const CompletionSignal = "<promise>COMPLETE</promise>"
