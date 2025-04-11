package db

import "github.com/stretchr/testify/mock"

type Database interface {
	UpdateTask(taskId, threadId string, completed bool) error
}

type MockDB struct {
	mock.Mock
}

func (m *MockDB) UpdateTask(taskId, threadId string, workerSubscribed bool) error {
	args := m.Called(taskId, threadId, workerSubscribed)
	return args.Error(0)
}
