package videoRendering

import "cosmossdk.io/errors"

var (
	ErrIndexTooLong     = errors.Register(ModuleName, 2, "index too long")
	ErrDuplicateAddress = errors.Register(ModuleName, 3, "duplicate address")

	ErrWorkerAlreadyRegistered = errors.Register(ModuleName, 4, "worker already registered")
)
