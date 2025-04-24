package mocks

import (
	sdkcryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/stretchr/testify/mock"
)

type MockPubKey struct {
	mock.Mock
}

func (m *MockPubKey) Reset()         {}
func (m *MockPubKey) String() string { return "mockPubKey" }
func (m *MockPubKey) ProtoMessage()  {}

func (m *MockPubKey) Address() sdkcryptotypes.Address {
	args := m.Called()
	return args.Get(0).(sdkcryptotypes.Address)
}

func (m *MockPubKey) Bytes() []byte {
	args := m.Called()
	return args.Get(0).([]byte)
}

func (m *MockPubKey) VerifySignature(msg, sig []byte) bool {
	args := m.Called(msg, sig)
	return args.Bool(0)
}

func (m *MockPubKey) Equals(other sdkcryptotypes.PubKey) bool {
	args := m.Called(other)
	return args.Bool(0)
}

func (m *MockPubKey) Type() string {
	args := m.Called()
	return args.String(0)
}
