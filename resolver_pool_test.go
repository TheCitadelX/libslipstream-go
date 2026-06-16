package slipstream

import (
	"net"
	"testing"
	"time"
)

func TestResolverPoolSkipsResolverInBackoff(t *testing.T) {
	resolvers := []*net.UDPAddr{
		{IP: net.ParseIP("127.0.0.1"), Port: 53001},
		{IP: net.ParseIP("127.0.0.1"), Port: 53002},
	}
	pool := newResolverPool(resolvers, 100*time.Millisecond, time.Second)
	now := time.Unix(10, 0)

	if got := pool.next(now); got.String() != resolvers[0].String() {
		t.Fatalf("first resolver = %v, want %v", got, resolvers[0])
	}
	pool.reportFailure(resolvers[1], now)
	if got := pool.next(now.Add(10 * time.Millisecond)); got.String() != resolvers[0].String() {
		t.Fatalf("resolver during backoff = %v, want %v", got, resolvers[0])
	}
	if got := pool.next(now.Add(150 * time.Millisecond)); got.String() != resolvers[1].String() {
		t.Fatalf("resolver after backoff = %v, want %v", got, resolvers[1])
	}
}

func TestResolverPoolSuccessClearsBackoff(t *testing.T) {
	resolvers := []*net.UDPAddr{
		{IP: net.ParseIP("127.0.0.1"), Port: 53001},
		{IP: net.ParseIP("127.0.0.1"), Port: 53002},
	}
	pool := newResolverPool(resolvers, time.Second, time.Second)
	now := time.Unix(10, 0)

	pool.reportFailure(resolvers[0], now)
	pool.reportSuccess(resolvers[0])
	if got := pool.next(now.Add(10 * time.Millisecond)); got.String() != resolvers[0].String() {
		t.Fatalf("resolver after success = %v, want %v", got, resolvers[0])
	}
}

func TestResolverPoolCapsBackoff(t *testing.T) {
	pool := newResolverPool(nil, 100*time.Millisecond, 250*time.Millisecond)
	if got := pool.backoffForFailures(1); got != 100*time.Millisecond {
		t.Fatalf("backoff 1 = %v", got)
	}
	if got := pool.backoffForFailures(2); got != 200*time.Millisecond {
		t.Fatalf("backoff 2 = %v", got)
	}
	if got := pool.backoffForFailures(3); got != 250*time.Millisecond {
		t.Fatalf("backoff 3 = %v", got)
	}
}
