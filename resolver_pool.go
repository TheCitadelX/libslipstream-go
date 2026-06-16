package slipstream

import (
	"net"
	"sync"
	"time"
)

type resolverPool struct {
	mu          sync.Mutex
	resolvers   []*net.UDPAddr
	states      map[string]*resolverState
	nextIndex   int
	baseBackoff time.Duration
	maxBackoff  time.Duration
}

type resolverState struct {
	failures int
	retryAt  time.Time
}

func newResolverPool(resolvers []*net.UDPAddr, baseBackoff, maxBackoff time.Duration) *resolverPool {
	if baseBackoff <= 0 {
		baseBackoff = defaultResolverBackoff
	}
	if maxBackoff <= 0 {
		maxBackoff = defaultResolverMaxBackoff
	}
	if maxBackoff < baseBackoff {
		maxBackoff = baseBackoff
	}
	return &resolverPool{
		resolvers:   resolvers,
		states:      make(map[string]*resolverState, len(resolvers)),
		baseBackoff: baseBackoff,
		maxBackoff:  maxBackoff,
	}
}

func (p *resolverPool) next(now time.Time) *net.UDPAddr {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.resolvers) == 0 {
		return nil
	}

	var fallback *net.UDPAddr
	var fallbackRetry time.Time
	for i := 0; i < len(p.resolvers); i++ {
		index := (p.nextIndex + i) % len(p.resolvers)
		resolver := p.resolvers[index]
		state := p.states[resolver.String()]
		if state == nil || state.retryAt.IsZero() || !state.retryAt.After(now) {
			p.nextIndex = (index + 1) % len(p.resolvers)
			return resolver
		}
		if fallback == nil || state.retryAt.Before(fallbackRetry) {
			fallback = resolver
			fallbackRetry = state.retryAt
		}
	}

	p.nextIndex = (p.nextIndex + 1) % len(p.resolvers)
	return fallback
}

func (p *resolverPool) reportFailure(resolver *net.UDPAddr, now time.Time) {
	if resolver == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	key := resolver.String()
	state := p.states[key]
	if state == nil {
		state = &resolverState{}
		p.states[key] = state
	}
	state.failures++
	state.retryAt = now.Add(p.backoffForFailures(state.failures))
}

func (p *resolverPool) reportSuccess(resolver *net.UDPAddr) {
	if resolver == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.states, resolver.String())
}

func (p *resolverPool) backoffForFailures(failures int) time.Duration {
	if failures <= 1 {
		return p.baseBackoff
	}
	backoff := p.baseBackoff
	for i := 1; i < failures; i++ {
		if backoff >= p.maxBackoff/2 {
			return p.maxBackoff
		}
		backoff *= 2
	}
	if backoff > p.maxBackoff {
		return p.maxBackoff
	}
	return backoff
}
