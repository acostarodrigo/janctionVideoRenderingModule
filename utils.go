package videoRendering

import (
	fmt "fmt"
	"os/exec"
	"strings"

	"github.com/janction/videoRendering/videoRenderingLogger"
)

// Transforms a slice with format [key]=[value] to a map
func TransformSliceToMap(input []string) (map[string]string, error) {
	result := make(map[string]string)

	for _, item := range input {
		parts := strings.SplitN(item, "=", 2) // Split into 2 parts: filename and hash
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format: %s", item)
		}
		filename := parts[0]
		hash := parts[1]
		result[filename] = hash
	}

	return result, nil
}

// MapToKeyValueFormat converts a map[string]string to a "key=value,key=value" format
func MapToKeyValueFormat(inputMap map[string]string) []string {
	var parts []string

	// Iterate through the map and build the key=value pairs
	for key, value := range inputMap {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}

	// Join the key=value pairs with commas
	return parts
}

// Executes a cli command with their arguments
func ExecuteCli(args []string) error {
	executableName := "janctiond"
	cmd := exec.Command(executableName, args...)
	videoRenderingLogger.Logger.Info("Executing %s", cmd.String())

	_, err := cmd.Output()

	if err != nil {
		videoRenderingLogger.Logger.Error("Error Executing CLI %s: %s", cmd.String(), err.Error())
		return err
	}

	return nil
}
