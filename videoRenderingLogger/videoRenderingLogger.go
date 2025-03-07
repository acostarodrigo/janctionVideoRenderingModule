package videoRenderingLogger

import (
	"fmt"
	"log"
	"os"
)

// ANSI escape codes for colors
const (
	colorReset = "\033[0m"
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
	colorBlue  = "\033[34m"
)

// Log levels
const (
	LevelInfo  = 1
	LevelDebug = 2
	LevelError = 3
)

// VideoRenderingLogger defines a custom logger for the module
type VideoRenderingLogger struct {
	logger   *log.Logger
	logLevel int
}

// NewVideoRenderingLogger creates a new instance of the logger with a specified log level
func NewVideoRenderingLogger(level int) *VideoRenderingLogger {
	return &VideoRenderingLogger{
		logger:   log.New(os.Stdout, colorRed+"[VideoRendering] ", log.LstdFlags),
		logLevel: level,
	}
}

// GlobalLogger provides a globally accessible logger instance with default level INFO
var Logger = NewVideoRenderingLogger(LevelInfo)

// Info logs informational messages (Green) if log level allows
func (v *VideoRenderingLogger) Info(format string, args ...interface{}) {
	if v.logLevel <= LevelInfo {
		v.logger.Println(colorGreen + "INFO: " + fmt.Sprintf(format, args...) + colorReset)
	}
}

// Error logs error messages (Red) if log level allows
func (v *VideoRenderingLogger) Error(format string, args ...interface{}) {
	if v.logLevel <= LevelError {
		v.logger.Println(colorRed + "ERROR: " + fmt.Sprintf(format, args...) + colorReset)
	}
}

// Debug logs debug messages (Blue) if log level allows
func (v *VideoRenderingLogger) Debug(format string, args ...interface{}) {
	if v.logLevel <= LevelDebug {
		v.logger.Println(colorBlue + "DEBUG: " + fmt.Sprintf(format, args...) + colorReset)
	}
}
