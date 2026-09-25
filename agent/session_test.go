package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
)

type nopSubscriber struct{ mu sync.Mutex }

func (n *nopSubscriber) OnUserMessage(UserMessageChunk)                {}
func (n *nopSubscriber) OnAgentMessage(AgentMessageChunk)              {}
func (n *nopSubscriber) OnAgentThought(AgentThoughtChunk)              {}
func (n *nopSubscriber) OnToolCallInit(ToolCall)                       {}
func (n *nopSubscriber) OnToolCallUpdate(ToolCallUpdate)               {}
func (n *nopSubscriber) OnPlan(Plan)                                   {}
func (n *nopSubscriber) OnConfigOptionUpdate(ConfigOptionUpdate)       {}
func (n *nopSubscriber) OnRequestPermission(req PermissionGrantRequest) {}

// within fails the test if f doesn't return within d (a deadlock).
func within(t *testing.T, d time.Duration, what string, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { f(); close(done) }()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("%s: blocked for %v", what, d)
	}
}

func TestSubscribeUpdatesTwiceDoesNotDeadlock(t *testing.T) {
	sn := NewACPSession("s1", t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	within(t, time.Second, "second SubscribeUpdates + Title", func() {
		sn.SubscribeUpdates(ctx, &nopSubscriber{})
		sn.SubscribeUpdates(ctx, &nopSubscriber{}) // used to return with sn.mu held
		_ = sn.Title()                             // ...so this deadlocked
	})
}

func TestCloseUnblocksSendersAndIsIdempotent(t *testing.T) {
	sn := NewACPSession("s1", t.TempDir())
	// No subscriber: fill the update buffer so the next publish blocks.
	for i := 0; i < maxSessionUpdates; i++ {
		sn.PublishUpdate(AgentMessageChunk{})
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); sn.PublishUpdate(AgentMessageChunk{}) }()
	go func() {
		defer wg.Done()
		if sn.RequestPermission(acp.RequestPermissionRequest{}, make(chan acp.PermissionOptionId, 1)) {
			t.Error("RequestPermission delivered on a closed session")
		}
	}()
	time.Sleep(20 * time.Millisecond)

	within(t, time.Second, "Close + blocked senders", func() {
		sn.Close()
		sn.Close() // closing twice used to panic (close of closed channel)
		wg.Wait()
	})

	// Publishing after Close must not panic (it used to: send on closed channel).
	within(t, time.Second, "PublishUpdate after Close", func() { sn.PublishUpdate(AgentMessageChunk{}) })
}
