package actor

import "errors"

var (
	ErrActorStopped = errors.New("actor is stopped")

	ErrActorNotFound = errors.New("actor not found")

	ErrInvalidActorID = errors.New("invalid actor ID")

	ErrMailboxFull = errors.New("actor mailbox is full")
)
