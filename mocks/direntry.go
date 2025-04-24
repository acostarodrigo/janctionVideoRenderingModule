package mocks

import (
	"io/fs"
)

type MockDirEntry struct {
	Filename string
	IsDir_   bool
}

func (m MockDirEntry) Name() string               { return m.Filename }
func (m MockDirEntry) IsDir() bool                { return m.IsDir_ }
func (m MockDirEntry) Type() fs.FileMode          { return 0 }
func (m MockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }
