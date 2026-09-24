package realtime

import (
	"context"
	"testing"
)

func TestHub_DeliversOnlyToTopicSubscribers(t *testing.T) {
	h := NewHub(nil)
	h.Start()
	defer func() { _ = h.Close() }()

	a := h.Subscribe("guild:a")
	b := h.Subscribe("guild:b")

	if err := h.Publish(context.Background(), "guild:a", map[string]string{"type": "x"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case got := <-a.C:
		if string(got) != `{"type":"x"}` {
			t.Errorf("payload = %s", got)
		}
	default:
		t.Fatal("subscriber of guild:a got nothing")
	}
	select {
	case got := <-b.C:
		t.Fatalf("subscriber of guild:b got %s", got)
	default:
	}
}

func TestHub_CloseIsIdempotentAndEndsSubscriptions(t *testing.T) {
	h := NewHub(nil)
	h.Start()

	sub := h.Subscribe("t")
	sub.Close()
	sub.Close()
	if _, ok := <-sub.C; ok {
		t.Fatal("closed subscription should have a closed channel")
	}

	other := h.Subscribe("t")
	_ = h.Close()
	_ = h.Close()
	if _, ok := <-other.C; ok {
		t.Fatal("Hub.Close should close open subscriptions")
	}

	late := h.Subscribe("t")
	if _, ok := <-late.C; ok {
		t.Fatal("subscribing after Close should yield a closed subscription")
	}
}

func TestHub_DropsSlowSubscriber(t *testing.T) {
	h := NewHub(nil)
	h.Start()
	defer func() { _ = h.Close() }()

	slow := h.Subscribe("t")
	for i := 0; i <= subscriptionBuffer; i++ {
		_ = h.Publish(context.Background(), "t", i)
	}

	n := 0
	for range slow.C {
		n++
	}
	if n != subscriptionBuffer {
		t.Fatalf("received %d buffered events before drop, want %d", n, subscriptionBuffer)
	}
}
