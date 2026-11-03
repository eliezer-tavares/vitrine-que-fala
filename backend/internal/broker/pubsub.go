package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
)

// Pub/Sub implementation of Publisher/Subscriber (ADR-0008). Topics and
// subscriptions are created lazily; the emulator (PUBSUB_EMULATOR_HOST) is
// supported for local demos and CI. Delivery stays at-least-once: Requeue
// re-publishes with the attempts counter, Dead writes to "<topic>.dead".

type PubSubPublisher struct {
	C *pubsub.Client
}

func newPubSubClient(ctx context.Context) (*pubsub.Client, error) {
	project := envOrDefault("PUBSUB_PROJECT", "vitrine-local")
	return pubsub.NewClient(ctx, project)
}

func newPubSubPublisher(ctx context.Context) (*PubSubPublisher, error) {
	c, err := newPubSubClient(ctx)
	if err != nil {
		return nil, err
	}
	return &PubSubPublisher{C: c}, nil
}

func (p *PubSubPublisher) Publish(ctx context.Context, topic string, values map[string]any) error {
	t := p.C.Topic(topic)
	t.PublishSettings = pubsub.PublishSettings{CountThreshold: 1, DelayThreshold: 0}
	_, err := t.Publish(ctx, pubsubMessage(values)).Get(ctx)
	return err
}

// pubsubMessage packs values both as Data (JSON, for portability) and as
// string attributes (for filtering and the subscriber's map shape).
func pubsubMessage(values map[string]any) *pubsub.Message {
	data, err := json.Marshal(values)
	if err != nil {
		data = []byte("{}")
	}
	return &pubsub.Message{Data: data, Attributes: toStrings(values)}
}

// PubSubSubscriber consumes one subscription ("<group>-<consumer>") per
// worker goroutine, mapping the streaming Receive API onto the blocking
// Read/Ack interface the v1.0 processor loop already uses.
type PubSubSubscriber struct {
	C        *pubsub.Client
	Consumer string

	mu      sync.Mutex
	pending map[string]*pubsub.Message
	subs    map[string]*pubsub.Subscription
}

func newPubSubSubscriber(ctx context.Context, consumer string) (*PubSubSubscriber, error) {
	c, err := newPubSubClient(ctx)
	if err != nil {
		return nil, err
	}
	return &PubSubSubscriber{C: c, Consumer: consumer, pending: map[string]*pubsub.Message{}, subs: map[string]*pubsub.Subscription{}}, nil
}

func subName(group, consumer string) string {
	return fmt.Sprintf("%s-%s", group, consumer)
}

func (s *PubSubSubscriber) EnsureGroup(ctx context.Context, topic, group string) error {
	t := s.C.Topic(topic)
	if ok, err := t.Exists(ctx); err != nil {
		return err
	} else if !ok {
		if _, err := s.C.CreateTopic(ctx, topic); err != nil {
			return err
		}
		slog.Info("broker: created pubsub topic", "topic", topic)
	}
	name := subName(group, s.Consumer)
	sub := s.C.Subscription(name)
	if ok, err := sub.Exists(ctx); err != nil {
		return err
	} else if !ok {
		if _, err := s.C.CreateSubscription(ctx, name, pubsub.SubscriptionConfig{Topic: t}); err != nil {
			return err
		}
		slog.Info("broker: created pubsub subscription", "subscription", name)
	}
	s.mu.Lock()
	s.subs[group] = sub
	s.mu.Unlock()
	return nil
}

// Read pulls up to count messages, waiting at most readBlock before
// returning whatever arrived (possibly nothing) — the same contract as the
// Redis Streams blocking XREADGROUP.
func (s *PubSubSubscriber) Read(ctx context.Context, topic, group string, count int64) ([]Message, error) {
	s.mu.Lock()
	sub, ok := s.subs[group]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("broker: subscription for group %q not ensured", group)
	}

	ctx, cancel := context.WithTimeout(ctx, readBlock)
	defer cancel()
	ch := make(chan Message, count)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := sub.Receive(ctx, func(_ context.Context, m *pubsub.Message) {
			vals := make(map[string]any, len(m.Attributes))
			for k, v := range m.Attributes {
				vals[k] = v
			}
			s.mu.Lock()
			s.pending[m.ID] = m
			s.mu.Unlock()
			ch <- Message{ID: m.ID, Values: vals}
		})
		if err != nil && ctx.Err() == nil {
			slog.Warn("broker: pubsub receive error", "error", err)
		}
	}()

	var out []Message
	deadline := time.Now().Add(readBlock)
	for int64(len(out)) < count {
		select {
		case m := <-ch:
			out = append(out, m)
		case <-ctx.Done():
			wg.Wait()
			return out, nil
		}
		if time.Now().After(deadline) {
			break
		}
	}
	wg.Wait()
	return out, nil
}

func (s *PubSubSubscriber) Ack(_ context.Context, _, _, id string) error {
	s.mu.Lock()
	m := s.pending[id]
	delete(s.pending, id)
	s.mu.Unlock()
	if m != nil {
		m.Ack()
	}
	return nil
}

// Requeue re-publishes the values (attempts counter included) to the same
// topic. The processor acks the original delivery right after, mirroring the
// Redis Streams flow.
func (s *PubSubSubscriber) Requeue(ctx context.Context, topic string, values map[string]any) error {
	return s.PublishTo(ctx, topic, values)
}

// Dead moves exhausted jobs to the dead-letter topic ("<topic>.dead").
func (s *PubSubSubscriber) Dead(ctx context.Context, deadTopic string, values map[string]any) error {
	return s.PublishTo(ctx, deadTopic, values)
}

// PublishTo publishes through this subscriber's client (shared helper).
func (s *PubSubSubscriber) PublishTo(ctx context.Context, topic string, values map[string]any) error {
	t := s.C.Topic(topic)
	_, err := t.Publish(ctx, pubsubMessage(values)).Get(ctx)
	return err
}
