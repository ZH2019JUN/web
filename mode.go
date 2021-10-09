package gin

import (
	"io"
	"os"
)

const EnvGinMode = "GIN_MODE"

var DefaultWriter io.Writer = os.Stdout

// DefaultErrorWriter is the default io.Writer used by Gin to debug errors
var DefaultErrorWriter io.Writer = os.Stderr


const (
	DebugMode = "debug"
	ReleaseMode = "release"
	TestMode = "test"
)

const (
	debugCode = iota
	releaseCode
	testCode
)

var ginMode = debugCode
var modeName = DebugMode