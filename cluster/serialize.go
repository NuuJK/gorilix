package cluster

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kleeedolinux/gorilix/messaging"
)


type MessageWrapper struct {
	ID        string            `json:"id"`
	Type      int               `json:"type"`
	Sender    string            `json:"sender"`
	Receiver  string            `json:"receiver"`
	Timestamp int64             `json:"timestamp"`
	Headers   map[string]string `json:"headers"`
	Payload   []byte            `json:"payload"`
}


func SerializeMessage(msg *messaging.Message) ([]byte, error) {
	
	var payloadBytes []byte
	var err error

	switch p := msg.Payload.(type) {
	case []byte:
		payloadBytes = p
	case string:
		payloadBytes = []byte(p)
	default:
		
		payloadBytes, err = json.Marshal(p)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize payload: %w", err)
		}
	}

	wrapper := MessageWrapper{
		ID:        msg.ID,
		Type:      int(msg.Type),
		Sender:    msg.Sender,
		Receiver:  msg.Receiver,
		Timestamp: msg.Timestamp.UnixNano(),
		Headers:   msg.Headers,
		Payload:   payloadBytes,
	}

	data, err := json.Marshal(wrapper)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	return data, nil
}


func DeserializeMessage(data []byte) (*messaging.Message, error) {
	wrapper := &MessageWrapper{}
	if err := json.Unmarshal(data, wrapper); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	msg := &messaging.Message{
		ID:        wrapper.ID,
		Type:      messaging.MessageType(wrapper.Type),
		Sender:    wrapper.Sender,
		Receiver:  wrapper.Receiver,
		Timestamp: time.Unix(0, wrapper.Timestamp),
		Headers:   wrapper.Headers,
		Payload:   wrapper.Payload,
	}

	
	
	if contentType, ok := wrapper.Headers["content-type"]; ok && contentType == "application/json" {
		var jsonPayload interface{}
		if err := json.Unmarshal(wrapper.Payload, &jsonPayload); err == nil {
			msg.Payload = jsonPayload
		}
	}

	return msg, nil
}
