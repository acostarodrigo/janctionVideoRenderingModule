package mocks

import "github.com/stretchr/testify/mock"

type DB struct {
	mock.Mock
}

func (m *DB) UpdateTask(taskId, threadId string, workerSubscribed bool) error {
	args := m.Called(taskId, threadId, workerSubscribed)
	return args.Error(0)
}

func (m *DB) UpdateThread(id string, downloadStarted, downloadCompleted, workStarted, workCompleted, solProposed, verificationStarted, solutionRevealed bool, submitionStarted bool) error {
	args := m.Called(id, downloadStarted, downloadCompleted, workStarted, workCompleted, solProposed, verificationStarted, solutionRevealed, submitionStarted)
	return args.Error(0)
}

func (m *DB) AddLogEntry(threadId, log string, timestamp, severity int64) error {
	args := m.Called(threadId, log, timestamp, severity)
	return args.Error(0)
}
