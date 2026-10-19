// Copyright (c) 2024 Tencent Inc.
// SPDX-License-Identifier: Apache-2.0

package pool

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestDecrRefUnderflow(t *testing.T) {
	p := &pool{
		ref:     0,
		current: 1,
	}

	// Calling decrRef when ref is 0 should NOT drive p.ref below 0
	p.decrRef()
	if currentRef := atomic.LoadInt32(&p.ref); currentRef < 0 {
		t.Fatalf("expected ref >= 0 after underflow decrRef, got %d", currentRef)
	}

	// Multiple redundant decrRef calls should stay clamped at 0
	for i := 0; i < 5; i++ {
		p.decrRef()
	}
	if currentRef := atomic.LoadInt32(&p.ref); currentRef != 0 {
		t.Fatalf("expected ref == 0 after multiple redundant decrRefs, got %d", currentRef)
	}

	// Subsequent incrRef should correctly increment from 0 to 1
	newRef, ok := p.incrRef()
	if !ok {
		t.Fatal("expected incrRef to succeed")
	}
	if newRef != 1 {
		t.Fatalf("expected incrRef to return 1, got %d", newRef)
	}
	if currentRef := atomic.LoadInt32(&p.ref); currentRef != 1 {
		t.Fatalf("expected p.ref == 1, got %d", currentRef)
	}
}

func TestGracefulStopImmediateWhenZeroRef(t *testing.T) {
	p := &pool{
		ref:     0,
		current: 1,
	}

	start := time.Now()
	p.GracefulStop(1 * time.Second)
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Fatalf("GracefulStop took %v with ref=0; should have returned immediately", elapsed)
	}
}

func TestConcurrentIncrDecrRef(t *testing.T) {
	p := &pool{
		ref:     0,
		current: 1,
	}

	var wg sync.WaitGroup
	workers := 20
	iterations := 1000

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if _, ok := p.incrRef(); !ok {
					t.Errorf("incrRef saturated unexpectedly")
					return
				}
				if cur := atomic.LoadInt32(&p.ref); cur < 0 {
					t.Errorf("ref underflow during concurrent incr: %d", cur)
				}
				p.decrRef()
				if cur := atomic.LoadInt32(&p.ref); cur < 0 {
					t.Errorf("ref underflow during concurrent decr: %d", cur)
				}
			}
		}()
	}

	wg.Wait()

	if ref := atomic.LoadInt32(&p.ref); ref != 0 {
		t.Fatalf("expected ref == 0 after balanced concurrent ops, got: %d", ref)
	}
}

func TestGracefulStopTimeout(t *testing.T) {
	p := &pool{
		ref:     2,
		current: 1,
	}

	start := time.Now()
	p.GracefulStop(50 * time.Millisecond)
	elapsed := time.Since(start)

	if elapsed < 40*time.Millisecond || elapsed > 2*time.Second {
		t.Fatalf("GracefulStop waited unexpected duration: %v", elapsed)
	}
}

func TestIncrRefOverflowProtection(t *testing.T) {
	p := &pool{
		ref:     math.MaxInt32 - 1,
		current: 1,
	}

	ref, ok := p.incrRef()
	if !ok || ref != math.MaxInt32 {
		t.Fatalf("expected successful increment to MaxInt32, got ref=%d ok=%v", ref, ok)
	}
	if ref, ok = p.incrRef(); ok {
		t.Fatalf("expected saturation at MaxInt32, got ref=%d ok=%v", ref, ok)
	}
	if currentRef := atomic.LoadInt32(&p.ref); currentRef != math.MaxInt32 {
		t.Fatalf("expected p.ref to stay at MaxInt32, got %d", currentRef)
	}

	if _, err := p.Get(); err != ErrRefOverflow {
		t.Fatalf("expected ErrRefOverflow from Get, got %v", err)
	}
	if currentRef := atomic.LoadInt32(&p.ref); currentRef != math.MaxInt32 {
		t.Fatalf("Get on a saturated pool must not change ref, got %d", currentRef)
	}
}

func TestGetAfterCloseNoRefLeak(t *testing.T) {
	p := &pool{
		ref:     0,
		current: 0, // closed pool
	}

	_, err := p.Get()
	if err != ErrClosed {
		t.Fatalf("expected ErrClosed, got %v", err)
	}

	if ref := atomic.LoadInt32(&p.ref); ref != 0 {
		t.Fatalf("expected ref == 0 after Get() on closed pool, got %d", ref)
	}
}

func TestDecrRefShrinkPath(t *testing.T) {
	p := &pool{
		ref:     1,
		current: 4,
		opt: Options{
			MaxIdle:   2,
			MaxActive: 4,
		},
		conns: make([]*conn, 4),
	}

	for i := range p.conns {
		p.conns[i] = &conn{pool: p}
	}

	p.decrRef()

	if ref := atomic.LoadInt32(&p.ref); ref != 0 {
		t.Fatalf("expected ref == 0 after decrRef, got %d", ref)
	}
	if current := atomic.LoadInt32(&p.current); current != 2 {
		t.Fatalf("expected current reset to MaxIdle (2), got %d", current)
	}
	for i := 2; i < 4; i++ {
		if p.conns[i] != nil {
			t.Fatalf("expected conn at index %d to be nil after shrink, got %v", i, p.conns[i])
		}
	}
}

func TestGetDialFailureNoRefLeak(t *testing.T) {
	dialErr := errors.New("dial failure")
	p := &pool{
		ref:     2,
		current: 2,
		opt: Options{
			MaxIdle:              1,
			MaxActive:            2,
			MaxConcurrentStreams: 1,
			Reuse:                false,
			Dial: func(address string) (*grpc.ClientConn, error) {
				return nil, dialErr
			},
		},
		conns: make([]*conn, 2),
	}

	// Case 1: current >= MaxActive with Reuse=false dial error
	_, err := p.Get()
	if err != dialErr {
		t.Fatalf("expected dialErr, got %v", err)
	}
	if ref := atomic.LoadInt32(&p.ref); ref != 2 {
		t.Fatalf("expected ref == 2 after dial failure in Get(), got %d", ref)
	}

	// Case 2: expansion block dial error
	p2 := &pool{
		ref:     1,
		current: 1,
		opt: Options{
			MaxIdle:              1,
			MaxActive:            4,
			MaxConcurrentStreams: 1,
			Dial: func(address string) (*grpc.ClientConn, error) {
				return nil, dialErr
			},
		},
		conns: make([]*conn, 4),
	}
	_, err2 := p2.Get()
	if err2 != dialErr {
		t.Fatalf("expected dialErr, got %v", err2)
	}
	if ref := atomic.LoadInt32(&p2.ref); ref != 1 {
		t.Fatalf("expected ref == 1 after expansion dial failure in Get(), got %d", ref)
	}
}

func TestGetNilConnReturnsErrClosed(t *testing.T) {
	p := &pool{
		ref:     0,
		current: 1,
		opt: Options{
			OptionType: SingleConn,
		},
		conns: make([]*conn, 1),
	}
	_, err := p.Get()
	if err != ErrClosed {
		t.Fatalf("expected ErrClosed when conns[0] is nil, got %v", err)
	}
	if ref := atomic.LoadInt32(&p.ref); ref != 0 {
		t.Fatalf("expected ref == 0 after nil conn check, got %d", ref)
	}
}

func TestDecrRefRedundantDoesNotShrink(t *testing.T) {
	p := &pool{
		ref:     0,
		current: 4,
		opt: Options{
			MaxIdle:   2,
			MaxActive: 4,
		},
		conns: make([]*conn, 4),
	}
	for i := range p.conns {
		p.conns[i] = &conn{pool: p}
	}

	p.decrRef()

	if ref := atomic.LoadInt32(&p.ref); ref != 0 {
		t.Fatalf("expected ref == 0 after redundant decrRef, got %d", ref)
	}
	if current := atomic.LoadInt32(&p.current); current != 4 {
		t.Fatalf("redundant decrRef must not shrink the pool, current = %d", current)
	}
	for i, c := range p.conns {
		if c == nil {
			t.Fatalf("redundant decrRef must not reset conns[%d]", i)
		}
	}
}

func TestConcurrentGetCloseBalancesRef(t *testing.T) {
	p := &pool{
		current: 1,
		opt: Options{
			OptionType: SingleConn,
			MaxIdle:    1,
			MaxActive:  1,
		},
		conns: make([]*conn, 1),
	}
	p.conns[0] = &conn{pool: p}

	var wg sync.WaitGroup
	const workers = 8
	const iterations = 500

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				c, err := p.Get()
				if err != nil {
					t.Errorf("Get failed: %v", err)
					return
				}
				// Exercise the active-time read path concurrently with Get's
				// deferred write; -race must stay clean.
				_, _ = p.GetActiveTimeAndRef()
				if err := c.Close(); err != nil {
					t.Errorf("Close failed: %v", err)
					return
				}
			}
		}()
	}

	wg.Wait()

	if ref := atomic.LoadInt32(&p.ref); ref != 0 {
		t.Fatalf("expected ref == 0 after concurrent Get/Close, got %d", ref)
	}
}
