package engine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestApprovals_RegisterResolveApprove(t *testing.T) {
	r := NewApprovals()
	id, ch := r.Register()
	assert.NotEmpty(t, id)

	go r.Resolve(id, true)

	select {
	case ok := <-ch:
		assert.True(t, ok)
	case <-time.After(time.Second):
		t.Fatal("did not receive resolution")
	}
}

func TestApprovals_ResolveUnknownIsNoop(t *testing.T) {
	r := NewApprovals()
	r.Resolve("missing", true) // must not panic
}

func TestApprovals_AwaitTimeoutReturnsFalse(t *testing.T) {
	r := NewApprovals()
	ctx := context.Background()
	id, ch := r.Register()
	defer r.Cancel(id)
	approved := r.Await(ctx, id, ch, 5*time.Millisecond)
	assert.False(t, approved)
}

func TestApprovals_AwaitContextCancelReturnsFalse(t *testing.T) {
	r := NewApprovals()
	ctx, cancel := context.WithCancel(context.Background())
	id, ch := r.Register()
	defer r.Cancel(id)
	cancel()
	approved := r.Await(ctx, id, ch, time.Second)
	assert.False(t, approved)
}

func TestApprovals_AwaitApproveDelivers(t *testing.T) {
	r := NewApprovals()
	id, ch := r.Register()
	go r.Resolve(id, true)
	approved := r.Await(context.Background(), id, ch, time.Second)
	assert.True(t, approved)
}
