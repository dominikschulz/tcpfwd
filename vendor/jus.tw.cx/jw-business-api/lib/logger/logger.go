package logger

import "github.com/go-kit/kit/log"

// Logger is the global gokit logger for this project
var Logger = log.NewNopLogger()

// Log is a shorthand for Logger.Log
func Log(v ...interface{}) error { return Logger.Log(v...) }

// Error logs the error message with error level
func Error(err error) {
	Log("level", "error", "error", err.Error())
}
