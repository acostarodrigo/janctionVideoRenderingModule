package db

import "github.com/stretchr/testify/mock"

type Database interface {
	UpdateTask(taskId, threadId string, completed bool) error
	UpdateThread(id string, downloadStarted, downloadCompleted, workStarted, workCompleted, solProposed, verificationStarted, solutionRevealed bool, submitionStarted bool) error
	AddLogEntry(threadId, log string, timestamp, severity int64) error
}

type MockDB struct {
	mock.Mock
}

func (m *MockDB) UpdateTask(taskId, threadId string, workerSubscribed bool) error {
	args := m.Called(taskId, threadId, workerSubscribed)
	return args.Error(0)
}

func (m *MockDB) UpdateThread(id string, downloadStarted, downloadCompleted, workStarted, workCompleted, solProposed, verificationStarted, solutionRevealed bool, submitionStarted bool) error {
	args := m.Called(id, downloadStarted, downloadCompleted, workStarted, workCompleted, solProposed, verificationStarted, solutionRevealed, submitionStarted)
	return args.Error(0)
}

func (m *MockDB) AddLogEntry(threadId, log string, timestamp, severity int64) error {
	args := m.Called(threadId, log, timestamp, severity)
	return args.Error(0)
}
