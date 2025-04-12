package system

import "errors"

var (
	
	ErrNoActorsOfType = errors.New("no actors of specified type")
	
	ErrNoActorsWithTag = errors.New("no actors with specified tag")
	
	ErrActorNotRegistered = errors.New("actor not registered in the system")
	
	ErrSystemStopped = errors.New("actor system is stopped")
)
