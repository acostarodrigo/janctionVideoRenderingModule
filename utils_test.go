package videoRendering

import (
	fmt "fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
)

// --- Test for SliceToMap ---
func TestTransformSliceToMap_Success(t *testing.T) {
	input := []string{
		"file1.txt=hash1",
		"file2.txt=hash2",
		"file3.txt=hash3",
	}

	expected := map[string]string{
		"file1.txt": "hash1",
		"file2.txt": "hash2",
		"file3.txt": "hash3",
	}

	result, err := TransformSliceToMap(input)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTransformSliceToMap_InvalidFormat(t *testing.T) {
	input := []string{
		"file1.txt=hash1",
		"invalidfile", // Invalid entry
		"file2.txt=hash2",
	}

	result, err := TransformSliceToMap(input)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid format: invalidfile")
}

// --- Test for MapToKeyValueFormat ---
func TestMapToKeyValueFormat(t *testing.T) {
	testMap := map[string]string{
		"Key1": "Value1",
		"Key2": "Value2",
		"":     "Value3",
		"Key4": "",
	}
	expected := []string{
		"Key1=Value1",
		"Key2=Value2",
		"=Value3",
		"Key4=",
	}

	result := MapToKeyValueFormat(testMap)
	if len(result) != len(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
	sort.Strings(result)
	sort.Strings(expected)
	for i := range result {
		if result[i] != expected[i] {
			t.Errorf("At index %d, expected %v, got %v", i, expected[i], result[i])
		}
	}
}

// --- Test for ExecuteCli ---
func TestExecuteCli(t *testing.T) {
	tests := []struct {
		name       string
		mockOutput func(cmd *exec.Cmd) ([]byte, error)
		expectErr  bool
	}{
		{
			name: "Successful CLI execution",
			mockOutput: func(cmd *exec.Cmd) ([]byte, error) {
				return []byte("simulated output"), nil
			},
			expectErr: false,
		},
		{
			name: "CLI execution error",
			mockOutput: func(cmd *exec.Cmd) ([]byte, error) {
				return nil, fmt.Errorf("simulated error")
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Monkey patch exec.Command to return a minimal valid *exec.Cmd
			patchCmd := monkey.Patch(exec.Command, func(name string, args ...string) *exec.Cmd {
				return &exec.Cmd{
					Path: name,
					Args: append([]string{name}, args...),
				}
			})
			defer patchCmd.Unpatch()

			// Monkey patch the Output method of *exec.Cmd to simulate CLI output or error
			patchOutput := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", tt.mockOutput)
			defer patchOutput.Unpatch()

			// Execute the CLI function
			err := ExecuteCli([]string{"test"})

			// Assert based on expected error
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err != nil)
			}
		})
	}
}

// --- Test for FromCliToFrames ---
func TestFromCliToFrames(t *testing.T) {
	tests := []struct {
		name     string
		entries  []string
		expected map[string]VideoRenderingThread_Frame
	}{
		{
			name:    "Valid entry",
			entries: []string{"file1.png=cid123:hash123"},
			expected: map[string]VideoRenderingThread_Frame{
				"file1.png": {Filename: "file1.png", Cid: "cid123", Hash: "hash123"},
			},
		},
		{
			name:     "Invalid entry (missing '=')",
			entries:  []string{"invalidEntryWithoutEquals"},
			expected: map[string]VideoRenderingThread_Frame{},
		},
		{
			name:     "Invalid CID:Hash format",
			entries:  []string{"file2.png=missingColon"},
			expected: map[string]VideoRenderingThread_Frame{},
		},
		{
			name: "Mixed entries",
			entries: []string{
				"valid1.png=cidA:hashA",
				"invalidNoEquals",
				"badCidHash=justcid",
				"valid2.png=cidB:hashB",
			},
			expected: map[string]VideoRenderingThread_Frame{
				"valid1.png": {Filename: "valid1.png", Cid: "cidA", Hash: "hashA"},
				"valid2.png": {Filename: "valid2.png", Cid: "cidB", Hash: "hashB"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FromCliToFrames(tt.entries)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected: %+v, got: %+v", tt.expected, result)
			}
		})
	}
}

// --- Test for FromFramesToCli ---
func TestFromFramesToCli(t *testing.T) {
	tests := []struct {
		name     string
		frames   map[string]VideoRenderingThread_Frame
		expected []string
	}{
		{
			name: "Valid input",
			frames: map[string]VideoRenderingThread_Frame{
				"file1": {Filename: "file1", Cid: "cid1", Hash: "hash1"},
			},
			expected: []string{"file1=cid1:hash1"},
		},
		{
			name:     "Empty map",
			frames:   map[string]VideoRenderingThread_Frame{},
			expected: []string{},
		},
		{
			name: "Multiple frames",
			frames: map[string]VideoRenderingThread_Frame{
				"file1": {Filename: "file1", Cid: "cid1", Hash: "hash1"},
				"file2": {Filename: "file2", Cid: "cid2", Hash: "hash2"},
			},
			expected: []string{"file1=cid1:hash1", "file2=cid2:hash2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FromFramesToCli(tt.frames)
			if len(result) == 0 && len(tt.expected) == 0 {
				return
			}
			sort.Strings(result)
			sort.Strings(tt.expected)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func createTestImage(filePath string) error {
	// Create a simple image (100x100 white image)
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Create the image file
	imgFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer imgFile.Close()

	// Encode the image as PNG
	err = png.Encode(imgFile, img)
	if err != nil {
		return err
	}
	return nil
}

func createTestTextFile(filePath string) error {
	// Create a simple text file with some content
	content := []byte("This is a test text file.")

	// Create the text file
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		return err
	}
	return nil
}

// --- Test for CalculateFileHash ---
func TestCalculateFileHash(t *testing.T) {
	err := createTestImage("test_image.png")
	if err != nil {
		t.Fatalf("Failed to create a test image: %v", err)
	}
	defer os.Remove("test_image.png")

	err = createTestTextFile("text_test_file.txt")
	if err != nil {
		t.Fatalf("Failed to create a text test file: %v", err)
	}
	defer os.Remove("text_test_file.txt")

	tests := []struct {
		name            string
		filePath        string
		expectedToError bool
	}{
		{
			name:            "Valid image file",
			filePath:        "test_image.png",
			expectedToError: false,
		},
		{
			name:            "Non-existent file",
			filePath:        "non_existent_file.png",
			expectedToError: true,
		},
		{
			name:            "Invalid file format",
			filePath:        "text_test_file.txt",
			expectedToError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := CalculateFileHash(tt.filePath)
			if (err != nil) != tt.expectedToError {
				t.Errorf("For test case %s, expected error: %v, got: %v", tt.name, tt.expectedToError, err != nil)
			}
			if !tt.expectedToError && f == "" {
				t.Errorf("Hash calculated is empty")
			}
		})
	}
}

// --- Test for GenerateDirectoryFileHashes ---
func TestGenerateDirectoryFileHashes(t *testing.T) {
	type testCase struct {
		name              string
		setup             func(dir string) error
		expectedHashCount int
		expectError       bool
	}

	createPng := func(path string) error {
		img := image.NewRGBA(image.Rect(0, 0, 1, 1))
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
		return png.Encode(file, img)
	}

	tests := []testCase{
		{
			name: "Directory with 2 PNG files",
			setup: func(dir string) error {
				if err := createPng(filepath.Join(dir, "a.png")); err != nil {
					return err
				}
				if err := createPng(filepath.Join(dir, "b.png")); err != nil {
					return err
				}
				return nil
			},
			expectedHashCount: 2,
			expectError:       false,
		},
		{
			name: "Empty directory",
			setup: func(dir string) error {
				// Nothing to do
				return nil
			},
			expectedHashCount: 0,
			expectError:       false,
		},
		{
			name: "Non-existent directory",
			setup: func(dir string) error {
				return os.RemoveAll(dir)
			},
			expectedHashCount: 0,
			expectError:       true,
		},
		{
			name: "PNG file with unreadable permissions",
			setup: func(dir string) error {
				path := filepath.Join(dir, "unreadable.png")
				if err := createPng(path); err != nil {
					return err
				}
				return os.Chmod(path, 0000)
			},
			expectedHashCount: 0,
			expectError:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "testDir")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(dir)

			if err := tc.setup(dir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			hashes, err := GenerateDirectoryFileHashes(dir)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}
			if !tc.expectError && len(hashes) != tc.expectedHashCount {
				t.Errorf("Expected %d hashes, got %d", tc.expectedHashCount, len(hashes))
			}
		})
	}
}
