package log

import (
	"fmt"
	"log"
	"os"
)

// Fatal logs a message of level 'fatal'.
func Fatal(format string, args ...interface{}) {
	Log("fatal", format, args...)
	os.Exit(1)
}

// Debug logs a message of level 'debug', and then exit using os.Exit(1)
func Debug(format string, args ...interface{}) {
	Log("debug", format, args...)
}

// Warning logs a message of level 'warning'.
func Warning(format string, args ...interface{}) {
	Log("warning", format, args...)
}

// Error logs a message of level 'info'.
func Error(format string, args ...interface{}) {
	Log("error", format, args...)
}

// Info logs a message of level 'info'.
func Info(format string, args ...interface{}) {
	Log("info", format, args...)
}

// Log a message (level and content) to stderr and to the client's mqtt server.
// Level is typically info/warning et cetera. For fatal log messages, use MQTTLogger.Fatal
func Log(level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[%s] %s", level, msg)
}
