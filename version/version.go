package version

import (
	"fmt"
	"runtime"
)

const Version = "0.1.0"

var GitCommit string
var BuildDate = ""
var GoVersion = runtime.Version()
var OsArch = fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH)
