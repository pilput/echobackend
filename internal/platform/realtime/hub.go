// Package realtime fans events out to long-lived subscribers (SSE streams).
//
// Every instance keeps its subscribers in memory. When Redis is available,
// Publish goes through Redis pub/sub and each instance delivers what it
// receives from its single pattern subscription, so events reach subscribers
// connected to any instance. Without Redis, delivery is local only, which is
// correct for a single instance.
package realtime

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"echobackend/internal/platform/cache"
	"echobackend/pkg/applog"
)

var log = applog.Component("realtime")

// subscriptionBuffer is how many undelivered events a subscriber may lag
// behind before it is dropped.
const subscriptionBuffer = 64

// Subscription receives the events published to one topic. C is closed when
// the subscription ends: on Close, on Hub.Close, or when the subscriber falls
// too far behind. A dropped subscriber should reconnect and refetch history.
type Subscription struct {
	C     <-chan []byte
	c     chan []byte
	topic string
	hub   *Hub
}

// Close ends the subscription. It is safe to call more than once.
func (s *Subscription) Close() {
	s.hub.remove(s)
}

type Hub struct {
	redis  *cache.RedisCache
	prefix string

	mu     sync.Mutex
	subs   map[string]map[*Subscription]struct{}
	closed bool

	cancel context.CancelFunc
	done   chan struct{}
}

// NewHub creates a hub. redis may be nil, which keeps delivery in-process.
func NewHub(redis *cache.RedisCache) *Hub {
	h := &Hub{
		redis: redis,
		subs:  make(map[string]map[*Subscription]struct{}),
		done:  make(chan struct{}),
	}
	if redis != nil {
		h.prefix = redis.BuildKey("realtime") + ":"
	}
	return h
}

// Start begins relaying Redis messages to local subscribers. It is a no-op
// without Redis.
func (h *Hub) Start() {
	if h.redis == nil {
		close(h.done)
		return
	}
	ps := h.redis.PSubscribe(context.Background(), h.prefix+"*")
	if ps == nil {
		close(h.done)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	go func() {
		defer close(h.done)
		defer func() { _ = ps.Close() }()
		ch := ps.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				h.deliver(strings.TrimPrefix(msg.Channel, h.prefix), []byte(msg.Payload))
			}
		}
	}()
}

// Subscribe registers a subscriber for topic. After Close the returned
// subscription is already closed.
func (h *Hub) Subscribe(topic string) *Subscription {
	c := make(chan []byte, subscriptionBuffer)
	sub := &Subscription{C: c, c: c, topic: topic, hub: h}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		close(c)
		return sub
	}
	set, ok := h.subs[topic]
	if !ok {
		set = make(map[*Subscription]struct{})
		h.subs[topic] = set
	}
	set[sub] = struct{}{}
	return sub
}

// Publish marshals event and sends it to every subscriber of topic, across
// instances when Redis is available. If Redis rejects the message it is still
// delivered to this instance's subscribers.
func (h *Hub) Publish(ctx context.Context, topic string, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if h.redis != nil {
		if err := h.redis.Publish(ctx, h.prefix+topic, payload); err == nil {
			return nil
		}
	}
	h.deliver(topic, payload)
	return nil
}

// Close stops the Redis relay and ends every subscription, which lets open
// SSE handlers return so the HTTP server can shut down. Safe to call twice.
func (h *Hub) Close() error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil
	}
	h.closed = true
	for topic, set := range h.subs {
		for sub := range set {
			close(sub.c)
		}
		delete(h.subs, topic)
	}
	h.mu.Unlock()

	if h.cancel != nil {
		h.cancel()
		<-h.done
	}
	return nil
}

func (h *Hub) deliver(topic string, payload []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for sub := range h.subs[topic] {
		select {
		case sub.c <- payload:
		default:
			// A subscriber that cannot keep up is dropped rather than allowed
			// to block everyone else; it silently missing events would be worse.
			log.Warn("realtime: dropping slow subscriber", "topic", topic)
			h.removeLocked(sub)
		}
	}
}

func (h *Hub) remove(sub *Subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeLocked(sub)
}

// removeLocked closes sub if it is still registered. Channels are only closed
// here and in Close, both under h.mu, so a channel is never closed twice.
func (h *Hub) removeLocked(sub *Subscription) {
	set, ok := h.subs[sub.topic]
	if !ok {
		return
	}
	if _, ok := set[sub]; !ok {
		return
	}
	delete(set, sub)
	close(sub.c)
	if len(set) == 0 {
		delete(h.subs, sub.topic)
	}
}
