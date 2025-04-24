package videoRendering

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

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

// func TestExecuteCli(t *testing.T) {
// }

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

func TestGenerateDirectoryFileHashes(t *testing.T) {
	type testCase struct {
		dirPath           string
		expectedHashCount int
		expectedToError   bool
	}

	tempDir, err := os.MkdirTemp("", "testDir")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	imgPath := filepath.Join(tempDir, "image.png")
	if err := createTestImage(imgPath); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	testCases := []testCase{
		{
			dirPath:           tempDir,
			expectedHashCount: 1,
			expectedToError:   false,
		},
		{
			dirPath:           filepath.Join(tempDir, "nonExistentDir"),
			expectedHashCount: 0,
			expectedToError:   true,
		},
		{
			dirPath:           filepath.Join(tempDir, "emptyDir"),
			expectedHashCount: 0,
			expectedToError:   false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.dirPath, func(t *testing.T) {
			if tt.dirPath == filepath.Join(tempDir, "emptyDir") {
				if err := os.Mkdir(tt.dirPath, 0755); err != nil {
					t.Fatalf("Failed to create empty directory: %v", err)
				}
				defer os.RemoveAll(tt.dirPath)
			}
			hashes, err := GenerateDirectoryFileHashes(tt.dirPath)
			if (err != nil) != tt.expectedToError {
				t.Errorf("Expected error: %v, got: %v", tt.expectedToError, err != nil)
			}
			if len(hashes) != tt.expectedHashCount {
				t.Errorf("Expected %d hashes, got %d", tt.expectedHashCount, len(hashes))
			}
		})
	}
}
