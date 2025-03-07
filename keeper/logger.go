package keeper

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

// VideoRenderingLogger defines a custom logger for the module
type VideoRenderingLogger struct {
	logger *log.Logger
}

// NewVideoRenderingLogger creates a new instance of the logger
func NewVideoRenderingLogger() *VideoRenderingLogger {
	return &VideoRenderingLogger{
		logger: log.New(os.Stdout, colorRed+"[VideoRendering] ", log.LstdFlags),
	}
}

// Info logs informational messages (Green)
func (v *VideoRenderingLogger) Info(format string, args ...interface{}) {
	v.logger.Println(colorGreen + "INFO: " + fmt.Sprintf(format, args...) + colorReset)
}

// Error logs error messages (Red)
func (v *VideoRenderingLogger) Error(format string, args ...interface{}) {
	v.logger.Println(colorRed + "ERROR: " + fmt.Sprintf(format, args...) + colorReset)
}

// Debug logs debug messages (Blue)
func (v *VideoRenderingLogger) Debug(format string, args ...interface{}) {
	v.logger.Println(colorBlue + "DEBUG: " + fmt.Sprintf(format, args...) + colorReset)
}
