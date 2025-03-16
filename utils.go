package videoRendering

import (
	"crypto/sha256"
	"encoding/hex"
	fmt "fmt"
	io "io"
	"os"
	"os/exec"
	"path/filepath"
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

func FromCliToFrames(entries []string) map[string]VideoRenderingThread_Frame {
	result := make(map[string]VideoRenderingThread_Frame)

	for _, entry := range entries {
		parts := strings.Split(entry, "=")
		if len(parts) != 2 {
			fmt.Println("Invalid entry:", entry)
			continue
		}

		filename := parts[0]
		cidAndHash := strings.Split(parts[1], ":")
		if len(cidAndHash) != 2 {
			fmt.Println("Invalid CID:Hash format:", parts[1])
			continue
		}
		frame := VideoRenderingThread_Frame{Filename: filename, Cid: cidAndHash[0], Hash: cidAndHash[1]}
		result[filename] = frame
	}

	return result
}

func FromFramesToCli(frames map[string]VideoRenderingThread_Frame) []string {
	var result []string

	for filename, frame := range frames {
		entry := fmt.Sprintf("%s=%s:%s", filename, frame.Cid, frame.Hash)
		result = append(result, entry)
	}

	return result
}

// CalculateFileHash calculates the SHA-256 hash of a given file.
func CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// GenerateDirectoryFileHashes walks through a directory and computes SHA-256 hashes for all files.
func GenerateDirectoryFileHashes(dirPath string) (map[string]string, error) {
	hashes := make(map[string]string)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Compute file hash
		hash, err := CalculateFileHash(path)
		if err != nil {
			return err
		}

		// Store hash with filename (relative path)
		relPath, _ := filepath.Rel(dirPath, path)
		hashes[relPath] = hash

		return nil
	})

	if err != nil {
		return nil, err
	}

	return hashes, nil
}
