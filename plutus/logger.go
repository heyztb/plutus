package plutus

import (
	"os"
	"strings"

	"github.com/mbndr/logo"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger is a reference to our logger
var Logger *logo.Logger

func init() {
	lumberjackHandler := &lumberjack.Logger{
		Filename:   "pg.log",
		MaxSize:    200, // mb
		MaxBackups: 1,
		MaxAge:     14, // days
	}

	fileLog := logo.NewReceiver(lumberjackHandler, "PAYGATE")
	fileLog.Level = logo.INFO

	stdOut := logo.NewReceiver(os.Stdout, "PAYGATE")
	stdOut.Color = true
	if strings.ToLower(os.Getenv("DEBUG")) == "true" {
		stdOut.Level = logo.INFO
	} else {
		stdOut.Level = logo.FATAL
	}

	stdErr := logo.NewReceiver(os.Stderr, "PAYGATE")
	stdErr.Color = true
	stdErr.Level = logo.WARN

	Logger = logo.NewLogger(fileLog, stdOut, stdErr)
}
