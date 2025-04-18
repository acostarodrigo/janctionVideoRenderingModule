package mocks

import (
	"os"
	"time"
)

type MockFileInfo struct {
	Filename string
	Filesize int64
	Filemode os.FileMode
	ModTime_ time.Time
	IsDir_   bool
}

func (m MockFileInfo) Name() string       { return m.Filename }
func (m MockFileInfo) Size() int64        { return m.Filesize }
func (m MockFileInfo) Mode() os.FileMode  { return m.Filemode }
func (m MockFileInfo) ModTime() time.Time { return m.ModTime_ }
func (m MockFileInfo) IsDir() bool        { return m.IsDir_ }
func (m MockFileInfo) Sys() any           { return nil }
