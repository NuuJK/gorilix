package messaging

import "errors"

var (
	ErrAllDeliveriesFailed = errors.New("all message deliveries failed")

	ErrInvalidMessageFormat = errors.New("invalid message format")

	ErrNoSubscribers = errors.New("no subscribers for topic")

	ErrDeliveryTimedOut = errors.New("message delivery timed out")
)
