package cluster

import "errors"

var (
	
	ErrInvalidSystemReference = errors.New("invalid system reference")

	
	ErrClusterAlreadyRunning = errors.New("cluster already running")

	
	ErrClusterNotRunning = errors.New("cluster not running")

	
	ErrNodeNotFound = errors.New("node not found")
)
