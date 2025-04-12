package messaging

import (
	"context"
	"sync"
	"time"

	"github.com/kleeedolinux/gorilix/actor"
)

type MessageType int

const (
	Normal MessageType = iota

	Priority

	Broadcast

	System
)

type Message struct {
	Type      MessageType
	Payload   interface{}
	Sender    string
	Receiver  string
	Timestamp time.Time
	ID        string
	Headers   map[string]string
}

type MessageBus struct {
	subscribers      map[string][]actor.ActorRef
	topicLock        sync.RWMutex
	deliveryTimeout  time.Duration
	retries          int
	ackedDelivery    bool
	undeliveredQueue map[string][]Message
	mu               sync.RWMutex
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		subscribers:      make(map[string][]actor.ActorRef),
		deliveryTimeout:  5 * time.Second,
		retries:          3,
		ackedDelivery:    false,
		undeliveredQueue: make(map[string][]Message),
	}
}

func (m *MessageBus) SetDeliveryOptions(timeout time.Duration, retries int, ackedDelivery bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deliveryTimeout = timeout
	m.retries = retries
	m.ackedDelivery = ackedDelivery
}

func (m *MessageBus) Subscribe(topic string, subscriber actor.ActorRef) {
	m.topicLock.Lock()
	defer m.topicLock.Unlock()

	for _, sub := range m.subscribers[topic] {
		if sub.ID() == subscriber.ID() {
			return
		}
	}

	m.subscribers[topic] = append(m.subscribers[topic], subscriber)
}

func (m *MessageBus) Unsubscribe(topic string, subscriberID string) {
	m.topicLock.Lock()
	defer m.topicLock.Unlock()

	subs, exists := m.subscribers[topic]
	if !exists {
		return
	}

	var newSubs []actor.ActorRef
	for _, sub := range subs {
		if sub.ID() != subscriberID {
			newSubs = append(newSubs, sub)
		}
	}

	m.subscribers[topic] = newSubs
}

func (m *MessageBus) Publish(ctx context.Context, topic string, msg Message) error {
	m.topicLock.RLock()
	subscribers := m.subscribers[topic]
	m.topicLock.RUnlock()

	if len(subscribers) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	var failedDeliveries int
	var mu sync.Mutex

	for _, sub := range subscribers {
		wg.Add(1)

		go func(subscriber actor.ActorRef) {
			defer wg.Done()

			deliveryCtx, cancel := context.WithTimeout(ctx, m.deliveryTimeout)
			defer cancel()

			var err error
			for attempt := 0; attempt < m.retries; attempt++ {
				err = subscriber.Send(deliveryCtx, msg)
				if err == nil {
					break
				}

				select {
				case <-deliveryCtx.Done():
					return
				case <-time.After(100 * time.Millisecond):

				}
			}

			if err != nil {
				mu.Lock()
				failedDeliveries++

				if m.ackedDelivery {
					m.queueUndeliveredMessage(subscriber.ID(), msg)
				}
				mu.Unlock()
			}
		}(sub)
	}

	wg.Wait()

	if failedDeliveries > 0 && failedDeliveries == len(subscribers) {
		return ErrAllDeliveriesFailed
	}

	return nil
}

func (m *MessageBus) SendDirectMessage(ctx context.Context, to actor.ActorRef, msg Message) error {
	deliveryCtx, cancel := context.WithTimeout(ctx, m.deliveryTimeout)
	defer cancel()

	var err error
	for attempt := 0; attempt < m.retries; attempt++ {
		err = to.Send(deliveryCtx, msg)
		if err == nil {
			return nil
		}

		select {
		case <-deliveryCtx.Done():
			return deliveryCtx.Err()
		case <-time.After(100 * time.Millisecond):

		}
	}

	if m.ackedDelivery && err != nil {
		m.queueUndeliveredMessage(to.ID(), msg)
	}

	return err
}

func (m *MessageBus) queueUndeliveredMessage(actorID string, msg Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.undeliveredQueue[actorID] = append(m.undeliveredQueue[actorID], msg)
}

func (m *MessageBus) GetUndeliveredMessages(actorID string) []Message {
	m.mu.RLock()
	messages := m.undeliveredQueue[actorID]
	m.mu.RUnlock()

	return messages
}

func (m *MessageBus) ClearUndeliveredMessages(actorID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.undeliveredQueue, actorID)
}

func (m *MessageBus) RetryUndeliveredMessages(ctx context.Context, to actor.ActorRef) error {
	actorID := to.ID()

	m.mu.Lock()
	messages := m.undeliveredQueue[actorID]
	delete(m.undeliveredQueue, actorID)
	m.mu.Unlock()

	if len(messages) == 0 {
		return nil
	}

	var lastErr error
	for _, msg := range messages {
		err := m.SendDirectMessage(ctx, to, msg)
		if err != nil {
			lastErr = err

			m.queueUndeliveredMessage(actorID, msg)
		}
	}

	return lastErr
}
